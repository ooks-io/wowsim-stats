{
  lib,
  pkgs,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  ...
}: let
  inherit (lib.sim.simulation) mkSim;
  inherit (lib.sim) itemDatabase shellUtils;
  inherit (lib) hasAttr length listToAttrs imap0 replaceStrings concatMapStringsSep;

  # Import our trinket manipulation functions
  trinketLib = lib.sim.trinket;

  mkTrinketComparison = {
    class,
    spec,
    encounter,
    trinketCategory ? "meleeAgility", # Default to melee agility trinkets
    trinketIds ? null, # Optional: provide trinket IDs directly
    iterations ? 10000,
    phase ? "p1",
    encounterType ? "raid",
    targetCount ? "single",
    duration ? "long",
    template ? "singleTarget",
    wowsimsCommit ? inputs.wowsims-upstream.shortRev,
  }: let
    # Get the base spec configuration
    baseSpec = classes.${class}.${spec};
    defaultRace = baseSpec.defaultRace;

    # Get the base player configuration
    baseConfig =
      if
        hasAttr "template" baseSpec
        && hasAttr defaultRace baseSpec.template
        && hasAttr phase baseSpec.template.${defaultRace}
        && hasAttr encounterType baseSpec.template.${defaultRace}.${phase}
        && hasAttr template baseSpec.template.${defaultRace}.${phase}.${encounterType}
      then baseSpec.template.${defaultRace}.${phase}.${encounterType}.${template}
      else throw "Template ${template} not found for ${class}/${spec}/${defaultRace} at ${phase}.${encounterType}";

    # Load trinket list - either from direct input or from trinket presets
    actualTrinketIds =
      if trinketIds != null
      then trinketIds
      else trinket.presets.p1.${trinketCategory};

    # Create baseline gearset (no trinkets)
    baselineGearset = trinketLib.removeTrinkets baseConfig.equipment;

    # Create baseline player config
    baselineConfig = baseConfig // {equipment = baselineGearset;};

    # Trinket profession requirements mapping
    trinketProfessionRequirements = {
      "75274" = "Alchemy"; # Zen Alchemist Stone
      # Add other profession-specific trinkets here as needed
    };

    # Generate trinket gearsets
    trinketGearsets = trinketLib.generateTrinketGearsets baselineGearset actualTrinketIds;

    # Create trinket configs with gearsets and profession adjustments
    trinketConfigs =
      imap0 (index: gearset: let
        trinketId = builtins.elemAt actualTrinketIds index;
        trinketIdStr = toString trinketId;
        requiredProfession = trinketProfessionRequirements.${trinketIdStr} or null;

        # Modify profession if trinket requires it
        configWithProfession =
          if requiredProfession != null
          then
            baseConfig
            // {
              equipment = gearset;
              profession2 = requiredProfession;
            }
          else baseConfig // {equipment = gearset;};
      in {
        trinketId = trinketId;
        config = configWithProfession;
      })
      trinketGearsets;

    # Class-specific buff adjustments (copied from mkMassSim)
    classSpecificBuffs =
      if baseConfig.class == "ClassWarrior"
      then buffs.full // {skullBannerCount = 1;}
      else if baseConfig.class == "ClassShaman"
      then buffs.full // {stormlashTotemCount = 3;}
      else buffs.full;

    # Create baseline simulation
    baselineSim = let
      enrichedEquipment = itemDatabase.enrichEquipment baselineConfig.equipment;
      enrichedConsumables = itemDatabase.enrichConsumables baselineConfig.consumables;
      enrichedGlyphs = itemDatabase.enrichGlyphs baselineConfig.class baselineConfig.glyphs;
      enrichedTalents = itemDatabase.enrichTalents baselineConfig.class baselineConfig.talentsString;

      simInput = mkSim {
        inherit iterations;
        player = baselineConfig;
        buffs = classSpecificBuffs;
        debuffs = debuffs.full;
        inherit encounter;
      };
    in
      pkgs.runCommand "trinket-baseline-${class}-${spec}" {
        buildInputs = [pkgs.jq];
        nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
      } ''
        # Write enriched JSON data to files
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

        echo "Running baseline (no trinkets) simulation for ${class}/${spec}..."
        if wowsimcli sim --infile input.json --outfile output.json; then
          # Extract DPS statistics
          avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
          maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
          minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
          stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

          # Generate wowsim link from input file
          echo "Generating wowsim link..."
          simLink=$(wowsimcli encodelink input.json || echo "")

          # Create final result
          jq -n \
            --arg avgDps "$avgDps" \
            --arg maxDps "$maxDps" \
            --arg minDps "$minDps" \
            --arg stdevDps "$stdevDps" \
            --arg simLink "$simLink" \
            --slurpfile equipment enriched_equipment.json \
            --slurpfile consumables consumables.json \
            --arg talentsString "${baselineConfig.talentsString}" \
            --slurpfile talents talents.json \
            --slurpfile glyphs glyphs.json \
            --arg race "${baselineConfig.race}" \
            --arg class "${baselineConfig.class}" \
            --arg profession1 "${baselineConfig.profession1}" \
            --arg profession2 "${baselineConfig.profession2}" \
            '{
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
          echo "Baseline simulation failed for ${class}/${spec}"
          exit 1
        fi
      '';

    # Create individual trinket simulations
    trinketSims = listToAttrs (map (trinketConfig: let
        trinketItem = itemDatabase.getItem trinketConfig.trinketId;
        trinketName =
          if trinketItem != null
          then trinketItem.name
          else "Unknown_${toString trinketConfig.trinketId}";
        trinketIlvl =
          if trinketItem != null && trinketItem ? scalingOptions && trinketItem.scalingOptions ? "0"
          then toString trinketItem.scalingOptions."0".ilvl
          else "0";
        # Clean name for use as attribute key
        cleanName = replaceStrings [" " "'" "," "(" ")"] ["_" "" "" "" ""] trinketName;
      in {
        name = "${cleanName}_${trinketIlvl}";
        value = let
          enrichedEquipment = itemDatabase.enrichEquipment trinketConfig.config.equipment;
          enrichedConsumables = itemDatabase.enrichConsumables trinketConfig.config.consumables;
          enrichedGlyphs = itemDatabase.enrichGlyphs trinketConfig.config.class trinketConfig.config.glyphs;
          enrichedTalents = itemDatabase.enrichTalents trinketConfig.config.class trinketConfig.config.talentsString;

          simInput = mkSim {
            inherit iterations;
            player = trinketConfig.config;
            buffs = classSpecificBuffs;
            debuffs = debuffs.full;
            inherit encounter;
          };
        in
          pkgs.runCommand "trinket-sim-${class}-${spec}-${toString trinketConfig.trinketId}" {
            buildInputs = [pkgs.jq];
            nativeBuildInputs = [inputs.wowsims.packages.${pkgs.system}.wowsimcli];
          } ''
            # Write enriched JSON data to files
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

            echo "Running trinket simulation for ${class}/${spec} with trinket ${toString trinketConfig.trinketId}..."
            if wowsimcli sim --infile input.json --outfile output.json; then
              # Extract DPS statistics
              avgDps=$(jq -r '.raidMetrics.dps.avg // 0' output.json)
              maxDps=$(jq -r '.raidMetrics.dps.max // 0' output.json)
              minDps=$(jq -r '.raidMetrics.dps.min // 0' output.json)
              stdevDps=$(jq -r '.raidMetrics.dps.stdev // 0' output.json)

              # Generate wowsim link from input file
              echo "Generating wowsim link..."
              simLink=$(wowsimcli encodelink input.json || echo "")

              # Get trinket metadata
              trinketName="${trinketName}"
              trinketId="${toString trinketConfig.trinketId}"

              # Create final result with trinket metadata
              jq -n \
                --arg avgDps "$avgDps" \
                --arg maxDps "$maxDps" \
                --arg minDps "$minDps" \
                --arg stdevDps "$stdevDps" \
                --arg simLink "$simLink" \
                --arg trinketName "$trinketName" \
                --arg trinketId "$trinketId" \
                --slurpfile equipment enriched_equipment.json \
                --slurpfile consumables consumables.json \
                --arg talentsString "${trinketConfig.config.talentsString}" \
                --slurpfile talents talents.json \
                --slurpfile glyphs glyphs.json \
                --arg race "${trinketConfig.config.race}" \
                --arg class "${trinketConfig.config.class}" \
                --arg profession1 "${trinketConfig.config.profession1}" \
                --arg profession2 "${trinketConfig.config.profession2}" \
                '{
                  dps: ($avgDps | tonumber),
                  max: ($maxDps | tonumber),
                  min: ($minDps | tonumber),
                  stdev: ($stdevDps | tonumber),
                  trinket: {
                    id: ($trinketId | tonumber),
                    name: $trinketName
                  },
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
              echo "Trinket simulation failed for ${class}/${spec} with trinket ${toString trinketConfig.trinketId}"
              exit 1
            fi
          '';
      })
      trinketConfigs);

    # Generate structured output filename
    structuredOutput = "${class}_${spec}_trinket_${phase}_${encounterType}_${targetCount}_${duration}";

    # Aggregation script that combines baseline + all trinket results
    aggregationScript = pkgs.writeShellApplication {
      name = "${structuredOutput}-aggregator";
      text = ''
        ${shellUtils.parseArgsAndEnv}

        echo "Aggregating trinket comparison results for: ${class}/${spec}"
        echo "Trinkets tested: ${toString (builtins.length actualTrinketIds)}"

        # Start with baseline result
        result=$(jq -n '{}')

        # Add baseline
        baselineData=$(cat ${baselineSim})
        result=$(echo "$result" | jq \
          --argjson baseline "$baselineData" \
          '.baseline = $baseline')

        # Add each trinket result
        ${concatMapStringsSep "\n" (trinketConfig: let
            trinketItem = itemDatabase.getItem trinketConfig.trinketId;
            trinketName =
              if trinketItem != null
              then trinketItem.name
              else "Unknown_${toString trinketConfig.trinketId}";
            trinketIlvl =
              if trinketItem != null && trinketItem ? scalingOptions && trinketItem.scalingOptions ? "0"
              then toString trinketItem.scalingOptions."0".ilvl
              else "0";
            cleanName = replaceStrings [" " "'" "," "(" ")"] ["_" "" "" "" ""] trinketName;
            keyName = "${cleanName}_${trinketIlvl}";
          in ''
            trinketData=$(cat ${trinketSims.${keyName}})
            result=$(echo "$result" | jq \
              --argjson trinket "$trinketData" \
              --arg keyName "${keyName}" \
              '.[$keyName] = $trinket')
          '')
          trinketConfigs}

        # Create final output with metadata
        finalResult=$(echo "$result" | jq \
          --arg class "${class}" \
          --arg spec "${spec}" \
          --arg comparison "trinket" \
          --arg baseline "no_trinkets" \
          --arg encounter "${phase}_${encounterType}_${targetCount}_${duration}" \
          --arg timestamp "$(date -Iseconds)" \
          --arg iterations "${toString iterations}" \
          --arg encounterDuration "${toString encounter.duration}" \
          --arg encounterVariation "${toString encounter.durationVariation}" \
          --arg targetCount "${toString (length encounter.targets)}" \
          --arg wowsimsCommit "${wowsimsCommit}" \
          --arg trinketCategory "${trinketCategory}" \
          --argjson raidBuffs '${builtins.toJSON classSpecificBuffs}' \
          '{
            metadata: {
              spec: $spec,
              class: $class,
              comparison: $comparison,
              baseline: $baseline,
              encounter: $encounter,
              timestamp: $timestamp,
              iterations: ($iterations | tonumber),
              encounterDuration: ($encounterDuration | tonumber),
              encounterVariation: ($encounterVariation | tonumber),
              targetCount: ($targetCount | tonumber),
              wowsimsCommit: $wowsimsCommit,
              trinketCategory: $trinketCategory,
              raidBuffs: $raidBuffs
            },
            results: .
          }')

        ${shellUtils.conditionalOutput {
          inherit structuredOutput;
          webSetupCode = shellUtils.setupComparisonDirs {
            comparisonType = "trinkets";
            class = "${class}";
            spec = "${spec}";
          };
          webPath = "$comparison_dir";
          webMessage = "Copied to";
        }}

        echo ""
        echo "Trinket DPS Rankings for ${class}/${spec}:"
        echo "======================================="

        # Show baseline DPS
        echo "Baseline (no trinkets): $(echo "$finalResult" | jq -r '.results.baseline.dps | floor') DPS"
        echo ""

        # Show trinket rankings sorted by DPS
        echo "$finalResult" | jq -r '
          .results | to_entries[] |
          select(.key != "baseline" and .value.dps != null and .value.dps > 0) |
          "\(.value.trinket.name): \(.value.dps | floor) DPS"
        ' | sort -k2 -nr

      '';
      runtimeInputs = [pkgs.jq pkgs.coreutils];
    };
  in {
    # Individual simulation derivations
    simulations = trinketSims // {baseline = baselineSim;};

    # Main aggregation script
    script = aggregationScript;

    # Metadata
    metadata = {
      inherit class spec trinketCategory;
      output = structuredOutput;
      inherit iterations phase encounterType targetCount duration;
      trinketCount = length actualTrinketIds;
      trinkets = actualTrinketIds;
    };
  };
in {
  inherit mkTrinketComparison;
}
