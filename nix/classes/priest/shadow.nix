{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  shadow = {
    defaultRace = "troll";

    talents = {
      halo = "223113";
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
        "draenei"
        "worgen"
        "undead"
        "tauren"
        "troll"
        "blood_elf"
        "goblin"
        "alliance_pandaren"
      ];
      class = "priest";
      spec = "shadow";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 28;
      options = {
        classOptions = {armor = "InnerFire";};
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "pre_raid";
        talents = shadow.talents.halo;
        glyphs = shadow.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "pre_raid";
        talents = shadow.talents.halo;
        glyphs = shadow.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "pre_raid";
        talents = shadow.talents.halo;
        glyphs = shadow.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = shadow.talents.halo;
        glyphs = shadow.glyphs.default;
      };
    };
  };
in
  shadow
