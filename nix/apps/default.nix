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
    inherit (pkgs) callPackage writeShellApplication writers python3Packages;
    simulation = import ./simulation {inherit lib classes encounter buffs debuffs inputs trinket pkgs;};
    getDB = callPackage ./getDB.nix {inherit inputs;};
    testItemLookup = callPackage ./testItemLookup.nix {inherit inputs lib;};
    testEnrichmentOutput = callPackage ./testEnrichmentOutput.nix {inherit inputs lib;};
    testEquipmentEnrichment = callPackage ./testEquipmentEnrichment.nix {inherit inputs lib;};
    getCMLeaders = import ./challenge-mode-leaderboard.nix {inherit api writers python3Packages;};

    # Trinket testing apps
    trinketTest = callPackage ./trinket-test.nix {inherit lib classes encounter buffs debuffs inputs trinket writeShellApplication;};
    trinketBaselineTest = callPackage ./trinket-baseline-test.nix {inherit lib classes encounter buffs debuffs inputs trinket writeShellApplication;};
    trinketSingleTest = callPackage ./trinket-single-test.nix {inherit lib classes encounter buffs debuffs inputs trinket writeShellApplication;};
    trinketComparisonTest = callPackage ./trinket-comparison-test.nix {inherit lib classes encounter buffs debuffs inputs trinket writeShellApplication;};
  in {
    apps =
      simulation
      // {
        getDB = {
          type = "app";
          program = "${getDB}/bin/getDB";
        };
        testItemLookup = {
          type = "app";
          program = "${testItemLookup}/bin/testItemLookup";
        };
        testEnrichmentOutput = {
          type = "app";
          program = "${testEnrichmentOutput}/bin/testEnrichmentOutput";
        };
        testEquipmentEnrichment = {
          type = "app";
          program = "${testEquipmentEnrichment}/bin/testEquipmentEnrichment";
        };

        # Trinket testing apps
        trinket-test = {
          type = "app";
          program = "${trinketTest}/bin/trinket-test";
        };
        trinket-baseline-test = {
          type = "app";
          program = "${trinketBaselineTest}/bin/trinket-baseline-test";
        };
        trinket-single-test = {
          type = "app";
          program = "${trinketSingleTest}/bin/trinket-single-test";
        };
        trinket-comparison-test = {
          type = "app";
          program = "${trinketComparisonTest}/bin/trinket-comparison-test";
        };
        getCM = {
          type = "app";
          program = "${getCMLeaders}/bin/cm-leaderboard-fetcher";
        };
      };
  };
}
