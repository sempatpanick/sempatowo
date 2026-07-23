package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "user.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func mustLoad(t *testing.T, path string) Settings {
	t.Helper()
	s, _, err := loadFromFile(path, "tester")
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	return s
}

func TestLoadFromFileFillsMissingKeysFromDefaults(t *testing.T) {
	// A minimal config must still come back fully populated.
	path := writeConfig(t, `{"schemaVersion":2,"discord":{"defaultChannel":"123456789012345678"}}`)

	got := mustLoad(t, path)

	if got.DefaultChannel != "123456789012345678" {
		t.Errorf("user value lost: defaultChannel = %q", got.DefaultChannel)
	}
	if got.OwoBotID != Defaults().OwoBotID {
		t.Errorf("missing top-level key not defaulted: OwoBotID = %q, want %q", got.OwoBotID, Defaults().OwoBotID)
	}
	if got.SendMessageInterval != Defaults().SendMessageInterval {
		t.Errorf("missing key not defaulted: SendMessageInterval = %s, want %s",
			got.SendMessageInterval, Defaults().SendMessageInterval)
	}
}

// The defaults merge has to reach every level, not just the top two. An earlier
// version walked the JSON by hand and only filled two levels deep; anything
// below that fell back to Go zero values, which for a bool means "off".
func TestLoadFromFileMergeIsDeep(t *testing.T) {
	path := writeConfig(t, `{
		"schemaVersion": 2,
		"features": {"gamble": {"coinflip": {"enabled": true}}}
	}`)

	got := mustLoad(t, path)
	cf := got.Features.Gamble.Coinflip

	if !cf.Enabled {
		t.Fatal("explicit coinflip.enabled=true was lost")
	}
	if cf.StartValue != Defaults().Features.Gamble.Coinflip.StartValue {
		t.Errorf("startValue = %d, want default %d", cf.StartValue, Defaults().Features.Gamble.Coinflip.StartValue)
	}
	if !cf.Options.Heads {
		t.Error("coinflip.options.heads lost at depth 4 — bet side would be unset")
	}
	if cf.Cooldown.IsZero() {
		t.Error("coinflip.cooldown lost at depth 4")
	}
}

func TestLoadFromFileNestedMergeKeepsSiblings(t *testing.T) {
	path := writeConfig(t, `{"schemaVersion":2,"features":{"hunt":{"enabled":false}}}`)

	got := mustLoad(t, path)

	if got.Features.Hunt.Enabled {
		t.Error("explicit features.hunt.enabled=false was overwritten by the default")
	}
	if !got.Features.Battle.Enabled {
		t.Error("sibling features.battle lost during merge")
	}
	if !got.Features.Pray.Enabled {
		t.Error("sibling features.pray lost during merge")
	}
	if got.Features.Hunt.Delay.IsZero() {
		t.Error("features.hunt.delay lost when only enabled was supplied")
	}
}

func TestLoadFromFileChecklistDefaultsOff(t *testing.T) {
	// Existing configs predate the checklist loop; they must not silently start
	// sending checklist commands after upgrading.
	path := writeConfig(t, `{"schemaVersion":2,"features":{"hunt":{"enabled":true}}}`)

	if mustLoad(t, path).Features.Checklist.Enabled {
		t.Error("features.checklist defaulted to true; upgrade would change behavior")
	}
}

