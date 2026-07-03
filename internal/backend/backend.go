package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/befrvnk/shellpin/internal/model"
	"github.com/befrvnk/shellpin/internal/store"
)

type ExecSpec struct {
	Name string
	Args []string
	Dir  string
}

type Runner interface {
	Run(spec ExecSpec) error
}

type OSRunner struct{}

func (OSRunner) Run(spec ExecSpec) error {
	cmd := exec.Command(spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Run(s *store.Store, entry model.Entry, commandArgs []string, runner Runner) error {
	spec, err := BuildRunSpec(s, entry, commandArgs)
	if err != nil {
		return err
	}
	return runner.Run(spec)
}

func Shell(s *store.Store, entry model.Entry, runner Runner) error {
	spec, err := BuildShellSpec(s, entry)
	if err != nil {
		return err
	}
	return runner.Run(spec)
}

func BuildRunSpec(s *store.Store, entry model.Entry, commandArgs []string) (ExecSpec, error) {
	if len(commandArgs) == 0 && strings.TrimSpace(entry.DefaultCommand) == "" {
		return ExecSpec{}, fmt.Errorf("entry %q has no default command; pass a command after --", entry.ID)
	}
	switch entry.Backend {
	case model.BackendNixShell:
		return buildNixShellRunSpec(entry, commandArgs)
	case model.BackendDevenv:
		return buildDevenvRunSpec(s, entry, commandArgs)
	default:
		return ExecSpec{}, fmt.Errorf("unsupported backend %q", entry.Backend)
	}
}

func BuildShellSpec(s *store.Store, entry model.Entry) (ExecSpec, error) {
	switch entry.Backend {
	case model.BackendNixShell:
		return buildNixShellShellSpec(entry)
	case model.BackendDevenv:
		return buildDevenvShellSpec(s, entry)
	default:
		return ExecSpec{}, fmt.Errorf("unsupported backend %q", entry.Backend)
	}
}

func buildNixShellRunSpec(entry model.Entry, commandArgs []string) (ExecSpec, error) {
	if entry.NixShell == nil {
		return ExecSpec{}, fmt.Errorf("entry %q is missing nixShell config", entry.ID)
	}

	args := append([]string{"shell"}, entry.NixShell.Packages...)
	args = append(args, "--command", "bash", "-lc")

	if len(commandArgs) == 0 {
		script := shellPrelude(entry) + "\n" + entry.DefaultCommand + "\n"
		args = append(args, script)
	} else {
		script := shellPrelude(entry) + "\nexec \"$@\"\n"
		args = append(args, script, "shellpin-run")
		args = append(args, commandArgs...)
	}

	return ExecSpec{Name: "nix", Args: args, Dir: entry.Path}, nil
}

func buildNixShellShellSpec(entry model.Entry) (ExecSpec, error) {
	if entry.NixShell == nil {
		return ExecSpec{}, fmt.Errorf("entry %q is missing nixShell config", entry.ID)
	}
	args := append([]string{"shell"}, entry.NixShell.Packages...)
	script := shellPrelude(entry) + "\nexec \"${SHELL:-bash}\" -i\n"
	args = append(args, "--command", "bash", "-lc", script)
	return ExecSpec{Name: "nix", Args: args, Dir: entry.Path}, nil
}

func shellPrelude(entry model.Entry) string {
	lines := []string{"set -euo pipefail"}
	if entry.NixShell != nil {
		lines = append(lines, entry.NixShell.Setup...)
	}
	return strings.Join(lines, "\n")
}

func buildDevenvRunSpec(s *store.Store, entry model.Entry, commandArgs []string) (ExecSpec, error) {
	if entry.Devenv == nil {
		return ExecSpec{}, fmt.Errorf("entry %q is missing devenv config", entry.ID)
	}
	args := []string{"shell", "--from", devenvSource(s, entry)}
	if len(commandArgs) == 0 {
		args = append(args, "bash", "-lc", entry.DefaultCommand)
	} else {
		args = append(args, commandArgs...)
	}
	return ExecSpec{Name: "devenv", Args: args, Dir: entry.Path}, nil
}

func buildDevenvShellSpec(s *store.Store, entry model.Entry) (ExecSpec, error) {
	if entry.Devenv == nil {
		return ExecSpec{}, fmt.Errorf("entry %q is missing devenv config", entry.ID)
	}
	return ExecSpec{Name: "devenv", Args: []string{"shell", "--from", devenvSource(s, entry)}, Dir: entry.Path}, nil
}

func devenvSource(s *store.Store, entry model.Entry) string {
	configDir := s.EntryDir(entry.ID)
	if entry.Devenv != nil && entry.Devenv.ConfigFile != "devenv.nix" {
		configDir = filepath.Dir(filepath.Join(s.EntryDir(entry.ID), entry.Devenv.ConfigFile))
	}
	return "path:" + configDir
}
