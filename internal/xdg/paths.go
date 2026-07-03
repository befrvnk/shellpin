package xdg

import (
	"errors"
	"os"
	"path/filepath"
)

const appName = "shellpin"

func DataHome() (string, error) {
	if override := os.Getenv("SHELLPIN_HOME"); override != "" {
		return filepath.Clean(override), nil
	}

	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, appName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("could not determine home directory; set SHELLPIN_HOME")
	}
	return filepath.Join(home, ".local", "share", appName), nil
}
