{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkWindwalker = {
    race,
    apl ? "default",
    gearset ? "p1_bis_dw",
    talents,
    glyphs ? windwalker.glyphs.default,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "monk";
      spec = "windwalker";
      options = {};
      inherit race gearset talents apl consumables glyphs profession1 profession2 distanceFromTarget;
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

    template = {
      p1 = {
        raid = {
          singleTarget = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.xuen;
          };
          multiTarget = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.rjw;
          };
          cleave = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.rjw;
          };
        };
        dungeon = {
          singleTarget = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.xuen;
          };
          multiTarget = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.rjw;
          };
          cleave = mkWindwalker {
            race = "orc";
            talents = windwalker.talents.rjw;
          };
        };
      };
    };
  };
in
  windwalker
