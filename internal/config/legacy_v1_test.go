package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const v1Example = `{
  "schemaVersion": 1,
  "label": "someone",
  "typing": false,
  "prefix": "owo",
  "owoBotId": "408785106942164992",
  "defaultChannel": "123456789012345678",
  "sendMessageInterval": "3s",
  "stopWhenChecklistDone": true,
  "trackBalance": false,
  "features": {
    "hunt": {"enabled": false, "delay": {"min": "40s", "max": "90s"}},
    "checklist": {"enabled": true, "delay": "20m"},
    "gamble": {"allottedAmount": 777}
  }
}`

func TestMigrateV1MovesTheRegroupedKeys(t *testing.T) {
	got, notes, err := migrateV1([]byte(v1Example))
	if err != nil {
		t.Fatalf("migrateV1: %v", err)
	}

	if got.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", got.SchemaVersion, SchemaVersion)
	}
	if got.Schema != SchemaFileName {
		t.Errorf("$schema = %q, want %q", got.Schema, SchemaFileName)
	}

	if got.Discord.Prefix != "owo" {
		t.Errorf("discord.prefix = %q, want %q", got.Discord.Prefix, "owo")
	}
	if got.Discord.DefaultChannel != "123456789012345678" {
		t.Errorf("discord.defaultChannel = %q", got.Discord.DefaultChannel)
	}
	if got.Discord.OwoBotID != "408785106942164992" {
		t.Errorf("discord.owoBotId = %q", got.Discord.OwoBotID)
	}
	if got.Humanize.SendMessageInterval.Std() != 3*time.Second {
		t.Errorf("humanize.sendMessageInterval = %s, want 3s", got.Humanize.SendMessageInterval)
	}
	if !got.Features.Checklist.StopFarmingWhenDone {
		t.Error("stopWhenChecklistDone did not become features.checklist.stopFarmingWhenDone")
	}
	if len(notes) == 0 {
		t.Error("migration produced no note explaining where the keys went")
	}
}

// A plain bool field would read an absent key and an explicit false the same
// way, and the user's "off" would silently become the default "on".
func TestMigrateV1KeepsExplicitFalse(t *testing.T) {
	got, _, err := migrateV1([]byte(v1Example))
	if err != nil {
		t.Fatalf("migrateV1: %v", err)
	}
	if got.Typing {
		t.Error(`"typing": false was lost and defaulted back to true`)
	}
	if got.TrackBalance {
		t.Error(`"trackBalance": false was lost`)
	}
}

func TestMigrateV1DefaultsWhatTheFileOmits(t *testing.T) {
	got, _, err := migrateV1([]byte(`{"schemaVersion":1,"prefix":"x"}`))
	if err != nil {
		t.Fatalf("migrateV1: %v", err)
	}
	if got.Typing != Defaults().Typing {
		t.Error("an omitted typing key did not fall back to the default")
	}
	if got.OwoBotID != Defaults().OwoBotID {
		t.Error("an omitted owoBotId did not fall back to the default")
	}
}

// Everything that did not move must survive untouched.
func TestMigrateV1PassesFeaturesThrough(t *testing.T) {
	got, _, err := migrateV1([]byte(v1Example))
	if err != nil {
		t.Fatalf("migrateV1: %v", err)
	}
	if got.Label != "someone" {
		t.Errorf("label = %q", got.Label)
	}
	if got.Features.Hunt.Enabled {
		t.Error("features.hunt.enabled=false was lost")
	}
	if got.Features.Hunt.Delay.Min.Std() != 40*time.Second {
		t.Errorf("features.hunt.delay.min = %s, want 40s", got.Features.Hunt.Delay.Min)
	}
	if got.Features.Checklist.Delay.Min.Std() != 20*time.Minute {
		t.Errorf("features.checklist.delay = %s, want 20m", got.Features.Checklist.Delay.Min)
	}
	if got.Features.Gamble.AllottedAmount != 777 {
		t.Errorf("features.gamble.allottedAmount = %d, want 777", got.Features.Gamble.AllottedAmount)
	}
}

// The moved keys are not v2 keys, so the unknown-field probe would flag every
// one of them on the single load that fixes them.
func TestLoadFromFileDoesNotReportMovedKeysAsTypos(t *testing.T) {
	path := writeConfig(t, v1Example)

	_, res, err := loadFromFile(path, "")
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	if !res.Migrated || res.FromVersion != 1 {
		t.Fatalf("Migrated = %v, FromVersion = %d; want true, 1", res.Migrated, res.FromVersion)
	}
	for _, note := range res.Notes {
		if strings.Contains(note, "unknown field") {
			t.Errorf("a moved key was reported as a typo: %q", note)
		}
	}
}

func TestNewLoaderBacksUpByVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "229948970904846336.json")
	if err := os.WriteFile(path, []byte(v1Example), 0o644); err != nil {
		t.Fatal(err)
	}

	l, res, err := NewLoader(dir, "229948970904846336", "someone", nil, nil)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}

	if !res.Migrated || res.FromVersion != 1 {
		t.Fatalf("Migrated = %v, FromVersion = %d; want true, 1", res.Migrated, res.FromVersion)
	}
	if filepath.Base(res.BackupPath) != "229948970904846336.json.v1.bak" {
		t.Errorf("backup = %q, want a .v1.bak", filepath.Base(res.BackupPath))
	}
	if !fileExists(res.BackupPath) {
		t.Error("the pre-migration file was not preserved")
	}

	// The rewritten file has to load cleanly as v2, without the old keys.
	reloaded, res2, err := loadFromFile(l.Path(), "someone")
	if err != nil {
		t.Fatalf("reloading the migrated file: %v", err)
	}
	if res2.Migrated {
		t.Error("the migrated file still reads as an old version")
	}
	if len(res2.Notes) != 0 {
		t.Errorf("the migrated file reports notes: %v", res2.Notes)
	}
	if reloaded.Prefix != "owo" || !reloaded.Features.Checklist.StopFarmingWhenDone {
		t.Errorf("settings did not survive the rewrite: %+v", reloaded.Discord)
	}
}
