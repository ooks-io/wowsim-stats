export type SortBy = "dps" | "max" | "min" | "stdev" | "percent" | string;

export function calculateBarWidth(
  value: number,
  max: number,
  min: number,
  sortBy?: SortBy,
) {
  const valueRange = max - min;
  if (valueRange <= 0) return 100;
  return ((value - min) / valueRange) * 100;
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
