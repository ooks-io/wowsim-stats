import { formatDuration, formatTimestamp } from "./leaderboard-utils.js";
import { filterUniqueTeams } from "./leaderboard-filters.js";

let originalLeaderboardData = null;

export function showNoData() {
  document.getElementById("loading").classList.add("hidden");
  document.getElementById("error").classList.add("hidden");
  document.getElementById("no-data").classList.remove("hidden");
  document.getElementById("content").classList.add("hidden");
}

export function showError(message) {
  const errorDiv = document.getElementById("error");
  const loadingMessage = errorDiv.querySelector(".loading-message");
  if (loadingMessage) {
    loadingMessage.textContent = message;
  } else {
    errorDiv.innerHTML =
      '<div class="loading-state error"><p class="loading-message">' +
      message +
      "</p></div>";
  }
  errorDiv.classList.remove("hidden");
  document.getElementById("loading").classList.add("hidden");
  document.getElementById("no-data").classList.add("hidden");
  document.getElementById("content").classList.add("hidden");
}

export function showLoading() {
  document.getElementById("loading").classList.remove("hidden");
  document.getElementById("error").classList.add("hidden");
  document.getElementById("no-data").classList.add("hidden");
  document.getElementById("content").classList.add("content-loading");
}

export function hideLoading() {
  document.getElementById("loading").classList.add("hidden");
  document.getElementById("error").classList.add("hidden");
  document.getElementById("no-data").classList.add("hidden");
  document.getElementById("content").classList.remove("content-loading");
  document.getElementById("content").classList.remove("hidden");
}

export function createLeaderboardRow(group) {
  console.log("Creating row for group:", group);

  const row = document.createElement("div");
  row.className = "leaderboard-row";

  const currentRegion = document.getElementById("region").value;
  const currentRealm = document.getElementById("realm").value;
  const isIndividualRealmView =
    currentRegion &&
    currentRealm &&
    currentRealm !== "all" &&
    currentRegion !== "global";

  let teamHtml = "";
  group.members.forEach((member) => {
    const specId = member.spec_id || member.specialization?.id;
    const spec = specId
      ? window.WoW.getSpecInfo(specId)
      : {
          role: "dps",
          class: "Unknown",
          spec: "Unknown",
        };
    const iconUrl = window.WoW.getSpecIcon(spec.class, spec.spec);
    const iconHtml = iconUrl
      ? '<img src="' +
        iconUrl +
        '" alt="' +
        spec.spec +
        " " +
        spec.class +
        '" style="width: 16px; height: 16px; border-radius: 2px; margin-right: 4px; vertical-align: middle; flex-shrink: 0;">'
      : "";

    let playerName = member.name || member.profile?.name;

    const memberRealmSlug = member.realm_slug || member.profile?.realm?.slug;
    if (
      isIndividualRealmView &&
      memberRealmSlug &&
      memberRealmSlug !== currentRealm
    ) {
      playerName +=
        '<span style="color: #ff6b6b; font-weight: bold; margin-left: 2px;">*</span>';
    }

    teamHtml +=
      '<span style="display: inline-flex; align-items: center; margin-right: 8px; gap: 4px;">' +
      iconHtml +
      '<span style="color: ' +
      window.WoW.getClassColor(spec.class) +
      '; font-weight: 600; font-size: 0.9em;">' +
      playerName +
      "</span>" +
      "</span>";
  });

  row.innerHTML =
    '<div style="display: table-cell; padding: 15px 10px; vertical-align: middle; font-size: 1.2em; font-weight: bold; color: #d8a657; text-align: center; width: 80px;">#' +
    group.ranking +
    "</div>" +
    "<div style=\"display: table-cell; padding: 15px 10px; vertical-align: middle; font-family: 'Courier New', monospace; font-weight: bold; color: #ffffff; text-align: right; width: 120px;\">" +
    formatDuration(group.duration) +
    "</div>" +
    '<div style="display: table-cell; padding: 15px 10px; vertical-align: middle;">' +
    teamHtml +
    "</div>" +
    '<div style="display: table-cell; padding: 15px 10px; vertical-align: middle; color: #aaaaaa; font-size: 0.9em; text-align: center; width: 140px;">' +
    formatTimestamp(group.completed_timestamp) +
    "</div>";

  row.style.cssText =
    "display: table-row !important; border-bottom: 1px solid #4a4a4a !important; background-color: #32302f; transition: background-color 0.2s ease;";

  row.addEventListener("mouseenter", () => {
    row.style.backgroundColor = "rgba(255, 215, 0, 0.05)";
  });
  row.addEventListener("mouseleave", () => {
    row.style.backgroundColor = "#32302f";
  });

  console.log("Row created:", row);
  return row;
}

