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

  class = "druid";
  spec = "balance";

  raid = mkSim {
    requestId = "raidSimAsync-f2cf5e22118a43c7";
    iterations = 1000;
    player = classes.${class}.${spec}.template.p1.raid.singleTarget;
    buffs = buffs.full;
    debuffs = debuffs.full;
    encounter = encounter.raid.long.singleTarget;
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
