{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkSubtlety = {
    race,
    apl ? "subtlety",
    gearset ? "p1_subtlety_t14",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "rogue";
      spec = "subtlety";
      options = {
        classOptions = {
          lethalPoison = "DeadlyPoison";
          startingOverkillDuration = 20;
          vanishBreakTime = 0.1;
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42970; # Hemorraghing Veins
      };
    };

  subtlety = {
    # Talent configurations
    talents = {
      mfd = "321232";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
          multiTarget = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
          cleave = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
        };
        dungeon = {
          singleTarget = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
          multiTarget = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
          cleave = mkSubtlety {
            race = "human";
            talents = subtlety.talents.mfd;
          };
        };
      };
    };
  };
in
  subtlety
