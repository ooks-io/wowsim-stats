{
  description = "Description for the project";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default-linux";
    wowsims = {
      url = "github:ooks-io/mop/nix";
      inputs = {
        nixpkgs.follows = "nixpkgs";
        flake-parts.follows = "flake-parts";
        systems.follows = "systems";
      };
    };
  };

  outputs = inputs @ {
    flake-parts,
    self,
    ...
  }: let
    # extend nixpkgs library with our sim library
    lib = import ./nix/lib/extend.nix {inherit inputs self;};
  in
    flake-parts.lib.mkFlake {
      inherit inputs;
      specialArgs = {inherit lib;};
    } {
      systems = import inputs.systems;
      imports = [./nix];
    };
}
