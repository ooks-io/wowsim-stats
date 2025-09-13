function wzIconSmall(slug?: string | null) {
  return slug
    ? `https://wow.zamimg.com/images/wow/icons/small/${slug}.jpg`
    : "";
}

function wzIconLarge(slug?: string | null) {
  return slug
    ? `https://wow.zamimg.com/images/wow/icons/large/${slug}.jpg`
    : "";
}

export function formatTalents(talents: any) {
  if (!talents || !Array.isArray(talents.talents)) return [] as any[];
  const list = talents.talents
    .map((t: any) => {
      const icon = t.icon
        ? `<img src="${wzIconSmall(t.icon)}" alt="${t.name}" class="talent-icon-inline" loading="lazy" />`
        : "";
      const link = t.spellId
        ? `<a href="https://www.wowhead.com/mop-classic/spell=${t.spellId}" target="_blank" class="talent-link">${t.name}</a>`
        : `<span class="talent-name">${t.name}</span>`;
      return `<div class="talent-line">${icon}${link}</div>`;
    })
    .join("");
  return [
    {
      label: "Talents",
      value: `<div class="talents-list">${list}</div>`,
      isTalentList: true,
    },
  ];
}

export function formatGlyphs(glyphs: any) {
  const mk = (slots: string[], label: string) => {
    const items: string[] = [];
    slots.forEach((slot) => {
      if (!glyphs[slot]) return;
      const name = glyphs[`${slot}Name`] || `Glyph ${glyphs[slot]}`;
      const icon = glyphs[`${slot}Icon`];
      const spellId = glyphs[`${slot}SpellId`];
      const url = spellId
        ? `https://www.wowhead.com/mop-classic/spell=${spellId}`
        : `https://www.wowhead.com/mop-classic/item=${glyphs[slot]}`;
      const iconHtml = icon
        ? `<img src="${wzIconSmall(icon)}" alt="${name}" class="glyph-icon-inline" loading="lazy" />`
        : "";
      const nameHtml = url
        ? `<a href="${url}" target="_blank" class="glyph-link">${name}</a>`
        : `<span class="glyph-name">${name}</span>`;
      items.push(`<div class="glyph-line">${iconHtml}${nameHtml}</div>`);
    });
    return items.length
      ? {
          label,
          value: `<div class=\"glyphs-list\">${items.join("")}</div>`,
          isGlyphList: true,
        }
      : null;
  };
  return [
    mk(["major1", "major2", "major3"], "Major Glyphs"),
    mk(["minor1", "minor2", "minor3"], "Minor Glyphs"),
  ].filter(Boolean) as any[];
}

export function formatConsumables(c: any) {
  const mk = (
    label: string,
    idKey: string,
    nameKey: string,
    iconKey: string,
    qualityKey: string,
  ) => {
    if (!c[idKey]) return null;
    const url = `https://www.wowhead.com/mop-classic/item=${c[idKey]}`;
    const iconUrl = c[iconKey] ? wzIconLarge(c[iconKey]) : null;
    return {
      label,
      value: c[nameKey] || `Item ${c[idKey]}`,
      wowheadUrl: url,
      iconUrl,
      quality: c[qualityKey],
      isItem: true,
    };
  };
  const items: any[] = [];
  const f1 = mk("Flask", "flaskId", "flaskName", "flaskIcon", "flaskQuality");
  if (f1) items.push(f1);
  const f2 = mk("Food", "foodId", "foodName", "foodIcon", "foodQuality");
  if (f2) items.push(f2);
  const f3 = mk("Potion", "potId", "potName", "potIcon", "potQuality");
  if (f3) items.push(f3);
  const f4 = mk(
    "Pre-Potion",
    "prepotId",
    "prepotName",
    "prepotIcon",
    "prepotQuality",
  );
  if (f4) items.push(f4);
  return items;
}

export function formatLoadout(loadout: any) {
  if (!loadout) return [] as any[];
  const sections: any[] = [];
  const fmtRaceGlobal = (window as any).WoWConstants?.formatRace as
    | ((r: string) => string)
    | undefined;
  const fmtRaceLocal = (race: string) => {
    if (!race) return "Unknown";
    let s = String(race).trim();
    if (/^Race[A-Z]/.test(s)) s = s.replace(/^Race/, "");
    s = s.replace(/_/g, " ");
    s = s.replace(/([a-z])([A-Z])/g, "$1 $2");
    s = s.replace(/\s+/g, " ").trim();
    s = s.toLowerCase().replace(/\b\w/g, (l) => l.toUpperCase());
    return s;
  };
  if (loadout.race || loadout.profession1 || loadout.profession2) {
    const items: any[] = [];
    const fmtRace = fmtRaceGlobal || fmtRaceLocal;
    if (loadout.race)
      items.push({ label: "Race", value: fmtRace(loadout.race) });
    if (loadout.profession1)
      items.push({ label: "Profession 1", value: loadout.profession1 });
    if (loadout.profession2)
      items.push({ label: "Profession 2", value: loadout.profession2 });
    sections.push({ title: "Character", items });
  }
  if (loadout.talents || loadout.glyphs) {
    const items: any[] = [];
    if (loadout.talents) items.push(...formatTalents(loadout.talents));
    if (loadout.glyphs) items.push(...formatGlyphs(loadout.glyphs));
    sections.push({ title: "Talents & Glyphs", items });
  }
  if (loadout.consumables) {
    const items = formatConsumables(loadout.consumables);
    if (items.length > 0) sections.push({ title: "Consumables", items });
  }
  if (
    loadout.equipment &&
    (loadout.equipment.items || Array.isArray(loadout.equipment))
  ) {
    const itemsArr = loadout.equipment.items || loadout.equipment;
    if (Array.isArray(itemsArr) && itemsArr.length > 0) {
      sections.push({ title: "Equipment", items: itemsArr, isEquipment: true });
    }
  }
  return sections;
}

