{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  marksmanship = {
    defaultRace = "worgen";

    talents = {
      glaiveToss = "312111";
    };

    glyphs = {
      default = {
        major1 = 42909; # animal bond
        major2 = 42903; # deterrence
        major3 = 42914; # aimed shot
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "night_elf"
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
      class = "hunter";
      spec = "marksmanship";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "leatherworking";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          petType = "Wolf";
          petUptime = 1;
          useHunterMark = true;
        };
      };

      singleTarget = {
        apl = "mm";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = marksmanship.talents.glaiveToss;
        glyphs = marksmanship.glyphs.default;
      };

      multiTarget = {
        apl = "mm";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = marksmanship.talents.glaiveToss;
        glyphs = marksmanship.glyphs.default;
      };

      cleave = {
        apl = "mm";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = marksmanship.talents.glaiveToss;
        glyphs = marksmanship.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = marksmanship.talents.glaiveToss;
        glyphs = marksmanship.glyphs.default;
      };
    };
  };
in
  marksmanship
