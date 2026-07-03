{
  lib,
  buildGoModule,
  version ? "0.1.0",
  commit ? "unknown",
  date ? "unknown",
}:

buildGoModule {
  pname = "shellpin";
  inherit version;
  src = ./.;
  vendorHash = null;

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.commit=${commit}"
    "-X main.date=${date}"
  ];

  meta = {
    description = "User-local project development environment registry for Nix and devenv";
    homepage = "https://github.com/befrvnk/shellpin";
    license = lib.licenses.asl20;
    mainProgram = "shellpin";
  };
}
