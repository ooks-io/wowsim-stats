// shared equipment utilities for consistent rendering

// import color functions from canonical WoW constants
export { getClassColor, getClassColorClass } from "./wow-constants.ts";

export function getQualityColorClass(quality: string): string {
  const qualityMap = {
    POOR: "quality-poor",
    COMMON: "quality-common",
    UNCOMMON: "quality-uncommon",
    RARE: "quality-rare",
    EPIC: "quality-epic",
    LEGENDARY: "quality-legendary",
    ARTIFACT: "quality-artifact",
    HEIRLOOM: "quality-heirloom",
  };
  return qualityMap[quality as keyof typeof qualityMap] || "quality-common";
}

// equipment slot ordering for display
export const EQUIPMENT_SLOT_ORDER = [
  "HEAD",
  "NECK",
  "SHOULDER",
  "BACK",
  "CHEST",
  "WRIST",
  "HANDS",
  "WAIST",
  "LEGS",
  "FEET",
  "FINGER_1",
  "FINGER_2",
  "TRINKET_1",
  "TRINKET_2",
  "MAIN_HAND",
  "OFF_HAND",
  "RANGED",
] as const;

// simulation slot names mapping (from chart-logic.js)
export const SIMULATION_SLOT_NAMES = [
  "Head",
  "Neck",
  "Shoulders",
  "Back",
  "Chest",
  "Wrists",
  "Hands",
  "Waist",
  "Legs",
  "Feet",
  "Ring 1",
  "Ring 2",
  "Trinket 1",
  "Trinket 2",
  "Main Hand",
  "Off Hand",
] as const;

interface ItemData {
  slot?: string;
  slot_type?: string;
  item_id?: number;
  id?: number;
  item_name?: string;
  name?: string;
  quality?: string;
  upgrade_id?: number;
  stats?: { ilvl?: number };
  enchantments?: Array<{
    slot_id: number;
    slot_type?: string;
    display_string: string;
    gem_name?: string;
    gem_icon_slug?: string;
  }>;
  gems?: Array<
    | {
        name?: string;
        icon?: string;
        stats?: string[];
      }
    | number
  >;
  enchant?:
    | {
        name?: string;
      }
    | number;
  reforging?:
    | {
        description?: string;
      }
    | number;
  tinker?: number;
  item_icon_slug?: string;
  icon?: string;
  itemDetails?: string[];
}

interface CreateItemElementOptions {
  isHTML?: boolean;
  showIcon?: boolean;
  showSlotHeader?: boolean;
  iconSize?: "small" | "medium" | "large";
}

// unified function to create equipment item element/HTML
// handles both Blizzard API data and simulation data formats
export function createItemElement(
  itemData: ItemData,
  options: CreateItemElementOptions = {},
): string | HTMLElement {
  const {
    isHTML = false,
    showIcon = true,
    showSlotHeader = true,
    iconSize = "medium",
  } = options;

  // normalize slot name
  const slot = formatSlotName(itemData.slot || itemData.slot_type || "");

  // item basics
  const itemId = itemData.item_id || itemData.id;
  const itemName = itemData.item_name || itemData.name || `Item ${itemId}`;
  const quality = itemData.quality || "COMMON";
  const qualityClass = getQualityColorClass(quality);

  // build enhancement details
  const details = buildItemDetails(itemData);

  // icon handling (with fallback)
  const iconHtml = showIcon ? buildIconHtml(itemData, iconSize) : "";

  const itemHtml = `
    <div class="equipment-slot">
      ${
        showSlotHeader
          ? `
        <div class="equipment-slot-header">
          <span class="equipment-slot-name">${slot}</span>
        </div>
      `
          : ""
      }
      
      <div class="equipment-item-tooltip">
        <div class="equipment-item-header">
          ${iconHtml}
          <div class="equipment-item-info">
            <a 
              href="${getWowheadUrl(itemId!)}"
              class="equipment-item-link ${qualityClass}"
              target="_blank"
              rel="noopener noreferrer"
            >
              ${itemName}
            </a>
            ${itemData.upgrade_id ? `<div class="item-level">Item Level: Base + ${itemData.upgrade_id}</div>` : ""}
            ${itemData.stats?.ilvl ? `<div class="item-level">Item Level ${itemData.stats.ilvl}</div>` : ""}
          </div>
        </div>
        
        ${
          details.length > 0
            ? `
          <div class="item-tooltip-details">
            ${details.join("")}
          </div>
        `
            : ""
        }
      </div>
    </div>
  `;

  if (isHTML) {
    return itemHtml;
  }

  // create DOM element (client-side only)
  if (typeof document !== "undefined") {
    const div = document.createElement("div");
    div.innerHTML = itemHtml;
    return div.firstElementChild as HTMLElement;
  }

  // server-side fallback
  return itemHtml;
}

