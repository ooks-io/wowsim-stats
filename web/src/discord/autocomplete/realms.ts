// Realm autocomplete handler

import { getRealms } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * Handles autocomplete for realm selection
 * Filters by region and fuzzy matches on realm name
 */
export async function autocompleteRealm(
	query: string,
	region?: "us" | "eu" | "kr" | "tw",
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

	const realms = await getRealms(region);
	const lowerQuery = query.toLowerCase().trim();

	// sort by player count (most popular first)
	const sortedRealms = [...realms].sort(
		(a, b) => b.player_count - a.player_count,
	);

	// if no query, return top realms
	if (!lowerQuery) {
		return sortedRealms
			.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES)
			.map((r) => ({
				name: `${r.name} (${r.player_count.toLocaleString()} players)`,
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
		name: `${r.name} (${r.player_count.toLocaleString()} players)`,
		value: r.slug,
	}));
}
