{lib, ...}: let
  inherit (lib.sim.party) mkParty;

  # Extract class/spec from player
  getPlayerClassSpec = player:
    if builtins.isAttrs player && builtins.hasAttr "class" player
    then let
      # Extract class name from the enum-style class field
      # e.g., "ClassShaman" -> "shaman"
      classStr = player.class;
      className =
        if builtins.isString classStr && lib.hasPrefix "Class" classStr
        then lib.toLower (lib.removePrefix "Class" classStr)
        else null;
      
      # Extract spec from spec-specific fields like "elementalShaman", "brewmasterMonk", etc.
      specName = 
        if builtins.hasAttr "elementalShaman" player then "elemental"
        else if builtins.hasAttr "enhancementShaman" player then "enhancement"
        else if builtins.hasAttr "restorationShaman" player then "restoration"
        else if builtins.hasAttr "brewmasterMonk" player then "brewmaster"
        else if builtins.hasAttr "windwalkerMonk" player then "windwalker"
        else if builtins.hasAttr "mistweaverMonk" player then "mistweaver"
        else if builtins.hasAttr "survivalHunter" player then "survival"
        else if builtins.hasAttr "beastMasteryHunter" player then "beast_mastery"
        else if builtins.hasAttr "marksmanshipHunter" player then "marksmanship"
        else if builtins.hasAttr "balanceDruid" player then "balance"
        else if builtins.hasAttr "feralDruid" player then "feral"
        else if builtins.hasAttr "guardianDruid" player then "guardian"
        else if builtins.hasAttr "restorationDruid" player then "restoration"
        else if builtins.hasAttr "holyPaladin" player then "holy"
        else if builtins.hasAttr "protectionPaladin" player then "protection"
        else if builtins.hasAttr "retributionPaladin" player then "retribution"
        else null;
    in {
      class = className;
      spec = specName;
    }
    else {
      class = null;
      spec = null;
    };

  # Get all players from all parties
  getAllPlayers = parties: lib.flatten (builtins.filter (party: party != []) parties);

  # Generate dynamic buffs based on raid composition
  generateDynamicBuffs = players: let
    classSpecs = builtins.filter (cs: cs.class != null) (map getPlayerClassSpec players);
    classes = lib.unique (map (cs: cs.class) classSpecs);
    specs = lib.unique (builtins.filter (spec: spec != null) (map (cs: cs.spec) classSpecs));

    # Count classes for stacking buffs
    classCount = class: builtins.length (builtins.filter (cs: cs.class == class) classSpecs);

    # Check for class/spec presence
    hasClass = class: builtins.elem class classes;
    hasSpec = spec: builtins.elem spec specs;
  in {
    # Class-based buffs
    trueshotAura = hasClass "hunter";
    unholyAura = hasClass "death_knight";
    darkIntent = hasClass "warlock";
    moonkinAura = hasSpec "balance"; # Now properly detects balance druids
    leaderOfThePack = hasClass "druid";
    blessingOfMight = hasClass "paladin";
    legacyOfTheEmperor = hasClass "monk";

    # Universal buffs
    bloodlust = true; # Assume bloodlust/heroism is always available

    # Stacking buffs - count actual providers, capped at reasonable maximums
    stormlashTotemCount = lib.min 4 (classCount "shaman");
    skullBannerCount = lib.min 2 (classCount "warrior");

    # Spec-specific buffs
    legacyOfTheWhiteTiger = hasSpec "windwalker";
  };

  # Generate dynamic debuffs based on raid composition
  generateDynamicDebuffs = players: let
    classSpecs = builtins.filter (cs: cs.class != null) (map getPlayerClassSpec players);
    classes = lib.unique (map (cs: cs.class) classSpecs);

    hasClass = class: builtins.elem class classes;
  in {
    curseOfElements = hasClass "warlock";
    masterPoisoner = hasClass "rogue";
    physicalVulnerability = hasClass "warrior" || hasClass "death_knight";
    weakenedArmor = hasClass "warrior" || hasClass "rogue" || hasClass "druid";
  };

  mkRaid = {
    party1 ? [],
    party2 ? [],
    party3 ? [],
    party4 ? [],
    party5 ? [],
    buffs ? null,
    debuffs ? null,
    dynamicBuffs ? false,
    targetDummies ? 1,
  }: let
    allParties = [party1 party2 party3 party4 party5];
    allPlayers = getAllPlayers allParties;

    # Count non-empty parties
    # numActiveParties = builtins.length (builtins.filter (party: party != []) allParties);
    numActiveParties = 5;

    # Use dynamic buffs/debuffs if requested, otherwise use provided ones
    finalBuffs =
      if dynamicBuffs
      then generateDynamicBuffs allPlayers
      else buffs;
    finalDebuffs =
      if dynamicBuffs
      then generateDynamicDebuffs allPlayers
      else debuffs;
  in {
    buffs = finalBuffs;
    debuffs = finalDebuffs;
    inherit targetDummies numActiveParties;
    parties = [
      (mkParty party1)
      (mkParty party2)
      (mkParty party3)
      (mkParty party4)
      (mkParty party5)
    ];
  };
in {inherit mkRaid;}
