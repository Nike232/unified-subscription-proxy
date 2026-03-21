package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load() (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *FileStore) Save(data domain.PlatformData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(data)
}

func (s *FileStore) Mutate(fn func(*domain.PlatformData) error) (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadUnlocked()
	if err != nil {
		return domain.PlatformData{}, err
	}
	if err := fn(&data); err != nil {
		return domain.PlatformData{}, err
	}
	if err := s.saveUnlocked(data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *FileStore) loadUnlocked() (domain.PlatformData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data := BootstrapData()
			if err := s.saveUnlocked(data); err != nil {
				return domain.PlatformData{}, err
			}
			return data, nil
		}
		return domain.PlatformData{}, err
	}

	var data domain.PlatformData
	if len(bytes.TrimSpace(raw)) == 0 {
		data = BootstrapData()
		if err := s.saveUnlocked(data); err != nil {
			return domain.PlatformData{}, err
		}
		return data, nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *FileStore) saveUnlocked(data domain.PlatformData) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, "platform-*.json")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	if _, err := tmpFile.Write(raw); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}
