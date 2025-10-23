package blizzard

// GetHardcodedPeriodAndDungeons returns the primary period ID and dungeon list
// hardcoded values since the index endpoint is broken - most records are in period 1026
func GetHardcodedPeriodAndDungeons() (string, []DungeonInfo) {
	periodID := "1034"

	dungeons := []DungeonInfo{
		{ID: 2, Name: "Temple of the Jade Serpent", Slug: "temple-of-the-jade-serpent"},
		{ID: 56, Name: "Stormstout Brewery", Slug: "stormstout-brewery"},
		{ID: 57, Name: "Gate of the Setting Sun", Slug: "gate-of-the-setting-sun"},
		{ID: 58, Name: "Shado-Pan Monastery", Slug: "shado-pan-monastery"},
		{ID: 59, Name: "Siege of Niuzao Temple", Slug: "siege-of-niuzao-temple"},
		{ID: 60, Name: "Mogu'shan Palace", Slug: "mogu-shan-palace"},
		{ID: 76, Name: "Scholomance", Slug: "scholomance"},
		{ID: 77, Name: "Scarlet Halls", Slug: "scarlet-halls"},
		{ID: 78, Name: "Scarlet Monastery", Slug: "scarlet-monastery"},
	}

	return periodID, dungeons
}

// GetFallbackPeriods returns periods to try in order for multi-period fallback
// based on period analysis results showing sparse data distribution
func GetFallbackPeriods() []string {
	return []string{"1034", "1033", "1032", "1031", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023", "1022", "1021", "1020"}
}

// GetGlobalPeriods returns a global sweep order newest -> oldest
func GetGlobalPeriods() []string {
	return []string{"1034", "1033", "1032", "1031", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023"}
}

// GetRegionFallbackPeriods prioritizes the period order per region based on observed data
// This reduces 404 churn before finding data.
func GetRegionFallbackPeriods(region string) []string {
	switch region {
	case "eu":
		return []string{"1034", "1033", "1032", "1030", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023", "1022", "1021", "1020"}
	case "us":
		return []string{"1034", "1033", "1032", "1031", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023", "1022", "1021", "1020"}
	case "kr":
		return []string{"1034", "1033", "1032", "1031", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023", "1022", "1021", "1020"}
	case "tw":
		return []string{"1034", "1033", "1032", "1031", "1030", "1029", "1028", "1027", "1026", "1025", "1024", "1023", "1022", "1021", "1020"}
	default:
		return GetFallbackPeriods()
	}
}

// GetAllRealms returns the complete realm configuration
// this data comes from nix/api/realm.nix
func GetAllRealms() map[string]RealmInfo {
	return map[string]RealmInfo{
		// us realms
		"atiesh":               {ID: 4372, Region: "us", Name: "Atiesh", Slug: "atiesh"},
		"myzrael":              {ID: 4373, Region: "us", Name: "Myzrael", Slug: "myzrael"},
		"old-blanchy":          {ID: 4374, Region: "us", Name: "Old Blanchy", Slug: "old-blanchy"},
		"azuresong":            {ID: 4376, Region: "us", Name: "Azuresong", Slug: "azuresong"},
		"mankrik":              {ID: 4384, Region: "us", Name: "Mankrik", Slug: "mankrik"},
		"pagle":                {ID: 4385, Region: "us", Name: "Pagle", Slug: "pagle"},
		"ashkandi":             {ID: 4387, Region: "us", Name: "Ashkandi", Slug: "ashkandi"},
		"westfall":             {ID: 4388, Region: "us", Name: "Westfall", Slug: "westfall"},
		"whitemane":            {ID: 4395, Region: "us", Name: "Whitemane", Slug: "whitemane"},
		"faerlina":             {ID: 4408, Region: "us", Name: "Faerlina", Slug: "faerlina"},
		"grobbulus":            {ID: 4647, Region: "us", Name: "Grobbulus", Slug: "grobbulus"},
		"bloodsail-buccaneers": {ID: 4648, Region: "us", Name: "Bloodsail Buccaneers", Slug: "bloodsail-buccaneers"},
		// OCE realms are served under US region with -au slug suffix
		"remulos-au":  {ID: 4667, Region: "us", Name: "Remulos (AU)", Slug: "remulos-au"},
		"arugal-au":   {ID: 4669, Region: "us", Name: "Arugal (AU)", Slug: "arugal-au"},
		"yojamba-au":  {ID: 4670, Region: "us", Name: "Yojamba (AU)", Slug: "yojamba-au"},
		"skyfury":     {ID: 4725, Region: "us", Name: "Skyfury", Slug: "skyfury"},
		"sulfuras":    {ID: 4726, Region: "us", Name: "Sulfuras", Slug: "sulfuras"},
		"windseeker":  {ID: 4727, Region: "us", Name: "Windseeker", Slug: "windseeker"},
		"benediction": {ID: 4728, Region: "us", Name: "Benediction", Slug: "benediction"},
		"earthfury":   {ID: 4731, Region: "us", Name: "Earthfury", Slug: "earthfury"},
		"maladath":    {ID: 4738, Region: "us", Name: "Maladath", Slug: "maladath"},
		"angerforge":  {ID: 4795, Region: "us", Name: "Angerforge", Slug: "angerforge"},
		"eranikus":    {ID: 4800, Region: "us", Name: "Eranikus", Slug: "eranikus"},
		// New US realms (MoP Classic launch wave)
		"nazgrim":   {ID: 6359, Region: "us", Name: "Nazgrim", Slug: "nazgrim"},
		"galakras":  {ID: 6360, Region: "us", Name: "Galakras", Slug: "galakras"},
		"raden":     {ID: 6361, Region: "us", Name: "Ra-den", Slug: "raden"},
		"lei-shen":  {ID: 6362, Region: "us", Name: "Lei Shen", Slug: "lei-shen"},
		"immerseus": {ID: 6363, Region: "us", Name: "Immerseus", Slug: "immerseus"},

		// eu realms
		"everlook":             {ID: 4440, Region: "eu", Name: "Everlook", Slug: "everlook"},
		"auberdine":            {ID: 4441, Region: "eu", Name: "Auberdine", Slug: "auberdine"},
		"lakeshire":            {ID: 4442, Region: "eu", Name: "Lakeshire", Slug: "lakeshire"},
		"chromie":              {ID: 4452, Region: "eu", Name: "Chromie", Slug: "chromie"},
		"pyrewood-village":     {ID: 4453, Region: "eu", Name: "Pyrewood Village", Slug: "pyrewood-village"},
		"mirage-raceway":       {ID: 4454, Region: "eu", Name: "Mirage Raceway", Slug: "mirage-raceway"},
		"razorfen":             {ID: 4455, Region: "eu", Name: "Razorfen", Slug: "razorfen"},
		"nethergarde-keep":     {ID: 4456, Region: "eu", Name: "Nethergarde Keep", Slug: "nethergarde-keep"},
		"sulfuron":             {ID: 4464, Region: "eu", Name: "Sulfuron", Slug: "sulfuron"},
		"golemagg":             {ID: 4465, Region: "eu", Name: "Golemagg", Slug: "golemagg"},
		"patchwerk":            {ID: 4466, Region: "eu", Name: "Patchwerk", Slug: "patchwerk"},
		"firemaw":              {ID: 4467, Region: "eu", Name: "Firemaw", Slug: "firemaw"},
		"flamegor":             {ID: 4474, Region: "eu", Name: "Flamegor", Slug: "flamegor"},
		"gehennas":             {ID: 4476, Region: "eu", Name: "Gehennas", Slug: "gehennas"},
		"venoxis":              {ID: 4477, Region: "eu", Name: "Venoxis", Slug: "venoxis"},
		"hydraxian-waterlords": {ID: 4678, Region: "eu", Name: "Hydraxian Waterlords", Slug: "hydraxian-waterlords"},
		"mograine":             {ID: 4701, Region: "eu", Name: "Mograine", Slug: "mograine"},
		"amnennar":             {ID: 4703, Region: "eu", Name: "Amnennar", Slug: "amnennar"},
		"ashbringer":           {ID: 4742, Region: "eu", Name: "Ashbringer", Slug: "ashbringer"},
		"transcendence":        {ID: 4745, Region: "eu", Name: "Transcendence", Slug: "transcendence"},
		"earthshaker":          {ID: 4749, Region: "eu", Name: "Earthshaker", Slug: "earthshaker"},
		"giantstalker":         {ID: 4811, Region: "eu", Name: "Giantstalker", Slug: "giantstalker"},
		"mandokir":             {ID: 4813, Region: "eu", Name: "Mandokir", Slug: "mandokir"},
		"thekal":               {ID: 4815, Region: "eu", Name: "Thekal", Slug: "thekal"},
		"jindo":                {ID: 4816, Region: "eu", Name: "Jin'do", Slug: "jindo"},
		// New EU realms (MoP Classic launch wave)
		"shekzeer":  {ID: 6364, Region: "eu", Name: "Shek'zeer", Slug: "shekzeer"},
		"garalon":   {ID: 6365, Region: "eu", Name: "Garalon", Slug: "garalon"},
		"norushen":  {ID: 6366, Region: "eu", Name: "Norushen", Slug: "norushen"},
		"hoptallus": {ID: 6367, Region: "eu", Name: "Hoptallus", Slug: "hoptallus"},
		"ook-ook":   {ID: 6368, Region: "eu", Name: "Ook Ook", Slug: "ook-ook"},

		// kr realms
		"shimmering-flats": {ID: 4417, Region: "kr", Name: "Shimmering Flats", Slug: "shimmering-flats"},
		"lokholar":         {ID: 4419, Region: "kr", Name: "Lokholar", Slug: "lokholar"},
		"iceblood":         {ID: 4420, Region: "kr", Name: "Iceblood", Slug: "iceblood"},
		"ragnaros":         {ID: 4421, Region: "kr", Name: "Ragnaros", Slug: "ragnaros"},
		"frostmourne":      {ID: 4840, Region: "kr", Name: "Frostmourne", Slug: "frostmourne"},

		// tw realms (exclude PTR realms TW4 CWOW GMSS 1/2)
		"maraudon":     {ID: 4485, Region: "tw", Name: "Maraudon", Slug: "maraudon"},
		"ivus":         {ID: 4487, Region: "tw", Name: "Ivus", Slug: "ivus"},
		"wushoolay":    {ID: 4488, Region: "tw", Name: "Wushoolay", Slug: "wushoolay"},
		"zeliek":       {ID: 4489, Region: "tw", Name: "Zeliek", Slug: "zeliek"},
		"arathi-basin": {ID: 5740, Region: "tw", Name: "Arathi Basin", Slug: "arathi-basin"},
		"murloc":       {ID: 5741, Region: "tw", Name: "Murloc", Slug: "murloc"},
		// Use disambiguated map keys for slugs colliding across regions; Slug field remains the Blizzard slug.
		"golemagg-tw":   {ID: 5742, Region: "tw", Name: "Golemagg", Slug: "golemagg"},
		"windseeker-tw": {ID: 5743, Region: "tw", Name: "Windseeker", Slug: "windseeker"},
	}
}

// NormalizeRealmSlug applies known region-specific realm slug renames.
// Example: US OCE realms moved to -au slugs (e.g., arugal -> arugal-au).
func NormalizeRealmSlug(region, slug string) string {
	if region == "us" {
		switch slug {
		case "arugal":
			return "arugal-au"
		case "remulos":
			return "remulos-au"
		case "yojamba":
			return "yojamba-au"
		}
	}
	return slug
}
