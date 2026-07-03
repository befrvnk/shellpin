{
  description = "User-local project development environment registry for Nix and devenv";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      version = "0.1.0";
      commit = self.shortRev or (self.dirtyShortRev or "unknown");
      date = self.lastModifiedDate or "unknown";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.callPackage ./default.nix {
            inherit version commit date;
          };
          shellpin = self.packages.${system}.default;
        }
      );

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/shellpin";
          meta.description = "User-local project development environment registry";
        };
        shellpin = self.apps.${system}.default;
      });

      checks = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = self.packages.${system}.default;
          go-test = pkgs.runCommand "shellpin-go-test" { nativeBuildInputs = [ pkgs.go ]; } ''
            export HOME=$TMPDIR
            export GOCACHE=$TMPDIR/go-cache
            cp -R ${self} source
            chmod -R u+w source
            cd source
            go test ./...
            touch $out
          '';
          go-vet = pkgs.runCommand "shellpin-go-vet" { nativeBuildInputs = [ pkgs.go ]; } ''
            export HOME=$TMPDIR
            export GOCACHE=$TMPDIR/go-cache
            cp -R ${self} source
            chmod -R u+w source
            cd source
            go vet ./...
            touch $out
          '';
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gotools
              pkgs.nixfmt
            ];
          };
        }
      );
    };
}
