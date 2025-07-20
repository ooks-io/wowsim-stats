{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkUnholy = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "blacksmithing",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "death_knight";
      spec = "unholy";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 43533;
        major2 = 43548;
        major3 = 104047;
        minor1 = 43550;
        minor2 = 45806;
        minor3 = 43539;
      };
    };

  unholy = {
    # Talent configurations
    talents = {
      gorfiendsGrasp = "221111";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
          multiTarget = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
          cleave = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
        };
        dungeon = {
          singleTarget = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
          multiTarget = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
          cleave = mkUnholy {
            race = "orc";
            talents = unholy.talents.gorfiendsGrasp;
          };
        };
      };
    };
  };
in
  unholy
