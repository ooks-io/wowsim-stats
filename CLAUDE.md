# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Nix-based data pipeline for generating World of Warcraft simulation leaderboards and statistics. The project uses the wowsims CLI engine to create a theorycrafting website similar to bloodmallet, showing spec performance rankings and trinket data for WoW Mists of Pandaria.

### Project Vision
- Generate spec simulation leaderboards across different encounter types
- Provide trinket performance data and rankings
- Serve results via static website with JSON data files
- Automate updates via GitHub Actions when upstream wowsims changes
- Use Nix for reproducible builds and server configuration

## Development Environment

### Setup
- Use `nix develop` to enter the development shell which provides `wowsimcli`
- The project uses a Nix flake with inputs from the wowsims MoP fork

### Key Commands
- `nix develop` - Enter development environment with wowsimcli available
- `wowsimcli sim --infile input.json` - Run simulation with input file
- `wowsimcli sim --infile input.json --outfile results.json` - Save results to file
- `nix flake show` - Show available flake outputs
- `nix build` - Build packages (future: website and data generation)

## Architecture

### Core Structure
- `nix/lib/` - Core simulation library functions
- `nix/lib/default.nix` - Main library exports (target, encounter, player, raid)
- `nix/lib/extend.nix` - Extends nixpkgs lib with sim functions
- `docs/` - Example JSON configurations for CLI input

### Key Components

#### Player Configuration (`nix/lib/player.nix`)
- `mkPlayer` function creates player configurations from:
  - Race, class, spec selection
  - Gearsets loaded from wowsims JSON files
  - APL (Action Priority List) rotations
  - Consumables, talents, glyphs, professions
  - Challenge mode support

#### Target/Encounter System (`nix/lib/target.nix`)
- Predefined target types: `defaultRaidBoss`, `smallTrash`, `largeTrash`
- Configurable mob stats, damage, and mechanics

#### Simulation (`nix/lib/simulation.nix`)
- `mkSim` function creates full simulation configurations
- Supports iterations, random seeds, different sim types

### Data Sources
- Gearsets: `${wowsims}/ui/${class}/${spec}/gear_sets/${gearset}.gear.json`
- APLs: `${wowsims}/ui/${class}/${spec}/apls/${apl}.apl.json`
- References external wowsims repository for game data

## File Organization
- Configuration examples in `docs/example_cli_input_windwalker.json`
- Nix library functions organized by domain (player, target, encounter, simulation)
- No traditional package.json or test files - this is a pure Nix project

## Input JSON Schema

The wowsimcli expects complex JSON input files with this hierarchy:

### Top-Level Structure
```json
{
  "requestId": "string",
  "type": "SimTypeIndividual", 
  "raid": RaidConfig,
  "encounter": EncounterConfig,
  "simOptions": SimulationOptions
}
```

### Key Schema Components
- **Raid**: Contains 5 parties (25-person raid), each with 5 player slots, plus raid-wide buffs/debuffs
- **Player**: Complex nested structure with equipment (16 slots), consumables, talents, spec-specific options, and APL rotations
- **Equipment**: Items with IDs, gems, enchants, reforging, tinkers
- **APL (Action Priority Lists)**: Deep conditional logic trees for spell rotation decisions
- **Encounter**: Target configuration, fight duration, execute phases, enemy stats
- **Enumeration Patterns**: Prefixed types like "RaceBloodElf", "ClassMonk", "MobTypeMechanical"

### Critical Details
- Equipment items use game database IDs for items/gems/enchants
- APL system uses complex nested conditions (and/or/cmp operators)
- Stats arrays are fixed-length (22 elements for target stats)
- Empty `{}` objects serve as placeholders for unused slots
- Spec-specific config uses dynamic keys like `"windwalkerMonk": {...}`

## Data Pipeline

### Planned Workflow
1. **Input Generation**: Nix functions create JSON input files matching wowsims schema
2. **Simulation**: wowsimcli processes inputs and generates result data
3. **Aggregation**: Results are collected into static JSON files for website consumption
4. **Website**: Static site serves leaderboards and trinket data
5. **Automation**: GitHub Actions trigger updates when upstream wowsims changes

### Target Simulations
- **Encounter Types**: Single Target, Multi-Target, Short Single Target
- **Scope**: All specs initially, trinket comparisons later
- **Output**: JSON files for web frontend consumption

## Composable Component Design

### Class Component Pattern
The project uses a layered composition approach for building reusable class configurations:

```nix
{
  lib,
  components,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (components.consumables) agilityDps;

  # Spec-specific builder function
  mkWindwalker = {
    race,
    apl,
    gearset, 
    talents,
  }: mkPlayer {
    class = "monk";
    spec = "windwalker";
    inherit race gearset talents apl;
    consumables = agilityDps;  # Shared consumable set
  };

  # Organized configurations by tier and scenario
  windwalker = {
    talents = {
      xuen = "213322";  # Single target build
      rjw = "233321";   # AoE build
    };
    p1 = {
      singleTarget = mkWindwalker {
        race = "orc";
        apl = "default";
        gearset = "dw_p1_bis";
        talents = windwalker.talents.xuen;
      };
      aoeOrc = mkWindwalker {
        race = "orc";
        apl = "default"; 
        gearset = "dw_p1_bis";
        talents = windwalker.talents.rjw;
      };
    };
  };
in windwalker
```

### Design Benefits
- **Shared Defaults**: Common consumables, base configurations
- **Scenario Organization**: Grouped by content tier (p1, p2) and encounter type
- **Talent Variants**: Named talent configurations for different playstyles
- **Race Flexibility**: Easy to create variants for different races
- **Inheritance Chain**: `mkWindwalker` → `mkPlayer` → JSON output

## Working with the Code
- Build reusable Nix components following the layered composition pattern
- Create spec builders that inherit from `mkPlayer` with sensible defaults
- Organize configurations by content tier and encounter scenario
- Keep individual functions small and focused for maintainability
- Test changes by entering `nix develop` and running `wowsimcli` with generated configs
- All simulation data comes from the referenced wowsims fork
- Challenge mode can be enabled on equipment for different stat scaling

## Future Development
- GitHub Actions integration for automated simulation updates
- Website packaging and server configuration via Nix
- Trinket simulation expansion beyond basic spec rankings