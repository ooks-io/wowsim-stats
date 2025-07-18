{lib, ...}: let
  # Full raid debuff configuration - all debuffs enabled
  fullDebuffs = {
    physicalVulnerability = true;
    weakenedArmor = true;
  };

  # No raid debuffs
  noDebuffs = {
    physicalVulnerability = false;
    weakenedArmor = false;
  };

  # Function to merge custom debuffs with defaults
  mergeRaidDebuffs = customDebuffs: fullDebuffs // customDebuffs;

in {
  inherit fullDebuffs noDebuffs mergeRaidDebuffs;
}