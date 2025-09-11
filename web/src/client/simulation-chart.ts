type Mode = 'rankings' | 'comparison' | 'unified' | string;

export type InitOptions = {
  mode: Mode;
  fixedClass?: string;
  fixedSpec?: string;
  comparisonType?: 'trinket' | 'race' | string;
  isUnifiedMode?: boolean;
};

type RankingsJSON = {
  metadata: any;
  results: Record<string, Record<string, {
    dps: number; max: number; min: number; stdev: number; loadout?: any;
  }>>;
};

type ComparisonItem = {
  dps: number; max: number; min: number; stdev: number; loadout?: any;
  trinket?: { id: number; name: string };
};

type ComparisonJSON = {
  metadata: any;
  results: Record<string, ComparisonItem>;
};

class SimulationChart {
  private mode: Mode;
  private comparisonType: 'trinket' | 'race' | string;
  private fixedClass?: string;
  private fixedSpec?: string;

  constructor(opts: InitOptions) {
    this.mode = opts.mode;
    this.fixedClass = opts.fixedClass;
    this.fixedSpec = opts.fixedSpec;
    this.comparisonType = (opts.comparisonType || 'trinket') as any;
  }

  init() {
    this.bindControls();
    this.update();
  }

  private getEl<T extends HTMLElement>(id: string): T {
    const el = document.getElementById(id) as T | null;
    if (!el) throw new Error(`Missing element #${id}`);
    return el;
  }

  private bindControls() {
    const ids = [
      'simulationMode', 'comparisonType', 'class', 'spec',
      'targetCount', 'encounterType', 'duration', 'phase', 'sortBy',
    ];
    ids.forEach((id) => {
      const el = document.getElementById(id) as HTMLSelectElement | null;
      if (el) el.addEventListener('change', () => this.update());
    });

    // Populate spec options when class changes
    const classSel = document.getElementById('class') as HTMLSelectElement | null;
    const specSel = document.getElementById('spec') as HTMLSelectElement | null;
    if (classSel && specSel) {
      classSel.addEventListener('change', () => {
        this.populateSpecs();
        this.update();
      });
    }

    const modeSel = document.getElementById('simulationMode') as HTMLSelectElement | null;
    if (modeSel) {
      modeSel.addEventListener('change', () => {
        this.updateUIForMode();
        this.update();
      });
    }

    const compSel = document.getElementById('comparisonType') as HTMLSelectElement | null;
    if (compSel) {
      compSel.addEventListener('change', () => {
        this.comparisonType = (compSel.value as any) || 'trinket';
        this.updateUIForMode();
        this.update();
      });
    }

    // Initial UI state
    this.populateSpecs();
    this.updateUIForMode();
  }

  private populateSpecs() {
    const cls = this.getSelectedClass();
    const specSel = document.getElementById('spec') as HTMLSelectElement | null;
    if (!specSel) return;
    while (specSel.firstChild) specSel.removeChild(specSel.firstChild);
    if (!cls) {
      specSel.disabled = true;
      const opt = document.createElement('option');
      opt.value = '';
      opt.textContent = 'Select Class First';
      specSel.appendChild(opt);
      return;
    }
    const OPTIONS: Record<string, { value: string; label: string }[]> = (window as any).WoWConstants?.SPEC_OPTIONS || {};
    const specs = OPTIONS[cls] || [];
    specSel.disabled = false;
    specs.forEach((spObj) => {
      const opt = document.createElement('option');
      const slug = spObj.label.toLowerCase().replace(/\s+/g, '_');
      opt.value = slug;
      opt.textContent = spObj.label;
      specSel.appendChild(opt);
    });
    if (this.fixedSpec) specSel.value = this.fixedSpec;
  }

  private getSelected<T = string>(id: string): T | null {
    const el = document.getElementById(id) as HTMLSelectElement | null;
    return (el && el.value) ? (el.value as any as T) : null;
  }

  private getSelectedClass(): string | null {
    return this.fixedClass || this.getSelected('class');
  }
  private getSelectedSpec(): string | null {
    return this.fixedSpec || this.getSelected('spec');
  }

