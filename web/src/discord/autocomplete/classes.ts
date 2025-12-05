// Class autocomplete handler

import { getClasses } from "../indexes/cache.js";
import type { AutocompleteChoice } from "../types.js";
import { DISCORD_LIMITS } from "../constants.js";

/**
 * Handles autocomplete for class selection
 * Fuzzy matches on class name or key
 */
export async function autocompleteClass(
	query: string,
): Promise<AutocompleteChoice[]> {
	const classes = await getClasses();
	const lowerQuery = query.toLowerCase().trim();

	// if no query, return all classes
	if (!lowerQuery) {
		return classes.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES).map(
			(c) => ({
				name: `${c.name} (${c.specs.join(", ")})`,
				value: c.key,
			}),
		);
	}

	// filter classes by name or key
	const filtered = classes
		.filter(
			(c) =>
				c.name.toLowerCase().includes(lowerQuery) ||
				c.key.toLowerCase().includes(lowerQuery),
		)
		.slice(0, DISCORD_LIMITS.MAX_AUTOCOMPLETE_CHOICES);

	return filtered.map((c) => ({
		name: `${c.name} (${c.specs.join(", ")})`,
		value: c.key,
	}));
}
