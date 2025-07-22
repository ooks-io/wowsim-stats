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
    "troll"
    "blood_elf"
    "goblin"
    "alliance_pandaren"
  ];
  arcane = import ./arcane.nix {inherit lib consumables;};
  fire = import ./fire.nix {inherit lib consumables;};
  frost = import ./frost.nix {inherit lib consumables;};
}
