{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  trinket,
  ...
}: let
  config = import ./config.nix {inherit encounter;};

  # Utility functions
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

  # Function to generate mass simulations with pkgs
  generateMassSimulations = pkgs: let
    massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
    inherit (massSimFunctions) mkMassSim;
  in
    lib.listToAttrs (lib.flatten (map (
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

  # Function to generate race comparisons with pkgs
  generateRaceComparisons = pkgs: let
    massSimFunctions = import ./mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;};
    inherit (massSimFunctions) mkRaceComparison;
  in
    lib.listToAttrs (lib.flatten (map (
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

  # Function to generate trinket comparisons with pkgs
  generateTrinketComparisons = pkgs: let
    trinketComparison = import ./mkTrinketComparison.nix {inherit lib pkgs classes encounter buffs debuffs inputs trinket;};
    inherit (trinketComparison) mkTrinketComparison;
  in
    lib.listToAttrs (lib.flatten (map (
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

  # Function to generate all simulations script with pkgs
  generateAllSimulationsScript = pkgs: let
    massSimulations = generateMassSimulations pkgs;
    raceComparisons = generateRaceComparisons pkgs;
    trinketComparisons = generateTrinketComparisons pkgs;
  in
    pkgs.writeShellApplication {
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

  simulation = {
    # Generation functions that take pkgs as parameter
    inherit generateMassSimulations generateRaceComparisons generateTrinketComparisons generateAllSimulationsScript;
    
    # Configuration and utilities
    inherit config generateScenarios scenarios;
    
    # Helper functions
    inherit makeMassSimName;
  };
in {
  flake.simulation = simulation;
  _module.args.simulation = simulation;
}
