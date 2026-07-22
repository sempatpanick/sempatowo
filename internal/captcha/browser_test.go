package captcha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/semptpanick/sempatowo/internal/config"
)

// sanitize builds a directory name from a Discord username, which can contain
// anything — including path separators and leading dots.
func TestSanitize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"sempatpanick", "sempatpanick"},
		{"user.name-1_2", "user.name-1_2"},
		// Dots survive, separators do not, and leading dots are trimmed — so
		// what is left of a traversal attempt is inert filename text.
		{"../../etc/passwd", "_.._etc_passwd"},
		{"a/b\\c", "a_b_c"},
		// Mapping is per rune, not per byte.
		{"日本語", "___"},
		{"", "default"},
		{"...", "default"},
		{".hidden", "hidden"},
	}

	for _, tt := range tests {
		if got := sanitize(tt.in); got != tt.want {
			t.Errorf("sanitize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A sanitized label must never escape the profiles directory.
func TestSanitizeKeepsProfilesContained(t *testing.T) {
	base := "/var/browser-profiles"
	for _, label := range []string{"../../etc", "..", "a/../../b", "/absolute"} {
		got := filepath.Join(base, sanitize(label))
		if !strings.HasPrefix(filepath.Clean(got), base+"/") {
			t.Errorf("label %q escaped the base dir: %q", label, got)
		}
	}
}

func TestFindChromePrefersConfigured(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "chrome")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := findChrome(fake); got != fake {
		t.Errorf("findChrome(%q) = %q, want the configured path", fake, got)
	}
}

// A configured path that does not exist must fall through to the usual install
// locations rather than being returned and failing at exec time.
func TestFindChromeIgnoresMissingConfigured(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	if got := findChrome(missing); got == missing {
		t.Error("findChrome returned a path that does not exist")
	}
}

func TestResolveExtensionPathDirectHit(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir)

	env := config.BrowserEnv{ExtensionPath: dir}
	if got := resolveExtensionPath(dir, env); got != dir {
		t.Errorf("resolveExtensionPath = %q, want %q", got, dir)
	}
}

// Chrome unpacks extensions into per-version subdirectories; the newest wins.
func TestResolveExtensionPathPicksNewestVersion(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"1.0.0.6_0", "1.108.1_0", "1.2.0_0"} {
		writeManifest(t, filepath.Join(dir, v))
	}

	got := resolveExtensionPath(dir, config.BrowserEnv{ExtensionPath: dir})
	want := filepath.Join(dir, "1.2.0_0") // lexicographically last
	if got != want {
		t.Errorf("resolveExtensionPath = %q, want %q", got, want)
	}
}

func TestResolveExtensionPathFallsBackToCache(t *testing.T) {
	profiles := t.TempDir()
	const extID = "nmmhkkegccagdldgiimedpiccmgmieda"
	cached := filepath.Join(profiles, ".extension-cache", extID)
	writeManifest(t, cached)

	got := resolveExtensionPath(profiles, config.BrowserEnv{ExtensionID: extID})
	if got != cached {
		t.Errorf("resolveExtensionPath = %q, want the cached copy %q", got, cached)
	}
}

func TestResolveExtensionPathMissing(t *testing.T) {
	dir := t.TempDir()

	if got := resolveExtensionPath(dir, config.BrowserEnv{}); got != "" {
		t.Errorf("resolveExtensionPath = %q, want empty when nothing is configured", got)
	}
	// A configured path with no manifest anywhere under it is not an extension.
	if got := resolveExtensionPath(dir, config.BrowserEnv{ExtensionPath: dir}); got != "" {
		t.Errorf("resolveExtensionPath = %q, want empty for a directory with no manifest", got)
	}
}

func TestBrowserQueueEnabled(t *testing.T) {
	if BrowserQueueEnabled(true) {
		t.Error("isolated profiles need no queue — each account has its own browser")
	}
	if !BrowserQueueEnabled(false) {
		t.Error("a shared browser must be queued so accounts take turns")
	}
}

// Acquire and Release are no-ops in isolated mode; calling them must not block
// or leave the shared slot held.
func TestBrowserSlotNoopWhenIsolated(t *testing.T) {
	done := make(chan struct{})
	go func() {
		AcquireBrowserSlot("a", true)
		AcquireBrowserSlot("b", true)
		ReleaseBrowserSlot("a", true)
		ReleaseBrowserSlot("b", true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("isolated-mode slot calls blocked")
	}
}

func writeManifest(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
}
