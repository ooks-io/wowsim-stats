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
  # TODO: this is cursed, consolidate the functions.
  inherit (lib.sim.simulation) mkSim;

  # extract playable races from class definitions
  getPlayableRaces = className:
    if lib.hasAttr className classes && lib.hasAttr "playableRaces" classes.${className}
    then classes.${className}.playableRaces
    else throw "playableRaces not defined for class ${className}. Please add playableRaces = [...] to nix/classes/${className}/default.nix";

  # categorize specs by role
  # TODO: tanks
  getAllDPSSpecs = classes: template: phase: let
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

    # extract specs that exist in classes and have the template structure
    validSpecs = lib.flatten (lib.mapAttrsToList (
        className: specNames:
          lib.filter (spec: spec != null) (map (
              specName:
                if
                  lib.hasAttr className classes
                  && lib.hasAttr specName classes.${className}
                  && lib.hasAttr "defaultRace" classes.${className}.${specName}
                  && lib.hasAttr "template" classes.${className}.${specName}
                then let
                  defaultRace = classes.${className}.${specName}.defaultRace;
                  spec = classes.${className}.${specName};
                in
                  if
                    lib.hasAttr defaultRace spec.template
                    && lib.hasAttr phase spec.template.${defaultRace}
                    && lib.hasAttr "raid" spec.template.${defaultRace}.${phase}
                    && lib.hasAttr template spec.template.${defaultRace}.${phase}.raid
                  then {
                    inherit className specName;
                    config = spec.template.${defaultRace}.${phase}.raid.${template};
                  }
                  else null
                else null
            )
            specNames)
      )
      dpsSpecs);
  in
    validSpecs;

  getRaceConfigs = classes: className: specName: template: phase: encounterType: let
    availableRaces = getPlayableRaces className;
    baseSpec = classes.${className}.${specName};
  in
    map (raceName: {
      inherit className specName raceName;
      config =
        if
          lib.hasAttr "template" baseSpec
          && lib.hasAttr raceName baseSpec.template
          && lib.hasAttr phase baseSpec.template.${raceName}
          && lib.hasAttr encounterType baseSpec.template.${raceName}.${phase}
          && lib.hasAttr template baseSpec.template.${raceName}.${phase}.${encounterType}
        then baseSpec.template.${raceName}.${phase}.${encounterType}.${template}
        else throw "Template ${template} not found for ${className}/${specName}/${raceName} at ${phase}.${encounterType}";
    })
    availableRaces;

  mkRaceComparison = {
    class,
    spec,
    encounter,
    iterations ? 10000,
    phase ? "p1",
    encounterType ? "raid",
    targetCount ? "single",
    duration ? "long",
    template ? "singleTarget",
  }: let
    raceConfigs = getRaceConfigs classes class spec template phase encounterType;

    # create individual simulation derivations for each race
    simDerivations = lib.listToAttrs (map (raceConfig: {
        name = "${raceConfig.className}-${raceConfig.specName}-${raceConfig.raceName}";
        value = let
          simInput = mkSim {
            inherit iterations;
            player = raceConfig.config;
            buffs = buffs.full;
            debuffs = debuffs.full;
            inherit encounter;
          };
        in
          pkgs.runCommand "race-sim-${raceConfig.className}-${raceConfig.specName}-${raceConfig.raceName}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
          } ''
            cat > input.json << 'EOF'
            ${simInput}
            EOF

            echo "Running ${raceConfig.className}/${raceConfig.specName}/${raceConfig.raceName} simulation..."
            if wowsimcli sim --infile input.json --outfile output.json; then
              # extract dps statistics
              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # construct loadout info
              loadout=$(echo '${builtins.toJSON raceConfig.config}' | jq '{
                consumables,
                talentsString,
                glyphs,
                equipment,
                race,
                class,
                profession1,
                profession2
              }')

              # create final result
              jq -n \
                --arg raceName "${raceConfig.raceName}" \
                --arg avgDps "$avgDps" \
                --arg maxDps "$maxDps" \
                --arg minDps "$minDps" \
                --arg stdevDps "$stdevDps" \
                --argjson loadout "$loadout" \
                '{
                  race: $raceName,
                  dps: ($avgDps | tonumber),
                  max: ($maxDps | tonumber),
                  min: ($minDps | tonumber),
                  stdev: ($stdevDps | tonumber),
                  loadout: $loadout
                }' > $out
            else
              echo "Simulation failed for ${raceConfig.className}/${raceConfig.specName}/${raceConfig.raceName}"
              exit 1
            fi
          '';
      })
      raceConfigs);

    structuredOutput = "${class}_${spec}_race_${phase}_${encounterType}_${targetCount}_${duration}";

    aggregationScript = pkgs.writeShellApplication {
      name = "${structuredOutput}-aggregator";
      text = ''
        set -euo pipefail

        echo "Aggregating race comparison results for: ${class}/${spec}"
        echo "Races simulated: ${toString (lib.length raceConfigs)}"

        result=$(jq -n '{}')

        ${lib.concatMapStringsSep "\n" (raceConfig: ''
            # ${raceConfig.raceName} results
            raceData=$(cat ${simDerivations."${raceConfig.className}-${raceConfig.specName}-${raceConfig.raceName}"})

            result=$(echo "$result" | jq \
              --argjson race "$raceData" \
              --arg raceName "${raceConfig.raceName}" \
              '.[$raceName] = {
                dps: $race.dps,
                max: $race.max,
                min: $race.min,
                stdev: $race.stdev,
                loadout: $race.loadout
              }')
          '')
          raceConfigs}

        finalResult=$(echo "$result" | jq \
          --arg class "${class}" \
          --arg spec "${spec}" \
          --arg encounter "${phase}_${encounterType}_${targetCount}_${duration}" \
          --arg timestamp "$(date -Iseconds)" \
          --arg iterations "${toString iterations}" \
          --arg encounterDuration "${toString encounter.duration}" \
          --arg encounterVariation "${toString encounter.durationVariation}" \
          --arg targetCount "${toString (lib.length encounter.targets)}" \
          --argjson raidBuffs '${builtins.toJSON buffs.full}' \
          '{
            metadata: {
              spec: $spec,
              class: $class,
              encounter: $encounter,
              timestamp: $timestamp,
              iterations: ($iterations | tonumber),
              encounterDuration: ($encounterDuration | tonumber),
              encounterVariation: ($encounterVariation | tonumber),
              targetCount: ($targetCount | tonumber),
              raidBuffs: $raidBuffs
            },
            results: .
          }')

        echo "$finalResult" | jq -c '.' | tee "${structuredOutput}.json"

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

        comparison_dir="$repo_root/web/public/data/comparison/${class}/${spec}"
        mkdir -p "$comparison_dir"

        # copy race comparison file
        cp "${structuredOutput}.json" "$comparison_dir/"
        echo "Copied to: $comparison_dir/${structuredOutput}.json"

        echo ""
        echo "Race DPS Rankings for ${class}/${spec}:"
        echo "======================================="
        echo "$finalResult" | jq -r '
          .results | to_entries[] |
          select(.value.dps != null and .value.dps > 0) |
          "\(.key): \(.value.dps | floor) DPS"
        ' | sort -k2 -nr

        # Show any failed races
        failed_races=$(echo "$finalResult" | jq -r '
          .results | to_entries[] |
          select(.value.dps == null or .value.dps <= 0) |
          .key
        ')

        if [[ -n "$failed_races" ]]; then
          echo ""
          echo "Failed races (no valid DPS data):"
          echo "$failed_races"
        fi

        echo ""
        echo "Results written to: $comparison_dir/${structuredOutput}.json"
      '';
      runtimeInputs = [pkgs.jq pkgs.coreutils];
    };
  in {
    simulations = simDerivations;

    script = aggregationScript;

    metadata = {
      output = structuredOutput;
      inherit class spec iterations phase encounterType targetCount duration;
      raceCount = lib.length raceConfigs;
      races = map (r: r.raceName) raceConfigs;
    };
  };

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
    # get the list of specs based on the specs parameter
    specConfigs =
      if specs == "dps"
      then getAllDPSSpecs classes template phase
      else if builtins.isList specs
      then specs
      else throw "specs must be 'dps' or a list of spec configurations";

    # create individual simulation derivations for each spec
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

            echo "Running ${spec.className}/${spec.specName} simulation..."
            if wowsimcli sim --infile input.json --outfile output.json; then

              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # create loadout without rotation (only keep consumables, talents, glyphs, gear)
              # TODO: should we output the apl?
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

              # create final result with all DPS statistics
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

        echo "$finalResult" | jq -c '.' | tee "${structuredOutput}.json"

        # copy to web public directory for web frontend
        # find the repo root by looking for flake.nix
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
        rankings_dir="$web_data_dir/rankings"
        archive_dir="$rankings_dir/archive"
        existing_file="$rankings_dir/${structuredOutput}.json"

        mkdir -p "$web_data_dir"
        mkdir -p "$rankings_dir"
        mkdir -p "$archive_dir"

        if [[ -f "$existing_file" ]]; then
          timestamp=$(date +"%Y%m%d_%H%M%S")
          archived_name="${structuredOutput}_$timestamp.json"
          cp "$existing_file" "$archive_dir/$archived_name"
          echo "Archived existing file: $archived_name"
        fi

        cp "${structuredOutput}.json" "$rankings_dir/"
        echo "Copied to: $rankings_dir/${structuredOutput}.json"

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
  inherit mkMassSim getAllDPSSpecs mkRaceComparison getRaceConfigs getPlayableRaces;
}
