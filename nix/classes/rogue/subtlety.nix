{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  subtlety = {
    defaultRace = "troll";

    talents = {
      anticipation = "321233";
    };

    glyphs = {
      default = {
        major1 = 42970; # Hemorraghing Veins
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "night_elf"
        "gnome"
        "worgen"
        "orc"
        "undead"
        "troll"
        "blood_elf"
        "goblin"
        "alliance_pandaren"
      ];
      class = "rogue";
      spec = "subtlety";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "alchemy";
      distanceFromTarget = 5;
      options = {
        classOptions = {
          lethalPoison = "DeadlyPoison";
          startingOverkillDuration = 20;
          vanishBreakTime = 0.1;
        };
      };

      singleTarget = {
        apl = "subtlety";
        p1.gearset = "p1_subtlety_t14";
        preRaid.gearset = "preraid_subtlety";
        talents = subtlety.talents.anticipation;
        glyphs = subtlety.glyphs.default;
      };

      multiTarget = {
        apl = "subtlety";
        p1.gearset = "p1_subtlety_t14";
        preRaid.gearset = "preraid_subtlety";
        talents = subtlety.talents.anticipation;
        glyphs = subtlety.glyphs.default;
      };

      cleave = {
        apl = "subtlety";
        p1.gearset = "p1_subtlety_t14";
        preRaid.gearset = "preraid_subtlety";
        talents = subtlety.talents.anticipation;
        glyphs = subtlety.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_subtlety_t14";
        talents = subtlety.talents.anticipation;
        glyphs = subtlety.glyphs.default;
      };
    };
  };
in
  subtlety
