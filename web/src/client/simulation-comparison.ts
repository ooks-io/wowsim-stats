import { buildComparisonUrl, PARAM } from './sim-chart/paths';
import { renderComparisonMetadataHTML } from './sim-chart/render/meta';
import { renderComparisonChartHTML } from './sim-chart/render/charts';
import { formatRaidBuffs as utilFormatRaidBuffs } from '../lib/utils';
import { showLoading, hideLoading, renderError, clearContent, updateComparisonUI } from './sim-chart/ui';

function getParam(name: string, fallback?: string) {
  const url = new URL(window.location.href);
  return url.searchParams.get(name) || fallback || '';
}

async function init() {
  console.info('[sim] comparison init start');
  const classSlug = getParam(PARAM.class, '');
  const specSlug = getParam(PARAM.spec, '');
  const cmpTypeSel = document.getElementById('cmpType') as HTMLSelectElement | null;
  const comparisonType = ((cmpTypeSel?.value as any) || getParam(PARAM.type, '')) as 'trinket' | 'race' | '';

  const params = {
    mode: 'comparison' as const,
    classSlug,
    specSlug,
    comparisonType,
    phase: getParam(PARAM.phase, 'p1'),
    encounterType: getParam(PARAM.encounter, 'raid'),
    targetCount: getParam(PARAM.targets, 'single'),
    duration: getParam(PARAM.duration, 'long'),
  };
  const sortBy = getParam(PARAM.sort, comparisonType === 'trinket' ? 'percent' : 'dps');
  const chartEl = document.getElementById('chart-container') as HTMLElement | null;
  const metaEl = document.getElementById('metadata-container') as HTMLElement | null;
  const loadingEl = document.getElementById('loading');
  const errorEl = document.getElementById('error') as HTMLElement | null;
  updateComparisonUI(comparisonType);
  // Require comparison type and class/spec selection before loading
  if (!comparisonType || !classSlug || !specSlug) {
    if (metaEl) metaEl.innerHTML = '';
    if (chartEl) chartEl.innerHTML = '<div class="card"><div>Select a comparison type, class and spec to see comparisons.</div></div>';
    return;
  }

  const url = buildComparisonUrl(params);

  // Show loading state
  showLoading(metaEl, chartEl, loadingEl as any, errorEl);

  const resp = await fetch(url);
  if (!resp.ok) {
    console.error('[sim] comparison fetch failed', resp.status, resp.statusText, url);
    renderError(errorEl, `Error loading data: ${resp.status} ${resp.statusText}`, metaEl, chartEl, loadingEl as any);
    clearContent(metaEl, chartEl);
    return;
  }
  const data = await resp.json();
  hideLoading(metaEl, chartEl, loadingEl as any, errorEl);

  const fmt = {
    formatDuration: (window as any).WoWConstants?.formatDuration || ((s: number) => `${Math.floor(s/60)}m ${s%60}s`),
    formatSimulationDate: (window as any).WoWConstants?.formatSimulationDate || ((d: any) => String(d)),
    formatRaidBuffs: utilFormatRaidBuffs,
  };
  const label = comparisonType === 'trinket' ? 'Trinket' : 'Race';
  const CLASS_COLORS: Record<string, string> = (window as any).WoWConstants?.CLASS_COLORS || {};
  const barColor = CLASS_COLORS[classSlug] || '#666';

  if (metaEl) metaEl.innerHTML = renderComparisonMetadataHTML(data.metadata, classSlug, specSlug, label as any, fmt);
  if (chartEl) {
    chartEl.innerHTML = renderComparisonChartHTML(data, sortBy, comparisonType as any, barColor);
    chartEl.querySelectorAll('.chart-item-wrapper .chart-item-header').forEach((el) => {
      el.addEventListener('click', () => (el.parentElement as HTMLElement | null)?.classList.toggle('chart-item-expanded'));
    });
    // Final styles are handled in hideLoading
  }
}

export function __initSimulationComparison() {
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
  window.addEventListener('simulation:statechange', init as any);
  window.addEventListener('popstate', init as any);
}

(window as any).__initSimulationComparison = __initSimulationComparison;

// Auto-init when bundled script loads (avoids inline caller timing issues)
if (!(window as any).__simCompInitDone) {
  (window as any).__simCompInitDone = true;
  try { __initSimulationComparison(); } catch (e) { console.error('[sim] comparison init error', e); }
}
