{
  lib,
  consumables,
  ...
}: {
  # Death Knight specs
  frost = import ./frost.nix {inherit lib consumables;};
  unholy = import ./unholy.nix {inherit lib consumables;};
  # blood = import ./blood.nix {inherit lib consumables;};
}