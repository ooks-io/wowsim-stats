{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) strength;

  arms = {
    defaultRace = "orc";

    talents = {
      bloodbath = "113332";
    };

    glyphs = {
      default = {
        major1 = 67482; # bull rush
        major2 = 43399; # unending rage
        major3 = 45792; # death from above
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "night_elf"
        "gnome"
        "draenei"
        "worgen"
        "orc"
        "undead"
        "tauren"
        "troll"
        "blood_elf"
        "goblin"
        "alliance_pandaren"
      ];
      class = "warrior";
      spec = "arms";
      consumables = strength;
      profession1 = "engineering";
      profession2 = "blacksmithing";
      distanceFromTarget = 9;
      options = {};

      singleTarget = {
        apl = "arms";
        p1.gearset = "p1_arms_bis";
        preRaid.gearset = "p1_prebis_rich";
        talents = arms.talents.bloodbath;
        glyphs = arms.glyphs.default;
      };

      multiTarget = {
        apl = "arms";
        p1.gearset = "p1_arms_bis";
        preRaid.gearset = "p1_prebis_rich";
        talents = arms.talents.bloodbath;
        glyphs = arms.glyphs.default;
      };

      cleave = {
        apl = "arms";
        p1.gearset = "p1_arms_bis";
        preRaid.gearset = "p1_prebis_rich";
        talents = arms.talents.bloodbath;
        glyphs = arms.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_arms_bis";
        talents = arms.talents.bloodbath;
        glyphs = arms.glyphs.default;
      };
    };
  };
in
  arms
