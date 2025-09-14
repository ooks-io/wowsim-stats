import {
  formatDurationMMSS,
  formatTimestamp,
  getClassColor,
  getSpecInfo,
  getSpecIcon,
} from "../lib/client-utils.ts";
import { buildPlayerProfileURL, dungeonIdToSlug } from "../lib/utils.ts";

class LeaderboardTable {
  private container: HTMLElement;
  private region: string;
  private realm: string;
  private dungeon: string;
  private currentPage = 1;

  constructor(container: HTMLElement) {
    this.container = container;
    this.region = container.dataset.region || "";
    this.realm = container.dataset.realm || "";
    this.dungeon = container.dataset.dungeon || "";
    this.bindEvents();
  }

  private bindEvents() {
    const prevBtn =
      this.container.querySelector<HTMLButtonElement>("#prev-page");
    const nextBtn =
      this.container.querySelector<HTMLButtonElement>("#next-page");
    prevBtn?.addEventListener("click", () => this.previousPage());
    nextBtn?.addEventListener("click", () => this.nextPage());
  }

  private async previousPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
      await this.loadLeaderboard(
        undefined,
        undefined,
        undefined,
        undefined,
        false,
      );
    }
  }

  private async nextPage() {
    this.currentPage++;
    await this.loadLeaderboard(
      undefined,
      undefined,
      undefined,
      undefined,
      false,
    );
  }

  private async fetchLeaderboardData(
    region: string,
    realm: string,
    dungeonId: number,
    page = 1,
  ): Promise<any> {
    // Convert dungeon ID to slug via shared util
    const dungeonSlug = dungeonIdToSlug(dungeonId);

    let url: string;
    if (region === "global") {
      url = `/api/leaderboard/global/${dungeonSlug}/${page}.json`;
    } else if (realm === "all") {
      url = `/api/leaderboard/${region}/all/${dungeonSlug}/${page}.json`;
    } else {
      url = `/api/leaderboard/${region}/${realm}/${dungeonSlug}/${page}.json`;
    }

    const response = await fetch(url);
    if (!response.ok) {
      // Provide user-friendly error messages based on status codes
      if (response.status === 404) {
        throw new Error("Leaderboard data not found. This combination of region, realm, and dungeon may not have any recorded runs yet.");
      } else if (response.status >= 500) {
        throw new Error("Server error occurred. Please try again later.");
      } else {
        const errorText = await response.text();
        throw new Error(
          `Failed to load leaderboard: ${response.status} ${response.statusText}`,
        );
      }
    }
    return response.json();
  }

  // dungeonIdToSlug now provided by shared utils

  public async loadLeaderboard(
    newRegion?: string,
    newRealm?: string,
    newDungeon?: string,
    targetPage?: number,
    preserveScroll: boolean = true,
  ) {
    const prevScrollY = window.scrollY;
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
      const data = await this.fetchLeaderboardData(
        this.region,
        this.realm,
        parseInt(this.dungeon),
        this.currentPage,
      );
      this.renderLeaderboard(data);
      this.updatePagination(data.pagination);
      this.updateURL();
      // restore or adjust scroll
      if (preserveScroll) {
        window.scrollTo({ top: prevScrollY });
      } else {
        const rect = this.container.getBoundingClientRect();
        const top = window.scrollY + rect.top - 12;
        window.scrollTo({ top, behavior: "smooth" });
      }
    } catch (error) {
      this.showError(error as Error);
      console.error("Leaderboard loading error:", error);
    } finally {
      this.hideLoading();
    }
  }

  private renderLeaderboard(data: any) {
    const content = this.container.querySelector(
      "#leaderboard-content",
    ) as HTMLElement | null;
    if (!content) return;
    if (!data.leading_groups || data.leading_groups.length === 0) {
      content.innerHTML = `
        <div class="empty-state">
          <p>No runs found for this leaderboard.</p>
          <p>Try selecting a different region, realm, or dungeon.</p>
        </div>
      `;
      return;
    }

    content.innerHTML =
      '<div id="leaderboard-rows" class="leaderboard-table"></div>';
    const rowsContainer = content.querySelector(
      "#leaderboard-rows",
    ) as HTMLElement | null;
    data.leading_groups.forEach((run: any, index: number) => {
      const rank = run.ranking || (this.currentPage - 1) * 25 + index + 1;
      const duration = formatDurationMMSS(run.duration);
      const timestamp = formatTimestamp(run.completed_timestamp);

      const currentRegion = this.region;
      const currentRealm = this.realm;
      const isIndividualRealmView =
        currentRegion &&
        currentRealm &&
        currentRealm !== "all" &&
        currentRegion !== "global";

      let teamHTML = "";
      const roleOrder: Record<string, number> = { tank: 0, healer: 1, dps: 2 };
      const membersSorted = [...(run.members || [])].sort((a: any, b: any) => {
        const aSpec = a.spec_id || a.specialization?.id;
        const bSpec = b.spec_id || b.specialization?.id;
        const aRole = aSpec ? getSpecInfo(Number(aSpec))?.role || "dps" : "dps";
        const bRole = bSpec ? getSpecInfo(Number(bSpec))?.role || "dps" : "dps";
        const aw = roleOrder[aRole] ?? 99;
        const bw = roleOrder[bRole] ?? 99;
        if (aw !== bw) return aw - bw;
        return String(a.name || "").localeCompare(String(b.name || ""));
      });
      membersSorted.forEach((member: any) => {
        const specId = member.spec_id || member.specialization?.id;
        const spec = specId ? getSpecInfo(specId) : null;
        const iconUrl = spec ? getSpecIcon(spec.class, spec.spec) : null;
        const classColor = spec ? getClassColor(spec.class) : "#FFFFFF";

        const iconHTML =
          iconUrl && spec
            ? `<img src="${iconUrl}" alt="${spec.spec} ${spec.class}" style="width: 16px; height: 16px; border-radius: 2px; margin-right: 4px; vertical-align: middle; flex-shrink: 0;">`
            : "";

        const memberRealmSlug =
          member.realm_slug || member.profile?.realm?.slug;
        let crossRealmIndicator = "";
        if (
          isIndividualRealmView &&
          memberRealmSlug &&
          memberRealmSlug !== currentRealm
        ) {
          crossRealmIndicator =
            '<span style="color: #ff6b6b; font-weight: bold; margin-left: 2px;">*</span>';
        }

        const memberRegion = member.region || "us";
        const profileUrl = buildPlayerProfileURL(
          memberRegion,
          memberRealmSlug,
          member.name,
        );
        teamHTML += `<span style="display: inline-flex; align-items: center; margin-right: 8px; gap: 4px;">
          ${iconHTML}
          <a href="${profileUrl}" style="color: ${classColor}; font-weight: 600; font-size: 0.9em; text-decoration: none;">
            ${member.name}${crossRealmIndicator}
          </a>
        </span>`;
      });

      const row = document.createElement("div");
      row.className = "leaderboard-table-row";
      row.dataset.runId = String(run.id);
      row.innerHTML = `
        <div class="leaderboard-cell leaderboard-cell--rank">#${rank}</div>
        <div class="leaderboard-cell leaderboard-cell--time">${duration}</div>
        <div class="leaderboard-cell leaderboard-cell--content">${teamHTML}</div>
        <div class="leaderboard-cell leaderboard-cell--meta">${timestamp}</div>
      `;
      rowsContainer?.appendChild(row);
    });
  }

  private updatePagination(pagination: any) {
    const paginationContainer = this.container.querySelector(
      ".pagination-container",
    ) as HTMLElement | null;
    if (!paginationContainer) return;
    if (!pagination || pagination.totalPages <= 1) {
      paginationContainer.style.display = "none";
      return;
    }
    paginationContainer.style.display = "flex";
    const prevBtn =
      this.container.querySelector<HTMLButtonElement>("#prev-page");
    const nextBtn =
      this.container.querySelector<HTMLButtonElement>("#next-page");
    if (prevBtn) prevBtn.disabled = !pagination.hasPrevPage;
    if (nextBtn) nextBtn.disabled = !pagination.hasNextPage;
    // Left-side info line
    const info = this.container.querySelector(
      ".pagination-info span",
    ) as HTMLElement | null;
    if (info) {
      info.textContent = `Showing page ${pagination.currentPage} of ${pagination.totalPages} (${pagination.totalRuns} total runs)`;
    }
    // Center page indicator between buttons
    const pageInfo = this.container.querySelector(
      ".page-info",
    ) as HTMLElement | null;
    if (pageInfo) {
      pageInfo.textContent = `Page ${pagination.currentPage} of ${pagination.totalPages}`;
    }
  }

  private updateURL() {
    const params = new URLSearchParams(window.location.search);
    if (this.currentPage === 1) params.delete("page");
    else params.set("page", String(this.currentPage));
    const next = `${window.location.pathname}?${params.toString()}`;
    window.history.replaceState({}, "", next);
  }

  private showLoading() {
    const loading = this.container.querySelector(
      "#loading-container",
    ) as HTMLElement | null;
    const content = this.container.querySelector(
      "#leaderboard-content",
    ) as HTMLElement | null;
    const error = this.container.querySelector(
      "#error-container",
    ) as HTMLElement | null;
    if (loading) loading.style.display = "block";
    // Preserve layout to avoid scroll bump: hide content visually but keep space
    if (content) {
      const h = content.offsetHeight;
      if (h > 0) (content as any).dataset.prevMinHeight = String(h);
      if (h > 0) content.style.minHeight = h + "px";
      content.style.visibility = "hidden";
      content.style.pointerEvents = "none";
    }
    if (error) error.style.display = "none";
  }

  private hideLoading() {
    const loading = this.container.querySelector(
      "#loading-container",
    ) as HTMLElement | null;
    const content = this.container.querySelector(
      "#leaderboard-content",
    ) as HTMLElement | null;
    if (loading) loading.style.display = "none";
    if (content) {
      content.style.visibility = "visible";
      content.style.pointerEvents = "";
      content.style.minHeight = "";
    }
  }

  private showError(error: Error) {
    const errorContainer = this.container.querySelector(
      "#error-container",
    ) as HTMLElement | null;
    const errorMessage = errorContainer?.querySelector(
      ".loading-message",
    ) as HTMLElement | null;
    const loading = this.container.querySelector(
      "#loading-container",
    ) as HTMLElement | null;
    const content = this.container.querySelector(
      "#leaderboard-content",
    ) as HTMLElement | null;
    if (errorMessage) errorMessage.textContent = `Error: ${error.message}`;
    if (errorContainer) errorContainer.style.display = "block";
    if (loading) loading.style.display = "none";
    if (content) content.style.display = "none";
  }
}

document.addEventListener("DOMContentLoaded", () => {
  const container = document.querySelector(
    ".leaderboard-container",
  ) as HTMLElement | null;
  if (container) {
    (window as any).leaderboardTable = new LeaderboardTable(container);
    try {
      window.dispatchEvent(new CustomEvent("leaderboard:ready"));
    } catch {}
  }
});
