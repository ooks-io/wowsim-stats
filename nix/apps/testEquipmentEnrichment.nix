{
  inputs,
  writeShellApplication,
  lib,
  ...
}:

writeShellApplication {
  name = "testEquipmentEnrichment";
  text = let
    itemDb = lib.sim.itemDatabase;
    
    # Create test equipment similar to what might be in death_knight frost
    testEquipment = {
      items = [
        { id = 87126; enchant = 4804; gems = [76884 76680]; reforging = 167; }
        { id = 87166; enchant = 4444; gems = [89873]; reforging = 158; }
      ];
    };
    
    enrichedEquipment = itemDb.enrichEquipment testEquipment;
    enrichedEquipmentJson = builtins.toJSON enrichedEquipment;
    
  in ''
    echo "=== Testing Equipment Enrichment in Bash ==="
    echo ""
    
    # Test the exact pattern used in mkMassSim
    enrichedEquipment='${enrichedEquipmentJson}'
    
    echo "Success! No bash syntax errors with enriched equipment JSON."
    echo "Equipment has $(echo "$enrichedEquipment" | jq '.items | length') items"
    
    # Test the jq command structure
    echo '{"equipment": {}}' | jq --argjson enrichedEquipment "$enrichedEquipment" '{
      equipment: $enrichedEquipment
    }' > /dev/null
    
    echo "jq command also succeeded!"
  '';
}