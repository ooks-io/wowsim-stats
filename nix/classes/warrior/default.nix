{
  lib,
  consumables,
  ...
}: {
  # Warrior specs
  arms = import ./arms.nix {inherit lib consumables;};
  fury = import ./fury.nix {inherit lib consumables;};
  # protection = import ./protection.nix {inherit lib consumables;};
}