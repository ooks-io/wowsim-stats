{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  beast_mastery = {
    defaultRace = "worgen";

    talents = {
      glaiveToss = "312211";
    };

    glyphs = {
      default = {
        major1 = 42909; # animal bond
        major2 = 42903; # deterrence
        major3 = 42911; # pathfinding
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
      spec = "beast_mastery";
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
        apl = "bm";
        p1.gearset = "p1_bm";
        p2.gearset = "p2_bm";
        preRaid.gearset = "preraid_bm";
        talents = beast_mastery.talents.glaiveToss;
        glyphs = beast_mastery.glyphs.default;
      };

      multiTarget = {
        apl = "bm";
        p1.gearset = "p1_bm";
        p2.gearset = "p2_bm";
        preRaid.gearset = "preraid_bm";
        talents = beast_mastery.talents.glaiveToss;
        glyphs = beast_mastery.glyphs.default;
      };

      cleave = {
        apl = "bm";
        p1.gearset = "p1_bm";
        p2.gearset = "p2_bm";
        preRaid.gearset = "preraid_bm";
        talents = beast_mastery.talents.glaiveToss;
        glyphs = beast_mastery.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_bm";
        talents = beast_mastery.talents.glaiveToss;
        glyphs = beast_mastery.glyphs.default;
      };
    };
  };
in
  beast_mastery
