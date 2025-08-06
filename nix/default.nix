{
  imports = [
    ./shell.nix
    ./simulation
    ./apps
    #./checks.nix
    ./pkgs

    # components
    ./classes
    ./consumables
    ./buffs
    ./debuffs
    ./target
    ./encounter
    ./trinkets
    ./api
  ];
}
