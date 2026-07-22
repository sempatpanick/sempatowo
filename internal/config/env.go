package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Env holds everything the process reads from the environment.
//
// It is loaded and validated once at startup and passed down, rather than each
// package calling os.Getenv at the moment it happens to need a value. A
// mistyped CAPTCHA_SERVICE used to surface the first time a captcha appeared —
// which is exactly when nobody is watching. Now it fails before the bot
// connects, and the full set of knobs is one struct to read.
type Env struct {
	// Tokens are the Discord user tokens, one per account.
	Tokens []string
	// OCRAPIKey is the OCR.space key used for quest image parsing. It lives in
	// the environment rather than the config file because it is a credential,
	// like every other secret here.
	OCRAPIKey string
	// Notifications enables desktop notifications on captcha.
	Notifications bool

	Captcha CaptchaEnv
	Browser BrowserEnv
	Dirs    Dirs
}

type CaptchaEnv struct {
	APIKey       string
	Service      string
	SolveTimeout time.Duration
}

// AutoSolveEnabled reports whether an automatic solver is configured.
func (c CaptchaEnv) AutoSolveEnabled() bool { return c.APIKey != "" }

type BrowserEnv struct {
	Isolated      bool
	ProfilesDir   string
	Executable    string
	ExtensionID   string
	ExtensionPath string
}

// Dirs are the writable locations the process uses. They all hang off one root
// so there is a single directory to back up, gitignore, or delete.
type Dirs struct {
	Root            string
	Config          string
	Data            string
	BrowserProfiles string
}

const (
	defaultOCRKey     = "helloworld"
	defaultCaptchaSvc = "capsolver"
	defaultRoot       = "var"
)

var captchaServices = []string{"capsolver", "capmonster", "2captcha"}

// LoadEnv reads and validates the environment. It reports every problem at
// once so a misconfigured .env can be fixed in one pass.
func LoadEnv() (*Env, error) {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	e := &Env{
		Tokens:        splitTokens(getenv("TOKEN")),
		OCRAPIKey:     getenv("OCR_API_KEY"),
		Notifications: boolEnv("NOTIFICATIONS", true),
		Captcha: CaptchaEnv{
			APIKey:       getenv("CAPTCHA_API_KEY"),
			Service:      strings.ToLower(getenv("CAPTCHA_SERVICE")),
			SolveTimeout: 90 * time.Second,
		},
		Browser: BrowserEnv{
			Isolated:      boolEnv("BROWSER_ISOLATED", true),
			ProfilesDir:   getenv("BROWSER_PROFILES_DIR"),
			Executable:    getenv("BROWSER_EXECUTABLE"),
			ExtensionID:   getenv("BROWSER_EXTENSION_ID"),
			ExtensionPath: getenv("BROWSER_EXTENSION_PATH"),
		},
	}

	if len(e.Tokens) == 0 {
		add("TOKEN is empty — set a Discord user token in .env (comma-separated for multiple accounts)")
	}
	if e.OCRAPIKey == "" {
		e.OCRAPIKey = defaultOCRKey
	}
	if e.Captcha.Service == "" {
		e.Captcha.Service = defaultCaptchaSvc
	} else if !contains(captchaServices, e.Captcha.Service) {
		add("CAPTCHA_SERVICE %q is not one of %s", e.Captcha.Service, strings.Join(captchaServices, ", "))
	}
	if raw := getenv("CAPTCHA_SOLVE_TIMEOUT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			add("CAPTCHA_SOLVE_TIMEOUT must be a positive number of seconds, got %q", raw)
		} else {
			e.Captcha.SolveTimeout = time.Duration(n) * time.Second
		}
	}

	root := getenv("DATA_DIR")
	if root == "" {
		root = defaultRoot
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		add("DATA_DIR %q cannot be resolved: %v", root, err)
		abs = root
	}
	e.Dirs = Dirs{
		Root:            abs,
		Config:          filepath.Join(abs, "config"),
		Data:            filepath.Join(abs, "data"),
		BrowserProfiles: filepath.Join(abs, "browser-profiles"),
	}
	if e.Browser.ProfilesDir != "" {
		if p, err := filepath.Abs(e.Browser.ProfilesDir); err == nil {
			e.Dirs.BrowserProfiles = p
		}
	}
	e.Browser.ProfilesDir = e.Dirs.BrowserProfiles

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return e, nil
}

// EnsureDirs creates the writable directories up front, so a permission problem
// surfaces at startup instead of halfway through a captcha.
func (e *Env) EnsureDirs() error {
	for _, dir := range []string{e.Dirs.Config, e.Dirs.Data, e.Dirs.BrowserProfiles} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

func getenv(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func boolEnv(key string, def bool) bool {
	raw := strings.ToLower(getenv(key))
	switch raw {
	case "":
		return def
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func splitTokens(raw string) []string {
	var out []string
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
