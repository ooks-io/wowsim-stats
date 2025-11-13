// construct player profile url

export function buildPlayerProfileURL(
  region: string,
  realmSlug: string,
  playerName: string,
): string {
  return `/player/${region}/${realmSlug}/${playerName.toLowerCase()}`;
}
