{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkGuardian = {
    race,
    apl ? "guardian",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "leatherworking",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "druid";
      spec = "guardian";
      options = {
        startingRage = 0;
        innervateTarget = "self";
        okfUptime = 1;
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 94390; # frenzied regeneration
        major2 = 40897; # maul
        major3 = 54818; # survival instincts
      };
    };

  guardian = {
    # Talent configurations
    talents = {
      incarnation = "312122";
      heartOfTheWild = "312112";
    };

    p1 = {
      singleTarget = mkGuardian {
        race = "nightelf";
        talents = guardian.talents.incarnation;
      };
      aoe = mkGuardian {
        race = "nightelf";
        talents = guardian.talents.heartOfTheWild;
      };
    };
  };
in
  guardian