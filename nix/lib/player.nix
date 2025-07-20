{
  lib,
  inputs,
  ...
}: let
  inherit (lib.strings) toSentenceCase concatMapStrings toUpper substring;
  inherit (builtins) filter isString split readFile fromJSON;
  inherit (inputs) wowsims;

  # convert string to pascal case
  # credit to: https://github.com/LeoLuxo/dots/blob/main/lib/strings.nix
  toUpperCaseFirstLetter = string: let
    head = toUpper (substring 0 1 string);
    tail = substring 1 (-1) string;
  in
    head + tail;

  splitWords = string:
    filter isString (
      split "[ _-]" string
    );

  toPascalCase = string: concatMapStrings toUpperCaseFirstLetter (splitWords string);

  mkPlayer = {
    name ? "Player",
    race,
    class,
    spec,
    gearset,
    consumables,
    options ? {},
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
    inherit equipment name consumables glyphs rotation reactionTimeMs distanceFromTarget;
    race = toPascalCase "Race_${race}";
    class = toPascalCase "Class_${class}";
    profession1 = toSentenceCase profession1;
    profession2 = toSentenceCase profession2;
    talentsString = talents;

    "${lib.toCamelCase "${spec}-${class}"}" = {
      inherit options;
    };

    # TODO
    cooldowns = {};
    healingModel = {};
  };
in {inherit mkPlayer;}
