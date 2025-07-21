{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) strength;

  fury = {
    defaultRace = "worgen";

    talents = {
      stormBolt = "113133";
      avatar = "113131";
    };

    glyphs = {
      default = {
        major1 = 67482;
        major2 = 45792;
        major3 = 43399;
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
      spec = "fury";
      consumables = strength;
      profession1 = "engineering";
      profession2 = "blacksmithing";
      distanceFromTarget = 15;
      options = {};

      singleTarget = {
        apl = "default";
        p1.gearset = "p1_fury_tg";
        preRaid.gearset = "preraid_fury_tg";
        talents = fury.talents.stormBolt;
        glyphs = fury.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1_fury_tg";
        preRaid.gearset = "preraid_fury_tg";
        talents = fury.talents.stormBolt;
        glyphs = fury.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1_fury_tg";
        preRaid.gearset = "preraid_fury_tg";
        talents = fury.talents.stormBolt;
        glyphs = fury.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_fury_tg";
        talents = fury.talents.stormBolt;
        glyphs = fury.glyphs.default;
      };
    };
  };
in
  fury
