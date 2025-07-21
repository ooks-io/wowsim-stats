{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  elemental = {
    defaultRace = "troll";

    talents = {
      unleashedFury = "333121";
      primalElementalist = "333322";
    };

    glyphs = {
      default = {
        major1 = 41539; # flame shock
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "dwarf"
        "draenei"
        "orc"
        "tauren"
        "troll"
        "goblin"
        "alliance_pandaren"
      ];
      class = "shaman";
      spec = "elemental";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 20;
      options = {
        classOptions = {
          shield = "LightningShield";
          feleAutocast = {};
        };
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = elemental.talents.unleashedFury;
        glyphs = elemental.glyphs.default;
      };

      multiTarget = {
        apl = "aoe";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = elemental.talents.primalElementalist;
        glyphs = elemental.glyphs.default;
      };

      cleave = {
        apl = "cleave";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = elemental.talents.primalElementalist;
        glyphs = elemental.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = elemental.talents.unleashedFury;
        glyphs = elemental.glyphs.default;
      };
    };
  };
in
  elemental
