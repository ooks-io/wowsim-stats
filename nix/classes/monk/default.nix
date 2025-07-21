{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "human"
    "dwarf"
    "night_elf"
    "gnome"
    "draenei"
    "orc"
    "undead"
    "tauren"
    "troll"
    "blood_elf"
    "alliance_pandaren"
  ];
  # Monk specs
  windwalker = import ./windwalker.nix {inherit lib consumables;};
  #brewmaster = import ./brewmaster.nix {inherit lib consumables;};
}
