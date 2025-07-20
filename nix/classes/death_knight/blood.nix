{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkBlood = {
    race,
    apl ? "blood",
    gearset ? "p1",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "blacksmithing",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "deathknight";
      spec = "blood";
      options = {
        startingRunicPower = 0;
        petUptime = 1;
        precastGhoulFrenzy = true;
        precastHornOfWinter = true;
        drwPestiApply = true;
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 58677; # vampiric blood
        major2 = 58676; # bone shield
        major3 = 58647; # death strike
      };
    };

  blood = {
    # Talent configurations
    talents = {
      necroticStrike = "312112";
      soulReaper = "312122";
    };

    p1 = {
      singleTarget = mkBlood {
        race = "orc";
        talents = blood.talents.necroticStrike;
      };
      aoe = mkBlood {
        race = "orc";
        talents = blood.talents.soulReaper;
      };
    };
  };
in
  blood