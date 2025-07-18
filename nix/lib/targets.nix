{lib, ...}: let
  mkTarget = {
    id ? 31146,
    name ? "Raid Target",
    level ? 93,
    type ? "mechanical",
    minBaseDamage ? 550000,
    damageSpread ? 0.4,
    swingSpeed ? 2,
  }: {
    inherit id name level damageSpread swingSpeed minBaseDamage;
    mobType = "MobType${lib.string.toSentenceCase type}";
  };
  mobs = {
    defaultRaidBoss = mkTarget {};
    smallTrash = mkTarget {
      level = 90;
      minBaseDamage = 15000;
    };
    largeTrash = mkTarget {
      level = 90;
      minBaseDamage = 40000;
    };
    # todo add raid bosses
  };
in
  targets
