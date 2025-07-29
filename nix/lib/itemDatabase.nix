{
  inputs,
  lib,
  ...
}: let
  wowsimsDb = lib.importJSON "${inputs.wowsims}/assets/database/db.json";
  glyphDb = import ./database/glyphs.nix;
  talentDb = import ./database/talents.nix;

  itemsById = lib.listToAttrs (map (item: {
      name = toString item.id;
      value = item;
    })
    wowsimsDb.items);

  gemsById = lib.listToAttrs (map (gem: {
      name = toString gem.id;
      value = gem;
    })
    wowsimsDb.gems);

  consumablesById = lib.listToAttrs (map (consumable: {
      name = toString consumable.id;
      value = consumable;
    })
    wowsimsDb.consumables);

  glyphsById = lib.listToAttrs (lib.imap0 (index: glyph: {
      name = toString index;
      value = glyph;
    })
    wowsimsDb.glyphIds);

  enchantsByEffectId = lib.listToAttrs (map (enchant: {
      name = toString enchant.effectId;
      value = enchant;
    })
    wowsimsDb.enchants);

  reforgeData = lib.importJSON "${inputs.wowsims}/assets/db_inputs/wowhead_reforge_stats.json";

  # stat name mappings (for reforging)
  statNames = {
    "spi" = "Spirit";
    "dodgertng" = "Dodge";
    "parryrtng" = "Parry";
    "hitrtng" = "Hit";
    "critstrkrtng" = "Crit";
    "hastertng" = "Haste";
    "exprtng" = "Expertise";
    "mastrtng" = "Mastery";
  };

  # stat array index mappings (for gems/items)
  statIndexNames = {
    "0" = "Strength";
    "1" = "Agility";
    "2" = "Stamina";
    "3" = "Intellect";
    "4" = "Spirit";
    "5" = "Hit";
    "6" = "Crit";
    "7" = "Haste";
    "8" = "Expertise";
    "9" = "Dodge";
    "10" = "Parry";
    "11" = "Mastery";
  };
  # Helper functions for item database operations
  helpers = {
    # get item data by ID
    getItem = id: let
      idStr = toString id;
    in
      if itemsById ? ${idStr}
      then itemsById.${idStr}
      else null;

    # get gem data by ID
    getGem = id: let
      idStr = toString id;
    in
      if gemsById ? ${idStr}
      then gemsById.${idStr}
      else null;

    # get consumable data by ID
    getConsumable = id: let
      idStr = toString id;
    in
      if consumablesById ? ${idStr}
      then consumablesById.${idStr}
      else null;

    # get enchant data by effect ID
    getEnchant = effectId: let
      idStr = toString effectId;
    in
      if enchantsByEffectId ? ${idStr}
      then enchantsByEffectId.${idStr}
      else null;

    # get reforge data by ID
    getReforge = reforgeId: let
      idStr = toString reforgeId;
    in
      if reforgeData ? ${idStr}
      then reforgeData.${idStr}
      else null;

    # get glyph data by ID (from wowsims database)
    getGlyph = glyphId: let
      idStr = toString glyphId;
    in
      if glyphsById ? ${idStr}
      then glyphsById.${idStr}
      else null;

    # get glyph data from our custom glyph database
    getGlyphData = className: glyphId: let
      idStr = toString glyphId;
      classGlyphs = glyphDb.${className} or {};
    in
      if classGlyphs ? ${idStr}
      then classGlyphs.${idStr}
      else null;

    # get talent data from talent database
    getTalentData = className: tier: choice: let
      tierStr = toString tier;
      choiceStr = toString choice;
      classTalents = talentDb.${className} or {};
      tierTalents = classTalents.${tierStr} or {};
    in
      if tierTalents ? ${choiceStr}
      then tierTalents.${choiceStr}
      else null;

    # parse talent string into individual choices
    parseTalentString = talentString: let
      # Convert string to list of characters, then to list of numbers
      chars = lib.stringToCharacters talentString;
      # Convert each character to a number (tier choice 1-3)
      choices =
        map (
          char: let
            num = lib.toInt char;
          in
            if num >= 1 && num <= 3
            then num
            else 0
        )
        chars;
      # Create list of {tier, choice} objects, filtering out invalid choices (0)
      indexedChoices = lib.imap1 (tier: choice: {inherit tier choice;}) choices;
      validChoices = lib.filter (choice: choice.choice != 0) indexedChoices;
    in
      validChoices;

    # parse gem stats into readable format
    parseGemStats = gem:
      if gem ? stats
      then let
        # find non-zero stats and convert to readable format
        indexedStats = lib.imap0 (index: value: {inherit index value;}) gem.stats;
        activeStats = lib.filter (stat: stat.value > 0) indexedStats;
        statDescriptions =
          map (
            stat: let
              statName = statIndexNames.${toString stat.index} or "Unknown Stat";
            in "+${toString stat.value} ${statName}"
          )
          activeStats;
      in
        statDescriptions
      else [];

    # enrich a single gem with stats
    enrichGem = gemId:
      if gemId != null && gemId != 0
      then let
        gem = helpers.getGem gemId;
      in
        if gem != null
        then let
          stats = helpers.parseGemStats gem;
        in {
          inherit (gem) id name;
          icon =
            if gem ? icon
            then gem.icon
            else null;
          color =
            if gem ? color
            then gem.color
            else null;
          quality =
            if gem ? quality
            then gem.quality
            else null;
          stats = stats;
        }
        else null
      else null;

    # enrich enchant info
    enrichEnchant = enchantId:
      if enchantId != null && enchantId != 0
      then {
        id = enchantId;
        name = let
          enchant = helpers.getEnchant enchantId;
        in
          if enchant != null
          then enchant.name
          else "Unknown Enchant ${toString enchantId}";
      }
      else null;

    # enrich reforge info
    enrichReforge = reforgeId:
      if reforgeId != null && reforgeId != 0
      then {
        id = reforgeId;
        description = let
          reforge = helpers.getReforge reforgeId;
        in
          if reforge != null
          then let
            fromStat = statNames.${reforge.s1} or reforge.s1;
            toStat = statNames.${reforge.s2} or reforge.s2;
          in "${fromStat} -> ${toStat}"
          else "Unknown Reforge ${toString reforgeId}";
      }
      else null;

    # enrich a single equipment item
    enrichItem = item:
      if !(item ? id)
      then item
      else let
        itemData = helpers.getItem item.id;
        enrichedGems =
          if item ? gems
          then lib.filter (gem: gem != null) (map helpers.enrichGem item.gems)
          else [];
        enrichedEnchant =
          if item ? enchant
          then helpers.enrichEnchant item.enchant
          else null;
        enrichedReforge =
          if item ? reforging
          then helpers.enrichReforge item.reforging
          else null;

        # Extract item stats from scalingOptions if available
        itemStats =
          if itemData != null && itemData ? scalingOptions && itemData.scalingOptions ? "0"
          then itemData.scalingOptions."0"
          else null;
      in
        item
        // {
          name =
            if itemData != null
            then itemData.name
            else "Item ${toString item.id}";
          icon =
            if itemData != null && itemData ? icon
            then itemData.icon
            else null;
          quality =
            if itemData != null && itemData ? quality
            then itemData.quality
            else null;
          type =
            if itemData != null && itemData ? type
            then itemData.type
            else null;
        }
        // lib.optionalAttrs (itemStats != null) {
          stats = itemStats;
        }
        // lib.optionalAttrs (enrichedGems != []) {
          gems = enrichedGems;
        }
        // lib.optionalAttrs (enrichedEnchant != null) {
          enchant = enrichedEnchant;
        }
        // lib.optionalAttrs (enrichedReforge != null) {
          reforging = enrichedReforge;
        };
  };