  private async update() {
    const loading = this.getEl<HTMLDivElement>('loading');
    const errorEl = this.getEl<HTMLDivElement>('error');
    const meta = this.getEl<HTMLDivElement>('metadata-container');
    const chart = this.getEl<HTMLDivElement>('chart-container');
    errorEl.classList.add('hidden');
    loading.classList.remove('hidden');
    meta.style.opacity = '0.4';
    chart.style.opacity = '0.4';

    try {
      const data = await this.loadData();
      loading.classList.add('hidden');
      this.renderMetadata(data.metadata);
      this.renderChart(data);
      meta.style.opacity = '1';
      chart.style.opacity = '1';
    } catch (e: any) {
      loading.classList.add('hidden');
      meta.innerHTML = '';
      chart.innerHTML = '';
      errorEl.textContent = `Error loading data: ${e?.message || e}`;
      errorEl.classList.remove('hidden');
      meta.style.opacity = '1';
      chart.style.opacity = '1';
    }
  }

  private updateUIForMode() {
    const modeSel = document.getElementById('simulationMode') as HTMLSelectElement | null;
    const effectiveMode: Mode = this.mode === 'unified' ? (modeSel && modeSel.value === 'comparisons' ? 'comparison' : 'rankings') : this.mode;
    const comparisonRow = document.getElementById('comparison-controls-row');
    const classSel = document.getElementById('class') as HTMLSelectElement | null;
    const specSel = document.getElementById('spec') as HTMLSelectElement | null;
    const callout = document.getElementById('trinket-callout');
    const compTypeSel = document.getElementById('comparisonType') as HTMLSelectElement | null;

    if (comparisonRow) {
      comparisonRow.style.display = effectiveMode === 'comparison' ? 'flex' : 'none';
    }
    if (classSel) classSel.disabled = effectiveMode !== 'comparison';
    if (specSel) specSel.disabled = effectiveMode !== 'comparison' || !this.getSelectedClass();
    if (callout) {
      const isTrinket = (compTypeSel?.value || this.comparisonType) === 'trinket';
      callout.classList.toggle('hidden', !(effectiveMode === 'comparison' && isTrinket));
    }
  }

  private buildRankingsUrl() {
    const phase = this.getSelected<string>('phase') || 'p1';
    const encounter = this.getSelected<string>('encounterType') || 'raid';
    const target = this.getSelected<string>('targetCount') || 'single';
    const duration = this.getSelected<string>('duration') || 'long';
    const file = `dps_${phase}_${encounter}_${target}_${duration}.json`;
    return `/data/rankings/${file}`;
  }

  private buildComparisonUrl() {
    const cls = (this.getSelectedClass() || '').toLowerCase();
    const spec = (this.getSelectedSpec() || '').toLowerCase();
    if (!cls || !spec) throw new Error('Select a class and spec');
    const type = (this.getSelected<string>('comparisonType') || this.comparisonType || 'trinket') as string;
    const phase = this.getSelected<string>('phase') || 'p1';
    const encounter = this.getSelected<string>('encounterType') || 'raid';
    const target = this.getSelected<string>('targetCount') || 'single';
    const duration = this.getSelected<string>('duration') || 'long';
    const file = `${cls}_${spec}_${type}_${phase}_${encounter}_${target}_${duration}.json`;
    return `/data/comparison/${type}/${cls}/${spec}/${file}`;
  }

