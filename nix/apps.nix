{
  lib,
  classes,
  encounter,
  buffs,
  debuffs,
  inputs,
  ...
}: {
  perSystem = {pkgs, ...}: let
    # Test composition script
    inherit (lib.sim.simulation) mkSim;

    class = "druid";
    spec = "balance";

    testRaid = mkSim {
      requestId = "raidSimAsync-f2cf5e22118a43c7";
      iterations = 1000;
      player = classes.${class}.${spec}.template.p1.raid.singleTarget;
      buffs = buffs.full;
      debuffs = debuffs.full;
      encounter = encounter.raid.long.singleTarget;
    };
    testComposition = pkgs.writeShellApplication {
      name = "test-composition";
      text = ''
        # Generate test elemental simulation
        cat > ${spec}_input.json << 'EOF'
        ${testRaid}
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
      runtimeInputs = [pkgs.jq inputs.wowsims.packages.${pkgs.system}.wowsimcli];
    };

    # Import mass simulation
    massSimulation = import ./mass-simulation.nix {
      inherit lib pkgs classes encounter buffs debuffs inputs;
    };

    # Mass simulations using mkMassSim
    mkMassSim = (import ./lib/mkMassSim.nix {inherit lib pkgs classes encounter buffs debuffs inputs;}).mkMassSim;

    massSimulations = {
      singleTargetRaidLong = mkMassSim {
        specs = "dps"; # shortcut to all DPS classes templates
        encounter = encounter.raid.long.singleTarget;
        iterations = 10000;
        output = "singleTargetDPSraidp1long";
      };

      multiTargetRaidLong = mkMassSim {
        specs = "dps";
        encounter = encounter.raid.long.multiTarget;
        iterations = 10000;
        output = "multiTargetDPSraidp1long";
      };

      cleaveRaidLong = mkMassSim {
        specs = "dps";
        encounter = encounter.raid.long.cleave;
        iterations = 10000;
        output = "cleaveDPSraidp1long";
      };
    };
  in {
    apps =
      {
        test-composition = {
          type = "app";
          program = "${testComposition}/bin/test-composition";
        };
        mass-sim = {
          type = "app";
          program = "${massSimulation.massSimulationScript}/bin/mass-simulation";
        };
      }
      // (lib.mapAttrs (name: massSim: {
          type = "app";
          program = "${massSim.script}/bin/${massSim.metadata.output}-aggregator";
        })
        massSimulations);
  };
}
