let
  debuffs = {
    full = {
      physicalVulnerability = true;
      weakenedArmor = true;
      masterPoisoner = true;
      curseOfElements = true;
    };
    none = {
      physicalVulnerability = false;
      weakenedArmor = false;
      masterPoisoner = false;
      curseOfElements = false;
    };
    caster = {
      curseOfElements = true;
    };
    custom = opts: debuffs.full // opts;
  };
in {
  flake.debuffs = debuffs;
  _module.args.debuffs = debuffs;
}
