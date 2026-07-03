package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/befrvnk/shellpin/internal/backend"
	"github.com/befrvnk/shellpin/internal/model"
	"github.com/befrvnk/shellpin/internal/project"
	"github.com/befrvnk/shellpin/internal/store"
)

const defaultDevenvConfig = `{ pkgs, ... }:

{
  packages = [
    # pkgs.git
  ];

  # enterShell = ''
  #   echo "shellpin devenv is active"
  # '';
}
`

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func Main(args []string, out io.Writer, errOut io.Writer) int {
	return MainWithBuildInfo(args, out, errOut, BuildInfo{
		Version: "dev",
		Commit:  "unknown",
		Date:    "unknown",
	})
}

func MainWithBuildInfo(args []string, out io.Writer, errOut io.Writer, buildInfo BuildInfo) int {
	if len(args) == 0 {
		printHelp(out)
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp(out)
		return 0
	case "version", "--version", "-v":
		printVersion(out, buildInfo)
		return 0
	}

	s, err := store.New()
	if err != nil {
		fmt.Fprintf(errOut, "shellpin: %v\n", err)
		return 1
	}

	switch args[0] {
	case "add":
		return runAdd(s, args[1:], out, errOut)
	case "list", "ls":
		return runList(s, args[1:], out, errOut)
	case "context", "agent-context":
		return runContext(s, args[1:], out, errOut)
	case "show":
		return runShow(s, args[1:], out, errOut)
	case "rm", "remove", "delete":
		return runRemove(s, args[1:], out, errOut)
	case "run":
		return runRun(s, args[1:], errOut)
	case "shell":
		return runShell(s, args[1:], errOut)
	case "edit":
		return runEdit(s, args[1:], errOut)
	default:
		fmt.Fprintf(errOut, "shellpin: unknown command %q\n\n", args[0])
		printHelp(errOut)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `shellpin stores user-local project development environments.

Usage:
  shellpin add nix-shell <id> [flags]
  shellpin add devenv <id> [flags]
  shellpin list [--path <path>] [--all] [--json]
  shellpin context [--path <path>] [--json]
  shellpin show <id> [--json]
  shellpin run <id> [-- <command> [args...]]
  shellpin shell <id>
  shellpin edit <id>
  shellpin rm <id>
  shellpin version

Examples:
  shellpin add nix-shell khonshu-android \
    --description "JDK 25 + Android SDK env" \
    --package nixpkgs#zulu25 --package nixpkgs#wget --package nixpkgs#unzip \
    --setup 'export JAVA_HOME="$(dirname "$(dirname "$(command -v java)")")"' \
    --setup 'export ANDROID_HOME="$PWD/.gradle/android-sdk"' \
    --setup 'export ANDROID_SDK_ROOT="$ANDROID_HOME"' \
    --default './gradlew :codegen-compiler-test:test --no-daemon --stacktrace'

  shellpin add devenv khonshu-android --description "External devenv for Khonshu" --edit
  shellpin run khonshu-android -- ./gradlew test
  shellpin context
`)
}

func printVersion(w io.Writer, buildInfo BuildInfo) {
	fmt.Fprintf(w, "shellpin %s\n", buildInfo.Version)
	fmt.Fprintf(w, "commit: %s\n", buildInfo.Commit)
	fmt.Fprintf(w, "built: %s\n", buildInfo.Date)
}

func runAdd(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "shellpin add: expected backend: nix-shell or devenv")
		return 2
	}
	switch args[0] {
	case string(model.BackendNixShell):
		return runAddNixShell(s, args[1:], out, errOut)
	case string(model.BackendDevenv):
		return runAddDevenv(s, args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "shellpin add: unsupported backend %q\n", args[0])
		return 2
	}
}

