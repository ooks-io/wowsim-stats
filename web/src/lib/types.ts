// core data types for the application
export interface Player {
  id: number;
  player_id?: number;
  name: string;
  realm_slug: string;
  realm_name: string;
  region: string;
  class_name: string;
  active_spec_name: string;
  main_spec_id?: number;
  race_name?: string;
  gender?: string;
  guild_name?: string;
  level?: number;
  average_item_level?: number;
  equipped_item_level?: number;
  avatar_url?: string;
  global_ranking?: number;
  regional_ranking?: number;
  realm_ranking?: number;
  global_ranking_bracket?: string;
  regional_ranking_bracket?: string;
  realm_ranking_bracket?: string;
  ranking_percentile?: string; // contextual bracket based on leaderboard scope
  combined_best_time?: number;
  dungeons_completed?: number;
  total_runs?: number;
}

export interface TeamMember {
  id: number;
  name: string;
  spec_id: number;
  faction: string;
  realm_slug: string;
  region: string;
}

export interface ChallengeRun {
  id: number;
  duration: number;
  completed_timestamp: number;
  keystone_level: number;
  dungeon_name: string;
  realm_name: string;
  region: string;
  ranking?: number;
  percentile_bracket?: string;
  ranking_percentile?: string; // contextual bracket based on leaderboard scope
  members: TeamMember[];
}

export interface BestRun {
  dungeon_id: number;
  dungeon_name: string;
  dungeon_slug: string;
  duration: number;
  global_ranking?: number;
  global_ranking_filtered?: number;
  regional_ranking?: number;
  regional_ranking_filtered?: number;
  realm_ranking?: number;
  realm_ranking_filtered?: number;
  percentile_bracket?: string;
  global_percentile_bracket?: string;
  regional_percentile_bracket?: string;
  realm_percentile_bracket?: string;
  completed_timestamp: number;
  keystone_level: number;
  all_members: TeamMember[];
}

export interface Equipment {
  [slot: string]: EquipmentItem;
}

export interface ScalingOption {
  stats?: Record<string, number>;
  ilvl?: number;
  weaponDamageMin?: number;
  weaponDamageMax?: number;
}

export interface EquipmentItem {
  id: number;
  slot_type: string;
  item_id: number;
  item_name: string;
  quality: string;
  item_icon_slug?: string;
  enchantments?: Enchantment[];
  scaling_options?: Record<string, ScalingOption>;
}

export interface Enchantment {
  enchantment_id: number;
  slot_id: number;
  slot_type?: string;
  display_string: string;
  source_item_id?: number;
  gem_icon_slug?: string;
  gem_name?: string;
}

export interface LeaderboardData {
  leading_groups: ChallengeRun[];
  pagination: {
    currentPage: number;
    pageSize: number;
    totalPages: number;
    hasNextPage: boolean;
    hasPrevPage: boolean;
    totalRuns: number;
  };
}

export interface PlayerLeaderboardData {
  leaderboard: Player[];
  pagination: {
    currentPage: number;
    pageSize: number;
    totalPages: number;
    hasNextPage: boolean;
    hasPrevPage: boolean;
    totalPlayers: number;
    totalRuns: number;
  };
}

export interface PlayerSeasonData {
  main_spec_id?: number;
  dungeons_completed: number;
  total_runs: number;
  combined_best_time?: number;
  global_ranking?: number;
  regional_ranking?: number;
  realm_ranking?: number;
  global_ranking_bracket?: string;
  regional_ranking_bracket?: string;
  realm_ranking_bracket?: string;
  last_updated?: number;
  best_runs: Record<string, BestRun>;
}

export interface PlayerWithSeasons {
  id: number;
  name: string;
  realm_slug: string;
  realm_name: string;
  region: string;
  class_name?: string;
  active_spec_name?: string;
  race_name?: string;
  avatar_url?: string;
  guild_name?: string;
  average_item_level?: number;
  equipped_item_level?: number;
  seasons: Record<string, PlayerSeasonData>;
}

export interface PlayerProfileData {
  player: PlayerWithSeasons | null;
  equipment: Equipment;
  generated_at: number;
  version: string;
}

// API Response types
export interface APIResponse<T> {
  data?: T;
  error?: string;
}

// Home page types — mirror nix/pkgs/ookstats/src/internal/generator/home.go
// (HomeRunEntry / HomePlayerEntry). Keep field names in sync with that file.

export interface HomeRunEntry {
  rank?: number;
  bracket?: string;
  run_id: number;
  dungeon_id: number;
  dungeon_name: string;
  dungeon_slug: string;
  duration_ms: number;
  completed_timestamp: number;
  // Only set on entries in `recent_top_runs` (cross-season feed):
  season_id?: number;
  rankings?: { global?: number };
  team_members: TeamMember[];
}

export interface HomePlayerEntry {
  rank: number;
  player_id: number;
  name: string;
  realm_slug: string;
  realm_name?: string;
  region: string;
  class_name?: string;
  active_spec_id?: number;
  active_spec_name?: string;
  combined_best_time_ms: number;
  global_ranking?: number;
  global_ranking_bracket?: string;
  regional_ranking?: number;
  regional_ranking_bracket?: string;
  avatar_url?: string;
}

