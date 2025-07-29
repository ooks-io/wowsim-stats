{
  inputs,
  writeShellApplication,
  lib,
  ...
}:
writeShellApplication {
  name = "testItemLookup";
  text = let
    itemDb = lib.sim.itemDatabase;

    # Test some item lookups
    testItem1 = itemDb.getItem 87126; # From our example data
    testItem2 = itemDb.getItemName 87166;
    testGem = itemDb.getEnrichedGem 76884;
    testEnchant = itemDb.getEnchantName 4804;
    testReforge = itemDb.getReforgeDescription 167;

    # Create a test loadout structure
    testLoadout = {
      equipment = {
        items = [
          {
            id = 87126;
            enchant = 4804;
            gems = [76884 76680];
            reforging = 167;
          }
          {
            id = 87166;
            enchant = 4444;
            gems = [89873];
            reforging = 158;
          }
          {
            id = 89917;
            reforging = 166;
          }
        ];
      };
    };

    enrichedLoadout = itemDb.enrichLoadout testLoadout;
  in ''
    echo "=== Testing Item Database Functions ==="
    echo ""

    echo "Test item 87126:"
    echo '${builtins.toJSON testItem1}'
    echo ""

    echo "Test item name 87166: ${testItem2}"
    echo "Test gem 76884:"
    echo '${builtins.toJSON testGem}'
    echo "Test enchant name 4804: ${testEnchant}"
    echo "Test reforge 167: ${testReforge}"
    echo ""

    echo "Original loadout:"
    echo '${builtins.toJSON testLoadout}'
    echo ""

    echo "Enriched loadout:"
    echo '${builtins.toJSON enrichedLoadout}'
  '';
}
