import Fuse from "fuse.js";
import type { FuseResult } from "fuse.js";
import { buildPlayerProfileURL, debounce } from "../lib/utils.ts";
import { getClassColor } from "../lib/client-utils.ts";

type PlayerSearchResult = {
  id: number;
  name: string;
  realm_slug: string;
  realm_name: string;
  region: string;
  class_name?: string;
  active_spec_name?: string;
  global_ranking?: number;
};
type PlayerSearchIndex = { players: PlayerSearchResult[]; metadata: any };

class PlayerSearchClient {
  private container: HTMLElement;
  private searchInput: HTMLInputElement;
  private clearButton: HTMLButtonElement;
  private resultsContainer: HTMLElement;
  private resultsList: HTMLElement;
  private loadingDiv: HTMLElement;
  private noResultsDiv: HTMLElement;
  private playerIndex: PlayerSearchResult[] = [];
  private fuse: Fuse<PlayerSearchResult> | null = null;
  private isIndexLoaded = false;
  private selectedIndex = -1;
  private filteredResults: FuseResult<PlayerSearchResult>[] = [];
  private totalExpected = 0;

  constructor(container: HTMLElement) {
    this.container = container;
    this.searchInput = container.querySelector(
      "#player-search-input",
    ) as HTMLInputElement;
    this.clearButton = container.querySelector(
      "#search-clear",
    ) as HTMLButtonElement;
    this.resultsContainer = container.querySelector(
      "#search-results",
    ) as HTMLElement;
    this.resultsList = container.querySelector(
      "#search-results-list",
    ) as HTMLElement;
    this.loadingDiv = container.querySelector("#search-loading") as HTMLElement;
    this.noResultsDiv = container.querySelector(
      "#search-no-results",
    ) as HTMLElement;

    this.bindEvents();
    this.bindGlobalHotkeys();
  }

  private bindEvents() {
    const debouncedSearch = debounce((q: string) => this.performSearch(q), 300);
    this.searchInput.addEventListener("input", (e) => {
      if (!this.isIndexLoaded) this.loadPlayerIndex(1000);
      const q = (e.target as HTMLInputElement).value;
      this.updateClearButton(q);
      debouncedSearch(q);
    });
    this.searchInput.addEventListener("focus", () => {
      if (!this.isIndexLoaded) this.loadPlayerIndex(1000);
      if (this.searchInput.value.length > 0) this.showResults();
    });
    this.clearButton.addEventListener("click", () => this.clearSearch());
    this.searchInput.addEventListener("keydown", (e) => this.handleKeyboard(e));
    document.addEventListener("click", (e) => {
      if (!this.container.contains(e.target as Node)) this.hideResults();
    });
  }

  private bindGlobalHotkeys() {
    document.addEventListener("keydown", (e) => {
      if (e.key === "/" && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag =
          (document.activeElement &&
            (document.activeElement as HTMLElement).tagName) ||
          "";
        if (tag !== "INPUT" && tag !== "TEXTAREA") {
          e.preventDefault();
          this.searchInput.focus();
          this.searchInput.value = "";
          this.showResults();
        }
      }
    });
  }

  private updateClearButton(q: string) {
    this.clearButton.style.display = q.length > 0 ? "block" : "none";
  }

  private clearSearch() {
    this.searchInput.value = "";
    this.clearButton.style.display = "none";
    this.hideResults();
    this.searchInput.focus();
  }

  private showResults() {
    this.resultsContainer.style.display = "block";
    this.noResultsDiv.style.display = "none";
    this.resultsList.style.display = "block";
  }
  private hideResults() {
    this.resultsContainer.style.display = "none";
    this.selectedIndex = -1;
  }
  private showLoading() {
    this.resultsContainer.style.display = "block";
    this.loadingDiv.style.display = "flex";
    this.resultsList.style.display = "none";
    this.noResultsDiv.style.display = "none";
  }
  private hideLoading() {
    this.loadingDiv.style.display = "none";
  }
  private showNoResults() {
    this.resultsContainer.style.display = "block";
    this.noResultsDiv.style.display = "block";
    this.resultsList.style.display = "none";
    this.loadingDiv.style.display = "none";
  }

