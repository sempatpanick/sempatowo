package daily

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type accountStats struct {
	Daily float64 `json:"daily"`
}

// Store persists per-account daily timestamps (owo-dusk stats.json style).
type Store struct {
	mu   sync.Mutex
	path string
	data accountStats
}

func NewStore(dataDir, userID string) *Store {
	_ = os.MkdirAll(dataDir, 0o755)
	path := filepath.Join(dataDir, userID+"_stats.json")
	s := &Store{path: path}
	s.load()
	return s
}

func (s *Store) LastDaily() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Daily
}

func (s *Store) SetLastDaily(ts float64) {
	s.mu.Lock()
	s.data.Daily = ts
	s.mu.Unlock()
	_ = s.save()
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
