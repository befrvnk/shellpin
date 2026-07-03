package store

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/befrvnk/shellpin/internal/model"
	"github.com/befrvnk/shellpin/internal/project"
	"github.com/befrvnk/shellpin/internal/xdg"
)

type Store struct {
	Root string
}

func New() (*Store, error) {
	root, err := xdg.DataHome()
	if err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

func NewAt(root string) *Store {
	return &Store{Root: filepath.Clean(root)}
}

func (s *Store) EntriesDir() string {
	return filepath.Join(s.Root, "entries")
}

func (s *Store) EntryDir(id string) string {
	return filepath.Join(s.EntriesDir(), id)
}

func (s *Store) MetadataPath(id string) string {
	return filepath.Join(s.EntryDir(id), "metadata.json")
}

func (s *Store) DevenvConfigPath(id string) string {
	return filepath.Join(s.EntryDir(id), "devenv.nix")
}

func (s *Store) Ensure() error {
	return os.MkdirAll(s.EntriesDir(), 0o700)
}

func (s *Store) Exists(id string) bool {
	_, err := os.Stat(s.MetadataPath(id))
	return err == nil
}

func (s *Store) Save(entry model.Entry) error {
	if entry.SchemaVersion == 0 {
		entry.SchemaVersion = model.SchemaVersion
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	entry.UpdatedAt = time.Now().UTC()
	if err := entry.Validate(); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	entryDir := s.EntryDir(entry.ID)
	if err := os.MkdirAll(entryDir, 0o700); err != nil {
		return err
	}

	metadata, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	metadata = append(metadata, '\n')
	return atomicWriteFile(s.MetadataPath(entry.ID), metadata, 0o600)
}

func (s *Store) Load(id string) (model.Entry, error) {
	if err := model.ValidateID(id); err != nil {
		return model.Entry{}, err
	}
	content, err := os.ReadFile(s.MetadataPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return model.Entry{}, fmt.Errorf("entry %q not found", id)
		}
		return model.Entry{}, err
	}
	var entry model.Entry
	if err := json.Unmarshal(content, &entry); err != nil {
		return model.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return model.Entry{}, fmt.Errorf("invalid metadata for %q: %w", id, err)
	}
	return entry, nil
}

func (s *Store) List() ([]model.Entry, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	items, err := os.ReadDir(s.EntriesDir())
	if err != nil {
		return nil, err
	}

	entries := make([]model.Entry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		entry, err := s.Load(item.Name())
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	project.SortEntries(entries)
	return entries, nil
}

func (s *Store) Delete(id string) error {
	if err := model.ValidateID(id); err != nil {
		return err
	}
	if !s.Exists(id) {
		return fmt.Errorf("entry %q not found", id)
	}
	return os.RemoveAll(s.EntryDir(id))
}

func (s *Store) WriteDevenvConfig(id string, content []byte) error {
	if err := model.ValidateID(id); err != nil {
		return err
	}
	if err := os.MkdirAll(s.EntryDir(id), 0o700); err != nil {
		return err
	}
	return atomicWriteFile(s.DevenvConfigPath(id), content, 0o600)
}

func (s *Store) ReadDevenvConfig(id string) ([]byte, error) {
	return os.ReadFile(s.DevenvConfigPath(id))
}

func atomicWriteFile(path string, content []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
