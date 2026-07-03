# shellpin

`shellpin` stores user-local development environment recipes per project directory.
It is meant for Nix-heavy workflows and coding-agent sessions where the right
`nix shell` or external `devenv` setup should be reused without modifying the
project repository.

## Features

- Store entries outside projects under `$XDG_DATA_HOME/shellpin` or
  `~/.local/share/shellpin`.
- Match entries by project directory, including child directories.
- Support `nix shell` recipes with packages, setup lines, and an optional
  default command.
- Support external `devenv` recipes stored as per-entry `devenv.nix` files.
- Print agent-friendly context with `shellpin context`.

## Status

Experimental. The command names and storage format may still change before a
stable `1.0` release.

## Installation

Run directly with Nix flakes:

```sh
nix run github:befrvnk/shellpin
```

Install the binary into a profile:

```sh
nix profile install github:befrvnk/shellpin
```

Or install with Go:

```sh
go install github.com/befrvnk/shellpin@latest
```

You can also build the local checkout:

```sh
nix build
./result/bin/shellpin --version
```

## Development

This project includes both a `devenv` environment and a Nix flake dev shell.

```sh
devenv shell
# or
nix develop
```

Useful commands:

```sh
go test ./...
go vet ./...
go build ./...
nix flake check --accept-flake-config
```

Useful devenv scripts:

```sh
devenv shell check
devenv shell build
devenv shell fmt
```

## Usage

Add a raw `nix shell` recipe:

```sh
shellpin add nix-shell khonshu-android \
  --path ~/projects/khonshu \
  --description "JDK 25 + Android SDK env for Khonshu Gradle tests" \
  --package nixpkgs#zulu25 \
  --package nixpkgs#wget \
  --package nixpkgs#unzip \
  --setup 'export JAVA_HOME="$(dirname "$(dirname "$(command -v java)")")"' \
  --setup 'export ANDROID_HOME="$PWD/.gradle/android-sdk"' \
  --setup 'export ANDROID_SDK_ROOT="$ANDROID_HOME"' \
  --default './gradlew :codegen-compiler-test:test --no-daemon --stacktrace'
```

Add an external `devenv` recipe:

```sh
shellpin add devenv khonshu-android \
  --path ~/projects/khonshu \
  --description "External devenv for Khonshu Android/Gradle work" \
  --edit
```

List entries for the current directory:

```sh
shellpin list
```

Print context intended for agents:

```sh
shellpin context
```

Run the default command:

```sh
shellpin run khonshu-android
```

Run a custom command inside the stored environment:

```sh
shellpin run khonshu-android -- ./gradlew test
```

Open an interactive shell:

```sh
shellpin shell khonshu-android
```

## Agent instruction

A useful global agent instruction is:

> Before inventing an ad-hoc `nix shell` or `devenv` environment, run
> `shellpin context` in the project directory. If a suitable stored environment
> exists, prefer `shellpin run <id> -- <command>` or `shellpin shell <id>`. Do
> not modify project files just to create a development environment unless asked.

## Security

`shellpin` entries can contain executable shell snippets and external `devenv`
configuration. Entries are never executed automatically, but running
`shellpin run <id>` or `shellpin shell <id>` executes code from that entry.
Only run entries you created or trust.

`SHELLPIN_HOME` and the default XDG storage location are user-local. If you use
shared machines, keep the storage directory private.

## devenv compatibility

The `devenv` backend stores an external `devenv.nix` and runs it with
`devenv shell --from path:<entry-dir>`. This requires a `devenv` version with
out-of-tree `--from` support. Raw `nix-shell` entries only require `nix`.

## Storage

By default entries are stored under:

```text
~/.local/share/shellpin/entries/<id>/metadata.json
~/.local/share/shellpin/entries/<id>/devenv.nix
```

For tests or custom setups, set `SHELLPIN_HOME`.

## License

MIT
