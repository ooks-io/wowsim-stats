{lib, ...}: let
  # Full raid buff configuration - all buffs enabled
  fullBuffs = {
    trueshotAura = true;
    unholyAura = true;
    darkIntent = true;
    moonkinAura = true;
    leaderOfThePack = true;
    blessingOfMight = true;
    legacyOfTheEmperor = true;
    bloodlust = true;
    stormlashTotemCount = 5;
    skullBannerCount = 3;
  };

  # Full buffs without bloodlust/heroism
  fullBuffsNoLust = fullBuffs // {
    bloodlust = false;
  };

  # No raid buffs
  noBuffs = {
    trueshotAura = false;
    unholyAura = false;
    darkIntent = false;
    moonkinAura = false;
    leaderOfThePack = false;
    blessingOfMight = false;
    legacyOfTheEmperor = false;
    bloodlust = false;
    stormlashTotemCount = 0;
    skullBannerCount = 0;
  };

  # Function to merge custom buffs with defaults
  mergeRaidBuffs = customBuffs: fullBuffs // customBuffs;

in {
  inherit fullBuffs fullBuffsNoLust noBuffs mergeRaidBuffs;
}