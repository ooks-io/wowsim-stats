// Vertical-bar spec-distribution chart. Server emits the data + tab buttons,
// this script renders the SVG and wires up the bucket switcher + hover tooltips.

import { getSpecIcon } from "../../../lib/wow-constants";

interface SpecEntry {
  spec_id: number;
  class_name: string;
  spec_name: string;
  // Total spec slot occurrences (a run with two of the spec contributes 2).
  count: number;
  // Distinct runs containing this spec (a run with two of the spec contributes 1).
  runs_with_spec: number;
}

interface DungeonSpecBucket {
  total_runs: number;
  entries: SpecEntry[];
}

interface SpecBucket {
  total_runs: number;
  entries: SpecEntry[];
  by_dungeon: Record<string, DungeonSpecBucket>;
}

// "picks" = total spec slot occurrences (a stacked spec contributes once per slot).
// "runs"  = distinct runs containing the spec.
type Metric = "picks" | "runs";

const SVG_NS = "http://www.w3.org/2000/svg";
const XLINK_NS = "http://www.w3.org/1999/xlink";

// Canonical class+spec ordering (alpha by class, role-grouped within: tank, healer, dps).
// `gapAfter` is recomputed per-render based on which specs survive filtering, so it's
// derived rather than stored here.
type Role = "tank" | "healer" | "dps";
type Spec = { specId: number; className: string; specName: string; role: Role };

const SPECS: Spec[] = [
  // Death Knight
  { specId: 250, className: "Death Knight", specName: "Blood", role: "tank" },
  { specId: 251, className: "Death Knight", specName: "Frost", role: "dps" },
  { specId: 252, className: "Death Knight", specName: "Unholy", role: "dps" },
  // Druid
  { specId: 104, className: "Druid", specName: "Guardian", role: "tank" },
  { specId: 105, className: "Druid", specName: "Restoration", role: "healer" },
  { specId: 102, className: "Druid", specName: "Balance", role: "dps" },
  { specId: 103, className: "Druid", specName: "Feral", role: "dps" },
  // Hunter
  { specId: 253, className: "Hunter", specName: "Beast Mastery", role: "dps" },
  { specId: 254, className: "Hunter", specName: "Marksmanship", role: "dps" },
  { specId: 255, className: "Hunter", specName: "Survival", role: "dps" },
  // Mage
  { specId: 62, className: "Mage", specName: "Arcane", role: "dps" },
  { specId: 63, className: "Mage", specName: "Fire", role: "dps" },
  { specId: 64, className: "Mage", specName: "Frost", role: "dps" },
  // Monk
  { specId: 268, className: "Monk", specName: "Brewmaster", role: "tank" },
  { specId: 270, className: "Monk", specName: "Mistweaver", role: "healer" },
  { specId: 269, className: "Monk", specName: "Windwalker", role: "dps" },
  // Paladin
  { specId: 66, className: "Paladin", specName: "Protection", role: "tank" },
  { specId: 65, className: "Paladin", specName: "Holy", role: "healer" },
  { specId: 70, className: "Paladin", specName: "Retribution", role: "dps" },
  // Priest
  { specId: 256, className: "Priest", specName: "Discipline", role: "healer" },
  { specId: 257, className: "Priest", specName: "Holy", role: "healer" },
  { specId: 258, className: "Priest", specName: "Shadow", role: "dps" },
  // Rogue
  { specId: 259, className: "Rogue", specName: "Assassination", role: "dps" },
  { specId: 260, className: "Rogue", specName: "Combat", role: "dps" },
  { specId: 261, className: "Rogue", specName: "Subtlety", role: "dps" },
  // Shaman
  { specId: 264, className: "Shaman", specName: "Restoration", role: "healer" },
  { specId: 262, className: "Shaman", specName: "Elemental", role: "dps" },
  { specId: 263, className: "Shaman", specName: "Enhancement", role: "dps" },
  // Warlock
  { specId: 265, className: "Warlock", specName: "Affliction", role: "dps" },
  { specId: 266, className: "Warlock", specName: "Demonology", role: "dps" },
  { specId: 267, className: "Warlock", specName: "Destruction", role: "dps" },
  // Warrior
  { specId: 73, className: "Warrior", specName: "Protection", role: "tank" },
  { specId: 71, className: "Warrior", specName: "Arms", role: "dps" },
  { specId: 72, className: "Warrior", specName: "Fury", role: "dps" },
];

function classSlug(name: string): string {
  return name.toLowerCase().replace(/\s+/g, "-");
}

