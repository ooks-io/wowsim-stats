import {
  formatDurationFromMs,
  buildStaticPlayerProfilePath,
  formatDurationMMSS,
} from "../lib/utils";
import { getClassTextClass } from "../lib/wow-constants";
import { formatRankingWithBracket as formatRank } from "../lib/client-utils.ts";
import { renderBestRunsWithWrapper } from "../lib/bestRunsRenderer";
import {
  getOrderedEquipment,
  processEquipmentEnchantments,
  getZamimgIconUrl,
  getWowheadUrl,
  getQualityColorClass,
} from "../lib/equipment-utils";
import "./best-runs-decorate";

interface PlayerData {
  Player: {
    name: string;
    region: string;
    realm_name: string;
    realm_slug: string;
    class_name: string;
    spec_name: string;
    race: string;
    level: number;
    avatar_url?: string;
  };
  Equipment?: Record<string, any>;
  BestRuns?: Record<string, any>;
}

class PlayerProfileManager {
  private container: HTMLElement;
  private region: string;
  private realmSlug: string;
  private playerName: string;
  private debugInfo: string[] = [];

  constructor(container: HTMLElement) {
    this.container = container;
    this.region = container.dataset.region || "";
    this.realmSlug = container.dataset.realm || "";
    this.playerName = container.dataset.player || "";

    console.log("[INFO] PlayerProfile initialized:", {
      region: this.region,
      realmSlug: this.realmSlug,
      playerName: this.playerName,
    });

    // Only load if we have the required parameters
    if (this.region && this.realmSlug && this.playerName) {
      this.loadPlayerProfile();
    } else {
      this.showError("Missing required parameters");
      this.addDebugInfo("Missing URL parameters");
      this.addDebugInfo(`Region: ${this.region}`);
      this.addDebugInfo(`Realm: ${this.realmSlug}`);
      this.addDebugInfo(`Player: ${this.playerName}`);
    }
  }

  private addDebugInfo(info: string) {
    this.debugInfo.push(info);
  }

  private async loadPlayerProfile() {
    try {
      this.showLoading();
      this.addDebugInfo(
        `Loading player: ${this.region}/${this.realmSlug}/${this.playerName}`,
      );

      const staticPath = buildStaticPlayerProfilePath(
        this.region,
        this.realmSlug,
        this.playerName,
      );
      this.addDebugInfo(`Fetching: ${staticPath}`);

      console.log("[INFO] Fetching player data from:", staticPath);

      const response = await fetch(staticPath);
      this.addDebugInfo(`Response status: ${response.status}`);

      console.log("[INFO] Response status:", response.status);

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(
            `Player "${this.playerName}" not found. Only players with complete coverage (9/9 recorded dungeons) are included.`,
          );
        } else {
          const errorText = await response.text();
          this.addDebugInfo(`Error response: ${errorText.substring(0, 100)}`);
          throw new Error(
            `Failed to load player data: ${response.status} ${response.statusText}`,
          );
        }
      }

      const data: PlayerData = await response.json();
      console.log("[INFO] Player data loaded successfully:", data);
      this.addDebugInfo("Data loaded successfully");

