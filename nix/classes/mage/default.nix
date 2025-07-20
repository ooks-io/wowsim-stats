{
  lib,
  consumables,
  ...
}: {
  # Mage specs
  arcane = import ./arcane.nix {inherit lib consumables;};
  fire = import ./fire.nix {inherit lib consumables;};
  frost = import ./frost.nix {inherit lib consumables;};
}