function parseBuckets(el: HTMLElement): Record<string, SpecBucket> {
  const raw = el.getAttribute("data-buckets");
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function activeBucketKey(container: HTMLElement): string {
  const tab = container.querySelector<HTMLButtonElement>(
    ".spec-distribution-chart__tab.active",
  );
  return tab?.dataset.bucketKey ?? "all_runs";
}

function activeRoleKey(container: HTMLElement): string {
  const select = container.querySelector<HTMLSelectElement>("[data-role-select]");
  return select?.value ?? "all";
}

function activeMetric(container: HTMLElement): Metric {
  const select = container.querySelector<HTMLSelectElement>("[data-metric-select]");
  return select?.value === "runs" ? "runs" : "picks";
}

function activeDungeonKey(container: HTMLElement): string {
  const select = container.querySelector<HTMLSelectElement>("[data-dungeon-select]");
  return select?.value ?? "all";
}

// Pull the metric-relevant value out of an entry — bar height + tooltip both
// route through this so swapping the metric is one place.
function entryValue(e: SpecEntry, metric: Metric): number {
  return metric === "runs" ? e.runs_with_spec : e.count;
}

function renderChart(container: HTMLElement) {
  const canvas = container.querySelector(".spec-distribution-chart__canvas") as HTMLElement | null;
  if (!canvas) return;

  const buckets = parseBuckets(container);
  const bucketKey = activeBucketKey(container);
  const roleKey = activeRoleKey(container);
  const metric = activeMetric(container);
  const dungeonKey = activeDungeonKey(container);
  const fullBucket = buckets[bucketKey] ?? {
    total_runs: 0,
    entries: [],
    by_dungeon: {},
  };
  // When a dungeon is picked, slice to that dungeon's sub-bucket so total_runs,
  // entries, and the share denominator all come from the same scope.
  const bucket: { total_runs: number; entries: SpecEntry[] } =
    dungeonKey === "all"
      ? { total_runs: fullBucket.total_runs, entries: fullBucket.entries }
      : (fullBucket.by_dungeon?.[dungeonKey] ?? { total_runs: 0, entries: [] });
  const entries = bucket.entries;

  // build lookup spec_id -> entry for the active bucket
  const entryBySpec = new Map<number, SpecEntry>();
  for (const e of entries) {
    entryBySpec.set(e.spec_id, e);
  }
  const valueOf = (specId: number): number => {
    const e = entryBySpec.get(specId);
    return e ? entryValue(e, metric) : 0;
  };

  // Apply both filters: role + zero-hide. Specs with no count in this bucket
  // disappear entirely (no empty slot left behind).
  const visible = SPECS.filter((s) => {
    if (roleKey !== "all" && s.role !== roleKey) return false;
    return valueOf(s.specId) > 0;
  });

  // Recompute group breaks on the visible set (last spec of each class group).
  const visibleWithGap: Array<Spec & { gapAfter: boolean }> = visible.map((s, i) => ({
    ...s,
    gapAfter: i < visible.length - 1 && visible[i + 1].className !== s.className,
  }));

  // Empty state — nothing to render.
  if (visibleWithGap.length === 0) {
    canvas.innerHTML = '<p class="spec-distribution-chart__empty">No data for this filter.</p>';
    return;
  }

  // Sizing — fit the chart width to the container, but keep bars at a readable
  // minimum. If the natural layout exceeds container width, the canvas just scrolls.
  const containerW = canvas.clientWidth || container.clientWidth || 800;
  const minBarW = 14;
  const intraGap = 2;
  const interGap = 14;

  const naturalW = (() => {
    let w = 0;
    for (let i = 0; i < visibleWithGap.length; i++) {
      w += minBarW;
      if (i < visibleWithGap.length - 1) {
        w += visibleWithGap[i].gapAfter ? interGap : intraGap;
      }
    }
    return w;
  })();

  const padding = { top: 16, right: 12, bottom: 56, left: 48 };
  // expand barW if container has slack
  const innerWAvail = Math.max(containerW - padding.left - padding.right, naturalW);
  const slackPerBar = (innerWAvail - naturalW) / visibleWithGap.length;
  const barW = Math.max(minBarW, minBarW + slackPerBar);

  // re-compute slot positions with the chosen barW
  const slotX: number[] = [];
  let cursor = padding.left;
  for (let i = 0; i < visibleWithGap.length; i++) {
    slotX.push(cursor);
    cursor += barW;
    if (i < visibleWithGap.length - 1) {
      cursor += visibleWithGap[i].gapAfter ? interGap : intraGap;
    }
  }
  const innerW = cursor - padding.left;
  const totalW = innerW + padding.left + padding.right;
  const height = 280;
  const innerH = height - padding.top - padding.bottom;

  const maxCount = Math.max(1, ...visibleWithGap.map((s) => valueOf(s.specId)));
  // Denominator for the tooltip share %.
  //   - picks mode: fraction of all spec slots in the visible filter
  //   - runs mode:  fraction of distinct runs in the bucket containing the spec
  const shareDenominator =
    metric === "runs"
      ? bucket.total_runs
      : visibleWithGap.reduce((acc, s) => acc + valueOf(s.specId), 0);

  // Build SVG
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", `0 0 ${totalW} ${height}`);
  svg.setAttribute("width", String(totalW));
  svg.setAttribute("height", String(height));
  svg.setAttribute("class", "spec-distribution-chart__svg");

  // Y-axis ticks (5: 0/25/50/75/100)
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map((frac) => ({
    y: padding.top + innerH - frac * innerH,
    value: Math.round(maxCount * frac),
  }));
  for (const tick of yTicks) {
    const line = document.createElementNS(SVG_NS, "line");
    line.setAttribute("x1", String(padding.left));
    line.setAttribute("x2", String(padding.left + innerW));
    line.setAttribute("y1", String(tick.y));
    line.setAttribute("y2", String(tick.y));
    line.setAttribute("class", "spec-distribution-chart__gridline");
    svg.appendChild(line);

    const label = document.createElementNS(SVG_NS, "text");
    label.setAttribute("x", String(padding.left - 8));
    label.setAttribute("y", String(tick.y + 4));
    label.setAttribute("text-anchor", "end");
    label.setAttribute("class", "spec-distribution-chart__axis-label");
    label.textContent = tick.value.toLocaleString("en-US");
    svg.appendChild(label);
  }

  // Tooltip lives in the canvas DOM (positioned absolutely). Initialize top/left
  // so the hidden empty tooltip doesn't sit at the canvas's natural flow bottom
  // and inflate scrollHeight (which would otherwise trigger a phantom vertical
  // scrollbar when overflow-y is `auto`).
  const tooltip = document.createElement("div");
  tooltip.className = "spec-distribution-chart__tooltip";
  tooltip.style.opacity = "0";
  tooltip.style.top = "0";
  tooltip.style.left = "0";
  canvas.style.position = "relative";

  const iconSize = Math.min(barW + 4, 24);
  const iconY = padding.top + innerH + 8;

  // Highlight rect — drawn behind bars to mark the column the cursor is nearest to.
  // Goes into the SVG before bars so it paints underneath them.
  const highlight = document.createElementNS(SVG_NS, "rect");
  highlight.setAttribute("y", String(padding.top));
  highlight.setAttribute("height", String(innerH));
  highlight.setAttribute("width", String(barW));
  highlight.setAttribute("class", "spec-distribution-chart__highlight");
  highlight.setAttribute("opacity", "0");
  svg.appendChild(highlight);

  // Pre-compute per-bar metadata for the overlay's nearest-column lookup.
  const cols: Array<{ spec: Spec; value: number; x: number; y: number; cx: number }> = [];

  // Bars + icons
  for (let i = 0; i < visibleWithGap.length; i++) {
    const s = visibleWithGap[i];
    const value = valueOf(s.specId);
    const barH = (value / maxCount) * innerH;
    const x = slotX[i];
    const y = padding.top + innerH - barH;
    cols.push({ spec: s, value, x, y, cx: x + barW / 2 });

    // Bar
    const rect = document.createElementNS(SVG_NS, "rect");
    rect.setAttribute("x", String(x));
    rect.setAttribute("y", String(y));
    rect.setAttribute("width", String(barW));
    rect.setAttribute("height", String(Math.max(barH, 0.5)));
    rect.setAttribute(
      "class",
      `spec-distribution-chart__bar bar-${classSlug(s.className)}`,
    );
    svg.appendChild(rect);

    // Spec icon
    const iconURL = getSpecIcon(s.className, s.specName);
    if (iconURL) {
      const image = document.createElementNS(SVG_NS, "image");
      image.setAttribute("x", String(x + (barW - iconSize) / 2));
      image.setAttribute("y", String(iconY));
      image.setAttribute("width", String(iconSize));
      image.setAttribute("height", String(iconSize));
      image.setAttributeNS(XLINK_NS, "xlink:href", iconURL);
      image.setAttribute("href", iconURL);
      image.setAttribute("class", "spec-distribution-chart__icon");
      // native title fallback for non-hovering users / screen readers
      const title = document.createElementNS(SVG_NS, "title");
      title.textContent = `${s.specName} ${s.className}`;
      image.appendChild(title);
      svg.appendChild(image);
    }
  }

  // Single overlay across the chart area captures pointer events. We pick the
  // column whose center is closest to the cursor — so tiny bars are easy to
  // hover by pointing near them.
  const overlay = document.createElementNS(SVG_NS, "rect");
  overlay.setAttribute("x", String(padding.left));
  overlay.setAttribute("y", String(padding.top));
  overlay.setAttribute("width", String(innerW));
  overlay.setAttribute("height", String(innerH));
  overlay.setAttribute("fill", "transparent");
  overlay.setAttribute("class", "spec-distribution-chart__overlay");
  svg.appendChild(overlay);

  function nearestColIndex(svgX: number): number {
    let best = 0;
    let bestDist = Infinity;
    for (let i = 0; i < cols.length; i++) {
      const d = Math.abs(svgX - cols[i].cx);
      if (d < bestDist) {
        bestDist = d;
        best = i;
      }
    }
    return best;
  }

  function showHover(idx: number) {
    const c = cols[idx];
    highlight.setAttribute("x", String(c.x));
    highlight.setAttribute("opacity", "1");

    const sharePct =
      shareDenominator > 0 ? (c.value / shareDenominator) * 100 : 0;
    const valueText = c.value.toLocaleString("en-US");
    const mainLine =
      metric === "runs"
        ? `${valueText} of ${bucket.total_runs.toLocaleString("en-US")} runs`
        : `${valueText} picks`;
    tooltip.innerHTML =
      `<div class="spec-distribution-chart__tooltip-title">` +
      `<span class="text-${classSlug(c.spec.className)}">${c.spec.specName} ${c.spec.className}</span>` +
      `</div>` +
      `<div class="spec-distribution-chart__tooltip-value">${mainLine}` +
      ` <span class="spec-distribution-chart__tooltip-share">(${sharePct.toFixed(1)}%)</span></div>`;
    tooltip.style.opacity = "1";

    const canvasRect = canvas.getBoundingClientRect();
    const ratio = canvasRect.width / totalW;
    const px = c.cx * ratio;
    const py = c.y * ratio;
    const tipW = tooltip.offsetWidth;
    const tipH = tooltip.offsetHeight;
    let left = px - tipW / 2;
    if (left + tipW > canvasRect.width) left = canvasRect.width - tipW - 4;
    if (left < 0) left = 4;
    const top = Math.max(0, py - tipH - 8);
    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
  }

  function hideHover() {
    highlight.setAttribute("opacity", "0");
    tooltip.style.opacity = "0";
  }

  overlay.addEventListener("pointermove", (ev: PointerEvent) => {
    const rect = svg.getBoundingClientRect();
    const ratioX = totalW / rect.width;
    const svgX = (ev.clientX - rect.left) * ratioX;
    showHover(nearestColIndex(svgX));
  });
  overlay.addEventListener("pointerleave", hideHover);

  canvas.replaceChildren(svg, tooltip);
}

function wireControls(container: HTMLElement) {
  // Bucket = tab buttons. Role = standard select dropdown.
  const tabs = container.querySelectorAll<HTMLButtonElement>(".spec-distribution-chart__tab");
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      tabs.forEach((t) => {
        const active = t === tab;
        t.classList.toggle("active", active);
        t.setAttribute("aria-selected", active ? "true" : "false");
      });
      renderChart(container);
    });
  });

  const roleSelect = container.querySelector<HTMLSelectElement>("[data-role-select]");
  roleSelect?.addEventListener("change", () => renderChart(container));

  const metricSelect = container.querySelector<HTMLSelectElement>("[data-metric-select]");
  metricSelect?.addEventListener("change", () => renderChart(container));

  const dungeonSelect = container.querySelector<HTMLSelectElement>("[data-dungeon-select]");
  dungeonSelect?.addEventListener("change", () => renderChart(container));
}

export function initSpecDistributionCharts() {
  const charts = document.querySelectorAll<HTMLElement>(".spec-distribution-chart");
  charts.forEach((c) => {
    wireControls(c);
    renderChart(c);
  });

  // Re-render on resize so bars adapt to width changes (debounced)
  let resizeTimer: number | undefined;
  window.addEventListener(
    "resize",
    () => {
      if (resizeTimer) window.clearTimeout(resizeTimer);
      resizeTimer = window.setTimeout(() => {
        document
          .querySelectorAll<HTMLElement>(".spec-distribution-chart")
          .forEach((c) => renderChart(c));
      }, 150);
    },
    { passive: true },
  );
}
