import { renderBestRunsHeader, renderBestRunsRow } from "../lib/bestRunsRenderer";

async function loadStatus() {
  const loading = document.getElementById("status-loading");
  const error = document.getElementById("status-error");
  const content = document.getElementById("status-content");
  const regionFilter = document.getElementById("region-filter") as HTMLSelectElement | null;

  const timeAgo = (tsMs: number): string => {
    if (!tsMs) return "";
    const sec = Math.max(0, Math.floor((Date.now() - tsMs) / 1000));
    const mins = Math.floor(sec / 60);
    const hours = Math.floor(mins / 60);
    const days = Math.floor(hours / 24);
    if (days > 0) return `${days} day${days === 1 ? "" : "s"} ago`;
    if (hours > 0) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
    if (mins > 0) return `${mins} minute${mins === 1 ? "" : "s"} ago`;
    return `${sec} seconds ago`;
  };

  const renderLatest = (data: any) => {
    if (!content) return;
    const runs = Array.isArray(data?.latest_runs) ? data.latest_runs : [];
    const region = (regionFilter?.value || "").toLowerCase();
    const filtered = region
      ? runs.filter((r: any) => (r.region || "").toLowerCase() === region)
      : runs;

    const rowsHTML = filtered
      .map((r: any) => {
        const fakeRun: any = {
          dungeon_name: r.dungeon_name || r.dungeon_slug || "",
          dungeon_slug: r.dungeon_slug || "",
          duration: Number(r?.latest_run?.duration_ms) || 0,
          all_members: Array.isArray(r?.latest_run?.members)
            ? r.latest_run.members.map((m: any) => ({
                name: m.name,
                spec_id: m.spec_id,
                region: m.region || r.region,
                realm_slug: m.realm_slug || r.realm_slug,
              }))
            : [],
          __status: {
            realm_name: r.realm_name,
            realm_slug: r.realm_slug,
            region: r.region,
            most_recent_iso: r.most_recent_iso,
            most_recent_ts: r.most_recent,
            period_id: r.period_id,
          },
        };
        return renderBestRunsRow(fakeRun, "status");
      })
      .join("");

    const headerHTML = renderBestRunsHeader("status");
    content.innerHTML = `
      <div class="best-runs-table" data-mode="status">
        <div class="best-runs-header">${headerHTML}</div>
        <div class="best-runs-body">${rowsHTML}</div>
      </div>
    `;
    const decorate = (window as any).__decorateBestRuns as
      | ((root?: Document | HTMLElement) => void)
      | undefined;
    if (typeof decorate === "function") decorate(content);
  };

  const coverageContainer = document.getElementById("realm-coverage");
  const realmFilter = document.getElementById("realm-filter") as HTMLSelectElement | null;
  let lastCombinedData: any = null;

  const fetchRealmStatus = async (region: string, slug: string): Promise<any | null> => {
    if (!region || !slug) return null;
    const url = `/api/status/${region}/${slug}.json`;
    const res = await fetch(url, { cache: "no-store" });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return await res.json();
  };

  const populateRealmOptions = (data: any) => {
    if (!realmFilter) return;
    const region = (regionFilter?.value || "").toLowerCase();
    let realms: any[] = [];
    const rs = Array.isArray(data?.realm_status) ? data.realm_status : [];
    if (rs.length > 0) {
      realms = rs.filter((r: any) => !region || (r.region || "").toLowerCase() === region);
    } else {
      // Fallback: derive from latest_runs
      const runs = Array.isArray(data?.latest_runs) ? data.latest_runs : [];
      const seen = new Set<string>();
      for (const r of runs) {
        const key = `${(r.region || '').toLowerCase()}|${r.realm_slug}`;
        if (region && (r.region || '').toLowerCase() !== region) continue;
        if (seen.has(key)) continue;
        seen.add(key);
        realms.push({ region: r.region, realm_slug: r.realm_slug, realm_name: r.realm_name });
      }
      realms.sort((a, b) => String(a.realm_name || a.realm_slug).localeCompare(String(b.realm_name || b.realm_slug)));
    }
    const opts = ["<option value=\"\">(Select realm)</option>", ...realms.map((r: any) => `<option value="${r.region}|${r.realm_slug}">${(r.realm_name || r.realm_slug)} [${(r.region || "").toUpperCase()}]</option>`)].join("");
    realmFilter.innerHTML = opts;
  };

  const renderRealmRuns = (realmData: any) => {
    if (!content) return;
    const region = String(realmData?.region || "");
    const realmName = String(realmData?.realm_name || realmData?.realm_slug || "");
    const dungeons = Array.isArray(realmData?.dungeons) ? realmData.dungeons : [];
    const runsHTML = dungeons
      .filter((d: any) => d?.latest_run?.completed_timestamp)
      .map((d: any) => {
        const lr = d.latest_run || {};
        const fakeRun: any = {
          dungeon_name: d.dungeon_name || d.dungeon_slug || "",
          dungeon_slug: d.dungeon_slug || "",
          duration: Number(lr.duration_ms || 0),
          all_members: Array.isArray(lr.members)
            ? lr.members.map((m: any) => ({
                name: m.name,
                spec_id: m.spec_id,
                region: m.region || region,
                realm_slug: m.realm_slug || realmFilter?.value.split("|")[1] || "",
              }))
            : [],
          __status: {
            realm_name: realmName,
            realm_slug: realmFilter?.value.split("|")[1] || "",
            region,
            most_recent_iso: d.latest_iso,
            most_recent_ts: d.latest_ts,
            period_id: d.latest_period,
          },
        };
        return renderBestRunsRow(fakeRun, "status");
      })
      .join("");

    const headerHTML = renderBestRunsHeader("status");
    content.innerHTML = `
      <div class="best-runs-table" data-mode="status">
        <div class="best-runs-header">${headerHTML}</div>
        <div class="best-runs-body">${runsHTML}</div>
      </div>
    `;
    const decorate = (window as any).__decorateBestRuns as
      | ((root?: Document | HTMLElement) => void)
      | undefined;
    if (typeof decorate === "function") decorate(content);
  };

  const fetchAndRender = async () => {
    if (loading) loading.style.display = "block";
    if (error) error.style.display = "none";
    if (content) content.style.display = "none";
    if (coverageContainer) coverageContainer.style.display = "none";
    try {
      const res = await fetch("/api/status/latest-runs.json", { cache: "no-store" });
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      const data = await res.json();
      lastCombinedData = data;
      populateRealmOptions(data);
      // If no realm selected, show global latest list; otherwise show per-realm
      if (realmFilter && realmFilter.value) {
        const [reg, slug] = realmFilter.value.split("|");
        const realmData = await fetchRealmStatus(reg, slug);
        renderRealmRuns(realmData);
      } else {
        renderLatest(data);
      }
      if (content) content.style.display = "block";
    } catch (e) {
      if (error) error.style.display = "block";
      console.warn("status fetch error", e);
    } finally {
      if (loading) loading.style.display = "none";
    }
  };

  regionFilter?.addEventListener("change", () => fetchAndRender());
  realmFilter?.addEventListener("change", async () => {
    if (!realmFilter?.value) {
      // Return to global latest
      if (coverageContainer) { coverageContainer.style.display = "none"; coverageContainer.innerHTML = ""; }
      if (lastCombinedData) renderLatest(lastCombinedData);
      return;
    }
    const [reg, slug] = realmFilter.value.split("|");
    try {
      const realmData = await fetchRealmStatus(reg, slug);
      renderRealmRuns(realmData);
    } catch (e) {
      console.warn("realm status fetch error", e);
    }
  });
  fetchAndRender();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", loadStatus);
} else {
  loadStatus();
}
