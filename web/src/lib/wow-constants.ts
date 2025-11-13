export const CLASS_COLORS = {
  death_knight: "#C41E3A",
  druid: "#FF7C0A",
  hunter: "#AAD372",
  mage: "#3FC7EB",
  monk: "#00FF98",
  paladin: "#F48CBA",
  priest: "#FFFFFF",
  rogue: "#FFF468",
  shaman: "#0070DD",
  warlock: "#8788EE",
  warrior: "#C69B6D",
} as const;

export const CLASSES = [
  "death_knight",
  "druid",
  "hunter",
  "mage",
  "monk",
  "paladin",
  "priest",
  "rogue",
  "shaman",
  "warlock",
  "warrior",
] as const;

export const SPEC_MAP = {
  73: { role: "tank", class: "Warrior", spec: "Protection" },
  104: { role: "tank", class: "Druid", spec: "Guardian" },
  250: { role: "tank", class: "Death Knight", spec: "Blood" },
  268: { role: "tank", class: "Monk", spec: "Brewmaster" },
  66: { role: "tank", class: "Paladin", spec: "Protection" },

  105: { role: "healer", class: "Druid", spec: "Restoration" },
  270: { role: "healer", class: "Monk", spec: "Mistweaver" },
  65: { role: "healer", class: "Paladin", spec: "Holy" },
  256: { role: "healer", class: "Priest", spec: "Discipline" },
  257: { role: "healer", class: "Priest", spec: "Holy" },
  264: { role: "healer", class: "Shaman", spec: "Restoration" },

  71: { role: "dps", class: "Warrior", spec: "Arms" },
  72: { role: "dps", class: "Warrior", spec: "Fury" },
  70: { role: "dps", class: "Paladin", spec: "Retribution" },
  253: { role: "dps", class: "Hunter", spec: "Beast Mastery" },
  254: { role: "dps", class: "Hunter", spec: "Marksmanship" },
  255: { role: "dps", class: "Hunter", spec: "Survival" },
  259: { role: "dps", class: "Rogue", spec: "Assassination" },
  260: { role: "dps", class: "Rogue", spec: "Combat" },
  261: { role: "dps", class: "Rogue", spec: "Subtlety" },
  258: { role: "dps", class: "Priest", spec: "Shadow" },
  251: { role: "dps", class: "Death Knight", spec: "Frost" },
  252: { role: "dps", class: "Death Knight", spec: "Unholy" },
  262: { role: "dps", class: "Shaman", spec: "Elemental" },
  263: { role: "dps", class: "Shaman", spec: "Enhancement" },
  62: { role: "dps", class: "Mage", spec: "Arcane" },
  63: { role: "dps", class: "Mage", spec: "Fire" },
  64: { role: "dps", class: "Mage", spec: "Frost" },
  265: { role: "dps", class: "Warlock", spec: "Affliction" },
  266: { role: "dps", class: "Warlock", spec: "Demonology" },
  267: { role: "dps", class: "Warlock", spec: "Destruction" },
  269: { role: "dps", class: "Monk", spec: "Windwalker" },
  102: { role: "dps", class: "Druid", spec: "Balance" },
  103: { role: "dps", class: "Druid", spec: "Feral" },
} as const;

export const DUNGEON_MAP = {
  2: "Temple of the Jade Serpent",
  56: "Stormstout Brewery",
  57: "Gate of the Setting Sun",
  58: "Shado-Pan Monastery",
  59: "Siege of Niuzao Temple",
  60: "Mogu'shan Palace",
  76: "Scholomance",
  77: "Scarlet Halls",
  78: "Scarlet Monastery",
} as const;

export type ClassName = keyof typeof CLASS_COLORS;
export type SpecId = keyof typeof SPEC_MAP;
export type DungeonId = keyof typeof DUNGEON_MAP;

// utility functions
export function getClassColor(className: string): string {
  const normalizedClass = className
    .toLowerCase()
    .replace(/\s+/g, "_") as ClassName;
  return CLASS_COLORS[normalizedClass] || "#FFFFFF";
}

