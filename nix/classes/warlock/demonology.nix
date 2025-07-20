{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) intellect;

  mkDemonology = {
    race,
    apl ? "default",
    gearset ? "p1",
    talents,
    consumables ? intellect,
    profession1 ? "engineering",
    profession2 ? "tailoring",
    distanceFromTarget ? 25,
  }:
    mkPlayer {
      class = "warlock";
      spec = "demonology";
      options = {
        classOptions = {
          summon = "Felguard";
        };
      };
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 42470; # soulstone
        major2 = 45785; # life tap
        major3 = 42465; # imp swarm
        minor3 = 43389; # unending breath
      };
    };

  demonology = {
    # Talent configurations
    talents = {
      archimondesDarkness = "231211";
    };

    p1 = {
      singleTarget = mkDemonology {
        race = "orc";
        talents = demonology.talents.archimondesDarkness;
      };
      aoe = mkDemonology {
        race = "orc";
        talents = demonology.talents.archimondesDarkness;
      };
    };
  };
in
  demonology

