{ pkgs, ... }:

{
  packages = [
    pkgs.go
    pkgs.gotools
  ];

  scripts.check.exec = "go test ./...";
  scripts.build.exec = "go build ./...";
  scripts.fmt.exec = "gofmt -w .";

  enterShell = ''
    echo "shellpin development environment"
    go version
  '';
}
