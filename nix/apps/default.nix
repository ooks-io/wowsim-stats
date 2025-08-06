{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  api,
  simulation,
  ...
}: {
  perSystem = {pkgs, ...}: let
    inherit (pkgs) writers python3Packages;
    getDB = pkgs.callPackage ./getDB.nix {inherit inputs;};
    getCMLeaders = import ./challenge-mode-leaderboard.nix {inherit api writers python3Packages;};
    parseCMs = import ./challenge-mode-parser.nix {inherit api writers python3Packages;};
    # Convert simulation data to apps
    simulationApps =
      lib.mapAttrs (name: sim: {
        type = "app";
        program = "${sim.script}/bin/${sim.metadata.output}-aggregator";
      })
      (simulation.generateMassSimulations pkgs);

    raceComparisonApps =
      lib.mapAttrs (name: raceComp: {
        type = "app";
        program = "${raceComp.script}/bin/${raceComp.metadata.output}-aggregator";
      })
      (simulation.generateRaceComparisons pkgs);

    trinketComparisonApps =
      lib.mapAttrs (name: trinketComp: {
        type = "app";
        program = "${trinketComp.script}/bin/${trinketComp.metadata.output}-aggregator";
      })
      (simulation.generateTrinketComparisons pkgs);
  in {
    apps =
      simulationApps
      // raceComparisonApps
      // trinketComparisonApps
      // {
        allSimulations = {
          type = "app";
          program = "${simulation.generateAllSimulationsScript pkgs}/bin/all-simulations";
        };
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
