// Vertical-bar chart of run counts per dungeon for the active bucket.
// Visual treatment mirrors SpecDistributionChart so the two read consistently
// when stacked on the stats page.

import { getDungeonIconUrl } from "../../../lib/dungeonIcons";

interface SpecEntry {
  spec_id: number;
  class_name: string;
  spec_name: string;
  count: number;
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

const SVG_NS = "http://www.w3.org/2000/svg";
const XLINK_NS = "http://www.w3.org/1999/xlink";

// matches dungeon-thresholds.ts canonical id order
const DUNGEONS: Array<{ id: number; name: string; slug: string }> = [
  {
    id: 2,
    name: "Temple of the Jade Serpent",
    slug: "temple-of-the-jade-serpent",
  },
  { id: 56, name: "Stormstout Brewery", slug: "stormstout-brewery" },
  { id: 57, name: "Gate of the Setting Sun", slug: "gate-of-the-setting-sun" },
  { id: 58, name: "Shado-Pan Monastery", slug: "shado-pan-monastery" },
  { id: 59, name: "Siege of Niuzao Temple", slug: "siege-of-niuzao-temple" },
  { id: 60, name: "Mogu'shan Palace", slug: "mogu-shan-palace" },
  { id: 76, name: "Scholomance", slug: "scholomance" },
  { id: 77, name: "Scarlet Halls", slug: "scarlet-halls" },
  { id: 78, name: "Scarlet Monastery", slug: "scarlet-monastery" },
];

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
    ".dungeon-distribution-chart__tab.active",
  );
  return tab?.dataset.bucketKey ?? "all_runs";
}

