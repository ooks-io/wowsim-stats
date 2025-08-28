{
  lib,
  pkgs,
  classes,
  encounter,
  inputs,
  ...
}: let
  inherit (lib.sim.simulation) mkSim;
  inherit (lib.sim.raid) mkRaid;
  inherit (lib.sim.encounter) mkEncounter;
  inherit (lib.sim.target) mkTarget;
  inherit (classes) monk shaman paladin;

  testChallengeModeGroup = mkSim {
    type = "SimTypeRaid";
    iterations = 500;
    encounter = mkEncounter {
      targets = [
        # based off Sha of Doubt challenge mode logs
        (mkTarget {
          level = 93;
          mobType = "elemental";
          health = 19600000;
          minBaseDamage = 150000;
        })
      ];
      # irrelevant if useHealth true
      duration = 60;
      durationVariation = 20;

      useHealth = true;
    };
    raid = mkRaid {
      dynamicBuffs = true;
      party1 = [
        monk.brewmaster.template.orc.p1.challengeMode.multiTarget
        shaman.elemental.template.troll.p1.challengeMode.singleTarget
        paladin.retribution.template.blood_elf.p1.challengeMode.singleTarget
        shaman.elemental.template.troll.p1.challengeMode.singleTarget
      ];
    };
  };

  mkTestGroupScript = pkgs.writeShellApplication {
    name = "test-group-sim";
    text = ''
      echo "Generating test group simulation JSON..."
      cat > test-group-sim.json << 'EOF'
        ${testChallengeModeGroup}
      EOF
      echo "Running simulation with wowsimcli..."
      wowsimcli sim --infile test-group-sim.json --outfile test-out.json

      echo "=== Sha of Doubt - Challenge Mode - 19.6M HP ==="
      echo "=== Buffs ==="
      echo $(jq -r '.)
      echo "=== Simulation Results ==="
      echo "Avg Fight Duration: $(jq -r '.avgIterationDuration' test-out.json)s"
      echo "Iterations: $(jq -r '.iterationsDone' test-out.json)"
      echo ""
      echo "=== Group Performance ==="
      echo "Total Raid DPS: $(jq -r '.raidMetrics.dps.avg | floor' test-out.json)"
      echo "DPS Range: $(jq -r '.raidMetrics.dps.min | floor' test-out.json) - $(jq -r '.raidMetrics.dps.max | floor' test-out.json)"
      echo ""
      echo "=== Individual Players ==="
      jq -r '.raidMetrics.parties[0].players[] | select(.name and .name != "Target Dummy 1") | "Player: \(.name) - DPS: \(.dps.avg | floor)"' test-out.json
      echo ""
      echo "Files created: test-group-sim.json (input), test-out.json (output)"
    '';
    runtimeInputs = [pkgs.jq inputs.wowsims.packages.${pkgs.system}.wowsimcli];
  };
in {
  inherit testChallengeModeGroup mkTestGroupScript;
}
