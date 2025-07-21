{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) strength;

  unholy = {
    defaultRace = "orc";

    talents = {
      gorfiendsGrasp = "221111";
    };

    glyphs = {
      default = {
        major1 = 43533;
        major2 = 43548;
        major3 = 104047;
        minor1 = 43550;
        minor2 = 45806;
        minor3 = 43539;
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
      ];

      class = "death_knight";
      spec = "unholy";
      consumables = strength;
      profession1 = "engineering";
      profession2 = "blacksmithing";
      distanceFromTarget = 5;
      options = {};

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "prebis";
        talents = unholy.talents.gorfiendsGrasp;
        glyphs = unholy.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "prebis";
        talents = unholy.talents.gorfiendsGrasp;
        glyphs = unholy.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "prebis";
        talents = unholy.talents.gorfiendsGrasp;
        glyphs = unholy.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = unholy.talents.gorfiendsGrasp;
        glyphs = unholy.glyphs.default;
      };
    };
  };
in
  unholy
