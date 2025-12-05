// leaderboard command handler

import type {
  DiscordInteraction,
  InteractionResponseData,
  LeaderboardCommandOptions,
} from "../types.js";
import type {
  LeaderboardData,
  PlayerLeaderboardData,
  ChallengeRun,
  Player,
} from "../../lib/types.js";
import {
  fetchGlobalLeaderboard,
  fetchRegionalLeaderboard,
  fetchRealmLeaderboard,
  fetchPlayerLeaderboard,
} from "../../lib/api.js";
import { API_BASE_URL, DEFAULT_SEASON_ID } from "../constants.js";
import {
  createDungeonLeaderboardEmbed,
  createPlayerLeaderboardEmbed,
  createDungeonLeaderboardRefreshButton,
  createPlayerLeaderboardRefreshButton,
  type LeaderboardRun,
  type PlayerRanking,
} from "../embeds/leaderboard.js";
import {
  createAPIErrorEmbed,
  createInvalidInputEmbed,
} from "../embeds/error.js";
import { getDungeons } from "../indexes/cache.js";

/**
 * handles the /leaderboard command
 * routes to dungeon or players subcommand
 */
export async function handleLeaderboardCommand(
  interaction: DiscordInteraction,
): Promise<InteractionResponseData> {
  const subcommand = interaction.data?.options?.[0];

  if (!subcommand) {
    return {
      embeds: [createInvalidInputEmbed("subcommand", "missing")],
    };
  }

  if (subcommand.name === "dungeon") {
    return handleDungeonLeaderboard(subcommand.options || []);
  } else if (subcommand.name === "players") {
    return handlePlayersLeaderboard(subcommand.options || []);
  }

  return {
    embeds: [createInvalidInputEmbed("subcommand", subcommand.name)],
  };
}

/**
 * handles /leaderboard dungeon subcommand
 */
async function handleDungeonLeaderboard(
  options: Array<{ name: string; value?: string | number | boolean }>,
): Promise<InteractionResponseData> {
  const dungeon = getOptionValue(options, "dungeon") as string;
  const scope = getOptionValue(options, "scope") as string;
  const region = getOptionValue(options, "region") as string | undefined;
  const realm = getOptionValue(options, "realm") as string | undefined;
  const limit = (getOptionValue(options, "limit") as number) || 10;
  const season =
    (getOptionValue(options, "season") as string) || DEFAULT_SEASON_ID;
  const page = 1;

  // validate required options
  if (!dungeon || !scope) {
    return {
      embeds: [
        createInvalidInputEmbed("options", "dungeon and scope are required"),
      ],
    };
  }

  // validate scope
  if (!["global", "region", "realm"].includes(scope)) {
    return {
      embeds: [createInvalidInputEmbed("scope", scope)],
    };
  }

  // validate region if scope is region or realm
  if ((scope === "region" || scope === "realm") && !region) {
    return {
      embeds: [createInvalidInputEmbed("region", "required for this scope")],
    };
  }

  // validate realm if scope is realm
  if (scope === "realm" && !realm) {
    return {
      embeds: [createInvalidInputEmbed("realm", "required for realm scope")],
    };
  }

  try {
    // get dungeon name from cache
    const dungeons = await getDungeons();
    const dungeonInfo = dungeons.find((d) => d.slug === dungeon);
    if (!dungeonInfo) {
      return {
        embeds: [createInvalidInputEmbed("dungeon", dungeon)],
      };
    }

    // fetch leaderboard based on scope
    let data: LeaderboardData | null = null;
    const seasonNum = Number(season);
    if (scope === "global") {
      data = await fetchGlobalLeaderboard(
        dungeonInfo.id,
        page,
        undefined,
        API_BASE_URL,
        seasonNum,
      );
    } else if (scope === "region") {
      data = await fetchRegionalLeaderboard(
        region!,
        dungeonInfo.id,
        page,
        undefined,
        API_BASE_URL,
        seasonNum,
      );
    } else if (scope === "realm") {
      data = await fetchRealmLeaderboard(
        region!,
        realm!,
        dungeonInfo.id,
        page,
        undefined,
        API_BASE_URL,
        seasonNum,
      );
    }

    if (!data || !data.leading_groups) {
      return {
        embeds: [createAPIErrorEmbed("No leaderboard data found")],
      };
    }

    // map runs to our interface (timestamp is already in ms)
    const runs: LeaderboardRun[] = data.leading_groups.map((run) => ({
      rank: 0, // not used, calculated from index in embed
      duration: run.duration,
      timestamp: new Date(run.completed_timestamp).toISOString(),
      members: run.members,
    }));

    // create embed
    const embed = createDungeonLeaderboardEmbed(
      dungeonInfo.name,
      dungeon,
      scope,
      region,
      realm,
      runs,
      limit,
      season,
    );

    // create refresh button
    const components = createDungeonLeaderboardRefreshButton(
      dungeon,
      scope,
      region,
      realm,
      limit,
      season,
    );

    return {
      embeds: [embed],
      components,
    };
  } catch (error) {
    console.error("Error fetching dungeon leaderboard:", error);
    return {
      embeds: [createAPIErrorEmbed()],
    };
  }
}

