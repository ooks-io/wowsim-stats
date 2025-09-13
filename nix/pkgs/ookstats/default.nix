{
  buildGoModule,
  lib,
  gcc,
  pkgs,
  go-libsql-src,
  installShellFiles,
  makeWrapper,
  wowsims-db,
  ...
}: let
  # map nix platform to go-libsql's platform string
  libsqlArch =
    if pkgs.stdenv.hostPlatform.system == "x86_64-linux"
    then "linux_amd64"
    else if pkgs.stdenv.hostPlatform.system == "aarch64-linux"
    then "linux_arm64"
    else if pkgs.stdenv.hostPlatform.system == "aarch64-darwin"
    then "darwin_arm64"
    else throw "Unsupported platform for go-libsql pre-compiled library";
in
  buildGoModule {
    pname = "ookstats";
    version = "0.1.0";
    src = ./src;

    vendorHash = "sha256-o4h4etGvTukPC/EixnCP+Ik++odm4ckozexkUB6Sv5g=";

    nativeBuildInputs = [gcc installShellFiles makeWrapper];

    env = {
      CGO_ENABLED = "1";
      # see:
      # https://github.com/tursodatabase/go-libsql/issues/21
      # https://github.com/tursodatabase/go-libsql/issues/57
      CGO_CFLAGS = "-I${go-libsql-src}/lib/include";
      CGO_LDFLAGS = "-L${go-libsql-src}/lib/${libsqlArch} -lsql_experimental";
    };

    # Generate shell completions
    postInstall = ''
      installShellCompletion --cmd ookstats \
        --bash <($out/bin/ookstats completion bash) \
        --fish <($out/bin/ookstats completion fish) \
        --zsh <($out/bin/ookstats completion zsh)

      # If a wowsims input is provided, wrap the binary to point to the items DB in the store
      ${lib.optionalString (wowsims-db != null) ''
        wrapProgram $out/bin/ookstats \
          --set OOKSTATS_WOWSIMS_DB ${wowsims-db}
      ''}
    '';

    meta = {
      description = "WoW Challenge Mode statistics database CLI tool";
      license = lib.licenses.mit;
    };
  }
