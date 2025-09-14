import {
  getClassColor,
  getSpecInfo,
  getSpecIcon,
} from "../lib/client-utils.ts";

function decorate(container: Document | HTMLElement = document) {
  // Spec icons for team members and rows
  container
    .querySelectorAll(".best-runs-table .spec-icon-placeholder")
    .forEach((ph) => {
      const el = ph as HTMLElement;
      const specId = parseInt(el.dataset.specId || "0");
      if (!specId || el.querySelector("img")) return;
      const spec = getSpecInfo(specId);
      const icon = spec ? getSpecIcon(spec.class, spec.spec) : null;
      if (icon) {
        const img = document.createElement("img");
        img.src = icon;
        img.alt = `${spec?.spec} ${spec?.class}`;
        img.className = "spec-icon";
        el.appendChild(img);
      }
    });

  // Class colors for member links
  container.querySelectorAll(".best-runs-table .member-link").forEach((lnk) => {
    const a = lnk as HTMLElement;
    const specId = parseInt(a.dataset.specId || "0");
    if (!specId) return;
    const spec = getSpecInfo(specId);
    if (spec) a.style.color = getClassColor(spec.class);
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => decorate());
} else {
  decorate();
}

// Expose for dynamic content re-decoration if needed
(window as any).__decorateBestRuns = decorate;