export function getClassColorClass(className: string): string {
  const normalizedClass = className.toLowerCase().replace(/\s+/g, "-");
  return `class-${normalizedClass}`;
}

export function getClassTextClass(
  className: string | undefined | null,
): string {
  const key = String(className || "common")
    .toLowerCase()
    .replace(/[\s_]+/g, "-");
  return `text-${key}`;
}

export function getSpecInfo(specId: number) {
  return SPEC_MAP[specId as SpecId] || null;
}

export function getDungeonName(dungeonId: number): string {
  return DUNGEON_MAP[dungeonId as DungeonId] || "Unknown Dungeon";
}

export function dungeonNameToSlug(dungeonName: string): string {
  return dungeonName.toLowerCase().replace(/[^a-z0-9]/g, "-");
}

export function dungeonSlugToId(slug: string): string | null {
  for (const [id, name] of Object.entries(DUNGEON_MAP)) {
    if (dungeonNameToSlug(name) === slug) {
      return id;
    }
  }
  return null;
}

export function getSpecIcon(
  className: string,
  specName: string,
): string | null {
  const key = `${className}|${specName}`;
  return SPEC_ICON_MAP[key] || null;
}

// SPEC_OPTIONS for class/spec selection dropdowns
// REGION_OPTIONS for region selection dropdowns
export const REGION_OPTIONS = [
  { value: "", label: "All" },
  { value: "us", label: "US" },
  { value: "eu", label: "EU" },
  { value: "kr", label: "KR" },
  { value: "tw", label: "TW" },
] as const;

export const SPEC_OPTIONS = {
  death_knight: [
    { value: "250", label: "Blood" },
    { value: "251", label: "Frost" },
    { value: "252", label: "Unholy" },
  ],
  druid: [
    { value: "102", label: "Balance" },
    { value: "103", label: "Feral" },
    { value: "104", label: "Guardian" },
    { value: "105", label: "Restoration" },
  ],
  hunter: [
    { value: "253", label: "Beast Mastery" },
    { value: "254", label: "Marksmanship" },
    { value: "255", label: "Survival" },
  ],
  mage: [
    { value: "62", label: "Arcane" },
    { value: "63", label: "Fire" },
    { value: "64", label: "Frost" },
  ],
  monk: [
    { value: "268", label: "Brewmaster" },
    { value: "270", label: "Mistweaver" },
    { value: "269", label: "Windwalker" },
  ],
  paladin: [
    { value: "65", label: "Holy" },
    { value: "66", label: "Protection" },
    { value: "70", label: "Retribution" },
  ],
  priest: [
    { value: "256", label: "Discipline" },
    { value: "257", label: "Holy" },
    { value: "258", label: "Shadow" },
  ],
  rogue: [
    { value: "259", label: "Assassination" },
    { value: "260", label: "Combat" },
    { value: "261", label: "Subtlety" },
  ],
  shaman: [
    { value: "262", label: "Elemental" },
    { value: "263", label: "Enhancement" },
    { value: "264", label: "Restoration" },
  ],
  warlock: [
    { value: "265", label: "Affliction" },
    { value: "266", label: "Demonology" },
    { value: "267", label: "Destruction" },
  ],
  warrior: [
    { value: "71", label: "Arms" },
    { value: "72", label: "Fury" },
    { value: "73", label: "Protection" },
  ],
} as const;