  private async loadData(): Promise<any> {
    const modeSel = this.getSelected<string>('simulationMode');
    const effectiveMode: Mode = this.mode === 'unified' ? (modeSel === 'comparisons' ? 'comparison' : 'rankings') : this.mode;
    const url = effectiveMode === 'rankings' ? this.buildRankingsUrl() : this.buildComparisonUrl();
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`${resp.status} ${resp.statusText}`);
    return resp.json();
  }

  private renderMetadata(metadata: any) {
    const container = this.getEl<HTMLDivElement>('metadata-container');
    const fmtDate = (window as any).WoWConstants?.formatSimulationDate || ((d: string) => d);
    const fmtDuration = (window as any).WoWConstants?.formatDuration || ((s: number) => `${Math.floor(s/60)}m ${s%60}s`);
    const iter = metadata.iterations?.toLocaleString?.() || metadata.iterations;
    const date = metadata.timestamp ? fmtDate(metadata.timestamp) : '';
    const target = metadata.targetCount ?? '';
    const duration = metadata.encounterDuration ? fmtDuration(metadata.encounterDuration) : '';
    const type = metadata.spec ? 'Comparison' : 'DPS Rankings';
    container.innerHTML = `
      <div class="card">
        <h3 class="card-title">${type} Details</h3>
        <div class="info-grid">
          <div class="info-item"><span class="info-label">Iterations</span><span class="info-value">${iter}</span></div>
          <div class="info-item"><span class="info-label">Encounter Duration</span><span class="info-value">${duration}</span></div>
          <div class="info-item"><span class="info-label">Target Count</span><span class="info-value">${target}</span></div>
          ${date ? `<div class="info-item"><span class="info-label">Date Simulated</span><span class="info-value">${date}</span></div>` : ''}
        </div>
      </div>
    `;
  }

  private calculateBarWidth(value: number, max: number, min: number) {
    if (max <= 0) return 0;
    const range = Math.max(max - min, 1);
    const pct = (value - min) / range;
    const minPct = 0.15; // ensure visibility
    return Math.max(pct, minPct) * 100;
  }

  private renderChart(data: RankingsJSON | ComparisonJSON) {
    const chart = this.getEl<HTMLDivElement>('chart-container');
    const modeSel = this.getSelected<string>('simulationMode');
    const effectiveMode: Mode = this.mode === 'unified' ? (modeSel === 'comparisons' ? 'comparison' : 'rankings') : this.mode;
    const sortBy = this.getSelected<string>('sortBy') || (effectiveMode === 'rankings' ? 'dps' : 'percent');
    type Item = { label: string; value: number; min: number; max: number; stdev: number; percent?: number; color?: string; loadout?: any };
    let items: Item[] = [];

    if (effectiveMode === 'rankings') {
      const r = data as RankingsJSON;
      const CLASS_COLORS: Record<string, string> = (window as any).WoWConstants?.CLASS_COLORS || {};
      Object.entries(r.results).forEach(([cls, specs]) => {
        const color = CLASS_COLORS[cls] || '#FFFFFF';
        Object.entries(specs).forEach(([spec, v]) => {
          const specLabel = spec.replace(/_/g, ' ').replace(/\b\w/g, s => s.toUpperCase());
          items.push({ label: specLabel, value: v.dps, min: v.min, max: v.max, stdev: v.stdev, color, loadout: (v as any).loadout });
        });
      });
    } else {
      const c = data as ComparisonJSON;
      const baseline = c.results?.baseline?.dps || 0;
      Object.entries(c.results).forEach(([key, v]) => {
        if (key === 'baseline') return;
        const ilvlSuffix = (/_([0-9]{3})$/.exec(key)?.[1]) || '';
        const name = v.trinket?.name || key.replace(/_/g, ' ');
        const label = ilvlSuffix ? `${name} (${ilvlSuffix})` : name;
        const percent = baseline > 0 ? ((v.dps - baseline) / baseline) * 100 : undefined;
        items.push({ label, value: v.dps, min: v.min, max: v.max, stdev: v.stdev, percent, loadout: v.loadout });
      });
    }

    if (sortBy === 'stdev') items.sort((a, b) => a.stdev - b.stdev);
    else if (sortBy === 'dps') items.sort((a, b) => b.value - a.value);
    else if (sortBy === 'max') items.sort((a, b) => b.max - a.max);
    else if (sortBy === 'min') items.sort((a, b) => b.min - a.min);
    else if (sortBy === 'percent') items.sort((a, b) => (b.percent || 0) - (a.percent || 0));
    else items.sort((a, b) => b.value - a.value);

    const max = items.reduce((m, it) => Math.max(m, it.value), 0);
    const min = items.reduce((m, it) => Math.min(m, it.value), max);

    const bars = items.map((it, idx) => {
      const width = this.calculateBarWidth(it.value, max, min).toFixed(1);
      const percentStr = (it.percent != null) ? ` (+${it.percent.toFixed(2)}%)` : '';
      const colorStyle = it.color ? `style=\"color: ${it.color};\"` : '';
      const dropdownHtml = this.generateLoadoutDropdown(it.loadout, it);
      return `
        <div class="chart-item-wrapper" data-index="${idx}">
          <div class="chart-item-header">
            <div class="chart-item">
              <div class="chart-labels">
                <span class="chart-rank">#${idx + 1}</span>
                <span class="chart-label" ${colorStyle}>${it.label}</span>
              </div>
              <div class="chart-bar-container">
                <div class="chart-bar-track">
                  <div class="chart-bar" style="width: ${width}%; background: #6aa84f;"></div>
                </div>
                <div class="chart-value">${Math.round(it.value).toLocaleString()}${percentStr}</div>
              </div>
            </div>
            <span class="chart-expand-icon">›</span>
          </div>
          <div class="chart-dropdown">${dropdownHtml}</div>
        </div>
      `;
    }).join('');

    chart.innerHTML = `<div class="chart-bars">${bars}</div>`;

    // bind expand/collapse
    chart.querySelectorAll('.chart-item-wrapper').forEach((wrap) => {
      const header = (wrap as HTMLElement).querySelector('.chart-item-header');
      header?.addEventListener('click', () => {
        (wrap as HTMLElement).classList.toggle('chart-item-expanded');
      });
    });
  }

  // ===== Loadout rendering (ported) =====
  private formatLoadout(loadout: any) {
    if (!loadout) return [] as any[];
    const sections: any[] = [];
    if (loadout.race || loadout.profession1 || loadout.profession2) {
      const items: any[] = [];
      if (loadout.race) items.push({ label: 'Race', value: (window as any).WoWConstants?.formatRace?.(loadout.race) || loadout.race });
      if (loadout.profession1) items.push({ label: 'Profession 1', value: loadout.profession1 });
      if (loadout.profession2) items.push({ label: 'Profession 2', value: loadout.profession2 });
      sections.push({ title: 'Character', items });
    }
    if (loadout.talents || loadout.glyphs) {
      const items: any[] = [];
      if (loadout.talents) items.push(...this.formatTalents(loadout.talents));
      if (loadout.glyphs) items.push(...this.formatGlyphs(loadout.glyphs));
      sections.push({ title: 'Talents & Glyphs', items });
    }
    if (loadout.consumables) {
      const items = this.formatConsumables(loadout.consumables);
      if (items.length > 0) sections.push({ title: 'Consumables', items });
    }
    if (loadout.equipment && loadout.equipment.items) {
      const summary = (window as any).EquipmentUtils?.formatEquipmentSummary?.(loadout.equipment.items) || [];
      if (summary.length > 0) sections.push({ title: 'Equipment', items: summary, isEquipment: true });
    }
    return sections;
  }

  private formatTalents(talents: any) {
    if (!talents || !Array.isArray(talents.talents)) return [];
    const list = talents.talents.map((t: any) => {
      const icon = t.icon ? `<img src="https://wow.zamimg.com/images/wow/icons/small/${t.icon}.jpg" alt="${t.name}" class="talent-icon-inline" loading="lazy" />` : '';
      const url = t.spellId ? `https://www.wowhead.com/mop-classic/spell=${t.spellId}` : null;
      const name = url ? `<a href="${url}" target="_blank" class="talent-link">${t.name}</a>` : `<span class="talent-name">${t.name}</span>`;
      return `<div class="talent-line">${icon}${name}</div>`;
    }).join('');
    return [{ label: 'Talents', value: `<div class="talents-list">${list}</div>`, isTalentList: true }];
  }

  private formatGlyphs(g: any) {
    const mk = (slot: string, label: string) => {
      if (!g[slot]) return '';
      const name = g[`${slot}Name`] || `Glyph ${g[slot]}`;
      const icon = g[`${slot}Icon`] ? `<img src="https://wow.zamimg.com/images/wow/icons/small/${g[`${slot}Icon`]}.jpg" alt="${name}" class="glyph-icon-inline" loading="lazy" />` : '';
      const spellId = g[`${slot}SpellId`];
      const url = spellId ? `https://www.wowhead.com/mop-classic/spell=${spellId}` : `https://www.wowhead.com/mop-classic/item=${g[slot]}`;
      return `<div class="glyph-line">${icon}<a href="${url}" target="_blank" class="glyph-link">${name}</a></div>`;
    };
    const major = ['major1','major2','major3'].map(s => mk(s, 'Major')).filter(Boolean).join('');
    const minor = ['minor1','minor2','minor3'].map(s => mk(s, 'Minor')).filter(Boolean).join('');
    const out: any[] = [];
    if (major) out.push({ label: 'Major Glyphs', value: `<div class="glyphs-list">${major}</div>`, isGlyphList: true });
    if (minor) out.push({ label: 'Minor Glyphs', value: `<div class="glyphs-list">${minor}</div>`, isGlyphList: true });
    return out;
  }

  private formatConsumables(c: any) {
    const mk = (label: string, idKey: string, nameKey: string, iconKey: string, qualityKey: string) => {
      if (!c[idKey]) return null;
      const url = `https://www.wowhead.com/mop-classic/item=${c[idKey]}`;
      const icon = c[iconKey] ? `<img src="https://wow.zamimg.com/images/wow/icons/large/${c[iconKey]}.jpg" alt="${c[nameKey]}" class="consumable-icon" loading="lazy" />` : '';
      const qualityClass = c[qualityKey] ? `quality-${c[qualityKey]}` : '';
      return { label, value: c[nameKey] || `Item ${c[idKey]}`, wowheadUrl: url, iconUrl: c[iconKey] ? `https://wow.zamimg.com/images/wow/icons/large/${c[iconKey]}.jpg` : null, quality: c[qualityKey], isItem: true };
    };
    const items: any[] = [];
    const f1 = mk('Flask','flaskId','flaskName','flaskIcon','flaskQuality'); if (f1) items.push(f1);
    const f2 = mk('Food','foodId','foodName','foodIcon','foodQuality'); if (f2) items.push(f2);
    const f3 = mk('Potion','potId','potName','potIcon','potQuality'); if (f3) items.push(f3);
    const f4 = mk('Pre-Potion','prepotId','prepotName','prepotIcon','prepotQuality'); if (f4) items.push(f4);
    return items;
  }

  private generateLoadoutDropdown(loadout: any, _chartData: any): string {
    const sections = this.formatLoadout(loadout);
    if (!sections || sections.length === 0) return '';
    const sectionsHtml = sections.map((section: any) => {
      if (section.isEquipment) {
        const equipmentHtml = section.items.map((eq: any) => {
          const itemData = {
            slot: eq.slot,
            item_id: eq.itemId,
            item_name: eq.itemName,
            quality: eq.quality,
            item_icon_slug: eq.iconUrl ? (eq.iconUrl.match(/\/([^\/]+)\.jpg$/)?.[1] || null) : null,
            itemDetails: eq.itemDetails,
          } as any;
        if ((window as any).EquipmentUtils?.createItemElement) {
            return (window as any).EquipmentUtils.createItemElement(itemData, { isHTML: true, showIcon: true });
          }
          const iconHtml = eq.iconUrl ? `<img src="${eq.iconUrl}" alt="${eq.itemName}" class="equipment-icon" loading="lazy" />` : '';
          const qualityClass = eq.quality ? `quality-${eq.quality}` : '';
          const detailsHtml = eq.itemDetails?.length > 0 ? `<div class="item-tooltip-details">${eq.itemDetails.map((d: string) => `<div class="equipment-detail">${d}</div>`).join('')}</div>` : '';
          return `
            <div class="equipment-slot">
              <div class="equipment-slot-header"><span class="equipment-slot-name">${eq.slot}</span></div>
              <div class="equipment-item-tooltip">
                <div class="equipment-item-header">
                  ${iconHtml}
                  <a href="${eq.wowheadUrl}" target="_blank" class="equipment-item-link ${qualityClass}">${eq.itemName}</a>
                </div>
                <div class="equipment-item-details">${detailsHtml}</div>
              </div>
            </div>`;
        }).join('');
        return `
          <div class="loadout-section">
            <h4 class="loadout-title">${section.title}</h4>
            <div class="equipment-grid">${equipmentHtml}</div>
          </div>`;
      } else {
        const itemsHtml = section.items.map((item: any) => {
          if ((item.isItem || item.isGlyph || item.isTalent) && item.wowheadUrl) {
            const iconHtml = item.iconUrl ? `<img src="${item.iconUrl}" alt="${item.value}" class="consumable-icon" loading="lazy" />` : '';
            const qualityClass = item.quality ? `quality-${item.quality}` : '';
            return `<div class="consumable-item-header">${iconHtml}<a href="${item.wowheadUrl}" target="_blank" class="equipment-item-link ${qualityClass}">${item.value}</a></div>`;
          }
          if (item.isTalentList || item.isGlyphList) {
            return item.value;
          }
          return `<span class="loadout-value">${item.value}</span>`;
        }).join('');
        return `
          <div class="loadout-section">
            <h4 class="loadout-title">${section.title}</h4>
            <div class="loadout-grid"><div class="loadout-item">${itemsHtml}</div></div>
          </div>`;
      }
    }).join('');
    return sectionsHtml;
  }
}

export function initializeChart(options: InitOptions) {
  const chart = new SimulationChart(options);
  chart.init();
}

export default initializeChart;

// Auto-initialize when included via <script src> by reading dataset from container
function bootFromContainer() {
  const container = document.querySelector('.rankings-container') as HTMLElement | null;
  if (!container) return;
  const options: InitOptions = {
    mode: (container.dataset.mode as Mode) || 'unified',
    fixedClass: container.dataset.fixedClass || undefined,
    fixedSpec: container.dataset.fixedSpec || undefined,
    comparisonType: (container.dataset.comparisonType as any) || 'trinket',
    isUnifiedMode: container.dataset.isUnified === 'true',
  };
  try { initializeChart(options); } catch (e) { console.error('Sim chart init failed:', e); }
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', bootFromContainer);
} else {
  bootFromContainer();
}
