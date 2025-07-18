{
  lib,
  self,
  ...
}: let
  inherit (lib.strings) toSentenceCase;
  
  # Import utility modules
  raidBuffsLib = import ./raidBuffs.nix {inherit lib;};
  raidDebuffsLib = import ./raidDebuffs.nix {inherit lib;};
  encountersLib = import ./encounters.nix {inherit lib;};
  generateInfile = {
    class,
    spec,
    race,
    gearset,
    consumables,
    profession1,
    profession2,
    talents,
    buffs ? {},
    debuffs,
    apl,
    targetDummies ? 1,
    distanceFromTarget ? 5,
    numberOfTargets ? 1,
    iterations ? 12500,
    options ? {},
    glyphs,
    raidBuffs ? raidBuffsLib.fullBuffs,
    raidDebuffs ? raidDebuffsLib.fullDebuffs,
    reactionTimeMs ? 100,
    encounter ? encountersLib.long.singleTarget,
    randomSeed ? "4260890857",
    debugFirstIteration ? true,
    challengeMode ? false,
  }: let
    baseEquipment = builtins.fromJSON (builtins.readFile "${self}/ui/${class}/${spec}/gear_sets/${gearset}.gear.json");
    rotation = builtins.fromJSON (builtins.readFile "${self}/ui/${class}/${spec}/apls/${apl}.apl.json");
    
    # Add challengeMode flag to all equipment items if enabled
    equipment = if challengeMode then
      baseEquipment // {
        items = map (item: item // {challengeMode = true;}) baseEquipment.items;
      }
    else
      baseEquipment;
  in
    builtins.toJSON {
      requestId = "raidSimAsync-62a8c84a7df3627";
      raid = {
        parties = [
          {
            players = [
              {
                name = "Player";
                race = "Race${toSentenceCase race}";
                class = "Class${toSentenceCase class}";
                inherit equipment consumables buffs glyphs profession1 profession2 rotation reactionTimeMs distanceFromTarget;
                talentsString = talents;
                cooldowns = {};
                healingModel = {};
                # Add spec-specific options
                "${spec}${toSentenceCase class}" = {
                  inherit options;
                };
              }
              {}
              {}
              {}
              {}
            ];
            buffs = {};
          }
          {
            players = [{} {} {} {} {}];
            buffs = {};
          }
          {
            players = [{} {} {} {} {}];
            buffs = {};
          }
          {
            players = [{} {} {} {} {}];
            buffs = {};
          }
          {
            players = [{} {} {} {} {}];
            buffs = {};
          }
        ];
        numActiveParties = 5;
        buffs = raidBuffs;
        debuffs = raidDebuffs;
        inherit targetDummies;
      };
      inherit encounter;
      simOptions = {
        inherit iterations randomSeed debugFirstIteration;
      };
      type = "SimTypeIndividual";
    };
in
  generateInfile
