// Minimal client module for Player Leaderboard functionality

import { buildPlayerProfileURL } from "../lib/utils.ts";
import {
  getClassColor,
  getSpecInfo,
  getSpecIcon,
} from "../lib/client-utils.ts";

class PlayerLeaderboardClient {
  private container: HTMLElement;
  private content: HTMLElement | null;
  private error: HTMLElement | null;
  private loading: HTMLElement | null;
  private currentPage = 1;
  private scope: "global" | "regional" = "global";
  private region: string | null = null;

  constructor(container: HTMLElement) {
    this.container = container;
    this.content = container.querySelector(
      "#player-leaderboard-content",
    ) as HTMLElement | null;
    this.error = container.querySelector(
      "#error-container",
    ) as HTMLElement | null;
    this.loading = container.querySelector(
      "#loading-container",
    ) as HTMLElement | null;

    this.bindPagination();
    // initial load
    this.refreshCurrentPage();
  }

  public refreshCurrentPage() {
    const urlParams = new URLSearchParams(window.location.search);
    const pageParam = urlParams.get("page");
    this.currentPage = pageParam ? Math.max(1, parseInt(pageParam)) : 1;
    this.load(this.currentPage).catch((e) =>
      console.error("Player leaderboard load error:", e),
    );
  }

  private bindPagination() {
    const prevBtn = this.container.querySelector(
      "#prev-page",
    ) as HTMLButtonElement | null;
    const nextBtn = this.container.querySelector(
      "#next-page",
    ) as HTMLButtonElement | null;
    prevBtn?.addEventListener("click", () => {
      if (this.currentPage > 1) {
        this.currentPage--;
        this.load(this.currentPage, { preserveScroll: false });
      }
    });
    nextBtn?.addEventListener("click", () => {
      this.currentPage++;
      this.load(this.currentPage, { preserveScroll: false });
    });
  }

  private async fetchData(page: number) {
    let url: string;
    
    if (this.scope === "global") {
      url = `/api/leaderboard/players/global/${page}.json`;
    } else if (this.region) {
      url = `/api/leaderboard/players/${this.region}/${page}.json`;
    } else {
      url = `/api/leaderboard/players/global/${page}.json`;
    }
    
    console.log("[PlayerLeaderboard] fetching:", url);
    const res = await fetch(url);
    console.log("[PlayerLeaderboard] response:", res.status, res.statusText);
    if (!res.ok) {
      // Provide user-friendly error messages based on status codes
      if (res.status === 404) {
        throw new Error("Player rankings not found. No players have complete coverage for this scope yet.");
      } else if (res.status >= 500) {
        throw new Error("Server error occurred while loading player rankings. Please try again later.");
      } else {
        throw new Error(`Failed to load player rankings: ${res.status} ${res.statusText}`);
      }
    }
    const data = await res.json();
    console.log("[PlayerLeaderboard] data received:", {
      hasLeaderboard: Array.isArray(data?.leaderboard),
      count: Array.isArray(data?.leaderboard)
        ? data.leaderboard.length
        : undefined,
      pagination: data?.pagination,
    });
    return data;
  }

  private showLoading() {
    if (this.loading) this.loading.style.display = "block";
    if (this.content) {
      const h = this.content.offsetHeight;
      if (h > 0) (this.content as any).dataset.prevMinHeight = String(h);
      if (h > 0) this.content.style.minHeight = h + "px";
      this.content.style.visibility = "hidden";
      this.content.style.pointerEvents = "none";
    }
    if (this.error) this.error.style.display = "none";
  }

  private hideLoading() {
    if (this.loading) this.loading.style.display = "none";
    if (this.content) {
      this.content.style.visibility = "visible";
      this.content.style.pointerEvents = "";
      this.content.style.minHeight = "";
    }
  }

  private showError(message: string) {
    const errorContainer = this.error;
    const msg = errorContainer?.querySelector(
      ".loading-message",
    ) as HTMLElement | null;
    if (msg) msg.textContent = `Error: ${message}`;
    if (errorContainer) errorContainer.style.display = "block";
    if (this.loading) this.loading.style.display = "none";
    if (this.content) this.content.style.display = "none";
  }

