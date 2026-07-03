package project

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/befrvnk/shellpin/internal/model"
)

func ExpandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func Canonicalize(path string) (string, error) {
	expanded, err := ExpandHome(path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), nil
	}
	return filepath.Clean(abs), nil
}

func IsWithin(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func Matches(entries []model.Entry, path string) []model.Entry {
	var matched []model.Entry
	for _, entry := range entries {
		if IsWithin(entry.Path, path) {
			matched = append(matched, entry)
		}
	}
	SortEntries(matched)
	return matched
}

func SortEntries(entries []model.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		leftLen := len(filepath.Clean(entries[i].Path))
		rightLen := len(filepath.Clean(entries[j].Path))
		if leftLen != rightLen {
			return leftLen > rightLen
		}
		return entries[i].ID < entries[j].ID
	})
}
