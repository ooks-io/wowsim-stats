/**
 * Shared utility functions for WoW Stats application
 */
export function formatDuration(seconds) {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return minutes > 0
    ? `${minutes}m ${remainingSeconds}s`
    : `${remainingSeconds}s`;
}

export function formatDurationMMSS(milliseconds) {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return minutes + ":" + seconds.toString().padStart(2, "0");
}

export function formatTimestamp(timestamp) {
  return new Date(timestamp).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatSimulationDate(timestamp) {
  return new Date(timestamp).toLocaleString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
  });
}

export function formatRaidBuffs(buffs) {
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

export function formatRace(race) {
  if (!race) return "Unknown";
  return race.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
}

export function formatNumber(num) {
  if (num >= 1000000000) {
    return (num / 1000000000).toFixed(1) + "B";
  }
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + "M";
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + "K";
  }
  return num.toString();
}

export function formatDPS(dps) {
  return Math.round(dps).toLocaleString();
}

export function camelToReadable(str) {
  return str
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (char) => char.toUpperCase());
}

export function capitalizeWords(str) {
  return str.replace(/\b\w/g, (l) => l.toUpperCase());
}

export function toSlug(str) {
  return str.toLowerCase().replace(/[^a-z0-9]/g, "-");
}

export function deepClone(obj) {
  if (obj === null || typeof obj !== "object") return obj;
  if (obj instanceof Date) return new Date(obj.getTime());
  if (obj instanceof Array) return obj.map((item) => deepClone(item));
  if (typeof obj === "object") {
    const clonedObj = {};
    for (const key in obj) {
      if (obj.hasOwnProperty(key)) {
        clonedObj[key] = deepClone(obj[key]);
      }
    }
    return clonedObj;
  }
}

export function isEmpty(obj) {
  return obj && Object.keys(obj).length === 0 && obj.constructor === Object;
}

export function groupBy(array, key) {
  return array.reduce((result, currentValue) => {
    (result[currentValue[key]] = result[currentValue[key]] || []).push(
      currentValue,
    );
    return result;
  }, {});
}

export function sortBy(array, keys) {
  return array.sort((a, b) => {
    for (const { key, order = "asc" } of keys) {
      let aVal = a[key];
      let bVal = b[key];

      if (typeof aVal === "string") aVal = aVal.toLowerCase();
      if (typeof bVal === "string") bVal = bVal.toLowerCase();

      if (aVal < bVal) return order === "asc" ? -1 : 1;
      if (aVal > bVal) return order === "asc" ? 1 : -1;
    }
    return 0;
  });
}

export function addClass(element, ...classes) {
  if (element && element.classList) {
    element.classList.add(...classes);
  }
}

export function removeClass(element, ...classes) {
  if (element && element.classList) {
    element.classList.remove(...classes);
  }
}

export function toggleClass(element, className, force) {
  if (element && element.classList) {
    return element.classList.toggle(className, force);
  }
}

export function isValidNumber(value) {
  return !isNaN(value) && isFinite(value);
}

export function isValidUrl(str) {
  try {
    new URL(str);
    return true;
  } catch {
    return false;
  }
}
