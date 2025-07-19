let
  # TODO, dynamically set buffs/debuffs based on specs present
  mkRaid = {
    parties,
    buffs,
    debuffs,
    targetDummies,
  }: {
    inherit parties buffs debuffs targetDummies;
  };
in
  mkRaid