export function displayLeaderboard(data, DATA_MAP) {
  console.log("Displaying leaderboard data:", data);

  let leaderboardData = data.leaderboard || data.leading_groups;

  if (!leaderboardData || leaderboardData.length === 0) {
    showNoData();
    return;
  }

  originalLeaderboardData = [...leaderboardData];

  leaderboardData = filterUniqueTeams(leaderboardData);

  updateInfoDisplay(data, DATA_MAP);

  const rowsContainer = document.getElementById("leaderboard-rows");
  rowsContainer.innerHTML = "";

  console.log("Creating rows for", leaderboardData.length, "groups");
  leaderboardData.forEach((group, index) => {
    console.log("Creating row for group", index + 1);
    const displayGroup = { ...group, ranking: index + 1 };
    const row = createLeaderboardRow(displayGroup);
    rowsContainer.appendChild(row);
  });

  hideLoading();
  console.log("Leaderboard displayed successfully");
}

export function refreshLeaderboardDisplay() {
  if (!originalLeaderboardData) {
    console.log("No original data to refresh from");
    return;
  }

  let leaderboardData = filterUniqueTeams(originalLeaderboardData);

  const rowsContainer = document.getElementById("leaderboard-rows");
  rowsContainer.innerHTML = "";

  console.log("Refreshing display with", leaderboardData.length, "groups");
  leaderboardData.forEach((group, index) => {
    const displayGroup = { ...group, ranking: index + 1 };
    const row = createLeaderboardRow(displayGroup);
    rowsContainer.appendChild(row);
  });
}

function updateInfoDisplay(data, DATA_MAP) {
  let dungeonName;
  if (data.dungeon_name) {
    dungeonName = data.dungeon_name;
  } else if (data.map) {
    dungeonName =
      typeof data.map.name === "object"
        ? data.map.name.en_US || data.map.name
        : data.map.name;
  }
  document.getElementById("dungeon-name").textContent =
    dungeonName || "Unknown Dungeon";

  if (data.period) {
    document.getElementById("period").textContent = data.period;
    document.getElementById("period-start").textContent = formatTimestamp(
      data.period_start_timestamp,
    );
    document.getElementById("period-end").textContent = formatTimestamp(
      data.period_end_timestamp,
    );
  } else {
    document.getElementById("period").textContent = "Multiple Periods";
    document.getElementById("period-start").textContent = "-";
    document.getElementById("period-end").textContent = "-";
  }

  const currentRegion = document.getElementById("region").value;
  const currentRealm = document.getElementById("realm").value;
  let realmDisplayName;

  if (currentRegion === "global") {
    realmDisplayName = "All Regions";
  } else if (currentRealm === "all") {
    realmDisplayName = "All " + currentRegion.toUpperCase() + " Realms";
  } else {
    const regionData = DATA_MAP[currentRegion];
    realmDisplayName = regionData?.realms?.[currentRealm] || currentRealm;
  }
  document.getElementById("realm-name").textContent = realmDisplayName;

  const crossRealmNote = document.getElementById("cross-realm-note");
  const isIndividualRealmView =
    currentRegion &&
    currentRealm &&
    currentRealm !== "all" &&
    currentRegion !== "global";
  if (crossRealmNote) {
    crossRealmNote.style.display = isIndividualRealmView ? "block" : "none";
  }
}

export function displayTeamPlayerLeaderboard(data, mode) {
  console.log(`Displaying ${mode} leaderboard data:`, data);

  if (!data.leaderboard || data.leaderboard.length === 0) {
    showNoData();
    return;
  }

  const dungeonNameElement = document.getElementById("dungeon-name");
  if (dungeonNameElement) {
    dungeonNameElement.textContent = data.title;
  }

  const periodElement = document.getElementById("period");
  if (periodElement) {
    periodElement.textContent = data.generated_timestamp
      ? formatTimestamp(data.generated_timestamp)
      : "Unknown";
  }

  const rowsContainer = document.getElementById("team-rows");
  rowsContainer.innerHTML = "";

  data.leaderboard.forEach((item, index) => {
    item.ranking = index + 1;
    const wrapper =
      mode === "players" ? createPlayerRow(item) : createTeamRow(item);
    rowsContainer.appendChild(wrapper);
  });

  hideLoading();
}

