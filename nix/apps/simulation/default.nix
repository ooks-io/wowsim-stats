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
  config = import ./config.nix {inherit encounter;};

  massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
  inherit (massSimFunctions) mkMassSim mkRaceComparison;

  trinketComparison = import ./mkTrinketComparison.nix {inherit lib pkgs classes encounter buffs debuffs inputs trinket;};
  inherit (trinketComparison) mkTrinketComparison;

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

  makeMassSimName = phase: targetCount: duration: "dps-${phase}-raid-${targetCount}-${duration}";

  massSimulations = lib.listToAttrs (lib.flatten (map (
      phase:
        map (scenario: {
          name = makeMassSimName phase scenario.targetCount scenario.duration;
          value = mkMassSim {
            inherit (config.common) iterations specs encounterType;
            inherit (scenario) encounter targetCount duration template;
            inherit phase;
          };
        })
        scenarios
    )
    config.phases));

  # Generate race comparisons (specs × scenarios)
  raceComparisons = lib.listToAttrs (lib.flatten (map (
      specConfig:
        map (scenario: {
          name = "race-${specConfig.class}-${specConfig.spec}-p1-raid-${scenario.targetCount}-${scenario.duration}";
          value = mkRaceComparison {
            inherit (specConfig) class spec;
            inherit (scenario) encounter targetCount duration template;
            inherit (config.common) encounterType iterations;
            # only simulate race benchmarks for p1
            phase = "p1";
          };
        })
        scenarios
    )
    config.raceComparisonSpecs));

  # Generate trinket comparisons (specs × scenarios)
  trinketComparisons = lib.listToAttrs (lib.flatten (map (
      specConfig:
        map (scenario: {
          name = "trinket-${specConfig.class}-${specConfig.spec}-p1-raid-${scenario.targetCount}-${scenario.duration}";
          value = mkTrinketComparison {
            inherit (specConfig) class spec trinketCategory;
            inherit (scenario) encounter duration template targetCount;
            inherit (config.common) encounterType iterations;
            phase = "p1";
          };
        })
        scenarios
    )
    config.trinketComparisonSpecs));

  # Script that runs all simulations
  allSimulationsScript = pkgs.writeShellApplication {
    name = "all-simulations";
    text = ''
      echo "Running all WoW simulations..."

      echo ""
      echo "=== DPS Rankings ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${massSimulations.${name}.script}/bin/${massSimulations.${name}.metadata.output}-aggregator
      '') (lib.attrNames massSimulations)}

      echo ""
      echo "=== Race Comparisons ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${raceComparisons.${name}.script}/bin/${raceComparisons.${name}.metadata.output}-aggregator
      '') (lib.attrNames raceComparisons)}

      echo ""
      echo "=== Trinket Comparisons ==="
      ${lib.concatMapStringsSep "\n" (name: ''
        echo ""
        echo "Running ${name}..."
        ${trinketComparisons.${name}.script}/bin/${trinketComparisons.${name}.metadata.output}-aggregator
      '') (lib.attrNames trinketComparisons)}

      echo ""
      echo "All simulations completed successfully!"
      echo ""
      echo "Generated DPS rankings:"
      ls -la web/public/data/rankings/*.json 2>/dev/null || echo "No ranking files found"
      echo ""
      echo "Generated race comparisons:"
      find web/public/data/comparison/race -name "*.json" 2>/dev/null || echo "No race comparison files found"
      echo ""
      echo "Generated trinket comparisons:"
      find web/public/data/comparison/trinkets -name "*.json" 2>/dev/null || echo "No trinket comparison files found"
    '';
    runtimeInputs = [pkgs.coreutils pkgs.findutils];
  };

  # Convert simulation outputs to nix apps
  simulationApps =
    lib.mapAttrs (name: massSim: {
      type = "app";
      program = "${massSim.script}/bin/${massSim.metadata.output}-aggregator";
    })
    massSimulations;

  raceComparisonApps =
    lib.mapAttrs (name: raceComp: {
      type = "app";
      program = "${raceComp.script}/bin/${raceComp.metadata.output}-aggregator";
    })
    raceComparisons;

  trinketComparisonApps =
    lib.mapAttrs (name: trinketComp: {
      type = "app";
      program = "${trinketComp.script}/bin/${trinketComp.metadata.output}-aggregator";
    })
    trinketComparisons;
in
  # Export all apps
  simulationApps
  // raceComparisonApps
  // trinketComparisonApps
  // {
    allSimulations = {
      type = "app";
      program = "${allSimulationsScript}/bin/all-simulations";
    };
  }

