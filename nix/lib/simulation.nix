let
  mkSim = {
    requestId ? "raidSimAsync-b01aabae125f334e",
    type ? "SimTypeIndividual",
    iterations,
    randomSeed ? 937883464,
    raid,
  }: {
    inherit type raid requestId;
    simOptions = {
      inherit iterations randomSeed;
      debugFirstIteration = true;
    };
  };
in
  mkSim
