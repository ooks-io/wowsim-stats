{
  lib,
  components,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (components.consumables.preset) agility;

  mkWindwalker = {
    race,
    apl,
    gearset,
    talents,
    consumables ? agility,
  }:
    mkPlayer {
      class = "monk";
      spec = "windwalker";
      glyphs = windwalker.glyphs.default;
      inherit race gearset talents apl consumables;
    };

  windwalker = {
    # Talent configurations
    talents = {
      xuen = "213322"; # Single target build with Xuen
      rjw = "233321"; # AoE build with RJW
    };

    glyphs = {
      default = {
        major1 = 85697;
        major2 = 87900;
        minor1 = 90715;
      };
    };

    # Phase 1 configurations
    p1 = {
      singleTarget = mkWindwalker {
        race = "orc";
        apl = "default";
        gearset = "dw_p1_bis";
        talents = windwalker.talents.xuen;
      };
      aoe = mkWindwalker {
        race = "orc";
        apl = "default";
        gearset = "dw_p1_bis";
        talents = windwalker.talents.rjw;
      };
    };
  };
in
  windwalker
