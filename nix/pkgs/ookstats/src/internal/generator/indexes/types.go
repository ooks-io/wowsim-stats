package indexes

import "time"

// Common types

type Link struct {
	Href string `json:"href"`
}

type IndexMetadata struct {
	TotalCount  int    `json:"total_count"`
	LastUpdated string `json:"last_updated"`
}

func NewIndexMetadata(count int) IndexMetadata {
	return IndexMetadata{
		TotalCount:  count,
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
	}
}

// Root Index

type RootIndexLinks struct {
	Self Link `json:"self"`
}

type RootIndex struct {
	Links       RootIndexLinks    `json:"_links"`
	Indexes     map[string]string `json:"indexes"`
	Endpoints   map[string]string `json:"endpoints"`
	APIVersion  string            `json:"api_version"`
	LastUpdated string            `json:"last_updated"`
}

// Seasons Index

type SeasonLinks struct {
	Self   Link `json:"self"`
	Scopes Link `json:"scopes"`
}

type SeasonData struct {
	ID             int         `json:"id"`
	Name           string      `json:"name"`
	StartTimestamp *int64      `json:"start_timestamp"`
	EndTimestamp   *int64      `json:"end_timestamp"`
	IsCurrent      bool        `json:"is_current"`
	Links          SeasonLinks `json:"_links"`
}

type SeasonsIndex struct {
	Data     []SeasonData  `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Season Scope Index (lists: global, us, eu, kr, tw, players)

type ScopeLinks struct {
	Leaderboard Link `json:"leaderboard"`
}

type ScopeData struct {
	Scope string     `json:"scope"`
	Links ScopeLinks `json:"_links"`
}

type SeasonScopeIndex struct {
	Data     []ScopeData   `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Dungeons Index (for global scope)

type DungeonLinks struct {
	Leaderboard Link `json:"leaderboard"`
}

type DungeonData struct {
	ID                 int          `json:"id"`
	Slug               string       `json:"slug"`
	Name               string       `json:"name"`
	ShortName          string       `json:"short_name,omitempty"`
	MapChallengeModeID *int         `json:"map_challenge_mode_id"`
	Links              DungeonLinks `json:"_links"`
}

type DungeonsIndex struct {
	Data     []DungeonData `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Regional Realms Index (lists realms + "all" endpoint for a region)

type RealmLinks struct {
	Dungeons Link `json:"dungeons"`
}

type RealmData struct {
	Slug             string     `json:"slug"`
	Name             string     `json:"name"`
	ConnectedRealmID *int       `json:"connected_realm_id"`
	ParentRealm      *string    `json:"parent_realm"`
	PlayerCount      int        `json:"player_count"`
	Links            RealmLinks `json:"_links"`
}

type RegionalAllLink struct {
	Href string `json:"href"`
	Note string `json:"note"`
}

type RegionalRealmsIndex struct {
	All      RegionalAllLink `json:"all"`
	Data     []RealmData     `json:"data"`
	Metadata IndexMetadata   `json:"metadata"`
}

// Realm Dungeons Index (dungeons available for a specific realm)

type RealmDungeonsIndex struct {
	Data     []DungeonData `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Players Scope Index (lists: global, regional, realm, class)

type PlayerScopeLinks struct {
	Leaderboard Link `json:"leaderboard"`
}

type PlayerScopeData struct {
	Scope string           `json:"scope"`
	Links PlayerScopeLinks `json:"_links"`
}

type PlayersScopeIndex struct {
	Data     []PlayerScopeData `json:"data"`
	Metadata IndexMetadata     `json:"metadata"`
}

// Simple scope lists (just region codes)

type RegionData struct {
	Region string `json:"region"`
	Href   string `json:"href"`
}

type RegionsIndex struct {
	Data     []RegionData  `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Players Class Index

type ClassLinks struct {
	Scopes Link `json:"scopes"`
}

type ClassData struct {
	ID    int        `json:"id"`
	Key   string     `json:"key"`
	Name  string     `json:"name"`
	Specs []string   `json:"specs"`
	Links ClassLinks `json:"_links"`
}

type PlayersClassIndex struct {
	Data     []ClassData   `json:"data"`
	Metadata IndexMetadata `json:"metadata"`
}

// Class Scope Index (lists: global, regional, realm)

type ClassScopeData struct {
	Scope string `json:"scope"`
	Href  string `json:"href"`
}

type ClassScopeIndex struct {
	Data     []ClassScopeData `json:"data"`
	Metadata IndexMetadata    `json:"metadata"`
}
