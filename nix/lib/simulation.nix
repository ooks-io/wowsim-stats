{lib, ...}: let
  inherit (lib.sim.raid) mkRaid;
  mkSim = {
    requestId ? "raidSimAsync-b01aabae125f334e",
    type ? "SimTypeIndividual",
    iterations,
    encounter,
    randomSeed ? "1113857305",
    raid ? null,
    player ? null,
    buffs ? null,
    debuffs ? null,
  }:
    assert (player != null)
    != (raid != null)
    || throw "Must provide exactly one of 'player' or 'raid', not both";
    assert (player != null)
    -> (buffs != null && debuffs != null)
    || throw "When using 'player', must also provide 'buffs' and 'debuffs'";
    assert (raid != null)
    -> (buffs == null && debuffs == null)
    || throw "When using 'raid', do not provide 'buffs' or 'debuffs' (they should be in the raid)"; let
      actualRaid =
        if player != null
        then
          mkRaid {
            party1 = [player];
            inherit buffs debuffs;
          }
        else raid;
    in
      builtins.toJSON {
        inherit type requestId encounter;
        raid = actualRaid;
        simOptions = {
          inherit iterations randomSeed;
          debugFirstIteration = true;
        };
      };
in {inherit mkSim;}
