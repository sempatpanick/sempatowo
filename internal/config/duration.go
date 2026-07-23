package config

import (
	"bytes"
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

// String renders the duration for the config file. Go's own formatting spells
// five minutes "5m0s"; a whole number of minutes or hours is written without
// the redundant tail, since the file is meant to be read and edited by hand.
func (d Duration) String() string {
	std := time.Duration(d)
	switch {
	case std == 0:
		return "0s"
	case std%time.Hour == 0:
		return fmt.Sprintf("%dh", std/time.Hour)
	case std%time.Minute == 0:
		return fmt.Sprintf("%dm", std/time.Minute)
	default:
		return std.String()
	}
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
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
//
// In JSON it is written either as an object, {"min": "10s", "max": "30s"}, or —
// when both bounds are the same — as a bare duration, "10s". Most of the delays
// in a config file are fixed, and spelling those out as an object put the same
// string on the page twice and buried the ones that actually do jitter.
type Range struct {
	Min Duration `json:"min"`
	Max Duration `json:"max"`
}

// rangeJSON exists only to give Range's object form a type to decode into that
// does not inherit Range's own UnmarshalJSON.
type rangeJSON struct {
	Min Duration `json:"min"`
	Max Duration `json:"max"`
}

func (r Range) MarshalJSON() ([]byte, error) {
	if r.Min == r.Max {
		return json.Marshal(r.Min.String())
	}
	return json.Marshal(rangeJSON{Min: r.Min, Max: r.Max})
}

func (r *Range) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var d Duration
		if err := d.UnmarshalJSON(data); err != nil {
			return err
		}
		r.Min, r.Max = d, d
		return nil
	}

	// Taking over this subtree takes it away from the DisallowUnknownFields
	// probe in loader.go, so the check has to be repeated here — otherwise
	// {"min": "5s", "mx": "9s"} would silently load as a max of zero.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var obj rangeJSON
	if err := dec.Decode(&obj); err != nil {
		return fmt.Errorf("range must be a duration like \"15s\" or an object like "+
			"{\"min\": \"10s\", \"max\": \"30s\"}: %w", err)
	}
	r.Min, r.Max = obj.Min, obj.Max
	return nil
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
