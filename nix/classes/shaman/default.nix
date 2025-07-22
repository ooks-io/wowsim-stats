{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "dwarf"
    "draenei"
    "orc"
    "tauren"
    "troll"
    "goblin"
    "alliance_pandaren"
  ];
  elemental = import ./elemental.nix {inherit lib consumables;};
  enhancement = import ./enhancement.nix {inherit lib consumables;};
}
