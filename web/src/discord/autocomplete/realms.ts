// Realm autocomplete handler

import { getRealms, getPlayers } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * Handles autocomplete for realm selection
 * Filters by region and fuzzy matches on realm name
 * Optionally filters by player name if provided
 */
export async function autocompleteRealm(
  query: string,
  region?: "us" | "eu" | "kr" | "tw",
  playerName?: string,
): Promise<AutocompleteChoice[]> {
  // if no region specified, return empty (user must select region first)
  if (!region) {
    return [
      {
        name: "Please select a region first",
        value: "none",
      },
    ];
  }

  let realms = await getRealms(region);
  const lowerQuery = query.toLowerCase().trim();

  // if player name is provided, filter realms to only those where player exists
  if (playerName && playerName.trim() !== "") {
    const players = await getPlayers();
    const lowerPlayerName = playerName.toLowerCase().trim();

    // find all players matching the name in this region
    const matchingPlayers = players.filter(
      (p) =>
        p.name.toLowerCase().includes(lowerPlayerName) &&
        p.region === region.toLowerCase(),
    );

    // extract unique realm slugs from matching players
    const playerRealms = new Set(matchingPlayers.map((p) => p.realm_slug));

    // filter realms to only those where the player exists
    realms = realms.filter((r) => playerRealms.has(r.slug));

    // if no realms found, show helpful message
    if (realms.length === 0) {
      return [
        {
          name: `No players named "${playerName}" found in ${region.toUpperCase()}`,
          value: "none",
        },
      ];
    }
  }

  // sort by player count (most popular first)
  const sortedRealms = [...realms].sort(
    (a, b) => b.player_count - a.player_count,
  );

  // if no query, return top realms
  if (!lowerQuery) {
    return sortedRealms
      .slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES)
      .map((r) => ({
        name: r.name,
        value: r.slug,
      }));
  }

  // filter realms by name or slug
  const filtered = sortedRealms
    .filter(
      (r) =>
        r.name.toLowerCase().includes(lowerQuery) ||
        r.slug.toLowerCase().includes(lowerQuery),
    )
    .slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES);

  return filtered.map((r) => ({
    name: r.name,
    value: r.slug,
  }));
}
