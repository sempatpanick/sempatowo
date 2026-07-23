package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The real config file shipped with the project, so the migration is exercised
// against the shape people actually have on disk.
const legacyExample = `{
  "typing": true,
  "prefix": "w",
  "status": {
    "hunt": true, "battle": true, "zoo": false, "pray": true, "curse": false,
    "lootbox": false, "lootbox_fabled": false, "crate": false, "cookie": false,
    "gems": false, "inventory": false, "quest": false
  },
  "interval": {
    "send_message": 500,
    "zoo": 300000, "pray": 305000, "curse": 305000,
    "hunt": {"minDelay": 15000, "maxDelay": 20000},
    "battle": {"minDelay": 15000, "maxDelay": 20000},
    "inventory": 300000, "checklist": 1000000,
    "quest": {"owo": 32000, "check": 60000}
  },
  "channels": {"hunt": "1513744333579489310", "quest": "1513744333579489310"},
  "target": {"pray": "", "curse": "", "cookie": "469369739131617291"},
  "owoId": "408785106942164992",
  "checklist_completed": false,
  "cashCheck": false,
  "autoDaily": true,
  "allowAutoQuest": false,
  "ocrApi": "helloworld",
  "huntbot": {
    "enabled": true, "cashToSpend": 10000,
    "upgrader": {"enabled": false, "sleeptime": [10, 15],
      "traits": {"efficiency": true, "duration": true, "cost": true, "gain": true, "exp": true, "radar": true},
      "weights": {"efficiency": 4, "duration": 2, "cost": 5, "gain": 4, "exp": 3, "radar": 1}}
  },
  "gamble": {
    "allottedAmount": 10000,
    "goalSystem": {"enabled": false, "amount": 30000},
    "coinflip": {"enabled": false, "startValue": 200, "multiplierOnLose": 2, "cooldown": [16, 18],
      "options": {"heads": true, "tails": false}},
    "slots": {"enabled": false, "startValue": 200, "multiplierOnLose": 2, "cooldown": [16, 18]},
    "blackjack": {"enabled": false, "startValue": 200, "multiplierOnLose": 2, "cooldown": [16, 18]}
  },
  "autoQuest": {
    "enabled": false,
    "helpChannel": {"postInHelpChannel": false, "channelId": ""},
    "enableCommandsToCompleteQuest": true, "helpOthers": true,
    "checkCooldown": [10, 30]
  }
}`

func TestFileVersionReadsSchemaVersion(t *testing.T) {
	if got := fileVersion([]byte(legacyExample)); got != 0 {
		t.Errorf("fileVersion = %d for a file with no schemaVersion, want 0", got)
	}
	if got := fileVersion([]byte(`{"schemaVersion":1}`)); got != 1 {
		t.Errorf("fileVersion = %d, want 1", got)
	}
	// Unparseable JSON must not read as "old": the caller's own decode should
	// produce the real syntax error rather than a misleading migration attempt.
	if got := fileVersion([]byte(`{not json`)); got != SchemaVersion {
		t.Errorf("fileVersion = %d for malformed JSON, want %d", got, SchemaVersion)
	}
}

func TestMigrateLegacyPreservesBehaviour(t *testing.T) {
	got, notes, err := migrateLegacy([]byte(legacyExample))
	if err != nil {
		t.Fatalf("migrateLegacy: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("unexpected notes for a config using the default ocr key: %v", notes)
	}

	if got.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", got.SchemaVersion, SchemaVersion)
	}
	if got.OwoBotID != "408785106942164992" {
		t.Errorf("owoId lost: %q", got.OwoBotID)
	}
	if got.DefaultChannel != "1513744333579489310" {
		t.Errorf("channels.hunt did not become defaultChannel: %q", got.DefaultChannel)
	}
	// quest channel matched the hunt channel, so it should collapse to the default.
	if got.Features.Quest.Channel != "" {
		t.Errorf("quest channel = %q, want empty (same as default)", got.Features.Quest.Channel)
	}
	if got.SendMessageInterval.Std() != 500*time.Millisecond {
		t.Errorf("send_message = %s, want 500ms", got.SendMessageInterval)
	}
	if got.TrackBalance {
		t.Error("cashCheck=false did not carry over to trackBalance")
	}
	if !got.Features.Daily.Enabled {
		t.Error("autoDaily=true did not carry over to features.daily")
	}

	f := got.Features
	if !f.Hunt.Enabled || f.Hunt.Delay.Min.Std() != 15*time.Second || f.Hunt.Delay.Max.Std() != 20*time.Second {
		t.Errorf("hunt migrated wrong: %+v", f.Hunt)
	}
	if f.Pray.Delay.Min.Std() != 305*time.Second || f.Pray.Delay.Max.Std() != 305*time.Second {
		t.Errorf("pray interval should become a fixed range, got %+v", f.Pray.Delay)
	}
	if f.Lootbox.Enabled || f.Lootbox.Fabled {
		t.Errorf("lootbox flags migrated wrong: %+v", f.Lootbox)
	}
	if f.Cookie.Target != "469369739131617291" {
		t.Errorf("cookie target lost: %q", f.Cookie.Target)
	}
	if !f.Huntbot.Enabled || f.Huntbot.Upgrader.Enabled {
		t.Errorf("huntbot flags migrated wrong: %+v", f.Huntbot)
	}
	if f.Huntbot.Upgrader.Cooldown.Min.Std() != 10*time.Second || f.Huntbot.Upgrader.Cooldown.Max.Std() != 15*time.Second {
		t.Errorf("sleeptime did not become a cooldown range: %+v", f.Huntbot.Upgrader.Cooldown)
	}
	if f.Quest.OwoDelay.Min.Std() != 32*time.Second {
		t.Errorf("quest owo interval lost: %+v", f.Quest.OwoDelay)
	}
	if f.Quest.Auto.CheckCooldown.Min.Std() != 10*time.Second || f.Quest.Auto.CheckCooldown.Max.Std() != 30*time.Second {
		t.Errorf("autoQuest checkCooldown lost: %+v", f.Quest.Auto.CheckCooldown)
	}
	min, max := f.Gamble.Slots.CooldownSec()
	if min != 16 || max != 18 {
		t.Errorf("slots cooldown = %g/%g, want 16/18", min, max)
	}

	if err := got.Validate(); err != nil {
		t.Errorf("migrated config does not validate: %v", err)
	}
}

