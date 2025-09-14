export function showLoading(
  metaEl: HTMLElement | null,
  chartEl: HTMLElement | null,
  loadingEl: HTMLElement | null,
  errorEl: HTMLElement | null,
) {
  if (loadingEl) loadingEl.classList.remove("hidden");
  if (errorEl) errorEl.classList.add("hidden");
  if (metaEl) metaEl.style.opacity = "0.5";
  if (chartEl) chartEl.style.opacity = "0.5";
}

export function hideLoading(
  metaEl: HTMLElement | null,
  chartEl: HTMLElement | null,
  loadingEl: HTMLElement | null,
  errorEl: HTMLElement | null,
) {
  if (loadingEl) loadingEl.classList.add("hidden");
  if (errorEl) errorEl.classList.add("hidden");
  if (metaEl) metaEl.style.opacity = "1";
  if (chartEl) chartEl.style.opacity = "1";
}

export function renderError(
  errorEl: HTMLElement | null,
  message: string,
  metaEl?: HTMLElement | null,
  chartEl?: HTMLElement | null,
  loadingEl?: HTMLElement | null,
) {
  if (loadingEl) loadingEl.classList.add("hidden");
  if (errorEl) {
    errorEl.textContent = message;
    errorEl.classList.remove("hidden");
  }
  if (metaEl) metaEl.style.opacity = "1";
  if (chartEl) chartEl.style.opacity = "1";
}

export function clearContent(
  metaEl: HTMLElement | null,
  chartEl: HTMLElement | null,
) {
  if (metaEl) metaEl.innerHTML = "";
  if (chartEl) chartEl.innerHTML = "";
}

// Comparison UI helper: toggles UI elements based on comparison type
export function updateComparisonUI(comparisonType: string) {
  try {
    const trinketCallout = document.getElementById("trinket-callout");
    if (trinketCallout)
      trinketCallout.classList.toggle("hidden", comparisonType !== "trinket");
    const sortSel = document.getElementById("sort") as HTMLSelectElement | null;
    const percentOpt = sortSel
      ? (sortSel.querySelector(
          'option[value="percent"]',
        ) as HTMLOptionElement | null)
      : null;
    if (percentOpt) {
      percentOpt.style.display = comparisonType === "trinket" ? "" : "none";
      if (
        comparisonType !== "trinket" &&
        sortSel &&
        sortSel.value === "percent"
      )
        sortSel.value = "dps";
      if (
        comparisonType === "trinket" &&
        sortSel &&
        (!sortSel.value || sortSel.value === "dps")
      )
        sortSel.value = "percent";
    }
  } catch (e) {
    console.warn("[sim] updateComparisonUI error", e);
  }
}
