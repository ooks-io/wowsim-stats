{
  lib,
  inputs,
  api,
  simulation,
  ...
}: {
  perSystem = {
    pkgs,
    config,
    ...
  }: let
    inherit (pkgs) writers python3Packages;
    # utility functions

    # print wowsims wowhead database dump
    getDB = pkgs.callPackage ./utils/getDB.nix {inherit inputs;};

    # Go database CLI tool
    wowsimstats-cli = pkgs.callPackage ../pkgs/database {};

    # general database related scripts

    # creates database schema and tables
    databaseSchema = import ./database/database-schema.nix {inherit writers python3Packages;};
    # populates our database with item information from the wowhead database dump
    populateItems = pkgs.callPackage ./database/populate-items.nix {inherit inputs;};

    # challenge mode scripts

    # main challenge mode leaderboard script
    # setups up database schema, pulls all realms challenge-mode leaderboards
    getCMLeaders = import ./challenge-mode/challenge-mode-leaderboard.nix {inherit api writers python3Packages;};
    # populates player profiles from blizzard api, only includes players flagged with complete_coverage (9/9 recorded records)
    playerProfiles = import ./challenge-mode/player-profiles.nix {inherit writers python3Packages;};
    #
    playerAggregation = import ./challenge-mode/player-aggregation.nix {inherit writers python3Packages;};
    #
    rankingProcessor = import ./challenge-mode/ranking-processor.nix {inherit writers python3Packages;};
    # analyze periods across all dungeons to find optimal API strategy
    periodAnalyzer = import ./utils/period-analyzer.nix {inherit writers python3Packages;};
    # convert simulation data to apps

    # DEPRECATED SCRIPTS
    # parseCMs = import ./deprecated/challenge-mode-parser.nix {inherit api writers python3Packages;};
    # teamLeaderboards = import ./deprecated/team-leaderboard-generator.nix {inherit writers python3Packages;};
    # playerLeaderboards = import ./deprecated/player-leaderboard-generator.nix {inherit writers python3Packages;};

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
        ookstats = {
          type = "app";
          program = lib.getExe config.packages.ookstats;
        };
        default = config.apps.ookstats;
        allSimulations = {
          type = "app";
          program = "${simulation.generateAllSimulationsScript pkgs}/bin/all-simulations";
        };
        getDB = {
          type = "app";
          program = "${getDB}/bin/getDB";
        };
        databaseSchema = {
          type = "app";
          program = "${databaseSchema}/bin/database-schema";
        };
        getCM = {
          type = "app";
          program = "${getCMLeaders}/bin/cm-leaderboard-fetcher";
        };
        playerAggregation = {
          type = "app";
          program = "${playerAggregation}/bin/player-aggregation";
        };
        playerProfiles = {
          type = "app";
          program = "${playerProfiles}/bin/player-profiles";
        };
        populateItems = {
          type = "app";
          program = "${populateItems}/bin/populate-items";
        };
        rankingProcessor = {
          type = "app";
          program = "${rankingProcessor}/bin/ranking-processor";
        };
        periodAnalyzer = {
          type = "app";
          program = "${periodAnalyzer}/bin/period-analyzer";
        };
        wowstats-db = {
          type = "app";
          program = "${wowsimstats-cli}/bin/wowstats-db";
        };
        testGroupSim = {
          type = "app";
          program = "${simulation.generateTestGroupSim pkgs}/bin/test-group-sim";
        };
        # parseCM = {
        #   type = "app";
        #   program = "${parseCMs}/bin/cm-leaderboard-parser";
        # };
        # teamLeaderboards = {
        #   type = "app";
        #   program = "${teamLeaderboards}/bin/team-leaderboard-generator";
        # };
        # playerLeaderboards = {
        #   type = "app";
        #   program = "${playerLeaderboards}/bin/player-leaderboard-generator";
        # };
      };
  };
}
