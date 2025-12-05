// leaderboard embed formatter

import type { Embed, MessageComponent } from "../types.js";
import { ComponentType, ButtonStyle } from "../types.js";
import type { TeamMember } from "../../lib/types.js";
import {
  DUNGEON_EMOJI_IDS,
  SPEC_EMOJI_IDS,
  SCOPE_NAMES,
  REGION_NAMES,
} from "../constants.js";
import { formatDurationFromMs } from "../../lib/utils.js";
import { getSpecInfo } from "../../lib/wow-constants.js";

export interface LeaderboardRun {
  rank: number;
  duration: number;
  timestamp: string;
  members: TeamMember[];
}

export interface PlayerRanking {
  rank: number;
  name: string;
  realm_name: string;
  region: string;
  class_name: string;
  active_spec_name?: string;
  best_avg_time?: number;
  completed_dungeons: number;
  global_ranking_bracket?: string;
}

// main dungeon leaderboard embed
export function createDungeonLeaderboardEmbed(
  dungeonName: string,
  dungeonSlug: string,
  scope: string,
  region?: string,
  realm?: string,
  runs: LeaderboardRun[] = [],
  limit: number = 10,
  seasonId: string = "1",
): Embed {
  // Get dungeon emoji if available
  const emojiId = DUNGEON_EMOJI_IDS[dungeonSlug];
  const emojiName = dungeonSlug.replace(/-/g, "_");
  const emoji = emojiId ? `<:${emojiName}:${emojiId}>` : "🏆";

  // build scope display
  let scopeDisplay = SCOPE_NAMES[scope as keyof typeof SCOPE_NAMES] || scope;
  if (scope === "region" && region) {
    scopeDisplay += ` - ${REGION_NAMES[region.toUpperCase() as keyof typeof REGION_NAMES]}`;
  } else if (scope === "realm" && realm) {
    scopeDisplay += ` - ${realm}`;
  }

  // helper to get spec emoji
  const getSpecEmoji = (specId: number): string => {
    const specInfo = getSpecInfo(specId);
    if (!specInfo) return "";

    const specKey = `${specInfo.class}|${specInfo.spec}`;
    const emojiId = SPEC_EMOJI_IDS[specKey];
    if (!emojiId) return "";

    const emojiName = specInfo.spec.toLowerCase().replace(/\s+/g, "_");
    return `<:${emojiName}:${emojiId}>`;
  };

  // build run entries
  const runLines = runs.slice(0, limit).map((run, index) => {
    const rank = index + 1;
    const time = formatDurationFromMs(run.duration);
    const date = new Date(run.timestamp).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });

    // format team members with spec icons
    const team = run.members
      .map((m) => {
        const specEmoji = getSpecEmoji(m.spec_id);
        return specEmoji ? `${specEmoji} **${m.name}**` : `**${m.name}**`;
      })
      .join(", ");

    return `**${rank}.** \`${time}\` • *${date}*\n${team}`;
  });

  const description = `Season ${seasonId}\n\n${runLines.join("\n\n")}`;

  return {
    title: `${emoji} ${dungeonName} - ${scopeDisplay}`,
    description,
    footer: {
      text: "wowsimstats.com",
    },
    timestamp: new Date().toISOString(),
  };
}

// player leaderboard embed
export function createPlayerLeaderboardEmbed(
  scope: string,
  region?: string,
  realm?: string,
  className?: string,
  players: PlayerRanking[] = [],
  limit: number = 25,
  seasonId: string = "1",
): Embed {
  // build scope display
  let scopeDisplay = SCOPE_NAMES[scope as keyof typeof SCOPE_NAMES] || scope;
  if (scope === "region" && region) {
    scopeDisplay += ` - ${REGION_NAMES[region.toUpperCase() as keyof typeof REGION_NAMES]}`;
  } else if (scope === "realm" && realm) {
    scopeDisplay += ` - ${realm}`;
  }

  // add class filter to title if applicable
  const titleSuffix = className ? ` (${className})` : "";

  // build description with all player entries
  const playerLines = players.slice(0, limit).map((player) => {
    const timeStr = player.best_avg_time
      ? `\`${formatDurationFromMs(player.best_avg_time)}\``
      : "—";

    // get spec emoji if available (use class|spec format like frontend)
    const specKey = `${player.class_name}|${player.active_spec_name || ""}`;
    const emojiId = SPEC_EMOJI_IDS[specKey];

    // generate emoji name from spec name (lowercase, replace spaces with underscores)
    const emojiName = (player.active_spec_name || "")
      .toLowerCase()
      .replace(/\s+/g, "_");
    const specEmoji = emojiId ? `<:${emojiName}:${emojiId}>` : "";

    // formatting
    if (specEmoji) {
      return `**${player.rank}.** ${timeStr} ${specEmoji} **${player.name}** - *${player.realm_name} (${player.region.toUpperCase()})*`;
    } else {
      return `**${player.rank}.** ${timeStr} **${player.name}** - *${player.realm_name} (${player.region.toUpperCase()})*`;
    }
  });

  const description = `Season ${seasonId}\n\n${playerLines.join("\n")}`;

  return {
    title: `Player Rankings - ${scopeDisplay}${titleSuffix}`,
    description,
    footer: {
      text: "wowsimstats.com",
    },
    timestamp: new Date().toISOString(),
  };
}

// creates refresh button for dungeon leaderboard
export function createDungeonLeaderboardRefreshButton(
  dungeon: string,
  scope: string,
  region?: string,
  realm?: string,
  limit?: number,
  season?: string,
): MessageComponent[] {
  const params = [dungeon, scope, region || "", realm || "", limit || "10", season || "1"];
  return [
    {
      type: ComponentType.ACTION_ROW,
      components: [
        {
          type: ComponentType.BUTTON,
          style: ButtonStyle.PRIMARY,
          label: "🔄 Refresh",
          custom_id: `refresh_dungeon:${params.join(":")}`,
        },
      ],
    },
  ];
}

// creates refresh button for player leaderboard
export function createPlayerLeaderboardRefreshButton(
  scope: string,
  region?: string,
  realm?: string,
  className?: string,
  limit?: number,
  season?: string,
): MessageComponent[] {
  const params = [scope, region || "", realm || "", className || "", limit || "25", season || "1"];
  return [
    {
      type: ComponentType.ACTION_ROW,
      components: [
        {
          type: ComponentType.BUTTON,
          style: ButtonStyle.PRIMARY,
          label: "🔄 Refresh",
          custom_id: `refresh_players:${params.join(":")}`,
        },
      ],
    },
  ];
}
