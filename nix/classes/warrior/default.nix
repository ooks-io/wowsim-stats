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
    "worgen"
    "orc"
    "undead"
    "tauren"
    "troll"
    "blood_elf"
    "goblin"
    "alliance_pandaren"
  ];

  arms = import ./arms.nix {inherit lib consumables;};
  fury = import ./fury.nix {inherit lib consumables;};
  # protection = import ./protection.nix {inherit lib consumables;};
}

