// region autocomplete handler

import { getPlayers } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * handles autocomplete for region selection based on player name
 * filters regions to only show those where the specified player exists
 */
export async function autocompleteRegionByPlayer(
  query: string,
  playerName?: string,
): Promise<AutocompleteChoice[]> {
  // Add debug logging
  console.log("[Region Autocomplete] playerName:", playerName, "query:", query);

  // if no player name provided, return all regions
  if (!playerName || playerName.trim() === "") {
    console.log("[Region Autocomplete] No player name, returning all regions");
    return [
      { name: "US", value: "us" },
      { name: "EU", value: "eu" },
      { name: "KR", value: "kr" },
      { name: "TW", value: "tw" },
    ];
  }

  const players = await getPlayers();
  const lowerPlayerName = playerName.toLowerCase().trim();

  // find all players matching the name (exact match first, then partial)
  let matchingPlayers = players.filter(
    (p) => p.name.toLowerCase() === lowerPlayerName,
  );

  // if no exact matches, try partial match
  if (matchingPlayers.length === 0) {
    matchingPlayers = players.filter((p) =>
      p.name.toLowerCase().includes(lowerPlayerName),
    );
  }

  // extract unique regions from matching players
  const regions = new Set<string>();
  for (const player of matchingPlayers) {
    regions.add(player.region);
  }

  // convert to autocomplete choices
  const regionMap: Record<string, string> = {
    us: "US",
    eu: "EU",
    kr: "KR",
    tw: "TW",
  };

  const choices: AutocompleteChoice[] = [];
  for (const region of Array.from(regions).sort()) {
    choices.push({
      name: regionMap[region] || region.toUpperCase(),
      value: region,
    });
  }

  // if no regions found, show helpful message
  if (choices.length === 0) {
    return [
      {
        name: `No players found matching "${playerName}"`,
        value: "none",
      },
    ];
  }

  return choices;
}
