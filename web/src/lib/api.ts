import type { LeaderboardData, PlayerProfileData, BestRun } from "./types";
import {
  buildLeaderboardURL,
  buildPlayerProfileURL,
  buildStaticLeaderboardPath,
  buildStaticPlayerLeaderboardPath,
  buildStaticPlayerProfilePath,
} from "./utils";

// api base configuration (always use relative paths for Netlify/SSR)
const API_BASE = "";

// generic api request handler with error handling
async function apiRequest<T>(url: string, origin?: string): Promise<T> {
  let fullUrl: string;
  if (url.startsWith("http")) {
    fullUrl = url;
  } else if (origin) {
    // SSR context - need full URL
    fullUrl = `${origin}${url}`;
  } else {
    // Client-side context - relative URL works
    fullUrl = `${API_BASE}${url}`;
  }
  const response = await fetch(fullUrl);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Player not found");
    }

    throw new Error(
      `Failed to load data: ${response.status} ${response.statusText}`,
    );
  }

  return response.json();
}

// leaderboard api functions
export async function fetchGlobalLeaderboard(
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
  origin?: string,
  seasonId: number = 1,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath("global", "", dungeonId, page, seasonId)}`;
  console.log("Fetching global leaderboard:", url);
  return apiRequest<LeaderboardData>(url, origin);
}

export async function fetchRegionalLeaderboard(
  region: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
  origin?: string,
  seasonId: number = 1,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath(region, "all", dungeonId, page, seasonId)}`;
  console.log("Fetching regional leaderboard:", url);
  return apiRequest<LeaderboardData>(url, origin);
}

export async function fetchRealmLeaderboard(
  region: string,
  realmSlug: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
  origin?: string,
  seasonId: number = 1,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath(region, realmSlug, dungeonId, page, seasonId)}`;
  console.log("Fetching realm leaderboard:", url);
  return apiRequest<LeaderboardData>(url, origin);
}

export async function fetchPlayerLeaderboard(
  scope: "global" | "regional" | "realm" = "global",
  region?: string,
  page: number = 1,
  pageSize: number = 25,
  opts?: { realmSlug?: string; classKey?: string; seasonId?: number },
  origin?: string,
): Promise<any> {
  const url = `${API_BASE}${buildStaticPlayerLeaderboardPath(scope, region, page, opts)}`;
  console.log("Fetching player leaderboard:", url);
  return apiRequest(url, origin);
}

// player profile API functions
export async function fetchPlayerProfile(
  region: string,
  realmSlug: string,
  playerName: string,
  origin?: string,
): Promise<PlayerProfileData> {
  const url = `${API_BASE}${buildStaticPlayerProfilePath(region, realmSlug, playerName)}`;
  console.log("Fetching player profile:", url);
  return apiRequest<PlayerProfileData>(url, origin);
}

// fetchPlayerBestRuns removed - best runs are now included in fetchPlayerProfile response

// leaderboard router function - determines which API to call based on filters
export async function fetchLeaderboard(
  region: string,
  realm: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
  origin?: string,
  seasonId: number = 1,
): Promise<LeaderboardData> {
  if (region === "global") {
    return fetchGlobalLeaderboard(dungeonId, page, teamFilter, origin, seasonId);
  } else if (realm === "all") {
    return fetchRegionalLeaderboard(region, dungeonId, page, teamFilter, origin, seasonId);
  } else {
    return fetchRealmLeaderboard(region, realm, dungeonId, page, teamFilter, origin, seasonId);
  }
}

// url helpers are centralized in lib/utils.ts and re-exported here for convenience
export { buildLeaderboardURL, buildPlayerProfileURL };
