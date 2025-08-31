# Player Profiles Data Structure

## File Organization

```
/players/
├── README.md                              # This file
├── index.json                             # Player directory/search index
├── by-realm/
│   ├── arugal/
│   │   ├── girthquake.json               # Individual player profile
│   │   ├── blursw.json
│   │   └── yorty.json
│   ├── gehennas/
│   │   ├── copypastt.json
│   │   ├── trubble.json
│   │   └── tradias.json
│   └── venoxis/
│       └── tradias.json
└── by-region/
    ├── us.json                           # All US players index
    └── eu.json                           # All EU players index
```

## File Size Analysis

**Individual Player Profile**: ~5.4KB (based on Girthquake example)

- Basic info + API data: ~0.5KB
- Challenge mode stats: ~0.3KB
- Best runs (9 dungeons): ~3.5KB
- Team history + teammates: ~1.1KB

**Estimated Repository Impact**:

- 800 players × 5.4KB = ~4.3MB total
- Index files: ~200KB
- **Total addition**: ~4.5MB

## Player Profile Schema

See `girthquake-arugal.json` for complete example structure.

### Key Sections:

1. **player_info**: Basic character data from WoW API
2. **challenge_mode_stats**: Performance summary statistics
3. **best_runs_per_dungeon**: Detailed best run data with rankings
4. **team_history**: Teams the player has been part of
5. **frequent_teammates**: Regular partners and collaboration rates

## API Integration Requirements

To generate complete profiles, we need:

1. WoW Character API calls for guild/race/class/level data
2. Rate limiting (1000 requests/hour typical)
3. Caching strategy for API responses
4. Error handling for deleted/transferred characters

## Implementation Strategy

1. **Phase 1**: Generate profiles for extended roster players only (~800 players)
2. **Phase 2**: Add API data integration
3. **Phase 3**: Expand to all qualifying players (>5 runs)
