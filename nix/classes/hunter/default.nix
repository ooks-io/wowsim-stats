{
  lib,
  consumables,
  ...
}: {
  playableRaces = [
    "human"
    "dwarf"
    "night_elf"
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
  # Hunter specs
  beast_mastery = import ./beast_mastery.nix {inherit lib consumables;};
  marksmanship = import ./marksmanship.nix {inherit lib consumables;};
  survival = import ./survival.nix {inherit lib consumables;};
}
