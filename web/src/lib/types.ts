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
