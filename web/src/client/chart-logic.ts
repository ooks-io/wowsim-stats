// Thin TS wrapper that defers to the legacy ESM in /public/scripts.
// This lets us bundle a TS entry while we port the full logic.

export type InitOptions = {
  mode: string;
  fixedClass?: string;
  fixedSpec?: string;
  comparisonType?: string;
  isUnifiedMode?: boolean;
};

export async function initializeSimulationChart(options: InitOptions) {
  const mod = await import(/* @vite-ignore */ '/scripts/chart-logic.js');
  if (mod && typeof mod.initializeChart === 'function') {
    return mod.initializeChart(options);
  }
  throw new Error('initializeChart not found in /scripts/chart-logic.js');
}

// Also expose on window for simple invocation from inline scripts
(window as any).initializeSimulationChart = initializeSimulationChart;

export default initializeSimulationChart;

