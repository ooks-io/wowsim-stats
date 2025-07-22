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
    "worgen"
    "orc"
    "undead"
    "troll"
    "blood_elf"
    "goblin"
    "alliance_pandaren"
  ];
  assassination = import ./assassination.nix {inherit lib consumables;};
  combat = import ./combat.nix {inherit lib consumables;};
  subtlety = import ./subtlety.nix {inherit lib consumables;};
}
