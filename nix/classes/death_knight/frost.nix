{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.classes) mkClassTemplate;
  inherit (consumables.preset) strength;

  frost = {
    talents = {
      obliterate = "221111";
    };
    defaultRace = "troll";

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
        major2 = 43548; # pestilence
        major3 = 104047; # loud horn
        minor1 = 43550; # army of the dead
        minor2 = 45806; # tranquil grip
        minor3 = 43673; # death gate
      };
    };

    template = mkClassTemplate {
      playableRaces = [
        "human"
        "dwarf"
        "night_elf"
        "gnome"
        "draenei"
        "worgen"
        "orc"
        "undead"
        "tauren"
        "troll"
        "blood_elf"
        "goblin"
      ];
      class = "death_knight";
      spec = "frost";
      consumables = strength;
      profession1 = "engineering";
      profession2 = "alchemy";
      distanceFromTarget = 5;
      options = {};

      singleTarget = {
        apl = "obliterate";
        p1.gearset = "p1.2h-obliterate";
        preRaid.gearset = "prebis";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
      };

      multiTarget = {
        apl = "obliterate";
        p1.gearset = "p1.2h-obliterate";
        preRaid.gearset = "prebis";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
      };

      cleave = {
        apl = "obliterate";
        p1.gearset = "p1.2h-obliterate";
        preRaid.gearset = "prebis";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
      };

      challengeMode = {
        gearset = "p1.2h-obliterate";
        talents = frost.talents.obliterate;
        glyphs = frost.glyphs.obliterate;
      };
    };
  };
in
  frost
