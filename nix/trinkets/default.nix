let
  # aglity
  terrorInTheMists = {
    celestial = 86890;
    normal = 86332;
    heroic = 87167;
  };
  bottleOfInfiniteStars = {
    celestial = 86791;
    normal = 86132;
    heroic = 87057;
  };
  jadeBanditFigurine = {
    normal = 86043;
    celestial = 86772;
  };
  # rep
  hawkmastersTelon = 89082;
  # pvp
  insigniaOfConquest = {
    conquest = 84935;
    honor = 84349;
  };
  badgeOfConquest = {
    conquest = 84934;
    honor = 84344;
  };
  # heroic dungeons
  flashingSteelTalisman = 81265;
  windsweptPages = 81125;
  searingWords = 81267;

  # intellect

  essenceOfTerror = {
    celestial = 86907;
    normal = 86388;
    heroic = 87175;
  };

  lightOfTheCosmos = {
    celestial = 86792;
    normal = 86133;
    heroic = 87065;
  };

  jadeMagistrateFigurine = {
    celestial = 86773;
    normal = 86044;
  };

  blossomOfPureSnow = 89081;

  badgeOfDominiance = {
    conquest = 84940;
    honor = 84488;
  };

  insigniaOfDominance = {
    conquest = 84941;
    honor = 84489;
  };

  mithrilWristwatch = 87572;
  visionOfThePredator = 81192;
  flashfrozenResinGlobule = 81263;

  relicOfYulon = 79331;

  # strength

  darkmistVortex = {
    celestial = 86894;
    normal = 86336;
    heroic = 87172;
  };

  leiShensFinalOrders = {
    celestial = 86802;
    normal = 86144;
    heroic = 87072;
  };

  jadeCharioteerFigurine = {
    celestial = 86771;
    normal = 86042;
  };

  ironBellyWok = 89083;

  insigniaOfVictory = {
    conquest = 84937;
    honor = 84495;
  };

  badgeOfVictory = {
    conquest = 84942;
    honor = 84490;
  };

  carbonicCarbuncle = 81138;
  lessonOfTheDarkmaster = 81268;

  # spirit

  relicOfChiji = 79330;
  priceOfProgress = 81266;
  emptyFruitBarrel = 81133;
  vialOfIchorousBlood = 81264;

  spiritsOfTheSun = {
    celestial = 86885;
    normal = 86327;
    heroic = 87163;
  };

  qinXiPolarizingSeal = {
    celestial = 86805;
    normal = 86147;
    heroic = 87075;
  };

  scrollOfReveredAncestors = 89080;

  thousandYearPickledEgg = 87573;

  jadeCourtesanFigurine = {
    celestial = 86774;
    normal = 86045;
  };

  # neutral
  zenAlchemistStone = 75274;
  jadeWarlordFigurine = {
    celestial = 86775;
    normal = 86046;
  };
  laoChinsLiquidCourage = 89079;
  quilenStatuette = 89611;
  # agility and strength
  corensColdChromiumCoaster = 87574;

  relicOfXuen = {
    agility = 79328;
    strength = 79327;
  };

  trinket = {
    presets = {
      p1 = {
        agility = [
          terrorInTheMists.celestial
          terrorInTheMists.normal
          terrorInTheMists.heroic
          bottleOfInfiniteStars.celestial
          bottleOfInfiniteStars.normal
          bottleOfInfiniteStars.heroic
          jadeWarlordFigurine.celestial
          jadeWarlordFigurine.normal
          jadeBanditFigurine.celestial
          jadeBanditFigurine.normal
          insigniaOfConquest.conquest
          insigniaOfConquest.honor
          badgeOfConquest.conquest
          badgeOfConquest.honor
          relicOfXuen.agility
          hawkmastersTelon
          flashingSteelTalisman
          windsweptPages
          searingWords
          zenAlchemistStone
          quilenStatuette
          corensColdChromiumCoaster
          laoChinsLiquidCourage
        ];
        intellect = [
          essenceOfTerror.celestial
          essenceOfTerror.normal
          essenceOfTerror.heroic
          lightOfTheCosmos.celestial
          lightOfTheCosmos.normal
          lightOfTheCosmos.heroic
          jadeMagistrateFigurine.celestial
          jadeMagistrateFigurine.normal
          blossomOfPureSnow
          badgeOfDominiance.conquest
          badgeOfDominiance.honor
          insigniaOfDominance.conquest
          insigniaOfDominance.honor
          mithrilWristwatch
          visionOfThePredator
          flashfrozenResinGlobule
          zenAlchemistStone
          laoChinsLiquidCourage
          jadeWarlordFigurine.celestial
          jadeWarlordFigurine.normal
          quilenStatuette
          relicOfYulon
        ];
        intellectHybrid =
          trinket.presets.p1.intellect
          ++ [
            priceOfProgress
            emptyFruitBarrel
            vialOfIchorousBlood
            spiritsOfTheSun.celestial
            spiritsOfTheSun.normal
            spiritsOfTheSun.heroic
            qinXiPolarizingSeal.celestial
            qinXiPolarizingSeal.normal
            qinXiPolarizingSeal.heroic
            scrollOfReveredAncestors
            thousandYearPickledEgg
            jadeCourtesanFigurine.celestial
            jadeCourtesanFigurine.normal
            relicOfChiji
          ];
        strength = [
          darkmistVortex.celestial
          darkmistVortex.normal
          darkmistVortex.heroic
          leiShensFinalOrders.celestial
          leiShensFinalOrders.normal
          leiShensFinalOrders.heroic
          jadeCharioteerFigurine.celestial
          jadeCharioteerFigurine.normal
          ironBellyWok
          insigniaOfVictory.conquest
          insigniaOfVictory.honor
          badgeOfVictory.conquest
          badgeOfVictory.honor
          carbonicCarbuncle
          lessonOfTheDarkmaster
          relicOfXuen.strength
          zenAlchemistStone
          quilenStatuette
          corensColdChromiumCoaster
          laoChinsLiquidCourage
          jadeWarlordFigurine.celestial
          jadeWarlordFigurine.normal
        ];
      };
    };
  };
in {
  flake.trinket = trinket;
  _module.args.trinket = trinket;
}
