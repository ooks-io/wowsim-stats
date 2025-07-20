{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkProtection = {
    race,
    apl ? "protection",
    gearset ? "p1",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "blacksmithing",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "warrior";
      spec = "protection";
      options = {
        startingRage = 0;
        shout = "BattleShout";
        stance = "DefensiveStance";
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 58375; # shield slam
        major2 = 58388; # revenge
        major3 = 58387; # devastate
      };
    };

  protection = {
    # Talent configurations
    talents = {
      shockwave = "132112";
      dragonRoar = "132122";
    };

    p1 = {
      singleTarget = mkProtection {
        race = "human";
        talents = protection.talents.shockwave;
      };
      aoe = mkProtection {
        race = "human";
        talents = protection.talents.dragonRoar;
      };
    };
  };
in
  protection

