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

  getAllDPSSpecs = classes: template: let
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
                  && lib.hasAttr template classes.${className}.${specName}.template.p1.raid
                then {
                  inherit className specName;
                  config = classes.${className}.${specName}.template.p1.raid.${template};
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
    template ? "singleTarget",
  }: let
    # Get the list of specs based on the specs parameter
    specConfigs =
      if specs == "dps"
      then getAllDPSSpecs classes template
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
        archive_dir="$web_data_dir/archive"
        existing_file="$web_data_dir/${structuredOutput}.json"

        mkdir -p "$web_data_dir"
        mkdir -p "$archive_dir"

        # Archive existing file if it exists
        if [[ -f "$existing_file" ]]; then
          timestamp=$(date +"%Y%m%d_%H%M%S")
          archived_name="${structuredOutput}_$timestamp.json"
          cp "$existing_file" "$archive_dir/$archived_name"
          echo "Archived existing file: $archived_name"

          # Generate changelog by comparing with archived version
          echo "Generating changelog..."
          changes=$(echo "$finalResult" | jq --slurpfile archived "$existing_file" '
            .results as $current |
            $archived[0].results as $previous |
            [
              $current | to_entries[] | . as $class_entry |
              $class_entry.value | to_entries[] | . as $spec_entry |
              {
                class: $class_entry.key,
                spec: $spec_entry.key,
                current_dps: $spec_entry.value.dps,
                previous_dps: ($previous[$class_entry.key][$spec_entry.key].dps // null)
              } |
              select(.previous_dps != null and .current_dps != .previous_dps) |
              {
                class: .class,
                spec: .spec,
                current_dps: (.current_dps | round),
                previous_dps: (.previous_dps | round),
                absolute_change: ((.current_dps - .previous_dps) | round),
                percent_change: (((.current_dps - .previous_dps) / .previous_dps * 100) | (. * 100 | round) / 100)
              }
            ]
          ')

          change_count=$(echo "$changes" | jq length)
          if [[ "$change_count" -gt 0 ]]; then
            echo "Found $change_count DPS changes:"
            echo "$changes" | jq -r '.[] | "  \(.class)/\(.spec): \(if .absolute_change > 0 then "+" else "" end)\(.absolute_change) DPS (\(if .percent_change > 0 then "+" else "" end)\(.percent_change)%)"'

            # Update or create changelog
            changelog_file="$web_data_dir/changelog.json"
            if [[ -f "$changelog_file" ]]; then
              current_changelog=$(cat "$changelog_file")
            else
              current_changelog='{}'
            fi

            updated_changelog=$(echo "$current_changelog" | jq \
              --arg sim "${structuredOutput}" \
              --arg timestamp "$(date -Iseconds)" \
              --argjson changes "$changes" '
              .[$sim] = (.[$sim] // []) + [{
                timestamp: $timestamp,
                changes: $changes
              }]')

            echo "$updated_changelog" > "$changelog_file"
            echo "Updated changelog: $changelog_file"
          else
            echo "No DPS changes detected."
          fi
        fi

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
