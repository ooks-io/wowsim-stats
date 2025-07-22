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
    "undead"
    "tauren"
    "troll"
    "blood_elf"
    "goblin"
    "alliance_pandaren"
  ];
  shadow = import ./shadow.nix {inherit lib consumables;};
}
