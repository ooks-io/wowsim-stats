{
  lib,
  consumables,
  ...
}: {
  # Paladin specs
  retribution = import ./retribution.nix {inherit lib consumables;};
  # protection = import ./protection.nix {inherit lib consumables;};
}