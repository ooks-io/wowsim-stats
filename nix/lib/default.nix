{
  inputs,
  lib,
  ...
}: {
  # class = import ./class.nix;
  target = import ./target.nix {inherit lib;};
  encounter = import ./encounter.nix {inherit lib;};
  player = import ./player.nix {inherit lib inputs;};
  raid = import ./raid.nix {};
}
