{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) intellect;

  frost = {
    defaultRace = "troll";

    talents = {
      livingBomb = "311122";
      netherTempest = "311112";
    };

    glyphs = {
      default = {
        major1 = 42745; # splitting ice
        major2 = 42753; # icy veins
        major3 = 45736; # water elemental
        minor1 = 42743; # momentum
        minor2 = 45739; # mirror image
        minor3 = 104104; # the unbound elemental
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
      spec = "frost";
      consumables = intellect;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 25;
      options = {
        classOptions = {
          defaultMageArmor = "MageArmorFrostArmor";
        };
      };

      singleTarget = {
        apl = "frost";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = frost.talents.livingBomb;
        glyphs = frost.glyphs.default;
      };

      multiTarget = {
        apl = "frost_aoe";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = frost.talents.netherTempest;
        glyphs = frost.glyphs.default;
      };

      cleave = {
        apl = "frost_cleave";
        p1.gearset = "p1_bis";
        preRaid.gearset = "p1_prebis";
        talents = frost.talents.netherTempest;
        glyphs = frost.glyphs.default;
      };

      challengeMode = {
        gearset = "p1_bis";
        talents = frost.talents.netherTempest;
        glyphs = frost.glyphs.default;
      };
    };
  };
in
  frost
