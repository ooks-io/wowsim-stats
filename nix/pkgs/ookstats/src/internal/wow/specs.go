package wow

// SpecInfo represents class and spec information
type SpecInfo struct {
	ClassName string
	SpecName  string
}

// SpecByID maps spec IDs to their class and spec names
var SpecByID = map[int]SpecInfo{
	// Tanks
	73:  {ClassName: "Warrior", SpecName: "Protection"},
	104: {ClassName: "Druid", SpecName: "Guardian"},
	250: {ClassName: "Death Knight", SpecName: "Blood"},
	268: {ClassName: "Monk", SpecName: "Brewmaster"},
	66:  {ClassName: "Paladin", SpecName: "Protection"},
	// Healers
	105: {ClassName: "Druid", SpecName: "Restoration"},
	270: {ClassName: "Monk", SpecName: "Mistweaver"},
	65:  {ClassName: "Paladin", SpecName: "Holy"},
	256: {ClassName: "Priest", SpecName: "Discipline"},
	257: {ClassName: "Priest", SpecName: "Holy"},
	264: {ClassName: "Shaman", SpecName: "Restoration"},
	// DPS - Warriors
	71: {ClassName: "Warrior", SpecName: "Arms"},
	72: {ClassName: "Warrior", SpecName: "Fury"},
	// DPS - Paladins
	70: {ClassName: "Paladin", SpecName: "Retribution"},
	// DPS - Hunters
	253: {ClassName: "Hunter", SpecName: "Beast Mastery"},
	254: {ClassName: "Hunter", SpecName: "Marksmanship"},
	255: {ClassName: "Hunter", SpecName: "Survival"},
	// DPS - Rogues
	259: {ClassName: "Rogue", SpecName: "Assassination"},
	260: {ClassName: "Rogue", SpecName: "Outlaw"},
	261: {ClassName: "Rogue", SpecName: "Subtlety"},
	// DPS - Priests
	258: {ClassName: "Priest", SpecName: "Shadow"},
	// DPS - Death Knights
	251: {ClassName: "Death Knight", SpecName: "Frost"},
	252: {ClassName: "Death Knight", SpecName: "Unholy"},
	// DPS - Shamans
	262: {ClassName: "Shaman", SpecName: "Elemental"},
	263: {ClassName: "Shaman", SpecName: "Enhancement"},
	// DPS - Mages
	62: {ClassName: "Mage", SpecName: "Arcane"},
	63: {ClassName: "Mage", SpecName: "Fire"},
	64: {ClassName: "Mage", SpecName: "Frost"},
	// DPS - Warlocks
	265: {ClassName: "Warlock", SpecName: "Affliction"},
	266: {ClassName: "Warlock", SpecName: "Demonology"},
	267: {ClassName: "Warlock", SpecName: "Destruction"},
	// DPS - Monks
	269: {ClassName: "Monk", SpecName: "Windwalker"},
	// DPS - Druids
	102: {ClassName: "Druid", SpecName: "Balance"},
	103: {ClassName: "Druid", SpecName: "Feral"},
}

var specClassIDs = map[int]int{
	73:  1,
	71:  1,
	72:  1,
	66:  2,
	65:  2,
	70:  2,
	253: 3,
	254: 3,
	255: 3,
	259: 4,
	260: 4,
	261: 4,
	256: 5,
	257: 5,
	258: 5,
	250: 6,
	251: 6,
	252: 6,
	262: 7,
	263: 7,
	264: 7,
	62:  8,
	63:  8,
	64:  8,
	265: 9,
	266: 9,
	267: 9,
	268: 10,
	269: 10,
	270: 10,
	102: 11,
	103: 11,
	104: 11,
	105: 11,
}

// GetClassAndSpec returns the class and spec name for a given spec ID.
// Returns empty strings and false if the spec ID is not found.
func GetClassAndSpec(specID int) (className, specName string, ok bool) {
	info, exists := SpecByID[specID]
	if !exists {
		return "", "", false
	}
	return info.ClassName, info.SpecName, true
}

// FallbackClassAndSpec attempts to populate missing class/spec fields using the spec ID.
// If both className and specName are already set, returns them unchanged.
// If specID is nil, returns the original values.
func FallbackClassAndSpec(className, specName string, specID *int) (string, string) {
	// If both are already populated, nothing to do
	if className != "" && specName != "" {
		return className, specName
	}

	// If no spec ID available, can't fallback
	if specID == nil {
		return className, specName
	}

	// Try to get from spec ID
	cls, spec, ok := GetClassAndSpec(*specID)
	if !ok {
		return className, specName
	}

	// Fill in missing fields
	if className == "" {
		className = cls
	}
	if specName == "" {
		specName = spec
	}

	return className, specName
}

// GetClassIDForSpec returns the numeric class ID for a spec ID.
func GetClassIDForSpec(specID int) (int, bool) {
	classID, ok := specClassIDs[specID]
	return classID, ok
}
