{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  fire = {
    defaultRace = "troll";

    talents = {
      livingBomb = "111122";
      netherTempest = "111112";
    };

    glyphs = {
      default = {
        major1 = 42739; # combustion
        major2 = 63539; # inferno blast
        major3 = 42748; # rapid displacement
        minor1 = 42743; # momentum
        minor2 = 42735; # loose mana
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
        "troll"
        "blood_elf"
        "goblin"
        "alliance_pandaren"
      ];
      class = "mage";
      spec = "fire";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "jewelcrafting";
      distanceFromTarget = 20;
      options = {};

      singleTarget = {
        apl = "fire";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = fire.talents.livingBomb;
        glyphs = fire.glyphs.default;
      };

      multiTarget = {
        apl = "fire_cleave";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = fire.talents.netherTempest;
        glyphs = fire.glyphs.default;
      };

      cleave = {
        apl = "fire_cleave";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = fire.talents.netherTempest;
        glyphs = fire.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_bis";
        talents = fire.talents.netherTempest;
        glyphs = fire.glyphs.default;
      };
    };
  };
in
  fire
