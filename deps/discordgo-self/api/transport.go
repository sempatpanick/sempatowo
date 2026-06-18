package api

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func InternalTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
}

func ConfigureTransport() *http.Transport {
	return InternalTransport()
}

type TransportWithUA struct {
	Transport http.RoundTripper
	UserAgent string
}

func (t *TransportWithUA) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	return t.Transport.RoundTrip(req)
}

type RetryTransport struct {
	Transport  http.RoundTripper
	MaxRetries int
	RetryDelay time.Duration
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= t.MaxRetries; i++ {
		resp, err = t.Transport.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if i < t.MaxRetries {
			time.Sleep(t.RetryDelay * time.Duration(i+1))
		}
	}

	return resp, err
}

type TimeoutTransport struct {
	Transport http.RoundTripper
	Timeout   time.Duration
}

func (t *TimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), t.Timeout)
	defer cancel()

	req = req.WithContext(ctx)
	return t.Transport.RoundTrip(req)
}
