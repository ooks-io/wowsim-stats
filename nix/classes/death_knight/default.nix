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
  ];
  frost = import ./frost.nix {inherit lib consumables;};
  unholy = import ./unholy.nix {inherit lib consumables;};
  # blood = import ./blood.nix {inherit lib consumables;};
}
