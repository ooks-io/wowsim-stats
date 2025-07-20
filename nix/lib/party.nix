{lib, ...}: let
  mkParty = players: {
    players = players ++ (lib.genList (_: {}) (5 - (lib.length players)));
    buffs = {};
  };
in {inherit mkParty;}
