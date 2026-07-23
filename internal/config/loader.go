package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Logger is the subset of the project logger this package needs. Keeping it an
// interface means config stays a leaf package with no internal imports.
type Logger interface {
	Info(msg string)
	Danger(msg string)
}

// ChangeFunc is called after a successful hot-reload, with the settings that
// were replaced and the ones now in force. Passing both lets the caller restart
// only the subsystems whose settings actually moved.
type ChangeFunc func(old, new Settings)

// Loader loads and hot-reloads one account's JSON config file.
type Loader struct {
	mu       sync.RWMutex
	settings Settings
	path     string
	log      Logger
}

// LoadResult reports what happened during the initial load, so the caller can
// surface it without the loader deciding how to phrase things.
type LoadResult struct {
	// Created is true when the file did not exist and defaults were written.
	Created bool
	// Migrated is true when an older file was converted and rewritten.
	Migrated bool
	// FromVersion is the schema version the file was migrated from. 0 is the
	// pre-1.0 shape, which has no schemaVersion key.
	FromVersion int
	// BackupPath is where the pre-migration file was preserved.
	BackupPath string
	// Notes are migration remarks and unknown-key warnings.
	Notes []string

	// legacyRaw is the pre-migration file, kept so the loader can back it up.
	// Inspect deliberately does not write it: checking a config must not
	// change anything on disk.
	legacyRaw []byte
}

// NewLoader opens (or creates) the config file for one account and starts
// watching it. Files are keyed by Discord user ID rather than username: usernames
// change, IDs do not, and the runtime data files in data/ are already keyed that
// way. The human-readable name is stored inside the file as "label".
func NewLoader(configDir, userID, label string, log Logger, onChange ChangeFunc) (*Loader, LoadResult, error) {
	var res LoadResult

	if userID == "" {
		return nil, res, fmt.Errorf("config: empty user ID")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, res, err
	}

	path := filepath.Join(configDir, userID+".json")
	l := &Loader{path: path, log: log}

	if legacy := legacyPathFor(configDir, label); legacy != "" && !fileExists(path) {
		if err := os.Rename(legacy, path); err != nil {
			return nil, res, fmt.Errorf("config: moving %s to %s: %w", legacy, path, err)
		}
		res.Notes = append(res.Notes, "renamed "+filepath.Base(legacy)+" to "+filepath.Base(path)+" (config files are keyed by user ID now)")
	}

	if !fileExists(path) {
		l.settings = Defaults()
		l.settings.Label = label
		if err := l.save(); err != nil {
			return nil, res, err
		}
		res.Created = true
	} else {
		s, r, err := loadFromFile(path, label)
		if err != nil {
			return nil, res, err
		}
		l.settings = s
		res.Migrated = r.Migrated
		res.FromVersion = r.FromVersion
		res.Notes = append(res.Notes, r.Notes...)

		if r.Migrated {
			res.BackupPath = fmt.Sprintf("%s.v%d.bak", path, r.FromVersion)
			if err := os.WriteFile(res.BackupPath, r.legacyRaw, 0o644); err != nil {
				return nil, res, fmt.Errorf("backing up config: %w", err)
			}
			if err := l.save(); err != nil {
				return nil, res, err
			}
			res.Notes = append(res.Notes,
				fmt.Sprintf("migrated from schemaVersion %d to %d; original kept at %s",
					r.FromVersion, SchemaVersion, filepath.Base(res.BackupPath)))
		}
	}

	if err := l.settings.Validate(); err != nil {
		return nil, res, fmt.Errorf("invalid config %s:\n%w", path, err)
	}

	go l.watch(onChange)
	return l, res, nil
}

// Get returns the settings currently in force.
func (l *Loader) Get() Settings {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.settings
}

// Path is the file this loader watches.
func (l *Loader) Path() string { return l.path }

func (l *Loader) set(s Settings) Settings {
	l.mu.Lock()
	defer l.mu.Unlock()
	old := l.settings
	l.settings = s
	return old
}

func (l *Loader) save() error {
	l.mu.RLock()
	data, err := marshalReadable(l.settings)
	l.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, data, 0o644)
}