      this.renderPlayerProfile(data);
      this.showContent();
    } catch (error: any) {
      console.error("[ERROR] Error loading player profile:", error);
      this.addDebugInfo(`Fetch error: ${error.message}`);
      this.showError(error.message);
    }
  }

  private renderPlayerProfile(data: PlayerData) {
    const contentContainer = this.container.querySelector("#profile-content");
    if (!contentContainer) return;

    // Handle both possible field names and log the data structure for debugging
    console.log("[INFO] Player data structure:", data);
    const player: any = (data as any).Player || (data as any).player;
    const equipment: Record<string, any> =
      (data as any).Equipment || (data as any).equipment || {};
    const bestRuns: Record<string, any> =
      (data as any).BestRuns || (data as any).bestRuns || {};

    if (!player) {
      console.error("[ERROR] No player data found in response:", data);
      this.addDebugInfo("No player data found in response");
      this.addDebugInfo(`Data keys: ${Object.keys(data).join(", ")}`);
      this.showError("Player data is missing from response");
      return;
    }

    console.log("[INFO] Player object:", player);

    // Build header markup matching PlayerHeader styles
    const avatar = player.avatar_url ? `${player.avatar_url}` : "";
    const combined = player.combined_best_time
      ? formatDurationMMSS(player.combined_best_time)
      : "-";
    const guild = player.guild_name ? `&lt;${player.guild_name}&gt;` : "";
    const globalRank = player.global_ranking;
    const regRank = player.regional_ranking;
    const realmRank = player.realm_ranking;
    const globalBracket = player.global_ranking_bracket || "";
    const regBracket = player.regional_ranking_bracket || "";
    const realmBracket = player.realm_ranking_bracket || "";

    const classText = getClassTextClass(player.class_name || "common");
    const headerHTML = `
      <div class="player-header-horizontal">
        <div class="player-avatar">
          ${avatar ? `<img src="${avatar}" alt="${player.name} avatar" class="avatar-image" />` : ""}
        </div>
        <div class="player-info-column">
          <h1 class="player-name ${classText}">${player.name}</h1>
          ${guild ? `<div class="guild-name">${guild}</div>` : ""}
          <div class="character-details">${player.race_name || ""} ${player.active_spec_name || player.spec_name || ""} ${player.class_name || ""}</div>
          <div class="item-level">Item Level: ${player.equipped_item_level ?? "-"} equipped / ${player.average_item_level ?? "-"} average</div>
        </div>
        <div class="combined-time-column">
          <div class="stat-label">Combined Time</div>
          <div class="stat-value combined-time-value ${globalBracket ? `bracket-${globalBracket}` : ""}">${combined}</div>
        </div>
        <div class="ranking-column">
          <div class="stat-label">Global Rank</div>
          <div class="stat-value">${globalRank ? formatRank(globalRank, globalBracket) : "-"}</div>
        </div>
        <div class="ranking-column">
          <div class="stat-label">${(player.region || "").toUpperCase()} Rank</div>
          <div class="stat-value">${regRank ? formatRank(regRank, regBracket) : "-"}</div>
        </div>
        <div class="ranking-column">
          <div class="stat-label">${player.realm_name || player.realm_slug} Rank</div>
          <div class="stat-value">${realmRank ? formatRank(realmRank, realmBracket) : "-"}</div>
        </div>
        <div class="ranking-column">
          <div class="stat-label">Total Runs</div>
          <div class="stat-value">${player.total_runs || 0}</div>
        </div>
      </div>
    `;

    // Build equipment grid similar to PlayerEquipment
    const equipmentHTML = this.renderEquipmentCompact(equipment);

    // Build best runs table using shared renderer for consistent markup/styles
    const bestRunsHTML = renderBestRunsWithWrapper(
      bestRuns,
      { showSectionWrapper: true },
      player.region,
      player.realm_slug,
    );

    const profileHTML = `${headerHTML}${equipmentHTML}${bestRunsHTML}`;
    (contentContainer as HTMLElement).innerHTML = profileHTML;

    // Decorate best runs (spec icons and colors) if helper is available
    const decorate = (window as any).__decorateBestRuns;
    if (typeof decorate === "function") {
      try {
        decorate(contentContainer as HTMLElement);
      } catch {}
    }
  }

  private renderEquipmentCompact(equipment: Record<string, any>): string {
    if (!equipment || Object.keys(equipment).length === 0) {
      return `
        <div class="player-equipment">
          <div class="equipment-section">
            <h3 class="equipment-title">Current Equipment</h3>
            <div class="no-equipment"><p>No equipment data available</p></div>
          </div>
        </div>
      `;
    }

    let ordered: any[] = [];
    try {
      ordered = processEquipmentEnchantments(getOrderedEquipment(equipment));
    } catch (e) {
      console.warn("Equipment processing failed:", e);
      // fallback to raw order
      ordered = Object.values(equipment).filter(Boolean);
    }

    const iconsHTML = ordered
      .map((item: any) => {
        if (!item) return "";
        const iconSlug = item.item_icon_slug || item.icon || "";
        const iconUrl = iconSlug ? getZamimgIconUrl(iconSlug, "large") : "";
        const qualityClass = getQualityColorClass(item.quality || "COMMON");
        const itemId = item.item_id || item.id;
        const href = itemId ? getWowheadUrl(itemId) : undefined;
        const title = item.item_name || item.name || "Item";
        const gems = (item.enchantments || []).filter(
          (e: any) => e.gem_icon_slug,
        );

        return `
          <a ${href ? `href="${href}"` : ""} class="equip-icon-wrap ${qualityClass}" title="${title}" target="_blank" rel="noopener noreferrer">
            <div class="equip-icon-inner">
              ${iconUrl ? `<img class="equip-icon" src="${iconUrl}" alt="${title}" loading="lazy" />` : `<div class="equip-icon placeholder" aria-label="${title}"></div>`}
              ${
                gems && gems.length > 0
                  ? `
                <div class="gem-stack" aria-hidden="true">
                  ${gems
                    .slice(0, 3)
                    .map((g: any) => {
                      const gUrl = g.gem_icon_slug
                        ? getZamimgIconUrl(g.gem_icon_slug, "small")
                        : "";
                      const gAlt = g.gem_name || g.display_string || "Gem";
                      return `<img class=\"gem-badge\" src=\"${gUrl}\" alt=\"${gAlt}\" title=\"${gAlt}\" />`;
                    })
                    .join("")}
                </div>
              `
                  : ""
              }
            </div>
          </a>
        `;
      })
      .join("");

    return `
      <div class="player-equipment">
        <div class="equipment-section">
          <h3 class="equipment-title">Current Equipment</h3>
          <div class="equipment-compact">
            <div class="equipment-icon-grid">${iconsHTML}</div>
          </div>
        </div>
      </div>
    `;
  }

  private showLoading() {
    this.hideAll();
    const loading = this.container.querySelector("#profile-loading");
    if (loading) (loading as HTMLElement).style.display = "block";
  }

  private showError(message: string) {
    this.hideAll();
    const error = this.container.querySelector("#profile-error");
    if (error) {
      (error as HTMLElement).style.display = "block";
      const messageEl = error.querySelector(".loading-message");
      if (messageEl) messageEl.textContent = message;
    }
  }

  private showContent() {
    this.hideAll();
    const content = this.container.querySelector("#profile-content");
    if (content) (content as HTMLElement).style.display = "block";
  }

  private hideAll() {
    const states = ["#profile-loading", "#profile-error", "#profile-content"];
    states.forEach((selector) => {
      const el = this.container.querySelector(selector);
      if (el) (el as HTMLElement).style.display = "none";
    });
  }
}

// Initialize player profile when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
  const containers = document.querySelectorAll(".player-profile-container");
  containers.forEach((container) => {
    new PlayerProfileManager(container as HTMLElement);
  });
});

// Also initialize if DOM is already ready (for dynamic loading)
if (document.readyState === "loading") {
  // DOM is still loading, wait for DOMContentLoaded
} else {
  // DOM is already ready
  const containers = document.querySelectorAll(".player-profile-container");
  containers.forEach((container) => {
    new PlayerProfileManager(container as HTMLElement);
  });
}
