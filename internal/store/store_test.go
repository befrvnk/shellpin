package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/befrvnk/shellpin/internal/model"
)

func TestSaveLoadListDelete(t *testing.T) {
	s := NewAt(t.TempDir())
	projectDir := t.TempDir()
	entry := model.NewEntry("example", model.BackendNixShell, projectDir, "Example shell")
	entry.NixShell = &model.NixShellConfig{Packages: []string{"nixpkgs#go"}}
	entry.DefaultCommand = "go test ./..."

	if err := s.Save(entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := s.Load("example")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ID != entry.ID || loaded.Path != entry.Path || loaded.DefaultCommand != entry.DefaultCommand {
		t.Fatalf("loaded entry mismatch: %#v", loaded)
	}
	if loaded.CreatedAt.IsZero() || loaded.UpdatedAt.IsZero() {
		t.Fatalf("timestamps should be populated: %#v", loaded)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "example" {
		t.Fatalf("List() = %#v, want one example", entries)
	}

	if err := s.Delete("example"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Load("example"); err == nil {
		t.Fatalf("Load() after delete succeeded, want error")
	}
}

func TestWriteDevenvConfig(t *testing.T) {
	s := NewAt(t.TempDir())
	content := []byte("{ pkgs, ... }: { packages = [ pkgs.go ]; }\n")
	if err := s.WriteDevenvConfig("dev", content); err != nil {
		t.Fatalf("WriteDevenvConfig() error = %v", err)
	}
	path := filepath.Join(s.EntryDir("dev"), "devenv.nix")
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(read) != string(content) {
		t.Fatalf("config = %q, want %q", read, content)
	}
}
