{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) agility;

  mkBrewmaster = {
    race,
    apl ? "brewmaster",
    gearset ? "p1",
    talents,
    consumables ? agility,
    profession1 ? "engineering",
    profession2 ? "jewelcrafting",
    distanceFromTarget ? 5,
  }:
    mkPlayer {
      class = "monk";
      spec = "brewmaster";
      options = {
        chiWave = true;
        expelHarm = true;
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 123392; # fortifying brew
        major2 = 123394; # guard
        major3 = 123396; # keg smash
      };
    };

  brewmaster = {
    # Talent configurations
    talents = {
      xuen = "213322"; # Single target build with Xuen
      rjw = "233321"; # AoE build with RJW
    };

    glyphs = {
      default = {
        major1 = 124997;
        major2 = 123394;
        minor1 = 125660;
      };
    };

    p1 = {
      singleTarget = mkBrewmaster {
        race = "AlliancePandaren";
        talents = brewmaster.talents.xuen;
      };
      aoe = mkBrewmaster {
        race = "AlliancePandaren";
        talents = brewmaster.talents.rjw;
      };
    };
  };
in
  brewmaster