export const SPEC_ICON_MAP: Record<string, string> = {
  "Mage|Frost":
    "https://wow.zamimg.com/images/wow/icons/large/spell_frost_frostbolt02.jpg",
  "Mage|Fire":
    "https://wow.zamimg.com/images/wow/icons/large/spell_fire_flamebolt.jpg",
  "Mage|Arcane":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_magicalsentry.jpg",

  "Paladin|Holy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_holybolt.jpg",
  "Paladin|Protection":
    "https://wow.zamimg.com/images/wow/icons/large/ability_paladin_shieldofthetemplar.jpg",
  "Paladin|Retribution":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_auraoflight.jpg",

  "Death Knight|Blood":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_bloodpresence.jpg",
  "Death Knight|Frost":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_frostpresence.jpg",
  "Death Knight|Unholy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_unholypresence.jpg",

  "Druid|Balance":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_starfall.jpg",
  "Druid|Feral":
    "https://wow.zamimg.com/images/wow/icons/large/ability_druid_catform.jpg",
  "Druid|Guardian":
    "https://wow.zamimg.com/images/wow/icons/large/ability_racial_bearform.jpg",
  "Druid|Restoration":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_healingtouch.jpg",

  "Hunter|Beast Mastery":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_bestialdiscipline.jpg",
  "Hunter|Marksmanship":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_focusedaim.jpg",
  "Hunter|Survival":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_camouflage.jpg",

  "Monk|Brewmaster":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_brewmaster_spec.jpg",
  "Monk|Windwalker":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_windwalker_spec.jpg",
  "Monk|Mistweaver":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_mistweaver_spec.jpg",

  "Priest|Discipline":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_powerwordshield.jpg",
  "Priest|Holy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_guardianspirit.jpg",
  "Priest|Shadow":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_shadowwordpain.jpg",

  "Rogue|Subtlety":
    "https://wow.zamimg.com/images/wow/icons/large/ability_stealth.jpg",
  "Rogue|Assassination":
    "https://wow.zamimg.com/images/wow/icons/large/ability_rogue_eviscerate.jpg",
  "Rogue|Combat":
    "https://wow.zamimg.com/images/wow/icons/large/ability_backstab.jpg",

  "Shaman|Elemental":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_lightning.jpg",
  "Shaman|Enhancement":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shaman_improvedstormstrike.jpg",
  "Shaman|Restoration":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_magicimmunity.jpg",

  "Warlock|Affliction":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_deathcoil.jpg",
  "Warlock|Destruction":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_rainoffire.jpg",
  "Warlock|Demonology":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_metamorphosis.jpg",

  "Warrior|Arms":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_savageblow.jpg",
  "Warrior|Fury":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_innerrage.jpg",
  "Warrior|Protection":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_defensivestance.jpg",
};

// create player element with class colors and spec info
export function createPlayerElement(player: {
  name: string;
  class_name?: string;
  spec_id?: number;
  realm_name?: string;
  region?: string;
}): string {
  const classColor = getClassColor(player.class_name || "");
  const specInfo = player.spec_id ? getSpecInfo(player.spec_id) : null;

  return `
    <span class="player-element" style="color: ${classColor};">
      <span class="player-name">${player.name}</span>
      ${specInfo ? `<span class="player-spec">${specInfo.spec}</span>` : ""}
      ${player.realm_name ? `<span class="player-realm">-${player.realm_name}</span>` : ""}
    </span>
  `;
}

// equipment quality constants
export const ITEM_QUALITY_COLORS = {
  POOR: "#9d9d9d",
  COMMON: "#ffffff",
  UNCOMMON: "#1eff00",
  RARE: "#0070dd",
  EPIC: "#a335ee",
  LEGENDARY: "#ff8000",
  ARTIFACT: "#e6cc80",
  HEIRLOOM: "#00ccff",
} as const;

export function getQualityColorClass(quality: string): string {
  const qualityMap = {
    POOR: "quality-poor",
    COMMON: "quality-common",
    UNCOMMON: "quality-uncommon",
    RARE: "quality-rare",
    EPIC: "quality-epic",
    LEGENDARY: "quality-legendary",
    ARTIFACT: "quality-artifact",
    HEIRLOOM: "quality-heirloom",
  };
  return qualityMap[quality as keyof typeof qualityMap] || "quality-common";
}

// equipment slot ordering for display
export const EQUIPMENT_SLOT_ORDER = [
  "HEAD",
  "NECK",
  "SHOULDER",
  "BACK",
  "CHEST",
  "WRIST",
  "HANDS",
  "WAIST",
  "LEGS",
  "FEET",
  "FINGER_1",
  "FINGER_2",
  "TRINKET_1",
  "TRINKET_2",
  "MAIN_HAND",
  "OFF_HAND",
  "RANGED",
] as const;
