import { buildComparisonUrl } from './sim-chart/paths';
import { renderComparisonMetadataHTML } from './sim-chart/render/meta';
import { renderComparisonChartHTML } from './sim-chart/render/charts';
import { formatRaidBuffs as utilFormatRaidBuffs } from '../lib/utils';

function getParam(name: string, fallback?: string) {
  const url = new URL(window.location.href);
  return url.searchParams.get(name) || fallback || '';
}

async function init() {
  console.info('[sim] comparison init start');
  const classSlug = getParam('class', '');
  const specSlug = getParam('spec', '');
  const cmpTypeSel = document.getElementById('cmpType') as HTMLSelectElement | null;
  const comparisonType = ((cmpTypeSel?.value as any) || getParam('type', '')) as 'trinket' | 'race' | '';

  const params = {
    mode: 'comparison' as const,
    classSlug,
    specSlug,
    comparisonType,
    phase: getParam('phase', 'p1'),
    encounterType: getParam('encounter', 'raid'),
    targetCount: getParam('targets', 'single'),
    duration: getParam('duration', 'long'),
  };
  const sortBy = getParam('sort', comparisonType === 'trinket' ? 'percent' : 'dps');
  const chartEl = document.getElementById('chart-container') as HTMLElement | null;
  const metaEl = document.getElementById('metadata-container') as HTMLElement | null;
  const loadingEl = document.getElementById('loading');
  const errorEl = document.getElementById('error') as HTMLElement | null;
  const trinketCallout = document.getElementById('trinket-callout');
  if (trinketCallout) trinketCallout.classList.toggle('hidden', comparisonType !== 'trinket');
  // Require comparison type and class/spec selection before loading
  if (!comparisonType || !classSlug || !specSlug) {
    if (metaEl) metaEl.innerHTML = '';
    if (chartEl) chartEl.innerHTML = '<div class="card"><div>Select a comparison type, class and spec to see comparisons.</div></div>';
    return;
  }

  const url = buildComparisonUrl(params);

  // Show loading state
  if (loadingEl) loadingEl.classList.remove('hidden');
  if (errorEl) errorEl.classList.add('hidden');
  if (chartEl) chartEl.style.opacity = '0.5';
  if (metaEl) metaEl.style.opacity = '0.5';

  const resp = await fetch(url);
  if (!resp.ok) {
    console.error('[sim] comparison fetch failed', resp.status, resp.statusText, url);
    if (loadingEl) loadingEl.classList.add('hidden');
    if (errorEl) {
      errorEl.textContent = `Error loading data: ${resp.status} ${resp.statusText}`;
      errorEl.classList.remove('hidden');
    }
    if (metaEl) metaEl.innerHTML = '';
    if (chartEl) chartEl.innerHTML = '';
    if (chartEl) chartEl.style.opacity = '1';
    if (metaEl) metaEl.style.opacity = '1';
    return;
  }
  const data = await resp.json();
  if (loadingEl) loadingEl.classList.add('hidden');
  if (errorEl) errorEl.classList.add('hidden');

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
    chartEl.style.opacity = '1';
    if (metaEl) metaEl.style.opacity = '1';
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