function renderChart(container: HTMLElement) {
  const canvas = container.querySelector(
    ".dungeon-distribution-chart__canvas",
  ) as HTMLElement | null;
  if (!canvas) return;

  const buckets = parseBuckets(container);
  const bucketKey = activeBucketKey(container);
  const fullBucket = buckets[bucketKey] ?? {
    total_runs: 0,
    entries: [],
    by_dungeon: {},
  };

  // skip dungeons with 0 runs in this bucket
  const visible = DUNGEONS.map((d) => {
    const sub = fullBucket.by_dungeon?.[String(d.id)];
    return { ...d, count: sub?.total_runs ?? 0 };
  }).filter((d) => d.count > 0);

  if (visible.length === 0) {
    canvas.innerHTML =
      '<p class="dungeon-distribution-chart__empty">No data for this filter.</p>';
    return;
  }

  // smaller bar minimum than the spec chart - only ~9 bars so they fit fine
  const containerW = canvas.clientWidth || container.clientWidth || 800;
  const minBarW = 32;
  const gap = 18;
  const padding = { top: 16, right: 12, bottom: 56, left: 56 };

  const naturalW = visible.length * minBarW + (visible.length - 1) * gap;
  const innerWAvail = Math.max(
    containerW - padding.left - padding.right,
    naturalW,
  );
  const slackPerBar = (innerWAvail - naturalW) / visible.length;
  const barW = Math.max(minBarW, minBarW + slackPerBar);

  const slotX: number[] = [];
  let cursor = padding.left;
  for (let i = 0; i < visible.length; i++) {
    slotX.push(cursor);
    cursor += barW;
    if (i < visible.length - 1) cursor += gap;
  }
  const innerW = cursor - padding.left;
  const totalW = innerW + padding.left + padding.right;
  const height = 280;
  const innerH = height - padding.top - padding.bottom;

  const maxCount = Math.max(1, ...visible.map((d) => d.count));
  const totalDenom = visible.reduce((acc, d) => acc + d.count, 0);

  // Build SVG
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", `0 0 ${totalW} ${height}`);
  svg.setAttribute("width", String(totalW));
  svg.setAttribute("height", String(height));
  svg.setAttribute("class", "dungeon-distribution-chart__svg");

  // Y-axis ticks
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
    line.setAttribute("class", "dungeon-distribution-chart__gridline");
    svg.appendChild(line);

    const label = document.createElementNS(SVG_NS, "text");
    label.setAttribute("x", String(padding.left - 8));
    label.setAttribute("y", String(tick.y + 4));
    label.setAttribute("text-anchor", "end");
    label.setAttribute("class", "dungeon-distribution-chart__axis-label");
    label.textContent = tick.value.toLocaleString("en-US");
    svg.appendChild(label);
  }

  // Tooltip + initial position so it doesn't inflate scrollHeight (same trick
  // as the spec chart).
  const tooltip = document.createElement("div");
  tooltip.className = "dungeon-distribution-chart__tooltip";
  tooltip.style.opacity = "0";
  tooltip.style.top = "0";
  tooltip.style.left = "0";
  canvas.style.position = "relative";

  const iconSize = Math.min(barW + 4, 36);
  const iconY = padding.top + innerH + 8;

  // Highlight rect drawn behind bars.
  const highlight = document.createElementNS(SVG_NS, "rect");
  highlight.setAttribute("y", String(padding.top));
  highlight.setAttribute("height", String(innerH));
  highlight.setAttribute("width", String(barW));
  highlight.setAttribute("class", "dungeon-distribution-chart__highlight");
  highlight.setAttribute("opacity", "0");
  svg.appendChild(highlight);

  const cols: Array<{
    dungeon: (typeof visible)[number];
    count: number;
    x: number;
    y: number;
    cx: number;
  }> = [];

  for (let i = 0; i < visible.length; i++) {
    const d = visible[i];
    const barH = (d.count / maxCount) * innerH;
    const x = slotX[i];
    const y = padding.top + innerH - barH;
    cols.push({ dungeon: d, count: d.count, x, y, cx: x + barW / 2 });

    const rect = document.createElementNS(SVG_NS, "rect");
    rect.setAttribute("x", String(x));
    rect.setAttribute("y", String(y));
    rect.setAttribute("width", String(barW));
    rect.setAttribute("height", String(Math.max(barH, 0.5)));
    rect.setAttribute("class", "dungeon-distribution-chart__bar");
    svg.appendChild(rect);

    const iconURL = getDungeonIconUrl(d.slug, d.name);
    if (iconURL) {
      const image = document.createElementNS(SVG_NS, "image");
      image.setAttribute("x", String(x + (barW - iconSize) / 2));
      image.setAttribute("y", String(iconY));
      image.setAttribute("width", String(iconSize));
      image.setAttribute("height", String(iconSize));
      image.setAttributeNS(XLINK_NS, "xlink:href", iconURL);
      image.setAttribute("href", iconURL);
      image.setAttribute("class", "dungeon-distribution-chart__icon");
      const title = document.createElementNS(SVG_NS, "title");
      title.textContent = d.name;
      image.appendChild(title);
      svg.appendChild(image);
    }
  }

  // Single overlay for nearest-column hover (cheap to hit even with thin bars).
  const overlay = document.createElementNS(SVG_NS, "rect");
  overlay.setAttribute("x", String(padding.left));
  overlay.setAttribute("y", String(padding.top));
  overlay.setAttribute("width", String(innerW));
  overlay.setAttribute("height", String(innerH));
  overlay.setAttribute("fill", "transparent");
  overlay.setAttribute("class", "dungeon-distribution-chart__overlay");
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

    const sharePct = totalDenom > 0 ? (c.count / totalDenom) * 100 : 0;
    tooltip.innerHTML =
      `<div class="dungeon-distribution-chart__tooltip-title">${c.dungeon.name}</div>` +
      `<div class="dungeon-distribution-chart__tooltip-value">${c.count.toLocaleString("en-US")} runs` +
      ` <span class="dungeon-distribution-chart__tooltip-share">(${sharePct.toFixed(1)}%)</span></div>`;
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
  const tabs = container.querySelectorAll<HTMLButtonElement>(
    ".dungeon-distribution-chart__tab",
  );
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
}

export function initDungeonDistributionCharts() {
  const charts = document.querySelectorAll<HTMLElement>(
    ".dungeon-distribution-chart",
  );
  charts.forEach((c) => {
    wireControls(c);
    renderChart(c);
  });

  let resizeTimer: number | undefined;
  window.addEventListener(
    "resize",
    () => {
      if (resizeTimer) window.clearTimeout(resizeTimer);
      resizeTimer = window.setTimeout(() => {
        document
          .querySelectorAll<HTMLElement>(".dungeon-distribution-chart")
          .forEach((c) => renderChart(c));
      }, 150);
    },
    { passive: true },
  );
}
