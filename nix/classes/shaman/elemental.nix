{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkElemental = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 20,
  }:
    mkPlayer {
      class = "shaman";
      spec = "elemental";
      options = {
        classOptions = {
          shield = "LightningShield";
          feleAutocast = {};
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 41539; # flame shock
      };
    };

  elemental = {
    # Talent configurations
    talents = {
      unleashedFury = "333121";
      primalElementalist = "333322";
    };

    p1 = {
      singleTarget = mkElemental {
        race = "troll";
        talents = elemental.talents.unleashedFury;
        apl = "default";
      };
      cleave = mkElemental {
        race = "troll";
        talents = elemental.talents.primalElementalist;
        apl = "cleave";
      };
      aoe = mkElemental {
        race = "troll";
        talents = elemental.talents.primalElementalist;
        apl = "aoe";
      };
    };
  };
in
  elemental
