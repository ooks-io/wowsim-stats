{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  windwalker = {
    defaultRace = "orc";

    talents = {
      xuen = "213322";
      rjw = "233321";
    };

    glyphs = {
      default = {
        major1 = 85697; # spinning crane kick
        major2 = 87900; # touch of karma
        minor1 = 90715; # blackout kick
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "night_elf"
        "gnome"
        "draenei"
        "orc"
        "undead"
        "tauren"
        "troll"
        "blood_elf"
        "alliance_pandaren"
      ];
      class = "monk";
      spec = "windwalker";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 5;
      options = {};

      singleTarget = {
        apl = "default";
        p1.gearset = "p1_bis_dw";
        preRaid.gearset = "p1_prebis_dw";
        talents = windwalker.talents.xuen;
        glyphs = windwalker.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1_bis_dw";
        preRaid.gearset = "p1_prebis_dw";
        talents = windwalker.talents.rjw;
        glyphs = windwalker.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1_bis_dw";
        preRaid.gearset = "p1_prebis_dw";
        talents = windwalker.talents.rjw;
        glyphs = windwalker.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_bis_dw";
        talents = windwalker.talents.xuen;
        glyphs = windwalker.glyphs.default;
      };
    };
  };
in
  windwalker