// handles /leaderboard players subcommand
async function handlePlayersLeaderboard(
  options: Array<{ name: string; value?: string | number | boolean }>,
): Promise<InteractionResponseData> {
  const scopeValue = getOptionValue(options, "scope") as string;
  const region = getOptionValue(options, "region") as string | undefined;
  const realm = getOptionValue(options, "realm") as string | undefined;
  const className = getOptionValue(options, "class") as string | undefined;
  const limit = (getOptionValue(options, "limit") as number) || 25;
  const season =
    (getOptionValue(options, "season") as string) || DEFAULT_SEASON_ID;
  const page = 1;

  // validate required options
  if (!scopeValue) {
    return {
      embeds: [createInvalidInputEmbed("scope", "required")],
    };
  }

  // validate scope
  if (!["global", "region", "realm"].includes(scopeValue)) {
    return {
      embeds: [createInvalidInputEmbed("scope", scopeValue)],
    };
  }

  // map scope value to API format ("region" -> "regional")
  const scope: "global" | "regional" | "realm" =
    scopeValue === "region" ? "regional" : (scopeValue as "global" | "realm");

  try {
    // build API options
    const opts: { classKey?: string; realmSlug?: string; seasonId?: number } =
      {};
    if (className) opts.classKey = className;
    if (realm && scope === "realm") opts.realmSlug = realm;
    if (season) opts.seasonId = Number(season);

    // fetch player leaderboard
    const data: PlayerLeaderboardData | null = await fetchPlayerLeaderboard(
      scope,
      region,
      page,
      10,
      opts,
      API_BASE_URL,
    );

    if (!data || !data.leaderboard) {
      return {
        embeds: [createAPIErrorEmbed("No player leaderboard data found")],
      };
    }

    // map players to our interface
    const players: PlayerRanking[] = data.leaderboard.map((player, index) => ({
      rank: player.global_ranking || index + 1, // use index+1 as fallback
      name: player.name,
      realm_name: player.realm_name,
      region: player.region,
      class_name: player.class_name,
      active_spec_name: player.active_spec_name,
      best_avg_time: player.combined_best_time,
      completed_dungeons: player.dungeons_completed || 0,
      global_ranking_bracket: player.global_ranking_bracket,
    }));

    // create embed
    const embed = createPlayerLeaderboardEmbed(
      scope,
      region,
      realm,
      className,
      players,
      limit,
      season,
    );

    // create refresh button
    const components = createPlayerLeaderboardRefreshButton(
      scope,
      region,
      realm,
      className,
      limit,
      season,
    );

    return {
      embeds: [embed],
      components,
    };
  } catch (error) {
    console.error("Error fetching player leaderboard:", error);
    return {
      embeds: [createAPIErrorEmbed()],
    };
  }
}

// helper to get option value by name
function getOptionValue(
  options: Array<{ name: string; value?: string | number | boolean }>,
  name: string,
): string | number | boolean | undefined {
  return options.find((opt) => opt.name === name)?.value;
}
