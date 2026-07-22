package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// clearEnv blanks every variable LoadEnv reads, so a test starts from a known
// state regardless of the developer's own shell.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TOKEN", "OCR_API_KEY", "NOTIFICATIONS", "DATA_DIR",
		"CAPTCHA_API_KEY", "CAPTCHA_SERVICE", "CAPTCHA_SOLVE_TIMEOUT",
		"BROWSER_ISOLATED", "BROWSER_PROFILES_DIR", "BROWSER_EXECUTABLE",
		"BROWSER_EXTENSION_ID", "BROWSER_EXTENSION_PATH",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadEnvDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", "abc")

	e, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}

	if len(e.Tokens) != 1 || e.Tokens[0] != "abc" {
		t.Errorf("Tokens = %v, want [abc]", e.Tokens)
	}
	if e.Captcha.Service != defaultCaptchaSvc {
		t.Errorf("Service = %q, want %q", e.Captcha.Service, defaultCaptchaSvc)
	}
	if e.Captcha.SolveTimeout != 90*time.Second {
		t.Errorf("SolveTimeout = %s, want 90s", e.Captcha.SolveTimeout)
	}
	if e.Captcha.AutoSolveEnabled() {
		t.Error("AutoSolveEnabled = true with no API key")
	}
	if e.OCRAPIKey != defaultOCRKey {
		t.Errorf("OCRAPIKey = %q, want the shared test key", e.OCRAPIKey)
	}
	if !e.Notifications {
		t.Error("Notifications should default on")
	}
	if !e.Browser.Isolated {
		t.Error("Browser.Isolated should default on")
	}
}

func TestLoadEnvSplitsTokens(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", " one , two,,three ")

	e, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	want := []string{"one", "two", "three"}
	if len(e.Tokens) != len(want) {
		t.Fatalf("Tokens = %v, want %v", e.Tokens, want)
	}
	for i := range want {
		if e.Tokens[i] != want[i] {
			t.Errorf("Tokens[%d] = %q, want %q", i, e.Tokens[i], want[i])
		}
	}
}

func TestLoadEnvRequiresToken(t *testing.T) {
	clearEnv(t)

	if _, err := LoadEnv(); err == nil {
		t.Fatal("expected an error when TOKEN is unset")
	}
}

// The whole point of loading the environment up front: a typo fails now rather
// than the first time a captcha appears.
func TestLoadEnvRejectsUnknownCaptchaService(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", "abc")
	t.Setenv("CAPTCHA_SERVICE", "capsolvr")

	_, err := LoadEnv()
	if err == nil {
		t.Fatal("expected an error for an unknown solver")
	}
	if !strings.Contains(err.Error(), "capsolvr") {
		t.Errorf("error does not name the bad value: %v", err)
	}
}

func TestLoadEnvRejectsBadTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", "abc")

	for _, bad := range []string{"soon", "-5", "0"} {
		t.Setenv("CAPTCHA_SOLVE_TIMEOUT", bad)
		if _, err := LoadEnv(); err == nil {
			t.Errorf("CAPTCHA_SOLVE_TIMEOUT=%q was accepted", bad)
		}
	}
}

func TestLoadEnvReportsEveryProblemAtOnce(t *testing.T) {
	clearEnv(t)
	t.Setenv("CAPTCHA_SERVICE", "nope")
	t.Setenv("CAPTCHA_SOLVE_TIMEOUT", "later")

	err := func() error { _, err := LoadEnv(); return err }()
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"TOKEN", "CAPTCHA_SERVICE", "CAPTCHA_SOLVE_TIMEOUT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

func TestLoadEnvBoolParsing(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want bool
	}{
		{"", true}, {"false", false}, {"0", false}, {"no", false},
		{"off", false}, {"FALSE", false}, {"true", true}, {"1", true},
	} {
		clearEnv(t)
		t.Setenv("TOKEN", "abc")
		t.Setenv("NOTIFICATIONS", tt.raw)

		e, err := LoadEnv()
		if err != nil {
			t.Fatalf("LoadEnv: %v", err)
		}
		if e.Notifications != tt.want {
			t.Errorf("NOTIFICATIONS=%q gave %v, want %v", tt.raw, e.Notifications, tt.want)
		}
	}
}

// Every writable directory hangs off one root, so there is a single thing to
// back up or delete.
func TestLoadEnvDirsHangOffOneRoot(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", "abc")
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)

	e, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}

	if e.Dirs.Config != filepath.Join(root, "config") {
		t.Errorf("Config = %q", e.Dirs.Config)
	}
	if e.Dirs.Data != filepath.Join(root, "data") {
		t.Errorf("Data = %q", e.Dirs.Data)
	}
	if e.Dirs.BrowserProfiles != filepath.Join(root, "browser-profiles") {
		t.Errorf("BrowserProfiles = %q", e.Dirs.BrowserProfiles)
	}

	if err := e.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, dir := range []string{e.Dirs.Config, e.Dirs.Data, e.Dirs.BrowserProfiles} {
		if !dirExists(dir) {
			t.Errorf("%s was not created", dir)
		}
	}
}

// An explicit BROWSER_PROFILES_DIR still wins, for people pointing at a profile
// tree they already have.
func TestLoadEnvBrowserProfilesOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("TOKEN", "abc")
	t.Setenv("DATA_DIR", t.TempDir())
	custom := t.TempDir()
	t.Setenv("BROWSER_PROFILES_DIR", custom)

	e, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	if e.Dirs.BrowserProfiles != custom {
		t.Errorf("BrowserProfiles = %q, want the override %q", e.Dirs.BrowserProfiles, custom)
	}
	if e.Browser.ProfilesDir != custom {
		t.Errorf("Browser.ProfilesDir = %q, want %q", e.Browser.ProfilesDir, custom)
	}
}
