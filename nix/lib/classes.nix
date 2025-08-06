{lib, ...}: let
  inherit (lib.sim.player) mkPlayer;
  mkClassTemplate = {
    playableRaces,
    class,
    spec,
    singleTarget,
    consumables,
    cleave,
    multiTarget,
    profession1,
    profession2,
    distanceFromTarget,
    options,
    challengeMode,
  }:
    lib.listToAttrs (map (race: {
        name = race;
        value = {
          p1 = {
            raid = {
              singleTarget = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (singleTarget) glyphs talents apl;
                gearset = singleTarget.p1.gearset;
              };
              multiTarget = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (multiTarget) glyphs talents apl;
                gearset = multiTarget.p1.gearset;
              };
              cleave = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (cleave) glyphs talents apl;
                gearset = cleave.p1.gearset;
              };
            };
            challengeMode = {
              singleTarget = mkPlayer {
                challengeMode = true;
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (singleTarget) apl;
                inherit (challengeMode) glyphs gearset talents;
              };
              multiTarget = mkPlayer {
                challengeMode = true;
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (multiTarget) apl;
                inherit (challengeMode) glyphs gearset talents;
              };
              cleave = mkPlayer {
                challengeMode = true;
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (cleave) apl;
                inherit (challengeMode) glyphs gearset talents;
              };
            };
          };
          preRaid = {
            raid = {
              singleTarget = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (singleTarget) glyphs talents apl;
                inherit (singleTarget.preRaid) gearset;
              };
              multiTarget = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (multiTarget) glyphs talents apl;
                inherit (multiTarget.preRaid) gearset;
              };
              cleave = mkPlayer {
                inherit race class spec options distanceFromTarget profession1 profession2 consumables;
                inherit (cleave) glyphs talents apl;
                inherit (cleave.preRaid) gearset;
              };
            };
          };
        };
      })
      playableRaces);
in {inherit mkClassTemplate;}
