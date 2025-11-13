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
  config = import ./config.nix {inherit encounter;};
  massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
  inherit (massSimFunctions) getAllDPSSpecs getRaceConfigs getPlayableRaces;

  # Generate scenarios (target count × duration combinations)
  generateScenarios = targetConfigs: durations:
    lib.flatten (lib.mapAttrsToList (
        targetCount: targetConfig:
          map (duration: {
            inherit targetCount duration;
            inherit (targetConfig) template;
            encounter = targetConfig.encounters.${duration};
          })
          durations
      )
      targetConfigs);

  scenarios = generateScenarios config.targetConfigs config.durations;

  # Helper to make simulation ID
  makeSimID = {
    class,
    spec,
    phase,
    targetCount,
    duration,
    race ? null,
    trinket ? null,
  }:
    if race != null
    then "${class}-${spec}-${race}-${phase}-${targetCount}-${duration}"
    else if trinket != null
    then "${class}-${spec}-${toString trinket}-${phase}-${targetCount}-${duration}"
    else "${class}-${spec}-${phase}-${targetCount}-${duration}";

  # Generate all benchmark (DPS) specs across scenarios
  allBenchmarkSpecs = lib.flatten (map (
      phase:
        lib.flatten (map (scenario: let
          validSpecs = getAllDPSSpecs classes scenario.template phase;
        in
          map (spec: {
            id = makeSimID {
              class = spec.className;
              spec = spec.specName;
              inherit phase;
              inherit (scenario) targetCount duration;
            };
            inherit (spec) className specName config;
            inherit phase;
            inherit (scenario) targetCount duration encounter;
          })
          validSpecs)
        scenarios)
    )
    config.phases);

  # Generate all race comparison configs
  allRaceConfigs = lib.flatten (map (
      specConfig:
        lib.flatten (map (scenario: let
          raceConfigs = getRaceConfigs classes specConfig.class specConfig.spec scenario.template "p1" "raid";
        in
          map (raceConfig: {
            id = makeSimID {
              class = raceConfig.className;
              spec = raceConfig.specName;
              race = raceConfig.raceName;
              phase = "p1";
              inherit (scenario) targetCount duration;
            };
            inherit (raceConfig) className specName raceName config;
            phase = "p1";
            inherit (scenario) targetCount duration encounter;
          })
          raceConfigs)
        scenarios)
    )
    config.raceComparisonSpecs);

  # Generate all trinket comparison configs
  allTrinketConfigs = lib.flatten (map (
      specConfig:
        lib.flatten (map (scenario: let
          # Get the spec's default race configuration
          spec = classes.${specConfig.class}.${specConfig.spec};
          defaultRace = spec.defaultRace;
          baseConfig =
            if
              lib.hasAttr "template" spec
              && lib.hasAttr defaultRace spec.template
              && lib.hasAttr "p1" spec.template.${defaultRace}
              && lib.hasAttr "raid" spec.template.${defaultRace}.p1
              && lib.hasAttr scenario.template spec.template.${defaultRace}.p1.raid
            then spec.template.${defaultRace}.p1.raid.${scenario.template}
            else throw "Template ${scenario.template} not found for ${specConfig.class}/${specConfig.spec}/${defaultRace}";

          # Get trinket list for this category from presets.p1
          trinketList = trinket.presets.p1.${specConfig.trinketCategory} or [];
        in
          map (trinketID: {
            id = makeSimID {
              class = specConfig.class;
              spec = specConfig.spec;
              trinket = trinketID;
              phase = "p1";
              inherit (scenario) targetCount duration;
            };
            className = specConfig.class;
            specName = specConfig.spec;
            inherit trinketID;
            # Create config with trinket equipped
            config =
              baseConfig
              // {
                equipment =
                  baseConfig.equipment
                  // {
                    items = let
                      items = baseConfig.equipment.items;
                      # Replace trinket slot 12 (first trinket slot)
                      replaceAt = index: value: list:
                        lib.imap0 (i: v:
                          if i == index
                          then value
                          else v)
                        list;
                    in
                      replaceAt 12 {id = trinketID;} items;
                  };
              };
            phase = "p1";
            inherit (scenario) targetCount duration encounter;
          })
          trinketList)
        scenarios)
    )
    config.trinketComparisonSpecs);

  wowsimsCommit = inputs.wowsims-upstream.shortRev;

  # Create class-specific buff adjustments
  getClassSpecificBuffs = class:
    if class == "ClassWarrior"
    then buffs.full // {skullBannerCount = 1;}
    else if class == "ClassShaman"
    then buffs.full // {stormlashTotemCount = 3;}
    else buffs.full;

  # Main derivation that generates all input files
  simInputs = pkgs.runCommand "sim-inputs" {} ''
    mkdir -p $out/inputs/{benchmark,race,trinket}

    echo "Generating benchmark simulation inputs..."
    ${lib.concatMapStringsSep "\n" (spec: ''
        cat > "$out/inputs/benchmark/${spec.id}.json" <<'EOF'
        ${mkSim {
          iterations = config.common.iterations;
          player = spec.config;
          buffs = getClassSpecificBuffs spec.config.class;
          debuffs = debuffs.full;
          encounter = spec.encounter;
        }}
        EOF
      '')
      allBenchmarkSpecs}

    echo "Generating race comparison inputs..."
    ${lib.concatMapStringsSep "\n" (raceConfig: ''
        cat > "$out/inputs/race/${raceConfig.id}.json" <<'EOF'
        ${mkSim {
          iterations = config.common.iterations;
          player = raceConfig.config;
          buffs = getClassSpecificBuffs raceConfig.config.class;
          debuffs = debuffs.full;
          encounter = raceConfig.encounter;
        }}
        EOF
      '')
      allRaceConfigs}

    echo "Generating trinket comparison inputs..."
    ${lib.concatMapStringsSep "\n" (trinketConfig: ''
        cat > "$out/inputs/trinket/${trinketConfig.id}.json" <<'EOF'
        ${mkSim {
          iterations = config.common.iterations;
          player = trinketConfig.config;
          buffs = getClassSpecificBuffs trinketConfig.config.class;
          debuffs = debuffs.full;
          encounter = trinketConfig.encounter;
        }}
        EOF
      '')
      allTrinketConfigs}

    echo "Generating manifest..."
    cat > "$out/inputs/manifest.json" <<'EOF'
    ${builtins.toJSON {
      version = wowsimsCommit;
      simulations = {
        benchmark =
          map (s: {
            inherit (s) id className specName phase targetCount duration;
            file = "benchmark/${s.id}.json";
            class = s.className;
            spec = s.specName;
          })
          allBenchmarkSpecs;
        race =
          map (r: {
            inherit (r) id className specName raceName phase targetCount duration;
            file = "race/${r.id}.json";
            class = r.className;
            spec = r.specName;
            race = r.raceName;
          })
          allRaceConfigs;
        trinket =
          map (t: {
            inherit (t) id className specName trinketID phase targetCount duration;
            file = "trinket/${t.id}.json";
            class = t.className;
            spec = t.specName;
            trinket = t.trinketID;
          })
          allTrinketConfigs;
      };
    }}
    EOF

    echo "Generated simulation inputs:"
    echo "  Benchmark: ${toString (lib.length allBenchmarkSpecs)} inputs"
    echo "  Race: ${toString (lib.length allRaceConfigs)} inputs"
    echo "  Trinket: ${toString (lib.length allTrinketConfigs)} inputs"
  '';
in {
  inherit simInputs;
  # Export for debugging
  inherit allBenchmarkSpecs allRaceConfigs allTrinketConfigs;
}
