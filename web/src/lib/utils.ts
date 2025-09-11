// Common utility functions
import { DUNGEON_MAP, dungeonNameToSlug } from "./wow-constants";

// Time formatting utilities
export function formatDuration(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return minutes > 0
    ? `${minutes}m ${remainingSeconds}s`
    : `${remainingSeconds}s`;
}

export function formatDurationMMSS(milliseconds: number): string {
  return `${Math.floor(milliseconds / 60000)}:${String(Math.floor((milliseconds % 60000) / 1000)).padStart(2, "0")}`;
}

// Alias for clarity when working with milliseconds
export const formatDurationFromMs = formatDurationMMSS;

export function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatSimulationDate(timestamp: string | number): string {
  return new Date(timestamp).toLocaleString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
  });
}

export function formatRaidBuffs(
  buffs: Record<string, boolean | number>,
): string {
  const activeBuffs = Object.entries(buffs)
    .filter(([_, value]) => value !== false && value !== 0)
    .map(([key, value]) => {
      const readable = key
        .replace(/([A-Z])/g, " $1")
        .replace(/^./, (str) => str.toUpperCase());
      return typeof value === "number" && value > 1
        ? `${readable} (${value})`
        : readable;
    });
  return activeBuffs.join(", ");
}

export function formatRace(race: string): string {
  if (!race) return "Unknown";
  return race.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
}

// DOM utilities
export function createElement(
  tag: string,
  className?: string,
  textContent?: string,
): HTMLElement {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (textContent) element.textContent = textContent;
  return element;
}

export function toggleElementVisibility(
  element: HTMLElement,
  show: boolean,
): void {
  element.style.display = show ? "block" : "none";
}

// URL utilities
export function updateBrowserURL(url: string): void {
  window.history.pushState({}, "", url);
}

export function getURLSearchParams(): URLSearchParams {
  return new URLSearchParams(window.location.search);
}

export function buildLeaderboardURL(
  region: string,
  realm: string,
  dungeonSlug: string,
  page?: number,
): string {
  let baseURL: string;

  if (region === "global") {
    baseURL = `/challenge-mode/global/${dungeonSlug}`;
  } else if (realm === "all") {
    baseURL = `/challenge-mode/${region}/all/${dungeonSlug}`;
  } else {
    baseURL = `/challenge-mode/${region}/${realm}/${dungeonSlug}`;
  }

  if (page && page > 1) {
    baseURL += `?page=${page}`;
  }

  return baseURL;
}

export function buildPlayerProfileURL(
  region: string,
  realmSlug: string,
  playerName: string,
): string {
  return `/player/${region}/${realmSlug}/${playerName.toLowerCase()}`;
}

// Form utilities
export function getFormValue(elementId: string): string {
  const element = document.getElementById(elementId) as
    | HTMLInputElement
    | HTMLSelectElement;
  return element?.value || "";
}

export function setFormValue(elementId: string, value: string): void {
  const element = document.getElementById(elementId) as
    | HTMLInputElement
    | HTMLSelectElement;
  if (element) element.value = value;
}

// Validation utilities
export function validateRequiredFields(
  ...values: (string | undefined)[]
): boolean {
  return values.every((value) => value && value.trim().length > 0);
}

// Error handling utilities
export function handleAPIError(error: Error, context: string): void {
  console.error(`${context} error:`, error);

  // Could extend this to show user-friendly error messages
  const errorMessage = error.message.includes("404")
    ? "Data not found"
    : error.message.includes("500")
      ? "Server error - please try again later"
      : "An error occurred loading data";

  console.warn(`User-friendly error: ${errorMessage}`);
}

// Loading state utilities
export function showLoadingState(
  containerId: string,
  message: string = "Loading...",
): void {
  const container = document.getElementById(containerId);
  if (container) {
    container.innerHTML = `
      <div class="loading-state">
        <div class="loading-spinner"></div>
        <div class="loading-message">${message}</div>
      </div>
    `;
  }
}

export function hideLoadingState(containerId: string): void {
  const container = document.getElementById(containerId);
  if (container) {
    const loadingState = container.querySelector(".loading-state");
    if (loadingState) {
      loadingState.remove();
    }
  }
}

// Debouncing utility for search inputs
export function debounce<T extends (...args: any[]) => void>(
  func: T,
  wait: number,
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout;

  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

// Local storage utilities
export function saveToLocalStorage(key: string, data: any): void {
  try {
    localStorage.setItem(key, JSON.stringify(data));
  } catch (error) {
    console.warn("Failed to save to localStorage:", error);
  }
}

export function loadFromLocalStorage<T>(key: string): T | null {
  try {
    const item = localStorage.getItem(key);
    return item ? JSON.parse(item) : null;
  } catch (error) {
    console.warn("Failed to load from localStorage:", error);
    return null;
  }
}

// Array utilities
export function chunk<T>(array: T[], size: number): T[][] {
  const chunks: T[][] = [];
  for (let i = 0; i < array.length; i += size) {
    chunks.push(array.slice(i, i + size));
  }
  return chunks;
}

export function unique<T>(array: T[]): T[] {
  return Array.from(new Set(array));
}

// Object utilities
export function isEmpty(obj: any): boolean {
  if (!obj) return true;
  if (Array.isArray(obj)) return obj.length === 0;
  if (typeof obj === "object") return Object.keys(obj).length === 0;
  return false;
}

// Static API utilities for file-based endpoints
export function dungeonIdToSlug(dungeonId: number): string {
  const dungeonName = DUNGEON_MAP[dungeonId as keyof typeof DUNGEON_MAP];
  if (!dungeonName) {
    throw new Error(`Unknown dungeon ID: ${dungeonId}`);
  }
  return dungeonNameToSlug(dungeonName);
}

// Build static API file paths
export function buildStaticLeaderboardPath(
  region: string,
  realm: string,
  dungeonId: number,
  page: number = 1,
): string {
  const dungeonSlug = dungeonIdToSlug(dungeonId);

  if (region === "global") {
    return `/api/leaderboard/global/${dungeonSlug}/${page}.json`;
  } else if (realm === "all") {
    return `/api/leaderboard/${region}/all/${dungeonSlug}/${page}.json`;
  } else {
    return `/api/leaderboard/${region}/${realm}/${dungeonSlug}/${page}.json`;
  }
}

export function buildStaticPlayerLeaderboardPath(
  scope: string,
  region?: string,
  page: number = 1,
): string {
  if (scope === "regional" && region) {
    return `/api/leaderboard/players/regional/${region}/${page}.json`;
  } else {
    return `/api/leaderboard/players/global/${page}.json`;
  }
}

export function buildStaticPlayerProfilePath(
  region: string,
  realmSlug: string,
  playerName: string,
): string {
  // Normalize to lowercase to match public/ directory structure on case-sensitive hosts
  const normRegion = (region || "").toLowerCase();
  const normRealm = (realmSlug || "").toLowerCase();
  const normPlayer = (playerName || "").toLowerCase();
  return `/api/player/${normRegion}/${normRealm}/${normPlayer}.json`;
}

export function buildStaticSearchIndexPath(shardNumber: number): string {
  const paddedNumber = shardNumber.toString().padStart(3, "0");
  return `/api/search/players-${paddedNumber}.json`;
}