function getSpecColor(specId) {
  const specColors = {
    250: "#C41F3B",
    251: "#C41F3B",
    252: "#C41F3B",
    102: "#FF7D0A",
    103: "#FF7D0A",
    104: "#FF7D0A",
    105: "#FF7D0A",
    253: "#ABD473",
    254: "#ABD473",
    255: "#ABD473",
    62: "#69CCF0",
    63: "#69CCF0",
    64: "#69CCF0",
    268: "#00FF96",
    269: "#00FF96",
    270: "#00FF96",
    65: "#F58CBA",
    66: "#F58CBA",
    70: "#F58CBA",
    256: "#FFFFFF",
    257: "#FFFFFF",
    258: "#FFFFFF",
    259: "#FFF569",
    260: "#FFF569",
    261: "#FFF569",
    262: "#0070DE",
    263: "#0070DE",
    264: "#0070DE",
    265: "#9482C9",
    266: "#9482C9",
    267: "#9482C9",
    71: "#C79C6E",
    72: "#C79C6E",
    73: "#C79C6E",
  };
  return specColors[specId] || "#FFFFFF";
}

export function createPlayerRow(player) {
  const wrapper = document.createElement("div");
  wrapper.className = "chart-item-wrapper";

  const spec = player.main_spec_id
    ? window.WoW.getSpecInfo(player.main_spec_id)
    : null;
  const iconUrl = spec ? window.WoW.getSpecIcon(spec.class, spec.spec) : null;
  const classColor = spec ? window.WoW.getClassColor(spec.class) : "#FFFFFF";

  const iconHtml = iconUrl
    ? `<img src="${iconUrl}" alt="${spec.spec} ${spec.class}" style="width: 20px; height: 20px; border-radius: 2px; margin-right: 8px; vertical-align: middle; flex-shrink: 0;">`
    : "";

  const playerDetails = createPlayerDetailsContent(player);

  wrapper.innerHTML = `
    <div class="leaderboard-row chart-item-header" onclick="toggleChartItem(this.parentElement)" style="display: table-row !important; border-bottom: 1px solid #4a4a4a !important; transition: background-color 0.2s ease !important; cursor: pointer;">
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; font-size: 1.2em; font-weight: bold; color: #d8a657; text-align: center; width: 80px;">#${player.ranking}</div>
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; font-family: 'Courier New', monospace; font-weight: bold; color: #ffffff; text-align: right; width: 120px; white-space: nowrap;">${formatDuration(player.combined_best_time)}</div>
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; width: 100%;">
        <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
          <div style="display: flex; align-items: center;">
            <span class="chart-expand-icon" style="margin-right: 8px;">▶</span>
            ${iconHtml}
            <span style="color: ${classColor}; font-weight: 600; font-size: 1.1em;">${player.name}</span>
          </div>
          <span style="color: var(--text-secondary); font-size: 0.9em;">@${player.realm_slug}</span>
        </div>
      </div>
    </div>
    <div class="chart-dropdown">
      ${playerDetails}
    </div>
  `;

  return wrapper;
}

export function createTeamRow(team) {
  const wrapper = document.createElement("div");
  wrapper.className = "chart-item-wrapper";

  const extendedRoster = team.extended_roster
    .map((member) => {
      const specColor = member.spec_id
        ? getSpecColor(member.spec_id)
        : "#FFFFFF";
      return `<span style="display: inline-flex; align-items: center; margin-right: 8px; gap: 4px;">
      <span style="color: ${specColor}; font-weight: 600; font-size: 0.9em;">${member.name}</span>
      <span style="color: var(--text-secondary); font-size: 0.8em;">@${member.realm_slug}</span>
    </span>`;
    })
    .join("");

  const teamDetails = createTeamDetailsContent(team);

  wrapper.innerHTML = `
    <div class="leaderboard-row chart-item-header" onclick="toggleChartItem(this.parentElement)" style="display: table-row !important; border-bottom: 1px solid #4a4a4a !important; transition: background-color 0.2s ease !important; cursor: pointer;">
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; font-size: 1.2em; font-weight: bold; color: #d8a657; text-align: center; width: 80px;">#${team.ranking}</div>
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; font-family: 'Courier New', monospace; font-weight: bold; color: #ffffff; text-align: right; width: 120px;">${formatDuration(team.combined_best_time)}</div>
      <div style="display: table-cell; padding: 15px 10px; vertical-align: middle; display: flex; align-items: center; overflow: hidden;">
        <span class="chart-expand-icon" style="margin-right: 8px; flex-shrink: 0;">▶</span>
        <div style="display: flex; flex-wrap: wrap; align-items: center; overflow: hidden; min-width: 0;">${extendedRoster}</div>
      </div>
    </div>
    <div class="chart-dropdown">
      ${teamDetails}
    </div>
  `;

  return wrapper;
}

