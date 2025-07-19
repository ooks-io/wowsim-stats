{lib, ...}: let
  # TODO: implement a system for elixirs.
  consumable = {
    potion = {
      verminsBite = 76089;
      agility = consumable.potion.verminsBite;

      jadeSerpent = 76093;
      intellect = consumable.potion.jadeSerpent;

      moguPower = 76095;
      strength = consumable.potion.moguPower;

      mightyRage = 13442;
      rage = consumable.potion.mightyRage;

      snapRootTuber = 91803;
      haste = consumable.potion.snapRootTuber;

      mountains = 76090;
      armor = consumable.potion.mountains;
    };

    flask = {
      springBlossom = 76084;
      agility = consumable.flask.springBlossom;

      wintersBite = 76088;
      strength = consumable.flask.wintersBite;

      warmSun = 76085;
      intellect = consumable.flask.warmSun;

      earth = 76087;
      stamina = consumable.flask.earth;
    };

    food = {
      seaMistRiceNoodles = 74648;
      agility = consumable.food.seaMistRiceNoodles;

      moguFishStew = 74650;
      intellect = consumable.food.moguFishStew;

      blackPepperRibsAndShrimp = 74646;
      strength = consumable.food.blackPepperRibsAndShrimp;

      greenFishCurry = 81410;
      crit = consumable.food.greenFishCurry;

      spicySalmon = 86073;
      hit = consumable.food.spicySalmon;

      spicyVegetableChips = 86074;
      expertise = consumable.food.spicyVegetableChips;

      mangoIce = 101745;
      mastery = consumable.food.mangoIce;

      chunTianSpringRolls = 74656;
      stamina = consumable.food.chunTianSpringRolls;
    };

    preset = {
      agility = {
        prepotId = consumable.potion.agility;
        potId = consumable.potion.agility;
        flaskId = consumable.flask.agility;
        foodId = consumable.food.agility;
      };
      strength = {
        prepotId = consumable.potion.strength;
        potId = consumable.potion.strength;
        flaskId = consumable.flask.strength;
        foodId = consumable.food.strength;
      };
      intellect = {
        prepotId = consumable.potion.intellect;
        potId = consumable.potion.intellect;
        flaskId = consumable.flask.intellect;
        foodId = consumable.food.intellect;
      };
      tank = {
        prepotId = consumable.potion.armor;
        potId = consumable.potion.armor;
        flaskId = consumable.flask.stamina;
        foodId = consumable.food.stamina;
      };
    };
  };
in
  consumable

