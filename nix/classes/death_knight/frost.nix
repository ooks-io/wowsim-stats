{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkFrost = {
    race,
    apl ? "obliterate",
    gearset ? "p1.2h-obliterate",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "alchemy",
    distanceFromTarget ? 5,
    glyphs,
  }:
    mkPlayer {
      class = "death_knight";
      spec = "frost";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget glyphs;
    };

  frost = {
    # Talent configurations
    talents = {
      obliterate = "221111";
    };

    glyphs = {
      obliterate = {
        major1 = 43533; # anti-magic shell
        major2 = 104048; # regenerative magic
        major3 = 104047; # load horn
        minor1 = 43550; # army of the dead
        minor2 = 45806; # tranquil grip
        minor3 = 43673; # death gate
      };
      masterfrost = {
        major1 = 43533; # anti-magic shell
        major2 = 58657; # pestilence
        major3 = 104047; # loud horn
        minor1 = 43550; # army of the dead
        minor2 = 45806; # tranquil grip
        minor3 = 43673; # death gate
      };
    };

    p1 = {
      singleTarget = mkFrost {
        race = "troll";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
        apl = "obliterate";
      };
      aoe = mkFrost {
        race = "troll";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
        apl = "obliterate";
      };
    };
  };
in
  frost