  private loadFromCache(): PlayerSearchIndex | null {
    try {
      const cached = localStorage.getItem("playerSearchIndex");
      if (!cached) return null;
      const data = JSON.parse(cached);
      if (!data?.players || data.players.length === 0) {
        console.log("[PlayerSearch] cache present but empty; ignoring");
        localStorage.removeItem("playerSearchIndex");
        return null;
      }
      const age = Date.now() - new Date(data.metadata.cached_at).getTime();
      if (age < 24 * 60 * 60 * 1000) return data;
      localStorage.removeItem("playerSearchIndex");
      return null;
    } catch {
      localStorage.removeItem("playerSearchIndex");
      return null;
    }
  }

  private saveToCache(data: PlayerSearchIndex) {
    try {
      const payload = {
        ...data,
        metadata: { ...data.metadata, cached_at: new Date().toISOString() },
      };
      localStorage.setItem("playerSearchIndex", JSON.stringify(payload));
    } catch {}
  }

  private async loadPlayerIndex(limit = 5000) {
    if (this.isIndexLoaded) return;
    const cached = this.loadFromCache();
    if (cached) {
      console.log("[PlayerSearch] using cached index:", cached.players.length);
      this.setupIndex(cached);
      return;
    }
    this.showLoading();
    try {
      // Fetch first shard to get total and initialize
      const firstUrl = `/api/search/players-000.json`;
      console.log("[PlayerSearch] fetching index:", firstUrl);
      const firstRes = await fetch(firstUrl);
      console.log(
        "[PlayerSearch] index response:",
        firstRes.status,
        firstRes.statusText,
      );
      if (!firstRes.ok)
        throw new Error(`${firstRes.status} ${firstRes.statusText}`);
      const firstData: PlayerSearchIndex = await firstRes.json();
      const firstPlayers = Array.isArray(firstData?.players)
        ? firstData.players
        : [];
      this.totalExpected = Number(
        firstData?.metadata?.total_players || firstPlayers.length,
      );
      const shardSize = Number(firstData?.metadata?.limit || limit);
      console.log(
        "[PlayerSearch] first shard size:",
        firstPlayers.length,
        "total expected:",
        this.totalExpected,
        "shard size:",
        shardSize,
      );

      // Initialize with first shard
      this.playerIndex = firstPlayers;
      this.isIndexLoaded = true;
      this.fuse = new Fuse(this.playerIndex, {
        keys: [
          { name: "name", weight: 0.6 },
          { name: "realm_name", weight: 0.2 },
          { name: "region", weight: 0.2 },
          { name: "class_name", weight: 0.1 },
          { name: "active_spec_name", weight: 0.1 },
        ],
        threshold: 0.35,
        includeScore: true,
      });

      // Continue fetching remaining shards in background
      const total = this.totalExpected;
      if (total > firstPlayers.length) {
        const totalShards = Math.ceil(total / shardSize);
        for (let s = 1; s < totalShards; s++) {
          const shardNum = s.toString().padStart(3, '0');
          const url = `/api/search/players-${shardNum}.json`;
          try {
            const res = await fetch(url);
            if (!res.ok) break;
            const data: PlayerSearchIndex = await res.json();
            const chunk = Array.isArray(data?.players) ? data.players : [];
            if (chunk.length === 0) break;
            // merge and rebuild index periodically
            this.playerIndex.push(...chunk);
            if (s % 2 === 0 || s === totalShards - 1) {
              this.fuse = new Fuse(this.playerIndex, {
                keys: [
                  { name: "name", weight: 0.6 },
                  { name: "realm_name", weight: 0.2 },
                  { name: "region", weight: 0.2 },
                  { name: "class_name", weight: 0.1 },
                  { name: "active_spec_name", weight: 0.1 },
                ],
                threshold: 0.35,
                includeScore: true,
              });
              // rerun current query if user typed something
              const q = this.searchInput.value.trim();
              if (q) this.performSearch(q);
            }
          } catch (e) {
            console.warn(`[PlayerSearch] shard ${shardNum} fetch failed:`, e);
            break;
          }
        }
        // Save final combined index
        this.saveToCache({
          players: this.playerIndex,
          metadata: { total_players: this.playerIndex.length },
        });
      } else {
        // Save small index
        this.saveToCache({
          players: this.playerIndex,
          metadata: firstData.metadata,
        });
      }
    } catch (e) {
      console.error("Failed to load player search index:", e);
      this.showNoResults();
    } finally {
      this.hideLoading();
    }
  }

