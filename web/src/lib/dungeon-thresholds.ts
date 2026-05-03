// Per-dungeon timer thresholds for MoP Challenge Modes, in milliseconds.
// Gold = the achievement par-time. Platinum / Title (aka "Remarkable")
// are tighter community/in-game cutoffs.

export type TimeTier = "title" | "platinum" | "gold";

export interface DungeonThresholds {
  goldMs: number;
  platinumMs: number;
  titleMs: number;
}

export const DUNGEON_THRESHOLDS: Record<number, DungeonThresholds> = {
  2:  { goldMs: 900000,  platinumMs: 615000, titleMs: 510000 }, // Temple of the Jade Serpent (15:00 / 10:15 / 8:30)
  56: { goldMs: 720000,  platinumMs: 495000, titleMs: 390000 }, // Stormstout Brewery        (12:00 / 8:15 / 6:30)
  57: { goldMs: 780000,  platinumMs: 480000, titleMs: 330000 }, // Gate of the Setting Sun   (13:00 / 8:00 / 5:30)
  58: { goldMs: 1260000, platinumMs: 840000, titleMs: 630000 }, // Shado-Pan Monastery       (21:00 / 14:00 / 10:30)
  59: { goldMs: 1050000, platinumMs: 735000, titleMs: 615000 }, // Siege of Niuzao Temple    (17:30 / 12:15 / 10:15)
  60: { goldMs: 720000,  platinumMs: 495000, titleMs: 405000 }, // Mogu'shan Palace          (12:00 / 8:15 / 6:45)
  76: { goldMs: 1140000, platinumMs: 615000, titleMs: 435000 }, // Scholomance               (19:00 / 10:15 / 7:15)
  77: { goldMs: 780000,  platinumMs: 480000, titleMs: 255000 }, // Scarlet Halls             (13:00 / 8:00 / 4:15)
  78: { goldMs: 780000,  platinumMs: 540000, titleMs: 330000 }, // Scarlet Monastery         (13:00 / 9:00 / 5:30)
};

// Slug → dungeon_id, for callsites that have the URL slug but not the numeric id.
// Slugs match those produced by the Go generator.
export const DUNGEON_SLUG_TO_ID: Record<string, number> = {
  "temple-of-the-jade-serpent": 2,
  "stormstout-brewery": 56,
  "gate-of-the-setting-sun": 57,
  "shado-pan-monastery": 58,
  "siege-of-niuzao-temple": 59,
  "mogu-shan-palace": 60,
  "scholomance": 76,
  "scarlet-halls": 77,
  "scarlet-monastery": 78,
};

export function dungeonIdFromSlug(slug: string | undefined | null): number | null {
  if (!slug) return null;
  return DUNGEON_SLUG_TO_ID[slug] ?? null;
}

/**
 * In-game cutoffs are lenient: the displayed cutoff is the first second that
 * still earns the medal. e.g. a 5:30.800 run beats a "5:30" title cutoff,
 * because the timer reads "5:30" until it ticks over to 5:31. So the actual
 * pass condition is `duration < threshold + 1000ms`.
 */
const LENIENCY_MS = 1000;

/**
 * Returns the highest tier the run beat, or null if it didn't beat gold (or
 * the dungeon is unknown).
 */
export function getTimeTier(dungeonId: number, durationMs: number): TimeTier | null {
  const t = DUNGEON_THRESHOLDS[dungeonId];
  if (!t) return null;
  if (durationMs < t.titleMs + LENIENCY_MS) return "title";
  if (durationMs < t.platinumMs + LENIENCY_MS) return "platinum";
  if (durationMs < t.goldMs + LENIENCY_MS) return "gold";
  return null;
}

/**
 * CSS class slug for the tier the run hit, e.g. "time--title".
 * Use this for client-rendered widgets that build DOM in JS.
 * Returns null when the run didn't beat gold or the dungeon is unknown.
 */
export function getTimeClass(dungeonId: number, durationMs: number): string | null {
  const tier = getTimeTier(dungeonId, durationMs);
  return tier ? `time--${tier}` : null;
}

const TIER_LABEL: Record<TimeTier, string> = {
  title: "Title",
  platinum: "Platinum",
  gold: "Gold",
};

/**
 * Hover text describing the tier and its cutoff for the given dungeon.
 * E.g. "Title — cutoff 8:30". Returns null when no tier was hit.
 * Says "cutoff" rather than "under" because the in-game check is lenient
 * (a 8:30.999 run still earns title when the cutoff is 8:30).
 */
export function getTierTooltip(dungeonId: number, durationMs: number): string | null {
  const tier = getTimeTier(dungeonId, durationMs);
  if (!tier) return null;
  const t = DUNGEON_THRESHOLDS[dungeonId];
  if (!t) return null;
  const cutoffMs =
    tier === "title" ? t.titleMs : tier === "platinum" ? t.platinumMs : t.goldMs;
  return `${TIER_LABEL[tier]} — cutoff ${formatCutoff(cutoffMs)}`;
}

function formatCutoff(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}
