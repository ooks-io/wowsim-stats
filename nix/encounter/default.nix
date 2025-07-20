{
  lib,
  target,
  ...
}: let
  inherit (lib.sim.encounter) mkEncounter;
  inherit (lib) genList;
  encounter = {
    raid = {
      long = {
        singleTarget = mkEncounter {
          targets = [
            target.defaultRaidBoss
          ];
        };
        cleave = mkEncounter {
          targets = [
            target.defaultRaidBoss
            target.defaultRaidBoss
          ];
        };
        threeTarget = mkEncounter {
          targets = genList (_: target.defaultRaidBoss) 3;
        };
        fiveTarget = mkEncounter {
          targets = genList (_: target.defaultRaidBoss) 5;
        };
        tenTarget = mkEncounter {
          targets = genList (_: target.defaultRaidBoss) 10;
        };
        twentyTarget = mkEncounter {
          targets = genList (_: target.defaultRaidBoss) 20;
        };
      };
    };
  };
in {
  flake.encounter = encounter;
  _module.args.encounter = encounter;
}
