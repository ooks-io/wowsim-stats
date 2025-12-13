// index cache system for Discord bot
// loads and caches API indexes for autocomplete and validation

import { API_BASE_URL, CACHE } from "../constants.js";

// simplified types for cached data
export interface CachedDungeon {
  id: number;
  slug: string;
  name: string;
  short_name: string;
}

export interface CachedClass {
  id: number;
  key: string;
  name: string;
  specs: string[];
}

export interface CachedPlayer {
  id: number;
  name: string;
  region: string;
  realm_slug: string;
  realm_name: string;
  class_name: string;
  global_ranking: number;
  global_ranking_bracket: string;
}

export interface CachedRealm {
  slug: string;
  name: string;
  connected_realm_id: number;
  parent_realm?: string | null;
  player_count: number;
}

export interface CachedSeason {
  id: string;
  name: string;
  start_timestamp: string;
  end_timestamp?: string;
  is_current: boolean;
}

interface Cache {
  dungeons: CachedDungeon[];
  classes: CachedClass[];
  players: CachedPlayer[];
  realms: {
    us: CachedRealm[];
    eu: CachedRealm[];
    kr: CachedRealm[];
    tw: CachedRealm[];
  };
  seasons: CachedSeason[];
  lastUpdated: {
    dungeons: number;
    classes: number;
    players: number;
    realms: Record<string, number>;
    seasons: number;
  };
}

// module-level cache
const cache: Cache = {
  dungeons: [],
  classes: [],
  players: [],
  realms: {
    us: [],
    eu: [],
    kr: [],
    tw: [],
  },
  seasons: [],
  lastUpdated: {
    dungeons: 0,
    classes: 0,
    players: 0,
    realms: {},
    seasons: 0,
  },
};

// loads dungeon index from API
async function loadDungeons(): Promise<CachedDungeon[]> {
  const now = Date.now();
  if (
    cache.dungeons.length > 0 &&
    now - cache.lastUpdated.dungeons < CACHE.INDEX_TTL_MS
  ) {
    return cache.dungeons;
  }

  try {
    const url = `${API_BASE_URL}/api/leaderboard/season/2/global/index.json`;
    const response = await fetch(url);
    if (!response.ok)
      throw new Error(`Failed to fetch dungeons: ${response.statusText}`);

    const data = await response.json();
    cache.dungeons = data.data || [];
    cache.lastUpdated.dungeons = now;
    console.log(`Loaded ${cache.dungeons.length} dungeons`);
    return cache.dungeons;
  } catch (error) {
    console.error("Failed to load dungeons:", error);
    return cache.dungeons; // return stale cache if available
  }
}

// loads class index from API
async function loadClasses(): Promise<CachedClass[]> {
  const now = Date.now();
  if (
    cache.classes.length > 0 &&
    now - cache.lastUpdated.classes < CACHE.INDEX_TTL_MS
  ) {
    return cache.classes;
  }

  try {
    const url = `${API_BASE_URL}/api/leaderboard/season/2/players/class/index.json`;
    const response = await fetch(url);
    if (!response.ok)
      throw new Error(`Failed to fetch classes: ${response.statusText}`);

    const data = await response.json();
    cache.classes = data.data || [];
    cache.lastUpdated.classes = now;
    console.log(`Loaded ${cache.classes.length} classes`);
    return cache.classes;
  } catch (error) {
    console.error("Failed to load classes:", error);
    return cache.classes;
  }
}

// loads player search index from API (all shards)
async function loadPlayers(): Promise<CachedPlayer[]> {
  const now = Date.now();
  if (
    cache.players.length > 0 &&
    now - cache.lastUpdated.players < CACHE.PLAYER_SEARCH_TTL_MS
  ) {
    return cache.players;
  }

  try {
    // load all shards (000, 001, 002, ...)
    const shards = ["000", "001", "002"];
    const allPlayers: CachedPlayer[] = [];

    for (const shard of shards) {
      const url = `${API_BASE_URL}/api/search/players-${shard}.json`;
      const response = await fetch(url);
      if (!response.ok) {
        console.warn(
          `Failed to fetch player shard ${shard}: ${response.statusText}`,
        );
        continue;
      }

      const data = await response.json();
      if (data.players) {
        allPlayers.push(...data.players);
      }
    }

    cache.players = allPlayers;
    cache.lastUpdated.players = now;
    console.log(`Loaded ${cache.players.length} players`);
    return cache.players;
  } catch (error) {
    console.error("Failed to load players:", error);
    return cache.players;
  }
}

