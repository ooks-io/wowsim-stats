{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  destruction = {
    defaultRace = "orc";

    talents = {
      archimondesDarkness = "221211";
    };

    glyphs = {
      default = {};
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "gnome"
        "worgen"
        "orc"
        "undead"
        "troll"
        "blood_elf"
        "goblin"
      ];
      class = "warlock";
      spec = "destruction";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          summon = "Imp";
        };
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "p1-prebis";
        talents = destruction.talents.archimondesDarkness;
        glyphs = destruction.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "p1-prebis";
        talents = destruction.talents.archimondesDarkness;
        glyphs = destruction.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "p1-prebis";
        talents = destruction.talents.archimondesDarkness;
        glyphs = destruction.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = destruction.talents.archimondesDarkness;
        glyphs = destruction.glyphs.default;
      };
    };
  };
in
  destruction
