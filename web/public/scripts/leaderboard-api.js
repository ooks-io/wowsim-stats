export function dungeonNameToSlug(dungeonName) {
  return dungeonName.toLowerCase().replace(/[^a-z0-9]/g, "-");
}

export function getDataFileName(dungeonNames) {
  const region = document.getElementById("region").value;
  const realm = document.getElementById("realm").value;
  const dungeon = document.getElementById("dungeon").value;

  console.log("Getting data filename with:", { region, realm, dungeon });
  console.log("Dungeon name lookup:", dungeonNames[dungeon]);

  const dungeonSlug = dungeonNameToSlug(dungeonNames[dungeon]);

  let fileName;

  if (region === "global") {
    fileName = "leaderboards/global/" + dungeonSlug + "/leaderboard.json";
  } else if (realm === "all") {
    fileName =
      "leaderboards/regional/" +
      region +
      "/" +
      dungeonSlug +
      "/leaderboard.json";
  } else {
    fileName =
      "challenge-mode/" +
      region +
      "/" +
      realm +
      "/" +
      dungeonSlug +
      "/" +
      realm +
      "-" +
      dungeonSlug +
      "-leaderboard.json";
  }

  console.log("Generated filename:", fileName);
  return fileName;
}

export function canLoadLeaderboard() {
  const region = document.getElementById("region").value;
  const realm = document.getElementById("realm").value;
  const dungeon = document.getElementById("dungeon").value;

  if (region === "global") {
    return region && dungeon;
  }

  return region && realm && dungeon;
}

export async function loadLeaderboard(dungeonNames) {
  if (!canLoadLeaderboard()) {
    console.log("Cannot load leaderboard - missing selections");
    return;
  }

  try {
    const fileName = getDataFileName(dungeonNames);
    console.log("Loading file:", fileName);
    const fullUrl = "/data/" + fileName;
    console.log("Full fetch URL:", fullUrl);
    const response = await fetch(fullUrl);

    if (!response.ok) {
      throw new Error(
        "Failed to load leaderboard data: " + response.statusText,
      );
    }

    const data = await response.json();
    console.log("Data loaded:", data);
    return data;
  } catch (error) {
    console.error("Error loading leaderboard:", error);
    throw error;
  }
}

export async function loadTeamPlayerLeaderboard(mode, baseUrl) {
  try {
    const dataPath =
      mode === "players" ? "player-leaderboards" : "team-leaderboards";
    const fileName = "best-overall.json";
    const fullUrl = `${baseUrl}data/${dataPath}/${fileName}`;
    console.log(`Loading ${mode} leaderboard from:`, fullUrl);

    const response = await fetch(fullUrl);
    if (!response.ok) {
      throw new Error(
        `Failed to load ${mode} leaderboard data: ` + response.statusText,
      );
    }

    const data = await response.json();
    console.log(`${mode} leaderboard data loaded:`, data);
    return data;
  } catch (error) {
    console.error(`Error loading ${mode} leaderboard:`, error);
    throw error;
  }
}

export function updateURL(dungeonNames) {
  const region = document.getElementById("region").value;
  const realm = document.getElementById("realm").value;
  const dungeon = document.getElementById("dungeon").value;

  if (!canLoadLeaderboard()) {
    return;
  }

  const dungeonName = dungeonNameToSlug(dungeonNames[dungeon]);

  let newURL;

  if (region === "global") {
    newURL = "/challenge-mode/global/" + dungeonName;
  } else if (realm === "all") {
    newURL = "/challenge-mode/" + region + "/all/" + dungeonName;
  } else {
    newURL = "/challenge-mode/" + region + "/" + realm + "/" + dungeonName;
  }

  window.history.pushState({}, "", newURL);
}

export function updateLeaderboardURL(leaderboardType) {
  let newURL;
  if (leaderboardType === "players") {
    newURL = "/challenge-mode/players";
  } else if (leaderboardType === "teams") {
    newURL = "/challenge-mode/teams";
  } else {
    newURL = "/challenge-mode/";
  }

  if (window.location.pathname !== newURL) {
    window.history.pushState({}, "", newURL);
  }
}

export function updateRealmOptions(DATA_MAP) {
  const region = document.getElementById("region").value;
  const realmSelect = document.getElementById("realm");

  console.log("=== updateRealmOptions START ===");
  console.log("Region:", region);

  realmSelect.innerHTML = "";

  if (!region) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = "Select Region First";
    option.selected = true;
    realmSelect.appendChild(option);
    realmSelect.disabled = true;
    console.log("No region selected, showing placeholder");
    return;
  }

  if (region === "global") {
    const option = document.createElement("option");
    option.value = "global";
    option.textContent = "N/A (Global View)";
    option.selected = true;
    realmSelect.appendChild(option);
    realmSelect.disabled = true;
    console.log("Global region selected, disabled realm options");
    return;
  }

  realmSelect.disabled = false;

  const allRealmsOption = document.createElement("option");
  allRealmsOption.value = "all";
  allRealmsOption.textContent = "All " + region.toUpperCase() + " Realms";
  realmSelect.appendChild(allRealmsOption);

  const defaultOption = document.createElement("option");
  defaultOption.value = "";
  defaultOption.textContent = "Select Realm";
  defaultOption.selected = true;
  realmSelect.appendChild(defaultOption);

  const regionData = DATA_MAP[region];
  if (regionData && regionData.realms) {
    Object.entries(regionData.realms).forEach(([slug, name]) => {
      const option = document.createElement("option");
      option.value = slug;
      option.textContent = name;
      realmSelect.appendChild(option);
    });
  }

  console.log("=== updateRealmOptions END ===");
}
