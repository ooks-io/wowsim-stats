{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "human"
    "dwarf"
    "gnome"
    "worgen"
    "orc"
    "undead"
    "troll"
    "blood_elf"
    "goblin"
  ];
  affliction = import ./affliction.nix {inherit lib consumables;};
  demonology = import ./demonology.nix {inherit lib consumables;};
  destruction = import ./destruction.nix {inherit lib consumables;};
}