func runAddNixShell(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "shellpin add nix-shell: expected id")
		return 2
	}
	id := args[0]
	fs := flag.NewFlagSet("add nix-shell", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", ".", "project path")
	description := fs.String("description", "", "short description")
	defaultCommand := fs.String("default", "", "default command")
	var packages multiFlag
	var setup multiFlag
	fs.Var(&packages, "package", "nix package flake ref, repeatable")
	fs.Var(&setup, "setup", "shell setup line, repeatable")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(errOut, "shellpin add nix-shell: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	projectPath, err := project.Canonicalize(*pathFlag)
	if err != nil {
		fmt.Fprintf(errOut, "shellpin add nix-shell: %v\n", err)
		return 1
	}
	entry := model.NewEntry(id, model.BackendNixShell, projectPath, *description)
	entry.DefaultCommand = *defaultCommand
	entry.NixShell = &model.NixShellConfig{Packages: packages, Setup: setup}

	if s.Exists(id) {
		fmt.Fprintf(errOut, "shellpin add nix-shell: entry %q already exists\n", id)
		return 1
	}
	if err := s.Save(entry); err != nil {
		fmt.Fprintf(errOut, "shellpin add nix-shell: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "Added nix-shell environment %q for %s\n", id, projectPath)
	return 0
}

func runAddDevenv(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "shellpin add devenv: expected id")
		return 2
	}
	id := args[0]
	fs := flag.NewFlagSet("add devenv", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", ".", "project path")
	description := fs.String("description", "", "short description")
	defaultCommand := fs.String("default", "", "default command")
	configPath := fs.String("config", "", "copy an existing devenv.nix into the entry")
	editAfter := fs.Bool("edit", false, "open devenv.nix in $EDITOR after creating it")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(errOut, "shellpin add devenv: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	projectPath, err := project.Canonicalize(*pathFlag)
	if err != nil {
		fmt.Fprintf(errOut, "shellpin add devenv: %v\n", err)
		return 1
	}
	if s.Exists(id) {
		fmt.Fprintf(errOut, "shellpin add devenv: entry %q already exists\n", id)
		return 1
	}

	config := []byte(defaultDevenvConfig)
	if *configPath != "" {
		content, err := os.ReadFile(*configPath)
		if err != nil {
			fmt.Fprintf(errOut, "shellpin add devenv: reading config: %v\n", err)
			return 1
		}
		config = content
	}
	if err := s.WriteDevenvConfig(id, config); err != nil {
		fmt.Fprintf(errOut, "shellpin add devenv: writing config: %v\n", err)
		return 1
	}

	entry := model.NewEntry(id, model.BackendDevenv, projectPath, *description)
	entry.DefaultCommand = *defaultCommand
	entry.Devenv = &model.DevenvConfig{ConfigFile: "devenv.nix"}
	if err := s.Save(entry); err != nil {
		fmt.Fprintf(errOut, "shellpin add devenv: %v\n", err)
		_ = os.RemoveAll(s.EntryDir(id))
		return 1
	}
	if *editAfter {
		if err := openEditor(s.DevenvConfigPath(id)); err != nil {
			fmt.Fprintf(errOut, "shellpin add devenv: opening editor: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(out, "Added devenv environment %q for %s\n", id, projectPath)
	fmt.Fprintf(out, "Config: %s\n", s.DevenvConfigPath(id))
	return 0
}

func runList(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", ".", "path to match")
	all := fs.Bool("all", false, "list all entries")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(errOut, "shellpin list: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	entries, err := s.List()
	if err != nil {
		fmt.Fprintf(errOut, "shellpin list: %v\n", err)
		return 1
	}
	matchedPath := ""
	if !*all {
		matchedPath, err = project.Canonicalize(*pathFlag)
		if err != nil {
			fmt.Fprintf(errOut, "shellpin list: %v\n", err)
			return 1
		}
		entries = project.Matches(entries, matchedPath)
	}
	if *jsonOut {
		return writeJSON(out, errOut, entries)
	}
	if len(entries) == 0 {
		if *all {
			fmt.Fprintln(out, "No shellpin environments registered.")
		} else {
			fmt.Fprintf(out, "No shellpin environments registered for %s.\n", matchedPath)
		}
		return 0
	}
	for _, entry := range entries {
		fmt.Fprintf(out, "%s [%s]\n", entry.ID, entry.Backend)
		fmt.Fprintf(out, "  %s\n", entry.Description)
		fmt.Fprintf(out, "  Project: %s\n", entry.Path)
		if entry.DefaultCommand != "" {
			fmt.Fprintf(out, "  Default: shellpin run %s\n", entry.ID)
		}
		fmt.Fprintf(out, "  Run: shellpin run %s -- <command>\n", entry.ID)
	}
	return 0
}

func runContext(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", ".", "path to match")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(errOut, "shellpin context: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	path, err := project.Canonicalize(*pathFlag)
	if err != nil {
		fmt.Fprintf(errOut, "shellpin context: %v\n", err)
		return 1
	}
	entries, err := s.List()
	if err != nil {
		fmt.Fprintf(errOut, "shellpin context: %v\n", err)
		return 1
	}
	entries = project.Matches(entries, path)
	if *jsonOut {
		payload := struct {
			Path    string        `json:"path"`
			Entries []model.Entry `json:"entries"`
		}{Path: path, Entries: entries}
		return writeJSON(out, errOut, payload)
	}
	if len(entries) == 0 {
		fmt.Fprintf(out, "No shellpin environments registered for %s.\n", path)
		return 0
	}
	fmt.Fprintf(out, "Known shellpin environments for %s:\n\n", path)
	for _, entry := range entries {
		fmt.Fprintf(out, "- %s\n", entry.ID)
		fmt.Fprintf(out, "  Backend: %s\n", entry.Backend)
		fmt.Fprintf(out, "  Project: %s\n", entry.Path)
		fmt.Fprintf(out, "  Description: %s\n", entry.Description)
		fmt.Fprintf(out, "  Open shell: shellpin shell %s\n", entry.ID)
		fmt.Fprintf(out, "  Run command: shellpin run %s -- <command>\n", entry.ID)
		if entry.DefaultCommand != "" {
			fmt.Fprintf(out, "  Run default: shellpin run %s\n", entry.ID)
		}
		if entry.Backend == model.BackendDevenv {
			fmt.Fprintf(out, "  Config: %s\n", s.DevenvConfigPath(entry.ID))
		}
	}
	return 0
}

func runShow(s *store.Store, args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) != 1 {
		fmt.Fprintln(errOut, "shellpin show: expected exactly one id")
		return 2
	}
	entry, err := s.Load(fs.Args()[0])
	if err != nil {
		fmt.Fprintf(errOut, "shellpin show: %v\n", err)
		return 1
	}
	if *jsonOut {
		return writeJSON(out, errOut, entry)
	}
	fmt.Fprintf(out, "ID: %s\n", entry.ID)
	fmt.Fprintf(out, "Backend: %s\n", entry.Backend)
	fmt.Fprintf(out, "Project: %s\n", entry.Path)
	fmt.Fprintf(out, "Description: %s\n", entry.Description)
	if entry.DefaultCommand != "" {
		fmt.Fprintf(out, "Default command: %s\n", entry.DefaultCommand)
	}
	switch entry.Backend {
	case model.BackendNixShell:
		fmt.Fprintln(out, "Packages:")
		for _, pkg := range entry.NixShell.Packages {
			fmt.Fprintf(out, "  - %s\n", pkg)
		}
		if len(entry.NixShell.Setup) > 0 {
			fmt.Fprintln(out, "Setup:")
			for _, setup := range entry.NixShell.Setup {
				fmt.Fprintf(out, "  %s\n", setup)
			}
		}
	case model.BackendDevenv:
		fmt.Fprintf(out, "Config: %s\n", s.DevenvConfigPath(entry.ID))
	}
	return 0
}

func runRemove(s *store.Store, args []string, _ io.Writer, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "shellpin rm: expected exactly one id")
		return 2
	}
	if err := s.Delete(args[0]); err != nil {
		fmt.Fprintf(errOut, "shellpin rm: %v\n", err)
		return 1
	}
	return 0
}

func runRun(s *store.Store, args []string, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "shellpin run: expected id")
		return 2
	}
	id := args[0]
	commandArgs := args[1:]
	if len(commandArgs) > 0 && commandArgs[0] == "--" {
		commandArgs = commandArgs[1:]
	}
	entry, err := s.Load(id)
	if err != nil {
		fmt.Fprintf(errOut, "shellpin run: %v\n", err)
		return 1
	}
	if err := backend.Run(s, entry, commandArgs, backend.OSRunner{}); err != nil {
		fmt.Fprintf(errOut, "shellpin run: %v\n", err)
		return exitCode(err)
	}
	return 0
}

func runShell(s *store.Store, args []string, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "shellpin shell: expected exactly one id")
		return 2
	}
	entry, err := s.Load(args[0])
	if err != nil {
		fmt.Fprintf(errOut, "shellpin shell: %v\n", err)
		return 1
	}
	if err := backend.Shell(s, entry, backend.OSRunner{}); err != nil {
		fmt.Fprintf(errOut, "shellpin shell: %v\n", err)
		return exitCode(err)
	}
	return 0
}

func runEdit(s *store.Store, args []string, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "shellpin edit: expected exactly one id")
		return 2
	}
	entry, err := s.Load(args[0])
	if err != nil {
		fmt.Fprintf(errOut, "shellpin edit: %v\n", err)
		return 1
	}
	target := s.MetadataPath(entry.ID)
	if entry.Backend == model.BackendDevenv {
		target = s.DevenvConfigPath(entry.ID)
	}
	if err := openEditor(target); err != nil {
		fmt.Fprintf(errOut, "shellpin edit: %v\n", err)
		return 1
	}
	return 0
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return errors.New("EDITOR is empty")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeJSON(out io.Writer, errOut io.Writer, value any) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(errOut, "shellpin: writing JSON: %v\n", err)
		return 1
	}
	return 0
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func AbsForDisplay(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