// build comprehensive item details (gems, enchants, reforging, tinkers)
function buildItemDetails(itemData: ItemData): string[] {
  const details: string[] = [];

  // handle chart format pre-processed details
  if (itemData.itemDetails && Array.isArray(itemData.itemDetails)) {
    return itemData.itemDetails;
  }

  // handle Blizzard API enchantments format
  if (itemData.enchantments) {
    console.log(
      "Processing enchantments for item:",
      itemData.item_name,
      itemData.enchantments,
    );
    itemData.enchantments.forEach((enchant) => {
      const isGem =
        enchant.slot_id >= 2 && enchant.slot_id <= 5 && !enchant.slot_type;
      const isEnchant = enchant.slot_type === "PERMANENT";
      const isTinker = enchant.slot_type === "ON_USE_SPELL";

      console.log(
        "Enchantment:",
        enchant,
        "isGem:",
        isGem,
        "gem_icon_slug:",
        enchant.gem_icon_slug,
      );

      if (isGem) {
        // use gem icon from database if available
        const gemIcon = enchant.gem_icon_slug
          ? `<img src="${getZamimgIconUrl(enchant.gem_icon_slug, "small")}" alt="${enchant.gem_name || enchant.display_string}" class="gem-icon-inline" />`
          : "";
        console.log("Building gem with icon:", gemIcon);
        details.push(
          `<div class="gem-line">${gemIcon}<span class="gem-stats-white">${enchant.display_string}</span></div>`,
        );
      } else if (isEnchant || isTinker) {
        details.push(
          `<div class="${isEnchant ? "item-enchant" : "equipment-detail"}">${enchant.display_string}</div>`,
        );
      } else {
        details.push(
          `<div class="equipment-detail">${enchant.display_string}</div>`,
        );
      }
    });
  }

  // handle simulation format gems
  if (itemData.gems) {
    itemData.gems.forEach((gem) => {
      if (gem && typeof gem === "object" && gem.name) {
        const gemIcon = gem.icon
          ? `<img src="https://wow.zamimg.com/images/wow/icons/small/${gem.icon}.jpg" alt="${gem.name}" class="gem-icon-inline" />`
          : "";
        const gemStats = gem.stats?.length ? gem.stats.join(", ") : gem.name;
        details.push(
          `<div class="gem-line">${gemIcon}<span class="gem-stats-white">${gemStats}</span></div>`,
        );
      } else if (gem && typeof gem === "number") {
        details.push(
          `<div class="gem-line"><span class="gem-stats-white">Gem ${gem}</span></div>`,
        );
      }
    });
  }

  // handle simulation format enchant
  if (itemData.enchant) {
    const enchantText =
      typeof itemData.enchant === "object" && itemData.enchant.name
        ? itemData.enchant.name
        : `Enchant ${itemData.enchant}`;
    details.push(`<div class="item-enchant">${enchantText}</div>`);
  }

  // handle simulation format reforging
  if (itemData.reforging) {
    let reforgeText = "";
    if (
      typeof itemData.reforging === "object" &&
      itemData.reforging.description
    ) {
      reforgeText = `Reforged ${itemData.reforging.description}`;
    } else if (typeof itemData.reforging === "number") {
      // use ItemDatabase if available for reforge lookup
      reforgeText =
        (window as any).ItemDatabase?.formatReforge?.(itemData.reforging) ||
        `Reforge ${itemData.reforging}`;
    }
    if (reforgeText) {
      details.push(`<div class="item-reforge">${reforgeText}</div>`);
    }
  }

  // handle simulation format tinker
  if (itemData.tinker) {
    details.push(
      `<div class="equipment-detail">Tinker ${itemData.tinker}</div>`,
    );
  }

  return details;
}

