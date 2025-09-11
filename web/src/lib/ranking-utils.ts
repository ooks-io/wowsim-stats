// Ranking and bracket utility functions
import { getBracketClass } from "./bracket-colors.ts";

// Server-side ranking formatter
export function formatRankingWithBracket(
  ranking: any,
  bracket: string,
): string {
  if (!ranking || ranking === "—") return "—";
  const bracketClass = getBracketClass(bracket);
  return `<span class="ranking-number ${bracketClass}">${ranking}</span>`;
}