function createPlayerDetailsContent(player) {
  const dungeonEntries = Object.entries(player.best_runs_per_dungeon || {})
    .map(([_dungeonSlug, run]) => {
      const teamComposition = (run.all_members || [])
        .map((member) => {
          const spec = member.spec_id
            ? window.WoW.getSpecInfo(member.spec_id)
            : null;
          const iconUrl = spec
            ? window.WoW.getSpecIcon(spec.class, spec.spec)
            : null;
          const classColor = spec
            ? window.WoW.getClassColor(spec.class)
            : "#FFFFFF";

          const iconHtml = iconUrl
            ? `<img src="${iconUrl}" alt="${spec.spec} ${spec.class}" style="width: 16px; height: 16px; border-radius: 2px; margin-right: 4px; vertical-align: middle; flex-shrink: 0;">`
            : "";

          return `<span style="display: inline-flex; align-items: center; margin-right: 8px; gap: 4px;">
        ${iconHtml}
        <span style="color: ${classColor}; font-weight: 600; font-size: 0.9em;">${member.name}@${member.realm_slug}</span>
      </span>`;
        })
        .join("");

      return `
      <div style="display: grid; grid-template-columns: 80px 120px 2fr 100px 140px; gap: 20px; align-items: center; padding: 15px 20px; border-bottom: 1px solid #3a3a3a; min-height: 60px;">
        <div style="font-size: 1.2em; font-weight: bold; color: #d8a657; text-align: center;">${run.ranking === "~" ? "~" : "#" + run.ranking}</div>
        <div style="font-family: 'Courier New', monospace; font-weight: bold; color: #ffffff; text-align: right;">${formatDuration(run.duration)}</div>
        <div style="display: flex; flex-wrap: wrap; gap: 6px; justify-content: flex-start; align-items: center;">${teamComposition}</div>
        <div style="color: var(--text-primary); text-align: center; font-weight: 600;">${run.dungeon_name}</div>
        <div style="color: #aaaaaa; font-size: 0.9em; text-align: center;">${formatTimestamp(run.completed_timestamp)}</div>
      </div>
    `;
    })
    .join("");

  return `
    <div>
      <h4 style="color: var(--highlight-color); margin-bottom: 10px;">Best Times Per Dungeon</h4>
      <div style="border-radius: 6px; overflow: hidden; border: 1px solid #4a4a4a;">
        <div style="display: grid; grid-template-columns: 80px 120px 2fr 100px 140px; gap: 20px; padding: 15px 20px; background-color: rgba(255, 255, 255, 0.03); border-bottom: 1px solid #4a4a4a; font-weight: 600; color: var(--text-secondary); font-size: 0.9em; text-transform: uppercase; letter-spacing: 0.5px;">
          <div>Global Rank</div>
          <div style="text-align: right;">Time</div>
          <div>Team Composition</div>
          <div style="text-align: center;">Dungeon</div>
          <div style="text-align: center;">Date</div>
        </div>
        ${dungeonEntries}
      </div>
    </div>
  `;
}

function createMemberLookup(team) {
  const memberLookup = {};

  team.all_runs.forEach((run) => {
    run.members.forEach((member) => {
      if (!memberLookup[member.name]) {
        memberLookup[member.name] = {
          name: member.name,
          realm_slug: member.realm_slug,
          spec_id: member.spec_id,
          id: member.id,
        };
      }
    });
  });

  return memberLookup;
}

