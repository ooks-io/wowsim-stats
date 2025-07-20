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
    mobType = "MobType${lib.strings.toSentenceCase type}";
    # default stats
    stats = [
      0
      0
      0
      0
      0
      0
      0
      0
      0
      0
      0
      0
      650
      0
      0
      0
      0
      24835
      0
      120016403
      0
      0
    ];
  };
in {inherit mkTarget;}
