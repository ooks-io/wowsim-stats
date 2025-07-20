{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkAssassination = {
    race,
    apl ? "assassination",
    gearset ? "p1_assassination_t14",
    talents,
    glyphs ? {},
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "rogue";
      spec = "assassination";
      options = {
        classOptions = {
          lethalPoison = "DeadlyPoison";
          startingOverkillDuration = 20;
          vanishBreakTime = 0.1;
        };
      };
      inherit race gearset talents apl consumables glyphs profession1 profession2 distanceFromTarget;
    };

  assassination = {
    # Talent configurations
    talents = {
      mfd = "321232";
    };

    p1 = {
      singleTarget = mkAssassination {
        race = "human";
        talents = assassination.talents.mfd;
      };
      aoe = mkAssassination {
        race = "human";
        talents = assassination.talents.mfd;
      };
    };
  };
in
  assassination
