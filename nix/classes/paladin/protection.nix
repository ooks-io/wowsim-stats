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
      class = "paladin";
      spec = "protection";
      options = {
        consecration = true;
        holyWrath = true;
        sealChoice = "SealOfInsight";
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 41101; # consecration
        major2 = 54928; # hammer of the righteous
        major3 = 54927; # focused shield
      };
    };

  protection = {
    talents = {
      hammer = "311111";
      zealotry = "311121";
    };

    p1 = {
      singleTarget = mkProtection {
        race = "human";
        talents = protection.talents.hammer;
      };
      aoe = mkProtection {
        race = "human";
        talents = protection.talents.zealotry;
      };
    };
  };
in
  protection
