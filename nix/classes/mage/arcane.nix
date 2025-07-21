{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  arcane = {
    defaultRace = "troll";

    talents = {
      livingBomb = "311122";
      netherTempest = "311112";
    };

    glyphs = {
      default = {
        major1 = 44955; # arcane power
        major2 = 42748; # rapid displacement
        major3 = 42746; # cone of cold
        minor1 = 42743; # momentum
        minor2 = 63416; # rapid teleportation
        minor3 = 42735; # loose mana
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
      spec = "arcane";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 20;
      options = {};

      singleTarget = {
        apl = "default";
        p1.gearset = "p1_bis";
        preRaid.gearset = "rich_prebis";
        talents = arcane.talents.livingBomb;
        glyphs = arcane.glyphs.default;
      };

      multiTarget = {
        apl = "arcane_cleave";
        p1.gearset = "p1_bis";
        preRaid.gearset = "rich_prebis";
        talents = arcane.talents.netherTempest;
        glyphs = arcane.glyphs.default;
      };

      cleave = {
        apl = "arcane_cleave";
        p1.gearset = "p1_bis";
        preRaid.gearset = "rich_prebis";
        talents = arcane.talents.netherTempest;
        glyphs = arcane.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_bis";
        talents = arcane.talents.netherTempest;
        glyphs = arcane.glyphs.default;
      };
    };
  };
in
  arcane
