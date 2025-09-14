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
  self,
  ...
}: let
  mkTrinketComparison =
    (import "${self}/nix/apps/simulation/mkTrinketComparison.nix" {
      inherit lib pkgs classes encounter buffs debuffs inputs trinket;
    }).mkTrinketComparison;

  # Test creating a trinket comparison for windwalker with minimal iterations
  trinketTest = mkTrinketComparison {
    class = "monk";
    spec = "windwalker";
    encounter = encounter.raid.long.singleTarget;
    trinketIds = trinket.presets.meleeAgility; # Terror in Mists (heroic), Bottle of Stars (heroic), Relic of Xuen
    iterations = 100; # Keep very low for testing
  };
in
  writeShellApplication {
    name = "trinket-comparison-test";
    runtimeInputs = [pkgs.jq];
    text = ''
      echo "=== Running Full Trinket Comparison Test ==="
      echo "This will run the aggregation script to combine all trinket results"
      echo "Building aggregation script..."
      echo ""

      # Run the aggregation script
      echo "Running trinket comparison aggregation..."
      ${trinketTest.script}/bin/${trinketTest.metadata.output}-aggregator

      output_file="${trinketTest.metadata.output}.json"

      if [ -f "$output_file" ]; then
        echo ""
        echo "[OK] Full trinket comparison completed successfully!"
        echo ""

        echo "=== Final Results Summary ==="
        echo "Metadata:"
        jq -r '.metadata | "Class: \(.class), Spec: \(.spec), Trinkets: \(.trinketCategory), Iterations: \(.iterations)"' "$output_file"

        echo ""
        echo "Baseline DPS:"
        jq -r '.results.baseline.dps' "$output_file"

        echo ""
        echo "Top 3 Trinkets:"
        jq -r '.results | to_entries[] | select(.key != "baseline") | "\(.value.trinket.name): \(.value.dps | floor) DPS"' "$output_file" | sort -k2 -nr | head -3

        echo ""
        echo "Output written to: $output_file"
        echo ""
        echo "[OK] Full trinket comparison test completed!"
        echo ""
        echo "You can now examine the complete JSON output or run individual tests again."
      else
        echo "[ERROR] Trinket comparison aggregation failed!"
        exit 1
      fi
    '';
  }
