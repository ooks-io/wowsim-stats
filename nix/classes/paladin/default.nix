{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "human"
    "dwarf"
    "draenei"
    "tauren"
    "blood_elf"
  ];
  # Paladin specs
  retribution = import ./retribution.nix {inherit lib consumables;};
  # protection = import ./protection.nix {inherit lib consumables;};
}

