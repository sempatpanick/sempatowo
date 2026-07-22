package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Loader loads and hot-reloads per-user JSON config files.
type Loader struct {
	mu       sync.RWMutex
	settings Settings
	path     string
}

// NewLoader creates config/{username}.json, merges missing keys from defaults, and watches for changes.
func NewLoader(configDir, username string, onChange func(Settings)) (*Loader, error) {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(configDir, username+".json")
	l := &Loader{path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		l.settings = Defaults()
		if err := l.save(); err != nil {
			return nil, err
		}
	} else {
		s, err := loadFromFile(path)
		if err != nil {
			return nil, err
		}
		l.settings = s
	}

	if onChange != nil {
		onChange(l.settings)
	}

	go l.watch(onChange)
	return l, nil
}

func (l *Loader) Get() Settings {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.settings
}

func (l *Loader) Set(s Settings) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.settings = s
}

func (l *Loader) save() error {
	data, err := json.MarshalIndent(l.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, data, 0o644)
}

func (l *Loader) watch(onChange func(Settings)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("config watch error: %v\n", err)
		return
	}
	defer watcher.Close()

	// Without this the loop below runs forever receiving nothing, and edits to
	// the config file are silently ignored with no hint as to why.
	if err := watcher.Add(l.path); err != nil {
		fmt.Printf("config watch error: cannot watch %s: %v (hot-reload disabled)\n", l.path, err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				s, err := loadFromFile(l.path)
				if err != nil {
					fmt.Printf("config reload error: %v\n", err)
					continue
				}
				l.Set(s)
				if onChange != nil {
					onChange(s)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("config watcher: %v\n", err)
		}
	}
}

func loadFromFile(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Settings{}, err
	}

	merged := Defaults()
	if err := mergeJSON(&merged, raw); err != nil {
		return Settings{}, err
	}

	normalizeDelays(&merged)
	return merged, nil
}

// mergeJSON overlays the user's raw JSON onto dst, filling any key the user
// omitted from Defaults. It reports an error rather than leaving dst partially
// populated, which would silently hand back a half-valid config.
func mergeJSON(dst *Settings, raw map[string]json.RawMessage) error {
	defaults := Defaults()
	defBytes, _ := json.Marshal(defaults)
	var defMap map[string]json.RawMessage
	_ = json.Unmarshal(defBytes, &defMap)

	for key, defVal := range defMap {
		userVal, ok := raw[key]
		if !ok {
			raw[key] = defVal
			continue
		}
		// Shallow merge for nested objects (status, interval, channels, etc.)
		if isObject(defVal) && isObject(userVal) {
			var defObj, userObj map[string]json.RawMessage
			_ = json.Unmarshal(defVal, &defObj)
			_ = json.Unmarshal(userVal, &userObj)
			for subKey, subDef := range defObj {
				if _, exists := userObj[subKey]; !exists {
					userObj[subKey] = subDef
				}
			}
			merged, _ := json.Marshal(userObj)
			raw[key] = merged
		}
	}

	final, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(final, dst)
}

func isObject(raw json.RawMessage) bool {
	return len(raw) > 0 && raw[0] == '{'
}

func normalizeDelays(s *Settings) {
	for _, pair := range []struct {
		delay *ActionDelay
		def   ActionDelay
	}{
		{&s.Interval.Hunt, Defaults().Interval.Hunt},
		{&s.Interval.Battle, Defaults().Interval.Battle},
	} {
		d := pair.delay
		if d.MinDelay == 0 && d.SlowestTime > 0 {
			d.MinDelay = d.SlowestTime
		}
		if d.MaxDelay == 0 && d.FastestTime > 0 {
			d.MaxDelay = d.FastestTime
		}
		if d.MinDelay == 0 {
			d.MinDelay = pair.def.MinDelay
		}
		if d.MaxDelay == 0 {
			d.MaxDelay = pair.def.MaxDelay
		}
	}
}

// UnmarshalJSON for jsonSleeptime: number or [min, max].
func (j *jsonSleeptime) UnmarshalJSON(data []byte) error {
	var single float64
	if err := json.Unmarshal(data, &single); err == nil {
		j.Single = &single
		return nil
	}
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err == nil {
		j.Range = &arr
		return nil
	}
	return fmt.Errorf("invalid sleeptime: %s", string(data))
}

func (j jsonSleeptime) MarshalJSON() ([]byte, error) {
	if j.Range != nil {
		return json.Marshal(j.Range)
	}
	if j.Single != nil {
		return json.Marshal(*j.Single)
	}
	return json.Marshal(nil)
}

func (j *jsonSecRange) UnmarshalJSON(data []byte) error {
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("invalid cooldown: %s", string(data))
	}
	j.Range = &arr
	return nil
}

func (j jsonSecRange) MarshalJSON() ([]byte, error) {
	if j.Range != nil {
		return json.Marshal(j.Range)
	}
	return json.Marshal(nil)
}
