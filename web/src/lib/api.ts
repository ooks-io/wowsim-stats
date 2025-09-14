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
async function apiRequest<T>(url: string): Promise<T> {
  const fullUrl = url.startsWith("http") ? url : `${API_BASE}${url}`;
  const response = await fetch(fullUrl);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(
      `API request failed: ${response.status} ${response.statusText} - ${errorText}`,
    );
  }

  return response.json();
}

// leaderboard api functions
export async function fetchGlobalLeaderboard(
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath("global", "", dungeonId, page)}`;
  console.log("Fetching global leaderboard:", url);
  return apiRequest<LeaderboardData>(url);
}

export async function fetchRegionalLeaderboard(
  region: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath(region, "all", dungeonId, page)}`;
  console.log("Fetching regional leaderboard:", url);
  return apiRequest<LeaderboardData>(url);
}

export async function fetchRealmLeaderboard(
  region: string,
  realmSlug: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
): Promise<LeaderboardData> {
  const url = `${API_BASE}${buildStaticLeaderboardPath(region, realmSlug, dungeonId, page)}`;
  console.log("Fetching realm leaderboard:", url);
  return apiRequest<LeaderboardData>(url);
}

export async function fetchPlayerLeaderboard(
  scope: string = "global",
  region?: string,
  page: number = 1,
  pageSize: number = 25,
): Promise<any> {
  const url = `${API_BASE}${buildStaticPlayerLeaderboardPath(scope, region, page)}`;
  console.log("Fetching player leaderboard:", url);
  return apiRequest(url);
}

// player profile API functions
export async function fetchPlayerProfile(
  region: string,
  realmSlug: string,
  playerName: string,
): Promise<PlayerProfileData> {
  const url = `${API_BASE}${buildStaticPlayerProfilePath(region, realmSlug, playerName)}`;
  console.log("Fetching player profile:", url);
  return apiRequest<PlayerProfileData>(url);
}

// fetchPlayerBestRuns removed - best runs are now included in fetchPlayerProfile response

// leaderboard router function - determines which API to call based on filters
export async function fetchLeaderboard(
  region: string,
  realm: string,
  dungeonId: number,
  page: number = 1,
  teamFilter: boolean = true,
): Promise<LeaderboardData> {
  if (region === "global") {
    return fetchGlobalLeaderboard(dungeonId, page, teamFilter);
  } else if (realm === "all") {
    return fetchRegionalLeaderboard(region, dungeonId, page, teamFilter);
  } else {
    return fetchRealmLeaderboard(region, realm, dungeonId, page, teamFilter);
  }
}

// url helpers are centralized in lib/utils.ts and re-exported here for convenience
export { buildLeaderboardURL, buildPlayerProfileURL };
