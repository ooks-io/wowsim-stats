package playerid

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"ookstats/internal/blizzard"
)

//go:embed account_fingerprint_trusted.json
var trustedJSON []byte

// regenerate via `ookstats investigate-account --pairs-file` then rebuild
var trustedAccountWideIDs map[int]struct{}

func init() {
	type entry struct {
		ID int `json:"id"`
	}
	var rep struct {
		Trusted []entry `json:"trusted"`
	}
	if err := json.Unmarshal(trustedJSON, &rep); err != nil {
		panic(fmt.Sprintf("playerid: parse embedded trusted set: %v", err))
	}
	trustedAccountWideIDs = make(map[int]struct{}, len(rep.Trusted))
	for _, e := range rep.Trusted {
		trustedAccountWideIDs[e.ID] = struct{}{}
	}
	if len(trustedAccountWideIDs) == 0 {
		panic("playerid: embedded trusted set is empty")
	}
}

type TrustedTuple struct {
	ID int   `json:"id"`
	Ts int64 `json:"ts"`
}

func TrustedAccountWideIDCount() int {
	return len(trustedAccountWideIDs)
}

// sorted by id for deterministic downstream serialisation
func ExtractTrustedTuples(resp *blizzard.CharacterAchievementsResponse) []TrustedTuple {
	if resp == nil {
		return nil
	}
	out := make([]TrustedTuple, 0, len(trustedAccountWideIDs))
	for _, a := range resp.Achievements {
		if a.CompletedTimestamp == nil {
			continue
		}
		if _, ok := trustedAccountWideIDs[a.ID]; !ok {
			continue
		}
		out = append(out, TrustedTuple{ID: a.ID, Ts: *a.CompletedTimestamp})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
