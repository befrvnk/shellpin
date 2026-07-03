package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const SchemaVersion = 1

type Backend string

const (
	BackendNixShell Backend = "nix-shell"
	BackendDevenv   Backend = "devenv"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Entry struct {
	SchemaVersion  int             `json:"schemaVersion"`
	ID             string          `json:"id"`
	Path           string          `json:"path"`
	Description    string          `json:"description"`
	Backend        Backend         `json:"backend"`
	DefaultCommand string          `json:"defaultCommand,omitempty"`
	NixShell       *NixShellConfig `json:"nixShell,omitempty"`
	Devenv         *DevenvConfig   `json:"devenv,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type NixShellConfig struct {
	Packages []string `json:"packages"`
	Setup    []string `json:"setup,omitempty"`
}

type DevenvConfig struct {
	ConfigFile string `json:"configFile"`
}

func NewEntry(id string, backend Backend, path string, description string) Entry {
	now := time.Now().UTC()
	return Entry{
		SchemaVersion: SchemaVersion,
		ID:            id,
		Path:          path,
		Description:   description,
		Backend:       backend,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func ValidateID(id string) error {
	if id == "" {
		return errors.New("id must not be empty")
	}
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q: use letters, numbers, '.', '_' or '-', starting with a letter or number", id)
	}
	return nil
}

func (e Entry) Validate() error {
	if err := ValidateID(e.ID); err != nil {
		return err
	}
	if e.SchemaVersion == 0 {
		return errors.New("schemaVersion must be set")
	}
	if e.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schemaVersion %d", e.SchemaVersion)
	}
	if strings.TrimSpace(e.Path) == "" {
		return errors.New("path must not be empty")
	}
	if strings.TrimSpace(e.Description) == "" {
		return errors.New("description must not be empty")
	}

	switch e.Backend {
	case BackendNixShell:
		if e.NixShell == nil {
			return errors.New("nix-shell backend requires nixShell config")
		}
		if len(e.NixShell.Packages) == 0 {
			return errors.New("nix-shell backend requires at least one package")
		}
		for _, pkg := range e.NixShell.Packages {
			if strings.TrimSpace(pkg) == "" {
				return errors.New("nix-shell package must not be empty")
			}
		}
		if e.Devenv != nil {
			return errors.New("nix-shell backend must not include devenv config")
		}
	case BackendDevenv:
		if e.Devenv == nil {
			return errors.New("devenv backend requires devenv config")
		}
		if strings.TrimSpace(e.Devenv.ConfigFile) == "" {
			return errors.New("devenv backend requires configFile")
		}
		if e.NixShell != nil {
			return errors.New("devenv backend must not include nixShell config")
		}
	default:
		return fmt.Errorf("unsupported backend %q", e.Backend)
	}
	return nil
}