// sleeptime could be a bare number as well as a pair.
func TestMigrateLegacyScalarSleeptime(t *testing.T) {
	got, _, err := migrateLegacy([]byte(`{"huntbot":{"upgrader":{"sleeptime": 12}}}`))
	if err != nil {
		t.Fatalf("migrateLegacy: %v", err)
	}
	cd := got.Features.Huntbot.Upgrader.Cooldown
	if cd.Min.Std() != 12*time.Second || cd.Max.Std() != 12*time.Second {
		t.Errorf("scalar sleeptime = %+v, want a fixed 12s range", cd)
	}
}

// The even older slowest/fastest spelling, which normalizeDelays used to patch.
func TestMigrateLegacySlowestFastest(t *testing.T) {
	got, _, err := migrateLegacy([]byte(`{"interval":{"hunt":{"slowestTime":7000,"fastestTime":3000}}}`))
	if err != nil {
		t.Fatalf("migrateLegacy: %v", err)
	}
	d := got.Features.Hunt.Delay
	if d.Min.Std() != 7*time.Second || d.Max.Std() != 3*time.Second {
		t.Errorf("delay = %+v, want 7s/3s from slowestTime/fastestTime", d)
	}
}

// A partial legacy file merges against the old defaults, not the new ones, so
// values the user never wrote keep the meaning they had.
func TestMigrateLegacyPartialFileUsesOldDefaults(t *testing.T) {
	got, _, err := migrateLegacy([]byte(`{"status":{"hunt":false}}`))
	if err != nil {
		t.Fatalf("migrateLegacy: %v", err)
	}
	if got.Features.Hunt.Enabled {
		t.Error("explicit hunt=false lost")
	}
	if !got.Features.Battle.Enabled {
		t.Error("battle defaulted off; the old default was on")
	}
	if got.Features.Hunt.Delay.Min.Std() != 50*time.Second {
		t.Errorf("hunt delay = %+v, want the old 50s default", got.Features.Hunt.Delay)
	}
}

func TestMigrateLegacyReportsMovedOCRKey(t *testing.T) {
	_, notes, err := migrateLegacy([]byte(`{"ocrApi":"K123456"}`))
	if err != nil {
		t.Fatalf("migrateLegacy: %v", err)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "OCR_API_KEY") {
		t.Errorf("notes = %v, want one telling the user where the key went", notes)
	}
}

func TestNewLoaderMigratesAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "229948970904846336.json")
	if err := os.WriteFile(path, []byte(legacyExample), 0o644); err != nil {
		t.Fatal(err)
	}

	l, res, err := NewLoader(dir, "229948970904846336", "tester", nil, nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	if !res.Migrated {
		t.Fatal("Migrated = false for a legacy file")
	}
	if l.Get().Label != "tester" {
		t.Errorf("label = %q, want the username", l.Get().Label)
	}

	backup, err := os.ReadFile(res.BackupPath)
	if err != nil {
		t.Fatalf("backup not written: %v", err)
	}
	if !strings.Contains(string(backup), `"send_message"`) {
		t.Error("backup does not contain the original file")
	}
}

// Inspect must never write: it is what -check-config runs, and checking a
// config file should not change anything on disk.
func TestInspectDoesNotRewriteLegacyFile(t *testing.T) {
	path := writeConfig(t, legacyExample)

	if _, res, err := Inspect(path); err != nil {
		t.Fatalf("Inspect: %v", err)
	} else if !res.Migrated {
		t.Error("Inspect did not report the file as legacy")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `"send_message"`) {
		t.Error("Inspect rewrote the file it was only asked to check")
	}
	if fileExists(path + ".v0.bak") {
		t.Error("Inspect wrote a backup file")
	}
}

// Migrating rewrites the file, and the rewritten file must load cleanly the
// next time without migrating again.
func TestMigrationIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "229948970904846336.json")
	if err := os.WriteFile(path, []byte(legacyExample), 0o644); err != nil {
		t.Fatal(err)
	}

	first, res, err := NewLoader(dir, "229948970904846336", "tester", nil, nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	if !res.Migrated {
		t.Fatal("first load did not migrate")
	}

	second, res2, err := NewLoader(dir, "229948970904846336", "tester", nil, nil)
	if err != nil {
		t.Fatalf("second NewLoader: %v", err)
	}
	if res2.Migrated {
		t.Error("second load migrated again — the rewrite did not take")
	}
	if first.Get().Features.Hunt.Delay != second.Get().Features.Hunt.Delay {
		t.Error("settings changed between the migrating load and the plain one")
	}
}
