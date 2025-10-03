import type { BestRun } from "./types";
import { formatDurationMMSS, buildLeaderboardURL } from "./utils";
import { formatRankingWithBracket } from "./ranking-utils";
import { getDungeonIconUrl } from "./dungeonIcons";
import { buildPlayerProfileURL } from "./utils";
import { getSpecInfo } from "./wow-constants";

export interface BestRunsOptions {
  mode?: "full" | "compact" | "status";
  showSectionWrapper?: boolean;
  className?: string;
}

export function renderBestRunsHeader(
  mode: "full" | "compact" | "status" = "full",
): string {
  const cells = [
    '<div class="header-cell header-cell--dungeon">Dungeon</div>',
    '<div class="header-cell header-cell--time">Time</div>',
    '<div class="header-cell header-cell--team">Team</div>',
  ];

  if (mode === "status") {
    cells.push(
      '<div class="header-cell header-cell--realm">Realm</div>',
      '<div class="header-cell header-cell--last">Last Run</div>',
      '<div class="header-cell header-cell--period">Period</div>',
    );
  } else {
    cells.push('<div class="header-cell header-cell--rank">Global</div>');
  }

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
  mode: "full" | "compact" | "status" = "full",
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

  // team composition
  const rawMembers = ((run as any).all_members ||
    (run as any).team_members ||
    []) as any[];
  // order members by role: tank, healer, dps
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

  const dungeonSlug = (run as any).dungeon_slug || "";
  if (mode !== "status") {
    const globalRankingValue =
      typeof (run as any).global_ranking_filtered === "number" &&
      (run as any).global_ranking_filtered > 0
        ? (run as any).global_ranking_filtered
        : (run as any).global_ranking;
    const globalBracket =
      (run as any).global_percentile_bracket || (run as any).percentile_bracket || "";
    const globalRankHTML = formatRankingWithBracket(
      globalRankingValue,
      globalBracket,
    );
    const globalURL = buildLeaderboardURL("global", "all", dungeonSlug);
    cells.push(
      `<div class="runs-cell runs-cell--rank"><a class="rank-link" href="${globalURL}">${globalRankHTML}</a></div>`,
    );
  } else {
    const meta = (run as any).__status || {};
    const realmText = meta.realm_name || meta.realm_slug || "-";
    const lastIso = meta.most_recent_iso || "";
    const lastTs = Number(meta.most_recent_ts || 0);
    const rel = lastTs ? timeAgo(lastTs) : "";
    const period = meta.period_id ? String(meta.period_id) : "";
    cells.push(
      `<div class="runs-cell runs-cell--realm">${realmText}</div>`,
      `<div class="runs-cell runs-cell--last">${lastIso}${rel ? ` <span class=\"text-subtle\">(${rel})</span>` : ""}</div>`,
      `<div class="runs-cell runs-cell--period">${period}</div>`,
    );
  }

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

  // generate mobile card structure
  const mobileCardHeader = `
    <div class="mobile-card-header">
      <div class="mobile-dungeon-info">
        ${iconUrl ? `<img class="dungeon-icon" src="${iconUrl}" alt="${run.dungeon_name}" />` : ""}
        <span class="dungeon-name">${run.dungeon_name}</span>
      </div>
      <div class="mobile-time">${duration}</div>
    </div>
  `;

  let mobileBodyContent = "";

  if (mode === "full" || mode === "compact") {
    const globalRankingValue =
      typeof (run as any).global_ranking_filtered === "number" &&
      (run as any).global_ranking_filtered > 0
        ? (run as any).global_ranking_filtered
        : (run as any).global_ranking;
    const globalBracket =
      (run as any).global_percentile_bracket || (run as any).percentile_bracket || "";
    const globalRankHTML = formatRankingWithBracket(
      globalRankingValue,
      globalBracket,
    );
    const globalURL = buildLeaderboardURL("global", "all", dungeonSlug);

    let mobileRankingsContent = `
      <div class="mobile-rank-item">
        <span class="mobile-rank-label">Global</span>
        <div class="mobile-rank-value">
          <a class="rank-link" href="${globalURL}">${globalRankHTML}</a>
        </div>
      </div>
    `;

    if (mode === "full") {
    // recalculate rankings for mobile (since they're scoped to the full mode block)
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

      mobileRankingsContent += `
      <div class="mobile-rank-item">
        <span class="mobile-rank-label">Region</span>
        <div class="mobile-rank-value">
          <a class="rank-link" href="${regionalURL}">${regionalRankHTML}</a>
        </div>
      </div>
      <div class="mobile-rank-item">
        <span class="mobile-rank-label">Realm</span>
        <div class="mobile-rank-value">
          <a class="rank-link" href="${realmURL}">${realmRankHTML}</a>
        </div>
      </div>
    `;
    }

    mobileBodyContent += `
      <div class="mobile-rankings">
        ${mobileRankingsContent}
      </div>
    `;
  } else if (mode === "status") {
    const meta = (run as any).__status || {};
    const realmText = meta.realm_name || meta.realm_slug || "-";
    const lastIso = meta.most_recent_iso || "";
    const lastTs = Number(meta.most_recent_ts || 0);
    const rel = lastTs ? timeAgo(lastTs) : "";
    const period = meta.period_id ? String(meta.period_id) : "";
    mobileBodyContent += `
      <div class="mobile-meta">
        <div class="mobile-meta-item">
          <span class="mobile-meta-label">Realm</span>
          <div class="mobile-meta-value">${realmText}</div>
        </div>
        <div class="mobile-meta-item">
          <span class="mobile-meta-label">Last Run</span>
          <div class="mobile-meta-value">${lastIso}${rel ? ` <span class=\"text-subtle\">(${rel})</span>` : ""}</div>
        </div>
        <div class="mobile-meta-item">
          <span class="mobile-meta-label">Period</span>
          <div class="mobile-meta-value">${period}</div>
        </div>
      </div>
    `;
  }

  const mobileTeam = `
    <div class="mobile-team">
      <div class="mobile-team-label">Team</div>
      <div class="mobile-team-composition">
        ${membersSorted
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
            <div class="mobile-team-member">
              <div class="spec-icon-placeholder" data-spec-id="${member.spec_id || 0}"></div>
              <a href="${profileUrl}" class="member-link" data-spec-id="${member.spec_id || 0}">
                ${memberName}
              </a>
            </div>
          `;
          })
          .join("")}
      </div>
    </div>
  `;

  const mobileCardBody = `
    <div class="mobile-card-body">
      ${mobileBodyContent}
      ${mobileTeam}
    </div>
  `;

  return `<div class="best-runs-row">${cells.join("")}${mobileCardHeader}${mobileCardBody}</div>`;
}

export function renderBestRunsTable(
  bestRuns: Record<string, BestRun> | BestRun[],
  options: BestRunsOptions = {},
  playerRegion?: string,
  playerRealmSlug?: string,
): string {
  const { mode = "full", className = "" } = options;

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

export const GRID_TEMPLATES = {
  full: "200px 120px 2fr 100px 100px 100px",
  compact: "150px 100px 2fr 80px",
  status: "200px 120px 2fr 200px 220px 90px",
  mobile: "1fr 100px 80px",
} as const;

function timeAgo(tsMs: number): string {
  if (!tsMs) return "";
  const sec = Math.max(0, Math.floor((Date.now() - tsMs) / 1000));
  const mins = Math.floor(sec / 60);
  const hours = Math.floor(mins / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days} day${days === 1 ? "" : "s"} ago`;
  if (hours > 0) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  if (mins > 0) return `${mins} minute${mins === 1 ? "" : "s"} ago`;
  return `${sec} seconds ago`;
}
