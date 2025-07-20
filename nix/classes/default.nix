{
  lib,
  consumables,
  ...
}: let
  classes = {
    # TODO: all tank classes, they require extra work
    # Import class-specific component modules
    monk = import ./monk {inherit lib consumables;};
    rogue = import ./rogue {inherit lib consumables;};
    mage = import ./mage {inherit lib consumables;};
    hunter = import ./hunter {inherit lib consumables;};
    warrior = import ./warrior {inherit lib consumables;};
    paladin = import ./paladin {inherit lib consumables;};
    priest = import ./priest {inherit lib consumables;};
    shaman = import ./shaman {inherit lib consumables;};
    warlock = import ./warlock {inherit lib consumables;};
    druid = import ./druid {inherit lib consumables;};
    deathknight = import ./death_knight {inherit lib consumables;};
  };
in {
  flake.classes = classes;
  _module.args.classes = classes;
}
