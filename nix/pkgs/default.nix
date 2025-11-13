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

    go-libsql-src = pkgs.fetchFromGitHub {
      owner = "tursodatabase";
      repo = "go-libsql";
      rev = "60e59c7150f4";
      hash = "sha256-TuD/7AWkC13lQct2QguO31dP1th+nD0ZTPqD+RUfnu8=";
    };

    ookstats = callPackage ./ookstats {inherit go-libsql-src wowsims-db;};
  in {
    packages =
      simulationPackages
      // racePackages
      // trinketPackages
      // {
        inherit ookstats;
        allSimulations = simulation.generateAllSimulationsScript pkgs;
        simInputs = simulation.generateSimInputs pkgs;
        testRaid = callPackage ./testRaid.nix {
          inherit lib classes encounter buffs debuffs wowsimcli;
        };
        testGroupSim = simulation.generateTestGroupSim pkgs;
      };
  };
}
