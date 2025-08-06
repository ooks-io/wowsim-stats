{lib, ...}: let
  mobTypeMap = {
    unknown = "MobTypeUnknown";
    beast = "MobTypeBeast";
    demon = "MobTypeDemon";
    dragonkin = "MobTypeDragonkin";
    elemental = "MobTypeElemental";
    giant = "MobTypeGiant";
    humanoid = "MobTypeHumanoid";
    mechanical = "MobTypeMechanical";
    undead = "MobTypeUndead";
  };
  mkTarget = {
    id ? 31146,
    name ? "Raid Target",
    level ? 93,
    mobType ? "mechanical",
    minBaseDamage ? 550000,
    damageSpread ? 0.4,
    swingSpeed ? 2,
    tankIndex ? 0,
    suppressDodge ? false,
    parryHaste ? false,
    dualWield ? false,
    dualWieldPenalty ? false,
    spellSchool ? "SpellSchoolPhysical",
    attackPower ? 650,
    armor ? 24835,
    health ? 120016403,
    targetInputs ? [],
  }: let
    base = {
      inherit id name level minBaseDamage damageSpread swingSpeed spellSchool;
      mobType = mobTypeMap.${mobType} or (throw "Invalid mobType: ${mobType}");
      stats = [
        0
        0
        0
        0
        0
        0
        0
        0
        0
        0
        0
        0 # Positions 0-11
        attackPower # StatAttackPower (12)
        0
        0
        0
        0 # Positions 13-16
        armor # StatArmor (17)
        0 # StatBonusArmor (18)
        health # StatHealth (19)
        0
        0 # StatMana, StatMP5 (20-21)
      ];
    };

    # Filter out default values
    optionalFields =
      lib.filterAttrs (
        _: v:
          if builtins.isBool v
          then v != false
          else if builtins.isList v
          then v != []
          else if builtins.isInt v
          then v != 0
          else true
      ) {
        inherit
          suppressDodge
          parryHaste
          dualWield
          dualWieldPenalty
          targetInputs
          tankIndex
          ;
      };
  in
    base // optionalFields;
in {inherit mkTarget;}
