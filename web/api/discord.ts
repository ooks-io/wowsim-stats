// discord bot interaction endpoint

import type { VercelRequest, VercelResponse } from "@vercel/node";
import {
  InteractionType,
  InteractionResponseType,
  type DiscordInteraction,
} from "../src/discord/types.js";
import {
  verifyDiscordSignature,
  extractDiscordHeaders,
} from "../src/discord/verify.js";
import { initializeCaches } from "../src/discord/indexes/cache.js";
import { handlePlayerCommand } from "../src/discord/commands/player.js";
import { handleLeaderboardCommand } from "../src/discord/commands/leaderboard.js";
import { autocompleteDungeon } from "../src/discord/autocomplete/dungeons.js";
import { autocompleteRealm } from "../src/discord/autocomplete/realms.js";
import { autocompletePlayer } from "../src/discord/autocomplete/players.js";
import { autocompleteClass } from "../src/discord/autocomplete/classes.js";
import { createGenericErrorEmbed } from "../src/discord/embeds/error.js";

// cache initialization flag
let cacheInitialized = false;

/**
 * Main Discord interaction handler
 */
export default async function handler(
  req: VercelRequest,
  res: VercelResponse,
): Promise<void> {
  // only accept POST requests
  if (req.method !== "POST") {
    res.status(405).json({ error: "Method not allowed" });
    return;
  }

  // verify Discord signature
  const publicKey = process.env.DISCORD_PUBLIC_KEY;
  if (!publicKey) {
    console.error("DISCORD_PUBLIC_KEY not configured");
    res.status(500).json({ error: "Bot not configured" });
    return;
  }

  const headers = extractDiscordHeaders(req.headers);
  if (!headers) {
    res.status(401).json({ error: "Missing Discord headers" });
    return;
  }

  // get raw body for verification
  const rawBody = JSON.stringify(req.body);
  const isValid = await verifyDiscordSignature(
    headers.signature,
    headers.timestamp,
    rawBody,
    publicKey,
  );

  if (!isValid) {
    console.error("Invalid Discord signature");
    res.status(401).json({ error: "Invalid signature" });
    return;
  }

  // initialize caches on first request (cold start)
  if (!cacheInitialized) {
    console.log("Cold start - initializing caches...");
    await initializeCaches();
    cacheInitialized = true;
  }

  const interaction = req.body as DiscordInteraction;

  try {
    // handle PING (Discord verification)
    if (interaction.type === InteractionType.PING) {
      res.json({ type: InteractionResponseType.PONG });
      return;
    }

    // handle AUTOCOMPLETE
    if (interaction.type === InteractionType.APPLICATION_COMMAND_AUTOCOMPLETE) {
      const choices = await handleAutocomplete(interaction);
      res.json({
        type: InteractionResponseType.APPLICATION_COMMAND_AUTOCOMPLETE_RESULT,
        data: { choices },
      });
      return;
    }

    // handle COMMAND
    if (interaction.type === InteractionType.APPLICATION_COMMAND) {
      const response = await handleCommand(interaction);
      res.json({
        type: InteractionResponseType.CHANNEL_MESSAGE_WITH_SOURCE,
        data: response,
      });
      return;
    }

    // handle BUTTON (pagination)
    if (interaction.type === InteractionType.MESSAGE_COMPONENT) {
      const response = await handleButton(interaction);
      res.json({
        type: InteractionResponseType.UPDATE_MESSAGE,
        data: response,
      });
      return;
    }

    // unknown interaction type
    console.warn("Unknown interaction type:", interaction.type);
    res.status(400).json({ error: "Unknown interaction type" });
  } catch (error) {
    console.error("Error handling interaction:", error);
    res.json({
      type: InteractionResponseType.CHANNEL_MESSAGE_WITH_SOURCE,
      data: {
        embeds: [createGenericErrorEmbed()],
      },
    });
  }
}

/**
 * Routes autocomplete to appropriate handler
 */
