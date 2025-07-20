{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkBalance = {
    race,
    apl ? "standard",
    gearset ? "t14",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 20,
  }:
    mkPlayer {
      class = "druid";
      spec = "balance";
      options = {
        classOptions = {
          innervateTarget = {};
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 40914;
        major2 = 40906;
        major3 = 40909;
      };
    };

  balance = {
    # Talent configurations
    talents = {
      dreamOfCenarius = "113222";
    };

    p1 = {
      singleTarget = mkBalance {
        race = "troll";
        talents = balance.talents.dreamOfCenarius;
      };
      aoe = mkBalance {
        race = "troll";
        talents = balance.talents.dreamOfCenarius;
      };
    };
  };
in
  balance
