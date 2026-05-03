// Renders a simple line+area chart into each .activity-chart on the page.
// Reads its data from the data-points attribute (JSON-encoded array of
// { week_start, run_count }). Hover anywhere on the chart to see exact
// values for the nearest week.

interface Point {
  week_start: string;
  run_count: number;
}

const SVG_NS = "http://www.w3.org/2000/svg";

function parseData(el: HTMLElement): Point[] {
  const raw = el.getAttribute("data-points");
  if (!raw) return [];
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}

function formatDateLabel(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

function renderChart(container: HTMLElement, points: Point[]) {
  const canvas = container.querySelector(".activity-chart__canvas") as HTMLElement | null;
  if (!canvas) return;

  // Render width — measure container, fall back to a reasonable default
  const width = Math.max(canvas.clientWidth || container.clientWidth || 600, 320);
  const height = 220;
  const padding = { top: 16, right: 16, bottom: 32, left: 56 };
  const innerW = width - padding.left - padding.right;
  const innerH = height - padding.top - padding.bottom;

  if (points.length === 0) {
    canvas.innerHTML = '<p class="activity-chart__empty">No activity data.</p>';
    return;
  }

  const maxCount = Math.max(...points.map((p) => p.run_count), 1);

  // Map each point to (x, y) inside the chart area
  const xStep = points.length > 1 ? innerW / (points.length - 1) : innerW;
  const xy = points.map((p, i) => ({
    x: padding.left + i * xStep,
    y: padding.top + innerH - (p.run_count / maxCount) * innerH,
    point: p,
  }));

  // Build path strings for the line and the filled area
  const linePath = xy
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x.toFixed(2)} ${p.y.toFixed(2)}`)
    .join(" ");
  const areaPath =
    `${linePath} L ${xy[xy.length - 1].x.toFixed(2)} ${(padding.top + innerH).toFixed(2)} ` +
    `L ${xy[0].x.toFixed(2)} ${(padding.top + innerH).toFixed(2)} Z`;

  // Y-axis tick values (5 ticks: 0, 25%, 50%, 75%, 100%)
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map((frac) => ({
    y: padding.top + innerH - frac * innerH,
    value: Math.round(maxCount * frac),
  }));

  // X-axis labels — first, middle, last (more would clutter)
  const xTickIdxs =
    points.length <= 2
      ? points.map((_, i) => i)
      : [0, Math.floor(points.length / 2), points.length - 1];

  // Build SVG
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.setAttribute("width", String(width));
  svg.setAttribute("height", String(height));
  svg.setAttribute("class", "activity-chart__svg");

  // Y-axis gridlines + labels
  for (const tick of yTicks) {
    const line = document.createElementNS(SVG_NS, "line");
    line.setAttribute("x1", String(padding.left));
    line.setAttribute("x2", String(padding.left + innerW));
    line.setAttribute("y1", String(tick.y));
    line.setAttribute("y2", String(tick.y));
    line.setAttribute("class", "activity-chart__gridline");
    svg.appendChild(line);

    const label = document.createElementNS(SVG_NS, "text");
    label.setAttribute("x", String(padding.left - 8));
    label.setAttribute("y", String(tick.y + 4));
    label.setAttribute("text-anchor", "end");
    label.setAttribute("class", "activity-chart__axis-label");
    label.textContent = tick.value.toLocaleString("en-US");
    svg.appendChild(label);
  }

  // X-axis labels
  for (const i of xTickIdxs) {
    const label = document.createElementNS(SVG_NS, "text");
    label.setAttribute("x", String(xy[i].x));
    label.setAttribute("y", String(padding.top + innerH + 20));
    label.setAttribute("text-anchor", "middle");
    label.setAttribute("class", "activity-chart__axis-label");
    label.textContent = new Date(points[i].week_start).toLocaleDateString("en-US", {
      month: "short",
      year: "2-digit",
    });
    svg.appendChild(label);
  }

  // Filled area under line
  const area = document.createElementNS(SVG_NS, "path");
  area.setAttribute("d", areaPath);
  area.setAttribute("class", "activity-chart__area");
  svg.appendChild(area);

  // Line on top of area
  const line = document.createElementNS(SVG_NS, "path");
  line.setAttribute("d", linePath);
  line.setAttribute("class", "activity-chart__line");
  svg.appendChild(line);

  // Hover crosshair + dot (initially hidden)
  const crosshair = document.createElementNS(SVG_NS, "line");
  crosshair.setAttribute("y1", String(padding.top));
  crosshair.setAttribute("y2", String(padding.top + innerH));
  crosshair.setAttribute("class", "activity-chart__crosshair");
  crosshair.setAttribute("opacity", "0");
  svg.appendChild(crosshair);

  const hoverDot = document.createElementNS(SVG_NS, "circle");
  hoverDot.setAttribute("r", "4");
  hoverDot.setAttribute("class", "activity-chart__dot");
  hoverDot.setAttribute("opacity", "0");
  svg.appendChild(hoverDot);

  // Invisible overlay rect that captures pointer events across the chart area
  const overlay = document.createElementNS(SVG_NS, "rect");
  overlay.setAttribute("x", String(padding.left));
  overlay.setAttribute("y", String(padding.top));
  overlay.setAttribute("width", String(innerW));
  overlay.setAttribute("height", String(innerH));
  overlay.setAttribute("fill", "transparent");
  overlay.setAttribute("class", "activity-chart__overlay");
  svg.appendChild(overlay);

  // Tooltip (DOM, positioned absolutely)
  const tooltip = document.createElement("div");
  tooltip.className = "activity-chart__tooltip";
  tooltip.style.opacity = "0";
  canvas.style.position = "relative";

  function nearestIndex(svgX: number): number {
    // Convert pointer x in SVG units to nearest point index
    const idx = Math.round((svgX - padding.left) / xStep);
    return Math.max(0, Math.min(points.length - 1, idx));
  }

  function showHover(idx: number, clientX: number) {
    const p = xy[idx];
    crosshair.setAttribute("x1", String(p.x));
    crosshair.setAttribute("x2", String(p.x));
    crosshair.setAttribute("opacity", "1");
    hoverDot.setAttribute("cx", String(p.x));
    hoverDot.setAttribute("cy", String(p.y));
    hoverDot.setAttribute("opacity", "1");

    tooltip.innerHTML =
      `<div class="activity-chart__tooltip-date">${formatDateLabel(p.point.week_start)}</div>` +
      `<div class="activity-chart__tooltip-value">${p.point.run_count.toLocaleString("en-US")} runs</div>`;
    tooltip.style.opacity = "1";

    // Position tooltip relative to canvas: convert SVG x back to pixels
    const canvasRect = canvas.getBoundingClientRect();
    const ratio = canvasRect.width / width;
    let left = p.x * ratio + 8;
    const tipW = tooltip.offsetWidth;
    if (left + tipW > canvasRect.width) {
      left = p.x * ratio - tipW - 8;
    }
    tooltip.style.left = `${Math.max(0, left)}px`;
    tooltip.style.top = `${Math.max(0, p.y * ratio - 30)}px`;
  }

  function hideHover() {
    crosshair.setAttribute("opacity", "0");
    hoverDot.setAttribute("opacity", "0");
    tooltip.style.opacity = "0";
  }

  overlay.addEventListener("pointermove", (ev: PointerEvent) => {
    const rect = svg.getBoundingClientRect();
    const ratioX = width / rect.width;
    const svgX = (ev.clientX - rect.left) * ratioX;
    showHover(nearestIndex(svgX), ev.clientX);
  });
  overlay.addEventListener("pointerleave", hideHover);

  canvas.replaceChildren(svg, tooltip);
}

export function initActivityCharts() {
  const charts = document.querySelectorAll<HTMLElement>(".activity-chart");
  charts.forEach((c) => {
    const points = parseData(c);
    renderChart(c, points);
  });

  // Re-render on resize so the chart adapts to width changes (debounced)
  let resizeTimer: number | undefined;
  window.addEventListener(
    "resize",
    () => {
      if (resizeTimer) window.clearTimeout(resizeTimer);
      resizeTimer = window.setTimeout(() => {
        document.querySelectorAll<HTMLElement>(".activity-chart").forEach((c) => {
          renderChart(c, parseData(c));
        });
      }, 150);
    },
    { passive: true },
  );
}