  private render(data: any) {
    if (!this.content) return;
    const container = this.container.querySelector(
      "#player-leaderboard-rows",
    ) as HTMLElement | null;
    if (!container) {
      this.content.innerHTML =
        '<div id="player-leaderboard-rows" class="leaderboard-table"></div>';
    }
    const rows = this.container.querySelector(
      "#player-leaderboard-rows",
    ) as HTMLElement | null;
    if (!rows) return;
    rows.innerHTML = "";

    const players = data.leaderboard || [];
    if (!players.length) {
      console.warn("[PlayerLeaderboard] empty leaderboard payload");
      this.content.innerHTML = `
        <div class="empty-state">
          <p>No player rankings available.</p>
          <p>Player data is updated regularly based on challenge mode runs.</p>
        </div>
      `;
      return;
    }

    players.forEach((player: any, index: number) => {
      const rank = data.pagination?.currentPage
        ? (data.pagination.currentPage - 1) * 25 + index + 1
        : index + 1;
      const combinedTime = player.combined_best_time
        ? this.formatDuration(player.combined_best_time)
        : "—";
      const row = document.createElement("div");
      const classColor = player.class_name
        ? getClassColor(player.class_name)
        : "#FFFFFF";
      const profileUrl = buildPlayerProfileURL(
        player.region || "us",
        player.realm_slug,
        player.name,
      );
      row.className = "leaderboard-table-row";
      row.innerHTML = `
        <div class="leaderboard-cell leaderboard-cell--rank">#${rank}</div>
        <div class="leaderboard-cell leaderboard-cell--time">${combinedTime}</div>
        <div class="leaderboard-cell leaderboard-cell--spec-icon">
          <div class="spec-icon-placeholder" data-spec-id="${player.main_spec_id || 0}"></div>
        </div>
        <div class="leaderboard-cell leaderboard-cell--content">
          <a href="${profileUrl}" class="player-link" data-class="${player.class_name || ""}"
            style="font-weight: 600; font-size: 1.0em; text-decoration: none; color: ${classColor};">
            ${player.name}
          </a>
        </div>
        <div class="leaderboard-cell leaderboard-cell--meta">${player.realm_name || player.realm_slug}</div>`;
      rows.appendChild(row);
    });


    // pagination
    const paginationContainer = this.container.querySelector(
      ".pagination-container",
    ) as HTMLElement | null;
    if (paginationContainer) {
      paginationContainer.style.display =
        data.pagination?.totalPages > 1 ? "flex" : "none";
      const prevBtn = this.container.querySelector(
        "#prev-page",
      ) as HTMLButtonElement | null;
      const nextBtn = this.container.querySelector(
        "#next-page",
      ) as HTMLButtonElement | null;
      if (prevBtn) prevBtn.disabled = !data.pagination?.hasPrevPage;
      if (nextBtn) nextBtn.disabled = !data.pagination?.hasNextPage;
      const pageInfo = this.container.querySelector(
        ".page-info",
      ) as HTMLElement | null;
      if (pageInfo)
        pageInfo.textContent = `Page ${data.pagination?.currentPage} of ${data.pagination?.totalPages}`;
    }

    // spec icons
    const placeholders = this.container.querySelectorAll(
      ".spec-icon-placeholder",
    );
    placeholders.forEach((ph) => {
      const el = ph as HTMLElement;
      const specId = parseInt(el.dataset.specId || "0");
      if (!specId) return;
      const spec = getSpecInfo(specId);
      const icon = spec ? getSpecIcon(spec.class, spec.spec) : null;
      if (icon) {
        const img = document.createElement("img");
        img.src = icon;
        img.alt = `${spec?.spec} ${spec?.class}`;
        img.className = "spec-icon";
        el.innerHTML = "";
        el.appendChild(img);
      }
    });
  }


  private async load(page: number, opts: { preserveScroll?: boolean } = {}) {
    const prevScrollY = window.scrollY;
    this.showLoading();
    try {
      const data = await this.fetchData(page);
      this.render(data);
    } catch (e: any) {
      this.showError(String(e?.message || e));
    } finally {
      this.hideLoading();
      if (opts.preserveScroll !== false) {
        window.scrollTo({ top: prevScrollY });
      } else {
        const rect = this.container.getBoundingClientRect();
        const top = window.scrollY + rect.top - 12;
        window.scrollTo({ top, behavior: "smooth" });
      }
    }
  }

  private formatDuration(ms: number) {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${s.toString().padStart(2, "0")}`;
  }
}
// Initialize immediately if DOM is already ready, otherwise on DOMContentLoaded
function initPlayerLeaderboard() {
  const attempt = () =>
    document.querySelector(
      ".player-leaderboard-container",
    ) as HTMLElement | null;
  let container = attempt();
  if (!container) {
    console.log(
      "[PlayerLeaderboard] container not found yet; observing DOM...",
    );
    const obs = new MutationObserver(() => {
      container = attempt();
      if (container) {
        obs.disconnect();
        doInit(container);
      }
    });
    obs.observe(document.documentElement || document.body, {
      childList: true,
      subtree: true,
    });
    // Also do a timed retry after a brief delay
    setTimeout(() => {
      container = attempt();
      if (container) {
        obs.disconnect();
        doInit(container);
      }
    }, 150);
    return;
  }
  doInit(container);
}

function doInit(container: HTMLElement) {
  const w: any = window as any;
  if (!w.playerLeaderboard) {
    try {
      w.playerLeaderboard = new PlayerLeaderboardClient(container);
      console.log("[PlayerLeaderboard] initialized");
      try {
        window.dispatchEvent(new CustomEvent("playerLeaderboard:ready"));
      } catch {}
    } catch (e) {
      console.error("[PlayerLeaderboard] init failed:", e);
    }
  } else {
    console.log("[PlayerLeaderboard] already initialized");
  }
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initPlayerLeaderboard);
} else {
  initPlayerLeaderboard();
}

// Expose a manual initializer for fallbacks in layout logic
try {
  (window as any).__initPlayerLeaderboard = initPlayerLeaderboard;
} catch {}
