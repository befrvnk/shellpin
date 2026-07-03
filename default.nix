{
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
}
