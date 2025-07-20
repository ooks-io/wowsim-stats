{
  lib,
  consumables,
  ...
}: {
  # Rogue specs
  assassination = import ./assassination.nix {inherit lib consumables;};
  combat = import ./combat.nix {inherit lib consumables;};
  subtlety = import ./subtlety.nix {inherit lib consumables;};
}