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

  # test creating a trinket comparison for windwalker with minimal iterations
  trinketTest = mkTrinketComparison {
    class = "monk";
    spec = "windwalker";
    encounter = encounter.raid.long.singleTarget;
    trinketIds = [87167 87057 79328];
    iterations = 100; # keep very low for testing
  };
in
  writeShellApplication {
    name = "trinket-baseline-test";
    runtimeInputs = [pkgs.jq];
    text = ''
      echo "=== Running Baseline (No Trinkets) Simulation Test ==="
      echo "Building baseline simulation for Windwalker Monk..."
      echo ""

      # Build and run the baseline simulation
      echo "Running baseline simulation..."
      baseline_result=$(nix build --no-link --print-out-paths ${trinketTest.simulations.baseline})

      if [ -f "$baseline_result" ]; then
        echo "Baseline simulation completed successfully!"
        echo ""

        echo "=== Baseline Results ==="
        dps=$(jq -r '.dps' "$baseline_result")
        echo "DPS: $dps"

        max_dps=$(jq -r '.max' "$baseline_result")
        echo "Max DPS: $max_dps"

        min_dps=$(jq -r '.min' "$baseline_result")
        echo "Min DPS: $min_dps"

        stdev=$(jq -r '.stdev' "$baseline_result")
        echo "StdDev: $stdev"

        echo ""
        echo "=== Equipment Check ==="
        echo "Trinket slots (should be empty):"
        jq -r '.loadout.equipment.items[12:14]' "$baseline_result"

        echo ""
        echo "Baseline test completed"
        echo ""
        echo "Next: Run 'nix run .#trinket-single-test' to test a single trinket simulation"
      else
        echo "Baseline simulation failed"
        exit 1
      fi
    '';
  }
