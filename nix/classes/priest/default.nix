{
  lib,
  consumables,
  ...
}: {
  # Priest specs
  shadow = import ./shadow.nix {inherit lib consumables;};
}