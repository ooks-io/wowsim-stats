{
  writeShellApplication,
  wowsimcli,
  jq,
  self,
  lib,
  ...
}: let
  classConfigs = import ./utils/classConfigs.nix {inherit lib self;};

  # Test the new generateSimulation function
  testInfile = classConfigs.generateSimulation 
    classConfigs.monk.windwalker.singleTarget 
    classConfigs.encounters.long.singleTarget;
    
  # Test challenge mode
  testChallengeModeInfile = classConfigs.generateSimulation 
    classConfigs.monk.windwalker.challengeMode 
    classConfigs.encounters.long.singleTarget;
in
  writeShellApplication {
    name = "generate-rankings";
    runtimeInputs = [
      wowsimcli
      jq
    ];
    text = ''
      set -euo pipefail

      echo "Testing class configuration system..."

      # Generate different configurations
      echo "1. Monk Windwalker Single Target (Long):"
      cat > "./monk_windwalker_st_long.json" << 'EOF'
        ${testInfile}
      EOF
      
      echo "✅ Generated: monk_windwalker_st_long.json"
      echo "File size: $(du -h "./monk_windwalker_st_long.json" | cut -f1)"
      
      echo ""
      echo "2. Monk Windwalker Challenge Mode:"
      cat > "./monk_windwalker_challenge_mode.json" << 'EOF'
        ${testChallengeModeInfile}
      EOF
      
      echo "✅ Generated: monk_windwalker_challenge_mode.json"
      echo "File size: $(du -h "./monk_windwalker_challenge_mode.json" | cut -f1)"

      # Test JSON validation
      echo ""
      echo "JSON validation:"
      if jq empty "./monk_windwalker_st_long.json" 2>/dev/null && jq empty "./monk_windwalker_challenge_mode.json" 2>/dev/null; then
        echo "✅ Both files have valid JSON"
      else
        echo "❌ Invalid JSON found"
        echo "Standard mode errors:"
        jq empty "./monk_windwalker_st_long.json"
        echo "Challenge mode errors:"
        jq empty "./monk_windwalker_challenge_mode.json"
      fi

      echo ""
      echo "Configuration demonstrates:"
      echo "Standard Mode:"
      echo "- Class: $(jq -r '.raid.parties[0].players[0].class' "./monk_windwalker_st_long.json")"
      echo "- Spec field: $(jq -r '. | keys | .[] | select(. == "windwalkerMonk")' "./monk_windwalker_st_long.json" || echo "windwalkerMonk")"
      echo "- Encounter duration: $(jq -r '.encounter.duration' "./monk_windwalker_st_long.json")s"
      echo "- Target count: $(jq -r '.encounter.targets | length' "./monk_windwalker_st_long.json")"
      echo "- Raid buffs enabled: $(jq -r '.raid.buffs | [to_entries[] | select(.value == true) | .key] | length' "./monk_windwalker_st_long.json")"
      echo "- Challenge mode items: $(jq -r '.raid.parties[0].players[0].equipment.items | map(select(.challengeMode == true)) | length' "./monk_windwalker_st_long.json")"
      echo ""
      echo "Challenge Mode:"
      echo "- Challenge mode items: $(jq -r '.raid.parties[0].players[0].equipment.items | map(select(.challengeMode == true)) | length' "./monk_windwalker_challenge_mode.json")"
      echo "- Total equipment items: $(jq -r '.raid.parties[0].players[0].equipment.items | length' "./monk_windwalker_challenge_mode.json")"

      echo ""
      echo "Usage examples:"
      echo "  classConfigs.generateSimulation classConfigs.monk.windwalker.singleTarget classConfigs.encounters.long.singleTarget"
      echo "  classConfigs.generateSimulation classConfigs.monk.windwalker.multiTarget classConfigs.encounters.short.multiTarget8"
      echo "  classConfigs.generateSimulation classConfigs.monk.windwalker.challengeMode classConfigs.encounters.long.singleTarget"
      echo "  classConfigs.generateSimulation classConfigs.monk.windwalker.challengeModeMultiTarget classConfigs.encounters.short.multiTarget8"
      echo ""
      echo "Available encounters:"
      echo "  - encounters.long.singleTarget"
      echo "  - encounters.long.multiTarget2"
      echo "  - encounters.long.multiTarget8"
      echo "  - encounters.short.singleTarget"
      echo "  - encounters.short.multiTarget2"
      echo "  - encounters.short.multiTarget8"
      echo "  - encounters.patchwerk.long"
      echo "  - encounters.cleave.short"
      echo "  - encounters.aoe.long"
      echo ""
      echo "Available raid buff presets:"
      echo "  - raidBuffs.fullBuffs (all buffs including bloodlust)"
      echo "  - raidBuffs.fullBuffsNoLust (all buffs except bloodlust)"
      echo "  - raidBuffs.noBuffs (no buffs)"
      echo "  - raidBuffs.mergeRaidBuffs {bloodlust = false; trueshotAura = false;} (custom)"
      echo ""
      echo "Available raid debuff presets:"
      echo "  - raidDebuffs.fullDebuffs (all debuffs)"
      echo "  - raidDebuffs.noDebuffs (no debuffs)"
      echo "  - raidDebuffs.mergeRaidDebuffs {physicalVulnerability = false;} (custom)"
    '';
  }
