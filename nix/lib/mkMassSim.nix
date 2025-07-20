{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  ...
}: let
  inherit (lib.sim.simulation) mkSim;

  getAllDPSSpecs = classes: let
    dpsSpecs = {
      death_knight = ["frost" "unholy"];
      druid = ["balance" "feral"];
      hunter = ["beast_mastery" "marksmanship" "survival"];
      mage = ["arcane" "fire" "frost"];
      monk = ["windwalker"];
      paladin = ["retribution"];
      priest = ["shadow"];
      rogue = ["assassination" "combat" "subtlety"];
      shaman = ["elemental" "enhancement"];
      warlock = ["affliction" "demonology" "destruction"];
      warrior = ["arms" "fury"];
    };

    # Extract specs that exist in classes and have the template structure
    validSpecs = lib.flatten (lib.mapAttrsToList (
        className: specNames:
          lib.filter (spec: spec != null) (map (
              specName:
                if
                  lib.hasAttr className classes
                  && lib.hasAttr specName classes.${className}
                  && lib.hasAttr "template" classes.${className}.${specName}
                  && lib.hasAttr "p1" classes.${className}.${specName}.template
                  && lib.hasAttr "raid" classes.${className}.${specName}.template.p1
                  && lib.hasAttr "singleTarget" classes.${className}.${specName}.template.p1.raid
                then {
                  inherit className specName;
                  config = classes.${className}.${specName}.template.p1.raid.singleTarget;
                }
                else null
            )
            specNames)
      )
      dpsSpecs);
  in
    validSpecs;

  # Main mkMassSim function
  mkMassSim = {
    specs ? "dps",
    encounter,
    iterations ? 10000,
    phase ? "p1",
    encounterType ? "raid",
    targetCount ? "single",
    duration ? "long",
  }: let
    # Get the list of specs based on the specs parameter
    specConfigs =
      if specs == "dps"
      then getAllDPSSpecs classes
      else if builtins.isList specs
      then specs
      else throw "specs must be 'dps' or a list of spec configurations";

    # Create individual simulation derivations for each spec
    simDerivations = lib.listToAttrs (map (spec: {
        name = "${spec.className}-${spec.specName}";
        value = let
          simInput = mkSim {
            inherit iterations;
            player = spec.config;
            buffs = buffs.full;
            debuffs = debuffs.full;
            inherit encounter;
          };
        in
          pkgs.runCommand "sim-${spec.className}-${spec.specName}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
          } ''
            # Generate input JSON file using HERE document approach (same as test-composition)
            cat > input.json << 'EOF'
            ${simInput}
            EOF

            # Run simulation
            echo "Running ${spec.className}/${spec.specName} simulation..."
            if wowsimcli sim --infile input.json --outfile output.json; then
              # Extract DPS statistics
              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # Create loadout without rotation (only keep consumables, talents, glyphs, gear)
              loadout=$(echo '${builtins.toJSON spec.config}' | jq '{
                consumables,
                talentsString,
                glyphs,
                equipment,
                race,
                class,
                profession1,
                profession2
              }')

              # Create final result with all DPS statistics
              jq -n \
                --arg className "${spec.className}" \
                --arg specName "${spec.specName}" \
                --arg avgDps "$avgDps" \
                --arg maxDps "$maxDps" \
                --arg minDps "$minDps" \
                --arg stdevDps "$stdevDps" \
                --argjson loadout "$loadout" \
                '{
                  className: $className,
                  specName: $specName,
                  dps: ($avgDps | tonumber),
                  max: ($maxDps | tonumber),
                  min: ($minDps | tonumber),
                  stdev: ($stdevDps | tonumber),
                  loadout: $loadout
                }' > $out
            else
              echo "Simulation failed for ${spec.className}/${spec.specName}"
              exit 1
            fi
          '';
      })
      specConfigs);

    # generate structured output filename: <type>_<phase>_<encounter-type>_<target-count>_<duration>
    structuredOutput = "${specs}_${phase}_${encounterType}_${targetCount}_${duration}";

    # aggregation script that combines all results
    aggregationScript = pkgs.writeShellApplication {
      name = "${structuredOutput}-aggregator";
      text = ''
        set -euo pipefail

        echo "Aggregating mass simulation results for: ${structuredOutput}"
        echo "Specs simulated: ${toString (lib.length specConfigs)}"

        # create base structure
        result=$(jq -n '{}')

        ${lib.concatMapStringsSep "\n" (spec: ''
            # Add ${spec.className}/${spec.specName} results
            specData=$(cat ${simDerivations."${spec.className}-${spec.specName}"})

            result=$(echo "$result" | jq \
              --argjson spec "$specData" \
              --arg className "${spec.className}" \
              --arg specName "${spec.specName}" \
              '.[$className][$specName] = {
                dps: $spec.dps,
                max: $spec.max,
                min: $spec.min,
                stdev: $spec.stdev,
                loadout: $spec.loadout
              }')
          '')
          specConfigs}

        # Create final output with metadata including encounter information
        finalResult=$(echo "$result" | jq \
          --arg output "${structuredOutput}" \
          --arg timestamp "$(date -Iseconds)" \
          --arg iterations "${toString iterations}" \
          --arg specCount "${toString (lib.length specConfigs)}" \
          --arg encounterDuration "${toString encounter.duration}" \
          --arg encounterVariation "${toString encounter.durationVariation}" \
          --arg targetCount "${toString (lib.length encounter.targets)}" \
          --argjson raidBuffs '${builtins.toJSON buffs.full}' \
          '{
            metadata: {
              name: $output,
              timestamp: $timestamp,
              iterations: ($iterations | tonumber),
              specCount: ($specCount | tonumber),
              encounterDuration: ($encounterDuration | tonumber),
              encounterVariation: ($encounterVariation | tonumber),
              targetCount: ($targetCount | tonumber),
              raidBuffs: $raidBuffs
            },
            results: .
          }')

        echo "$finalResult" | tee "${structuredOutput}.json"

        # copy to web public directory for web frontend
        # Find the repo root by looking for flake.nix
        repo_root=""
        current_dir="$PWD"
        while [[ "$current_dir" != "/" ]]; do
          if [[ -f "$current_dir/flake.nix" ]]; then
            repo_root="$current_dir"
            break
          fi
          current_dir="$(dirname "$current_dir")"
        done
        
        if [[ -z "$repo_root" ]]; then
          echo "Warning: Could not find repo root (flake.nix), using current directory"
          repo_root="$PWD"
        fi
        
        web_data_dir="$repo_root/web/public/data"
        mkdir -p "$web_data_dir"
        cp "${structuredOutput}.json" "$web_data_dir/"
        echo "Copied to: $web_data_dir/${structuredOutput}.json"

        echo ""
        echo "Top DPS Rankings:"
        echo "=================="
        echo "$finalResult" | jq -r '
          .results | to_entries[] as $class |
          $class.key as $className |
          $class.value | to_entries[] as $spec |
          "\($className)/\($spec.key): \($spec.value.dps | floor) DPS"
        ' | sort -k2 -nr | head -10

        echo ""
        echo "Results written to: ${structuredOutput}.json"
      '';
      runtimeInputs = [pkgs.jq pkgs.coreutils];
    };
  in {
    # individual simulation derivations (for debugging)
    simulations = simDerivations;

    # main aggregation script
    script = aggregationScript;

    metadata = {
      output = structuredOutput;
      inherit iterations phase encounterType targetCount duration;
      specCount = lib.length specConfigs;
      specs = map (s: "${s.className}/${s.specName}") specConfigs;
    };
  };
in {
  inherit mkMassSim getAllDPSSpecs;
}

