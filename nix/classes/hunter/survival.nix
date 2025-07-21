{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  survival = {
    defaultRace = "worgen";

    talents = {
      serpent = "321232";
    };

    glyphs = {
      default = {
        major1 = 42909; # animal bond
        major2 = 42903; # deterrence
        major3 = 42899; # liberation
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
      spec = "survival";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "jewelcrafting";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          petType = "Wolf";
          petUptime = 1;
          useHunterMark = true;
        };
      };

      singleTarget = {
        apl = "sv";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = survival.talents.serpent;
        glyphs = survival.glyphs.default;
      };

      multiTarget = {
        apl = "sv";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = survival.talents.serpent;
        glyphs = survival.glyphs.default;
      };

      cleave = {
        apl = "sv";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = survival.talents.serpent;
        glyphs = survival.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = survival.talents.serpent;
        glyphs = survival.glyphs.default;
      };
    };
  };
in
  survival
