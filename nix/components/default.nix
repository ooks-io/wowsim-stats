{lib, ...}: let
  components = {
    # Consumable sets organized by role/stat priority
    consumables = import ./consumables.nix {inherit lib;};

    # Class-specific configurations
    classes = import ./classes {inherit lib;};
  };
in {
  flake.components = components;
  _module.args.components = components;
}

