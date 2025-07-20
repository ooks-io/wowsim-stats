{
  lib,
  consumables,
  ...
}: let
  inherit (lib.sim.player) mkPlayer;
  inherit (consumables.preset) strength;

  mkFury = {
    race,
    apl ? "default",
    gearset ? "p1_fury_tg",
    talents,
    consumables ? strength,
    profession1 ? "engineering",
    profession2 ? "blacksmithing",
    distanceFromTarget ? 15,
  }:
    mkPlayer {
      class = "warrior";
      spec = "fury";
      options = {};
      inherit race gearset talents apl consumables profession1 profession2 distanceFromTarget;
      glyphs = {
        major1 = 67482;
        major2 = 45792;
        major3 = 43399;
      };
    };

  fury = {
    # Talent configurations
    talents = {
      stormBolt = "113133";
      avatar = "113131";
    };

    p1 = {
      singleTarget = mkFury {
        race = "orc";
        talents = fury.talents.stormBolt;
      };
      aoe = mkFury {
        race = "orc";
        talents = fury.talents.stormBolt;
      };
    };
  };
in
  fury
