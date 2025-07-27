{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  assassination = {
    defaultRace = "human";

    talents = {
      mfd = "321232";
    };

    glyphs = {
      default = {
        major1 = 45761; # vendetta
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
      spec = "assassination";
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
        apl = "assassination";
        p1.gearset = "p1_assassination_t14";
        preRaid.gearset = "preraid_assassination";
        talents = assassination.talents.mfd;
        glyphs = assassination.glyphs.default;
      };

      multiTarget = {
        apl = "assassination";
        p1.gearset = "p1_assassination_t14";
        preRaid.gearset = "preraid_assassination";
        talents = assassination.talents.mfd;
        glyphs = assassination.glyphs.default;
      };

      cleave = {
        apl = "assassination";
        p1.gearset = "p1_assassination_t14";
        preRaid.gearset = "preraid_assassination";
        talents = assassination.talents.mfd;
        glyphs = assassination.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_assassination_t14";
        talents = assassination.talents.mfd;
        glyphs = assassination.glyphs.default;
      };
    };
  };
in
  assassination
