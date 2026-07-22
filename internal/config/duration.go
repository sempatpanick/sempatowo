package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Duration is a time.Duration written in JSON as a string ("15s", "5m",
// "1m30s") — the same syntax time.ParseDuration accepts.
//
// The old schema wrote bare numbers whose unit depended on the field: intervals
// were milliseconds, cooldowns were seconds. Nothing in the JSON said which, so
// a value moved between two fields silently changed meaning by 1000x. Requiring
// the unit in the value removes the question. Legacy bare numbers are still
// read, but only by the migration in legacy.go, which knows each field's unit.
type Duration time.Duration

// Std returns the underlying time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Millis returns the duration in whole milliseconds.
func (d Duration) Millis() int { return int(time.Duration(d) / time.Millisecond) }

// Seconds returns the duration in fractional seconds.
func (d Duration) Seconds() float64 { return time.Duration(d).Seconds() }

func (d Duration) String() string { return time.Duration(d).String() }

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("duration must be a string like \"15s\" or \"5m\", got %s", string(data))
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Range is an inclusive min/max duration pair. Every "wait somewhere between
// these two times" setting uses it, so there is one shape to learn.
type Range struct {
	Min Duration `json:"min"`
	Max Duration `json:"max"`
}

// Pick returns a uniformly random duration in [Min, Max]. A zero or inverted
// range degrades to Min rather than panicking on a bad rand argument.
func (r Range) Pick() time.Duration {
	min, max := r.Min.Std(), r.Max.Std()
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)+1))
}

// SecondsRange returns min/max in fractional seconds, for callers that still
// speak float seconds.
func (r Range) SecondsRange() (min, max float64) {
	return r.Min.Seconds(), r.Max.Seconds()
}

// IsZero reports whether neither bound was set.
func (r Range) IsZero() bool { return r.Min == 0 && r.Max == 0 }

func secs(n float64) Duration {
	return Duration(time.Duration(n * float64(time.Second)))
}

func rangeSecs(min, max float64) Range {
	return Range{Min: secs(min), Max: secs(max)}
}

func rangeMillis(min, max int) Range {
	return Range{
		Min: Duration(time.Duration(min) * time.Millisecond),
		Max: Duration(time.Duration(max) * time.Millisecond),
	}
}
