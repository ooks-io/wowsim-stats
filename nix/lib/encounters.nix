{lib, ...}: let
  # Base encounter template
  inherit (lib) genList;
  mobs = import ./targets.nix;
  mkEncounter = {
    duration ? 300,
    durationVariation ? 60,
    targets ? [],
  }: {
    apiVersion = 1;
    inherit duration durationVariation targets;

    # default execute config
    executeProportion20 = 0.2;
    executeProportion25 = 0.25;
    executeProportion35 = 0.35;
    executeProportion45 = 0.45;
    executeProportion90 = 0.9;
  };

  # Duration-based encounter templates
  raid = {
    long = {
      singleTarget = mkEncounter {
        targets = [
          mobs.defaultRaidBoss
        ];
      };

      twoTarget = mkEncounter {
        targets = genList (_: mobs.defaultRaidBoss) 3;
      };

      threeTarget = mkEncounter {
        targets = genList (_: mobs.defaultRaidBoss) 3;
      };

      fiveTarget = mkEncounter {
        targets = genList (_: mobs.defaultRaidBoss) 10;
      };

      tenTarget = mkEncounter {
        targets = genList (_: mobs.defaultRaidBoss) 10;
      };

      twentyTarget = mkEncounter {
        targets = genList (_: mobs.defaultRaidBoss) 20;
      };
    };
  };
in
  raid

