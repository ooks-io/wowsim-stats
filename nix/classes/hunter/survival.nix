{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkSurvival = {
    race,
    apl ? "sv",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "hunter";
      spec = "survival";
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
        major3 = 42899; # liberation
      };
    };

  survival = {
    # talent configurations
    talents = {
      serpent = "321232";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
          multiTarget = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
          cleave = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
        };
        dungeon = {
          singleTarget = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
          multiTarget = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
          cleave = mkSurvival {
            race = "worgen";
            talents = survival.talents.serpent;
          };
        };
      };
    };
  };
in
  survival