func (l *Loader) info(msg string) {
	if l.log != nil {
		l.log.Info(msg)
	}
}

func (l *Loader) danger(msg string) {
	if l.log != nil {
		l.log.Danger(msg)
	}
}

func (l *Loader) watch(onChange ChangeFunc) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		l.danger(fmt.Sprintf("config watch error: %v (hot-reload disabled)", err))
		return
	}
	defer watcher.Close()

	// Without this the loop below runs forever receiving nothing, and edits to
	// the config file are silently ignored with no hint as to why.
	if err := watcher.Add(l.path); err != nil {
		l.danger(fmt.Sprintf("config watch error: cannot watch %s: %v (hot-reload disabled)", l.path, err))
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			l.reload(onChange)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			l.danger(fmt.Sprintf("config watcher: %v", err))
		}
	}
}

// reload re-reads the file. A file that fails to parse or validate is rejected
// and the previous settings stay in force — applying a half-broken config to a
// running bot is worse than ignoring the edit, as long as we say so loudly.
func (l *Loader) reload(onChange ChangeFunc) {
	label := l.Get().Label

	s, res, err := loadFromFile(l.path, label)
	if err != nil {
		l.danger("config reload rejected, keeping previous settings: " + err.Error())
		return
	}
	if err := s.Validate(); err != nil {
		l.danger("config reload rejected, keeping previous settings:\n" + err.Error())
		return
	}
	for _, note := range res.Notes {
		l.info("config: " + note)
	}
	for _, w := range s.Warnings() {
		l.info("config warning: " + w)
	}

	old := l.set(s)
	if onChange != nil {
		onChange(old, s)
	}
}

// Inspect reads and validates a config file without taking ownership of it: no
// watcher is started and, unlike the loader, a legacy file is not rewritten.
// It is what `sempatowo -check-config` runs.
func Inspect(path string) (Settings, LoadResult, error) {
	s, res, err := loadFromFile(path, "")
	if err != nil {
		return Settings{}, res, err
	}
	return s, res, s.Validate()
}

// loadFromFile reads path, migrating it forward from an older schema if needed.
func loadFromFile(path, label string) (Settings, LoadResult, error) {
	var res LoadResult

	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, res, err
	}

	// A migrating load returns here rather than falling through: the old
	// spellings are not current keys, and unknownKeyNotes below would report
	// every one of them as a typo on the single load that fixes them.
	if v := fileVersion(data); v < SchemaVersion {
		var (
			s     Settings
			notes []string
			err   error
		)
		switch v {
		case 0:
			s, notes, err = migrateLegacy(data)
		case 1:
			s, notes, err = migrateV1(data)
		default:
			err = fmt.Errorf("no migration from schemaVersion %d", v)
		}
		if err != nil {
			return Settings{}, res, err
		}
		if s.Label == "" {
			s.Label = label
		}
		res.Migrated = true
		res.FromVersion = v
		res.legacyRaw = data
		res.Notes = append(res.Notes, notes...)
		return s, res, nil
	}

	// Unmarshalling into an already-populated struct leaves fields the file
	// omits untouched, at every depth. That is the whole defaults merge — the
	// map-walking version this replaced was doing the same job twice.
	s := Defaults()
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, res, fmt.Errorf("parsing %s: %w", filepath.Base(path), err)
	}
	if s.Label == "" {
		s.Label = label
	}
	res.Notes = append(res.Notes, unknownKeyNotes(data)...)

	return s, res, nil
}

// unknownKeyNotes reports keys the schema does not know about. A typo like
// "enbaled" otherwise reads as "left at its default" and costs an evening.
func unknownKeyNotes(data []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var probe Settings
	if err := dec.Decode(&probe); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "unknown field") {
			return []string{msg + " — check for a typo; this setting is being ignored"}
		}
	}
	return nil
}

// legacyPathFor finds a pre-existing username-keyed config file to rename.
func legacyPathFor(configDir, label string) string {
	if label == "" {
		return ""
	}
	p := filepath.Join(configDir, label+".json")
	if fileExists(p) {
		return p
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
