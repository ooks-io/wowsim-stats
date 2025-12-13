// Player command handler

import type {
  DiscordInteraction,
  InteractionResponseData,
  PlayerCommandOptions,
} from "../types.js";
import { fetchPlayerProfile } from "../../lib/api.js";
import { API_BASE_URL, DEFAULT_SEASON_ID } from "../constants.js";
import {
  createPlayerProfileEmbed,
  createViewProfileButton,
} from "../embeds/player-profile.js";
import {
  createPlayerNotFoundEmbed,
  createAPIErrorEmbed,
  createInvalidInputEmbed,
} from "../embeds/error.js";

/**
 * Handles the /player command
 * Usage: /player <name> <region> <realm>
 */
export async function handlePlayerCommand(
  interaction: DiscordInteraction,
): Promise<InteractionResponseData> {
  const options = parsePlayerOptions(interaction);

  // validate options
  if (!options.name || !options.region || !options.realm) {
    return {
      embeds: [
        createInvalidInputEmbed(
          "command options",
          "name, region, and realm are required",
        ),
      ],
      flags: 64, // EPHEMERAL
    };
  }

  // normalize inputs
  const name = options.name.trim();
  const region = options.region.toLowerCase();
  const realm = options.realm.toLowerCase();
  const season = options.season || DEFAULT_SEASON_ID;

  // validate region
  if (!["us", "eu", "kr", "tw"].includes(region)) {
    return {
      embeds: [createInvalidInputEmbed("region", options.region)],
      flags: 64, // EPHEMERAL
    };
  }

  try {
    // fetch player profile from API
    const profile = await fetchPlayerProfile(region, realm, name, API_BASE_URL);

    if (!profile || !profile.player) {
      return {
        embeds: [createPlayerNotFoundEmbed(name, realm, region)],
        flags: 64, // EPHEMERAL
      };
    }

    // create embed and button
    const embed = createPlayerProfileEmbed(profile, season);
    const buttons = createViewProfileButton(name, realm, region, season);

    return {
      embeds: [embed],
      components: buttons,
    };
  } catch (error) {
    console.error("Error fetching player profile:", error);

    // check if it's a 404
    if (error instanceof Error && error.message.includes("404")) {
      return {
        embeds: [createPlayerNotFoundEmbed(name, realm, region)],
        flags: 64, // EPHEMERAL
      };
    }

    return {
      embeds: [createAPIErrorEmbed()],
      flags: 64, // EPHEMERAL
    };
  }
}

/**
 * Parses player command options from interaction
 */
function parsePlayerOptions(
  interaction: DiscordInteraction,
): PlayerCommandOptions {
  const options = interaction.data?.options || [];

  return {
    name: getOptionValue(options, "name") as string,
    region: getOptionValue(options, "region") as "us" | "eu" | "kr" | "tw",
    realm: getOptionValue(options, "realm") as string,
    season: getOptionValue(options, "season") as string | undefined,
  };
}

/**
 * Helper to get option value by name
 */
function getOptionValue(
  options: Array<{ name: string; value?: string | number | boolean }>,
  name: string,
): string | number | boolean | undefined {
  return options.find((opt) => opt.name === name)?.value;
}
