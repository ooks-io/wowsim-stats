{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkEnhancement = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "shaman";
      spec = "enhancement";
      options = {
        classOptions = {
          shield = "LightningShield";
          feleAutocast = {};
          imbueMh = "WindfuryWeapon";
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 71155; # lightning shield
        major2 = 41529; # fire elemental totem
        major3 = 41530; # fire nova
      };
    };

  enhancement = {
    # Talent configurations
    talents = {
      elementalBlast = "313133";
    };

    p1 = {
      singleTarget = mkEnhancement {
        race = "troll";
        talents = enhancement.talents.elementalBlast;
      };
      aoe = mkEnhancement {
        race = "troll";
        talents = enhancement.talents.elementalBlast;
      };
    };
  };
in
  enhancement
