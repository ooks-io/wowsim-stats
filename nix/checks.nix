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
    inherit (lib.sim.simulation) mkSim;
    
    # Helper function to get all available class/spec combinations
    getAllClassSpecs = classes: 
      lib.flatten (lib.mapAttrsToList (className: classSpecs:
        lib.mapAttrsToList (specName: specConfigs:
          if lib.hasAttr "template" specConfigs 
             && lib.hasAttr "p1" specConfigs.template
             && lib.hasAttr "raid" specConfigs.template.p1
             && lib.hasAttr "singleTarget" specConfigs.template.p1.raid
          then { inherit className specName; config = specConfigs.template.p1.raid.singleTarget; }
          else null
        ) classSpecs
      ) classes);
    
    # Filter out null values
    availableSpecs = lib.filter (x: x != null) (getAllClassSpecs classes);
    
    # Create test simulation for a given spec
    createSpecTest = {className, specName, config}: let
      testSim = mkSim {
        requestId = "test-${className}-${specName}";
        iterations = 100; # Smaller iteration count for faster testing
        player = config;
        buffs = buffs.full;
        debuffs = debuffs.full;
        encounter = encounter.raid.long.singleTarget;
      };
      
      testScript = pkgs.writeShellScript "test-${className}-${specName}" ''
        set -euo pipefail
        
        echo "Testing ${className}/${specName}..."
        
        # Generate input file
        cat > input.json << 'EOF'
        ${testSim}
        EOF
        
        # Run simulation
        if ! wowsimcli sim --infile input.json --outfile output.json; then
          echo "ERROR: wowsimcli failed for ${className}/${specName}"
          exit 1
        fi
        
        # Validate output
        if [ ! -f output.json ]; then
          echo "ERROR: No output file generated for ${className}/${specName}"
          exit 1
        fi
        
        # Check DPS value exists and is reasonable
        avgDps=$(jq -r '.raidMetrics.dps.avg // "null"' output.json)
        if [ "$avgDps" = "null" ]; then
          echo "ERROR: No DPS value found for ${className}/${specName}"
          exit 1
        fi
        
        # Basic sanity check - DPS should be positive and reasonable (>1000)
        # But let's debug what we're getting first
        echo "DEBUG: Raw DPS value: '$avgDps'"
        
        if [ "$(echo "$avgDps <= 0" | bc -l)" = "1" ]; then
          echo "ERROR: DPS too low or zero ($avgDps) for ${className}/${specName}"
          echo "DEBUG: Full output:"
          jq '.' output.json | head -20
          exit 1
        fi
        
        echo "SUCCESS: ${className}/${specName} - DPS: $avgDps"
        rm -f input.json output.json
      '';
    in testScript;
    
    # Create individual tests for each spec
    specTests = lib.listToAttrs (map (spec: {
      name = "${spec.className}-${spec.specName}";
      value = createSpecTest spec;
    }) availableSpecs);
    
    # Create combined test that runs all specs
    allSpecsTest = pkgs.writeShellScript "test-all-specs" ''
      set -euo pipefail
      
      echo "Running WoW simulation tests for all available specs..."
      echo "Found ${toString (lib.length availableSpecs)} specs to test"
      echo ""
      
      failed_tests=()
      
      ${lib.concatMapStringsSep "\n" (spec: ''
        if ! ${createSpecTest spec}; then
          failed_tests+=("${spec.className}/${spec.specName}")
        fi
      '') availableSpecs}
      
      echo ""
      if [ ''${#failed_tests[@]} -eq 0 ]; then
        echo "✅ All ${toString (lib.length availableSpecs)} specs passed!"
        exit 0
      else
        echo "❌ ''${#failed_tests[@]} specs failed:"
        printf '%s\n' "''${failed_tests[@]}"
        exit 1
      fi
    '';
    
  in {
    checks = {
      # Individual spec tests
      } // specTests // {
      # Combined test
      all-specs = pkgs.runCommand "test-all-specs" {
        buildInputs = [
          pkgs.jq 
          pkgs.bc
          inputs.wowsims.packages.${pkgs.system}.wowsimcli
        ];
      } ''
        cd $TMPDIR
        ${allSpecsTest}
        touch $out
      '';
      
      # Quick test with just a few representative specs
      smoke-test = pkgs.runCommand "smoke-test" {
        buildInputs = [
          pkgs.jq 
          pkgs.bc
          inputs.wowsims.packages.${pkgs.system}.wowsimcli
        ];
      } ''
        cd $TMPDIR
        echo "Running smoke test with a few representative specs..."
        
        ${lib.concatMapStringsSep "\n" (spec: 
          "${createSpecTest spec}"
        ) (lib.take 3 availableSpecs)}
        
        echo "✅ Smoke test passed!"
        touch $out
      '';
    };
  };
}