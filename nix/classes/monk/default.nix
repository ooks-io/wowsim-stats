{
  lib,
  consumables,
  ...
}: {
  # Monk specs
  windwalker = import ./windwalker.nix {inherit lib consumables;};
  #brewmaster = import ./brewmaster.nix {inherit lib consumables;};
}
