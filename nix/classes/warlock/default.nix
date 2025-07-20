{
  lib,
  consumables,
  ...
}: {
  # Warlock specs
  affliction = import ./affliction.nix {inherit lib consumables;};
  demonology = import ./demonology.nix {inherit lib consumables;};
  destruction = import ./destruction.nix {inherit lib consumables;};
}