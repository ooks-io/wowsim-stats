{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkArms = {
    race,
    apl ? "arms",
    gearset ? "p1_arms_bis",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "blacksmithing",
    distanceFromTarget ? 9,
  }:
    mkPlayer {
      class = "warrior";
      spec = "arms";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 67482; # bull rush
        major2 = 43399; # unending rage
        major3 = 45792; # death from above
      };
    };

  arms = {
    # Talent configurations
    talents = {
      bloodbath = "113332";
    };

    p1 = {
      singleTarget = mkArms {
        race = "human";
        talents = arms.talents.bloodbath;
      };
      aoe = mkArms {
        race = "human";
        talents = arms.talents.bloodbath;
      };
    };
  };
in
  arms

