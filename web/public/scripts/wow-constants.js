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
};

export const SPEC_MAP = {
  // Tank specs
  73: { role: "tank", class: "Warrior", spec: "Protection" },
  104: { role: "tank", class: "Druid", spec: "Guardian" },
  250: { role: "tank", class: "Death Knight", spec: "Blood" },
  268: { role: "tank", class: "Monk", spec: "Brewmaster" },
  66: { role: "tank", class: "Paladin", spec: "Protection" },

  // Healer specs
  65: { role: "healer", class: "Paladin", spec: "Holy" },
  256: { role: "healer", class: "Priest", spec: "Discipline" },
  257: { role: "healer", class: "Priest", spec: "Holy" },
  264: { role: "healer", class: "Shaman", spec: "Restoration" },
  105: { role: "healer", class: "Druid", spec: "Restoration" },
  270: { role: "healer", class: "Monk", spec: "Mistweaver" },

  // DPS specs - Physical
  71: { role: "dps", class: "Warrior", spec: "Arms" },
  72: { role: "dps", class: "Warrior", spec: "Fury" },
  70: { role: "dps", class: "Paladin", spec: "Retribution" },
  255: { role: "dps", class: "Hunter", spec: "Survival" },
  254: { role: "dps", class: "Hunter", spec: "Marksmanship" },
  253: { role: "dps", class: "Hunter", spec: "Beast Mastery" },
  259: { role: "dps", class: "Rogue", spec: "Assassination" },
  260: { role: "dps", class: "Rogue", spec: "Outlaw" },
  261: { role: "dps", class: "Rogue", spec: "Subtlety" },
  251: { role: "dps", class: "Death Knight", spec: "Frost" },
  252: { role: "dps", class: "Death Knight", spec: "Unholy" },
  263: { role: "dps", class: "Shaman", spec: "Enhancement" },
  269: { role: "dps", class: "Monk", spec: "Windwalker" },
  103: { role: "dps", class: "Druid", spec: "Feral" },

  // DPS specs - Magical
  262: { role: "dps", class: "Shaman", spec: "Elemental" },
  62: { role: "dps", class: "Mage", spec: "Arcane" },
  63: { role: "dps", class: "Mage", spec: "Fire" },
  64: { role: "dps", class: "Mage", spec: "Frost" },
  265: { role: "dps", class: "Warlock", spec: "Affliction" },
  266: { role: "dps", class: "Warlock", spec: "Demonology" },
  267: { role: "dps", class: "Warlock", spec: "Destruction" },
  258: { role: "dps", class: "Priest", spec: "Shadow" },
  102: { role: "dps", class: "Druid", spec: "Balance" },
};

// ========================================
// SPEC ICONS
// ========================================

