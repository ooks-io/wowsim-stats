// Utilities for building Simulation SPEC options

type SpecOption = { value: string | number; label: string };
type SpecOptionsByClass = Record<string, SpecOption[]>;

// Only expose supported DPS specs (exclude tanks/healers and feral druid)
export const ALLOWED_SPEC_IDS = new Set([
  // Death Knight
  '251', '252',
  // Druid (no feral 103)
  '102',
  // Hunter
  '253', '254', '255',
  // Mage
  '62', '63', '64',
  // Monk (only Windwalker)
  '269',
  // Paladin (only Retribution)
  '70',
  // Priest (only Shadow)
  '258',
  // Rogue
  '259', '260', '261',
  // Shaman (no Restoration 264)
  '262', '263',
  // Warlock
  '265', '266', '267',
  // Warrior (no Protection 73)
  '71', '72',
]);

function labelToSlug(label: string): string {
  return String(label || '')
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '_');
}

// Pure function: transforms canonical SPEC_OPTIONS into filtered + slugified options
export function buildSimSpecOptions(opts: SpecOptionsByClass): SpecOptionsByClass {
  const out: SpecOptionsByClass = {};
  for (const [cls, arr] of Object.entries(opts || {})) {
    const filtered = (arr || []).filter((o) => ALLOWED_SPEC_IDS.has(String(o.value)));
    if (filtered.length === 0) continue; // hide classes with no supported specs
    out[cls] = filtered.map((o) => ({ value: labelToSlug(String(o.label)), label: o.label }));
  }
  return out;
}

