import type { BestRun } from "./types";
import { formatDurationMMSS, buildLeaderboardURL } from "./utils";
import { formatRankingWithBracket } from "./ranking-utils";
import { getDungeonIconUrl } from "./dungeonIcons";
import { buildPlayerProfileURL } from "./utils";
import { getSpecInfo } from "./wow-constants";

export interface BestRunsOptions {
  mode?: "full" | "compact";
  showSectionWrapper?: boolean;
  className?: string;
}

export function renderBestRunsHeader(
  mode: "full" | "compact" = "full",
): string {
  const cells = [
    '<div class="header-cell header-cell--dungeon">Dungeon</div>',
    '<div class="header-cell header-cell--time">Time</div>',
    '<div class="header-cell header-cell--team">Team</div>',
    '<div class="header-cell header-cell--rank">Global</div>',
  ];

  if (mode === "full") {
    cells.push(
      '<div class="header-cell header-cell--rank">Region</div>',
      '<div class="header-cell header-cell--rank">Realm</div>',
    );
  }

  return cells.join("");
}

export function renderBestRunsRow(
  run: BestRun,
  mode: "full" | "compact" = "full",
  playerRegion?: string,
  playerRealmSlug?: string,
): string {
  const duration = formatDurationMMSS(run.duration);

  const iconUrl = getDungeonIconUrl(
    (run as any).dungeon_slug,
    run.dungeon_name,
  );
  const dungeonCell = iconUrl
    ? `<div class="runs-cell runs-cell--dungeon"><img class="dungeon-icon" src="${iconUrl}" alt="${run.dungeon_name}" /> <span class="dungeon-name">${run.dungeon_name}</span></div>`
    : `<div class="runs-cell runs-cell--dungeon"><span class="dungeon-name">${run.dungeon_name}</span></div>`;

  const cells = [
    dungeonCell,
    `<div class="runs-cell runs-cell--time">${duration}</div>`,
  ];

  // Team composition (always included). Support SSR (team_members) and API (all_members)
  const rawMembers = ((run as any).all_members ||
    (run as any).team_members ||
    []) as any[];
  // Order members by role: tank, healer, dps
  const roleOrder: Record<string, number> = { tank: 0, healer: 1, dps: 2 };
  const membersSorted = [...rawMembers].sort((a, b) => {
    const aSpec = a.spec_id || a.specialization?.id;
    const bSpec = b.spec_id || b.specialization?.id;
    const aRole = aSpec ? getSpecInfo(Number(aSpec))?.role || "dps" : "dps";
    const bRole = bSpec ? getSpecInfo(Number(bSpec))?.role || "dps" : "dps";
    const aw = roleOrder[aRole] ?? 99;
    const bw = roleOrder[bRole] ?? 99;
    if (aw !== bw) return aw - bw;
    // stable fallback by name
    return String(a.name || "").localeCompare(String(b.name || ""));
  });

  const teamHTML = membersSorted
    .map((member) => {
      const memberRegion = member.region || "us";
      const memberRealm = member.realm_slug || "unknown";
      const memberName = member.name || "Unknown";
      const profileUrl = buildPlayerProfileURL(
        memberRegion,
        memberRealm,
        memberName,
      );

      return `
        <span class="team-member">
          <div class="spec-icon-placeholder" data-spec-id="${member.spec_id || 0}"></div>
          <a href="${profileUrl}" class="member-link" data-spec-id="${member.spec_id || 0}">
            ${memberName}
          </a>
        </span>
      `;
    })
    .join("");

  cells.push(`
    <div class="runs-cell runs-cell--team">
      <div class="team-composition">
        ${teamHTML}
      </div>
    </div>
  `);

  // Rankings (prefer filtered and scope-specific percentile brackets when available)
  const globalRankingValue =
    typeof (run as any).global_ranking_filtered === "number" &&
    (run as any).global_ranking_filtered > 0
      ? (run as any).global_ranking_filtered
      : run.global_ranking;
  const globalBracket =
    (run as any).global_percentile_bracket || run.percentile_bracket || "";
  const globalRankHTML = formatRankingWithBracket(
    globalRankingValue,
    globalBracket,
  );
  const dungeonSlug = (run as any).dungeon_slug || "";
  const globalURL = buildLeaderboardURL("global", "all", dungeonSlug);
  cells.push(
    `<div class="runs-cell runs-cell--rank"><a class="rank-link" href="${globalURL}">${globalRankHTML}</a></div>`,
  );

  if (mode === "full") {
    const regionalBracket =
      (run as any).regional_percentile_bracket || run.percentile_bracket || "";
    const realmBracket =
      (run as any).realm_percentile_bracket || run.percentile_bracket || "";
    const regionalRankingValue =
      typeof (run as any).regional_ranking_filtered === "number" &&
      (run as any).regional_ranking_filtered > 0
        ? (run as any).regional_ranking_filtered
        : run.regional_ranking;
    const realmRankingValue =
      typeof (run as any).realm_ranking_filtered === "number" &&
      (run as any).realm_ranking_filtered > 0
        ? (run as any).realm_ranking_filtered
        : run.realm_ranking;
    const regionalRankHTML = formatRankingWithBracket(
      regionalRankingValue,
      regionalBracket,
    );
    const realmRankHTML = formatRankingWithBracket(
      realmRankingValue,
      realmBracket,
    );
    const region = playerRegion || "";
    const realmSlug = playerRealmSlug || "";
    const regionalURL = region
      ? buildLeaderboardURL(region, "all", dungeonSlug)
      : "#";
    const realmURL =
      region && realmSlug
        ? buildLeaderboardURL(region, realmSlug, dungeonSlug)
        : "#";

    cells.push(
      `<div class="runs-cell runs-cell--rank"><a class="rank-link" href="${regionalURL}">${regionalRankHTML}</a></div>`,
      `<div class="runs-cell runs-cell--rank"><a class="rank-link" href="${realmURL}">${realmRankHTML}</a></div>`,
    );
  }

  return `<div class="best-runs-row">${cells.join("")}</div>`;
}