export const SPEC_ICON_MAP = {
  // Mage
  "Mage|Frost":
    "https://wow.zamimg.com/images/wow/icons/large/spell_frost_frostbolt02.jpg",
  "Mage|Fire":
    "https://wow.zamimg.com/images/wow/icons/large/spell_fire_flamebolt.jpg",
  "Mage|Arcane":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_magicalsentry.jpg",

  // Paladin
  "Paladin|Holy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_holybolt.jpg",
  "Paladin|Protection":
    "https://wow.zamimg.com/images/wow/icons/large/ability_paladin_shieldofthetemplar.jpg",
  "Paladin|Retribution":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_auraoflight.jpg",

  // Death Knight
  "Death Knight|Blood":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_bloodpresence.jpg",
  "Death Knight|Frost":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_frostpresence.jpg",
  "Death Knight|Unholy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_deathknight_unholypresence.jpg",

  // Druid
  "Druid|Balance":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_starfall.jpg",
  "Druid|Feral":
    "https://wow.zamimg.com/images/wow/icons/large/ability_druid_catform.jpg",
  "Druid|Guardian":
    "https://wow.zamimg.com/images/wow/icons/large/ability_racial_bearform.jpg",
  "Druid|Restoration":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_healingtouch.jpg",

  // Hunter
  "Hunter|Beast Mastery":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_bestialdiscipline.jpg",
  "Hunter|Marksmanship":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_focusedaim.jpg",
  "Hunter|Survival":
    "https://wow.zamimg.com/images/wow/icons/large/ability_hunter_camouflage.jpg",

  // Monk
  "Monk|Brewmaster":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_brewmaster_spec.jpg",
  "Monk|Windwalker":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_windwalker_spec.jpg",
  "Monk|Mistweaver":
    "https://wow.zamimg.com/images/wow/icons/large/spell_monk_mistweaver_spec.jpg",

  // Priest
  "Priest|Discipline":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_powerwordshield.jpg",
  "Priest|Holy":
    "https://wow.zamimg.com/images/wow/icons/large/spell_holy_guardianspirit.jpg",
  "Priest|Shadow":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_shadowwordpain.jpg",

  // Rogue
  "Rogue|Subtlety":
    "https://wow.zamimg.com/images/wow/icons/large/ability_stealth.jpg",
  "Rogue|Assassination":
    "https://wow.zamimg.com/images/wow/icons/large/ability_rogue_eviscerate.jpg",
  "Rogue|Outlaw":
    "https://wow.zamimg.com/images/wow/icons/large/ability_backstab.jpg",

  // Shaman
  "Shaman|Elemental":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_lightning.jpg",
  "Shaman|Enhancement":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shaman_improvedstormstrike.jpg",
  "Shaman|Restoration":
    "https://wow.zamimg.com/images/wow/icons/large/spell_nature_magicimmunity.jpg",

  // Warlock
  "Warlock|Affliction":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_deathcoil.jpg",
  "Warlock|Destruction":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_rainoffire.jpg",
  "Warlock|Demonology":
    "https://wow.zamimg.com/images/wow/icons/large/spell_shadow_metamorphosis.jpg",

  // Warrior
  "Warrior|Arms":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_savageblow.jpg",
  "Warrior|Fury":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_innerrage.jpg",
  "Warrior|Protection":
    "https://wow.zamimg.com/images/wow/icons/large/ability_warrior_defensivestance.jpg",
};

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
};

export const CLASS_OPTIONS = [
  { value: "death_knight", label: "Death Knight" },
  { value: "druid", label: "Druid" },
  { value: "hunter", label: "Hunter" },
  { value: "mage", label: "Mage" },
  { value: "monk", label: "Monk" },
  { value: "paladin", label: "Paladin" },
  { value: "priest", label: "Priest" },
  { value: "rogue", label: "Rogue" },
  { value: "shaman", label: "Shaman" },
  { value: "warlock", label: "Warlock" },
  { value: "warrior", label: "Warrior" },
];

export const SPEC_OPTIONS = {
  death_knight: [
    { value: "blood", label: "Blood" },
    { value: "frost", label: "Frost" },
    { value: "unholy", label: "Unholy" },
  ],
  druid: [
    { value: "balance", label: "Balance" },
    { value: "feral", label: "Feral" },
    { value: "guardian", label: "Guardian" },
    { value: "restoration", label: "Restoration" },
  ],
  hunter: [
    { value: "beast_mastery", label: "Beast Mastery" },
    { value: "marksmanship", label: "Marksmanship" },
    { value: "survival", label: "Survival" },
  ],
  mage: [
    { value: "arcane", label: "Arcane" },
    { value: "fire", label: "Fire" },
    { value: "frost", label: "Frost" },
  ],
  monk: [{ value: "windwalker", label: "Windwalker" }],
  paladin: [{ value: "retribution", label: "Retribution" }],
  priest: [{ value: "shadow", label: "Shadow" }],
  rogue: [
    { value: "assassination", label: "Assassination" },
    { value: "outlaw", label: "Outlaw" },
    { value: "subtlety", label: "Subtlety" },
  ],
  shaman: [
    { value: "elemental", label: "Elemental" },
    { value: "enhancement", label: "Enhancement" },
    { value: "restoration", label: "Restoration" },
  ],
  warlock: [
    { value: "affliction", label: "Affliction" },
    { value: "demonology", label: "Demonology" },
    { value: "destruction", label: "Destruction" },
  ],
  warrior: [
    { value: "arms", label: "Arms" },
    { value: "fury", label: "Fury" },
    { value: "protection", label: "Protection" },
  ],
};

