// WoW item quality colors for percentile brackets (CSS variables)
export const BRACKET_COLORS = {
  artifact: "var(--bracket-artifact)", // Artifact (orange/gold)
  legendary: "var(--bracket-legendary)", // Legendary (orange)
  epic: "var(--bracket-epic)", // Epic (purple)
  rare: "var(--bracket-rare)", // Rare (blue)
  uncommon: "var(--bracket-uncommon)", // Uncommon (green)
  common: "var(--bracket-common)", // Common (gray)
} as const;

export type PercentileBracket = keyof typeof BRACKET_COLORS;

// Get color for a percentile bracket
export function getBracketColor(bracket: string | null | undefined): string {
  if (!bracket || !(bracket in BRACKET_COLORS)) {
    return BRACKET_COLORS.common; // Default fallback
  }
  return BRACKET_COLORS[bracket as PercentileBracket];
}

// Get bracket display name (capitalize first letter)
export function getBracketDisplayName(
  bracket: string | null | undefined,
): string {
  if (!bracket) return "Common";
  return bracket.charAt(0).toUpperCase() + bracket.slice(1);
}

// Get bracket CSS class name for styling
export function getBracketClass(bracket: string | null | undefined): string {
  if (!bracket || !(bracket in BRACKET_COLORS)) {
    return "bracket-common";
  }
  return `bracket-${bracket}`;
}

// Check if bracket is high tier (legendary or artifact)
export function isHighTierBracket(bracket: string | null | undefined): boolean {
  return bracket === "legendary" || bracket === "artifact";
}

// Get bracket rank priority (lower number = higher priority)
export function getBracketPriority(bracket: string | null | undefined): number {
  switch (bracket) {
    case "artifact":
      return 1;
    case "legendary":
      return 2;
    case "epic":
      return 3;
    case "rare":
      return 4;
    case "uncommon":
      return 5;
    case "common":
    default:
      return 6;
  }
}
