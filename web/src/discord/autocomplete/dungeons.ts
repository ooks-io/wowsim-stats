// Dungeon autocomplete handler

import { getDungeons } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * Handles autocomplete for dungeon selection
 * Fuzzy matches on dungeon name, short name, or slug
 */
export async function autocompleteDungeon(
	query: string,
): Promise<AutocompleteChoice[]> {
	const dungeons = await getDungeons();
	const lowerQuery = query.toLowerCase().trim();

	// if no query, return all dungeons
	if (!lowerQuery) {
		return dungeons.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES).map(
			(d) => ({
				name: `${d.name} (${d.short_name})`,
				value: d.slug,
			}),
		);
	}

	// filter dungeons by name, short name, or slug
	const filtered = dungeons
		.filter(
			(d) =>
				d.name.toLowerCase().includes(lowerQuery) ||
				d.short_name.toLowerCase().includes(lowerQuery) ||
				d.slug.toLowerCase().includes(lowerQuery),
		)
		.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES);

	return filtered.map((d) => ({
		name: `${d.name} (${d.short_name})`,
		value: d.slug,
	}));
}
