package backend

import (
	"strings"
	"testing"

	"github.com/befrvnk/shellpin/internal/model"
	"github.com/befrvnk/shellpin/internal/store"
)

func TestBuildNixShellRunSpecWithCommandArgs(t *testing.T) {
	entry := model.NewEntry("go-test", model.BackendNixShell, "/tmp/project", "Go tests")
	entry.NixShell = &model.NixShellConfig{
		Packages: []string{"nixpkgs#go", "nixpkgs#git"},
		Setup:    []string{"export FOO=bar"},
	}
	spec, err := BuildRunSpec(store.NewAt(t.TempDir()), entry, []string{"go", "test", "./..."})
	if err != nil {
		t.Fatalf("BuildRunSpec() error = %v", err)
	}
	if spec.Name != "nix" || spec.Dir != entry.Path {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	joined := strings.Join(spec.Args, "\x00")
	for _, want := range []string{"shell", "nixpkgs#go", "nixpkgs#git", "--command", "bash", "-lc", "export FOO=bar", "exec \"$@\"", "go", "test", "./..."} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args do not contain %q: %#v", want, spec.Args)
		}
	}
}

func TestBuildNixShellRunSpecWithDefaultCommand(t *testing.T) {
	entry := model.NewEntry("go-test", model.BackendNixShell, "/tmp/project", "Go tests")
	entry.DefaultCommand = "go test ./..."
	entry.NixShell = &model.NixShellConfig{Packages: []string{"nixpkgs#go"}}
	spec, err := BuildRunSpec(store.NewAt(t.TempDir()), entry, nil)
	if err != nil {
		t.Fatalf("BuildRunSpec() error = %v", err)
	}
	joined := strings.Join(spec.Args, "\x00")
	if !strings.Contains(joined, "go test ./...") {
		t.Fatalf("default command missing in args: %#v", spec.Args)
	}
}

func TestBuildRunSpecRequiresDefaultOrArgs(t *testing.T) {
	entry := model.NewEntry("go-test", model.BackendNixShell, "/tmp/project", "Go tests")
	entry.NixShell = &model.NixShellConfig{Packages: []string{"nixpkgs#go"}}
	_, err := BuildRunSpec(store.NewAt(t.TempDir()), entry, nil)
	if err == nil {
		t.Fatalf("BuildRunSpec() error = nil, want error")
	}
}

func TestBuildDevenvRunSpec(t *testing.T) {
	s := store.NewAt("/tmp/shellpin")
	entry := model.NewEntry("dev", model.BackendDevenv, "/tmp/project", "Devenv")
	entry.Devenv = &model.DevenvConfig{ConfigFile: "devenv.nix"}
	spec, err := BuildRunSpec(s, entry, []string{"go", "test"})
	if err != nil {
		t.Fatalf("BuildRunSpec() error = %v", err)
	}
	want := []string{"shell", "--from", "path:/tmp/shellpin/entries/dev", "go", "test"}
	if len(spec.Args) != len(want) {
		t.Fatalf("args = %#v, want %#v", spec.Args, want)
	}
	for i := range want {
		if spec.Args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q; all args %#v", i, spec.Args[i], want[i], spec.Args)
		}
	}
}
