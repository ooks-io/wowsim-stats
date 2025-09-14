import { buildRankingsUrl, PARAM } from './sim-chart/paths';
import { renderRankingsMetadataHTML } from './sim-chart/render/meta';
import { renderRankingsChartHTML } from './sim-chart/render/charts';
import { formatRaidBuffs as utilFormatRaidBuffs } from '../lib/utils';
import { showLoading, hideLoading, renderError, clearContent } from './sim-chart/ui';

function getParam(name: string, fallback?: string) {
  const url = new URL(window.location.href);
  return url.searchParams.get(name) || fallback || '';
}

async function init() {
  console.info('[sim] benchmarks init start');
  const params = {
    mode: 'rankings' as const,
    phase: getParam(PARAM.phase, 'p1'),
    encounterType: getParam(PARAM.encounter, 'raid'),
    targetCount: getParam(PARAM.targets, 'single'),
    duration: getParam(PARAM.duration, 'long'),
  };
  const url = buildRankingsUrl(params);
  const chartEl = document.getElementById('chart-container') as HTMLElement | null;
  const metaEl = document.getElementById('metadata-container') as HTMLElement | null;
  const loadingEl = document.getElementById('loading');
  const errorEl = document.getElementById('error') as HTMLElement | null;

  // show loading
  showLoading(metaEl, chartEl, loadingEl as any, errorEl);

  const resp = await fetch(url);
  if (!resp.ok) {
    console.error('[sim] benchmarks fetch failed', resp.status, resp.statusText);
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
  if (metaEl) metaEl.innerHTML = renderRankingsMetadataHTML(data.metadata, fmt);
  const CLASS_COLORS: Record<string, string> = (window as any).WoWConstants?.CLASS_COLORS || {};
  const sort = getParam(PARAM.sort, 'dps');
  if (chartEl) {
    chartEl.innerHTML = renderRankingsChartHTML(data, sort, CLASS_COLORS);
    chartEl.querySelectorAll('.chart-item-wrapper .chart-item-header').forEach((el) => {
      el.addEventListener('click', () => (el.parentElement as HTMLElement | null)?.classList.toggle('chart-item-expanded'));
    });
    // Final styles are handled in hideLoading
  }
}

export function __initSimulationBenchmarks() {
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
  window.addEventListener('simulation:statechange', init as any);
  window.addEventListener('popstate', init as any);
}

(window as any).__initSimulationBenchmarks = __initSimulationBenchmarks;

// Auto-init when bundled script loads (avoids inline caller timing issues)
if (!(window as any).__simBenchInitDone) {
  (window as any).__simBenchInitDone = true;
  try { __initSimulationBenchmarks(); } catch (e) { console.error('[sim] benchmarks init error', e); }
}
