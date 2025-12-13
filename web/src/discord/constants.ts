// class colors for embeds (hex converted to decimal)
export const CLASS_COLORS = {
  death_knight: 0xc41e3a,
  druid: 0xff7c0a,
  hunter: 0xaad372,
  mage: 0x3fc7eb,
  monk: 0x00ff98,
  paladin: 0xf48cba,
  priest: 0xffffff,
  rogue: 0xfff468,
  shaman: 0x0070dd,
  warlock: 0x8788ee,
  warrior: 0xc69b6d,
} as const;

// discord custom emoji IDs for specs
// use "ClassName|SpecName" format to match frontend SPEC_ICON_MAP
// format: <:emoji_name:emoji_id>
export const SPEC_EMOJI_IDS: Record<string, string> = {
  "Death Knight|Blood": "1446313246582640700",
  "Death Knight|Frost": "1446313363926814730",
  "Death Knight|Unholy": "1446313642441310239",

  "Druid|Balance": "1446313226588655777",
  "Druid|Feral": "1446313337955815615",
  "Druid|Guardian": "1446313405865791618",
  "Druid|Restoration": "1446313490917883945",

  "Hunter|Beast Mastery": "1446313235866325132",
  "Hunter|Marksmanship": "1446313445636182091",
  "Hunter|Survival": "1446313630369972305",

  "Mage|Arcane": "1446313199522807901",
  "Mage|Fire": "1446313352640073729",
  "Mage|Frost": "1446313378581577898",

  "Monk|Brewmaster": "1446305474919006208",
  "Monk|Mistweaver": "1446313458420158566",
  "Monk|Windwalker": "1446313663232479304",

  "Paladin|Holy": "1446313417530019882",
  "Paladin|Protection": "1446313469161766936",
  "Paladin|Retribution": "1446313518076006566",

  "Priest|Discipline": "1446313298227232838",
  "Priest|Holy": "1446313434206568549",
  "Priest|Shadow": "1446313552125362257",

  "Rogue|Assassination": "1446313217797259334",
  "Rogue|Combat": "1446313266530750555",
  "Rogue|Subtlety": "1446313576225837206",

  "Shaman|Elemental": "1446313313276530833",
  "Shaman|Enhancement": "1446313326836449350",
  "Shaman|Restoration": "1446313506264584192",

  "Warlock|Affliction": "1446313184163004447",
  "Warlock|Demonology": "1446313277695983739",
  "Warlock|Destruction": "1446313287401607310",

  "Warrior|Arms": "1446313209031168105",
  "Warrior|Fury": "1446313391315619932",
  "Warrior|Protection": "1446313478842482728",
};

// discord custom emoji IDs for dungeons
// format: <:emoji_name:emoji_id>
export const DUNGEON_EMOJI_IDS: Record<string, string> = {
  "gate-of-the-setting-sun": "1446337471221596291",
  "mogu-shan-palace": "1446337472844796057",
  "scarlet-halls": "1446337474912583730",
  "scarlet-monastery": "1446337477504794755",
  scholomance: "1446337479425658880",
  "shado-pan-monastery": "1446337481166291027",
  "siege-of-niuzao-temple": "1446337482793943213",
  "stormstout-brewery": "1446337557234188351",
  "temple-of-the-jade-serpent": "1446337487612936332",
};

// ranking bracket display names
export const BRACKET_NAMES = {
  artifact: "Artifact",
  excellent: "Excellent",
  epic: "Epic",
  rare: "Rare",
  uncommon: "Uncommon",
  common: "Common",
} as const;

// region display names
export const REGION_NAMES = {
  us: "US",
  eu: "EU",
  kr: "KR",
  tw: "TW",
} as const;

// scope display names
export const SCOPE_NAMES = {
  global: "Global",
  region: "Regional",
  realm: "Realm",
} as const;

// pagination constants
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 10,
  MAX_PAGE_SIZE: 25,
  LEADERBOARD_PAGE_SIZE: 10,
} as const;

// API configuration
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "https://wowsimstats.com";

// discord limits
export const DISCORD_LIMITS = {
  MAX_EMBED_TITLE: 256,
  MAX_EMBED_DESCRIPTION: 4096,
  MAX_EMBED_FIELDS: 25,
  MAX_EMBED_FIELD_NAME: 256,
  MAX_EMBED_FIELD_VALUE: 1024,
  MAX_EMBED_FOOTER: 2048,
  MAX_EMBED_AUTHOR_NAME: 256,
  MAX_EMBEDS_PER_MESSAGE: 10,
  MAX_COMPONENTS_PER_ROW: 5,
  MAX_ROWS_PER_MESSAGE: 5,
  MAX_AUTOCOMPLETE_CHOICES: 25,
  INTERACTION_TIMEOUT_MS: 3000,
} as const;

// cache configuration
export const CACHE = {
  INDEX_TTL_MS: 1000 * 60 * 60, // 1 hour
  PLAYER_SEARCH_TTL_MS: 1000 * 60 * 30, // 30 minutes
  REGION_INDEX_TTL_MS: 1000 * 60 * 60, // 1 hour
} as const;

// default season (current)
export const DEFAULT_SEASON_ID = "2";
