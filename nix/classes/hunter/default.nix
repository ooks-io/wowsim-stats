{
  lib,
  consumables,
  ...
}: {
  # Hunter specs
  beast_mastery = import ./beast_mastery.nix {inherit lib consumables;};
  marksmanship = import ./marksmanship.nix {inherit lib consumables;};
  survival = import ./survival.nix {inherit lib consumables;};
}

