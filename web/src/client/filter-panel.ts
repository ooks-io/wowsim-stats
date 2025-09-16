import { buildLeaderboardURL } from "../lib/utils.ts";
import { dungeonSlugToId } from "../lib/wow-constants.ts";

class FilterPanel {
  private container: HTMLElement;
  private regionSelect: HTMLSelectElement;
  private realmSelect: HTMLSelectElement;
  private dungeonSelect: HTMLSelectElement;
  private statusDiv: HTMLElement;
  private didInitFromURL = false;

  constructor(container: HTMLElement) {
    this.container = container;
    this.regionSelect = container.querySelector(
      "#region-select",
    ) as HTMLSelectElement;
    this.realmSelect = container.querySelector(
      "#realm-select",
    ) as HTMLSelectElement;
    this.dungeonSelect = container.querySelector(
      "#dungeon-select",
    ) as HTMLSelectElement;
    this.statusDiv = container.querySelector("#filter-status") as HTMLElement;

    this.bindEvents();
    this.updateRealmOptions();

    // initialize defaults from data attributes
    const initRegion = container.dataset.initialRegion || "";
    const initRealm = container.dataset.initialRealm || "";
    const initDungeon = container.dataset.initialDungeon || "";
    if (initRegion) {
      this.regionSelect.value = initRegion;
      this.updateRealmOptions();
    }
    if (initRealm) this.realmSelect.value = initRealm;
    if (initDungeon) this.dungeonSelect.value = initDungeon;

    this.initializeFromURL();
    // If URL didn't drive an initial load, only auto-load if the dungeon section is active
    if (!this.didInitFromURL && this.isDungeonSectionActive()) {
      const region = this.regionSelect.value;
      const dungeon = this.dungeonSelect.value;
      if (region && dungeon) this.loadWhenReady(1);
    }
  }

  private bindEvents() {
    this.regionSelect.addEventListener("change", () => {
      this.updateRealmOptions();
      this.loadLeaderboard();
    });
    this.realmSelect.addEventListener("change", () => this.loadLeaderboard());
    this.dungeonSelect.addEventListener("change", () => this.loadLeaderboard());
  }

  private isDungeonSectionActive(): boolean {
    try {
      const section = this.container.closest("#dungeon-section") as
        | HTMLElement
        | null;
      return !!section && section.classList.contains("active");
    } catch {
      return false;
    }
  }

  private updateRealmOptions() {
    const region = this.regionSelect.value;
    const allOptions = this.realmSelect.querySelectorAll("option[data-region]");
    allOptions.forEach(
      (opt) => ((opt as HTMLOptionElement).style.display = "none"),
    );
    if (region === "global") {
      this.realmSelect.disabled = true;
      this.realmSelect.value = "";
    } else {
      this.realmSelect.disabled = false;
      const regionOptions = this.realmSelect.querySelectorAll(
        `option[data-region="${region}"]`,
      );
      regionOptions.forEach(
        (opt) => ((opt as HTMLOptionElement).style.display = "block"),
      );
    }
  }

  private updateStatus(message: string, type: string) {
    const statusText = this.statusDiv.querySelector(
      ".status-text",
    ) as HTMLElement | null;
    if (statusText) statusText.textContent = message;
    this.statusDiv.style.display = "block";
    this.statusDiv.className = `filter-status ${type}`;
  }

  public async loadLeaderboard(targetPage = 1) {
    const region = this.regionSelect.value;
    const realm = this.realmSelect.value;
    const dungeon = this.dungeonSelect.value;
    if (!region || !dungeon) return;
    const effectiveRealm = realm || (region === "global" ? "" : "all");
    try {
      const table = (window as any).leaderboardTable;
      if (table && typeof table.loadLeaderboard === "function") {
        await table.loadLeaderboard(
          region,
          effectiveRealm,
          dungeon,
          targetPage,
        );
      }
      const selected = this.dungeonSelect.querySelector(
        `option[value="${dungeon}"]`,
      ) as HTMLOptionElement | null;
      const dungeonName = selected?.textContent ?? "Unknown Dungeon";
      const slug = dungeonName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "");
      let newURL = buildLeaderboardURL(region, effectiveRealm, slug);
      if (targetPage > 1) newURL += `?page=${targetPage}`;
      window.history.pushState({}, "", newURL);
    } catch (e) {
      console.error("Filter panel error:", e);
      this.updateStatus(`Error loading leaderboard: ${e}`, "invalid");
    }
  }

  private loadWhenReady(targetPage = 1) {
    const tryLoad = () => {
      const table = (window as any).leaderboardTable;
      if (table && typeof table.loadLeaderboard === "function") {
        this.loadLeaderboard(targetPage);
        return true;
      }
      return false;
    };

    if (!tryLoad()) {
      const onReady = () => {
        window.removeEventListener("leaderboard:ready", onReady as any);
        tryLoad();
      };
      window.addEventListener("leaderboard:ready", onReady as any, {
        once: true,
      });
    }
  }

  private initializeFromURL() {
    const path = window.location.pathname;
    const urlParams = new URLSearchParams(window.location.search);
    const pageParam = urlParams.get("page");
    const targetPage = pageParam ? parseInt(pageParam) : 1;
    const m = path.match(/^\/challenge-mode\/([^\/]+)\/([^\/]+)\/([^\/]+)$/);
    if (m) {
      const [, region, realm, dungeonSlug] = m;
      const dungeonId = dungeonSlugToId(dungeonSlug);
      if (dungeonId) {
        this.regionSelect.value = region;
        this.updateRealmOptions();
        if (realm !== "all") this.realmSelect.value = realm;
        this.dungeonSelect.value = String(dungeonId);
        setTimeout(() => this.loadWhenReady(targetPage), 100);
        this.didInitFromURL = true;
      }
    }
  }
}

document.addEventListener("DOMContentLoaded", () => {
  const container = document.querySelector(
    ".filter-panel-island",
  ) as HTMLElement | null;
  if (container) (window as any).filterPanel = new FilterPanel(container);
});
