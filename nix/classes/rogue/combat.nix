{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  combat = {
    defaultRace = "human";

    talents = {
      anticipation = "321213";
    };

    glyphs = {
      default = {};
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
      spec = "combat";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "jewelcrafting";
      distanceFromTarget = 5;
      options = {
        classOptions = {
          lethalPoison = "DeadlyPoison";
          startingOverkillDuration = 20;
          vanishBreakTime = 0.1;
        };
      };

      singleTarget = {
        apl = "combat";
        p1.gearset = "p1_combat_t14";
        preRaid.gearset = "preraid_combat";
        talents = combat.talents.anticipation;
        glyphs = combat.glyphs.default;
      };

      multiTarget = {
        apl = "combat";
        p1.gearset = "p1_combat_t14";
        preRaid.gearset = "preraid_combat";
        talents = combat.talents.anticipation;
        glyphs = combat.glyphs.default;
      };

      cleave = {
        apl = "combat";
        p1.gearset = "p1_combat_t14";
        preRaid.gearset = "preraid_combat";
        talents = combat.talents.anticipation;
        glyphs = combat.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_combat_t14";
        talents = combat.talents.anticipation;
        glyphs = combat.glyphs.default;
      };
    };
  };
in
  combat
