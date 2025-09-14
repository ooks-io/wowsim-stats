export type SortBy = "dps" | "max" | "min" | "stdev" | "percent" | string;

export function calculateBarWidth(value: number, max: number, min: number) {
  if (max <= 0) return 0;
  // Hybrid scaling with minimum visibility
  const valueRange = Math.max(max - min, 0);
  const average = (max + min) / 2 || max;
  const cov = average ? valueRange / average : 0;
  const rangeWeight = Math.min(cov * 1.5, 0.8);
  const zeroWeight = 1 - rangeWeight;
  const zeroPct = max ? value / max : 0;
  const rangePct = valueRange > 0 ? (value - min) / valueRange : 1;
  const pct = zeroPct * zeroWeight + rangePct * rangeWeight;
  const minPct = 0.15;
  return Math.max(pct, minPct) * 100;
}

export type ItemForSort = {
  value: number;
  max: number;
  min: number;
  stdev: number;
  percent?: number;
};

export function sortItems<T extends ItemForSort>(
  items: T[],
  sortBy: SortBy,
): T[] {
  if (sortBy === "stdev") return items.sort((a, b) => a.stdev - b.stdev);
  if (sortBy === "max") return items.sort((a, b) => b.max - a.max);
  if (sortBy === "min") return items.sort((a, b) => b.min - a.min);
  if (sortBy === "percent")
    return items.sort((a, b) => (b.percent || 0) - (a.percent || 0));
  return items.sort((a, b) => b.value - a.value);
}
