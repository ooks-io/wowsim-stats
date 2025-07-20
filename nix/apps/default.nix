{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  ...
}: {
  perSystem = {pkgs, ...}: let
    simulation = import ./simulation {inherit lib classes encounter buffs debuffs inputs pkgs;};
  in {
    apps = simulation;
  };
}
