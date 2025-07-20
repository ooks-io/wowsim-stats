{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkFrost = {
    race,
    apl ? "frost",
    gearset ? "p1_bis",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "mage";
      spec = "frost";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42745; # splitting ice
        major2 = 42753; # icy veins
        major3 = 45736; # water elemental
        minor1 = 42743; # momentum
        minor2 = 45739; # mirror image
        minor3 = 104104; # the unbound elemental
      };
    };

  frost = {
    # Talent configurations
    talents = {
      livingBomb = "311122";
      netherTempest = "311112";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkFrost {
            race = "troll";
            talents = frost.talents.livingBomb;
          };
          multiTarget = mkFrost {
            race = "troll";
            apl = "frost_aoe";
            talents = frost.talents.netherTempest;
          };
          cleave = mkFrost {
            apl = "frost_cleave";
            race = "troll";
            talents = frost.talents.netherTempest;
          };
        };
        dungeon = {
          singleTarget = mkFrost {
            race = "troll";
            talents = frost.talents.livingBomb;
          };
          multiTarget = mkFrost {
            race = "troll";
            apl = "frost_aoe";
            talents = frost.talents.netherTempest;
          };
          cleave = mkFrost {
            apl = "frost_cleave";
            race = "troll";
            talents = frost.talents.netherTempest;
          };
        };
      };
    };
  };
in
  frost

