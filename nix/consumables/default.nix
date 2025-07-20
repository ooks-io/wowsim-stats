let
  # TODO: implement a system for elixirs.
  consumables = {
    potion = {
      verminsBite = 76089;
      agility = consumables.potion.verminsBite;

      jadeSerpent = 76093;
      intellect = consumables.potion.jadeSerpent;

      moguPower = 76095;
      strength = consumables.potion.moguPower;

      mightyRage = 13442;
      rage = consumables.potion.mightyRage;

      snapRootTuber = 91803;
      haste = consumables.potion.snapRootTuber;

      mountains = 76090;
      armor = consumables.potion.mountains;
    };

    flask = {
      springBlossom = 76084;
      agility = consumables.flask.springBlossom;

      wintersBite = 76088;
      strength = consumables.flask.wintersBite;

      warmSun = 76085;
      intellect = consumables.flask.warmSun;

      earth = 76087;
      stamina = consumables.flask.earth;
    };

    food = {
      seaMistRiceNoodles = 74648;
      agility = consumables.food.seaMistRiceNoodles;

      moguFishStew = 74650;
      intellect = consumables.food.moguFishStew;

      blackPepperRibsAndShrimp = 74646;
      strength = consumables.food.blackPepperRibsAndShrimp;

      greenFishCurry = 81410;
      crit = consumables.food.greenFishCurry;

      spicySalmon = 86073;
      hit = consumables.food.spicySalmon;

      spicyVegetableChips = 86074;
      expertise = consumables.food.spicyVegetableChips;

      mangoIce = 101745;
      mastery = consumables.food.mangoIce;

      chunTianSpringRolls = 74656;
      stamina = consumables.food.chunTianSpringRolls;
    };

    preset = {
      agility = {
        prepotId = consumables.potion.agility;
        potId = consumables.potion.agility;
        flaskId = consumables.flask.agility;
        foodId = consumables.food.agility;
      };
      strength = {
        prepotId = consumables.potion.strength;
        potId = consumables.potion.strength;
        flaskId = consumables.flask.strength;
        foodId = consumables.food.strength;
      };
      intellect = {
        prepotId = consumables.potion.intellect;
        potId = consumables.potion.intellect;
        flaskId = consumables.flask.intellect;
        foodId = consumables.food.intellect;
      };
      tank = {
        prepotId = consumables.potion.armor;
        potId = consumables.potion.armor;
        flaskId = consumables.flask.stamina;
        foodId = consumables.food.stamina;
      };
    };
  };
in {
  flake.consumables = consumables;
  _module.args.consumables = consumables;
}
