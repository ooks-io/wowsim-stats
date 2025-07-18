{
  lib,
  self,
  ...
}: let
  raidBuffsLib = import ./raidBuffs.nix {inherit lib;};
  raidDebuffsLib = import ./raidDebuffs.nix {inherit lib;};
  encountersLib = import ./encounters.nix {inherit lib;};

  # Base configuration that most specs will use
  baseConfig = {
    iterations = 12500;
    randomSeed = "4260890857";
    debugFirstIteration = true;
    distanceFromTarget = 5;
    reactionTimeMs = 100;
    raidBuffs = raidBuffsLib.fullBuffs;
    raidDebuffs = raidDebuffsLib.fullDebuffs;
  };

  # Melee base config
  meleeConfig = baseConfig;

  # Ranged base config
  rangedConfig =
    baseConfig
    // {
      distanceFromTarget = 25;
    };

  # Caster base config
  casterConfig =
    baseConfig
    // {
      distanceFromTarget = 25;
    };

  # Class-specific configurations
  monk = {
    windwalker = {
      singleTarget =
        meleeConfig
        // {
          class = "monk";
          spec = "windwalker";
          race = "orc";
          gearset = "p1_bis_dw";
          apl = "default";
          consumables = {
            prepotId = 76089;
            potId = 76089;
            flaskId = 76084;
            foodId = 74648;
          };
          profession1 = "Engineering";
          profession2 = "Tailoring";
          talents = "213322";
          glyphs = {
            major1 = 85697;
            major2 = 87900;
            minor1 = 90715;
          };
          buffs = {};
          debuffs = {};
          targetDummies = 1;
          options = {};
        };

      multiTarget =
        monk.windwalker.singleTarget
        // {
          targetDummies = 9;
        };

      # Challenge Mode variants
      challengeMode =
        monk.windwalker.singleTarget
        // {
          challengeMode = true;
        };

      challengeModeMultiTarget =
        monk.windwalker.singleTarget
        // {
          challengeMode = true;
          targetDummies = 9;
        };
    };

    brewmaster = {
      singleTarget =
        meleeConfig
        // {
          class = "monk";
          spec = "brewmaster";
          race = "Pandaren";
          gearset = "p1_bis";
          apl = "default";
          consumables = {
            prepotId = 76089;
            potId = 76089;
            flaskId = 76084;
            foodId = 74648;
          };
          profession1 = "Engineering";
          profession2 = "Tailoring";
          talents = "213322";
          glyphs = {
            major1 = 85697;
            major2 = 87900;
            minor1 = 90715;
          };
          buffs = {};
          debuffs = {};
          targetDummies = 1;
          options = {};
        };

      # Challenge Mode variant
      challengeMode =
        monk.brewmaster.singleTarget
        // {
          challengeMode = true;
        };
    };

    mistweaver = {
      singleTarget =
        casterConfig
        // {
          class = "monk";
          spec = "mistweaver";
          race = "Pandaren";
          gearset = "p1_bis";
          apl = "default";
          consumables = {
            prepotId = 76089;
            potId = 76089;
            flaskId = 76084;
            foodId = 74648;
          };
          profession1 = "Engineering";
          profession2 = "Tailoring";
          talents = "213322";
          glyphs = {
            major1 = 85697;
            major2 = 87900;
            minor1 = 90715;
          };
          buffs = {};
          debuffs = {};
          targetDummies = 1;
          options = {};
        };

      # Challenge Mode variant
      challengeMode =
        monk.mistweaver.singleTarget
        // {
          challengeMode = true;
        };
    };
  };

  # Generate simulation function
  generateSimulation = classConfig: encounterType: let
    generateInfile = import ./generateInfile.nix {inherit lib self;};
    encounter = encounterType;
  in
    generateInfile (classConfig // {inherit encounter;});
in {
  inherit baseConfig meleeConfig rangedConfig casterConfig monk generateSimulation;

  # Export encounters for easy access
  encounters = encountersLib;

  # Export buff/debuff libs for customization
  raidBuffs = raidBuffsLib;
  raidDebuffs = raidDebuffsLib;
}

