package utils

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Slugify converts a string to a URL-friendly slug
func Slugify(text string) string {
	// convert to lowercase
	text = strings.ToLower(text)

	// replace spaces, apostrophes, and non-word characters with dashes
	re := regexp.MustCompile(`[\s'\W]+`)
	text = re.ReplaceAllString(text, "-")

	// trim leading and trailing dashes
	text = strings.Trim(text, "-")

	return text
}

// ComputeTeamSignature generates a team signature from a list of player IDs
// returns a sorted, comma-separated string of player IDs
func ComputeTeamSignature(playerIDs []int) string {
	if len(playerIDs) == 0 {
		return ""
	}

	// sort integers first
	sort.Ints(playerIDs)

	// convert to strings
	strIDs := make([]string, len(playerIDs))
	for i, id := range playerIDs {
		strIDs[i] = strconv.Itoa(id)
	}

	return strings.Join(strIDs, ",")
}

// CalculatePercentileBracket determines the WoW quality bracket based on ranking percentile
// mapping: #1 = artifact, top 5% = legendary, top 20% = epic, top 40% = rare, top 60% = uncommon, top 80% = common
func CalculatePercentileBracket(ranking int, totalCount int) string {
	if totalCount <= 0 || ranking <= 0 || ranking > totalCount {
		return "common"
	}

	// rank 1 always gets artifact
	if ranking == 1 {
		return "artifact"
	}

	// Calculate percentile (what percentage of results are worse than this ranking)
	percentile := float64(totalCount-ranking) / float64(totalCount) * 100

	// apply quality thresholds
	if percentile >= 95.0 { // top 5%
		return "legendary"
	} else if percentile >= 80.0 { // top 20%
		return "epic"
	} else if percentile >= 60.0 { // top 40%
		return "rare"
	} else if percentile >= 40.0 { // top 60%
		return "uncommon"
	} else { // bottom 20%
		return "common"
	}
}

