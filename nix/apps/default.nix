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
    inherit (pkgs) callPackage writeShellApplication;
    simulation = import ./simulation {inherit lib classes encounter buffs debuffs inputs pkgs;};
    getDB = callPackage ./getDB.nix { inherit inputs; };
    testItemLookup = callPackage ./testItemLookup.nix { inherit inputs lib; };
    testEnrichmentOutput = callPackage ./testEnrichmentOutput.nix { inherit inputs lib; };
    testEquipmentEnrichment = callPackage ./testEquipmentEnrichment.nix { inherit inputs lib; };
  in {
    apps = simulation // {
      getDB = {
        type = "app";
        program = "${getDB}/bin/getDB";
      };
      testItemLookup = {
        type = "app";
        program = "${testItemLookup}/bin/testItemLookup";
      };
      testEnrichmentOutput = {
        type = "app";
        program = "${testEnrichmentOutput}/bin/testEnrichmentOutput";
      };
      testEquipmentEnrichment = {
        type = "app";
        program = "${testEquipmentEnrichment}/bin/testEquipmentEnrichment";
      };
    };
  };
}
