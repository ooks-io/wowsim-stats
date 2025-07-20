{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  ...
}: let
  # Mass simulations using mkMassSim
  mkMassSim = (import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;}).mkMassSim;

  massSimulations = {
    dps-p1-raid-single-long = mkMassSim {
      specs = "dps"; # shortcut to all DPS classes templates
      encounter = encounter.raid.long.singleTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "single";
      duration = "long";
    };

    dps-p1-raid-three-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.threeTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "three";
      duration = "long";
    };

    dps-p1-raid-cleave-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.cleave;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "long";
    };

    dps-p1-raid-ten-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.tenTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "ten";
      duration = "long";
    };
  };

  # Script that runs all simulations
  allSimulationsScript = pkgs.writeShellApplication {
    name = "all-simulations";
    text = ''
      echo "Running all WoW simulations..."

      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${massSimulations.${name}.script}/bin/${massSimulations.${name}.metadata.output}-aggregator
      '') (lib.attrNames massSimulations)}

      echo ""
      echo "All simulations completed successfully!"
      echo "Generated files:"
      ls -la web/public/data/*.json 2>/dev/null || echo "No JSON files found"
    '';
    runtimeInputs = [pkgs.coreutils];
  };

  # Convert mass simulations to apps and add the all-simulations app
  simulationApps =
    lib.mapAttrs (name: massSim: {
      type = "app";
      program = "${massSim.script}/bin/${massSim.metadata.output}-aggregator";
    })
    massSimulations;
in
  simulationApps
  // {
    allSimulations = {
      type = "app";
      program = "${allSimulationsScript}/bin/all-simulations";
    };
  }
