{
  wowsimcli,
  jq,
  writeShellApplication,
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  ...
}: let
  # quickly sim a single spec for testing purposes
  inherit (lib.sim.simulation) mkSim;

  class = "rogue";
  spec = "subtlety";
  encounterType = "raid";

  raid = mkSim {
    requestId = "raidSimAsync-c8e6f5783f4f390b";
    iterations = 1000;
    player = classes.${class}.${spec}.template.troll.p1.${encounterType}.singleTarget;
    buffs = buffs.full;
    debuffs = debuffs.full;
    encounter = encounter.${encounterType}.long.singleTarget;
  };
  testRaid = writeShellApplication {
    name = "testRaid";
    text = ''
      cat > ${spec}_input.json << 'EOF'
      ${raid}
      EOF

      echo "Generated ${spec}_input.json"
      echo "File size: $(wc -c < ${spec}_input.json) bytes"
      echo "Player name: $(jq -r '.raid.parties[0].players[0].name // "missing"' ${spec}_input.json)"
      echo ""

      echo "Running wowsimcli simulation..."
      wowsimcli sim --infile ${spec}_input.json --outfile ${spec}_output.json

      if [ -f ${spec}_output.json ]; then
        echo "Simulation completed successfully!"
        avgDps=$(jq -r '.raidMetrics.dps.avg' ${spec}_output.json)
        echo "Average DPS: $avgDps"
      else
        echo "Error: wowsimcli failed to generate output"
        exit 1
      fi
    '';
    runtimeInputs = [jq wowsimcli];
  };
in
  testRaid
