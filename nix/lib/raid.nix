{lib, ...}: let
  inherit (lib.sim.party) mkParty;
  # TODO, dynamically set buffs/debuffs based on specs present
  mkRaid = {
    party1 ? [],
    party2 ? [],
    party3 ? [],
    party4 ? [],
    party5 ? [],
    buffs,
    debuffs,
    targetDummies ? 1,
  }: let
    allParties = [party1 party2 party3 party4 party5];
    # Count non-empty parties
    # numActiveParties = builtins.length (builtins.filter (party: party != []) allParties);
    numActiveParties = 5;
  in {
    inherit buffs debuffs targetDummies numActiveParties;
    parties = [
      (mkParty party1)
      (mkParty party2)
      (mkParty party3)
      (mkParty party4)
      (mkParty party5)
    ];
  };
in {inherit mkRaid;}
