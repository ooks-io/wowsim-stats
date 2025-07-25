let
  buffs = {
    full = {
      trueshotAura = true;
      unholyAura = true;
      darkIntent = true;
      moonkinAura = true;
      leaderOfThePack = true;
      blessingOfMight = true;
      legacyOfTheEmperor = true;
      bloodlust = true;
      stormlashTotemCount = 4;
      skullBannerCount = 2;
    };
    caster = {
      serpentsSwiftness = true;
      arcaneBrilliance = true;
      moonkinAura = true;
      leaderOfThePack = true;
      blessingOfMight = true;
      blessingOfKings = true;
      bloodlust = true;
      manaTideTotemCount = 1;
      stormlashTotemCount = 4;
      skullBannerCount = 2;
    };
    fullNoLust = buffs.full // {bloodlust = false;};
    none = {
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
    custom = opts: buffs.full // opts;
  };
in {
  flake.buffs = buffs;
  _module.args.buffs = buffs;
}