function createTeamComposition(memberNames, memberLookup) {
  return memberNames
    .map((memberName) => {
      const member = memberLookup[memberName];
      if (!member) {
        return `<span style="color: var(--text-primary); margin-right: 8px;">${memberName}</span>`;
      }

      const spec = member.spec_id
        ? window.WoW.getSpecInfo(member.spec_id)
        : null;
      const iconUrl = spec
        ? window.WoW.getSpecIcon(spec.class, spec.spec)
        : null;
      const classColor = spec
        ? window.WoW.getClassColor(spec.class)
        : "#FFFFFF";

      const iconHtml = iconUrl
        ? `<img src="${iconUrl}" alt="${spec.spec} ${spec.class}" style="width: 16px; height: 16px; border-radius: 2px; margin-right: 4px; vertical-align: middle; flex-shrink: 0;">`
        : "";

      return `<span style="display: inline-flex; align-items: center; margin-right: 8px; gap: 4px;">
      ${iconHtml}
      <span style="color: ${classColor}; font-weight: 600; font-size: 0.9em;">${member.name}@${member.realm_slug}</span>
    </span>`;
    })
    .join("");
}

function createTeamDetailsContent(team) {
  const memberLookup = createMemberLookup(team);

  const coreMembers = team.core_members
    .map((member) => {
      const spec = member.spec_id
        ? window.WoW.getSpecInfo(member.spec_id)
        : null;
      const iconUrl = spec
        ? window.WoW.getSpecIcon(spec.class, spec.spec)
        : null;
      const classColor = spec
        ? window.WoW.getClassColor(spec.class)
        : "#FFFFFF";

      const iconHtml = iconUrl
        ? `<img src="${iconUrl}" alt="${spec.spec} ${spec.class}" style="width: 16px; height: 16px; border-radius: 2px; margin-right: 4px; vertical-align: middle; flex-shrink: 0;">`
        : "";

      return `<span style="display: inline-flex; align-items: center; margin-right: 15px; gap: 4px;">
      ${iconHtml}
      <span style="color: ${classColor}; font-weight: 600; font-size: 0.9em;">${member.name}@${member.realm_slug}</span>
    </span>`;
    })
    .join("");

  const dungeonEntries = Object.entries(team.best_runs_per_dungeon || {})
    .map(([_dungeonSlug, run]) => {
      const teamComposition = createTeamComposition(run.members, memberLookup);

      return `
      <div style="display: grid; grid-template-columns: 80px 120px 2fr 100px 140px; gap: 20px; align-items: center; padding: 15px 20px; border-bottom: 1px solid #3a3a3a; min-height: 60px;">
        <div style="font-size: 1.2em; font-weight: bold; color: #d8a657; text-align: center;">#${run.ranking}</div>
        <div style="font-family: 'Courier New', monospace; font-weight: bold; color: #ffffff; text-align: right;">${formatDuration(run.duration)}</div>
        <div style="display: flex; flex-wrap: wrap; gap: 6px; justify-content: flex-start; align-items: center;">${teamComposition}</div>
        <div style="color: var(--text-primary); text-align: center; font-weight: 600;">${run.dungeon_name}</div>
        <div style="color: #aaaaaa; font-size: 0.9em; text-align: center;">${formatTimestamp(run.completed_timestamp)}</div>
      </div>
    `;
    })
    .join("");

  return `
    <div style="margin-bottom: 20px;">
      <h4 style="color: var(--highlight-color); margin-bottom: 10px;">Identified Core Team </h4>
      <div style="display: flex; flex-wrap: wrap; align-items: center;">${coreMembers}</div>
    </div>
    
    <div>
      <h4 style="color: var(--highlight-color); margin-bottom: 10px;">Best Times Per Dungeon</h4>
      <div style="border-radius: 6px; overflow: hidden; border: 1px solid #4a4a4a;">
        <div style="display: grid; grid-template-columns: 80px 120px 2fr 100px 140px; gap: 20px; padding: 15px 20px; background-color: rgba(255, 255, 255, 0.03); border-bottom: 1px solid #4a4a4a; font-weight: 600; color: var(--text-secondary); font-size: 0.9em; text-transform: uppercase; letter-spacing: 0.5px;">
          <div>Rank</div>
          <div style="text-align: right;">Time</div>
          <div>Team Composition</div>
          <div style="text-align: center;">Dungeon</div>
          <div style="text-align: center;">Date</div>
        </div>
        ${dungeonEntries}
      </div>
    </div>
  `;
}
