// Client-side utility functions

import {
  CLASS_COLORS,
  SPEC_MAP,
  SPEC_ICON_MAP,
  getClassColor,
  getSpecInfo,
  getSpecIcon,
} from "./wow-constants";

// Time formatting utilities
export function formatDurationMMSS(milliseconds: number): string {
  return `${Math.floor(milliseconds / 60000)}:${String(Math.floor((milliseconds % 60000) / 1000)).padStart(2, "0")}`;
}

export function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleDateString();
}

export function formatTimestampDetailed(timestamp: number): string {
  return new Date(timestamp).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// DOM utility functions (commonly used in client scripts)
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

// Type helper functions for DOM elements (commonly used pattern)
export const asHtml = (el: Element | null): HTMLElement | null =>
  el as HTMLElement | null;

export const asButton = (el: Element | null): HTMLButtonElement | null =>
  el as HTMLButtonElement | null;

// Client-side ranking formatter (similar to server-side but for client scripts)
export function formatRankingWithBracket(
  ranking: any,
  bracket: string,
): string {
  if (!ranking || ranking === "—") return "—";
  const bracketClass = getBracketClass(bracket);
  return `<span class="ranking-number ${bracketClass}">#${ranking}</span>`;
}

// Client-side bracket class helper
function getBracketClass(bracket: string): string {
  if (!bracket) return "bracket-common";
  return `bracket-${bracket}`;
}

// Re-exports for backward compatibility in client code
export { getClassColor, getSpecInfo, getSpecIcon };
export const specMap = SPEC_MAP;
export const specIconMap = SPEC_ICON_MAP;
export const classColors = CLASS_COLORS;
