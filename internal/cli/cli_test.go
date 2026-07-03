package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestVersionDoesNotRequireStore(t *testing.T) {
	t.Setenv("SHELLPIN_HOME", string(os.PathSeparator)+"definitely"+string(os.PathSeparator)+"missing"+string(os.PathSeparator)+"parent")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := MainWithBuildInfo([]string{"--version"}, &out, &errOut, BuildInfo{
		Version: "1.2.3",
		Commit:  "abc123",
		Date:    "2026-07-03",
	})
	if code != 0 {
		t.Fatalf("version exit code = %d, stderr = %s", code, errOut.String())
	}
	got := out.String()
	for _, want := range []string{"shellpin 1.2.3", "commit: abc123", "built: 2026-07-03"} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output missing %q: %s", want, got)
		}
	}
}

func TestAddNixShellListContextShow(t *testing.T) {
	t.Setenv("SHELLPIN_HOME", t.TempDir())
	projectDir := t.TempDir()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Main([]string{
		"add", "nix-shell", "go-test",
		"--path", projectDir,
		"--description", "Go test environment",
		"--package", "nixpkgs#go",
		"--setup", "export FOO=bar",
		"--default", "go test ./...",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("add exit code = %d, stderr = %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = Main([]string{"list", "--path", projectDir}, &out, &errOut)
	if code != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "go-test [nix-shell]") || !strings.Contains(got, "Go test environment") {
		t.Fatalf("list output missing entry: %s", got)
	}

	out.Reset()
	errOut.Reset()
	code = Main([]string{"context", "--path", projectDir}, &out, &errOut)
	if code != 0 {
		t.Fatalf("context exit code = %d, stderr = %s", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "shellpin run go-test -- <command>") || !strings.Contains(got, "shellpin run go-test") {
		t.Fatalf("context output missing usage: %s", got)
	}

	out.Reset()
	errOut.Reset()
	code = Main([]string{"show", "go-test"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("show exit code = %d, stderr = %s", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "nixpkgs#go") || !strings.Contains(got, "export FOO=bar") {
		t.Fatalf("show output missing details: %s", got)
	}
}

func TestAddDevenvCreatesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLPIN_HOME", home)
	projectDir := t.TempDir()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Main([]string{
		"add", "devenv", "dev",
		"--path", projectDir,
		"--description", "Devenv environment",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("add devenv exit code = %d, stderr = %s", code, errOut.String())
	}

	configPath := home + string(os.PathSeparator) + "entries" + string(os.PathSeparator) + "dev" + string(os.PathSeparator) + "devenv.nix"
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("devenv config was not created: %v", err)
	}
	if !strings.Contains(string(content), "packages = [") {
		t.Fatalf("unexpected devenv config: %s", content)
	}
}
