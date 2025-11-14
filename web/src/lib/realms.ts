// format: slug -> display name

export const REALM_DATA = {
  us: {
    atiesh: "Atiesh",
    myzrael: "Myzrael",
    "old-blanchy": "Old Blanchy",
    azuresong: "Azuresong",
    mankrik: "Mankrik",
    pagle: "Pagle",
    ashkandi: "Ashkandi",
    westfall: "Westfall",
    whitemane: "Whitemane",
    faerlina: "Faerlina",
    grobbulus: "Grobbulus",
    "bloodsail-buccaneers": "Bloodsail Buccaneers",
    "remulos-au": "Remulos (AU)",
    "arugal-au": "Arugal (AU)",
    "yojamba-au": "Yojamba (AU)",
    skyfury: "Skyfury",
    sulfuras: "Sulfuras",
    windseeker: "Windseeker",
    benediction: "Benediction",
    earthfury: "Earthfury",
    maladath: "Maladath",
    angerforge: "Angerforge",
    eranikus: "Eranikus",
    // nazgrim: "Nazgrim",
    // galakras: "Galakras",
    // raden: "Ra-den",
    // "lei-shen": "Lei Shen",
    // immerseus: "Immerseus",
  },
  eu: {
    everlook: "Everlook",
    auberdine: "Auberdine",
    lakeshire: "Lakeshire",
    chromie: "Chromie",
    "pyrewood-village": "Pyrewood Village",
    "mirage-raceway": "Mirage Raceway",
    razorfen: "Razorfen",
    "nethergarde-keep": "Nethergarde Keep",
    sulfuron: "Sulfuron",
    golemagg: "Golemagg",
    patchwerk: "Patchwerk",
    firemaw: "Firemaw",
    flamegor: "Flamegor",
    gehennas: "Gehennas",
    venoxis: "Venoxis",
    "hydraxian-waterlords": "Hydraxian Waterlords",
    mograine: "Mograine",
    amnennar: "Amnennar",
    ashbringer: "Ashbringer",
    transcendence: "Transcendence",
    earthshaker: "Earthshaker",
    giantstalker: "Giantstalker",
    mandokir: "Mandokir",
    thekal: "Thekal",
    jindo: "Jin'do",
    //shekzeer: "Shek'zeer",
    //garalon: "Garalon",
    //norushen: "Norushen",
    //hoptallus: "Hoptallus",
    // "ook-ook": "Ook Ook",
  },
  kr: {
    "shimmering-flats": "Shimmering Flats",
    lokholar: "Lokholar",
    iceblood: "Iceblood",
    ragnaros: "Ragnaros",
    frostmourne: "Frostmourne",
  },
  tw: {
    maraudon: "Maraudon",
    ivus: "Ivus",
    wushoolay: "Wushoolay",
    zeliek: "Zeliek",
    "arathi-basin": "Arathi Basin",
    murloc: "Murloc",
    golemagg: "Golemagg",
    windseeker: "Windseeker",
  },
} as const;

export type Region = keyof typeof REALM_DATA;
export type RealmSlug<T extends Region> = keyof (typeof REALM_DATA)[T];

// Realm merge mapping: child realm slug -> parent leaderboard realm slug
export const REALM_PARENT_MAP: Partial<Record<Region, Record<string, string>>> =
  {
    us: {
      nazgrim: "pagle",
      galakras: "pagle",
      "ra-den": "pagle",
      raden: "pagle",
      "lei-shen": "pagle",
      leishen: "pagle",
      immerseus: "pagle",
    },
    eu: {
      shekzeer: "mirage-raceway",
      garalon: "mirage-raceway",
      norushen: "mirage-raceway",
      hoptallus: "mirage-raceway",
      "ook-ook": "everlook",
      ookook: "everlook",
    },
  };

function normalizeSlug(value?: string | null): string | undefined {
  if (!value) return undefined;
  return value.toLowerCase();
}

function normalizeRegion(value?: string | null): Region | undefined {
  if (!value) return undefined;
  const regionKey = value.toLowerCase() as Region;
  return regionKey;
}

export function getEffectiveRealmSlug(
  region?: string | null,
  realmSlug?: string | null,
): string | undefined {
  const normalizedRealm = normalizeSlug(realmSlug);
  if (!normalizedRealm) return undefined;
  const normalizedRegion = normalizeRegion(region);
  if (!normalizedRegion) return normalizedRealm;

  const regionMap = REALM_PARENT_MAP[normalizedRegion];
  if (regionMap && regionMap[normalizedRealm]) {
    return regionMap[normalizedRealm];
  }
  return normalizedRealm;
}

export function areRealmsEquivalent(
  region?: string | null,
  realmA?: string | null,
  realmB?: string | null,
): boolean {
  const effectiveA = getEffectiveRealmSlug(region, realmA);
  const effectiveB = getEffectiveRealmSlug(region, realmB);
  if (!effectiveA || !effectiveB) {
    return effectiveA === effectiveB;
  }
  return effectiveA === effectiveB;
}
