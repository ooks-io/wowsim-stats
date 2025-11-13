// LeaderboardTable client-side logic
// Handles both dungeon and player leaderboards

import {
  formatDurationMMSS,
  getClassColor,
  getSpecInfo,
  getSpecIcon,
} from "../../lib/client-utils";
import { buildPlayerProfileURL, dungeonIdToSlug } from "../../lib/utils";

type LeaderboardType = "dungeon" | "player";

export class LeaderboardTableClient {
  private container: HTMLElement;
  private type: LeaderboardType;
  private region: string;
  private realm: string;
  private dungeon: string;
  private currentPage = 1;

  constructor(container: HTMLElement) {
    this.container = container;
    this.type = (container.dataset.type || "dungeon") as LeaderboardType;
    this.region = container.dataset.region || "";
    this.realm = container.dataset.realm || "";
    this.dungeon = container.dataset.dungeon || "";
  }

  public async loadLeaderboard(
    newRegion?: string,
    newRealm?: string,
    newDungeon?: string,
    targetPage?: number,
  ) {
    if (newRegion) this.region = newRegion;
    if (newRealm) this.realm = newRealm;
    if (newDungeon) this.dungeon = newDungeon;
    if (newRegion || newRealm || newDungeon) {
      this.currentPage = targetPage || 1;
    } else if (targetPage) {
      this.currentPage = targetPage;
    }

    try {
      this.showLoading();

      const data = await this.fetchData();

      if (this.type === "dungeon") {
        this.renderDungeonLeaderboard(data);
      } else {
        this.renderPlayerLeaderboard(data);
      }

      this.updateURL();
    } catch (error) {
      this.showError(error as Error);
      console.error("Leaderboard loading error:", error);
    } finally {
      this.hideLoading();
    }
  }

