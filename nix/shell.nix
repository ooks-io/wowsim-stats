{
  perSystem = {
    pkgs,
    inputs',
    config,
    ...
  }: {
    devShells.default = pkgs.mkShell {
      name = "project devshell";
      packages = builtins.attrValues {
        inherit
          (inputs'.wowsims.packages)
          wowsimcli
          ;
        inherit (pkgs) nodejs just sqlite awscli2 go gcc pkg-config sqld yarn;
        inherit (pkgs.python3Packages) python requests;
        inherit (config.packages) ookstats;
        inherit (pkgs.nodePackages) vercel;
      };
    };
  };
}
