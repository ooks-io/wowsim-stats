{
  lib,
  inputs,
  ...
}: let
  inherit (lib.string) toSentenceCase;
  inherit (inputs) wowsims;
  inherit (builtins) readFile fromJSON;
  mkPlayer = {
    name ? "Player",
    race,
    class,
    spec,
    gearset,
    consumables,
    options,
    talents,
    glyphs,
    profession1,
    profession2,
    apl,
    distanceFromTarget,
    challengeMode ? false,
    reactionTimeMs ? 100,
  }: let
    baseEquipment = fromJSON (readFile "${wowsims}/ui/${class}/${spec}/gear_sets/${gearset}.gear.json");
    equipment =
      if challengeMode
      then
        baseEquipment
        // {
          items = map (item: item // {challengeMode = true;}) baseEquipment.items;
        }
      else baseEquipment;

    rotation = fromJSON (readFile "${wowsims}/ui/${class}/${spec}/apls/${apl}.apl.json");
  in {
    inherit equipment name consumables glyphs profession1 profession2 rotation reactionTimeMs distanceFromTarget;
    race = "Race${toSentenceCase race}";
    class = "Class${toSentenceCase class}";
    talentString = talents;

    "${spec}${toSentenceCase class}" = {
      inherit options;
    };

    # TODO
    cooldowns = {};
    healingModel = {};
  };
in
  mkPlayer
