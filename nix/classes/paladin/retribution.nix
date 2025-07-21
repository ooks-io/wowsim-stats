{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) strength;

  retribution = {
    defaultRace = "human";

    talents = {
      executionSentence = "221223";
    };

    glyphs = {
      default = {
        major1 = 41097; # templar's verdict
        major2 = 41092; # double jeopardy
        major3 = 83107; # mass exorcism
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "draenei"
        "tauren"
        "blood_elf"
      ];
      class = "paladin";
      spec = "retribution";
      consumables = strength;
      profession1 = "engineering";
      profession2 = "blacksmithing";
      distanceFromTarget = 5;
      options = {classOptions = {};};

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = retribution.talents.executionSentence;
        glyphs = retribution.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = retribution.talents.executionSentence;
        glyphs = retribution.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        preRaid.gearset = "preraid";
        talents = retribution.talents.executionSentence;
        glyphs = retribution.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = retribution.talents.executionSentence;
        glyphs = retribution.glyphs.default;
      };
    };
  };
in
  retribution
