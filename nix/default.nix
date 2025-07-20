{
  imports = [
    ./shell.nix
    ./apps.nix
    ./checks.nix
    ./test.nix

    # components
    ./classes
    ./consumables
    ./buffs
    ./debuffs
    ./target
    ./encounter
  ];
}
