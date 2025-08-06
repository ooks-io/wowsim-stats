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
  # TODO: use writers.writePython3Bin
  inherit (lib.sim.simulation) mkSim;
  inherit (lib.sim) itemDatabase shellUtils;
  inherit (lib) length listToAttrs hasAttr flatten mapAttrsToList;

  # extract playable races from class definitions
  getPlayableRaces = className:
    if lib.hasAttr className classes && hasAttr "playableRaces" classes.${className}
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
    validSpecs = flatten (mapAttrsToList (
        className: specNames:
          lib.filter (spec: spec != null) (map (
              specName:
                if
                  hasAttr className classes
                  && hasAttr specName classes.${className}
                  && hasAttr "defaultRace" classes.${className}.${specName}
                  && hasAttr "template" classes.${className}.${specName}
                then let
                  inherit (classes.${className}.${specName}) defaultRace;
                  spec = classes.${className}.${specName};
                in
                  if
                    hasAttr defaultRace spec.template
                    && hasAttr phase spec.template.${defaultRace}
                    && hasAttr "raid" spec.template.${defaultRace}.${phase}
                    && hasAttr template spec.template.${defaultRace}.${phase}.raid
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
          # validate template exists
          hasAttr "template" baseSpec
          && hasAttr raceName baseSpec.template
          && hasAttr phase baseSpec.template.${raceName}
          && hasAttr encounterType baseSpec.template.${raceName}.${phase}
          && hasAttr template baseSpec.template.${raceName}.${phase}.${encounterType}
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
    wowsimsCommit ? inputs.wowsims-upstream.shortRev,
  }: let
    raceConfigs = getRaceConfigs classes class spec template phase encounterType;

    # create individual simulation derivations for each race
    simDerivations = listToAttrs (map (raceConfig: {
        name = "${raceConfig.className}-${raceConfig.specName}-${raceConfig.raceName}";
        value = let
          # pre enrich the equipment, consumables, glyphs, and talents for this race config
          enrichedEquipment = itemDatabase.enrichEquipment raceConfig.config.equipment;
          enrichedConsumables = itemDatabase.enrichConsumables raceConfig.config.consumables;
          enrichedGlyphs = itemDatabase.enrichGlyphs raceConfig.config.class raceConfig.config.glyphs;
          enrichedTalents = itemDatabase.enrichTalents raceConfig.config.class raceConfig.config.talentsString;

          # warrior and shaman should be included in the their respective buff count, not in addition to
          classSpecificBuffs =
            if raceConfig.config.class == "ClassWarrior"
            then buffs.full // {skullBannerCount = 1;}
            else if raceConfig.config.class == "ClassShaman"
            then buffs.full // {stormlashTotemCount = 3;}
            else buffs.full;

          simInput = mkSim {
            inherit iterations;
            player = raceConfig.config;
            buffs = classSpecificBuffs;
            debuffs = debuffs.full;
            inherit encounter;
          };
        in
          pkgs.runCommand "race-sim-${raceConfig.className}-${raceConfig.specName}-${raceConfig.raceName}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
          } ''
            # Write enriched JSON data to files using cat with EOF to handle any quotes safely
            cat > enriched_equipment.json << 'EQUIPMENT_EOF'
              ${builtins.toJSON enrichedEquipment}
            EQUIPMENT_EOF

            cat > consumables.json << 'CONSUMABLES_EOF'
              ${builtins.toJSON enrichedConsumables}
            CONSUMABLES_EOF

            cat > glyphs.json << 'GLYPHS_EOF'
              ${builtins.toJSON enrichedGlyphs}
            GLYPHS_EOF

            cat > talents.json << 'TALENTS_EOF'
              ${builtins.toJSON enrichedTalents}
            TALENTS_EOF

            cat > input.json << 'INPUT_EOF'
              ${simInput}
            INPUT_EOF

            echo "Running ${raceConfig.className}/${raceConfig.specName}/${raceConfig.raceName} simulation..."
            if wowsimcli sim --infile input.json --outfile output.json; then
              # extract dps statistics
              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # Generate wowsim link from input file
              echo "Generating wowsim link..."
              simLink=$(wowsimcli encodelink input.json || echo "")

              # create final result with enriched data
              jq -n \
                --arg raceName "${raceConfig.raceName}" \
                --arg avgDps "$avgDps" \
                --arg maxDps "$maxDps" \
                --arg minDps "$minDps" \
                --arg stdevDps "$stdevDps" \
                --arg simLink "$simLink" \
                --slurpfile equipment enriched_equipment.json \
                --slurpfile consumables consumables.json \
                --arg talentsString "${raceConfig.config.talentsString}" \
                --slurpfile talents talents.json \
                --slurpfile glyphs glyphs.json \
                --arg race "${raceConfig.config.race}" \
                --arg class "${raceConfig.config.class}" \
                --arg profession1 "${raceConfig.config.profession1}" \
                --arg profession2 "${raceConfig.config.profession2}" \
                '{
                  race: $raceName,
                  dps: ($avgDps | tonumber),
                  max: ($maxDps | tonumber),
                  min: ($minDps | tonumber),
                  stdev: ($stdevDps | tonumber),
                  loadout: {
                    consumables: $consumables[0],
                    talentsString: $talentsString,
                    talents: $talents[0],
                    glyphs: $glyphs[0],
                    equipment: $equipment[0],
                    race: $race,
                    class: $class,
                    profession1: $profession1,
                    profession2: $profession2,
                    simLink: $simLink
                  }
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
        ${shellUtils.parseArgsAndEnv}

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
          --arg wowsimsCommit "${wowsimsCommit}" \
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
              wowsimsCommit: $wowsimsCommit,
              raidBuffs: $raidBuffs
            },
            results: .
          }')

        ${shellUtils.conditionalOutput {
          inherit structuredOutput;
          webSetupCode = shellUtils.setupComparisonDirs {
            comparisonType = "race";
            class = "${class}";
            spec = "${spec}";
          };
          webPath = "$comparison_dir";
          webMessage = "Copied to";
        }}

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
    wowsimsCommit ? inputs.wowsims-upstream.shortRev,
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
          # Pre-enrich the equipment, consumables, glyphs, and talents for this spec
          enrichedEquipment = itemDatabase.enrichEquipment spec.config.equipment;
          enrichedConsumables = itemDatabase.enrichConsumables spec.config.consumables;
          enrichedGlyphs = itemDatabase.enrichGlyphs spec.config.class spec.config.glyphs;
          enrichedTalents = itemDatabase.enrichTalents spec.config.class spec.config.talentsString;

          # Create class-specific buff adjustments
          classSpecificBuffs =
            if spec.config.class == "ClassWarrior"
            then buffs.full // {skullBannerCount = 1;}
            else if spec.config.class == "ClassShaman"
            then buffs.full // {stormlashTotemCount = 3;}
            else buffs.full;

          simInput = mkSim {
            inherit iterations;
            player = spec.config;
            buffs = classSpecificBuffs;
            debuffs = debuffs.full;
            inherit encounter;
          };
        in
          pkgs.runCommand "sim-${spec.className}-${spec.specName}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
          } ''
            cat > enriched_equipment.json << 'EQUIPMENT_EOF'
              ${builtins.toJSON enrichedEquipment}
            EQUIPMENT_EOF

            cat > consumables.json << 'CONSUMABLES_EOF'
              ${builtins.toJSON enrichedConsumables}
            CONSUMABLES_EOF

            cat > glyphs.json << 'GLYPHS_EOF'
              ${builtins.toJSON enrichedGlyphs}
            GLYPHS_EOF

            cat > talents.json << 'TALENTS_EOF'
              ${builtins.toJSON enrichedTalents}
            TALENTS_EOF

            cat > input.json << 'INPUT_EOF'
              ${simInput}
            INPUT_EOF

            echo "Running ${spec.className}/${spec.specName} simulation..."
            if wowsimcli sim --infile input.json --outfile output.json; then

              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # Generate wowsim link from input file
              echo "Generating wowsim link..."
              simLink=$(wowsimcli encodelink input.json || echo "")

              # create final result with all DPS statistics and enriched data
              jq -n \
                --arg className "${spec.className}" \
                --arg specName "${spec.specName}" \
                --arg avgDps "$avgDps" \
                --arg maxDps "$maxDps" \
                --arg minDps "$minDps" \
                --arg stdevDps "$stdevDps" \
                --arg simLink "$simLink" \
                --slurpfile equipment enriched_equipment.json \
                --slurpfile consumables consumables.json \
                --arg talentsString "${spec.config.talentsString}" \
                --slurpfile talents talents.json \
                --slurpfile glyphs glyphs.json \
                --arg race "${spec.config.race}" \
                --arg class "${spec.config.class}" \
                --arg profession1 "${spec.config.profession1}" \
                --arg profession2 "${spec.config.profession2}" \
                '{
                  className: $className,
                  specName: $specName,
                  dps: ($avgDps | tonumber),
                  max: ($maxDps | tonumber),
                  min: ($minDps | tonumber),
                  stdev: ($stdevDps | tonumber),
                  loadout: {
                    consumables: $consumables[0],
                    talentsString: $talentsString,
                    talents: $talents[0],
                    glyphs: $glyphs[0],
                    equipment: $equipment[0],
                    race: $race,
                    class: $class,
                    profession1: $profession1,
                    profession2: $profession2,
                    simLink: $simLink
                  }
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

        ${shellUtils.parseArgsAndEnv}

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
          --arg wowsimsCommit "${wowsimsCommit}" \
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
              wowsimsCommit: $wowsimsCommit,
              raidBuffs: $raidBuffs
            },
            results: .
          }')

        ${shellUtils.conditionalOutput {
          inherit structuredOutput;
          webSetupCode = shellUtils.setupRankingsDirs;
          webPath = "$rankings_dir";
          webMessage = "Copied to";
        }}

        echo ""
        echo "Top DPS Rankings:"
        echo "=================="
        echo "$finalResult" | jq -r '
          .results | to_entries[] as $class |
          $class.key as $className |
          $class.value | to_entries[] as $spec |
          "\($className)/\($spec.key): \($spec.value.dps | floor) DPS"
        ' | sort -k2 -nr | head -10

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
      specCount = length specConfigs;
      specs = map (s: "${s.className}/${s.specName}") specConfigs;
    };
  };
in {
  inherit mkMassSim getAllDPSSpecs mkRaceComparison getRaceConfigs getPlayableRaces;
}
