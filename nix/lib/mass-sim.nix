{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  ...
}: let
  inherit (lib.sim.simulation) mkSim;

  # Helper to get all available class/spec combinations for a given encounter type
  getAllSpecConfigs = encounterCategory: targetType: duration: classes:
    lib.flatten (lib.mapAttrsToList (
        className: classSpecs:
          lib.mapAttrsToList (
            specName: specConfigs:
              if
                lib.hasAttr "template" specConfigs
                && lib.hasAttr "p1" specConfigs.template
                && lib.hasAttr encounterCategory specConfigs.template.p1
                && lib.hasAttr targetType specConfigs.template.p1.${encounterCategory}
              then {
                inherit className specName;
                config = specConfigs.template.p1.${encounterCategory}.${targetType};
              }
              else null
          )
          classSpecs
      )
      classes);

  # Filter out null values
  filterValidConfigs = configs: lib.filter (x: x != null) configs;

  # Create simulation input for a specific spec
  createSimInput = {
    className,
    specName,
    config,
    encounterType,
    encounterCategory,
    targetType,
    duration,
  }:
    mkSim {
      requestId = "mass-sim-${className}-${specName}-${encounterCategory}-${targetType}-${duration}";
      iterations = 1000;
      player = config;
      buffs = buffs.full;
      debuffs = debuffs.full;
      encounter = encounter.${encounterCategory}.${duration}.${targetType};
    };

  # Mass simulation function
  mkMassSimulation = {
    encounterCategory ? "raid",
    targetType ? "singleTarget",
    duration ? "long",
  }: let
    # Get all valid spec configurations
    allSpecs = filterValidConfigs (getAllSpecConfigs encounterCategory targetType duration classes);

    # Create individual simulation derivations
    simDerivations = lib.listToAttrs (map (spec: {
        name = "${spec.className}-${spec.specName}";
        value = let
          simInput = createSimInput {
            inherit (spec) className specName config;
            encounterType = encounterCategory;
            inherit encounterCategory targetType duration;
          };
        in
          pkgs.runCommand "sim-${spec.className}-${spec.specName}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [pkgs.wowsimcli];
          } ''
            # Create input file
            cat > input.json << 'EOF'
            ${simInput}
            EOF

            # Run simulation
            wowsimcli sim --infile input.json --outfile output.json

            # Extract DPS and create result JSON
            avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)

            # Create output with DPS and loadout info
            jq -n --arg dps "$avgDps" --arg className "${spec.className}" --arg specName "${spec.specName}" \
               --argjson loadout '${builtins.toJSON spec.config}' \
               '{
                 dps: ($dps | tonumber),
                 className: $className,
                 specName: $specName,
                 loadout: $loadout
               }' > $out
          '';
      })
      allSpecs);

    # Aggregate all results into final JSON structure
    aggregatedResults =
      pkgs.runCommand "mass-sim-${encounterCategory}-${targetType}-${duration}" {
        buildInputs = [pkgs.jq];
      } ''
        # Create the nested JSON structure
        jq -n --argjson specs '{}' \
          '{ "${encounterCategory}": { "${targetType}": { "${duration}": $specs } } }' > result.json

        ${lib.concatMapStringsSep "\n" (spec: ''
            # Read result for ${spec.className}/${spec.specName}
            specResult=$(cat ${simDerivations."${spec.className}-${spec.specName}"})

            # Add to the nested structure
            jq --argjson spec "$specResult" \
               '.${encounterCategory}.${targetType}.${duration}."${spec.className}"."${spec.specName}" = {
                 dps: $spec.dps,
                 loadout: $spec.loadout
               }' result.json > temp.json && mv temp.json result.json
          '')
          allSpecs}

        # Final output
        cp result.json $out
      '';
  in {
    # Individual simulation results
    simulations = simDerivations;

    # Aggregated JSON output
    result = aggregatedResults;

    # Metadata
    metadata = {
      inherit encounterCategory targetType duration;
      specCount = lib.length allSpecs;
      specs = map (s: "${s.className}/${s.specName}") allSpecs;
    };
  };
in {
  inherit mkMassSimulation;

  # Pre-defined common simulation sets
  presets = {
    raidSingleTargetLong = mkMassSimulation {
      encounterCategory = "raid";
      targetType = "singleTarget";
      duration = "long";
    };

    raidMultiTargetLong = mkMassSimulation {
      encounterCategory = "raid";
      targetType = "multiTarget";
      duration = "long";
    };

    raidCleaveLong = mkMassSimulation {
      encounterCategory = "raid";
      targetType = "cleave";
      duration = "long";
    };
  };
}

