import type { Enchantment, ScalingOption } from "../../lib/types";

const STAT_NAMES: Record<string, string> = {
  "0": "Strength",
  "1": "Agility",
  "2": "Stamina",
  "3": "Intellect",
  "4": "Spirit",
  "5": "Hit",
  "6": "Crit",
  "7": "Haste",
  "8": "Expertise",
  "9": "Dodge",
  "10": "Parry",
  "11": "Mastery",
};

function getScaling(scalingOptions: Record<string, ScalingOption>): ScalingOption | null {
  return scalingOptions["0"] ?? null;
}

function populateTooltip(
  tooltip: HTMLElement,
  scalingOptions: Record<string, ScalingOption>,
  itemName: string,
  quality: string,
  enchants: Enchantment[],
  spellDescription: string | null,
) {
  const nameEl = tooltip.querySelector<HTMLElement>(".tooltip-item-name");
  const ilvlEl = tooltip.querySelector<HTMLElement>(".tooltip-ilvl");
  const slotArmorEl = tooltip.querySelector<HTMLElement>(".tooltip-slot-armor");
  const statsEl = tooltip.querySelector<HTMLElement>(".tooltip-stats");
  const gemsEl = tooltip.querySelector<HTMLElement>(".tooltip-gems");
  const enchantEl = tooltip.querySelector<HTMLElement>(".tooltip-enchant");
  const effectEl = tooltip.querySelector<HTMLElement>(".tooltip-effect");
  const setEl = tooltip.querySelector<HTMLElement>(".tooltip-set");

  if (nameEl) {
    nameEl.textContent = itemName;
    nameEl.className = `tooltip-item-name quality-${quality.toLowerCase()}`;
  }

  const scaling = getScaling(scalingOptions);

  if (ilvlEl) {
    ilvlEl.textContent = scaling?.ilvl ? `Item Level ${scaling.ilvl}` : "";
  }

  if (slotArmorEl) {
    const parts: string[] = [];
    if (scaling?.stats?.["17"]) {
      parts.push(`${scaling.stats["17"].toLocaleString()} Armor`);
    }
    slotArmorEl.textContent = parts.join(" — ");
  }

  // stats (skip armor key 17)
  if (statsEl) {
    statsEl.innerHTML = "";
    if (scaling?.stats) {
      for (const [key, value] of Object.entries(scaling.stats)) {
        if (key === "17") continue;
        const name = STAT_NAMES[key] || `Stat ${key}`;
        const div = document.createElement("div");
        div.className = "tooltip-stat-line";
        div.textContent = `+${value.toLocaleString()} ${name}`;
        statsEl.appendChild(div);
      }
    }
  }

  // gems (slot_id 2-5, no slot_type)
  if (gemsEl) {
    gemsEl.innerHTML = "";
    const gems = enchants.filter(
      (e) => e.slot_id >= 2 && e.slot_id <= 5 && !e.slot_type,
    );
    for (const gem of gems) {
      const div = document.createElement("div");
      div.className = "tooltip-gem-line";
      if (gem.gem_icon_slug) {
        const img = document.createElement("img");
        img.src = `https://wow.zamimg.com/images/wow/icons/small/${gem.gem_icon_slug}.jpg`;
        img.className = "tooltip-gem-icon";
        img.alt = "";
        div.appendChild(img);
      }
      const span = document.createElement("span");
      span.textContent = gem.display_string;
      div.appendChild(span);
      gemsEl.appendChild(div);
    }
  }

  // enchant (slot_type === "PERMANENT")
  if (enchantEl) {
    enchantEl.innerHTML = "";
    const permanent = enchants.filter((e) => e.slot_type === "PERMANENT");
    for (const ench of permanent) {
      const div = document.createElement("div");
      div.className = "tooltip-enchant-line";
      div.textContent = ench.display_string;
      enchantEl.appendChild(div);
    }
  }

  // spell description from Blizzard API
  if (effectEl) {
    effectEl.innerHTML = "";
    if (spellDescription) {
      // handle multiple descriptions (separated by newlines)
      const descriptions = spellDescription.split("\n");
      for (const desc of descriptions) {
        if (desc.trim()) {
          const div = document.createElement("div");
          div.className = "tooltip-effect-line";
          div.textContent = desc;
          effectEl.appendChild(div);
        }
      }
    }
  }

  if (setEl) {
    setEl.textContent = "";
    setEl.style.display = "none";
  }
}

function positionTooltip(tooltip: HTMLElement, anchor: HTMLElement) {
  const rect = anchor.getBoundingClientRect();
  const tooltipRect = tooltip.getBoundingClientRect();
  const padding = 8;

  let left = rect.right + padding;
  let top = rect.top;

  if (left + tooltipRect.width > window.innerWidth) {
    left = rect.left - tooltipRect.width - padding;
  }
  if (top + tooltipRect.height > window.innerHeight) {
    top = window.innerHeight - tooltipRect.height - padding;
  }
  if (top < padding) top = padding;

  tooltip.style.left = `${left}px`;
  tooltip.style.top = `${top}px`;
}

export function initItemTooltip() {
  const tooltip = document.getElementById("item-tooltip");
  if (!tooltip) return;

  const icons = document.querySelectorAll<HTMLElement>(".equip-icon-wrap[data-item-id]");

  for (const icon of icons) {
    icon.addEventListener("mouseenter", () => {
      const itemName = icon.dataset.itemName || "Unknown";
      const quality = icon.dataset.quality || "COMMON";

      let enchants: Enchantment[] = [];
      try {
        enchants = JSON.parse(icon.dataset.enchants || "[]");
      } catch { /* ignore */ }

      let scalingOptions: Record<string, ScalingOption> = {};
      try {
        scalingOptions = JSON.parse(icon.dataset.scaling || "{}");
      } catch { /* ignore */ }

      const spellDescription = icon.dataset.spellDescription || null;

      populateTooltip(tooltip, scalingOptions, itemName, quality, enchants, spellDescription);
      tooltip.style.display = "block";
      positionTooltip(tooltip, icon);
    });

    icon.addEventListener("mouseleave", () => {
      tooltip.style.display = "none";
    });
  }
}
