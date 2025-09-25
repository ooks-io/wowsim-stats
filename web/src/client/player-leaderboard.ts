import { buildPlayerProfileURL, buildStaticPlayerLeaderboardPath } from "../lib/utils.ts";
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
  private scope: "global" | "regional" | "realm" = "global";
  private region: string | null = null;
  private realmSlug: string | null = null;
  private classKey: string | null = null;

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
    // initial load only if players section is active; otherwise the layout will trigger refresh on tab switch
    if (this.isPlayersSectionActive()) {
      this.refreshCurrentPage();
    }
  }

  private isPlayersSectionActive(): boolean {
    try {
      const section = this.container.closest("#players-section") as
        | HTMLElement
        | null;
      return !!section && section.classList.contains("active");
    } catch {
      return false;
    }
  }

  public refreshCurrentPage() {
    const { scope, region, realm, klass, page } = this.parsePlayersPath();
    this.currentPage = page || 1;
    this.scope = scope;
    this.region = region || null;
    this.realmSlug = realm || null;
    this.classKey = klass || null;
    this.load(this.currentPage).catch((e) =>
      console.error("Player leaderboard load error:", e),
    );
  }

  private parsePlayersPath(): { scope: "global" | "regional" | "realm"; region?: string; realm?: string; klass?: string; page?: number } {
    const path = window.location.pathname.replace(/\/+$/, "");
    const parts = path.split("/").filter(Boolean);
    const url = new URL(window.location.href);
    const page = parseInt(url.searchParams.get("page") || "1", 10);
    const pageNum = isNaN(page) || page <= 1 ? undefined : page;
    if (parts.length >= 2 && parts[0] === "challenge-mode" && parts[1] === "players") {
      const rest = parts.slice(2);
      if (rest.length === 0) return { scope: "global", page: pageNum };
      if (rest[0] === "global") return { scope: "global", klass: rest[1], page: pageNum };
      const region = rest[0];
      if (rest.length === 1) return { scope: "regional", region, page: pageNum };
      if (rest.length === 2) {
        const second = rest[1];
        if (/^[a-z_]+$/.test(second)) return { scope: "regional", region, klass: second, page: pageNum };
        return { scope: "realm", region, realm: second, page: pageNum };
      }
      return { scope: "realm", region, realm: rest[1], klass: rest[2], page: pageNum };
    }
    // Fallback to query params
    const regionQ = url.searchParams.get("region") || undefined;
    const realmQ = url.searchParams.get("realm") || undefined;
    const klassQ = url.searchParams.get("class") || undefined;
    if (!regionQ) return { scope: "global", klass: klassQ, page: pageNum };
    if (realmQ) return { scope: "realm", region: regionQ, realm: realmQ, klass: klassQ, page: pageNum };
    return { scope: "regional", region: regionQ, klass: klassQ, page: pageNum };
  }

  public setFilters(opts: {
    scope: "global" | "regional" | "realm";
    region?: string;
    realmSlug?: string;
    classKey?: string;
    page?: number;
  }) {
    this.scope = opts.scope;
    this.region = opts.region || null;
    this.realmSlug = opts.realmSlug || null;
    this.classKey = opts.classKey || null;
    this.currentPage = Math.max(1, opts.page || 1);
    this.load(this.currentPage).catch((e) =>
      console.error("Player leaderboard setFilters error:", e),
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
        this.load(this.currentPage);
      }
    });
    nextBtn?.addEventListener("click", () => {
      this.currentPage++;
      this.load(this.currentPage);
    });
  }

  private async fetchData(page: number) {
    const url = buildStaticPlayerLeaderboardPath(
      this.scope,
      this.region || undefined,
      page,
      { realmSlug: this.realmSlug || undefined, classKey: this.classKey || undefined },
    );

    console.log("[PlayerLeaderboard] fetching:", url);
    const res = await fetch(url);
    console.log("[PlayerLeaderboard] response:", res.status, res.statusText);
    if (!res.ok) {
      if (res.status === 404) {
        throw new Error(
          "Player rankings not found. No players have complete coverage for this scope yet.",
        );
      } else if (res.status >= 500) {
        throw new Error(
          "Server error occurred while loading player rankings. Please try again later.",
        );
      } else {
        throw new Error(
          `Failed to load player rankings: ${res.status} ${res.statusText}`,
        );
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
        : "-";
      const row = document.createElement("div");
      let classColor = "#FFFFFF";
      if (player.class_name) {
        classColor = getClassColor(player.class_name);
      } else if (player.main_spec_id) {
        const si = getSpecInfo(Number(player.main_spec_id));
        if (si && si.class) {
          classColor = getClassColor(si.class);
        }
      }
      const profileUrl = buildPlayerProfileURL(
        player.region || "us",
        player.realm_slug,
        player.name,
      );
      row.className = "leaderboard-table-row";
      row.dataset.rank = `#${rank}`;
      row.dataset.time = combinedTime;
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
        <div class="leaderboard-cell leaderboard-cell--meta">${player.realm_name || player.realm_slug}</div>
        <div class="mobile-player-info">
          <div class="mobile-player-spec">
            <div class="spec-icon-placeholder" data-spec-id="${player.main_spec_id || 0}"></div>
            <div>
              <a href="${profileUrl}" class="player-link mobile-player-name" data-class="${player.class_name || ""}"
                style="color: ${classColor}; text-decoration: none;">
                ${player.name}
              </a>
              <div class="mobile-player-realm">
                ${player.realm_name || player.realm_slug}
              </div>
            </div>
          </div>
        </div>`;
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
      // Update URL to reflect current filters + page
      try {
        // Keep current path; only update page param to reflect pagination
        const url = new URL(window.location.href);
        if (this.currentPage > 1) url.searchParams.set("page", String(this.currentPage)); else url.searchParams.delete("page");
        window.history.replaceState({}, "", url.toString());
      } catch {}
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
