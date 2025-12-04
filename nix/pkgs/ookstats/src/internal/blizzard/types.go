package blizzard

// RealmInfo represents a realm
type RealmInfo struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Region          string `json:"region"`
	Slug            string `json:"slug"`
	ParentRealmSlug string `json:"parent_realm_slug,omitempty"`
}

// DungeonInfo represents a challenge mode dungeon
type DungeonInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// LeaderboardResponse is the top-level response from the mythic leaderboard API
type LeaderboardResponse struct {
	LeadingGroups        []ChallengeRun `json:"leading_groups"`
	Period               int            `json:"period"`
	PeriodStartTimestamp int64          `json:"period_start_timestamp"`
	PeriodEndTimestamp   int64          `json:"period_end_timestamp"`
}

// ChallengeRun represents a single challenge mode run
type ChallengeRun struct {
	Duration           int      `json:"duration"`
	CompletedTimestamp int64    `json:"completed_timestamp"`
	KeystoneLevel      int      `json:"keystone_level"`
	Members            []Member `json:"members"`
}

// Member represents a player in a challenge mode run
// supports both old format (with nested profile) and new optimized format
type Member struct {
	// New optimized format fields
	ID        *int    `json:"id,omitempty"`
	Name      *string `json:"name,omitempty"`
	RealmSlug *string `json:"realm_slug,omitempty"`
	SpecID    *int    `json:"spec_id,omitempty"`
	Faction   *string `json:"faction,omitempty"`

	// old format with nested profile
	Profile        *Profile        `json:"profile,omitempty"`
	Specialization *Specialization `json:"specialization,omitempty"`
	FactionType    *FactionType    `json:"faction,omitempty"`
}

// Profile represents nested profile data in the old API format
type Profile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Realm Realm  `json:"realm"`
}

// Realm represents realm data in the old API format
type Realm struct {
	Slug string `json:"slug"`
}

// Specialization represents spec data
type Specialization struct {
	ID int `json:"id"`
}

// FactionType represents faction data
type FactionType struct {
	Type string `json:"type"`
}

// GetPlayerID extracts player ID from either format
func (m *Member) GetPlayerID() (int, bool) {
	if m.ID != nil {
		return *m.ID, true
	}
	if m.Profile != nil {
		return m.Profile.ID, true
	}
	return 0, false
}

// GetPlayerName extracts player name from either format
func (m *Member) GetPlayerName() (string, bool) {
	if m.Name != nil {
		return *m.Name, true
	}
	if m.Profile != nil {
		return m.Profile.Name, true
	}
	return "", false
}

// GetRealmSlug extracts realm slug from either format
func (m *Member) GetRealmSlug() (string, bool) {
	if m.RealmSlug != nil {
		return *m.RealmSlug, true
	}
	if m.Profile != nil {
		return m.Profile.Realm.Slug, true
	}
	return "", false
}

// GetSpecID extracts spec ID from either format
func (m *Member) GetSpecID() (int, bool) {
	if m.SpecID != nil {
		return *m.SpecID, true
	}
	if m.Specialization != nil {
		return m.Specialization.ID, true
	}
	return 0, false
}

// GetFaction extracts faction from either format
func (m *Member) GetFaction() (string, bool) {
	if m.Faction != nil {
		return *m.Faction, true
	}
	if m.FactionType != nil {
		return m.FactionType.Type, true
	}
	return "", false
}

// player profile api types

// CharacterSummaryResponse represents the character summary from the Profile API
type CharacterSummaryResponse struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Level              int             `json:"level"`
	Race               CharacterRace   `json:"race"`
	CharacterClass     CharacterClass  `json:"character_class"`
	ActiveSpec         CharacterSpec   `json:"active_spec"`
	Gender             CharacterGender `json:"gender"`
	Guild              *CharacterGuild `json:"guild,omitempty"`
	AverageItemLevel   int             `json:"average_item_level"`
	EquippedItemLevel  int             `json:"equipped_item_level"`
	LastLoginTimestamp *int64          `json:"last_login_timestamp,omitempty"`
}

// CharacterRace represents race information
type CharacterRace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CharacterClass represents class information
type CharacterClass struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CharacterSpec represents specialization information
type CharacterSpec struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CharacterGender represents gender information
type CharacterGender struct {
	Type string `json:"type"`
}

// CharacterGuild represents guild information
type CharacterGuild struct {
	Name string `json:"name"`
}

