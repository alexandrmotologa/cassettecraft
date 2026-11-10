package cassette

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Storage manages cassette files on disk and in memory.
type Storage struct {
	baseDir    string
	mu         sync.RWMutex
	cache      map[string]*Cassette
	activeName string
}

// NewStorage initializes a cassette storage manager.
func NewStorage(baseDir string, defaultActive string) (*Storage, error) {
	if baseDir == "" {
		baseDir = "./cassettes"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cassette directory %q: %w", baseDir, err)
	}

	if defaultActive == "" {
		defaultActive = "default.yaml"
	}
	if !strings.HasSuffix(defaultActive, ".yaml") && !strings.HasSuffix(defaultActive, ".yml") && !strings.HasSuffix(defaultActive, ".json") {
		defaultActive += ".yaml"
	}

	s := &Storage{
		baseDir:    baseDir,
		cache:      make(map[string]*Cassette),
		activeName: defaultActive,
	}

	// Preload existing cassettes from disk
	_ = s.ReloadAll()
	return s, nil
}

// GetActiveName returns the currently active cassette filename.
func (s *Storage) GetActiveName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeName
}

// GetBaseDir returns the base storage directory path.
func (s *Storage) GetBaseDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseDir
}

// SetActiveName sets the currently active cassette filename.
func (s *Storage) SetActiveName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".json") {
		name += ".yaml"
	}
	s.activeName = name
}

// Load loads a cassette by name. If not already cached or on disk, creates an empty one.
func (s *Storage) Load(name string) (*Cassette, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normName := normalizeName(name)
	if cass, ok := s.cache[normName]; ok {
		return cass, nil
	}

	filePath := filepath.Join(s.baseDir, normName)
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read cassette %q: %w", filePath, err)
		}

		var cass *Cassette
		if strings.HasSuffix(normName, ".json") {
			cass, err = FromJSON(data)
		} else {
			cass, err = FromYAML(data)
		}
		if err != nil {
			return nil, err
		}

		s.cache[normName] = cass
		return cass, nil
	}

	// Create fresh cassette
	cass := NewCassette()
	s.cache[normName] = cass
	return cass, nil
}

// Save writes a cassette to disk deterministically.
func (s *Storage) Save(name string, cass *Cassette) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normName := normalizeName(name)
	filePath := filepath.Join(s.baseDir, normName)

	cass.RecordedAt = time.Now().UTC()
	var data []byte
	var err error
	if strings.HasSuffix(normName, ".json") {
		data, err = cass.ToJSON()
	} else {
		data, err = cass.ToYAML()
	}
	if err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cassette file %q: %w", filePath, err)
	}

	s.cache[normName] = cass
	return nil
}

// AppendInteraction adds an interaction to the named cassette and saves it to disk.
func (s *Storage) AppendInteraction(name string, interaction Interaction) error {
	cass, err := s.Load(name)
	if err != nil {
		return err
	}

	s.mu.Lock()
	// Check if ID already exists or generate if empty
	if interaction.ID == "" {
		interaction.ID = fmt.Sprintf("int_%d", time.Now().UnixNano())
	}
	if interaction.RecordedAt.IsZero() {
		interaction.RecordedAt = time.Now().UTC()
	}

	cass.Interactions = append(cass.Interactions, interaction)
	s.mu.Unlock()

	return s.Save(name, cass)
}

// ListCassettes returns metadata for all cassettes found in the base directory.
func (s *Storage) ListCassettes() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json") {
			names = append(names, name)
		}
	}
	return names, nil
}

// DeleteCassette removes a cassette file from disk and memory.
func (s *Storage) DeleteCassette(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normName := normalizeName(name)
	filePath := filepath.Join(s.baseDir, normName)

	delete(s.cache, normName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReloadAll reloads all cassettes from the base directory into memory.
func (s *Storage) ReloadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}

	s.cache = make(map[string]*Cassette)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json") {
			filePath := filepath.Join(s.baseDir, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			var cass *Cassette
			if strings.HasSuffix(name, ".json") {
				cass, _ = FromJSON(data)
			} else {
				cass, _ = FromYAML(data)
			}
			if cass != nil {
				s.cache[name] = cass
			}
		}
	}
	return nil
}

func normalizeName(name string) string {
	clean := filepath.Clean(name)
	base := filepath.Base(clean)
	if !strings.HasSuffix(base, ".yaml") && !strings.HasSuffix(base, ".yml") && !strings.HasSuffix(base, ".json") {
		base += ".yaml"
	}
	return base
}
