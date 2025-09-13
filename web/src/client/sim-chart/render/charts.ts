import { calculateBarWidth, sortItems, type SortBy } from '../sorting';
import { generateLoadoutDropdown } from '../loadout';
import { formatRace as utilFormatRace } from '../../../lib/utils';

type Item = { label: string; value: number; min: number; max: number; stdev: number; percent?: number; color?: string; loadout?: any };

export function renderRankingsChartHTML(data: any, sortBy: SortBy, classColors: Record<string,string>): string {
  const items: Item[] = [];
  for (const [className, classData] of Object.entries<any>(data.results || {})) {
    const colorKey = String(className).toLowerCase().replace(/\s+/g, '_');
    const barColor = classColors[colorKey] || '#666';
    for (const [specName, specData] of Object.entries<any>(classData)) {
      items.push({
        label: String(specName).replace(/_/g, ' ').replace(/\b\w/g, s => s.toUpperCase()),
        value: specData.dps,
        min: specData.min,
        max: specData.max,
        stdev: specData.stdev,
        color: barColor,
        loadout: specData.loadout || null,
      });
    }
  }
  sortItems(items, sortBy);
  const metric = (it: Item) => (String(sortBy) === 'max' ? it.max : String(sortBy) === 'min' ? it.min : String(sortBy) === 'stdev' ? it.stdev : it.value);
  const max = Math.max(...items.map(metric), 0);
  const min = Math.min(...items.map(metric), max);

  const bars = items.map((it, idx) => {
    const mval = metric(it);
    const width = calculateBarWidth(mval, max, min).toFixed(1);
    const displayRaw = String(sortBy) === 'stdev' ? it.stdev : mval;
    const display = Math.round(displayRaw).toLocaleString();
    const tooltip = display;
    const colorStyle = it.color ? `style=\"color: ${it.color};\"` : '';
    const dropdownHtml = generateLoadoutDropdown(it.loadout);
    const hasDropdown = dropdownHtml && dropdownHtml.length > 0;
    return `
      <div class="chart-item-wrapper" data-index="${idx}">
        <div class="chart-item-header" title="${tooltip}">
          <div class="chart-item">
            <div class="chart-labels">
              <span class="chart-rank">#${idx + 1}</span>
              <span class="chart-label" ${colorStyle}>${it.label}</span>
            </div>
            <div class="chart-bar-container">
              <div class="chart-bar-track">
                <div class="chart-bar" style="width: ${width}%; background: ${it.color || '#6aa84f'};"></div>
              </div>
              <div class="chart-value">${display}</div>
            </div>
          </div>
        </div>
        ${hasDropdown ? `<div class="chart-dropdown">${dropdownHtml}</div>` : ''}
      </div>`;
  }).join('');
  const titles: Record<string,string> = {
    dps: 'DPS Rankings (Average)',
    max: 'DPS Rankings (Maximum)',
    min: 'DPS Rankings (Minimum)',
    stdev: 'DPS Consistency Rankings (Low StdDev = More Consistent)'
  };
  const title = titles[String(sortBy)] || 'DPS Rankings';
  return `<h2 class="card-title-large">${title}</h2><div class="chart-bars">${bars}</div>`;
}

function trinketIconDataFromLoadout(loadout: any) {
  try {
    const items = loadout?.equipment?.items;
    const t1 = items?.[12];
    const t2 = items?.[13];
    const tr = (t1 && t1.id) ? t1 : ((t2 && t2.id) ? t2 : null);
    if (!tr) return null;
    const ilvl = tr.stats?.ilvl ?? '';
    return { icon: tr.icon, ilvl, name: tr.name };
  } catch { return null; }
}

