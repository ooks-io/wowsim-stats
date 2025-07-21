{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "night_elf"
    "worgen"
    "tauren"
    "troll"
  ];
  # Druid specs
  balance = import ./balance.nix {inherit lib consumables;};
  #feral = import ./feral.nix {inherit lib consumables;};
  # guardian = import ./guardian.nix {inherit lib consumables;};
}
