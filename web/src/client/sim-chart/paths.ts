export type Mode = 'rankings' | 'comparison' | 'unified' | string;

export type PathParams = {
  mode: Mode;
  classSlug?: string | null;
  specSlug?: string | null;
  comparisonType?: 'trinket' | 'race' | string;
  phase?: string;
  encounterType?: string;
  targetCount?: string;
  duration?: string;
  unifiedSelection?: 'benchmarks' | 'comparisons' | string | null;
};

export function buildRankingsUrl(params: PathParams): string {
  const phase = params.phase || 'p1';
  const encounter = params.encounterType || 'raid';
  const target = params.targetCount || 'single';
  const duration = params.duration || 'long';
  const file = `dps_${phase}_${encounter}_${target}_${duration}.json`;
  return `/data/rankings/${file}`;
}

export function buildComparisonUrl(params: PathParams): string {
  const cls = (params.classSlug || '').toLowerCase();
  const spec = (params.specSlug || '').toLowerCase();
  if (!cls || !spec) throw new Error('Select a class and spec');
  const type = (params.comparisonType || 'trinket') as string;
  const phase = params.phase || 'p1';
  const encounter = params.encounterType || 'raid';
  const target = params.targetCount || 'single';
  const duration = params.duration || 'long';
  const file = `${cls}_${spec}_${type}_${phase}_${encounter}_${target}_${duration}.json`;
  const folder = type === 'trinket' ? 'trinkets' : 'race';
  return `/data/comparison/${folder}/${cls}/${spec}/${file}`;
}
