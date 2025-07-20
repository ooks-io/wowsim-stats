{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkDestruction = {
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
      spec = "destruction";
      options = {
        classOptions = {
          summon = "Imp";
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {};
    };

  destruction = {
    # Talent configurations
    talents = {
      archimondesDarkness = "221211";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
          multiTarget = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
          cleave = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
        };
        dungeon = {
          singleTarget = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
          multiTarget = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
          cleave = mkDestruction {
            race = "orc";
            talents = destruction.talents.archimondesDarkness;
          };
        };
      };
    };
  };
in
  destruction

