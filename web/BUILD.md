# WoW Challenge Mode Stats - System Overview

## What This System Does

This is a comprehensive **Challenge Mode leaderboard tracking system** for World of Warcraft Mists of Pandaria Classic. It automatically fetches leaderboard data from the Blizzard API, processes player statistics, and generates a fast SPA website for browsing rankings.

**Key Features:**
- 🏆 **Live Leaderboards**: Dungeon leaderboards by region, realm, and globally
- 👤 **Player Profiles**: Individual player stats with equipment and best runs  
- 📊 **Rankings**: Global, regional, and realm rankings with percentile brackets
- 🔍 **Search**: Fast player search across all tracked players
- ⚡ **Performance**: SPA architecture with static JSON files for instant navigation

## System Architecture

The system consists of two main components:

1. **Backend Data Pipeline** (`ookstats` Go program)
   - Fetches data from Blizzard API
   - Processes and aggregates statistics in local SQLite database
   - Exports to static JSON files

2. **Frontend Website** (Astro SPA)
   - Serves static JSON files  
   - Client-side routing for instant navigation
   - Dynamic loading of leaderboard data

**Data Flow:**
```
Blizzard API → Local Database → JSON Files → SPA Website
```

## Database Schema Overview

The system uses SQLite/libSQL with these core tables:

**Core Data:**
- `players` - Player names and realm associations
- `realms` - WoW realm information (region, name, connected realms)
- `dungeons` - Challenge mode dungeon reference data
- `challenge_runs` - Individual dungeon completion records
- `run_members` - Players in each run with spec information

**Computed Data:**
- `player_profiles` - Aggregated player statistics and rankings
- `player_details` - Player character data (class, spec, avatar, equipment)
- `player_best_runs` - Best run per player per dungeon
- `player_equipment` - Equipment snapshots with gems and enchants

**Reference Data:**
- `items` - Item database from WoW Sims for tooltips and icons

## Complete Data Pipeline

### 1. Database Setup & Population

```bash
# Create and populate reference data
ookstats populate items --wowsims-db /path/to/wowsims/database.bin
```

**What this does:**
- Creates the SQLite database schema
- Populates the `items` table with equipment data from WoW Sims
- Provides item names, icons, and stats for equipment tooltips

### 2. Fetch Live Data from Blizzard API

```bash
# Fetch challenge mode leaderboards
ookstats fetch cm [--fallback-depth 5] [--verbose]
```

**What this does:**
- Connects to Blizzard API with OAuth2
- Fetches leaderboards for all realms and dungeons
- Stores runs in `challenge_runs` and players in `players`
- Links team members in `run_members`
- Fetches player profile data (class, spec, avatar URLs)

**Environment Variables Required:**
```bash
BLIZZARD_CLIENT_ID=your_client_id
BLIZZARD_CLIENT_SECRET=your_client_secret
```

### 3. Process and Aggregate Statistics

```bash
# Aggregate player statistics and compute rankings
ookstats process players
```

**What this does:**
- Computes best runs per player per dungeon
- Calculates combined best times across all dungeons
- Generates global, regional, and realm rankings
- Creates percentile brackets (artifact, legendary, epic, etc.)
- Identifies players with "complete coverage" (runs in all 9 dungeons)

### 4. Generate Static API Files

```bash
# Export data to JSON files for the website
ookstats generate api --out web/public
```

**What this creates:**
- `api/leaderboard/global/{dungeon}/{page}.json` - Global leaderboards
- `api/leaderboard/{region}/{realm}/{dungeon}/{page}.json` - Realm leaderboards  
- `api/leaderboard/players/global/{page}.json` - Global player rankings
- `api/player/{region}/{realm}/{name}.json` - Individual player profiles
- `api/search/players-{shard}.json` - Search index files

### 5. Build and Deploy Website

```bash
# Build the Astro SPA
cd web
npm install
npm run build
```

**Result:** Static website with server-side rendering for dynamic routes, ready for Netlify deployment.

## Development Workflow

### Initial Setup
```bash
# 1. Setup database and populate items
ookstats populate items --wowsims-db /path/to/wowsims.bin

# 2. Fetch initial data (this takes ~30 minutes for all realms)
ookstats fetch cm

# 3. Process player statistics  
ookstats process players

# 4. Generate API files
ookstats generate api --out web/public

# 5. Start development server
cd web && npm run dev
```

### Regular Updates
```bash
# Quick incremental update (much faster)
ookstats fetch cm --fallback-depth 3
ookstats process players  
ookstats generate api --out web/public
```

## Configuration

**Database Location:**
- Default: `local.db` in current directory
- Override with `--db-file path/to/db` or `OOKSTATS_DB` environment variable
- Supports both local SQLite and remote Turso connections

**Blizzard API:**
- Requires API credentials from Blizzard Developer Portal
- Set `BLIZZARD_CLIENT_ID` and `BLIZZARD_CLIENT_SECRET`

**Performance Tuning:**
- Use `--fallback-depth N` to limit API calls during incremental updates
- Use `--verbose` to see detailed API call information
- Database is optimized for read queries with proper indexes

---

# Build Process (SPA Architecture)

This website uses a **pure SPA (Single Page Application)** architecture with static JSON files for optimal performance and simpler deployment.

## Architecture Overview

- **Backend**: Go program (`ookstats`) generates static JSON API files
- **Frontend**: Astro in server mode with client-side routing (SPA behavior)
- **Data Flow**: JSON files → Client-side fetching → Dynamic UI updates
- **Deployment**: Netlify with server-side rendering for dynamic routes

## Build Process

### 1. Generate Static API Files
First, generate the JSON API files using the Go tool:

```bash
# From the project root
ookstats generate api --out web/public
```

This creates static JSON files served at `/api/...`:
- `web/public/api/leaderboard/global/{dungeon-slug}/{page}.json`
- `web/public/api/leaderboard/{region}/{realm}/{dungeon-slug}/{page}.json`  
- `web/public/api/leaderboard/players/global/{page}.json`
- `web/public/api/leaderboard/players/regional/{region}/{page}.json`
- `web/public/api/player/{region}/{realm}/{player-name}.json`
- `web/public/api/search/players-{shard}.json`

### 2. Build the Website
Then build the Astro site:

```bash
cd web
npm install
npm run build
```

### 3. Full Build Command
For convenience:

```bash
# Complete build pipeline
ookstats generate api --out web/public && cd web && npm run build
```

## Current Architecture

✅ **Astro Config**: `output: "server"` with `@astrojs/netlify` adapter  
✅ **Dynamic Routes**: `prerender: false` for SPA behavior on challenge-mode and player pages  
✅ **Static Routes**: `prerender: true` for homepage and other static content  
✅ **Loading States**: Built-in loading components for smooth UX  
✅ **Error Handling**: User-friendly error messages for missing data  
✅ **Client-Side Routing**: Instant navigation between leaderboards  
✅ **Player Avatars**: Avatar URLs included in JSON generation

## Development

For development:

```bash
# Generate API files first (required)
ookstats generate api --out web/public

# Start dev server
cd web
npm run dev
```

## Benefits of SPA Architecture

- **Fast Navigation**: No full page reloads between views
- **Better UX**: Loading states and smooth transitions  
- **Scalable Builds**: Build time stays constant as data grows
- **Simple Pipeline**: Generate JSON → Build → Deploy
- **Flexible Caching**: JSON files cached independently of HTML
