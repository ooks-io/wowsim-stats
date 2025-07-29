{
  inputs,
  writeShellApplication,
  lib,
  ...
}:
writeShellApplication {
  name = "testEnrichmentOutput";
  text = let
    itemDb = lib.sim.itemDatabase;

    # Create a test config similar to what would be in death_knight frost
    testConfig = {
      class = "death_knight";
      race = "orc";
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
        ];
      };
    };

    enrichedConfig = itemDb.enrichLoadout testConfig;
    enrichedConfigJson = builtins.toJSON enrichedConfig;
  in ''
    echo "=== Testing JSON embedding in bash ==="
    echo ""

    # Test if the JSON can be used in bash without syntax errors
    loadout=$(echo '${enrichedConfigJson}' | jq '{
      equipment,
      race,
      class
    }')

    echo "Success! JSON was processed without bash syntax errors."
    echo "Loadout keys:"
    echo "$loadout" | jq -r 'keys[]'
  '';
}
