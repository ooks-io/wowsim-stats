{
  lib,
  inputs,
  simulation,
  ...
}: {
  perSystem = {
    pkgs,
    config,
    ...
  }: let
    # print wowsims wowhead database dump
    getDB = pkgs.callPackage ./utils/getDB.nix {inherit inputs;};

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
        ookstats-deploy = {
          type = "app";
          program = lib.getExe config.packages.ookstats-deploy;
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
      };
  };
}
