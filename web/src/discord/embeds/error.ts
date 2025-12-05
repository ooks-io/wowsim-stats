// error embed formatter

import type { Embed } from "../types.js";

// creates an error embed for display in Discord
export function createErrorEmbed(title: string, description: string): Embed {
  return {
    title: `❌ ${title}`,
    description,
    footer: {
      text: "wowsimstats.com",
    },
    timestamp: new Date().toISOString(),
  };
}

// creates a player not found error embed
export function createPlayerNotFoundEmbed(
  name: string,
  realm: string,
  region: string,
): Embed {
  return createErrorEmbed(
    "Player Not Found",
    `Could not find player **${name}** on **${realm}** (${region.toUpperCase()}).\n\nMake sure:\n• The player name is spelled correctly\n• The player has completed at least one Challenge Mode dungeon\n• The realm is correct`,
  );
}

// creates an API error embed
export function createAPIErrorEmbed(error?: string): Embed {
  return createErrorEmbed(
    "API Error",
    error || "Failed to fetch data from the API. Please try again in a moment.",
  );
}

// creates an invalid input error embed
export function createInvalidInputEmbed(field: string, value: string): Embed {
  return createErrorEmbed(
    "Invalid Input",
    `Invalid ${field}: **${value}**\n\nPlease use the autocomplete suggestions or check your input.`,
  );
}

// creates a generic error embed
export function createGenericErrorEmbed(): Embed {
  return createErrorEmbed(
    "Something Went Wrong",
    "An unexpected error occurred. Please try again later.",
  );
}
