{lib, ...}: let
  inherit (lib.sim.target) mkTarget;
  target = {
    defaultRaidBoss = mkTarget {};
    smallTrash = mkTarget {
      level = 90;
      minBaseDamage = 15000;
    };
    largeTrash = mkTarget {
      level = 90;
      minBaseDamage = 40000;
    };
  };
in {
  flake.target = target;
  _module.args.target = target;
}