// CharacterEquipmentResponse represents the equipment response from the Profile API
type CharacterEquipmentResponse struct {
	EquippedItems []EquippedItem `json:"equipped_items"`
}

// EquippedItem represents a single equipped item
type EquippedItem struct {
	Item         ItemInfo          `json:"item"`
	Slot         ItemSlot          `json:"slot"`
	Name         string            `json:"name"`
	Quality      ItemQuality       `json:"quality"`
	UpgradeID    *int              `json:"upgrade_id,omitempty"`
	Enchantments []ItemEnchantment `json:"enchantments,omitempty"`
}

// ItemInfo represents basic item information
type ItemInfo struct {
	ID int `json:"id"`
}

// ItemSlot represents the equipment slot
type ItemSlot struct {
	Type string `json:"type"`
}

// ItemQuality represents item quality
type ItemQuality struct {
	Type string `json:"type"`
}

// CharacterAchievementsResponse represents the achievements summary payload.
type CharacterAchievementsResponse struct {
	Achievements []CharacterAchievement `json:"achievements"`
	Character    CharacterReference     `json:"character"`
}

// CharacterAchievement represents an individual achievement entry.
type CharacterAchievement struct {
	ID                 int                          `json:"id"`
	CompletedTimestamp *int64                       `json:"completed_timestamp,omitempty"`
	Criteria           CharacterAchievementCriteria `json:"criteria"`
}

// CharacterAchievementCriteria captures completion metadata.
type CharacterAchievementCriteria struct {
	ID          int  `json:"id"`
	IsCompleted bool `json:"is_completed"`
}

// CharacterReference includes the canonical ID + realm info for a character.
type CharacterReference struct {
	Name  string              `json:"name"`
	ID    int                 `json:"id"`
	Realm CharacterRealmBrief `json:"realm"`
}

// CharacterRealmBrief is a slim realm reference within profile APIs.
type CharacterRealmBrief struct {
	Name string `json:"name,omitempty"`
	ID   int    `json:"id,omitempty"`
	Slug string `json:"slug"`
}

// CharacterStatusResponse represents the /status payload for a character.
type CharacterStatusResponse struct {
	IsValid   bool               `json:"is_valid"`
	Reason    string             `json:"reason,omitempty"`
	Character CharacterReference `json:"character"`
}

// ItemEnchantment represents an enchantment, gem, or tinker
type ItemEnchantment struct {
	EnchantmentID   *int         `json:"enchantment_id,omitempty"`
	EnchantmentSlot *EnchantSlot `json:"enchantment_slot,omitempty"`
	DisplayString   *string      `json:"display_string,omitempty"`
	SourceItem      *SourceItem  `json:"source_item,omitempty"`
	Spell           *SpellInfo   `json:"spell,omitempty"`
}

// EnchantSlot represents the enchantment slot
type EnchantSlot struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

// SourceItem represents the source item for gems/enchants
type SourceItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SpellInfo represents spell information for enchants
type SpellInfo struct {
	Spell SpellDetail `json:"spell"`
}

// SpellDetail represents detailed spell information
type SpellDetail struct {
	ID int `json:"id"`
}

// CharacterMediaResponse represents the media response from the Profile API
type CharacterMediaResponse struct {
	Assets []MediaAsset `json:"assets"`
}

// MediaAsset represents a media asset (avatar, etc.)
type MediaAsset struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PeriodDetailResponse represents the response from a specific period detail API
type PeriodDetailResponse struct {
	ID             int   `json:"id"`
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

// SeasonIndexResponse represents the response from the mythic keystone season index API
type SeasonIndexResponse struct {
	Seasons []struct {
		Key struct {
			Href string `json:"href"`
		} `json:"key"`
		ID int `json:"id"`
	} `json:"seasons"`
	CurrentSeason struct {
		Key struct {
			Href string `json:"href"`
		} `json:"key"`
		ID int `json:"id"`
	} `json:"current_season"`
}

// SeasonDetailResponse represents the response from a specific season detail API
type SeasonDetailResponse struct {
	ID             int    `json:"id"`
	StartTimestamp int64  `json:"start_timestamp"`
	SeasonName     string `json:"season_name"`
	Periods        []struct {
		Key struct {
			Href string `json:"href"`
		} `json:"key"`
		ID int `json:"id"`
	} `json:"periods"`
}
