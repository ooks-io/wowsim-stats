{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  affliction = {
    defaultRace = "troll";

    talents = {
      archimondesDarkness = "231211";
    };

    glyphs = {
      default = {
        major1 = 42472;
        minor3 = 43389;
      };
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
      spec = "affliction";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          summon = "Felhunter";
        };
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = affliction.talents.archimondesDarkness;
        glyphs = affliction.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = affliction.talents.archimondesDarkness;
        glyphs = affliction.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = affliction.talents.archimondesDarkness;
        glyphs = affliction.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = affliction.talents.archimondesDarkness;
        glyphs = affliction.glyphs.default;
      };
    };
  };
in
  affliction
