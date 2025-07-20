{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkFire = {
    race,
    apl ? "fire",
    gearset ? "p1_bis",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 20,
  }:
    mkPlayer {
      class = "mage";
      spec = "fire";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42739; # combustion
        major2 = 63539; # inferno blast
        major3 = 42748; # rapid displacement
        minor1 = 42743; # momentum
        minor2 = 42735; # loose mana
      };
    };

  fire = {
    # Talent configurations
    talents = {
      livingBomb = "111122";
      netherTempest = "111112";
    };

    template = {
      p1 = {
        raid = {
          singleTarget = mkFire {
            race = "troll";
            talents = fire.talents.livingBomb;
          };
          multiTarget = mkFire {
            race = "troll";
            talents = fire.talents.netherTempest;
            apl = "fire_cleave";
          };
          cleave = mkFire {
            race = "troll";
            talents = fire.talents.netherTempest;
            apl = "fire_cleave";
          };
        };
        dungeon = {
          singleTarget = mkFire {
            race = "troll";
            talents = fire.talents.livingBomb;
          };
          multiTarget = mkFire {
            race = "troll";
            talents = fire.talents.netherTempest;
            apl = "fire_cleave";
          };
          cleave = mkFire {
            race = "troll";
            talents = fire.talents.netherTempest;
            apl = "fire_cleave";
          };
        };
      };
    };
  };
in
  fire

