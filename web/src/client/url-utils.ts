export function getParam(name: string, fallback?: string) {
  try {
    const url = new URL(window.location.href);
    const v = url.searchParams.get(name);
    return v == null || v === "" ? fallback || "" : v;
  } catch {
    return fallback || "";
  }
}

export function pushParams(params: Record<string, any>, pathOverride?: string) {
  try {
    const url = new URL(window.location.href);
    if (pathOverride) url.pathname = pathOverride;
    Object.entries(params).forEach(([k, v]) => {
      if (v == null || v === "") url.searchParams.delete(k);
      else url.searchParams.set(k, String(v));
    });
    window.history.pushState({}, "", url.toString());
    window.dispatchEvent(new CustomEvent("simulation:statechange"));
  } catch (e) {
    console.warn("[url] pushParams failed", e);
  }
}

// Expose on window for inline scripts
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(window as any).SimURL = { getParam, pushParams };

export const PARAM = {
  phase: "phase",
  encounter: "encounter",
  targets: "targets",
  duration: "duration",
  sort: "sort",
  type: "type",
  class: "class",
  spec: "spec",
} as const;

(window as any).SimURL.PARAM = PARAM;
