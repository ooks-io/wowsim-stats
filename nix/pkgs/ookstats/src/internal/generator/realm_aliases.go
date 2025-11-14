package generator

import "strings"

// mergedRealmParents maps child realm slugs to their parent leaderboard realm per region.
var mergedRealmParents = map[string]map[string]string{
	"us": {
		"nazgrim":   "pagle",
		"galakras":  "pagle",
		"raden":     "pagle",
		"ra-den":    "pagle",
		"lei-shen":  "pagle",
		"leishen":   "pagle",
		"immerseus": "pagle",
	},
	"eu": {
		"shekzeer":  "mirage-raceway",
		"garalon":   "mirage-raceway",
		"norushen":  "mirage-raceway",
		"hoptallus": "mirage-raceway",
		"hotallus":  "mirage-raceway",
		"ook-ook":   "everlook",
		"ookook":    "everlook",
	},
}

type realmGroup struct {
	Parent string
	Slugs  []string
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func normalizeRegion(region string) string {
	return strings.ToLower(strings.TrimSpace(region))
}

// effectiveRealmSlug returns the parent leaderboard slug for a given region/realm slug.
func effectiveRealmSlug(region, slug string) string {
	normalizedRegion := normalizeRegion(region)
	normalizedSlug := normalizeSlug(slug)
	if normalizedSlug == "" {
		return ""
	}

	if parents, ok := mergedRealmParents[normalizedRegion]; ok {
		if parent, ok := parents[normalizedSlug]; ok {
			return parent
		}
	}
	return normalizedSlug
}

// realmGroupSlugs returns all realm slugs (parent + children) that belong to the same leaderboard group.
func realmGroupSlugs(region, slug string) []string {
	normalizedRegion := normalizeRegion(region)
	parent := effectiveRealmSlug(region, slug)
	if parent == "" {
		if s := normalizeSlug(slug); s != "" {
			return []string{s}
		}
		return nil
	}

	unique := make(map[string]struct{})
	ordered := make([]string, 0, 4)
	add := func(value string) {
		if value == "" {
			return
		}
		if _, exists := unique[value]; exists {
			return
		}
		unique[value] = struct{}{}
		ordered = append(ordered, value)
	}

	add(parent)
	if parents, ok := mergedRealmParents[normalizedRegion]; ok {
		for child, p := range parents {
			if p == parent {
				add(child)
			}
		}
	}
	add(normalizeSlug(slug))

	return ordered
}

// groupRealmSlugs groups a list of realm slugs by their effective leaderboard parent.
func groupRealmSlugs(region string, slugs []string) []realmGroup {
	normalizedRegion := normalizeRegion(region)
	index := make(map[string]int)
	var groups []realmGroup

	for _, slug := range slugs {
		parent := effectiveRealmSlug(normalizedRegion, slug)
		if parent == "" {
			continue
		}
		if idx, ok := index[parent]; ok {
			groups[idx].Slugs = appendUnique(groups[idx].Slugs, normalizeSlug(slug))
		} else {
			groups = append(groups, realmGroup{
				Parent: parent,
				Slugs:  []string{normalizeSlug(slug)},
			})
			index[parent] = len(groups) - 1
		}
	}

	for i := range groups {
		groups[i].Slugs = appendUnique(groups[i].Slugs, groups[i].Parent)
	}

	return groups
}

func appendUnique(list []string, value string) []string {
	if value == "" {
		return list
	}
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func makeSQLPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func stringSliceToInterface(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
