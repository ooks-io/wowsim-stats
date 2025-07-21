{
  lib,
  target,
  ...
}: let
  inherit (lib.sim.encounter) mkEncounter;
  inherit (lib) genList;

  # Target count configurations
  targetConfigs = {
    singleTarget = [target.defaultRaidBoss];
    cleave = [target.defaultRaidBoss target.defaultRaidBoss];
    threeTarget = genList (_: target.defaultRaidBoss) 3;
    fiveTarget = genList (_: target.defaultRaidBoss) 5;
    tenTarget = genList (_: target.defaultRaidBoss) 10;
    twentyTarget = genList (_: target.defaultRaidBoss) 20;
  };

  # Duration configurations
  durations = {
    long = {duration = 300; durationVariation = 60;};
    short = {duration = 120; durationVariation = 30;};
    burst = {duration = 30; durationVariation = 10;};
  };

  # Generate encounters for a duration type
  mkDurationEncounters = durationConfig:
    lib.mapAttrs (name: targets: 
      mkEncounter (durationConfig // {inherit targets;})
    ) targetConfigs;

  encounter = {
    raid = lib.mapAttrs (_: mkDurationEncounters) durations;
  };
in {
  flake.encounter = encounter;
  _module.args.encounter = encounter;
}