export function generateLoadoutDropdown(loadout: any): string {
  const sections = formatLoadout(loadout);
  if (!sections || sections.length === 0) return "";
  const toQualityClass = (q: any) => {
    const map: Record<string, string> = {
      "0": "quality-poor",
      "1": "quality-common",
      "2": "quality-uncommon",
      "3": "quality-rare",
      "4": "quality-epic",
      "5": "quality-legendary",
      "6": "quality-artifact",
      "7": "quality-heirloom",
    };
    if (typeof q === "number") return map[String(q)] || "quality-common";
    if (typeof q === "string") {
      const s = q.toLowerCase();
      if (s.startsWith("quality-")) return s;
      // Accept both enum (EPIC) and numeric-like strings
      return map[s] || `quality-${s}`;
    }
    return "quality-common";
  };

  const renderEquipmentCompact = (items: any[]) => {
    if (!Array.isArray(items) || items.length === 0) return "";
    const cells = items
      .map((it: any) => {
        const id = it.id || it.item_id || it.itemId;
        if (!id) return "";
        const href = `https://www.wowhead.com/mop-classic/item=${id}`;
        const title = it.name || it.item_name || `Item ${id}`;
        const qClass = toQualityClass(it.quality);
        const iconSlug = it.icon || it.item_icon_slug || null;
        const img = iconSlug
          ? iconSlug.startsWith?.("http")
            ? iconSlug
            : `https://wow.zamimg.com/images/wow/icons/large/${iconSlug}.jpg`
          : it.iconUrl || "";
        // Build gem/enchant badges from either enriched enchantments or raw gems
        let badgeHtml = "";
        if (Array.isArray(it.enchantments)) {
          const egems = it.enchantments.filter(
            (e: any) => e && e.gem_icon_slug,
          );
          if (egems.length) {
            badgeHtml = `<div class="gem-stack">${egems
              .slice(0, 3)
              .map(
                (g: any) =>
                  `<img class="gem-badge" src="https://wow.zamimg.com/images/wow/icons/small/${g.gem_icon_slug}.jpg" alt="${g.gem_name || "Gem"}" title="${g.gem_name || g.display_string || "Gem"}" />`,
              )
              .join("")}</div>`;
          }
        } else if (Array.isArray(it.gems)) {
          const ggems = it.gems.filter(
            (g: any) => g && typeof g === "object" && g.icon,
          );
          if (ggems.length) {
            badgeHtml = `<div class="gem-stack">${ggems
              .slice(0, 3)
              .map(
                (g: any) =>
                  `<img class="gem-badge" src="https://wow.zamimg.com/images/wow/icons/small/${g.icon}.jpg" alt="${g.name || "Gem"}" title="${(g.stats && g.stats.join(", ")) || g.name || "Gem"}" />`,
              )
              .join("")}</div>`;
          }
        }
        return `
          <a href="${href}" class="equip-icon-wrap ${qClass}" title="${title}" target="_blank" rel="noopener noreferrer">
            <div class="equip-icon-inner">
              ${img ? `<img class="equip-icon" src="${img}" alt="${title}" loading="lazy" />` : `<div class="equip-icon placeholder" aria-label="${title}"></div>`}
              ${badgeHtml}
            </div>
          </a>`;
      })
      .join("");
    return `<div class="equipment-compact"><div class="equipment-icon-grid">${cells}</div></div>`;
  };

  const sectionsHtml = sections
    .map((section: any) => {
      if (section.isEquipment) {
        const compact = renderEquipmentCompact(
          loadout?.equipment?.items || section.items || [],
        );
        return `
        <div class="loadout-section">
          <h4 class="loadout-title">${section.title}</h4>
          ${compact}
        </div>`;
      } else {
        const itemsHtml = section.items
          .map((item: any) => {
            const labelHtml = item.label
              ? `<span class="loadout-label">${item.label}</span>`
              : "";
            // Consumables and linked items
            if (
              (item.isItem || item.isGlyph || item.isTalent) &&
              item.wowheadUrl
            ) {
              const iconHtml = item.iconUrl
                ? `<img src="${item.iconUrl}" alt="${item.value}" class="consumable-icon" loading="lazy" />`
                : "";
              const qualityClass = item.quality
                ? `quality-${item.quality}`
                : "";
              const valueHtml = `<div class="consumable-item-header">${iconHtml}<a href="${item.wowheadUrl}" target="_blank" class="equipment-item-link ${qualityClass}">${item.value}</a></div>`;
              return `<div class="loadout-item">${labelHtml}${valueHtml}</div>`;
            }
            // Talents & Glyphs lists come as prebuilt HTML
            if (item.isTalentList || item.isGlyphList) {
              return `<div class="loadout-item">${labelHtml}${item.value}</div>`;
            }
            // Plain values
            return `<div class="loadout-item">${labelHtml}<span class="loadout-value">${item.value}</span></div>`;
          })
          .join("");
        return `
        <div class="loadout-section">
          <h4 class="loadout-title">${section.title}</h4>
          <div class="loadout-grid">${itemsHtml}</div>
        </div>`;
      }
    })
    .join("");

  const wowSimButton = loadout?.simLink
    ? `<a href="${loadout.simLink}" target="_blank" class="loadout-button wowsim-button">Open in WoWSims</a>`
    : "";

  // Return just the content; caller wraps with chart-dropdown.
  return `${wowSimButton}
      ${sectionsHtml}`;
}