func TestLoadFromFileRejectsMalformedJSON(t *testing.T) {
	path := writeConfig(t, `{"features":`)

	if _, _, err := loadFromFile(path, ""); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestLoadFromFileMissingFileErrors(t *testing.T) {
	if _, _, err := loadFromFile(filepath.Join(t.TempDir(), "absent.json"), ""); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadFromFileReportsUnknownKeys(t *testing.T) {
	// A typo has to be reported: it otherwise reads as "left at its default".
	path := writeConfig(t, `{"schemaVersion":2,"features":{"hunt":{"enbaled":true}}}`)

	_, res, err := loadFromFile(path, "")
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	if len(res.Notes) == 0 {
		t.Fatal("no note for the misspelled key")
	}
	if !strings.Contains(res.Notes[0], "enbaled") {
		t.Errorf("note does not name the bad key: %q", res.Notes[0])
	}
}

func TestDurationRoundTrip(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`"1m30s"`), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.Std() != 90*time.Second {
		t.Errorf("parsed %s, want 1m30s", d)
	}

	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"1m30s"` {
		t.Errorf("marshalled %s, want \"1m30s\"", out)
	}
}

// A bare number is exactly the ambiguity the string form removes, so it must be
// rejected rather than guessed at.
func TestDurationRejectsBareNumber(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`5000`), &d); err == nil {
		t.Fatal("expected an error for a unitless number")
	}
}

// Most delays in a config are fixed, and writing those as an object put the
// same string on the page twice and buried the ones that really do jitter.
func TestRangeScalarShorthand(t *testing.T) {
	var r Range
	if err := json.Unmarshal([]byte(`"30s"`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Min.Std() != 30*time.Second || r.Max.Std() != 30*time.Second {
		t.Errorf("a bare duration should set both bounds, got %+v", r)
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"30s"` {
		t.Errorf("marshalled %s, want \"30s\"", out)
	}
}

func TestRangeObjectFormRoundTrips(t *testing.T) {
	var r Range
	if err := json.Unmarshal([]byte(`{"min":"10s","max":"30s"}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Min.Std() != 10*time.Second || r.Max.Std() != 30*time.Second {
		t.Errorf("bounds lost: %+v", r)
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `{"min":"10s","max":"30s"}` {
		t.Errorf("marshalled %s, want the object form", out)
	}
}

// A fixed range written the long way collapses on the next write. Every config
// in the wild uses the object form, so this is the path that tidies them up.
func TestRangeObjectWithEqualBoundsCollapses(t *testing.T) {
	var r Range
	if err := json.Unmarshal([]byte(`{"min":"5m","max":"5m"}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"5m"` {
		t.Errorf("marshalled %s, want \"5m\"", out)
	}
}

// Range's own UnmarshalJSON takes this subtree away from the DisallowUnknownFields
// probe in loadFromFile, so it has to do the check itself.
func TestRangeRejectsUnknownKey(t *testing.T) {
	var r Range
	if err := json.Unmarshal([]byte(`{"min":"5s","mx":"9s"}`), &r); err == nil {
		t.Fatal("expected an error for a misspelled bound; it would have loaded as max=0")
	}
}

func TestRangeRejectsBareNumber(t *testing.T) {
	var r Range
	if err := json.Unmarshal([]byte(`5000`), &r); err == nil {
		t.Fatal("expected an error for a unitless number")
	}
}

func TestRangePickStaysInBounds(t *testing.T) {
	r := Range{Min: secs(2), Max: secs(4)}
	for i := 0; i < 200; i++ {
		got := r.Pick()
		if got < 2*time.Second || got > 4*time.Second {
			t.Fatalf("Pick() = %v, outside [2s, 4s]", got)
		}
	}
}

func TestRangePickHandlesInvertedRange(t *testing.T) {
	r := Range{Min: secs(5), Max: secs(1)}
	if got := r.Pick(); got != 5*time.Second {
		t.Errorf("Pick() = %v on an inverted range, want the min (5s)", got)
	}
}

func TestNewLoaderCreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()

	l, res, err := NewLoader(dir, "229948970904846336", "someone", nil, nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	if !res.Created {
		t.Error("Created = false for a fresh config")
	}

	if _, err := os.Stat(filepath.Join(dir, "229948970904846336.json")); err != nil {
		t.Errorf("config file not created under the user ID: %v", err)
	}
	if l.Get().OwoBotID != Defaults().OwoBotID {
		t.Error("new loader did not start from defaults")
	}
	if l.Get().Label != "someone" {
		t.Errorf("label = %q, want the username", l.Get().Label)
	}
}

// The file the loader writes must be readable by the loader, including every
// custom duration type.
func TestNewLoaderWritesReloadableFile(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := NewLoader(dir, "229948970904846336", "someone", nil, nil); err != nil {
		t.Fatalf("NewLoader: %v", err)
	}

	got := mustLoad(t, filepath.Join(dir, "229948970904846336.json"))
	if got.Features.Hunt.Delay != Defaults().Features.Hunt.Delay {
		t.Errorf("hunt delay did not survive the round trip: %+v", got.Features.Hunt.Delay)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("the freshly written default config does not validate: %v", err)
	}
}

// Config files used to be keyed by username. An existing one gets adopted so
// upgrading does not silently start from defaults.
func TestNewLoaderAdoptsUsernameKeyedFile(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "someone.json")
	if err := os.WriteFile(old, []byte(`{"schemaVersion":2,"discord":{"prefix":"owo"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	l, res, err := NewLoader(dir, "229948970904846336", "someone", nil, nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}

	if l.Get().Prefix != "owo" {
		t.Errorf("prefix = %q, want the value from the adopted file", l.Get().Prefix)
	}
	if fileExists(old) {
		t.Error("username-keyed file still present after adoption")
	}
	if len(res.Notes) == 0 {
		t.Error("no note about the rename")
	}
}

func TestNewLoaderRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "229948970904846336.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":2,"discord":{"prefix":""}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := NewLoader(dir, "229948970904846336", "someone", nil, nil); err == nil {
		t.Fatal("expected NewLoader to reject a config that fails validation")
	}
}

// unwatchedLoader is a Loader over a real file with no watcher goroutine. The
// reload tests below drive reload() by hand; going through NewLoader would leave
// its watcher racing them for the same edit, and whichever won would decide
// whether the test saw the old settings or the new ones.
func unwatchedLoader(t *testing.T) *Loader {
	t.Helper()
	l := &Loader{
		path:     filepath.Join(t.TempDir(), "229948970904846336.json"),
		settings: Defaults(),
	}
	if err := l.save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	return l
}

// A bad edit to a running bot's config must not be applied.
func TestReloadKeepsPreviousSettingsOnInvalidEdit(t *testing.T) {
	l := unwatchedLoader(t)
	before := l.Get()

	// min > max is rejected by Validate.
	bad := `{"schemaVersion":2,"features":{"hunt":{"enabled":true,"delay":{"min":"30s","max":"5s"}}}}`
	if err := os.WriteFile(l.Path(), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	changed := false
	l.reload(func(old, new Settings) { changed = true })

	if changed {
		t.Error("onChange fired for a config that failed validation")
	}
	if l.Get().Features.Hunt.Delay != before.Features.Hunt.Delay {
		t.Error("invalid delay was applied to the running bot")
	}
}

func TestReloadAppliesValidEditAndReportsOldValue(t *testing.T) {
	l := unwatchedLoader(t)

	good := `{"schemaVersion":2,"discord":{"prefix":"owo"}}`
	if err := os.WriteFile(l.Path(), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotOld, gotNew Settings
	l.reload(func(old, new Settings) { gotOld, gotNew = old, new })

	if gotOld.Prefix != "w" {
		t.Errorf("old prefix = %q, want %q", gotOld.Prefix, "w")
	}
	if gotNew.Prefix != "owo" {
		t.Errorf("new prefix = %q, want %q", gotNew.Prefix, "owo")
	}
	if l.Get().Prefix != "owo" {
		t.Errorf("Get().Prefix = %q, want the reloaded value", l.Get().Prefix)
	}
}

// Whole minutes and hours are written without Go's redundant tail, and every
// form still parses back to the same value.
func TestDurationStringRoundTrips(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{500 * time.Millisecond, "500ms"},
		{15 * time.Second, "15s"},
		{5 * time.Minute, "5m"},
		{305 * time.Second, "5m5s"},
		{1000 * time.Second, "16m40s"},
		{time.Hour, "1h"},
		{90 * time.Minute, "90m"},
	}

	for _, tt := range tests {
		got := Duration(tt.d).String()
		if got != tt.want {
			t.Errorf("Duration(%v).String() = %q, want %q", tt.d, got, tt.want)
		}

		var back Duration
		if err := json.Unmarshal([]byte(`"`+got+`"`), &back); err != nil {
			t.Errorf("%q does not parse back: %v", got, err)
			continue
		}
		if back.Std() != tt.d {
			t.Errorf("%q round-tripped to %v, want %v", got, back.Std(), tt.d)
		}
	}
}

// Every duration in the shipped defaults has to survive a save/load cycle
// unchanged, or a config file would drift each time it is rewritten.
func TestDefaultsSurviveMarshalRoundTrip(t *testing.T) {
	data, err := json.Marshal(Defaults())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back Settings
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != Defaults() {
		t.Error("defaults changed across a marshal round trip")
	}
}