/**
 * Get class color for a given class name
 * @param {string} className - The class name (e.g., 'Warrior', 'death_knight')
 * @returns {string} Hex color code
 */
export function getClassColor(className) {
  // Normalize class name to lowercase with underscores
  const normalizedClass = className.toLowerCase().replace(/\s+/g, "_");
  return CLASS_COLORS[normalizedClass] || "#ffffff";
}

/**
 * get specialization info by spec ID
 * @param {number} specId - the specialization id
 * @returns {Object} Spec info object with role, class, and spec name
 */
export function getSpecInfo(specId) {
  return (
    SPEC_MAP[specId] || {
      role: "dps",
      class: "Unknown",
      spec: "Unknown",
    }
  );
}

/**
 * Get spec icon URL
 * @param {string} className - The class name
 * @param {string} specName - The spec name
 * @returns {string|null} Icon URL or null if not found
 */
export function getSpecIcon(className, specName) {
  if (className === "Unknown" || specName === "Unknown") {
    return null;
  }

  const iconKey = `${className}|${specName}`;
  return SPEC_ICON_MAP[iconKey] || null;
}

/**
 * get dungeon name by id
 * @param {string} dungeonId - the dungeon id
 * @returns {string} dungeon name or 'unknown dungeon'
 */
export function getDungeonName(dungeonId) {
  return DUNGEON_MAP[dungeonId] || "Unknown Dungeon";
}

/**
 * Convert dungeon name to URL slug
 * @param {string} dungeonName - The dungeon name
 * @returns {string} URL-safe slug
 */
export function dungeonNameToSlug(dungeonName) {
  return dungeonName.toLowerCase().replace(/[^a-z0-9]/g, "-");
}

/**
 * create a player element for display in challenge mode leaderboards
 * @param {Object} member - player member object
 * @returns {HTMLElement} player display element
 */
export function createPlayerElement(member) {
  // handle missing specialization data (deleted/transferred characters)
  const specId = member.specialization?.id;
  const spec = specId
    ? getSpecInfo(specId)
    : {
        role: "dps",
        class: "Unknown",
        spec: "Unknown",
      };

  // create container for icon + name
  const playerContainer = document.createElement("span");
  playerContainer.className = "player-container";
  playerContainer.style.cssText = `
    display: inline-flex !important;
    align-items: center !important;
    gap: 4px !important;
    margin-right: 8px !important;
  `;

  // Create spec icon if available
  const iconUrl = getSpecIcon(spec.class, spec.spec);
  if (iconUrl) {
    const iconImg = document.createElement("img");
    iconImg.src = iconUrl;
    iconImg.alt = `${spec.spec} ${spec.class}`;
    iconImg.className = "spec-icon";
    iconImg.style.cssText = `
      width: 16px !important;
      height: 16px !important;
      border-radius: 2px !important;
      flex-shrink: 0 !important;
    `;
    playerContainer.appendChild(iconImg);
  }

  // create name span
  const nameSpan = document.createElement("span");
  nameSpan.className = "player-name";
  nameSpan.style.cssText = `
    color: ${getClassColor(spec.class)} !important;
    font-weight: 600 !important;
    font-size: 0.9em !important;
  `;
  nameSpan.textContent = member.profile.name;
  playerContainer.appendChild(nameSpan);

  return playerContainer;
}
