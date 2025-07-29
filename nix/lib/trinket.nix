{lib, ...}: let
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
in {
  inherit removeTrinkets addTrinket generateTrinketGearsets;
}
