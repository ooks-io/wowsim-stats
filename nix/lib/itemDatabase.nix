{
  inputs,
  lib,
  ...
}: let
  wowsimsDb = lib.importJSON "${inputs.wowsims}/assets/database/db.json";

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

  reforgeData = {
    "113" = {
      "i1" = 6;
      "s1" = "spi";
      "i2" = 13;
      "s2" = "dodgertng";
    };
    "138" = {
      "i1" = 31;
      "s1" = "hitrtng";
      "i2" = 36;
      "s2" = "hastertng";
    };
    "158" = {
      "i1" = 37;
      "s1" = "exprtng";
      "i2" = 31;
      "s2" = "hitrtng";
    };
    "159" = {
      "i1" = 37;
      "s1" = "exprtng";
      "i2" = 32;
      "s2" = "critstrkrtng";
    };
    "166" = {
      "i1" = 49;
      "s1" = "mastrtng";
      "i2" = 32;
      "s2" = "critstrkrtng";
    };
    "167" = {
      "i1" = 49;
      "s1" = "mastrtng";
      "i2" = 36;
      "s2" = "hastertng";
    };
    "168" = {
      "i1" = 49;
      "s1" = "mastrtng";
      "i2" = 37;
      "s2" = "exprtng";
    };
  };

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

    # get glyph data by ID
    getGlyph = glyphId: let
      idStr = toString glyphId;
    in
      if glyphsById ? ${idStr}
      then glyphsById.${idStr}
      else null;

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
          icon = if gem ? icon then gem.icon else null;
          color = if gem ? color then gem.color else null;
          quality = if gem ? quality then gem.quality else null;
          stats = stats;
          description =
            if stats != []
            then "${gem.name} -- ${lib.concatStringsSep ", " stats}"
            else gem.name;
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
      if !(item ? id) then item
      else let
        itemData = helpers.getItem item.id;
        enrichedGems = if item ? gems 
          then lib.filter (gem: gem != null) (map helpers.enrichGem item.gems)
          else [];
        enrichedEnchant = if item ? enchant 
          then helpers.enrichEnchant item.enchant
          else null;
        enrichedReforge = if item ? reforging 
          then helpers.enrichReforge item.reforging
          else null;
        
        # Extract item stats from scalingOptions if available
        itemStats = if itemData != null && itemData ? scalingOptions && itemData.scalingOptions ? "0"
          then itemData.scalingOptions."0"
          else null;
      in
        item // {
          name = if itemData != null then itemData.name else "Item ${toString item.id}";
          icon = if itemData != null && itemData ? icon then itemData.icon else null;
          quality = if itemData != null && itemData ? quality then itemData.quality else null;
          type = if itemData != null && itemData ? type then itemData.type else null;
        } // lib.optionalAttrs (itemStats != null) {
          stats = itemStats;
        } // lib.optionalAttrs (enrichedGems != []) {
          gems = enrichedGems;
        } // lib.optionalAttrs (enrichedEnchant != null) {
          enchant = enrichedEnchant;
        } // lib.optionalAttrs (enrichedReforge != null) {
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
    item = if glyph != null then helpers.getItem glyph.itemId else null;
  in
    if item != null then item.name else "Unknown Glyph ${toString glyphId}";

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
    if consumables == null then null
    else let
      getItemDataSafe = id: let item = helpers.getItem id; in
        if item != null then {
          name = item.name;
          icon = if item ? icon then item.icon else null;
          quality = if item ? quality then item.quality else null;
        } else {
          name = "Item ${toString id}";
          icon = null;
          quality = null;
        };
    in
      consumables // lib.optionalAttrs (consumables ? flaskId && consumables.flaskId != 0) (
        let flaskData = getItemDataSafe consumables.flaskId; in {
          flaskName = flaskData.name;
          flaskIcon = flaskData.icon;
          flaskQuality = flaskData.quality;
        }
      ) // lib.optionalAttrs (consumables ? foodId && consumables.foodId != 0) (
        let foodData = getItemDataSafe consumables.foodId; in {
          foodName = foodData.name;
          foodIcon = foodData.icon;
          foodQuality = foodData.quality;
        }
      ) // lib.optionalAttrs (consumables ? potId && consumables.potId != 0) (
        let potData = getItemDataSafe consumables.potId; in {
          potName = potData.name;
          potIcon = potData.icon;
          potQuality = potData.quality;
        }
      ) // lib.optionalAttrs (consumables ? prepotId && consumables.prepotId != 0) (
        let prepotData = getItemDataSafe consumables.prepotId; in {
          prepotName = prepotData.name;
          prepotIcon = prepotData.icon;
          prepotQuality = prepotData.quality;
        }
      );

  # enrich glyphs with item names and icons
  enrichGlyphs = glyphs:
    if glyphs == null then null
    else let
      getGlyphDataSafe = glyphId: let 
        glyph = helpers.getGlyph glyphId;
        item = if glyph != null then helpers.getItem glyph.itemId else null;
      in
        if item != null then {
          name = item.name;
          icon = if item ? icon then item.icon else null;
          quality = if item ? quality then item.quality else null;
          spellId = if glyph != null then glyph.spellId else null;
        } else {
          name = "Glyph ${toString glyphId}";
          icon = null;
          quality = null;
          spellId = null;
        };
    in
      glyphs // lib.optionalAttrs (glyphs ? major1 && glyphs.major1 != 0) (
        let glyphData = getGlyphDataSafe glyphs.major1; in {
          major1Name = glyphData.name;
          major1Icon = glyphData.icon;
          major1Quality = glyphData.quality;
          major1SpellId = glyphData.spellId;
        }
      ) // lib.optionalAttrs (glyphs ? major2 && glyphs.major2 != 0) (
        let glyphData = getGlyphDataSafe glyphs.major2; in {
          major2Name = glyphData.name;
          major2Icon = glyphData.icon;
          major2Quality = glyphData.quality;
          major2SpellId = glyphData.spellId;
        }
      ) // lib.optionalAttrs (glyphs ? major3 && glyphs.major3 != 0) (
        let glyphData = getGlyphDataSafe glyphs.major3; in {
          major3Name = glyphData.name;
          major3Icon = glyphData.icon;
          major3Quality = glyphData.quality;
          major3SpellId = glyphData.spellId;
        }
      ) // lib.optionalAttrs (glyphs ? minor1 && glyphs.minor1 != 0) (
        let glyphData = getGlyphDataSafe glyphs.minor1; in {
          minor1Name = glyphData.name;
          minor1Icon = glyphData.icon;
          minor1Quality = glyphData.quality;
          minor1SpellId = glyphData.spellId;
        }
      ) // lib.optionalAttrs (glyphs ? minor2 && glyphs.minor2 != 0) (
        let glyphData = getGlyphDataSafe glyphs.minor2; in {
          minor2Name = glyphData.name;
          minor2Icon = glyphData.icon;
          minor2Quality = glyphData.quality;
          minor2SpellId = glyphData.spellId;
        }
      ) // lib.optionalAttrs (glyphs ? minor3 && glyphs.minor3 != 0) (
        let glyphData = getGlyphDataSafe glyphs.minor3; in {
          minor3Name = glyphData.name;
          minor3Icon = glyphData.icon;
          minor3Quality = glyphData.quality;
          minor3SpellId = glyphData.spellId;
        }
      );

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
                        equipment = result.loadout.equipment
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