  private async fetchData(): Promise<any> {
    let url: string;

    if (this.type === "dungeon") {
      // Dungeon leaderboard
      const dungeonSlug = dungeonIdToSlug(parseInt(this.dungeon));

      if (this.region === "global") {
        url = `/api/leaderboard/global/${dungeonSlug}/${this.currentPage}.json`;
      } else if (this.realm === "all") {
        url = `/api/leaderboard/${this.region}/all/${dungeonSlug}/${this.currentPage}.json`;
      } else {
        url = `/api/leaderboard/${this.region}/${this.realm}/${dungeonSlug}/${this.currentPage}.json`;
      }
    } else {
      // Player leaderboard
      // TODO: implement player leaderboard API paths
      url = `/api/leaderboard/players/global/${this.currentPage}.json`;
    }

    const response = await fetch(url);
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error("Leaderboard data not found.");
      }
      throw new Error(`Failed to load leaderboard: ${response.status}`);
    }

    return response.json();
  }

  private renderDungeonLeaderboard(data: any) {
    const content = this.container.querySelector(
      ".leaderboard-content",
    ) as HTMLElement | null;
    if (!content) return;

    const runs = data.leading_groups || [];
    const hasData = runs.length > 0;

    if (!hasData) {
      content.innerHTML = `
        <div class="table-empty">
          <p>No runs found for this leaderboard.</p>
        </div>
      `;
      return;
    }

    // Build table HTML
    let tableHTML = '<div class="table-container" data-mode="leaderboard">';
    tableHTML += '<div class="table-header">';
    tableHTML += '<div class="table-header-cell">Rank</div>';
    tableHTML += '<div class="table-header-cell">Time</div>';
    tableHTML += '<div class="table-header-cell">Team</div>';
    tableHTML += '<div class="table-header-cell">Date</div>';
    tableHTML += '</div>';
    tableHTML += '<div class="table-body">';

    const roleOrder: Record<string, number> = { tank: 0, healer: 1, dps: 2 };

    runs.forEach((run: any, index: number) => {
      const rank = (this.currentPage - 1) * 25 + index + 1;
      const duration = formatDurationMMSS(run.duration);
      const date = new Date(run.completed_timestamp).toLocaleDateString(
        "en-US",
        {
          month: "short",
          day: "numeric",
          year: "numeric",
        },
      );

      // Sort members by role
      const membersSorted = [...(run.members || [])].sort((a: any, b: any) => {
        const aSpec = getSpecInfo(a.spec_id);
        const bSpec = getSpecInfo(b.spec_id);
        const aRole = aSpec?.role || "dps";
        const bRole = bSpec?.role || "dps";
        const aw = roleOrder[aRole] ?? 99;
        const bw = roleOrder[bRole] ?? 99;
        if (aw !== bw) return aw - bw;
        return a.name.localeCompare(b.name);
      });

      // Build team HTML
      let teamHTML = '<div class="team-composition">';
      membersSorted.forEach((member: any) => {
        const spec = getSpecInfo(member.spec_id);
        const iconUrl = spec ? getSpecIcon(spec.class, spec.spec) : null;
        const classColor = spec ? getClassColor(spec.class) : "#FFFFFF";
        const profileUrl = buildPlayerProfileURL(
          member.region || "us",
          member.realm_slug,
          member.name,
        );

        teamHTML += '<span class="team-member">';
        teamHTML += '<a href="' + profileUrl + '" class="player-link">';
        if (iconUrl) {
          teamHTML +=
            '<img src="' +
            iconUrl +
            '" alt="' +
            (spec?.spec || "") +
            '" class="spec-icon" loading="lazy">';
        }
        teamHTML +=
          '<span class="player-name" style="color: ' +
          classColor +
          ';">' +
          member.name +
          "</span>";
        teamHTML += "</a>";
        teamHTML += "</span>";
      });
      teamHTML += "</div>";

      tableHTML += '<div class="table-row">';
      tableHTML +=
        '<div class="table-cell" data-label="Rank"><span class="rank">#' +
        rank +
        "</span></div>";
      tableHTML +=
        '<div class="table-cell" data-label="Time">' + duration + "</div>";
      tableHTML +=
        '<div class="table-cell" data-label="Team">' + teamHTML + "</div>";
      tableHTML += '<div class="table-cell" data-label="Date">' + date + "</div>";
      tableHTML += "</div>";
    });

    tableHTML += "</div>";
    tableHTML += "</div>";

    content.innerHTML = tableHTML;
  }

  private renderPlayerLeaderboard(data: any) {
    const content = this.container.querySelector(
      ".leaderboard-content",
    ) as HTMLElement | null;
    if (!content) return;

    const players = data.leaderboard || [];
    const hasData = players.length > 0;

    if (!hasData) {
      content.innerHTML = `
        <div class="table-empty">
          <p>No players found for this leaderboard.</p>
        </div>
      `;
      return;
    }

    // TODO: Build player table HTML similar to dungeon rendering
    content.innerHTML = "<p>Player leaderboard rendering coming soon...</p>";
  }

  private updateURL() {
    // Build path-based URL: /challenge-mode/{region}/{realm}/{dungeon}
    const dungeonSlug = dungeonIdToSlug(parseInt(this.dungeon));
    let path: string;

    if (this.region === "global") {
      path = `/challenge-mode/global/${dungeonSlug}`;
    } else if (this.realm === "all") {
      path = `/challenge-mode/${this.region}/all/${dungeonSlug}`;
    } else {
      path = `/challenge-mode/${this.region}/${this.realm}/${dungeonSlug}`;
    }

    // Add page query param if not page 1
    const params = new URLSearchParams();
    if (this.currentPage > 1) {
      params.set("page", String(this.currentPage));
    }

    const queryString = params.toString();
    const newURL = queryString ? `${path}?${queryString}` : path;

    window.history.replaceState({}, "", newURL);
  }

  private showLoading() {
    const overlay = this.container.querySelector(
      ".loading-overlay",
    ) as HTMLElement | null;
    if (overlay) overlay.style.display = "flex";
  }

  private hideLoading() {
    const overlay = this.container.querySelector(
      ".loading-overlay",
    ) as HTMLElement | null;
    if (overlay) overlay.style.display = "none";
  }

  private showError(error: Error) {
    const errorEl = this.container.querySelector(
      ".error-message",
    ) as HTMLElement | null;
    const errorText = errorEl?.querySelector("p");

    if (errorText) {
      errorText.textContent = `Error: ${error.message}`;
    }
    if (errorEl) {
      errorEl.style.display = "block";
    }
  }
}

// Auto-initialize if container exists
document.addEventListener("DOMContentLoaded", () => {
  const container = document.querySelector(
    ".leaderboard-table-container",
  ) as HTMLElement | null;

  if (container) {
    const instance = new LeaderboardTableClient(container);
    // Expose globally for filter components to use
    (window as any).leaderboardTable = instance;
  }
});