async function handleAutocomplete(interaction: DiscordInteraction) {
  const commandName = interaction.data?.name;
  const options = interaction.data?.options || [];

  // find focused option
  const focusedOption = findFocusedOption(options);
  if (!focusedOption) {
    return [];
  }

  const query = String(focusedOption.value || "");

  // route based on command and option name
  if (commandName === "leaderboard") {
    if (focusedOption.name === "dungeon") {
      return autocompleteDungeon(query);
    }
    if (focusedOption.name === "realm") {
      const region = getOptionValue(options[0]?.options || [], "region");
      const regionStr = typeof region === "string" ? region : undefined;
      return autocompleteRealm(query, regionStr as "us" | "eu" | "kr" | "tw" | undefined);
    }
    if (focusedOption.name === "class") {
      return autocompleteClass(query);
    }
  }

  if (commandName === "player") {
    if (focusedOption.name === "name") {
      const region = getOptionValue(options, "region");
      const realm = getOptionValue(options, "realm");
      return autocompletePlayer(
        query,
        typeof region === "string" ? region : undefined,
        typeof realm === "string" ? realm : undefined,
      );
    }
    if (focusedOption.name === "realm") {
      const region = getOptionValue(options, "region");
      const regionStr = typeof region === "string" ? region : undefined;
      return autocompleteRealm(query, regionStr as "us" | "eu" | "kr" | "tw" | undefined);
    }
  }

  return [];
}

/**
 * Routes commands to appropriate handler
 */
async function handleCommand(interaction: DiscordInteraction) {
  const commandName = interaction.data?.name;

  if (commandName === "player") {
    return handlePlayerCommand(interaction);
  }

  if (commandName === "leaderboard") {
    return handleLeaderboardCommand(interaction);
  }

  return {
    content: "Unknown command",
  };
}

/**
 * Handles button clicks
 */
async function handleButton(interaction: DiscordInteraction) {
  const customId = interaction.data?.custom_id || "";

  // handle refresh player profile
  if (customId.startsWith("refresh_player:")) {
    const [, region, realm, name] = customId.split(":");

    const { handlePlayerCommand } = await import("../src/discord/commands/player.js");

    const fakeInteraction: DiscordInteraction = {
      ...interaction,
      data: {
        name: "player",
        options: [
          { name: "name", type: 3, value: name }, // STRING
          { name: "region", type: 3, value: region }, // STRING
          { name: "realm", type: 3, value: realm }, // STRING
        ],
      },
    };

    const response = await handlePlayerCommand(fakeInteraction);
    return response;
  }

  // handle refresh dungeon leaderboard
  if (customId.startsWith("refresh_dungeon:")) {
    const [, dungeon, scope, region, realm, limit, season] = customId.split(":");

    const { handleLeaderboardCommand } = await import("../src/discord/commands/leaderboard.js");

    const options: any[] = [
      { name: "dungeon", type: 3, value: dungeon }, // STRING
      { name: "scope", type: 3, value: scope }, // STRING
    ];
    if (region) options.push({ name: "region", type: 3, value: region }); // STRING
    if (realm) options.push({ name: "realm", type: 3, value: realm }); // STRING
    if (limit) options.push({ name: "limit", type: 4, value: parseInt(limit) }); // INTEGER
    if (season) options.push({ name: "season", type: 3, value: season }); // STRING

    const fakeInteraction: DiscordInteraction = {
      ...interaction,
      data: {
        name: "leaderboard",
        options: [{ name: "dungeon", type: 1, options }], // SUB_COMMAND
      },
    };

    const response = await handleLeaderboardCommand(fakeInteraction);
    return response;
  }

  // handle refresh player leaderboard
  if (customId.startsWith("refresh_players:")) {
    const [, scope, region, realm, className, limit, season] = customId.split(":");

    const { handleLeaderboardCommand } = await import("../src/discord/commands/leaderboard.js");

    const options: any[] = [{ name: "scope", type: 3, value: scope }]; // STRING
    if (region) options.push({ name: "region", type: 3, value: region }); // STRING
    if (realm) options.push({ name: "realm", type: 3, value: realm }); // STRING
    if (className) options.push({ name: "class", type: 3, value: className }); // STRING
    if (limit) options.push({ name: "limit", type: 4, value: parseInt(limit) }); // INTEGER
    if (season) options.push({ name: "season", type: 3, value: season }); // STRING

    const fakeInteraction: DiscordInteraction = {
      ...interaction,
      data: {
        name: "leaderboard",
        options: [{ name: "players", type: 1, options }], // SUB_COMMAND
      },
    };

    const response = await handleLeaderboardCommand(fakeInteraction);
    return response;
  }

  return {
    content: "Unknown button",
  };
}

// recursive type for command options
interface CommandOptionTree {
  name: string;
  value?: unknown;
  focused?: boolean;
  options?: CommandOptionTree[];
}

/**
 * Helper to find focused option in nested options
 */
function findFocusedOption(
  options: CommandOptionTree[],
): { name: string; value: unknown } | null {
  for (const option of options) {
    if (option.focused) {
      return { name: option.name, value: option.value };
    }
    if (option.options) {
      const found = findFocusedOption(option.options);
      if (found) return found;
    }
  }
  return null;
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
