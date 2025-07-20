{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  ...
}: {
  perSystem = {
    pkgs,
    inputs',
    ...
  }: let
    inherit (pkgs) callPackage;
    inherit (inputs'.wowsims.packages) wowsimcli;
  in {
    packages = {
      testRaid = callPackage ./testRaid.nix {
        inherit lib classes encounter buffs debuffs wowsimcli;
      };
    };
  };
}
