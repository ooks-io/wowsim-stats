import { buildRankingsUrl } from './sim-chart/paths';
import { renderRankingsMetadataHTML } from './sim-chart/render/meta';
import { renderRankingsChartHTML } from './sim-chart/render/charts';
import { formatRaidBuffs as utilFormatRaidBuffs } from '../lib/utils';

function getParam(name: string, fallback?: string) {
  const url = new URL(window.location.href);
  return url.searchParams.get(name) || fallback || '';
}

async function init() {
  console.info('[sim] benchmarks init start');
  const params = {
    mode: 'rankings' as const,
    phase: getParam('phase', 'p1'),
    encounterType: getParam('encounter', 'raid'),
    targetCount: getParam('targets', 'single'),
    duration: getParam('duration', 'long'),
  };
  const url = buildRankingsUrl(params);
  const chartEl = document.getElementById('chart-container') as HTMLElement | null;
  const metaEl = document.getElementById('metadata-container') as HTMLElement | null;
  const loadingEl = document.getElementById('loading');
  const errorEl = document.getElementById('error') as HTMLElement | null;

  // show loading
  if (loadingEl) loadingEl.classList.remove('hidden');
  if (errorEl) errorEl.classList.add('hidden');
  if (chartEl) chartEl.style.opacity = '0.5';
  if (metaEl) metaEl.style.opacity = '0.5';

  const resp = await fetch(url);
  if (!resp.ok) {
    console.error('[sim] benchmarks fetch failed', resp.status, resp.statusText);
    if (loadingEl) loadingEl.classList.add('hidden');
    if (errorEl) {
      errorEl.textContent = `Error loading data: ${resp.status} ${resp.statusText}`;
      errorEl.classList.remove('hidden');
    }
    if (chartEl) chartEl.innerHTML = '';
    if (metaEl) metaEl.innerHTML = '';
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
  if (metaEl) metaEl.innerHTML = renderRankingsMetadataHTML(data.metadata, fmt);
  const CLASS_COLORS: Record<string, string> = (window as any).WoWConstants?.CLASS_COLORS || {};
  const sort = getParam('sort', 'dps');
  if (chartEl) {
    chartEl.innerHTML = renderRankingsChartHTML(data, sort, CLASS_COLORS);
    chartEl.querySelectorAll('.chart-item-wrapper .chart-item-header').forEach((el) => {
      el.addEventListener('click', () => (el.parentElement as HTMLElement | null)?.classList.toggle('chart-item-expanded'));
    });
    chartEl.style.opacity = '1';
    if (metaEl) metaEl.style.opacity = '1';
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
