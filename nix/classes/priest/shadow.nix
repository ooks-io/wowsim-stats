{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkShadow = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 28,
  }:
    mkPlayer {
      class = "priest";
      spec = "shadow";
      options = {
        classOptions = {armor = "InnerFire";};
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {};
      # TODO implement additional options
      # channelClipdelayMs = 40;
    };

  shadow = {
    # Talent configurations
    talents = {
      halo = "223113";
    };

    p1 = {
      singleTarget = mkShadow {
        race = "troll";
        talents = shadow.talents.halo;
      };
      aoe = mkShadow {
        race = "troll";
        talents = shadow.talents.halo;
      };
    };
  };
in
  shadow

