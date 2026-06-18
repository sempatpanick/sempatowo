package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	sync.Mutex
	global  *time.Time
	buckets map[string]*Bucket
}

type Bucket struct {
	sync.Mutex
	Remaining  int
	Limit      int
	Reset      time.Time
	ResetAfter float64
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*Bucket),
	}
}

func (r *RateLimiter) GetBucket(route string) *Bucket {
	r.Lock()
	defer r.Unlock()

	if bucket, ok := r.buckets[route]; ok {
		return bucket
	}

	bucket := &Bucket{
		Remaining: 1,
		Limit:     1,
	}
	r.buckets[route] = bucket
	return bucket
}

func (b *Bucket) Wait(ctx context.Context) error {
	b.Lock()
	defer b.Unlock()

	if b.Remaining <= 0 && time.Now().Before(b.Reset) {
		waitTime := time.Until(b.Reset)
		if waitTime > 0 {
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (b *Bucket) Update(header http.Header) {
	b.Lock()
	defer b.Unlock()

	if remaining := header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			b.Remaining = val
		}
	}

	if limit := header.Get("X-RateLimit-Limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			b.Limit = val
		}
	}

	if reset := header.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseFloat(reset, 64); err == nil {
			b.Reset = time.Unix(int64(val), 0)
		}
	}

	if resetAfter := header.Get("X-RateLimit-Reset-After"); resetAfter != "" {
		if val, err := strconv.ParseFloat(resetAfter, 64); err == nil {
			b.ResetAfter = val
		}
	}
}

func (r *RateLimiter) SetGlobal(resetAfter time.Duration) {
	r.Lock()
	defer r.Unlock()
	resetTime := time.Now().Add(resetAfter)
	r.global = &resetTime
}

func (r *RateLimiter) WaitForGlobal() {
	r.Lock()
	if r.global != nil && time.Now().Before(*r.global) {
		waitTime := time.Until(*r.global)
		r.Unlock()
		time.Sleep(waitTime)
		return
	}
	r.Unlock()
}
