import { REALM_DATA } from "../lib/realms";
import { buildPlayersLeaderboardURL } from "../lib/utils.ts";

class PlayerFilterPanelClient {
  private container: HTMLElement;
  private regionSelect: HTMLSelectElement;
  private realmSelect: HTMLSelectElement;
  private classSelect: HTMLSelectElement;

  constructor(container: HTMLElement) {
    this.container = container;
    this.regionSelect = container.querySelector(
      "#player-region-select",
    ) as HTMLSelectElement;
    this.realmSelect = container.querySelector(
      "#player-realm-select",
    ) as HTMLSelectElement;
    this.classSelect = container.querySelector(
      "#player-class-select",
    ) as HTMLSelectElement;

    this.bindEvents();
    this.updateRealmOptions();
    this.initFromURLOrDataset();
  }

  private bindEvents() {
    this.regionSelect.addEventListener("change", () => {
      this.updateRealmOptions();
      this.notifyLeaderboard(1);
    });
    this.realmSelect.addEventListener("change", () => this.notifyLeaderboard(1));
    this.classSelect.addEventListener("change", () => this.notifyLeaderboard(1));
  }

  private updateRealmOptions() {
    const region = this.regionSelect.value;
    const options = this.realmSelect.querySelectorAll("option[data-region]");
    options.forEach((opt) => ((opt as HTMLOptionElement).style.display = "none"));
    if (region === "global") {
      this.realmSelect.disabled = true;
      this.realmSelect.value = "all";
    } else {
      this.realmSelect.disabled = false;
      const regionOptions = this.realmSelect.querySelectorAll(
        `option[data-region="${region}"]`,
      );
      regionOptions.forEach(
        (opt) => ((opt as HTMLOptionElement).style.display = "block"),
      );
      // Ensure valid selection
      const current = this.realmSelect.value;
      if (current && current !== "all") {
        const exists = Array.from(regionOptions).some(
          (opt) => (opt as HTMLOptionElement).value === current,
        );
        if (!exists) this.realmSelect.value = "all";
      } else {
        this.realmSelect.value = "all";
      }
    }
  }

  private initFromURLOrDataset() {
    const { scope, region, realm, klass, page } = this.parsePlayersPath();
    this.regionSelect.value = region || "global";
    this.updateRealmOptions();
    if (region && region !== "global") this.realmSelect.value = realm || "all";
    this.classSelect.value = klass || "";
    // Only notify/load if players section is currently active; otherwise wait for tab switch
    if (this.isPlayersSectionActive()) {
      this.notifyLeaderboard(page || 1);
    }
  }

  private notifyLeaderboard(page: number) {
    const region = this.regionSelect.value;
    const realmVal = this.realmSelect.value || "all";
    const classKey = this.classSelect.value || "";

    const scope = region === "global" ? "global" : realmVal !== "all" ? "realm" : "regional";
    const realmSlug = scope === "realm" ? realmVal : undefined;

    const w: any = window as any;
    const pl = w.playerLeaderboard;
    if (pl && typeof pl.setFilters === "function") {
      pl.setFilters({ scope, region: region === "global" ? undefined : region, realmSlug, classKey, page });
    }

    const newPath = buildPlayersLeaderboardURL(scope, {
      region: region === "global" ? undefined : region,
      realmSlug,
      classKey,
      page,
    });
    try { sessionStorage.setItem("lastPlayersPath", newPath); } catch {}
    window.history.pushState({}, "", newPath);
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

  private parsePlayersPath(): { scope: "global" | "regional" | "realm"; region?: string; realm?: string; klass?: string; page?: number } {
    const path = window.location.pathname.replace(/\/+$/, "");
    const parts = path.split("/").filter(Boolean);
    // Expect: ["challenge-mode","players", ...]
    if (parts.length >= 2 && parts[0] === "challenge-mode" && parts[1] === "players") {
      const rest = parts.slice(2);
      if (rest.length === 0) {
        return { scope: "global", klass: undefined, page: this.readPageParam() };
      }
      if (rest[0] === "global") {
        const klass = rest[1] || "";
        return { scope: "global", klass, page: this.readPageParam() };
      }
      const region = rest[0];
      if (rest.length === 1) {
        return { scope: "regional", region, page: this.readPageParam() };
      }
      // Two or more segments: could be region/class or region/realm[/class]
      if (rest.length === 2) {
        // Ambiguous; assume region/class if second looks like a class (letters/underscores)
        const second = rest[1];
        if (/^[a-z_]+$/.test(second)) {
          return { scope: "regional", region, klass: second, page: this.readPageParam() };
        }
        return { scope: "realm", region, realm: second, page: this.readPageParam() };
      }
      // region/realm/class
      return { scope: "realm", region, realm: rest[1], klass: rest[2], page: this.readPageParam() };
    }
    // Fallback: derive from query params as legacy
    const url = new URL(window.location.href);
    const region = url.searchParams.get("region") || undefined;
    const realm = url.searchParams.get("realm") || undefined;
    const klass = url.searchParams.get("class") || undefined;
    const page = this.readPageParam();
    if (!region) return { scope: "global", klass, page };
    if (realm) return { scope: "realm", region, realm, klass, page };
    return { scope: "regional", region, klass, page };
  }

  private readPageParam(): number | undefined {
    const url = new URL(window.location.href);
    const page = parseInt(url.searchParams.get("page") || "1", 10);
    return isNaN(page) || page <= 1 ? undefined : page;
  }
}

function initPlayerFilterPanel() {
  const el = document.querySelector(
    ".player-filter-panel",
  ) as HTMLElement | null;
  if (el) new PlayerFilterPanelClient(el);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initPlayerFilterPanel);
} else {
  initPlayerFilterPanel();
}
