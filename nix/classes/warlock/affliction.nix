{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkAffliction = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "warlock";
      spec = "affliction";
      options = {
        classOptions = {
          summon = "Felhunter";
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42472;
        minor3 = 43389;
      };
    };

  affliction = {
    # Talent configurations
    talents = {
      archimondesDarkness = "231211";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
          multiTarget = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
          cleave = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
        };
        dungeon = {
          singleTarget = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
          multiTarget = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
          cleave = mkAffliction {
            race = "orc";
            talents = affliction.talents.archimondesDarkness;
          };
        };
      };
    };
  };
in
  affliction

