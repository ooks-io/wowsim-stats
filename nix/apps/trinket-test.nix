{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  writeShellApplication,
  ...
}: let
  mkTrinketComparison =
    (import ./simulation/mkTrinketComparison.nix {
      inherit lib pkgs classes encounter buffs debuffs inputs trinket;
    }).mkTrinketComparison;

  # Test creating a trinket comparison for windwalker with minimal iterations
  # Use direct trinket IDs to avoid path issues for now
  trinketTest = mkTrinketComparison {
    class = "monk";
    spec = "windwalker";
    encounter = encounter.raid.long.singleTarget;
    trinketIds = [87167 87057 79328]; # Terror in Mists (heroic), Bottle of Stars (heroic), Relic of Xuen
    iterations = 100; # Keep low for testing
  };
in
  writeShellApplication {
    name = "trinket-test";
    text = ''
      echo "=== Trinket Comparison Test ==="
      echo "Class: ${trinketTest.metadata.class}"
      echo "Spec: ${trinketTest.metadata.spec}"
      echo "Trinket Category: ${trinketTest.metadata.trinketCategory}"
      echo "Trinket Count: ${toString trinketTest.metadata.trinketCount}"
      echo "Output File: ${trinketTest.metadata.output}.json"
      echo ""

      echo "=== Trinket IDs ==="
      ${lib.concatMapStringsSep "\n" (
          trinketId: "echo \"- ${toString trinketId}\""
        )
        trinketTest.metadata.trinkets}
      echo ""

      echo "=== Available Simulations ==="
      ${lib.concatMapStringsSep "\n" (
        simName: "echo \"- ${simName}\""
      ) (builtins.attrNames trinketTest.simulations)}
      echo ""

      echo "=== Test Structure Complete ==="
      echo "Run 'nix run .#trinket-baseline-test' to test just the baseline simulation"
      echo "This test app validates that the trinket comparison structure is working"
    '';
  }
