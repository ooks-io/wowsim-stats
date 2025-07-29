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
  trinketTest = mkTrinketComparison {
    class = "monk";
    spec = "windwalker";
    encounter = encounter.raid.long.singleTarget;
    trinketIds = [87167 87057 79328]; # Terror in Mists (heroic), Bottle of Stars (heroic), Relic of Xuen
    iterations = 100; # Keep very low for testing
  };

  # Get the first trinket simulation for testing (Terror in the Mists Celestial)
  testTrinketName = builtins.head (builtins.attrNames (builtins.removeAttrs trinketTest.simulations ["baseline"]));
in
  writeShellApplication {
    name = "trinket-single-test";
    runtimeInputs = [pkgs.jq];
    text = ''
      echo "=== Running Single Trinket Simulation Test ==="
      echo "Building single trinket simulation for Windwalker Monk..."
      echo "Testing trinket: ${testTrinketName}"
      echo ""

      # Build and run the single trinket simulation
      echo "Running trinket simulation..."
      trinket_result=$(nix build --no-link --print-out-paths ${trinketTest.simulations.${testTrinketName}})

      if [ -f "$trinket_result" ]; then
        echo "✅ Trinket simulation completed successfully!"
        echo ""

        echo "=== Trinket Results ==="
        trinket_name=$(jq -r '.trinket.name' "$trinket_result")
        trinket_id=$(jq -r '.trinket.id' "$trinket_result")
        echo "Trinket: $trinket_name (ID: $trinket_id)"

        dps=$(jq -r '.dps' "$trinket_result")
        echo "DPS: $dps"

        max_dps=$(jq -r '.max' "$trinket_result")
        echo "Max DPS: $max_dps"

        min_dps=$(jq -r '.min' "$trinket_result")
        echo "Min DPS: $min_dps"

        echo ""
        echo "=== Equipment Check ==="
        echo "Trinket in slot 13 (should contain the trinket):"
        jq -r '.loadout.equipment.items[12]' "$trinket_result"
        echo "Trinket in slot 14 (should be empty):"
        jq -r '.loadout.equipment.items[13]' "$trinket_result"

        echo ""
        echo "✅ Single trinket test completed!"
        echo ""
        echo "Next: Run 'nix run .#trinket-comparison-test' to test the full comparison with aggregation"
      else
        echo "❌ Trinket simulation failed!"
        exit 1
      fi
    '';
  }
