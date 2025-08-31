export function getTeamSignature(members) {
  const playerIds = members
    .map((member) => {
      return member.id || member.profile?.id || 0;
    })
    .sort((a, b) => a - b);

  return playerIds.join("-");
}

export function filterUniqueTeams(leaderboardData) {
  const teamFilter = document.getElementById("team-filter");
  if (!teamFilter.checked) {
    return leaderboardData;
  }

  const uniqueTeams = new Map();

  leaderboardData.forEach((run) => {
    const teamSignature = getTeamSignature(run.members);

    if (
      !uniqueTeams.has(teamSignature) ||
      run.duration < uniqueTeams.get(teamSignature).duration
    ) {
      uniqueTeams.set(teamSignature, run);
    }
  });

  return Array.from(uniqueTeams.values()).sort(
    (a, b) => a.duration - b.duration,
  );
}
