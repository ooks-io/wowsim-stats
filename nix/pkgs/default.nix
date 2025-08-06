{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  simulation,
  ...
}: {
  perSystem = {
    pkgs,
    inputs',
    ...
  }: let
    inherit (pkgs) callPackage;
    inherit (inputs'.wowsims.packages) wowsimcli;
    
    # Convert simulation data to packages
    simulationPackages = lib.mapAttrs (name: sim: sim.script) (simulation.generateMassSimulations pkgs);
    racePackages = lib.mapAttrs (name: sim: sim.script) (simulation.generateRaceComparisons pkgs);
    trinketPackages = lib.mapAttrs (name: sim: sim.script) (simulation.generateTrinketComparisons pkgs);
  in {
    packages = 
      simulationPackages
      // racePackages
      // trinketPackages
      // {
        allSimulations = simulation.generateAllSimulationsScript pkgs;
        testRaid = callPackage ./testRaid.nix {
          inherit lib classes encounter buffs debuffs wowsimcli;
        };
      };
  };
}
