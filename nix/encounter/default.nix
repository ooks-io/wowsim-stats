{
  lib,
  target,
  ...
}: let
  inherit (lib.sim.encounter) mkEncounter;
  inherit (lib) mapAttrs genList;

  targetConfigs = {
    singleTarget = [target.defaultRaidBoss];
    cleave = [target.defaultRaidBoss target.defaultRaidBoss];
    threeTarget = genList (_: target.defaultRaidBoss) 3;
    fiveTarget = genList (_: target.defaultRaidBoss) 5;
    tenTarget = genList (_: target.defaultRaidBoss) 10;
    twentyTarget = genList (_: target.defaultRaidBoss) 20;
  };

  durations = {
    long = {
      duration = 300;
      durationVariation = 60;
    };
    short = {
      duration = 120;
      durationVariation = 30;
    };
    burst = {
      duration = 30;
      durationVariation = 10;
    };
  };

  mkDurationEncounters = durationConfig:
    mapAttrs (
      name: targets:
        mkEncounter (durationConfig // {inherit targets;})
    )
    targetConfigs;

  raid = mapAttrs (_: mkDurationEncounters) durations;

  encounter = {
    inherit raid;
  };
in {
  flake.encounter = encounter;
  _module.args.encounter = encounter;
}
