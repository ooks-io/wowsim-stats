{
  inputs,
  lib,
  ...
}: {
  # class = import ./class.nix;
  target = import ./target.nix {inherit lib;};
  encounter = import ./encounter.nix;
  player = import ./player.nix {inherit lib inputs;};
  raid = import ./raid.nix {inherit lib;};
  simulation = import ./simulation.nix {inherit lib;};
  party = import ./party.nix {inherit lib;};
}
