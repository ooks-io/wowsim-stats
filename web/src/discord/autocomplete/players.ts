// Player autocomplete handler

import { getPlayers } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * Handles autocomplete for player search
 * Searches through all 9,935 players with optional filters
 */
export async function autocompletePlayer(
	query: string,
	region?: string,
	realm?: string,
	className?: string,
): Promise<AutocompleteChoice[]> {
	const players = await getPlayers();
	const lowerQuery = query.toLowerCase().trim();

	// if no query and no filters, return top ranked players
	if (!lowerQuery && !region && !realm && !className) {
		return players.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES).map(
			(p) => ({
				name: p.name,
				value: p.name,
			}),
		);
	}

	// filter players
	let filtered = players;

	// apply region filter
	if (region) {
		filtered = filtered.filter((p) => p.region === region.toLowerCase());
	}

	// apply realm filter
	if (realm) {
		filtered = filtered.filter((p) => p.realm_slug === realm.toLowerCase());
	}

	// apply class filter
	if (className) {
		filtered = filtered.filter(
			(p) => p.class_name.toLowerCase() === className.toLowerCase(),
		);
	}

	// apply query filter (fuzzy match on name)
	if (lowerQuery) {
		filtered = filtered.filter((p) =>
			p.name.toLowerCase().includes(lowerQuery),
		);
	}

	// sort by ranking (best players first) and limit
	const results = filtered
		.sort((a, b) => a.global_ranking - b.global_ranking)
		.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES);

	return results.map((p) => ({
		name: p.name,
		value: p.name,
	}));
}
