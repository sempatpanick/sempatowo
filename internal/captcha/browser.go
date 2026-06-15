package captcha

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func isIsolated() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("BROWSER_ISOLATED")))
	return !(raw == "false" || raw == "0" || raw == "no" || raw == "off")
}

// OpenBrowserAsync opens the captcha page without blocking the caller.
// stillNeeded is checked after acquiring a shared-browser slot (non-isolated mode).
func OpenBrowserAsync(url, profileLabel string, stillNeeded func() bool) {
	go func() {
		if BrowserQueueEnabled() {
			fmt.Printf("[browser] waiting for captcha browser slot [%s]...\n", profileLabel)
			AcquireBrowserSlot(profileLabel)
			if stillNeeded != nil && !stillNeeded() {
				fmt.Printf("[browser] captcha already solved — skipping open [%s]\n", profileLabel)
				ReleaseBrowserSlot(profileLabel)
				return
			}
			fmt.Printf("[browser] captcha browser slot acquired [%s]\n", profileLabel)
		}

		if err := openBrowser(url, profileLabel); err != nil {
			fmt.Printf("[browser] could not open browser [%s]: %v — solve manually: %s\n", profileLabel, err, url)
			return
		}
		fmt.Printf("[browser] opened captcha [%s]: %s\n", profileLabel, url)
	}()
}

func openBrowser(url, profileLabel string) error {
	if isIsolated() {
		return openIsolatedChrome(url, profileLabel)
	}
	return openDefaultBrowser(url)
}

func openDefaultBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func openIsolatedChrome(url, profileLabel string) error {
	executable := findChrome()
	if executable == "" {
		fmt.Println("[browser] no Chrome found — falling back to default browser")
		return openDefaultBrowser(url)
	}

	profilesDir := strings.TrimSpace(os.Getenv("BROWSER_PROFILES_DIR"))
	if profilesDir == "" {
		profilesDir = "./browser-profiles"
	}
	profilePath, err := filepath.Abs(filepath.Join(profilesDir, sanitize(profileLabel)))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		fmt.Printf("[browser] could not create profile dir: %v — using default browser\n", err)
		return openDefaultBrowser(url)
	}

	args := []string{
		"--user-data-dir=" + profilePath,
		"--profile-directory=Default",
		"--new-window",
		"--no-first-run",
		"--no-default-browser-check",
	}

	if ext := resolveExtensionPath(profilesDir); ext != "" {
		args = append(args,
			"--disable-extensions-except="+ext,
			"--load-extension="+ext,
		)
	} else {
		fmt.Println("[browser] extension not found — opening without captcha extension")
	}
	args = append(args, url)

	cmd := exec.Command(executable, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return openDefaultBrowser(url)
	}
	// Detach so Chrome keeps running independently.
	go cmd.Wait()
	return nil
}

func resolveExtensionPath(profilesDir string) string {
	extID := strings.TrimSpace(os.Getenv("BROWSER_EXTENSION_ID"))
	candidates := []string{}
	if p := strings.TrimSpace(os.Getenv("BROWSER_EXTENSION_PATH")); p != "" {
		candidates = append(candidates, p)
	}
	if extID != "" {
		candidates = append(candidates, filepath.Join(profilesDir, ".extension-cache", extID))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if hasManifest(candidate) {
			return candidate
		}
		entries, err := os.ReadDir(candidate)
		if err != nil {
			continue
		}
		var versionDirs []string
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(candidate, e.Name())
			if hasManifest(p) {
				versionDirs = append(versionDirs, p)
			}
		}
		sort.Strings(versionDirs)
		if len(versionDirs) > 0 {
			return versionDirs[len(versionDirs)-1]
		}
	}
	return ""
}

func hasManifest(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "manifest.json"))
	return err == nil && !info.IsDir()
}

func findChrome() string {
	if p := strings.TrimSpace(os.Getenv("BROWSER_EXECUTABLE")); p != "" && fileExists(p) {
		return p
	}
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
		}
	case "darwin":
		candidates = []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
	default:
		candidates = []string{"/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium", "/usr/bin/chromium-browser"}
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode()&fs.ModeType == 0
}

func sanitize(s string) string {
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, s)
	cleaned = strings.TrimLeft(cleaned, ".")
	if cleaned == "" {
		return "default"
	}
	return cleaned
}
