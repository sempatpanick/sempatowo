package config

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLoadFromFileFillsMissingKeysFromDefaults(t *testing.T) {
	// A minimal config must still come back fully populated.
	path := writeConfig(t, `{"channels":{"hunt":"123"}}`)

	got, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}

	if got.Channels.Hunt != "123" {
		t.Errorf("user value lost: hunt = %q, want %q", got.Channels.Hunt, "123")
	}
	if got.OwoID != Defaults().OwoID {
		t.Errorf("missing top-level key not defaulted: OwoID = %q, want %q", got.OwoID, Defaults().OwoID)
	}
	if got.Interval.SendMessage != Defaults().Interval.SendMessage {
		t.Errorf("missing nested key not defaulted: SendMessage = %d, want %d",
			got.Interval.SendMessage, Defaults().Interval.SendMessage)
	}
}

func TestLoadFromFileNestedMergeKeepsSiblings(t *testing.T) {
	// Supplying one key inside "status" must not blank out the others.
	path := writeConfig(t, `{"status":{"hunt":false}}`)

	got, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}

	if got.Status.Hunt {
		t.Error("explicit status.hunt=false was overwritten by the default")
	}
	if !got.Status.Battle {
		t.Error("sibling status.battle lost during nested merge")
	}
	if !got.Status.Pray {
		t.Error("sibling status.pray lost during nested merge")
	}
}

func TestLoadFromFileChecklistDefaultsOff(t *testing.T) {
	// Existing configs predate status.checklist; they must not silently start
	// sending checklist commands after upgrading.
	path := writeConfig(t, `{"status":{"hunt":true}}`)

	got, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	if got.Status.Checklist {
		t.Error("status.checklist defaulted to true; upgrade would change behavior")
	}
}

func TestLoadFromFileRejectsMalformedJSON(t *testing.T) {
	path := writeConfig(t, `{"channels":`)

	if _, err := loadFromFile(path); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestLoadFromFileMissingFileErrors(t *testing.T) {
	if _, err := loadFromFile(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestNewLoaderCreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()

	l, err := NewLoader(dir, "someone", nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "someone.json")); err != nil {
		t.Errorf("config file not created: %v", err)
	}
	if l.Get().OwoID != Defaults().OwoID {
		t.Error("new loader did not start from defaults")
	}
}

func TestLoaderGetSetRoundTrip(t *testing.T) {
	l, err := NewLoader(t.TempDir(), "someone", nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}

	s := l.Get()
	s.Channels.Hunt = "999"
	l.Set(s)

	if got := l.Get().Channels.Hunt; got != "999" {
		t.Errorf("Get after Set = %q, want %q", got, "999")
	}
}

// normalizeDelays only fills in zero values; it deliberately does not reorder
// an inverted min/max — farm.actionDelay swaps those at the point of use.
func TestNormalizeDelaysFillsZerosFromDefaults(t *testing.T) {
	s := Defaults()
	s.Interval.Hunt = ActionDelay{}

	normalizeDelays(&s)

	if s.Interval.Hunt.MinDelay != Defaults().Interval.Hunt.MinDelay {
		t.Errorf("MinDelay = %d, want default %d",
			s.Interval.Hunt.MinDelay, Defaults().Interval.Hunt.MinDelay)
	}
	if s.Interval.Hunt.MaxDelay != Defaults().Interval.Hunt.MaxDelay {
		t.Errorf("MaxDelay = %d, want default %d",
			s.Interval.Hunt.MaxDelay, Defaults().Interval.Hunt.MaxDelay)
	}
}

func TestNormalizeDelaysPrefersLegacyFastestSlowest(t *testing.T) {
	// Older configs expressed the range as slowest/fastest.
	s := Defaults()
	s.Interval.Hunt = ActionDelay{SlowestTime: 7000, FastestTime: 3000}

	normalizeDelays(&s)

	if s.Interval.Hunt.MinDelay != 7000 {
		t.Errorf("MinDelay = %d, want 7000 from SlowestTime", s.Interval.Hunt.MinDelay)
	}
	if s.Interval.Hunt.MaxDelay != 3000 {
		t.Errorf("MaxDelay = %d, want 3000 from FastestTime", s.Interval.Hunt.MaxDelay)
	}
}
