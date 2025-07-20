{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkBeastmastery = {
    race,
    apl ? "bm",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "leatherworking",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "hunter";
      spec = "beast_mastery";
      options = {
        classOptions = {
          petType = "Wolf";
          petUptime = 1;
          useHunterMark = true;
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42909; # animal bond
        major2 = 42903; # deterrence
        major3 = 42911; # pathfinding
      };
    };

  beastmastery = {
    # Talent configurations
    talents = {
      glaiveToss = "312211";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
          multiTarget = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
          cleave = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
        };
        dungeon = {
          singleTarget = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
          multiTarget = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
          cleave = mkBeastmastery {
            race = "worgen";
            talents = beastmastery.talents.glaiveToss;
          };
        };
      };
    };
  };
in
  beastmastery
