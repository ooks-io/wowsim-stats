type Formatters = {
  formatDuration: (s: number) => string;
  formatSimulationDate: (d: any) => string;
  formatRaidBuffs: (b: any) => string;
};

export function renderRankingsMetadataHTML(metadata: any, fmt: Formatters) {
  const simulationDate = metadata.timestamp
    ? fmt.formatSimulationDate(metadata.timestamp)
    : "";
  return `
    <div class="card">
      <h3 class="card-title">DPS Rankings Details</h3>
      <div class="info-grid">
        <div class="info-item"><span class="info-label">Simulation Type</span><span class="info-value">DPS Rankings</span></div>
        <div class="info-item"><span class="info-label">Iterations</span><span class="info-value">${metadata.iterations?.toLocaleString?.() || metadata.iterations}</span></div>
        <div class="info-item"><span class="info-label">Specs Tested</span><span class="info-value">${metadata.specCount ?? ""}</span></div>
        <div class="info-item"><span class="info-label">Encounter Duration</span><span class="info-value">${fmt.formatDuration(metadata.encounterDuration)}</span></div>
        <div class="info-item"><span class="info-label">Duration Variation</span><span class="info-value">±${metadata.encounterVariation}s</span></div>
        <div class="info-item"><span class="info-label">Target Count</span><span class="info-value">${metadata.targetCount}</span></div>
        <div class="info-item"><span class="info-label">Date Simulated</span><span class="info-value">${simulationDate}</span></div>
        <div class="info-item info-item-wide">
          <span class="info-label">Active Raid Buffs</span>
          <div class="callout-note">
            <svg class="callout-note-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span class="callout-note-content">Warrior and Shaman simulations are counted within the Skull Banner/Stormlash Totem totals, not as additional buffs.</span>
          </div>
          <span class="info-value info-value-wrap">${fmt.formatRaidBuffs(metadata.raidBuffs)}</span>
        </div>
      </div>
    </div>`;
}

export function renderComparisonMetadataHTML(
  metadata: any,
  currentClass: string,
  currentSpec: string,
  label: "Race" | "Trinket",
  fmt: Formatters,
) {
  const simulationDate = metadata.timestamp
    ? fmt.formatSimulationDate(metadata.timestamp)
    : "";
  const itemCount =
    metadata?.itemsTested ??
    (metadata?.results ? Object.keys(metadata.results).length : "");
  return `
    <div class="card">
      <h3 class="card-title">${label} Comparison Details</h3>
      <div class="info-grid">
        <div class="info-item"><span class="info-label">Class/Spec</span><span class="info-value">${metadata.class || currentClass} ${metadata.spec || currentSpec}</span></div>
        <div class="info-item"><span class="info-label">Comparison Type</span><span class="info-value">${label}s</span></div>
        <div class="info-item"><span class="info-label">Iterations</span><span class="info-value">${metadata.iterations?.toLocaleString?.() || metadata.iterations}</span></div>
        <div class="info-item"><span class="info-label">${label}s Tested</span><span class="info-value">${itemCount}</span></div>
        <div class="info-item"><span class="info-label">Encounter Duration</span><span class="info-value">${fmt.formatDuration(metadata.encounterDuration)}</span></div>
        <div class="info-item"><span class="info-label">Duration Variation</span><span class="info-value">±${metadata.encounterVariation}s</span></div>
        <div class="info-item"><span class="info-label">Target Count</span><span class="info-value">${metadata.targetCount}</span></div>
        <div class="info-item"><span class="info-label">Date Simulated</span><span class="info-value">${simulationDate}</span></div>
        <div class="info-item info-item-wide"><span class="info-label">Active Raid Buffs</span><span class="info-value info-value-wrap">${fmt.formatRaidBuffs(metadata.raidBuffs)}</span></div>
      </div>
    </div>`;
}
