{lib, ...}: let
  inherit (lib.sim.target) mkTarget;
  target = {
    # default raid boss from wowsims
    # id 31146
    # level 93
    # mechanical
    # 550000 base damage
    # 0.4 damage spread
    # 2 swing timer
    defaultRaidBoss = mkTarget {};

    # modeled after the party monkeys in stormstout
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
