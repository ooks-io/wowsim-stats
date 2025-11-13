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

export interface EquipmentItem {
  id: number;
  slot_type: string;
  item_id: number;
  item_name: string;
  quality: string;
  item_icon_slug?: string;
  enchantments?: Enchantment[];
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
export interface LatestRunMember {
  name: string;
  spec_id: number;
  realm_slug: string;
  region: string;
}

export interface LatestRunDetails {
  completed_timestamp: number;
  duration_ms: number;
  keystone_level: number;
  members: LatestRunMember[];
}

export interface LatestRunEntry {
  region: string;
  realm_slug: string;
  realm_name: string;
  realm_id: number;
  most_recent: number;
  most_recent_iso: string;
  period_id: string;
  dungeon_slug: string;
  dungeon_name: string;
  run_count: number;
  has_runs: boolean;
  latest_run: LatestRunDetails;
}

export interface PeriodCoverage {
  period_id: string;
  has_runs: boolean;
  latest_ts: number;
  latest_iso: string;
  run_count: number;
}

export interface DungeonStatus {
  dungeon_slug: string;
  dungeon_name: string;
  latest_ts: number;
  latest_iso: string;
  latest_period: string;
  periods: PeriodCoverage[];
  missing_periods: string[];
  latest_run?: LatestRunDetails;
}

export interface RealmStatus {
  region: string;
  realm_slug: string;
  realm_name: string;
  realm_id: number;
  dungeons: DungeonStatus[];
}

export interface StatusApiResponse {
  generated_at: number;
  periods: string[];
  summary: {
    endpoints_tested: number;
    success: number;
    failed: number;
  };
  latest_runs: LatestRunEntry[];
  realm_status: RealmStatus[];
}

export interface RealmStatusApiResponse {
  generated_at: number;
  region: string;
  realm_slug: string;
  realm_name: string;
  realm_id: number;
  periods: string[];
  dungeons: DungeonStatus[];
}