in {
  # get item data by ID
  getItem = helpers.getItem;

  # get just the item name by ID
  getItemName = id: let
    item = helpers.getItem id;
  in
    if item != null
    then item.name
    else "Unknown Item ${toString id}";

  # get gem data by ID
  getGem = helpers.getGem;

  # get gem name by ID
  getGemName = id: let
    gem = helpers.getGem id;
  in
    if gem != null
    then gem.name
    else "Unknown Gem ${toString id}";

  # parse gem stats into readable format
  parseGemStats = helpers.parseGemStats;

  # get enriched gem data with stats (delegates to helpers)
  getEnrichedGem = helpers.enrichGem;

  # get enchant data by effect ID
  getEnchant = helpers.getEnchant;

  # get enchant name by effect ID
  getEnchantName = effectId: let
    enchant = helpers.getEnchant effectId;
  in
    if enchant != null
    then enchant.name
    else "Unknown Enchant ${toString effectId}";

  # get reforge data by ID
  getReforge = helpers.getReforge;

  # get reforge description by ID
  getReforgeDescription = reforgeId: let
    reforge = helpers.getReforge reforgeId;
  in
    if reforge != null
    then let
      fromStat = statNames.${reforge.s1} or reforge.s1;
      toStat = statNames.${reforge.s2} or reforge.s2;
    in "${fromStat} -> ${toStat}"
    else "Unknown Reforge ${toString reforgeId}";

  # get glyph data by ID
  getGlyph = helpers.getGlyph;

  # get glyph name by ID
  getGlyphName = glyphId: let
    glyph = helpers.getGlyph glyphId;
    item =
      if glyph != null
      then helpers.getItem glyph.itemId
      else null;
  in
    if item != null
    then item.name
    else "Unknown Glyph ${toString glyphId}";

  # enrich equipment item with name and other data
  enrichEquipmentItem = helpers.enrichItem;

  # enrich an entire equipment array
  enrichEquipment = equipment:
    if equipment ? items
    then
      equipment
      // {
        items = map helpers.enrichItem equipment.items;
      }
    else equipment;

  # enrich consumables with item names and icons
  enrichConsumables = consumables:
    if consumables == null
    then null
    else let
      getConsumableDataSafe = id: let
        consumable = helpers.getConsumable id;
      in
        if consumable != null
        then {
          name = consumable.name;
          icon =
            if consumable ? icon
            then consumable.icon
            else null;
        }
        else {
          name = "Consumable ${toString id}";
          icon = null;
        };
    in
      consumables
      // lib.optionalAttrs (consumables ? flaskId && consumables.flaskId != 0) (
        let
          flaskData = getConsumableDataSafe consumables.flaskId;
        in {
          flaskName = flaskData.name;
          flaskIcon = flaskData.icon;
        }
      )
      // lib.optionalAttrs (consumables ? foodId && consumables.foodId != 0) (
        let
          foodData = getConsumableDataSafe consumables.foodId;
        in {
          foodName = foodData.name;
          foodIcon = foodData.icon;
        }
      )
      // lib.optionalAttrs (consumables ? potId && consumables.potId != 0) (
        let
          potData = getConsumableDataSafe consumables.potId;
        in {
          potName = potData.name;
          potIcon = potData.icon;
        }
      )
      // lib.optionalAttrs (consumables ? prepotId && consumables.prepotId != 0) (
        let
          prepotData = getConsumableDataSafe consumables.prepotId;
        in {
          prepotName = prepotData.name;
          prepotIcon = prepotData.icon;
        }
      );

  # enrich glyphs with names and icons from glyph database
  enrichGlyphs = className: glyphs:
    if glyphs == null
    then null
    else let
      getGlyphDataSafe = glyphId: let
        glyphData = helpers.getGlyphData className glyphId;
      in
        if glyphData != null
        then {
          name = glyphData.name;
          icon =
            if glyphData ? icon
            then glyphData.icon
            else null;
          spellId =
            if glyphData ? spellId
            then glyphData.spellId
            else null;
        }
        else {
          name = "Glyph ${toString glyphId}";
          icon = null;
          spellId = null;
        };
    in
      glyphs
      // lib.optionalAttrs (glyphs ? major1 && glyphs.major1 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.major1;
        in {
          major1Name = glyphData.name;
          major1Icon = glyphData.icon;
          major1SpellId = glyphData.spellId;
        }
      )
      // lib.optionalAttrs (glyphs ? major2 && glyphs.major2 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.major2;
        in {
          major2Name = glyphData.name;
          major2Icon = glyphData.icon;
          major2SpellId = glyphData.spellId;
        }
      )
      // lib.optionalAttrs (glyphs ? major3 && glyphs.major3 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.major3;
        in {
          major3Name = glyphData.name;
          major3Icon = glyphData.icon;
          major3SpellId = glyphData.spellId;
        }
      )
      // lib.optionalAttrs (glyphs ? minor1 && glyphs.minor1 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.minor1;
        in {
          minor1Name = glyphData.name;
          minor1Icon = glyphData.icon;
          minor1SpellId = glyphData.spellId;
        }
      )
      // lib.optionalAttrs (glyphs ? minor2 && glyphs.minor2 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.minor2;
        in {
          minor2Name = glyphData.name;
          minor2Icon = glyphData.icon;
          minor2SpellId = glyphData.spellId;
        }
      )
      // lib.optionalAttrs (glyphs ? minor3 && glyphs.minor3 != 0) (
        let
          glyphData = getGlyphDataSafe glyphs.minor3;
        in {
          minor3Name = glyphData.name;
          minor3Icon = glyphData.icon;
          minor3SpellId = glyphData.spellId;
        }
      );

  # enrich talents with names and icons from talent database
  enrichTalents = className: talentString:
    if talentString == null || talentString == ""
    then null
    else let
      # Parse talent string into tier/choice pairs
      talentChoices = helpers.parseTalentString talentString;

      # Get talent data for each choice and create enriched talent objects
      enrichedTalents =
        map (choice: let
          talentData = helpers.getTalentData className choice.tier choice.choice;
        in {
          tier = choice.tier;
          choice = choice.choice;
          name =
            if talentData != null
            then talentData.name
            else "Unknown Talent";
          spellId =
            if talentData != null
            then talentData.spellId
            else null;
          icon =
            if talentData != null
            then talentData.icon
            else null;
        })
        talentChoices;
    in {
      talentString = talentString;
      talents = enrichedTalents;
    };

  # enrich a full loadout
  enrichLoadout = loadout:
    if loadout ? equipment
    then
      loadout
      // {
        equipment =
          if loadout.equipment ? items
          then
            loadout.equipment
            // {
              items = map helpers.enrichItem loadout.equipment.items;
            }
          else loadout.equipment;
        # TODO: Add consumables, gems, enchants enrichment
      }
    else loadout;

  # enrich simulation results (main function)
  enrichSimulationData = simData:
    if simData ? results
    then
      simData
      // {
        results =
          lib.mapAttrs (
            name: result:
              if result ? loadout
              then
                result
                // {
                  loadout =
                    if result.loadout ? equipment && result.loadout.equipment ? items
                    then
                      result.loadout
                      // {
                        equipment =
                          result.loadout.equipment
                          // {
                            items = map helpers.enrichItem result.loadout.equipment.items;
                          };
                        # TODO: Add consumables, gems, enchants enrichment
                      }
                    else result.loadout;
                }
              else result
          )
          simData.results;
      }
    else simData;
}
