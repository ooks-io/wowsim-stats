{encounter, ...}: {
  # Common configuration shared across all simulations
  common = {
    iterations = 10000;
    encounterType = "raid";
    specs = "dps"; # shortcut to all DPS classes templates
  };

  # Available phases for simulations
  phases = ["p1" "preRaid"];

  # Target count configurations with their corresponding templates and encounters
  targetConfigs = {
    single = {
      template = "singleTarget";
      encounters = {
        long = encounter.raid.long.singleTarget;
        short = encounter.raid.short.singleTarget;
        burst = encounter.raid.burst.singleTarget;
      };
    };
    three = {
      template = "multiTarget";
      encounters = {
        long = encounter.raid.long.threeTarget;
        short = encounter.raid.short.threeTarget;
        burst = encounter.raid.burst.threeTarget;
      };
    };
    cleave = {
      template = "cleave";
      encounters = {
        long = encounter.raid.long.cleave;
        short = encounter.raid.short.cleave;
        burst = encounter.raid.burst.cleave;
      };
    };
    ten = {
      template = "multiTarget";
      encounters = {
        long = encounter.raid.long.tenTarget;
        short = encounter.raid.short.tenTarget;
        burst = encounter.raid.burst.tenTarget;
      };
    };
  };

  # Available durations
  durations = ["long" "short" "burst"];

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
    # { class = "druid"; spec = "feral"; } #
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

  trinketComparisonSpecs = [
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
}

