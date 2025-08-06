{
  perSystem = {
    pkgs,
    inputs',
    ...
  }: {
    devShells.default = pkgs.mkShellNoCC {
      name = "project devshell";
      packages = builtins.attrValues {
        inherit
          (inputs'.wowsims.packages)
          wowsimcli
          ;
        inherit (pkgs) nodejs just;
      };
    };
  };
}
