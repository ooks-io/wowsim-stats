{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkRetribution = {
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
      class = "paladin";
      spec = "retribution";
      options = {classOptions = {};};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 41097; # templar's verdict
        major2 = 41092; # double jeopardy
        major3 = 83107; # mass exorcism
      };
    };

  retribution = {
    # Talent configurations
    talents = {
      executionSentence = "221223";
    };

    p1 = {
      singleTarget = mkRetribution {
        race = "human";
        talents = retribution.talents.executionSentence;
      };
      aoe = mkRetribution {
        race = "human";
        talents = retribution.talents.executionSentence;
      };
    };
  };
in
  retribution
