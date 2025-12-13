// player profile embed formatter

import type { Embed, MessageComponent } from "../types.js";
import { ComponentType, ButtonStyle } from "../types.js";
import type { PlayerProfileData } from "../../lib/types.js";
import {
  SPEC_EMOJI_IDS,
  API_BASE_URL,
  DUNGEON_EMOJI_IDS,
  CLASS_COLORS,
} from "../constants.js";
import { formatDurationFromMs } from "../../lib/utils.js";

// bracket to percentile mapping
const BRACKET_PERCENTILES: Record<string, string> = {
  artifact: "#1",
  excellent: "Top 1%",
  legendary: "Top 5%",
  epic: "Top 20%",
  rare: "Top 40%",
  uncommon: "Top 60%",
  common: "—",
};

// creates a player profile embed
export function createPlayerProfileEmbed(
  profile: PlayerProfileData,
  season: string = "2",
): Embed {
  const player = profile.player;
  if (!player) {
    throw new Error("Player data not found");
  }

  // get specified season data, fallback to latest season if requested season doesn't exist
  let seasonData = player.seasons?.[season];
  let actualSeason = season;

  if (!seasonData && player.seasons) {
    // find the highest season number available
    const availableSeasons = Object.keys(player.seasons)
      .map(Number)
      .sort((a, b) => b - a);
    if (availableSeasons.length > 0) {
      actualSeason = String(availableSeasons[0]);
      seasonData = player.seasons[actualSeason];
    }
  }

  // determine ranking bracket
  const bracket = seasonData?.global_ranking_bracket || "common";

  // spec emoji (use class|spec format like frontend)
  const specKey = `${player.class_name}|${player.active_spec_name || ""}`;
  const emojiId = SPEC_EMOJI_IDS[specKey];
  const emojiName = (player.active_spec_name || "")
    .toLowerCase()
    .replace(/\s+/g, "_");
  const specEmoji = emojiId ? `<:${emojiName}:${emojiId}> ` : "";

  // build title
  const title = `${specEmoji}${player.name} - ${player.realm_name} (${player.region.toUpperCase()})`;

  // build description
  let description = "";
  if (player.class_name && player.active_spec_name) {
    description = `**${player.class_name}** (${player.active_spec_name})`;
    if (player.guild_name) {
      description += `\n<${player.guild_name}>`;
    }
  }

  // build fields
  const fields = [];

  // rankings field
  if (seasonData) {
    let rankingsText = "";
    const globalPercentile = BRACKET_PERCENTILES[bracket] || "—";
    const regionalBracket = seasonData.regional_ranking_bracket || "";
    const regionalPercentile = BRACKET_PERCENTILES[regionalBracket] || "—";
    const realmBracket = seasonData.realm_ranking_bracket || "";
    const realmPercentile = BRACKET_PERCENTILES[realmBracket] || "—";

    if (seasonData.global_ranking) {
      // only show percentile if not rank 1 (would be redundant)
      const percentileText =
        seasonData.global_ranking === 1 ? "" : ` (${globalPercentile})`;
      rankingsText += `**Global:** #${seasonData.global_ranking}${percentileText}\n`;
    }
    if (seasonData.regional_ranking) {
      const percentileText =
        seasonData.regional_ranking === 1 ? "" : ` (${regionalPercentile})`;
      rankingsText += `**Regional:** #${seasonData.regional_ranking}${percentileText}\n`;
    }
    if (seasonData.realm_ranking) {
      const percentileText =
        seasonData.realm_ranking === 1 ? "" : ` (${realmPercentile})`;
      rankingsText += `**Realm:** #${seasonData.realm_ranking}${percentileText}`;
    }

    fields.push({
      name: "📊 Rankings",
      value: rankingsText,
      inline: false,
    });
  }

  // combined time field
  if (seasonData?.combined_best_time) {
    const combinedTime = formatDurationFromMs(seasonData.combined_best_time);
    const dungeonsCompleted = seasonData.dungeons_completed || 0;
    fields.push({
      name: "Combined Time",
      value: `\`${combinedTime}\` (${dungeonsCompleted}/9 dungeons)`,
      inline: false,
    });
  }

  // best runs field
  if (seasonData?.best_runs) {
    const bestRunsArray = Object.values(seasonData.best_runs);
    if (bestRunsArray.length > 0) {
      const bestRunsText = bestRunsArray
        .slice(0, 9)
        .map((run) => {
          const dungeonSlug = run.dungeon_slug || "";

          // get dungeon emoji
          const emojiId = DUNGEON_EMOJI_IDS[dungeonSlug];
          const emojiName = dungeonSlug.replace(/-/g, "_");
          const dungeonEmoji = emojiId ? `<:${emojiName}:${emojiId}>` : "";

          return `${dungeonEmoji} \`${formatDurationFromMs(run.duration)}\``;
        })
        .join("\n");

      fields.push({
        name: "⏱️ Best Times",
        value: bestRunsText || "No runs recorded",
        inline: false,
      });
    }
  }

  // get class color for embed sidebar
  const classKey = player.class_name
    ?.toLowerCase()
    .replace(/\s+/g, "_") as keyof typeof CLASS_COLORS;
  const color = CLASS_COLORS[classKey] || 0x5865f2; // fallback to discord blurple

  // add footer with season info
  const seasonNote =
    actualSeason !== season
      ? ` • Showing Season ${actualSeason} (Season ${season} data not available)`
      : ` • Season ${actualSeason}`;

  return {
    title,
    description,
    fields,
    color,
    footer: {
      text: `wowsimstats.com${seasonNote}`,
    },
    timestamp: new Date().toISOString(),
  };
}

// creates buttons to view profile on website and armory
export function createViewProfileButton(
  name: string,
  realm: string,
  region: string,
  season: string = "2",
): MessageComponent[] {
  const profileUrl = `${API_BASE_URL}/player/${region}/${realm}/${name}?season=${season}`;
  const armoryUrl = `https://classic-armory.org/character/${region}/mop/${realm}/${name}`;

  return [
    {
      type: ComponentType.ACTION_ROW,
      components: [
        {
          type: ComponentType.BUTTON,
          style: ButtonStyle.PRIMARY,
          label: "🔄 Refresh",
          custom_id: `refresh_player:${region}:${realm}:${name}:${season}`,
        },
        {
          type: ComponentType.BUTTON,
          style: ButtonStyle.LINK,
          label: "View Full Profile",
          url: profileUrl,
        },
        {
          type: ComponentType.BUTTON,
          style: ButtonStyle.LINK,
          label: "View Armory",
          url: armoryUrl,
        },
      ],
    },
  ];
}

// helper to get dungeon short name from slug
function getDungeonShortName(slug: string): string {
  const shortNames: Record<string, string> = {
    "gate-of-the-setting-sun": "GSS",
    "mogu-shan-palace": "MSP",
    "scarlet-halls": "SH",
    "scarlet-monastery": "SM",
    scholomance: "SCHOLO",
    "shado-pan-monastery": "SPM",
    "siege-of-niuzao-temple": "SNT",
    "stormstout-brewery": "SB",
    "temple-of-the-jade-serpent": "TJS",
  };
  return shortNames[slug] || slug;
}