// build icon HTML with fallback support
function buildIconHtml(
  itemData: ItemData,
  size: "small" | "medium" | "large" = "medium",
): string {
  const itemId = itemData.item_id || itemData.id;

  // use provided icon slug if available
  if (itemData.item_icon_slug || itemData.icon) {
    const iconSlug = itemData.item_icon_slug || itemData.icon!;
    return `<img class="equipment-icon" src="${getZamimgIconUrl(iconSlug, size)}" alt="${itemData.item_name || itemData.name || ""}" loading="lazy" />`;
  }

  // fallback: try item ID (won't work but placeholder for future)
  if (itemId) {
    return `<img class="equipment-icon" src="${getZamimgIconUrl(itemId.toString(), size)}" alt="${itemData.item_name || itemData.name || ""}" loading="lazy" onerror="this.style.display='none'" />`;
  }

  return "";
}

// format equipment from simulation data (extracted from chart-logic.js)
export function formatEquipmentSummary(items: any[]): any[] {
  const equipment: any[] = [];
  items.forEach((item, index) => {
    if (item) {
      const id = item.id || item.itemId || item.item_id;
      if (!id) return;
      const itemInfo = (window as any).ItemDatabase?.formatEquipmentItem?.(
        { ...item, id },
      ) || {
        itemId: id,
        itemName: item.name || item.item_name || `Item ${id}`,
        itemDetails: [],
        wowheadUrl: getWowheadUrl(id),
        iconUrl: item.icon ? getZamimgIconUrl(item.icon, 'large') : undefined,
        quality: item.quality || null,
      };

      if (itemInfo) {
        equipment.push({
          slot: SIMULATION_SLOT_NAMES[index] || `Slot ${index + 1}`,
          item_id: itemInfo.itemId,
          item_name: itemInfo.itemName,
          quality: itemInfo.quality,
          ...itemInfo,
        });
      }
    }
  });
  return equipment;
}

export function processEquipmentEnchantments(equipment: any[]): any[] {
  return equipment.map((item) => {
    if (!item.enchantments) return item;

    return {
      ...item,
      processedEnchantments: item.enchantments.map((enchant: any) => {
        const isGem =
          enchant.slot_id >= 2 && enchant.slot_id <= 5 && !enchant.slot_type;
        const isEnchant = enchant.slot_type === "PERMANENT";
        const isTinker = enchant.slot_type === "ON_USE_SPELL";

        return {
          ...enchant,
          isGem,
          isEnchant,
          isTinker,
        };
      }),
    };
  });
}

export function getOrderedEquipment(equipment: Record<string, any>): any[] {
  return EQUIPMENT_SLOT_ORDER.map((slot) => equipment[slot]).filter(
    (item) => item != null,
  );
}

export function formatSlotName(slotType: string): string {
  return slotType.replace("_", " ");
}

export function getWowheadUrl(
  itemId: number,
  expansion: string = "mop-classic",
): string {
  return `https://www.wowhead.com/${expansion}/item=${itemId}`;
}

export function getZamimgIconUrl(
  itemId: string | number,
  size: "small" | "medium" | "large" = "medium",
): string {
  return `https://wow.zamimg.com/images/wow/icons/${size}/${itemId}.jpg`;
}
