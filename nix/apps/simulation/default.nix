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
  massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
  inherit (massSimFunctions) mkMassSim mkRaceComparison;

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

  # Race comparison simulations for all DPS specs
  raceComparisonSpecs = [
    {
      class = "death_knight";
      spec = "frost";
      template = "singleTarget";
    }
    {
      class = "death_knight";
      spec = "unholy";
      template = "singleTarget";
    }
    {
      class = "druid";
      spec = "balance";
      template = "singleTarget";
    }
    # { class = "druid"; spec = "feral"; template = "singleTarget"; }
    {
      class = "hunter";
      spec = "beast_mastery";
      template = "singleTarget";
    }
    {
      class = "hunter";
      spec = "marksmanship";
      template = "singleTarget";
    }
    {
      class = "hunter";
      spec = "survival";
      template = "singleTarget";
    }
    {
      class = "mage";
      spec = "arcane";
      template = "singleTarget";
    }
    {
      class = "mage";
      spec = "fire";
      template = "singleTarget";
    }
    {
      class = "mage";
      spec = "frost";
      template = "singleTarget";
    }
    {
      class = "monk";
      spec = "windwalker";
      template = "singleTarget";
    }
    {
      class = "paladin";
      spec = "retribution";
      template = "singleTarget";
    }
    {
      class = "priest";
      spec = "shadow";
      template = "singleTarget";
    }
    {
      class = "rogue";
      spec = "assassination";
      template = "singleTarget";
    }
    {
      class = "rogue";
      spec = "combat";
      template = "singleTarget";
    }
    {
      class = "rogue";
      spec = "subtlety";
      template = "singleTarget";
    }
    {
      class = "shaman";
      spec = "elemental";
      template = "singleTarget";
    }
    {
      class = "shaman";
      spec = "enhancement";
      template = "singleTarget";
    }
    {
      class = "warlock";
      spec = "affliction";
      template = "singleTarget";
    }
    {
      class = "warlock";
      spec = "demonology";
      template = "singleTarget";
    }
    {
      class = "warlock";
      spec = "destruction";
      template = "singleTarget";
    }
    {
      class = "warrior";
      spec = "arms";
      template = "singleTarget";
    }
    {
      class = "warrior";
      spec = "fury";
      template = "singleTarget";
    }
  ];

  # Generate race comparison simulations for single target long encounters
  raceComparisons = lib.listToAttrs (map (specConfig: {
      name = "race-${specConfig.class}-${specConfig.spec}-p1-raid-single-long";
      value = mkRaceComparison {
        class = specConfig.class;
        spec = specConfig.spec;
        encounter = encounter.raid.long.singleTarget;
        iterations = 10000;
        phase = "p1";
        encounterType = "raid";
        targetCount = "single";
        duration = "long";
        template = specConfig.template;
      };
    })
    raceComparisonSpecs);

  # Script that runs all simulations
  allSimulationsScript = pkgs.writeShellApplication {
    name = "all-simulations";
    text = ''
      echo "Running all WoW simulations..."

      echo ""
      echo "=== DPS Rankings ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${massSimulations.${name}.script}/bin/${massSimulations.${name}.metadata.output}-aggregator
      '') (lib.attrNames massSimulations)}

      echo ""
      echo "=== Race Comparisons ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${raceComparisons.${name}.script}/bin/${raceComparisons.${name}.metadata.output}-aggregator
      '') (lib.attrNames raceComparisons)}

      echo ""
      echo "All simulations completed successfully!"
      echo ""
      echo "Generated DPS rankings:"
      ls -la web/public/data/rankings/*.json 2>/dev/null || echo "No ranking files found"
      echo ""
      echo "Generated race comparisons:"
      find web/public/data/comparison -name "*.json" 2>/dev/null || echo "No comparison files found"
    '';
    runtimeInputs = [pkgs.coreutils pkgs.findutils];
  };

  # Convert mass simulations to apps and add the all-simulations app
  simulationApps =
    lib.mapAttrs (name: massSim: {
      type = "app";
      program = "${massSim.script}/bin/${massSim.metadata.output}-aggregator";
    })
    massSimulations;

  # Convert race comparisons to apps
  raceComparisonApps =
    lib.mapAttrs (name: raceComp: {
      type = "app";
      program = "${raceComp.script}/bin/${raceComp.metadata.output}-aggregator";
    })
    raceComparisons;
in
  simulationApps
  // raceComparisonApps
  // {
    allSimulations = {
      type = "app";
      program = "${allSimulationsScript}/bin/all-simulations";
    };
  }
