{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  ...
}: let
  massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
  inherit (massSimFunctions) mkMassSim mkRaceComparison;

  trinketComparison = import ./mkTrinketComparison.nix {inherit lib pkgs classes encounter buffs debuffs inputs trinket;};
  inherit (trinketComparison) mkTrinketComparison;

  # TODO: abstract this.
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
      specs = "dps";
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
      specs = "dps";
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
    dps-preRaid-raid-single-long = mkMassSim {
      specs = "dps"; # shortcut to all DPS classes templates
      encounter = encounter.raid.long.singleTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "single";
      duration = "long";
      template = "singleTarget";
    };

    dps-preRaid-raid-three-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.threeTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "three";
      duration = "long";
      template = "multiTarget";
    };

    dps-preRaid-raid-cleave-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.cleave;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "long";
      template = "cleave";
    };

    dps-preRaid-raid-ten-long = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.long.tenTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "ten";
      duration = "long";
      template = "multiTarget";
    };
    dps-preRaid-raid-single-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.singleTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "single";
      duration = "short";
      template = "singleTarget";
    };

    dps-preRaid-raid-three-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.threeTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "three";
      duration = "short";
      template = "multiTarget";
    };

    dps-preRaid-raid-cleave-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.cleave;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "short";
      template = "cleave";
    };

    dps-preRaid-raid-ten-short = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.short.tenTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "ten";
      duration = "short";
      template = "multiTarget";
    };
    dps-preRaid-raid-single-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.singleTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "single";
      duration = "burst";
      template = "singleTarget";
    };

    dps-preRaid-raid-three-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.threeTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "three";
      duration = "burst";
      template = "multiTarget";
    };

    dps-preRaid-raid-cleave-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.cleave;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "cleave";
      duration = "burst";
      template = "cleave";
    };

    dps-preRaid-raid-ten-burst = mkMassSim {
      specs = "dps";
      encounter = encounter.raid.burst.tenTarget;
      iterations = 10000;
      phase = "preRaid";
      encounterType = "raid";
      targetCount = "ten";
      duration = "burst";
      template = "multiTarget";
    };
  };

  # for all race combinations
  # we will only run race benchmarks for p1 geartsets for now
  raceScenarios = [
    {
      targetCount = "single";
      duration = "long";
      encounter = encounter.raid.long.singleTarget;
      template = "singleTarget";
    }
    {
      targetCount = "single";
      duration = "short";
      encounter = encounter.raid.short.singleTarget;
      template = "singleTarget";
    }
    {
      targetCount = "single";
      duration = "burst";
      encounter = encounter.raid.burst.singleTarget;
      template = "singleTarget";
    }

    {
      targetCount = "three";
      duration = "long";
      encounter = encounter.raid.long.threeTarget;
      template = "multiTarget";
    }
    {
      targetCount = "three";
      duration = "short";
      encounter = encounter.raid.short.threeTarget;
      template = "multiTarget";
    }
    {
      targetCount = "three";
      duration = "burst";
      encounter = encounter.raid.burst.threeTarget;
      template = "multiTarget";
    }

    {
      targetCount = "cleave";
      duration = "long";
      encounter = encounter.raid.long.cleave;
      template = "cleave";
    }
    {
      targetCount = "cleave";
      duration = "short";
      encounter = encounter.raid.short.cleave;
      template = "cleave";
    }
    {
      targetCount = "cleave";
      duration = "burst";
      encounter = encounter.raid.burst.cleave;
      template = "cleave";
    }

    {
      targetCount = "ten";
      duration = "long";
      encounter = encounter.raid.long.tenTarget;
      template = "multiTarget";
    }
    {
      targetCount = "ten";
      duration = "short";
      encounter = encounter.raid.short.tenTarget;
      template = "multiTarget";
    }
    {
      targetCount = "ten";
      duration = "burst";
      encounter = encounter.raid.burst.tenTarget;
      template = "multiTarget";
    }
  ];

  raceComparisonSpecs = [
    {
      class = "death_knight";
      spec = "frost";
    }
    {
      class = "death_knight";
      spec = "unholy";
    }
    {
      class = "druid";
      spec = "balance";
    }
    # { class = "druid"; spec = "feral"; }
    {
      class = "hunter";
      spec = "beast_mastery";
    }
    {
      class = "hunter";
      spec = "marksmanship";
    }
    {
      class = "hunter";
      spec = "survival";
    }
    {
      class = "mage";
      spec = "arcane";
    }
    {
      class = "mage";
      spec = "fire";
    }
    {
      class = "mage";
      spec = "frost";
    }
    {
      class = "monk";
      spec = "windwalker";
    }
    {
      class = "paladin";
      spec = "retribution";
    }
    {
      class = "priest";
      spec = "shadow";
    }
    {
      class = "rogue";
      spec = "assassination";
    }
    {
      class = "rogue";
      spec = "combat";
    }
    {
      class = "rogue";
      spec = "subtlety";
    }
    {
      class = "shaman";
      spec = "elemental";
    }
    {
      class = "shaman";
      spec = "enhancement";
    }
    {
      class = "warlock";
      spec = "affliction";
    }
    {
      class = "warlock";
      spec = "demonology";
    }
    {
      class = "warlock";
      spec = "destruction";
    }
    {
      class = "warrior";
      spec = "arms";
    }
    {
      class = "warrior";
      spec = "fury";
    }
  ];

  # trinket scenarios - same as race scenarios for now
  trinketScenarios = raceScenarios;

  # trinket comparison specs - all DPS specs with appropriate trinket categories
  trinketComparisonSpecs = [
    # Agility specs
    {
      class = "monk";
      spec = "windwalker";
      trinketCategory = "agility";
    }
    {
      class = "hunter";
      spec = "beast_mastery";
      trinketCategory = "agility";
    }
    {
      class = "hunter";
      spec = "marksmanship";
      trinketCategory = "agility";
    }
    {
      class = "hunter";
      spec = "survival";
      trinketCategory = "agility";
    }
    {
      class = "rogue";
      spec = "assassination";
      trinketCategory = "agility";
    }
    {
      class = "rogue";
      spec = "combat";
      trinketCategory = "agility";
    }
    {
      class = "rogue";
      spec = "subtlety";
      trinketCategory = "agility";
    }
    {
      class = "shaman";
      spec = "enhancement";
      trinketCategory = "agility";
    }

    # Intellect specs
    {
      class = "mage";
      spec = "arcane";
      trinketCategory = "intellect";
    }
    {
      class = "mage";
      spec = "fire";
      trinketCategory = "intellect";
    }
    {
      class = "mage";
      spec = "frost";
      trinketCategory = "intellect";
    }
    {
      class = "druid";
      spec = "balance";
      trinketCategory = "intellectHybrid";
    }
    {
      class = "priest";
      spec = "shadow";
      trinketCategory = "intellectHybrid";
    }
    {
      class = "warlock";
      spec = "affliction";
      trinketCategory = "intellect";
    }
    {
      class = "warlock";
      spec = "demonology";
      trinketCategory = "intellect";
    }
    {
      class = "warlock";
      spec = "destruction";
      trinketCategory = "intellect";
    }
    {
      class = "shaman";
      spec = "elemental";
      trinketCategory = "intellectHybrid";
    }

    # Strength specs
    {
      class = "death_knight";
      spec = "frost";
      trinketCategory = "strength";
    }
    {
      class = "death_knight";
      spec = "unholy";
      trinketCategory = "strength";
    }
    {
      class = "paladin";
      spec = "retribution";
      trinketCategory = "strength";
    }
    {
      class = "warrior";
      spec = "arms";
      trinketCategory = "strength";
    }
    {
      class = "warrior";
      spec = "fury";
      trinketCategory = "strength";
    }
  ];

  # generate all trinket comparison combinations (specs × scenarios)
  trinketComparisons = lib.listToAttrs (lib.flatten (map (
      specConfig:
        map (scenario: {
          name = "trinket-${specConfig.class}-${specConfig.spec}-p1-raid-${scenario.targetCount}-${scenario.duration}";
          value = mkTrinketComparison {
            class = specConfig.class;
            spec = specConfig.spec;
            encounter = scenario.encounter;
            trinketCategory = specConfig.trinketCategory;
            iterations = 10000;
            phase = "p1";
            encounterType = "raid";
            targetCount = scenario.targetCount;
            duration = scenario.duration;
            template = scenario.template;
          };
        })
        trinketScenarios
    )
    trinketComparisonSpecs));

  # generate all race comparison combinations (specs × scenarios)
  raceComparisons = lib.listToAttrs (lib.flatten (map (
      specConfig:
        map (scenario: {
          name = "race-${specConfig.class}-${specConfig.spec}-p1-raid-${scenario.targetCount}-${scenario.duration}";
          value = mkRaceComparison {
            class = specConfig.class;
            spec = specConfig.spec;
            encounter = scenario.encounter;
            iterations = 10000;
            phase = "p1";
            encounterType = "raid";
            targetCount = scenario.targetCount;
            duration = scenario.duration;
            template = scenario.template;
          };
        })
        raceScenarios
    )
    raceComparisonSpecs));

  # script that runs all simulations
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
      echo "=== Trinket Comparisons ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${trinketComparisons.${name}.script}/bin/${trinketComparisons.${name}.metadata.output}-aggregator
      '') (lib.attrNames trinketComparisons)}

      echo ""
      echo "All simulations completed successfully!"
      echo ""
      echo "Generated DPS rankings:"
      ls -la web/public/data/rankings/*.json 2>/dev/null || echo "No ranking files found"
      echo ""
      echo "Generated race comparisons:"
      find web/public/data/comparison/race -name "*.json" 2>/dev/null || echo "No race comparison files found"
      echo ""
      echo "Generated trinket comparisons:"
      find web/public/data/comparison/trinkets -name "*.json" 2>/dev/null || echo "No trinket comparison files found"
    '';
    runtimeInputs = [pkgs.coreutils pkgs.findutils];
  };

  # nix app
  simulationApps =
    lib.mapAttrs (name: massSim: {
      type = "app";
      program = "${massSim.script}/bin/${massSim.metadata.output}-aggregator";
    })
    massSimulations;

  raceComparisonApps =
    lib.mapAttrs (name: raceComp: {
      type = "app";
      program = "${raceComp.script}/bin/${raceComp.metadata.output}-aggregator";
    })
    raceComparisons;

  trinketComparisonApps =
    lib.mapAttrs (name: trinketComp: {
      type = "app";
      program = "${trinketComp.script}/bin/${trinketComp.metadata.output}-aggregator";
    })
    trinketComparisons;
in
  simulationApps
  // raceComparisonApps
  // trinketComparisonApps
  // {
    allSimulations = {
      type = "app";
      program = "${allSimulationsScript}/bin/all-simulations";
    };
  }