export function renderComparisonChartHTML(data: any, sortBy: SortBy, comparisonType: 'race'|'trinket' = 'trinket', barColor: string = '#666'): string {
  const baseline = data?.results?.baseline?.dps || 0;
  const items: Item[] = [];
  for (const [key, v] of Object.entries<any>(data.results || {})) {
    if (key === 'baseline') continue;
    let label = key.replace(/_/g, ' ');
    if (comparisonType === 'race') {
      label = utilFormatRace(key);
    }
    let iconData: any = null;
    if (comparisonType === 'trinket') {
      iconData = trinketIconDataFromLoadout(v.loadout);
      if (iconData) label = iconData.ilvl ? `${iconData.ilvl}` : iconData.name;
    }
    const percent = baseline > 0 ? ((v.dps - baseline) / baseline) * 100 : undefined;
    const dpsIncrease = baseline > 0 ? (v.dps - baseline) : undefined;
    (items as any).push({ label, value: v.dps, min: v.min, max: v.max, stdev: v.stdev, percent, dpsIncrease, loadout: v.loadout, iconData });
  }
  // For races there is no baseline; percent sorting/display isn't meaningful
  const effectiveSort: SortBy = (comparisonType === 'race' && String(sortBy) === 'percent') ? 'dps' : sortBy;
  sortItems(items, effectiveSort);
  const metric = (it: Item) => (String(effectiveSort) === 'percent' ? (it.percent ?? 0) : String(effectiveSort) === 'max' ? it.max : String(effectiveSort) === 'min' ? it.min : String(effectiveSort) === 'stdev' ? it.stdev : it.value);
  const max = Math.max(...items.map(metric), 0);
  const min = Math.min(...items.map(metric), max);

  const bars = items.map((it, idx) => {
    const mval = metric(it);
    const width = calculateBarWidth(mval, max, min).toFixed(1);
    const display = Math.round(mval).toLocaleString();
    const percentStr = (it.percent != null) ? ` (+${it.percent.toFixed(1)}%)` : '';
    const dropdownHtml = generateLoadoutDropdown(it.loadout);
    const hasDropdown = dropdownHtml && dropdownHtml.length > 0;
    const labelContent = (comparisonType === 'trinket' && (it as any).iconData)
      ? `<img src="https://wow.zamimg.com/images/wow/icons/small/${(it as any).iconData.icon}.jpg" alt="${(it as any).iconData.name}" class="trinket-icon" title="${(it as any).iconData.name}" /><span class="trinket-ilvl">${it.label}</span>`
      : `<span class="chart-label">${it.label}</span>`;
    let chartDisplay = display;
    let tooltip = display;
    if (comparisonType === 'trinket' && it.percent != null) {
      const percentDisplay = it.percent === 0 ? 'Baseline' : `+${it.percent.toFixed(1)}%`;
      const dpsIncreaseDisplay = (it as any).dpsIncrease === 0 ? 'Baseline' : `+${Math.round((it as any).dpsIncrease||0).toLocaleString()}`;
      if (String(effectiveSort) === 'percent') {
        chartDisplay = percentDisplay;
        tooltip = `${percentDisplay} (${display} DPS)`;
      } else if (String(effectiveSort) === 'stdev') {
        chartDisplay = Math.round(it.stdev).toLocaleString();
        tooltip = `${chartDisplay} StdDev (${display} DPS avg, lower is more consistent)`;
      } else {
        chartDisplay = dpsIncreaseDisplay;
        tooltip = String(effectiveSort) === 'dps' ? `${display} DPS (${dpsIncreaseDisplay} vs baseline)` : `${display} (Avg: ${display} DPS, ${dpsIncreaseDisplay} vs baseline)`;
      }
    }
    return `
      <div class="chart-item-wrapper" data-index="${idx}">
        <div class="chart-item-header">
          <div class="chart-item">
            <div class="chart-labels">
              <span class="chart-rank">#${idx + 1}</span>
              ${labelContent}
            </div>
            <div class="chart-bar-container">
              <div class="chart-bar-track">
                <div class="chart-bar" style="width: ${width}%; background: ${barColor};"></div>
              </div>
              <div class="chart-value" title="${tooltip}">${(comparisonType === 'trinket' && String(effectiveSort) === 'percent') ? (it.percent === 0 ? 'Baseline' : `+${(it.percent||0).toFixed(1)}%`) : chartDisplay}</div>
            </div>
          </div>
        </div>
        ${hasDropdown ? `<div class="chart-dropdown">${dropdownHtml}</div>` : ''}
      </div>`;
  }).join('');
  const titles: Record<string,string> = {
    dps: `${comparisonType === 'race' ? 'Race' : 'Trinket'} DPS Rankings (Average)`,
    max: `${comparisonType === 'race' ? 'Race' : 'Trinket'} DPS Rankings (Maximum)`,
    min: `${comparisonType === 'race' ? 'Race' : 'Trinket'} DPS Rankings (Minimum)`,
    stdev: `${comparisonType === 'race' ? 'Race' : 'Trinket'} DPS Consistency Rankings (Low StdDev = More Consistent)`,
    percent: `${comparisonType === 'race' ? 'Race' : 'Trinket'} Performance Rankings (% Increase)`
  };
  const title = titles[String(effectiveSort)] || `${comparisonType === 'race' ? 'Race' : 'Trinket'} DPS Rankings`;
  return `<h2 class="card-title-large">${title}</h2><div class="chart-bars">${bars}</div>`;
}
