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
      template = "singleTarget";
    };

    dps-p1-raid-three-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.threeTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "three";
      duration = "long";
      template = "multiTarget";
    };

    dps-p1-raid-cleave-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.cleave;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "long";
      template = "cleave";
    };

    dps-p1-raid-ten-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.tenTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "ten";
      duration = "long";
      template = "multiTarget";
    };
    dps-p1-raid-single-short = mkMassSim {
      specs = "dps"; # shortcut to all DPS classes templates
      encounter = encounter.raid.short.singleTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "single";
      duration = "short";
      template = "singleTarget";
    };

    dps-p1-raid-three-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.threeTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "three";
      duration = "short";
      template = "multiTarget";
    };

    dps-p1-raid-cleave-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.cleave;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "short";
      template = "cleave";
    };

    dps-p1-raid-ten-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.tenTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "ten";
      duration = "short";
      template = "multiTarget";
    };
    dps-p1-raid-single-burst = mkMassSim {
      specs = "dps"; # burstcut to all DPS classes templates
      encounter = encounter.raid.burst.singleTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "single";
      duration = "burst";
      template = "singleTarget";
    };

    dps-p1-raid-three-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.threeTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "three";
      duration = "burst";
      template = "multiTarget";
    };

    dps-p1-raid-cleave-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.cleave;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "burst";
      template = "cleave";
    };

    dps-p1-raid-ten-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.tenTarget;
      iterations = 10000;
      phase = "p1";
      encounterType = "raid";
      targetCount = "ten";
      duration = "burst";
      template = "multiTarget";
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
