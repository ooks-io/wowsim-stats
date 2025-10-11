{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) agility;

  enhancement = {
    defaultRace = "orc";

    talents = {
      elementalBlast = "313133";
    };

    glyphs = {
      default = {
        major1 = 71155; # lightning shield
        major2 = 41529; # fire elemental totem
        major3 = 41530; # fire nova
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
      spec = "enhancement";
      consumables = agility;
      profession1 = "engineering";
      profession2 = "tailoring";
      distanceFromTarget = 5;
      options = {
        classOptions = {
          shield = "LightningShield";
          feleAutocast = {};
          imbueMh = "WindfuryWeapon";
        };
        syncType = "Auto";
        imbueOh = "FlametongueWeapon";
      };

      singleTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = enhancement.talents.elementalBlast;
        glyphs = enhancement.glyphs.default;
      };

      multiTarget = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = enhancement.talents.elementalBlast;
        glyphs = enhancement.glyphs.default;
      };

      cleave = {
        apl = "default";
        p1.gearset = "p1";
        p2.gearset = "p2";
        preRaid.gearset = "preraid";
        talents = enhancement.talents.elementalBlast;
        glyphs = enhancement.glyphs.default;
      };

      challengeMode = {
        gearset = "p1";
        talents = enhancement.talents.elementalBlast;
        glyphs = enhancement.glyphs.default;
      };
    };
  };
in
  enhancement
