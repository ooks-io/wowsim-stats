// Use globally available constants passed from Astro define:vars
const {
  CLASS_COLORS,
  SPEC_OPTIONS,
  formatDuration,
  formatRaidBuffs,
  formatSimulationDate,
  formatRace,
} = window.WoWConstants;
export function initializeChart({
  mode,
  fixedClass,
  fixedSpec,
  comparisonType,
  isUnifiedMode,
}) {
  console.log("Unified chart script loaded with:", {
    mode,
    fixedClass,
    fixedSpec,
    comparisonType,
  });

  // Loadout formatting functions
  const formatLoadout = (loadout) => {
    if (!loadout) return null;

    const sections = [];

    // Character Info
    if (loadout.race || loadout.profession1 || loadout.profession2) {
      const items = [];
      if (loadout.race)
        items.push({ label: "Race", value: formatRace(loadout.race) });
      if (loadout.profession1)
        items.push({ label: "Profession 1", value: loadout.profession1 });
      if (loadout.profession2)
        items.push({ label: "Profession 2", value: loadout.profession2 });
      sections.push({ title: "Character", items });
    }

    // Talents & Glyphs
    if (loadout.talents || loadout.glyphs) {
      const items = [];
      if (loadout.talents) {
        const talentItems = formatTalents(loadout.talents);
        if (talentItems && talentItems.length > 0) {
          items.push(...talentItems);
        }
      }
      if (loadout.glyphs) {
        const glyphItems = formatGlyphs(loadout.glyphs);
        if (glyphItems && glyphItems.length > 0) {
          // Add glyphs as structured items instead of text
          items.push(...glyphItems);
        }
      }
      sections.push({ title: "Talents & Glyphs", items });
    }

    // Consumables
    if (loadout.consumables) {
      const items = formatConsumables(loadout.consumables);
      if (items.length > 0) sections.push({ title: "Consumables", items });
    }

    // Equipment Summary
    if (loadout.equipment && loadout.equipment.items) {
      // Use existing ItemDatabase for equipment formatting for now
      const equipmentSummary =
        window.EquipmentUtils?.formatEquipmentSummary?.(
          loadout.equipment.items,
        ) || [];
      if (equipmentSummary.length > 0) {
        sections.push({
          title: "Equipment",
          items: equipmentSummary,
          isEquipment: true,
        });
      }
    }

    return sections;
  };

  // const formatClass = (className) => {
  //   return className.replace('Class', '').replace(/([A-Z])/g, ' $1').trim();
  // };

  const formatTalents = (talents) => {
    if (!talents || !talents.talents || !Array.isArray(talents.talents)) {
      return [];
    }

    // Create a single talents item with all talents listed
    const talentList = talents.talents
      .map((talent) => {
        const iconUrl = talent.icon
          ? `https://wow.zamimg.com/images/wow/icons/small/${talent.icon}.jpg`
          : null;
        const iconHtml = iconUrl
          ? `<img src="${iconUrl}" alt="${talent.name}" class="talent-icon-inline" loading="lazy" />`
          : "";
        const wowheadUrl = talent.spellId
          ? `https://www.wowhead.com/mop-classic/spell=${talent.spellId}`
          : null;
        const nameHtml = wowheadUrl
          ? `<a href="${wowheadUrl}" target="_blank" class="talent-link">${talent.name}</a>`
          : `<span class="talent-name">${talent.name}</span>`;
        return `<div class="talent-line">${iconHtml}${nameHtml}</div>`;
      })
      .join("");

    return [
      {
        label: "Talents",
        value: `<div class="talents-list">${talentList}</div>`,
        isTalentList: true,
      },
    ];
  };

  const formatGlyphs = (glyphs) => {
    const glyphContainers = [];

    // Major Glyphs
    const majorGlyphs = [];
    ["major1", "major2", "major3"].forEach((slot) => {
      if (glyphs[slot]) {
        const glyphName = glyphs[`${slot}Name`] || `Glyph ${glyphs[slot]}`;
        const iconUrl = glyphs[`${slot}Icon`]
          ? `https://wow.zamimg.com/images/wow/icons/small/${glyphs[`${slot}Icon`]}.jpg`
          : null;
        const spellId = glyphs[`${slot}SpellId`];
        const wowheadUrl = spellId
          ? `https://www.wowhead.com/mop-classic/spell=${spellId}`
          : `https://www.wowhead.com/mop-classic/item=${glyphs[slot]}`;

        const iconHtml = iconUrl
          ? `<img src="${iconUrl}" alt="${glyphName}" class="glyph-icon-inline" loading="lazy" />`
          : "";
        const nameHtml = wowheadUrl
          ? `<a href="${wowheadUrl}" target="_blank" class="glyph-link">${glyphName}</a>`
          : `<span class="glyph-name">${glyphName}</span>`;
        majorGlyphs.push(
          `<div class="glyph-line">${iconHtml}${nameHtml}</div>`,
        );
      }
    });

    // Minor Glyphs
    const minorGlyphs = [];
    ["minor1", "minor2", "minor3"].forEach((slot) => {
      if (glyphs[slot]) {
        const glyphName = glyphs[`${slot}Name`] || `Glyph ${glyphs[slot]}`;
        const iconUrl = glyphs[`${slot}Icon`]
          ? `https://wow.zamimg.com/images/wow/icons/small/${glyphs[`${slot}Icon`]}.jpg`
          : null;
        const spellId = glyphs[`${slot}SpellId`];
        const wowheadUrl = spellId
          ? `https://www.wowhead.com/mop-classic/spell=${spellId}`
          : `https://www.wowhead.com/mop-classic/item=${glyphs[slot]}`;

        const iconHtml = iconUrl
          ? `<img src="${iconUrl}" alt="${glyphName}" class="glyph-icon-inline" loading="lazy" />`
          : "";
        const nameHtml = wowheadUrl
          ? `<a href="${wowheadUrl}" target="_blank" class="glyph-link">${glyphName}</a>`
          : `<span class="glyph-name">${glyphName}</span>`;
        minorGlyphs.push(
          `<div class="glyph-line">${iconHtml}${nameHtml}</div>`,
        );
      }
    });

    // Add containers if glyphs exist
    if (majorGlyphs.length > 0) {
      glyphContainers.push({
        label: "Major Glyphs",
        value: `<div class="glyphs-list">${majorGlyphs.join("")}</div>`,
        isGlyphList: true,
      });
    }

    if (minorGlyphs.length > 0) {
      glyphContainers.push({
        label: "Minor Glyphs",
        value: `<div class="glyphs-list">${minorGlyphs.join("")}</div>`,
        isGlyphList: true,
      });
    }

    return glyphContainers;
  };

  const formatConsumables = (consumables) => {
    const items = [];
    if (consumables.flaskId) {
      items.push({
        label: "Flask",
        value: consumables.flaskName || `Item ${consumables.flaskId}`,
        wowheadUrl: `https://www.wowhead.com/mop-classic/item=${consumables.flaskId}`,
        iconUrl: consumables.flaskIcon
          ? `https://wow.zamimg.com/images/wow/icons/large/${consumables.flaskIcon}.jpg`
          : null,
        quality: consumables.flaskQuality,
        isItem: true,
      });
    }
    if (consumables.foodId) {
      items.push({
        label: "Food",
        value: consumables.foodName || `Item ${consumables.foodId}`,
        wowheadUrl: `https://www.wowhead.com/mop-classic/item=${consumables.foodId}`,
        iconUrl: consumables.foodIcon
          ? `https://wow.zamimg.com/images/wow/icons/large/${consumables.foodIcon}.jpg`
          : null,
        quality: consumables.foodQuality,
        isItem: true,
      });
    }
    if (consumables.potId) {
      items.push({
        label: "Potion",
        value: consumables.potName || `Item ${consumables.potId}`,
        wowheadUrl: `https://www.wowhead.com/mop-classic/item=${consumables.potId}`,
        iconUrl: consumables.potIcon
          ? `https://wow.zamimg.com/images/wow/icons/large/${consumables.potIcon}.jpg`
          : null,
        quality: consumables.potQuality,
        isItem: true,
      });
    }
    if (consumables.prepotId) {
      items.push({
        label: "Pre-Potion",
        value: consumables.prepotName || `Item ${consumables.prepotId}`,
        wowheadUrl: `https://www.wowhead.com/mop-classic/item=${consumables.prepotId}`,
        iconUrl: consumables.prepotIcon
          ? `https://wow.zamimg.com/images/wow/icons/large/${consumables.prepotIcon}.jpg`
          : null,
        quality: consumables.prepotQuality,
        isItem: true,
      });
    }
    return items;
  };

  // Generate loadout dropdown HTML
  const generateLoadoutDropdown = (loadout, _chartData) => {
    const sections = formatLoadout(loadout);
    if (!sections || sections.length === 0) return "";

    const sectionsHtml = sections
      .map((section) => {
        if (section.isEquipment) {
          // Use unified equipment rendering if available
          const equipmentHtml = section.items
            .map((eq) => {
              // Convert chart format to unified format
              const itemData = {
                slot: eq.slot,
                item_id: eq.itemId,
                item_name: eq.itemName,
                quality: eq.quality,
                item_icon_slug: eq.iconUrl
                  ? eq.iconUrl.match(/\/([^\/]+)\.jpg$/)?.[1]
                  : null,
                itemDetails: eq.itemDetails, // Pass through existing details
              };

              // Use global function if available, otherwise fallback
              if (window.EquipmentUtils?.createItemElement) {
                return window.EquipmentUtils.createItemElement(itemData, {
                  isHTML: true,
                  showIcon: true,
                });
              }

              // Fallback to original rendering
              const iconHtml = eq.iconUrl
                ? `<img src="${eq.iconUrl}" alt="${eq.itemName}" class="equipment-icon" loading="lazy" />`
                : "";
              const qualityClass = eq.quality ? `quality-${eq.quality}` : "";
              const detailsHtml =
                eq.itemDetails?.length > 0
                  ? `<div class="item-tooltip-details">${eq.itemDetails.map((detail) => `<div class="equipment-detail">${detail}</div>`).join("")}</div>`
                  : "";

              return `
                <div class="equipment-slot">
                  <div class="equipment-slot-header">
                    <span class="equipment-slot-name">${eq.slot}</span>
                  </div>
                  <div class="equipment-item-tooltip">
                    <div class="equipment-item-header">
                      ${iconHtml}
                      <a href="${eq.wowheadUrl}" target="_blank" class="equipment-item-link ${qualityClass}">${eq.itemName}</a>
                    </div>
                    <div class="equipment-item-details">
                      ${detailsHtml}
                    </div>
                  </div>
                </div>
              `;
            })
            .join("");

          return `
        <div class="loadout-section">
          <h4 class="loadout-title">${section.title}</h4>
          <div class="equipment-grid">
            ${equipmentHtml}
          </div>
        </div>
      `;
        } else {
          // Regular sections
          const itemsHtml = section.items
            .map((item) => {
              const valueClass =
                item.isSpecial === "talents"
                  ? "loadout-value loadout-talents"
                  : "loadout-value";

              let valueContent;
              if (item.isTalentList || item.isGlyphList) {
                // Special handling for talent and glyph lists
                valueContent = item.value;
              } else if (
                (item.isItem || item.isGlyph || item.isTalent) &&
                item.wowheadUrl
              ) {
                const iconHtml = item.iconUrl
                  ? `<img src="${item.iconUrl}" alt="${item.value}" class="consumable-icon" loading="lazy" />`
                  : "";
                const qualityClass = item.quality
                  ? `quality-${item.quality}`
                  : "";
                valueContent = `
            <div class="consumable-item-header">
              ${iconHtml}
              <a href="${item.wowheadUrl}" target="_blank" class="equipment-item-link ${qualityClass}">${item.value}</a>
            </div>
          `;
              } else {
                valueContent = `<span class="${valueClass}">${item.value}</span>`;
              }

              return `
          <div class="loadout-item">
            <span class="loadout-label">${item.label}</span>
            ${valueContent}
          </div>
        `;
            })
            .join("");

          return `
        <div class="loadout-section">
          <h4 class="loadout-title">${section.title}</h4>
          <div class="loadout-grid">
            ${itemsHtml}
          </div>
        </div>
      `;
        }
      })
      .join("");

    // Create WowSim button if simLink is available
    const wowSimButton = loadout.simLink
      ? `<a href="${loadout.simLink}" target="_blank" class="loadout-button wowsim-button">Open in WoWSims</a>`
      : "";

    return `
    <div class="chart-dropdown">
      ${wowSimButton}
      <div class="loadout-notice">
        <strong>Note:</strong> Profile details are a work in progress.
      </div>
      ${sectionsHtml}
    </div>
  `;
  };

  // Chart interaction handlers - global function for onclick handlers
  window.toggleChartItem = (element) => {
    const wrapper = element.closest(".chart-item-wrapper");
    if (!wrapper) return;

    const isExpanded = wrapper.classList.contains("chart-item-expanded");

    // Close all other expanded items
    document.querySelectorAll(".chart-item-expanded").forEach((item) => {
      if (item !== wrapper) {
        item.classList.remove("chart-item-expanded");
      }
    });

    // Toggle current item
    wrapper.classList.toggle("chart-item-expanded", !isExpanded);
  };

  const classSpecs = SPEC_OPTIONS;
  const classColors = CLASS_COLORS;

  class UnifiedChart {
    constructor() {
      this.currentData = null;
      this.isRankingsMode = mode === "rankings";
      this.isUnifiedMode = mode === "unified";
      this.isFixedClassSpec = fixedClass && fixedSpec;
      this.currentSimulationMode = "benchmarks";

      this.initializeControls();
      this.bindEvents();
      this.loadInitialData();
    }

    initializeControls() {
      if (this.isUnifiedMode) {
        console.log("Unified mode initialized");
        this.setupUnifiedMode();
      } else if (this.isRankingsMode) {
        console.log("Rankings mode initialized");
      } else if (this.isFixedClassSpec) {
        console.log("Fixed class/spec mode:", fixedClass, fixedSpec);
      } else {
        this.updateSpecOptions();
      }

      if (!this.isRankingsMode && comparisonType !== "both") {
        const comparisonSelect = document.getElementById("comparisonType");
        if (comparisonSelect) {
          comparisonSelect.value = comparisonType;
        }
      }

      // Initialize phase options for comparison modes
      if (!this.isRankingsMode || this.isUnifiedMode) {
        this.updatePhaseOptions();
        this.updateTrinketCallout();
      }
    }

    bindEvents() {
      if (this.isUnifiedMode) {
        // Unified mode event bindings
        const simulationModeSelect = document.getElementById("simulationMode");
        if (simulationModeSelect) {
          simulationModeSelect.addEventListener("change", () => {
            this.currentSimulationMode = simulationModeSelect.value;
            this.updateControlVisibility();
            this.clearData();
            this.loadData();
          });
        }
      }

      if (
        (this.isUnifiedMode || !this.isRankingsMode) &&
        !this.isFixedClassSpec
      ) {
        // class change updates spec options - comparison mode only
        const classSelect = document.getElementById("class");
        if (classSelect) {
          classSelect.addEventListener("change", () => {
            this.updateSpecOptions();
            this.clearData();
          });
        }

        const dataSelects = document.querySelectorAll("#spec, #comparisonType");
        dataSelects.forEach((select) => {
          select.addEventListener("change", () => {
            if (select.id === "comparisonType") {
              this.updateSortOptions();
            }
            this.loadData();
          });
        });
      }

      const commonSelects = document.querySelectorAll(
        "#targetCount, #encounterType, #duration, #phase",
      );
      commonSelects.forEach((select) => {
        select.addEventListener("change", () => this.loadData());
      });

      // sort by only re-renders chart
      const sortBySelect = document.getElementById("sortBy");
      if (sortBySelect) {
        sortBySelect.addEventListener("change", () => this.renderChart());
      }
    }

    updateSpecOptions() {
      if (this.isRankingsMode || this.isFixedClassSpec) return;

      const classSelect = document.getElementById("class");
      const specSelect = document.getElementById("spec");

      if (!classSelect || !specSelect) return;

      const selectedClass = classSelect.value;

      // clear existing options
      specSelect.innerHTML = '<option value="">Select Specialization</option>';

      if (selectedClass && classSpecs[selectedClass]) {
        specSelect.disabled = false;
        classSpecs[selectedClass].forEach((spec) => {
          const option = document.createElement("option");
          option.value = spec.value;
          option.textContent = spec.label;
          specSelect.appendChild(option);
        });
      } else {
        specSelect.disabled = true;
      }
    }

    getCurrentClass() {
      if (this.isFixedClassSpec) return fixedClass;
      if (this.isRankingsMode) return null;
      const classSelect = document.getElementById("class");
      return classSelect ? classSelect.value : "";
    }

    getCurrentSpec() {
      if (this.isFixedClassSpec) return fixedSpec;
      if (this.isRankingsMode) return null;
      const specSelect = document.getElementById("spec");
      return specSelect ? specSelect.value : "";
    }

    getCurrentComparisonType() {
      if (this.isRankingsMode) return null;
      if (comparisonType !== "both") return comparisonType;
      const comparisonSelect = document.getElementById("comparisonType");
      return comparisonSelect ? comparisonSelect.value : "race";
    }

    updateSortOptions() {
      const sortBySelect = document.getElementById("sortBy");
      if (!sortBySelect) return;

      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");
      const currentComparisonType = this.getCurrentComparisonType();
      const percentOption = sortBySelect.querySelector(
        'option[value="percent"]',
      );

      if (percentOption) {
        // Hide percentage option for benchmarks mode or non-trinket comparisons
        const showPercentOption =
          !isBenchmarksMode && currentComparisonType === "trinket";
        percentOption.style.display = showPercentOption ? "" : "none";

        if (showPercentOption) {
          // Default to percentage for trinket comparisons
          if (sortBySelect.value === "dps" || !sortBySelect.value) {
            sortBySelect.value = "percent";
          }
        } else {
          // If percentage is selected but not available, switch to DPS
          if (sortBySelect.value === "percent") {
            sortBySelect.value = "dps";
          }
        }
      }

      // Update phase options for trinket comparisons
      this.updatePhaseOptions();

      // Update trinket callout visibility
      this.updateTrinketCallout();
    }

    updateTrinketCallout() {
      const trinketCallout = document.getElementById("trinket-callout");
      if (!trinketCallout) return;

      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");
      const currentComparisonType = this.getCurrentComparisonType();

      // Only show trinket callout for trinket comparisons (not for benchmarks or race comparisons)
      if (!isBenchmarksMode && currentComparisonType === "trinket") {
        trinketCallout.classList.remove("hidden");
      } else {
        trinketCallout.classList.add("hidden");
      }
    }

    updatePhaseOptions() {
      const phaseSelect = document.getElementById("phase");
      if (!phaseSelect) return;

      // const currentComparisonType = this.getCurrentComparisonType();
      const preRaidOption = phaseSelect.querySelector(
        'option[value="preRaid"]',
      );

      if (preRaidOption) {
        const showPreRaid =
          this.isRankingsMode ||
          (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");
        if (!showPreRaid) {
          // Hide preRaid option for comparison modes and force p1
          preRaidOption.style.display = "none";
          if (phaseSelect.value === "preRaid") {
            phaseSelect.value = "p1";
          }
        } else {
          // Show preRaid option for rankings/benchmarks mode
          preRaidOption.style.display = "";
        }
      }
    }

    getFileName() {
      const targetCount = document.getElementById("targetCount").value;
      const encounterType = document.getElementById("encounterType").value;
      const duration = document.getElementById("duration").value;
      const phase = document.getElementById("phase").value;

      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");

      if (isBenchmarksMode) {
        return `dps_${phase}_${encounterType}_${targetCount}_${duration}.json`;
      } else {
        const currentClass = this.getCurrentClass();
        const currentSpec = this.getCurrentSpec();
        const currentComparisonType = this.getCurrentComparisonType();
        return `${currentClass}_${currentSpec}_${currentComparisonType}_${phase}_${encounterType}_${targetCount}_${duration}.json`;
      }
    }

    getDataPath() {
      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");

      if (isBenchmarksMode) {
        return `/data/rankings/`;
      } else {
        const currentClass = this.getCurrentClass();
        const currentSpec = this.getCurrentSpec();
        const currentComparisonType = this.getCurrentComparisonType();

        // Handle different comparison type paths
        if (currentComparisonType === "trinket") {
          return `/data/comparison/trinkets/${currentClass}/${currentSpec}/`;
        } else {
          return `/data/comparison/race/${currentClass}/${currentSpec}/`;
        }
      }
    }

    setupUnifiedMode() {
      this.updateControlVisibility();
    }

    updateControlVisibility() {
      const comparisonControlsRow = document.getElementById(
        "comparison-controls-row",
      );

      if (this.currentSimulationMode === "benchmarks") {
        // Hide the entire comparison controls row for benchmarks
        if (comparisonControlsRow) {
          comparisonControlsRow.style.display = "none";
        }
      } else {
        // Show comparison controls row for comparisons
        if (comparisonControlsRow) {
          comparisonControlsRow.style.display = "flex";
        }
      }

      // Update sort options and other dependent controls
      this.updateSortOptions();
    }

    async loadInitialData() {
      if (
        this.isRankingsMode ||
        this.isFixedClassSpec ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks")
      ) {
        await this.loadData();
      }
      // for dynamic comparison mode, wait for user to select class/spec
    }

    async loadData() {
      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");

      if (!isBenchmarksMode) {
        const currentClass = this.getCurrentClass();
        const currentSpec = this.getCurrentSpec();

        if (!currentClass || !currentSpec) {
          console.log("Class or spec not selected");
          return;
        }
      }

      // Update sort options based on comparison type
      this.updateSortOptions();

      const loadingEl = document.getElementById("loading");
      const errorEl = document.getElementById("error");
      const metadataContainer = document.getElementById("metadata-container");
      const chartContainer = document.getElementById("chart-container");

      if (!loadingEl || !errorEl || !metadataContainer || !chartContainer) {
        console.error("Required elements not found");
        return;
      }

      const scrollY = window.scrollY;

      metadataContainer.style.opacity = "0.5";
      chartContainer.style.opacity = "0.5";
      loadingEl.classList.remove("hidden");
      errorEl.classList.add("hidden");

      try {
        const fileName = this.getFileName();
        const dataPath = this.getDataPath();
        const fullPath = `${dataPath}${fileName}`;

        console.log("Loading:", fullPath);
        const response = await fetch(fullPath);

        if (!response.ok) {
          throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
        }

        this.currentData = await response.json();
        console.log("Data loaded successfully");

        loadingEl.classList.add("hidden");

        this.renderMetadata();
        this.renderChart();

        metadataContainer.style.opacity = "1";
        chartContainer.style.opacity = "1";

        requestAnimationFrame(() => {
          window.scrollTo(0, scrollY);
        });
      } catch (error) {
        console.error("Error loading data:", error);
        loadingEl.classList.add("hidden");
        errorEl.textContent = `Error loading data: ${error.message}`;
        errorEl.classList.remove("hidden");

        metadataContainer.innerHTML = "";
        chartContainer.innerHTML = "";
        metadataContainer.style.opacity = "1";
        chartContainer.style.opacity = "1";
      }
    }

    clearData() {
      this.currentData = null;
      const metadataContainer = document.getElementById("metadata-container");
      const chartContainer = document.getElementById("chart-container");
      if (metadataContainer) metadataContainer.innerHTML = "";
      if (chartContainer) chartContainer.innerHTML = "";
    }

    calculateBarWidth(itemValue, maxValue, minValue) {
      // Dynamic scaling based on data spread
      const minBarWidth = 15;
      const maxBarWidth = 100;

      // Calculate how spread out the data is
      const valueRange = maxValue - minValue;
      const averageValue = (maxValue + minValue) / 2;
      const coefficientOfVariation = valueRange / averageValue;

      // Dynamic scaling weight based on data spread
      // If data is close together (low variation), emphasize differences more
      // If data is spread out (high variation), use more zero-based scaling
      const rangeWeight = Math.min(coefficientOfVariation * 1.5, 0.8); // Cap at 80% range-based
      const zeroWeight = 1 - rangeWeight;

      // Calculate both scaling approaches
      const zeroBasedPercentage = itemValue / maxValue;
      const rangeBasedPercentage =
        valueRange > 0 ? (itemValue - minValue) / valueRange : 1;

      // Blend the two approaches
      const hybridPercentage =
        zeroBasedPercentage * zeroWeight + rangeBasedPercentage * rangeWeight;

      // Apply minimum visibility threshold
      const minThreshold = minBarWidth / maxBarWidth;
      const finalPercentage = Math.max(hybridPercentage, minThreshold);
      return finalPercentage * maxBarWidth;
    }

    sortChartItems(chartItems, sortBy) {
      // Sort based on selected metric
      if (sortBy === "stdev") {
        chartItems.sort((a, b) => a.value - b.value); // Lower stdev is better
      } else if (sortBy === "percent") {
        chartItems.sort(
          (a, b) => (b.percentIncrease || 0) - (a.percentIncrease || 0),
        ); // Higher percentage is better
      } else {
        chartItems.sort((a, b) => b.value - a.value); // Higher values are better
      }
      return chartItems;
    }

    renderMetadata() {
      if (!this.currentData) return;

      const container = document.getElementById("metadata-container");
      const metadata = this.currentData.metadata;

      if (this.isRankingsMode) {
        this.renderRankingsMetadata(container, metadata);
      } else {
        this.renderComparisonMetadata(container, metadata);
      }
    }

    renderRankingsMetadata(container, metadata) {
      const simulationDate = formatSimulationDate(metadata.timestamp);

      container.innerHTML = `
      <div class="card">
        <h3 class="card-title">DPS Rankings Details</h3>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">Simulation Type</span>
            <span class="info-value">DPS Rankings</span>
          </div>
          <div class="info-item">
            <span class="info-label">Iterations</span>
            <span class="info-value">${metadata.iterations.toLocaleString()}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Specs Tested</span>
            <span class="info-value">${metadata.specCount}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Encounter Duration</span>
            <span class="info-value">${formatDuration(metadata.encounterDuration)}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Duration Variation</span>
            <span class="info-value">±${metadata.encounterVariation}s</span>
          </div>
          <div class="info-item">
            <span class="info-label">Target Count</span>
            <span class="info-value">${metadata.targetCount}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Date Simulated</span>
            <span class="info-value">${simulationDate}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Simulation Hash</span>
            <span class="info-value">
              <a href="https://github.com/wowsims/mop/commit/${metadata.wowsimsCommit}" target="_blank" rel="noopener noreferrer">${metadata.wowsimsCommit}</a>
            </span>
          </div>
          <div class="info-item info-item-wide">
            <span class="info-label">Active Raid Buffs</span>
            <div class="callout-note">
              <svg class="callout-note-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span class="callout-note-content">Warrior and Shaman simulations are counted within the Skull Banner/Stormlash Totem totals, not as additional buffs.</span>
            </div>
            <span class="info-value info-value-wrap">${formatRaidBuffs(metadata.raidBuffs)}</span>
          </div>
        </div>
      </div>
    `;
    }

    renderComparisonMetadata(container, metadata) {
      const currentComparisonType = this.getCurrentComparisonType();
      const comparisonLabel =
        currentComparisonType === "race" ? "Race" : "Trinket";
      const simulationDate = formatSimulationDate(metadata.timestamp);
      const itemCount = Object.keys(this.currentData.results).length;

      container.innerHTML = `
      <div class="card">
        <h3 class="card-title">${comparisonLabel} Comparison Details</h3>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">Class/Spec</span>
            <span class="info-value">${metadata.class || this.getCurrentClass()} ${metadata.spec || this.getCurrentSpec()}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Comparison Type</span>
            <span class="info-value">${comparisonLabel}s</span>
          </div>
          <div class="info-item">
            <span class="info-label">Iterations</span>
            <span class="info-value">${metadata.iterations.toLocaleString()}</span>
          </div>
          <div class="info-item">
            <span class="info-label">${comparisonLabel}s Tested</span>
            <span class="info-value">${itemCount}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Encounter Duration</span>
            <span class="info-value">${formatDuration(metadata.encounterDuration)}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Duration Variation</span>
            <span class="info-value">±${metadata.encounterVariation}s</span>
          </div>
          <div class="info-item">
            <span class="info-label">Target Count</span>
            <span class="info-value">${metadata.targetCount}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Date Simulated</span>
            <span class="info-value">${simulationDate}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Simulation Hash</span>
            <span class="info-value">
              <a href="https://github.com/wowsims/mop/commit/${metadata.wowsimsCommit}" target="_blank" rel="noopener noreferrer">${metadata.wowsimsCommit}</a>
            </span>
          </div>
          <div class="info-item info-item-wide">
            <span class="info-label">Active Raid Buffs</span>
            <span class="info-value info-value-wrap">${formatRaidBuffs(metadata.raidBuffs)}</span>
          </div>
        </div>
      </div>
    `;
    }

    renderChart() {
      if (!this.currentData) return;

      const container = document.getElementById("chart-container");
      const sortBy = document.getElementById("sortBy").value;

      const isBenchmarksMode =
        this.isRankingsMode ||
        (this.isUnifiedMode && this.currentSimulationMode === "benchmarks");

      if (isBenchmarksMode) {
        this.renderRankingsChart(container, sortBy);
      } else {
        this.renderComparisonChart(container, sortBy);
      }
    }

    renderRankingsChart(container, sortBy) {
      const chartItems = [];
      // convert rankings data structure

      for (const [className, classData] of Object.entries(
        this.currentData.results,
      )) {
        for (const [specName, specData] of Object.entries(classData)) {
          chartItems.push({
            label: specName,
            dps: specData.dps,
            max: specData.max,
            min: specData.min,
            stdev: specData.stdev,
            value: specData[sortBy] || specData.dps,
            category: className,
            className: className,
            specName: specName,
            loadout: specData.loadout || null,
          });
        }
      }

      this.sortChartItems(chartItems, sortBy);

      const maxDps = Math.max(...chartItems.map((item) => item.value));
      const minDps = Math.min(...chartItems.map((item) => item.value));

      const chartTitles = {
        dps: "DPS Rankings (Average)",
        max: "DPS Rankings (Maximum)",
        min: "DPS Rankings (Minimum)",
        stdev: "DPS Consistency Rankings (Low StdDev = More Consistent)",
      };

      container.innerHTML = `
      <div class="card">
        <h2 class="card-title-large">${chartTitles[sortBy] || "DPS Rankings"}</h2>
        <div class="chart-bars">
          ${chartItems
            .map((item, index) => {
              const barWidth = this.calculateBarWidth(
                item.value,
                maxDps,
                minDps,
              );

              const displayValue = Math.round(item.value).toLocaleString();
              const avgDps = Math.round(item.dps).toLocaleString();
              const tooltip =
                sortBy === "dps"
                  ? displayValue
                  : `${displayValue} (Avg: ${avgDps})`;
              const chartDisplayValue = displayValue; // Rankings always show raw values

              const barColor = classColors[item.className] || "#666";
              const dropdownContent = item.loadout
                ? generateLoadoutDropdown(item.loadout, this.currentData)
                : "";

              return `
            <div class="chart-item-wrapper">
              <div class="chart-item" onclick="toggleChartItem(this.parentElement)">
                <div class="chart-labels">
                  <span class="chart-rank">#${index + 1}</span>
                  <span class="chart-label">${item.label}</span>
                  ${item.loadout ? '<span class="chart-expand-icon">▶</span>' : ""}
                </div>
                <div class="chart-bar-container">
                  <div class="chart-bar-track">
                    <div class="chart-bar" style="width: ${barWidth}%; background-color: ${barColor};"></div>
                  </div>
                  <span class="chart-value" title="${tooltip}">${chartDisplayValue}</span>
                </div>
              </div>
              ${dropdownContent}
            </div>
            `;
            })
            .join("")}
        </div>
      </div>
    `;
    }

    renderComparisonChart(container, sortBy) {
      const currentComparisonType = this.getCurrentComparisonType();
      const comparisonLabel =
        currentComparisonType === "race" ? "Race" : "Trinket";

      const chartItems = [];

      for (const [itemName, itemData] of Object.entries(
        this.currentData.results,
      )) {
        let label = itemName.replace(/_/g, " ");
        let iconData = null;

        // For trinket comparisons, extract icon and ilvl from equipment
        if (
          currentComparisonType === "trinket" &&
          itemData.loadout &&
          itemData.loadout.equipment
        ) {
          const trinket1 = itemData.loadout.equipment.items[12]; // Trinket slot 1
          const trinket2 = itemData.loadout.equipment.items[13]; // Trinket slot 2

          // Find the trinket that's not empty (baseline has no trinkets)
          const trinket =
            trinket1 && trinket1.id
              ? trinket1
              : trinket2 && trinket2.id
                ? trinket2
                : null;

          if (trinket) {
            const ilvl =
              trinket.stats && trinket.stats.ilvl ? trinket.stats.ilvl : "";
            iconData = {
              icon: trinket.icon,
              ilvl: ilvl,
              name: trinket.name,
            };
            label = ilvl ? `${ilvl}` : trinket.name; // Show just ilvl if available
          }
        }

        chartItems.push({
          label: label,
          dps: itemData.dps,
          max: itemData.max,
          min: itemData.min,
          stdev: itemData.stdev,
          value: itemData[sortBy] || itemData.dps,
          category: itemName,
          rawName: itemName,
          loadout: itemData.loadout || null,
          iconData: iconData,
        });
      }

      // Find baseline for percentage calculations (only for trinket comparisons)
      const baseline = this.currentData.results.baseline;
      const baselineDps = baseline ? baseline.dps : null;

      // Add percentage and DPS increase calculations to chart items
      if (baselineDps && currentComparisonType === "trinket") {
        chartItems.forEach((item) => {
          if (item.rawName === "baseline") {
            item.percentIncrease = 0;
            item.dpsIncrease = 0;
          } else {
            // Calculate increases based on sort type (compare like-to-like)
            const baselineValue = baseline[sortBy] || baseline.dps;
            const itemValue = item[sortBy] || item.dps;

            item.percentIncrease =
              ((itemValue - baselineValue) / baselineValue) * 100;
            item.dpsIncrease = itemValue - baselineValue;
          }
        });
      }

      this.sortChartItems(chartItems, sortBy);

      const maxDps = Math.max(...chartItems.map((item) => item.value));
      const minDps = Math.min(...chartItems.map((item) => item.value));

      const chartTitles = {
        dps: `${comparisonLabel} DPS Rankings (Average)`,
        max: `${comparisonLabel} DPS Rankings (Maximum)`,
        min: `${comparisonLabel} DPS Rankings (Minimum)`,
        stdev: `${comparisonLabel} DPS Consistency Rankings (Low StdDev = More Consistent)`,
        percent: `${comparisonLabel} Performance Rankings (% Increase)`,
      };

      // TODO: abstract
      container.innerHTML = `
      <div class="card">
        <h2 class="card-title-large">${chartTitles[sortBy] || `${comparisonLabel} DPS Rankings`}</h2>
        <div class="chart-bars">
          ${chartItems
            .map((item, index) => {
              const barWidth = this.calculateBarWidth(
                item.value,
                maxDps,
                minDps,
              );

              const displayValue = Math.round(item.value).toLocaleString();
              const avgDps = Math.round(item.dps).toLocaleString();

              // Format tooltip and display value based on sort type and comparison type
              let tooltip, chartDisplayValue;
              if (
                currentComparisonType === "trinket" &&
                item.percentIncrease !== undefined
              ) {
                const percentDisplay =
                  item.percentIncrease === 0
                    ? "Baseline"
                    : `+${item.percentIncrease.toFixed(1)}%`;
                const dpsIncreaseDisplay =
                  item.dpsIncrease === 0
                    ? "Baseline"
                    : `+${Math.round(item.dpsIncrease).toLocaleString()}`;

                if (sortBy === "percent") {
                  // When sorting by percentage, show percentage as main value
                  chartDisplayValue = percentDisplay;
                  tooltip = `${percentDisplay} (${avgDps} DPS)`;
                } else if (sortBy === "stdev") {
                  // For consistency sort, show actual standard deviation
                  chartDisplayValue = Math.round(item.stdev).toLocaleString();
                  tooltip = `${chartDisplayValue} StdDev (${avgDps} DPS avg, lower is more consistent)`;
                } else {
                  // For DPS-based sorts, show DPS increase from baseline
                  chartDisplayValue = dpsIncreaseDisplay;
                  tooltip =
                    sortBy === "dps"
                      ? `${displayValue} DPS (${dpsIncreaseDisplay} vs baseline)`
                      : `${displayValue} (Avg: ${avgDps} DPS, ${dpsIncreaseDisplay} vs baseline)`;
                }
              } else {
                chartDisplayValue = displayValue;
                tooltip =
                  sortBy === "dps"
                    ? displayValue
                    : `${displayValue} (Avg: ${avgDps})`;
              }

              const barColor = classColors[this.getCurrentClass()] || "#666";
              const dropdownContent = item.loadout
                ? generateLoadoutDropdown(item.loadout, this.currentData)
                : "";

              // Create the label content with icon if trinket
              const labelContent = item.iconData
                ? `<img src="https://wow.zamimg.com/images/wow/icons/small/${item.iconData.icon}.jpg" alt="${item.iconData.name}" class="trinket-icon" title="${item.iconData.name}" /><span class="trinket-ilvl">${item.label}</span>`
                : `<span class="chart-label">${item.label}</span>`;

              return `
            <div class="chart-item-wrapper">
              <div class="chart-item" onclick="toggleChartItem(this.parentElement)">
                <div class="chart-labels">
                  <span class="chart-rank">#${index + 1}</span>
                  ${labelContent}
                  ${item.loadout ? '<span class="chart-expand-icon">▶</span>' : ""}
                </div>
                <div class="chart-bar-container">
                  <div class="chart-bar-track">
                    <div class="chart-bar" style="width: ${barWidth}%; background-color: ${barColor};"></div>
                  </div>
                  <span class="chart-value" title="${tooltip}">${chartDisplayValue}</span>
                </div>
              </div>
              ${dropdownContent}
            </div>
            `;
            })
            .join("")}
        </div>
      </div>
    `;
    }
  }

  // Initialize when DOM is loaded
  document.addEventListener("DOMContentLoaded", () => {
    console.log("DOM loaded, initializing UnifiedChart...");
    new UnifiedChart();
  });
}
