{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkFeral = {
    race,
    apl ? "feral",
    gearset ? "p1_feral_t14",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "druid";
      spec = "feral";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 54812; # Rip
      };
    };

  feral = {
    # Talent configurations
    talents = {
      primal = "321232";
    };

    p1 = {
      singleTarget = mkFeral {
        race = "troll";
        talents = feral.talents.primal;
      };
      aoe = mkFeral {
        race = "troll";
        talents = feral.talents.primal;
      };
    };
  };
in
  feral

