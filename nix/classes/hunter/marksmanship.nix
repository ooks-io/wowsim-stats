{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkMarksmanship = {
    race,
    apl ? "mm",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "leatherworking",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "hunter";
      spec = "marksmanship";
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
        major3 = 42914; # aimed shot
      };
    };

  marksmanship = {
    # Talent configurations
    talents = {
      glaiveToss = "312111";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
          multiTarget = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
          cleave = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
        };
        dungeon = {
          singleTarget = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
          multiTarget = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
          cleave = mkMarksmanship {
            race = "worgen";
            talents = marksmanship.talents.glaiveToss;
          };
        };
      };
    };
  };
in
  marksmanship
