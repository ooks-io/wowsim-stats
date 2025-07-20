{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkArcane = {
    race,
    apl ? "default",
    gearset ? "p1_bis",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 20,
  }:
    mkPlayer {
      class = "mage";
      spec = "arcane";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 44955; # arcane power
        major2 = 42748; # rapid displacement
        major3 = 42746; # cone of cold
        minor1 = 42743; # momentum
        minor2 = 63416; # rapid teleportation
        minor3 = 42735; # loose mana
      };
    };

  arcane = {
    # Talent configurations
    talents = {
      livingBomb = "311122";
      netherTempest = "311112";
    };

    p1 = {
      singleTarget = mkArcane {
        race = "troll";
        talents = arcane.talents.livingBomb;
      };
      aoe = mkArcane {
        race = "troll";
        apl = "arcane_cleave";
        talents = arcane.talents.netherTempest;
      };
    };
  };
in
  arcane