  private setupIndex(data: PlayerSearchIndex) {
    this.playerIndex = data.players;
    this.isIndexLoaded = true;
    this.fuse = new Fuse(this.playerIndex, {
      keys: [
        { name: "name", weight: 0.6 },
        { name: "realm_name", weight: 0.2 },
        { name: "region", weight: 0.2 },
      ],
      threshold: 0.35,
      includeScore: true,
    });
  }

  private handleKeyboard(e: KeyboardEvent) {
    // If results are hidden, ignore navigation keys
    if (
      !this.resultsContainer.style.display ||
      this.resultsContainer.style.display === "none"
    )
      return;

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        this.selectedIndex = Math.min(
          this.selectedIndex + 1,
          this.filteredResults.length - 1,
        );
        this.updateSelection();
        break;
      case "ArrowUp":
        e.preventDefault();
        this.selectedIndex = Math.max(this.selectedIndex - 1, 0);
        this.updateSelection();
        break;
      case "Tab": {
        // Cycle with Tab; Shift+Tab cycles backwards
        e.preventDefault();
        if (this.filteredResults.length === 0) return;
        if (e.shiftKey) {
          this.selectedIndex =
            this.selectedIndex <= 0
              ? this.filteredResults.length - 1
              : this.selectedIndex - 1;
        } else {
          this.selectedIndex =
            this.selectedIndex >= this.filteredResults.length - 1
              ? 0
              : this.selectedIndex + 1;
        }
        this.updateSelection();
        break;
      }
      case "Enter": {
        e.preventDefault();
        if (
          this.selectedIndex >= 0 &&
          this.selectedIndex < this.filteredResults.length
        ) {
          const sel = this.resultsList.querySelectorAll(".search-result-item")[
            this.selectedIndex
          ] as HTMLElement | null;
          sel?.click();
        } else {
          const first = this.resultsList.querySelector(
            ".search-result-item",
          ) as HTMLElement | null;
          first?.click();
        }
        break;
      }
      case "Escape":
        this.hideResults();
        break;
    }
  }

  private updateSelection() {
    const items = Array.from(
      this.resultsList.querySelectorAll(".search-result-item"),
    ) as HTMLElement[];
    items.forEach((el, idx) => {
      el.classList.toggle("selected", idx === this.selectedIndex);
      if (idx === this.selectedIndex) el.scrollIntoView({ block: "nearest" });
    });
  }

  private performSearch(query: string) {
    if (!this.fuse || !query) {
      this.resultsList.innerHTML = "";
      this.hideLoading();
      return;
    }
    this.filteredResults = this.fuse.search(query, { limit: 20 });
    if (!this.filteredResults.length) {
      this.showNoResults();
      return;
    }
    this.showResults();
    this.resultsList.innerHTML = "";
    this.filteredResults.forEach(({ item, score }, idx) => {
      const url = buildPlayerProfileURL(
        item.region || "us",
        item.realm_slug,
        item.name,
      );
      const el = document.createElement("div");
      el.className = "search-result-item";
      const relevance =
        score && score > 0.1
          ? `<span class="search-relevance" title="Relevance: ${(1 - score).toFixed(2)}">~</span>`
          : "";
      const color = item.class_name ? getClassColor(item.class_name) : "";
      el.innerHTML = `
        <div class="search-result-content">
          <div class="search-player-identity">
            ${relevance}<span class="search-player-name" style="color: ${color};">${item.name}</span>
            <span class="search-region-badge">${(item.region || "").toUpperCase()}</span>
            <span class="search-player-realm">${item.realm_name}</span>
          </div>
        </div>
      `;
      // mouse interaction highlights
      el.addEventListener("mouseenter", () => {
        this.selectedIndex = idx;
        this.updateSelection();
      });
      el.addEventListener("click", () => {
        window.location.href = url;
      });
      this.resultsList.appendChild(el);
    });
    // reset selection to first result
    this.selectedIndex = this.filteredResults.length > 0 ? 0 : -1;
    this.updateSelection();
  }
}

function initPlayerSearch() {
  const container = document.querySelector(
    ".player-search-island",
  ) as HTMLElement | null;
  if (!container) return;
  if (!(window as any).__playerSearchInit) {
    try {
      (window as any).__playerSearchInit = true;
      new PlayerSearchClient(container);
      console.log("[PlayerSearch] initialized");
    } catch (e) {
      console.error("[PlayerSearch] init failed:", e);
    }
  }
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initPlayerSearch);
} else {
  initPlayerSearch();
}