// loads realm index for a specific region
async function loadRealms(
  region: "us" | "eu" | "kr" | "tw",
): Promise<CachedRealm[]> {
  const now = Date.now();
  const lastUpdate = cache.lastUpdated.realms[region] || 0;

  if (
    cache.realms[region].length > 0 &&
    now - lastUpdate < CACHE.REGION_INDEX_TTL_MS
  ) {
    return cache.realms[region];
  }

  try {
    const url = `${API_BASE_URL}/api/leaderboard/season/2/${region}/index.json`;
    const response = await fetch(url);
    if (!response.ok)
      throw new Error(
        `Failed to fetch ${region} realms: ${response.statusText}`,
      );

    const data = await response.json();
    cache.realms[region] = data.data || [];
    cache.lastUpdated.realms[region] = now;
    console.log(
      `Loaded ${cache.realms[region].length} ${region.toUpperCase()} realms`,
    );
    return cache.realms[region];
  } catch (error) {
    console.error(`Failed to load ${region} realms:`, error);
    return cache.realms[region];
  }
}

// loads season index from API
async function loadSeasons(): Promise<CachedSeason[]> {
  const now = Date.now();
  if (
    cache.seasons.length > 0 &&
    now - cache.lastUpdated.seasons < CACHE.INDEX_TTL_MS
  ) {
    return cache.seasons;
  }

  try {
    const url = `${API_BASE_URL}/api/leaderboard/season/index.json`;
    const response = await fetch(url);
    if (!response.ok)
      throw new Error(`Failed to fetch seasons: ${response.statusText}`);

    const data = await response.json();
    cache.seasons = data.data || [];
    cache.lastUpdated.seasons = now;
    console.log(`Loaded ${cache.seasons.length} seasons`);
    return cache.seasons;
  } catch (error) {
    console.error("Failed to load seasons:", error);
    return cache.seasons;
  }
}

// initializes all caches (call on serverless function cold start)
// only loads lightweight indexes - player search loads on-demand
export async function initializeCaches(): Promise<void> {
  console.log("Initializing Discord bot caches...");
  try {
    await Promise.all([
      loadDungeons(),
      loadClasses(),
      // loadPlayers() removed - loads on-demand for /player autocomplete only
      loadSeasons(),
    ]);
    console.log("Discord bot caches initialized successfully");
  } catch (error) {
    console.error("Failed to initialize caches:", error);
  }
}

// gets cached dungeons (loads if not cached)
export async function getDungeons(): Promise<CachedDungeon[]> {
  return loadDungeons();
}

/**
 * gets cached classes (loads if not cached)
 */
export async function getClasses(): Promise<CachedClass[]> {
  return loadClasses();
}

// gets cached players (loads if not cached)
export async function getPlayers(): Promise<CachedPlayer[]> {
  return loadPlayers();
}

// gets cached realms for a region (loads if not cached)
export async function getRealms(
  region: "us" | "eu" | "kr" | "tw",
): Promise<CachedRealm[]> {
  return loadRealms(region);
}

// gets cached seasons (loads if not cached)
export async function getSeasons(): Promise<CachedSeason[]> {
  return loadSeasons();
}

// clears all caches (useful for testing or manual refresh)
export function clearCaches(): void {
  cache.dungeons = [];
  cache.classes = [];
  cache.players = [];
  cache.realms = { us: [], eu: [], kr: [], tw: [] };
  cache.seasons = [];
  cache.lastUpdated = {
    dungeons: 0,
    classes: 0,
    players: 0,
    realms: {},
    seasons: 0,
  };
  console.log("Caches cleared");
}
