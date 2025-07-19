{
  lib,
  ...
}: {
  # Import class-specific component modules
  monk = import ./monk {inherit lib;};
  
  # Future class imports
  # warrior = import ./warrior {inherit lib;};
  # paladin = import ./paladin {inherit lib;};
  # hunter = import ./hunter {inherit lib;};
  # rogue = import ./rogue {inherit lib;};
  # priest = import ./priest {inherit lib;};
  # shaman = import ./shaman {inherit lib;};
  # mage = import ./mage {inherit lib;};
  # warlock = import ./warlock {inherit lib;};
  # druid = import ./druid {inherit lib;};
  # deathknight = import ./deathknight {inherit lib;};
}