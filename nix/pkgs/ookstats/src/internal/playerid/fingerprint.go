package playerid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

const (
	// Level90AchievementID is the ID for the Level 90 achievement.
	Level90AchievementID = 6193
	// Level85AchievementID is the ID for the Level 85 achievement.
	Level85AchievementID = 4826
)

// HeroicDungeonAchievementIDs enumerates the heroic MoP dungeon completion achievements.
var HeroicDungeonAchievementIDs = []int{
	6760, // Heroic: Scarlet Halls
	6761, // Heroic: Scarlet Monastery
	6762, // Heroic: Scholomance
	6763, // Heroic: Siege of Niuzao Temple
	6470, // Heroic: Shado-Pan Monastery
	6756, // Heroic: Mogu'shan Palace
	6758, // Heroic: Temple of the Jade Serpent
	6759, // Heroic: Gate of the Setting Sun
	6456, // Heroic: Stormstout Brewery
}

func init() {
	sort.Ints(HeroicDungeonAchievementIDs)
}

// FingerprintInput captures the required data for a canonical player identity.
type FingerprintInput struct {
	ClassID                 int
	Level85Timestamp        int64
	Level90Timestamp        int64
	EarliestHeroicTimestamp int64
}

// Validate ensures all required fields are populated.
func (fi FingerprintInput) Validate() error {
	if fi.ClassID <= 0 {
		return fmt.Errorf("missing class id")
	}
	if fi.Level85Timestamp <= 0 {
		return fmt.Errorf("missing level 85 timestamp")
	}
	if fi.Level90Timestamp <= 0 {
		return fmt.Errorf("missing level 90 timestamp")
	}
	if fi.EarliestHeroicTimestamp <= 0 {
		return fmt.Errorf("missing heroic dungeon timestamp")
	}
	return nil
}

// ComputeHash returns the SHA-256 hash representing the canonical fingerprint.
func ComputeHash(fi FingerprintInput) (string, error) {
	if err := fi.Validate(); err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%d:%d:%d:%d", fi.ClassID, fi.Level85Timestamp, fi.Level90Timestamp, fi.EarliestHeroicTimestamp)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}