export function renderBestRunsTable(
  bestRuns: Record<string, BestRun> | BestRun[],
  options: BestRunsOptions = {},
  playerRegion?: string,
  playerRealmSlug?: string,
): string {
  const { mode = "full", className = "" } = options;

  // Convert to array if needed
  const runsArray = Array.isArray(bestRuns)
    ? bestRuns
    : Object.values(bestRuns || {});

  if (runsArray.length === 0) {
    return `
      <div class="no-runs-message">
        <p>No best runs found for this player.</p>
      </div>
    `;
  }

  const headerHTML = renderBestRunsHeader(mode);
  const rowsHTML = runsArray
    .map((run) => renderBestRunsRow(run, mode, playerRegion, playerRealmSlug))
    .join("");

  return `
    <div class="best-runs-table ${className}" data-mode="${mode}">
      <div class="best-runs-header">
        ${headerHTML}
      </div>
      <div class="best-runs-body">
        ${rowsHTML}
      </div>
    </div>
  `;
}

export function renderBestRunsWithWrapper(
  bestRuns: Record<string, BestRun> | BestRun[],
  options: BestRunsOptions = {},
  playerRegion?: string,
  playerRealmSlug?: string,
): string {
  const { showSectionWrapper = false } = options;

  const tableHTML = renderBestRunsTable(
    bestRuns,
    options,
    playerRegion,
    playerRealmSlug,
  );

  if (showSectionWrapper) {
    return `
      <div class="profile-section">
        <h3 class="section-title">Best Times Per Dungeon</h3>
        ${tableHTML}
      </div>
    `;
  }

  return `<div class="best-runs-container">${tableHTML}</div>`;
}

// CSS classes that should be shared between components
export const BEST_RUNS_CSS_CLASSES = {
  container: "best-runs-container",
  profileSection: "profile-section",
  sectionTitle: "section-title",
  table: "best-runs-table",
  header: "best-runs-header",
  headerCell: "header-cell",
  body: "best-runs-body",
  row: "best-runs-row",
  cell: "runs-cell",
  teamComposition: "team-composition",
  teamMember: "team-member",
  specIconPlaceholder: "spec-icon-placeholder",
  specIcon: "spec-icon",
  memberLink: "member-link",
  noRuns: "no-runs-message",
};

// Grid template columns for different modes
export const GRID_TEMPLATES = {
  full: "200px 120px 2fr 100px 100px 100px",
  compact: "150px 100px 80px",
  mobile: "1fr 100px 80px",
} as const;
