{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  api,
  ...
}: {
  perSystem = {pkgs, ...}: let
    inherit (pkgs) writers python3Packages;
    simulation = import ./simulation {inherit lib classes encounter buffs debuffs inputs trinket pkgs;};
    getDB = pkgs.callPackage ./getDB.nix {inherit inputs;};
    getCMLeaders = import ./challenge-mode-leaderboard.nix {inherit api writers python3Packages;};
    parseCMs = import ./challenge-mode-parser.nix {inherit api writers python3Packages;};
  in {
    apps =
      simulation
      // {
        getDB = {
          type = "app";
          program = "${getDB}/bin/getDB";
        };
        getCM = {
          type = "app";
          program = "${getCMLeaders}/bin/cm-leaderboard-fetcher";
        };
        parseCM = {
          type = "app";
          program = "${parseCMs}/bin/cm-leaderboard-parser";
        };
      };
  };
}
