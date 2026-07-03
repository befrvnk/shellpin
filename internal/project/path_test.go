package project

import (
	"path/filepath"
	"testing"

	"github.com/befrvnk/shellpin/internal/model"
)

func TestIsWithin(t *testing.T) {
	root := filepath.Clean("/tmp/project")
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "same", path: "/tmp/project", want: true},
		{name: "child", path: "/tmp/project/sub/dir", want: true},
		{name: "sibling prefix", path: "/tmp/project-other", want: false},
		{name: "parent", path: "/tmp", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsWithin(root, filepath.Clean(test.path)); got != test.want {
				t.Fatalf("IsWithin(%q, %q) = %v, want %v", root, test.path, got, test.want)
			}
		})
	}
}

func TestMatchesSortsMostSpecificFirst(t *testing.T) {
	entries := []model.Entry{
		{ID: "root", Path: "/tmp/project"},
		{ID: "nested", Path: "/tmp/project/sub"},
		{ID: "other", Path: "/tmp/other"},
	}
	got := Matches(entries, "/tmp/project/sub/dir")
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	if got[0].ID != "nested" || got[1].ID != "root" {
		t.Fatalf("got order %q, %q; want nested, root", got[0].ID, got[1].ID)
	}
}
