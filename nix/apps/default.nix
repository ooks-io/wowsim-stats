{
  lib,
  inputs,
  api,
  simulation,
  ...
}: {
  perSystem = {pkgs, ...}: let
    inherit (pkgs) writers python3Packages;
    getDB = pkgs.callPackage ./getDB.nix {inherit inputs;};
    getCMLeaders = import ./challenge-mode-leaderboard.nix {inherit api writers python3Packages;};
    parseCMs = import ./challenge-mode-parser.nix {inherit api writers python3Packages;};
    teamLeaderboards = import ./team-leaderboard-generator.nix {inherit writers python3Packages;};
    playerLeaderboards = import ./player-leaderboard-generator.nix {inherit writers python3Packages;};
    # Convert simulation data to apps
    simulationApps =
      lib.mapAttrs (_name: sim: {
        type = "app";
        program = "${sim.script}/bin/${sim.metadata.output}-aggregator";
      })
      (simulation.generateMassSimulations pkgs);

    raceComparisonApps =
      lib.mapAttrs (_name: raceComp: {
        type = "app";
        program = "${raceComp.script}/bin/${raceComp.metadata.output}-aggregator";
      })
      (simulation.generateRaceComparisons pkgs);

    trinketComparisonApps =
      lib.mapAttrs (_name: trinketComp: {
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
        teamLeaderboards = {
          type = "app";
          program = "${teamLeaderboards}/bin/team-leaderboard-generator";
        };
        playerLeaderboards = {
          type = "app";
          program = "${playerLeaderboards}/bin/player-leaderboard-generator";
        };
        testGroupSim = {
          type = "app";
          program = "${simulation.generateTestGroupSim pkgs}/bin/test-group-sim";
        };
      };
  };
}
