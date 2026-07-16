{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  simulation,
  inputs,
  ...
}: {
  perSystem = {
    pkgs,
    inputs',
    ...
  }: let
    inherit (pkgs) callPackage;
    inherit (inputs'.wowsims.packages) wowsimcli;

    wowsims-db = "${inputs.wowsims}/assets/database/db.json";

    # Convert simulation data to packages
    simulationPackages = lib.mapAttrs (_name: sim: sim.script) (simulation.generateMassSimulations pkgs);
    racePackages = lib.mapAttrs (_name: sim: sim.script) (simulation.generateRaceComparisons pkgs);
    trinketPackages = lib.mapAttrs (_name: sim: sim.script) (simulation.generateTrinketComparisons pkgs);

    ookstats = callPackage ./ookstats {inherit wowsims-db;};
    ookstats-deploy = callPackage ./ookstats-deploy {inherit ookstats;};
  in {
    packages =
      simulationPackages
      // racePackages
      // trinketPackages
      // {
        inherit ookstats ookstats-deploy;
        allSimulations = simulation.generateAllSimulationsScript pkgs;
        simInputs = simulation.generateSimInputs pkgs;
        testRaid = callPackage ./testRaid.nix {
          inherit lib classes encounter buffs debuffs wowsimcli;
        };
        testGroupSim = simulation.generateTestGroupSim pkgs;
      };
  };
}
