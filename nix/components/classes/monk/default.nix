{
  lib,
  ...
}: {
  # Monk specs
  windwalker = import ./windwalker.nix {inherit lib;};
  
  # Future monk specs
  # brewmaster = import ./brewmaster.nix {inherit lib;};
  # mistweaver = import ./mistweaver.nix {inherit lib;};
}