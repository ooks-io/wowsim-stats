{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  demonology = {
    defaultRace = "troll";

    talents = {
      archimondesDarkness = "231211";
    };

    glyphs = {
      default = {
        major1 = 42470; # soulstone
        major2 = 45785; # life tap
        major3 = 42465; # imp swarm
        minor3 = 43389; # unending breath
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
      spec = "demonology";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          summon = "Felguard";
        };
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = demonology.talents.archimondesDarkness;
        glyphs = demonology.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = demonology.talents.archimondesDarkness;
        glyphs = demonology.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = demonology.talents.archimondesDarkness;
        glyphs = demonology.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = demonology.talents.archimondesDarkness;
        glyphs = demonology.glyphs.default;
      };
    };
  };
in
  demonology
