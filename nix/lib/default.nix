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
  classes = import ./classes.nix {inherit lib;};
  itemDatabase = import ./itemDatabase.nix {inherit inputs lib;};
  trinket = import ./trinket.nix {inherit inputs lib;};
  shellUtils = import ./shell-utils.nix {inherit lib;};
}
