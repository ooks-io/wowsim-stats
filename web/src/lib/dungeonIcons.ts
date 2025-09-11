// Static mapping of known MoP Challenge Mode dungeons to icon slugs.
// Keys are normalized forms (lowercase, alphanumeric only) of common slug/name variants.

function normalize(s?: string | null): string {
  return String(s || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "")
    .trim();
}

const ICONS: Record<string, string> = {
  // Gate of the Setting Sun
  gateofthesettingsun: "achievement_greatwall",
  gatesettingsun: "achievement_greatwall",

  // Temple of the Jade Serpent
  templeofthejadeserpent: "achievement_jadeserpent",
  jadeserpent: "achievement_jadeserpent",

  // Mogu'shan Palace
  mogushanpalace: "achievement_dungeon_mogupalace",
  mogushan: "achievement_dungeon_mogupalace",

  // Scarlet Monastery
  scarletmonastery: "spell_holy_resurrection",
  scarletmonestary: "spell_holy_resurrection", // common misspelling safeguard

  // Siege of Niuzao Temple
  siegeofniuzaotemple: "achievement_dungeon_siegeofniuzaotemple",
  siegeofniuzao: "achievement_dungeon_siegeofniuzaotemple",
  niuzaotemple: "achievement_dungeon_siegeofniuzaotemple",

  // Stormstout Brewery
  stormstoutbrewery: "achievement_brewery",
  brewery: "achievement_brewery",

  // Scarlet Halls
  scarlethalls: "inv_helmet_52",

  // Scholomance
  scholomance: "spell_holy_senseundead",

  // Shado-Pan Monastery
  shadopanmonastery: "achievement_shadowpan_hideout",
  shadopan: "achievement_shadowpan_hideout",
};

export function getDungeonIconSlug(
  dungeonSlug?: string,
  dungeonName?: string,
): string | null {
  const bySlug = ICONS[normalize(dungeonSlug)];
  if (bySlug) return bySlug;
  const byName = ICONS[normalize(dungeonName)];
  if (byName) return byName;
  return null;
}

export function getDungeonIconUrl(
  dungeonSlug?: string,
  dungeonName?: string,
): string | null {
  const icon = getDungeonIconSlug(dungeonSlug, dungeonName);
  if (!icon) return null;
  return `https://wow.zamimg.com/images/wow/icons/medium/${icon}.jpg`;
}
