{
  buildGoModule,
  lib,
  gcc,
  installShellFiles,
  makeWrapper,
  wowsims-db,
  ...
}:
  buildGoModule {
    pname = "ookstats";
    version = "0.2.0";
    src = ./src;

    vendorHash = "sha256-QJxCINaMiJ6OjRJaOnHMW4gSfXLkXOYQU8xfbKdgQuQ=";

    nativeBuildInputs = [gcc installShellFiles makeWrapper];

    env.CGO_ENABLED = "1";

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
