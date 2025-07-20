console.log('Dynamic rankings script loaded!');

// WoW class colors for consistency
const classColors = {
  death_knight: '#C41E3A',
  druid: '#FF7C0A', 
  hunter: '#AAD372',
  mage: '#3FC7EB',
  monk: '#00FF98',
  paladin: '#F48CBA',
  priest: '#FFFFFF',
  rogue: '#FFF468',
  shaman: '#0070DD',
  warlock: '#8788EE',
  warrior: '#C69B6D'
};

class DynamicRankings {
  constructor() {
    this.currentData = null;
    this.bindEvents();
    this.loadInitialData();
  }

  bindEvents() {
    const selects = document.querySelectorAll('#targetCount, #encounterType, #duration, #phase');
    selects.forEach(select => {
      select.addEventListener('change', () => this.loadData());
    });
  }

  async loadInitialData() {
    await this.loadData();
  }

  getFileName() {
    const targetCount = document.getElementById('targetCount').value;
    const encounterType = document.getElementById('encounterType').value;
    const duration = document.getElementById('duration').value;
    const phase = document.getElementById('phase').value;
    
    return `dps_${phase}_${encounterType}_${targetCount}_${duration}.json`;
  }

  async loadData() {
    const loadingEl = document.getElementById('loading');
    const errorEl = document.getElementById('error');
    const metadataContainer = document.getElementById('metadata-container');
    const chartContainer = document.getElementById('chart-container');

    if (!loadingEl || !errorEl || !metadataContainer || !chartContainer) {
      console.error('Required elements not found');
      return;
    }

    const scrollY = window.scrollY;

    metadataContainer.style.opacity = '0.5';
    chartContainer.style.opacity = '0.5';
    loadingEl.classList.remove('hidden');
    errorEl.classList.add('hidden');

    try {
      const fileName = this.getFileName();
      console.log('Loading:', fileName);
      const response = await fetch(`/data/${fileName}`);
      
      if (!response.ok) {
        throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
      }

      this.currentData = await response.json();
      console.log('Data loaded successfully');
      
      loadingEl.classList.add('hidden');
      
      this.renderMetadata();
      this.renderChart();
      
      metadataContainer.style.opacity = '1';
      chartContainer.style.opacity = '1';
      
      requestAnimationFrame(() => {
        window.scrollTo(0, scrollY);
      });
      
    } catch (error) {
      console.error('Error loading data:', error);
      loadingEl.classList.add('hidden');
      errorEl.textContent = `Error loading simulation data: ${error.message}`;
      errorEl.classList.remove('hidden');
      
      metadataContainer.innerHTML = '';
      chartContainer.innerHTML = '';
      metadataContainer.style.opacity = '1';
      chartContainer.style.opacity = '1';
    }
  }

  renderMetadata() {
    if (!this.currentData) return;

    const container = document.getElementById('metadata-container');
    const metadata = this.currentData.metadata;
    
    const simulationDate = new Date(metadata.timestamp).toLocaleString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      timeZoneName: 'short'
    });

    const formatDuration = (seconds) => {
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = seconds % 60;
      return minutes > 0 ? `${minutes}m ${remainingSeconds}s` : `${remainingSeconds}s`;
    };

    const formatRaidBuffs = (buffs) => {
      const activeBuffs = Object.entries(buffs)
        .filter(([_, value]) => value !== false && value !== 0)
        .map(([key, value]) => {
          const readable = key.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase());
          return typeof value === 'number' && value > 1 ? `${readable} (${value})` : readable;
        });
      return activeBuffs.join(', ');
    };

    container.innerHTML = `
      <div class="simulation-metadata">
        <h3 class="metadata-title">Simulation Details</h3>
        <div class="metadata-grid">
          <div class="metadata-item">
            <span class="metadata-label">Iterations</span>
            <span class="metadata-value">${metadata.iterations.toLocaleString()}</span>
          </div>
          <div class="metadata-item">
            <span class="metadata-label">Specs Tested</span>
            <span class="metadata-value">${metadata.specCount}</span>
          </div>
          <div class="metadata-item">
            <span class="metadata-label">Encounter Duration</span>
            <span class="metadata-value">${formatDuration(metadata.encounterDuration)}</span>
          </div>
          <div class="metadata-item">
            <span class="metadata-label">Duration Variation</span>
            <span class="metadata-value">±${metadata.encounterVariation}s</span>
          </div>
          <div class="metadata-item">
            <span class="metadata-label">Target Count</span>
            <span class="metadata-value">${metadata.targetCount}</span>
          </div>
          <div class="metadata-item">
            <span class="metadata-label">Date Simulated</span>
            <span class="metadata-value">${simulationDate}</span>
          </div>
          <div class="metadata-item metadata-item-wide">
            <span class="metadata-label">Active Raid Buffs</span>
            <span class="metadata-value metadata-value-wrap">${formatRaidBuffs(metadata.raidBuffs)}</span>
          </div>
        </div>
      </div>
    `;
  }

  renderChart() {
    if (!this.currentData) return;

    const container = document.getElementById('chart-container');
    
    const chartItems = [];
    
    for (const [className, classSpecs] of Object.entries(this.currentData.results)) {
      for (const [specName, specData] of Object.entries(classSpecs)) {
        chartItems.push({
          label: specName,
          sublabel: className,
          value: specData.dps,
          category: className
        });
      }
    }

    chartItems.sort((a, b) => b.value - a.value);

    const maxDps = Math.max(...chartItems.map(item => item.value));
    
    container.innerHTML = `
      <div class="chart-container">
        <h2 class="chart-title">DPS Rankings</h2>
        <div class="chart-bars">
          ${chartItems.map((item, index) => `
            <div class="chart-item">
              <div class="chart-labels">
                <span class="chart-rank">#${index + 1}</span>
                <span class="chart-label">${item.label}</span>
                <span class="chart-sublabel">${item.sublabel}</span>
              </div>
              <div class="chart-bar-container">
                <div class="chart-bar" style="width: ${(item.value / maxDps) * 100}%; background-color: ${classColors[item.category] || '#666'};">
                </div>
                <span class="chart-value">${Math.round(item.value).toLocaleString()}</span>
              </div>
            </div>
          `).join('')}
        </div>
      </div>
    `;
  }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  console.log('DOM loaded, initializing DynamicRankings...');
  new DynamicRankings();
});