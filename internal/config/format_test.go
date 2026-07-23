package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The written file is the one thing a user actually reads, so this test prints
// it on failure and checks the properties that make it readable.
func TestMarshalReadableRoundTrips(t *testing.T) {
	data, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}

	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("the file we write does not parse: %v\n%s", err, data)
	}
	if !reflect.DeepEqual(got, Defaults()) {
		t.Errorf("round trip changed the settings\n%s", data)
	}
}

// Rewriting the file on every start would churn it if the output were not
// stable, and a config under version control would show phantom diffs.
func TestMarshalReadableIsStable(t *testing.T) {
	first, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	second, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	if string(first) != string(second) {
		t.Error("two renderings of the same settings differ")
	}
}

func TestMarshalReadableInlinesShortBlocks(t *testing.T) {
	data, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	out := string(data)

	for _, want := range []string{
		`"crate": { "enabled": true }`,
		`"hunt": { "enabled": true, "delay": { "min": "50s", "max": "3m20s" } }`,
		`"humanize": { "typing": true, "sendMessageInterval": "5s" }`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %s\n%s", want, out)
		}
	}

	// The whole point of the printer. Comparing against the encoder it replaced
	// keeps the assertion meaningful as settings are added.
	indented, err := json.MarshalIndent(Defaults(), "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	got, was := strings.Count(out, "\n"), strings.Count(string(indented), "\n")
	if got*3 >= was*2 {
		t.Errorf("output is %d lines against MarshalIndent's %d; expected under two thirds\n%s", got, was, out)
	}
}

// Nothing may exceed the width budget except a single value that cannot be
// broken, which the config has none of.
func TestMarshalReadableRespectsWidth(t *testing.T) {
	data, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) > inlineWidth {
			t.Errorf("line exceeds %d columns (%d): %s", inlineWidth, len(line), line)
		}
	}
}

func TestMarshalReadableHandlesEmptyContainers(t *testing.T) {
	type holder struct {
		Empty    map[string]string `json:"empty"`
		EmptyArr []string          `json:"emptyArr"`
	}
	data, err := marshalReadable(holder{Empty: map[string]string{}, EmptyArr: []string{}})
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	if got, want := strings.TrimSpace(string(data)), `{ "empty": {}, "emptyArr": [] }`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
