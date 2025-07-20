{
  lib,
  consumables,
  ...
}: {
  # Shaman specs
  elemental = import ./elemental.nix {inherit lib consumables;};
  enhancement = import ./enhancement.nix {inherit lib consumables;};
}