// Stats page types — mirror nix/pkgs/ookstats/src/internal/generator/stats.go.
// Keep field names in sync.

export interface StatsJSON {
  generated_at: number;
  // Outer key: region ("global" | "us" | "eu" | "kr" | "tw").
  // Inner key: season key ("all_time" | "season_1" | "season_2").
  scopes: Record<string, Record<string, StatsScope>>;
}

export interface StatsScope {
  total_runs: number;
  total_players: number;
  nine_of_nine_players: number;
  completion_tiers: {
    "9_of_9_gold": StatsCompletionTier;
    "9_of_9_platinum": StatsCompletionTier;
    "9_of_9_title": StatsCompletionTier;
  };
  spec_counts: Record<string, StatsSpecCountBucket>; // keys: "all_runs" | "gold_runs" | "platinum_runs" | "title_runs" | "top_50_runs"
  weekly_activity: StatsWeeklyActivityEntry[];
}

export interface StatsSpecCountBucket {
  // Distinct runs in this bucket (denominator for the "runs with spec" metric).
  total_runs: number;
  entries: StatsSpecCountEntry[];
  // Per-dungeon breakdown — same shape, keyed by numeric dungeon_id.
  by_dungeon: Record<string, StatsDungeonSpecBucket>;
}

export interface StatsDungeonSpecBucket {
  total_runs: number;
  entries: StatsSpecCountEntry[];
}

export interface StatsCompletionTier {
  count: number;
  percentile_of_all_players: number;
  percentile_of_completed_players: number;
  percentile_of_all_time_players: number;
}

export interface StatsSpecCountEntry {
  spec_id: number;
  class_name: string;
  spec_name: string;
  // Total spec slot occurrences (e.g. 2 Combat Rogues in one run = 2).
  count: number;
  // Distinct runs containing this spec (e.g. 2 Combat Rogues in one run = 1).
  runs_with_spec: number;
}

// Gear popularity (gear.json) — top items per slot for the top players in
// each (season, spec) combo. See nix/.../generator/gear.go for details.
export interface GearJSON {
  generated_at: number;
  // Outer key: season key ("season_1" | "season_2"). Inner key: spec_id (string).
  scopes: Record<string, Record<string, GearSpecBucket>>;
}

export interface GearSpecBucket {
  // Number of qualifying players whose gear contributed to this bucket.
  total_players: number;
  // Keyed by canonical slot name (HEAD, CHEST, FINGER, TRINKET, ...).
  slots: Record<string, GearSlotBucket>;
}

export interface GearSlotBucket {
  // Number of qualifying players who had any item in this slot.
  players_with_slot: number;
  items: GearItemEntry[];
}

export interface GearItemEntry {
  item_id: number;
  name: string;
  icon?: string;
  quality: number;
  count: number;
}

export interface StatsWeeklyActivityEntry {
  week_start: string; // ISO date or YYYY-MM-DD
  run_count: number;
}

// frontend component props
export interface LeaderboardTableProps {
  initialData?: LeaderboardData;
  region: string;
  realm: string;
  dungeon: string;
}

export interface PlayerSearchProps {
  onPlayerSelect: (player: Player) => void;
}

export interface FilterPanelProps {
  initialRegion?: string;
  initialRealm?: string;
  initialDungeon?: string;
  onFilterChange: (filters: FilterState) => void;
}

export interface FilterState {
  region: string;
  realm: string;
  dungeon: string;
  teamFilter: boolean;
}

// player Search types
export interface PlayerSearchResult {
  id: number;
  name: string;
  realm_slug: string;
  realm_name: string;
  region: string;
  class_name: string;
  active_spec_name: string;
  global_ranking?: number;
  regional_ranking?: number;
  realm_ranking?: number;
  global_ranking_bracket?: string;
  regional_ranking_bracket?: string;
  realm_ranking_bracket?: string;
  combined_best_time?: number;
  last_seen?: string;
}

export interface PlayerSearchIndex {
  players: PlayerSearchResult[];
  metadata: {
    total_players: number;
    last_updated: string;
    version: string;
  };
}

export interface FuseSearchResult<T> {
  item: T;
  refIndex: number;
  score?: number;
}

// Status page types
export type CoverageHealth = "ok" | "some_missing" | "no_data";

export interface DungeonCoverage {
  dungeon_id: number;
  dungeon_slug: string;
  dungeon_name: string;
  status: CoverageHealth;
  periods: number[];
  missing_periods: number[];
  error_periods: number[];
}

export interface RealmCoverage {
  region: string;
  realm_slug: string;
  realm_name: string;
  health: CoverageHealth;
  total_periods: number;
  missing_periods: number;
  error_periods: number;
  dungeons: DungeonCoverage[];
}

export interface StatusApiResponse {
  generated_at: string;
  realms: RealmCoverage[];
}

export interface RealmStatusApiResponse {
  generated_at: string;
  realm: RealmCoverage;
}
