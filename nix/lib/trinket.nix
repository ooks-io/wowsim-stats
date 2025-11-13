{lib, inputs, ...}: let
  wowsimsDb = lib.importJSON "${inputs.wowsims}/assets/database/db.json";

  professionEnchantsByProfession = let
    enchantsByEffectId = lib.listToAttrs (map (enchant: {
      name = toString enchant.effectId;
      value = enchant;
    }) wowsimsDb.enchants);
  in {
    "Engineering" = lib.filter (e: e.requiredProfession or null == 4) wowsimsDb.enchants;
    "Tailoring" = lib.filter (e: e.requiredProfession or null == 11) wowsimsDb.enchants;
    "Leatherworking" = lib.filter (e: e.requiredProfession or null == 8) wowsimsDb.enchants;
    "Inscription" = lib.filter (e: e.requiredProfession or null == 6) wowsimsDb.enchants;
    "Enchanting" = lib.filter (e: e.requiredProfession or null == 3) wowsimsDb.enchants;
  };

  professionEnchantEffectIds = lib.mapAttrs (prof: enchants:
    map (e: e.effectId) enchants
  ) professionEnchantsByProfession;
  # Remove trinkets from gearset (set slots 12-13 to empty objects)
  removeTrinkets = gearset:
    if !(gearset ? items)
    then gearset
    else
      gearset
      // {
        items =
          lib.imap0 (
            index: item:
              if index == 12 || index == 13
              then {}
              else item
          )
          gearset.items;
      };

  # Add a single trinket to slot 13 (index 12) of a gearset
  addTrinket = gearset: trinketId:
    if !(gearset ? items)
    then gearset
    else
      gearset
      // {
        items =
          lib.imap0 (
            index: item:
              if index == 12
              then {id = trinketId;}
              else item
          )
          gearset.items;
      };

  # Generate multiple gearsets with different trinkets
  # Takes a baseline gearset (should have no trinkets) and a list of trinket IDs
  # Returns a list of gearsets, each with one trinket in slot 13
  generateTrinketGearsets = baselineGearset: trinketIds:
    map (trinketId: addTrinket baselineGearset trinketId) trinketIds;

  # Remove profession-specific enchants from gearset
  # Note: This operates on raw equipment BEFORE enrichment, so item.enchant is just a number (effectId)
  removeProfessionEnchants = gearset: professionName:
    if !(gearset ? items) || !(professionEnchantEffectIds ? ${professionName})
    then gearset
    else let
      enchantIds = professionEnchantEffectIds.${professionName};
      hasProfessionEnchant = item:
        (item ? enchant) && (builtins.elem item.enchant enchantIds);
    in
      gearset
      // {
        items = map (item:
          if hasProfessionEnchant item
          then builtins.removeAttrs item ["enchant"]
          else item
        ) gearset.items;
      };
in {
  inherit removeTrinkets addTrinket generateTrinketGearsets removeProfessionEnchants;
}
