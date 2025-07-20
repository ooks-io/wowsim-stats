{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkCombat = {
    race,
    apl ? "combat",
    gearset ? "p1_combat_t14",
    talents,
    glyphs ? {},
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "rogue";
      spec = "combat";
      options = {
        classOptions = {
          lethalPoison = "DeadlyPoison";
          startingOverkillDuration = 20;
          vanishBreakTime = 0.1;
        };
      };
      inherit race gearset talents apl consumables glyphs profession1 profession2 distanceFromTarget;
    };

  combat = {
    # Talent configurations
    talents = {
      anticipation = "321213";
    };

    p1 = {
      singleTarget = mkCombat {
        race = "human";
        talents = combat.talents.anticipation;
      };
      aoe = mkCombat {
        race = "human";
        talents = combat.talents.anticipation;
      };
    };
  };
in
  combat
