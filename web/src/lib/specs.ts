// Spec catalog — single source of truth for the 33 MoP specs in the order they
// appear across the site (canonical class+spec ordering, role-grouped within
// class). Used by the gear page's spec selector and slug routing.

export interface SpecEntry {
  id: number;
  className: string;
  specName: string;
  // URL slug, e.g. "monk-mistweaver". Includes the class so it disambiguates
  // shared spec names like Holy Paladin vs Holy Priest.
  slug: string;
  role: "tank" | "healer" | "dps";
}

function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/'/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

function entry(id: number, className: string, specName: string, role: SpecEntry["role"]): SpecEntry {
  return {
    id,
    className,
    specName,
    role,
    slug: `${slugify(className)}-${slugify(specName)}`,
  };
}

export const SPECS: SpecEntry[] = [
  // Death Knight
  entry(250, "Death Knight", "Blood", "tank"),
  entry(251, "Death Knight", "Frost", "dps"),
  entry(252, "Death Knight", "Unholy", "dps"),
  // Druid
  entry(104, "Druid", "Guardian", "tank"),
  entry(105, "Druid", "Restoration", "healer"),
  entry(102, "Druid", "Balance", "dps"),
  entry(103, "Druid", "Feral", "dps"),
  // Hunter
  entry(253, "Hunter", "Beast Mastery", "dps"),
  entry(254, "Hunter", "Marksmanship", "dps"),
  entry(255, "Hunter", "Survival", "dps"),
  // Mage
  entry(62, "Mage", "Arcane", "dps"),
  entry(63, "Mage", "Fire", "dps"),
  entry(64, "Mage", "Frost", "dps"),
  // Monk
  entry(268, "Monk", "Brewmaster", "tank"),
  entry(270, "Monk", "Mistweaver", "healer"),
  entry(269, "Monk", "Windwalker", "dps"),
  // Paladin
  entry(66, "Paladin", "Protection", "tank"),
  entry(65, "Paladin", "Holy", "healer"),
  entry(70, "Paladin", "Retribution", "dps"),
  // Priest
  entry(256, "Priest", "Discipline", "healer"),
  entry(257, "Priest", "Holy", "healer"),
  entry(258, "Priest", "Shadow", "dps"),
  // Rogue
  entry(259, "Rogue", "Assassination", "dps"),
  entry(260, "Rogue", "Combat", "dps"),
  entry(261, "Rogue", "Subtlety", "dps"),
  // Shaman
  entry(264, "Shaman", "Restoration", "healer"),
  entry(262, "Shaman", "Elemental", "dps"),
  entry(263, "Shaman", "Enhancement", "dps"),
  // Warlock
  entry(265, "Warlock", "Affliction", "dps"),
  entry(266, "Warlock", "Demonology", "dps"),
  entry(267, "Warlock", "Destruction", "dps"),
  // Warrior
  entry(73, "Warrior", "Protection", "tank"),
  entry(71, "Warrior", "Arms", "dps"),
  entry(72, "Warrior", "Fury", "dps"),
];

const BY_SLUG = new Map<string, SpecEntry>(SPECS.map((s) => [s.slug, s]));
const BY_ID = new Map<number, SpecEntry>(SPECS.map((s) => [s.id, s]));

export function specBySlug(slug: string | undefined | null): SpecEntry | null {
  if (!slug) return null;
  return BY_SLUG.get(slug.toLowerCase()) ?? null;
}

export function specById(id: number | string | undefined | null): SpecEntry | null {
  if (id === undefined || id === null) return null;
  const n = typeof id === "number" ? id : parseInt(id, 10);
  if (Number.isNaN(n)) return null;
  return BY_ID.get(n) ?? null;
}
