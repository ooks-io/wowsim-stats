{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  balance = {
    defaultRace = "troll";

    talents = {
      dreamOfCenarius = "113222";
    };

    glyphs = {
      default = {
        major1 = 40914;
        major2 = 40906;
        major3 = 40909;
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "night_elf"
        "worgen"
        "tauren"
        "troll"
      ];

      class = "druid";
      spec = "balance";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 20;
      options = {
        classOptions = {
          innervateTarget = {};
        };
      };

      singleTarget = {
        apl = "standard";
        p1.gearset = "t14";
        preRaid.gearset = "preraid";
        talents = balance.talents.dreamOfCenarius;
        glyphs = balance.glyphs.default;
      };

      multiTarget = {
        apl = "standard";
        p1.gearset = "t14";
        preRaid.gearset = "preraid";
        talents = balance.talents.dreamOfCenarius;
        glyphs = balance.glyphs.default;
      };

      cleave = {
        apl = "standard";
        p1.gearset = "t14";
        preRaid.gearset = "preraid";
        talents = balance.talents.dreamOfCenarius;
        glyphs = balance.glyphs.default;
      };

      challengeMode = {
        gearset = "t14";
        talents = balance.talents.dreamOfCenarius;
        glyphs = balance.glyphs.default;
      };
    };
  };
in
  balance
