let
  mkTalent = name: spellId: icon: {
    inherit name spellId icon;
  };
in {
  # talent string format: "123456" where each digit represents the choice (1-3) for each tier (1-6)

  ClassDeathKnight = {
    "1" = {
      "1" = mkTalent "Roiling Blood" 108170 "ability_deathknight_roilingblood";
      "2" = mkTalent "Plague Leech" 123693 "ability_creature_disease_02";
      "3" = mkTalent "Unholy Blight" 115989 "spell_shadow_contagion";
    };
    "2" = {
      "1" = mkTalent "Lichborne" 49039 "spell_shadow_raisedead";
      "2" = mkTalent "Anti-Magic Zone" 51052 "spell_deathknight_antimagiczone";
      "3" = mkTalent "Purgatory" 114556 "inv_misc_shadowegg";
    };
    "3" = {
      "1" = mkTalent "Death's Advance" 96268 "spell_shadow_demonicempathy";
      "2" = mkTalent "Chilblains" 50041 "spell_frost_wisp";
      "3" = mkTalent "Asphyxiate" 108194 "ability_deathknight_asphixiate";
    };
    "4" = {
      "1" = mkTalent "Death Pact" 48743 "spell_shadow_deathpact";
      "2" = mkTalent "Death Siphon" 108196 "ability_deathknight_deathsiphon";
      "3" = mkTalent "Conversion" 119975 "ability_deathknight_deathsiphon2";
    };
    "5" = {
      "1" = mkTalent "Blood Tap" 45529 "spell_deathknight_bloodtap";
      "2" = mkTalent "Runic Empowerment" 81229 "inv_misc_rune_10";
      "3" = mkTalent "Runic Corruption" 51462 "spell_shadow_rune";
    };
    "6" = {
      "1" = mkTalent "Gorefiend's Grasp" 108199 "ability_deathknight_aoedeathgrip";
      "2" = mkTalent "Remorseless Winter" 108200 "ability_deathknight_remorselesswinters2";
      "3" = mkTalent "Desecrated Ground" 108201 "ability_deathknight_desecratedground";
    };
  };

  ClassDruid = {
    "1" = {
      "1" = mkTalent "Feline Swiftness" 131768 "spell_druid_tirelesspursuit";
      "2" = mkTalent "Displacer Best" 102280 "spell_druid_displacement";
      "3" = mkTalent "Wild Charge" 102401 "spell_druid_wildcharge";
    };
    "2" = {
      "1" = mkTalent "Ysera's Gift" 145108 "inv_misc_head_dragon_green";
      "2" = mkTalent "Renewal" 108238 "spell_nature_natureblessing";
      "3" = mkTalent "Cenarion Ward" 102351 "ability_druid_naturalperfection";
    };
    "3" = {
      "1" = mkTalent "Faerie Swarm" 106707 "spell_druid_swarm";
      "2" = mkTalent "Mass Entanglement" 102359 "spell_druid_massentanglement";
      "3" = mkTalent "Typhoon" 132469 "ability_druid_typhoon";
    };
    "4" = {
      "1" = mkTalent "Soul of the Forest" 114107 "ability_druid_manatree";
      "2" = mkTalent "Incarnation" 106731 "spell_druid_incarnation";
      "3" = mkTalent "Force of Nature" 106737 "ability_druid_forceofnature";
    };
    "5" = {
      "1" = mkTalent "Disorienting Roar" 99 "classic_ability_druid_demoralizingroar";
      "2" = mkTalent "Ursol's Vortex" 102793 "spell_druid_ursolsvortex";
      "3" = mkTalent "Mighty Bash" 5211 "ability_druid_bash";
    };
    "6" = {
      "1" = mkTalent "Heart of the Wild" 108288 "spell_holy_blessingofagility";
      "2" = mkTalent "Dream of Cenarius" 108373 "ability_druid_dreamstate";
      "3" = mkTalent "Nature's Vigil" 124974 "achievement_zone_feralas";
    };
  };

  ClassHunter = {
    "1" = {
      "1" = mkTalent "Posthaste" 109215 "ability_hunter_posthaste";
      "2" = mkTalent "Narrow Escape" 109298 "inv_misc_web_01";
      "3" = mkTalent "Crouching Tiger, Hidden Chimera" 118675 "ability_hunter_pet_chimera";
    };
    "2" = {
      "1" = mkTalent "Binding Shot" 109248 "spell_shaman_bindelemental";
      "2" = mkTalent "Wyvern Sting" 19386 "inv_spear_02";
      "3" = mkTalent "Intimidation" 19577 "ability_devour";
    };
    "3" = {
      "1" = mkTalent "Exhilaration" 109304 "ability_hunter_onewithnature";
      "2" = mkTalent "Aspect of the Iron Hawk" 109260 "spell_hunter_aspectoftheironhawk";
      "3" = mkTalent "Spirit Bond" 109212 "classic_ability_druid_demoralizingroar";
    };
    "4" = {
      "1" = mkTalent "Fervor" 82726 "ability_hunter_aspectoftheviper";
      "2" = mkTalent "Dire Beast" 120679 "ability_hunter_sickem";
      "3" = mkTalent "Thrill of the Hunt" 109306 "ability_hunter_thrillofthehunt";
    };
    "5" = {
      "1" = mkTalent "A Murder of Crows" 131894 "ability_hunter_murderofcrows";
      "2" = mkTalent "Blink Strikes" 130392 "spell_arcane_arcane04";
      "3" = mkTalent "Lynx Rush" 120697 "ability_hunter_catlikereflexes";
    };
    "6" = {
      "1" = mkTalent "Glaive Toss" 117050 "ability_glaivetoss";
      "2" = mkTalent "Powershot" 109259 "ability_hunter_resistanceisfutile";
      "3" = mkTalent "Barrage" 120360 "ability_hunter_rapidregeneration";
    };
  };

  ClassMage = {
    "1" = {
      "1" = mkTalent "Presence of Mind" 12043 "spell_nature_enchantarmor";
      "2" = mkTalent "Blazing Speed" 108843 "spell_fire_burningspeed";
      "3" = mkTalent "Ice Floes" 108839 "spell_mage_iceflows";
    };
    "2" = {
      "1" = mkTalent "Temporal Shield" 115610 "spell_mage_temporalshield";
      "2" = mkTalent "Flameglow" 140468 "inv_elemental_primal_fire";
      "3" = mkTalent "Ice Barrier" 11426 "spell_ice_lament";
    };
    "3" = {
      "1" = mkTalent "Ring of Frost" 113724 "spell_frost_ring-of-frost";
      "2" = mkTalent "Ice Ward" 111264 "spell_frost_frostward";
      "3" = mkTalent "Frostjaw" 102051 "ability_mage_frostjaw";
    };
    "4" = {
      "1" = mkTalent "Greater Invisibility" 110959 "ability_mage_greaterinvisibility";
      "2" = mkTalent "Cauterize" 86949 "spell_fire_rune";
      "3" = mkTalent "Cold Snap" 11958 "spell_frost_wizardmark";
    };
    "5" = {
      "1" = mkTalent "Nether Tempest" 114923 "spell_mage_nethertempest";
      "2" = mkTalent "Living Bomb" 44457 "ability_mage_livingbomb";
      "3" = mkTalent "Frost Bomb" 112948 "spell_mage_frostbomb";
    };
    "6" = {
      "1" = mkTalent "Invocation" 114003 "spell_arcane_arcane03";
      "2" = mkTalent "Rune of Power" 116011 "spell_mage_runeofpower";
      "3" = mkTalent "Incanter's Ward" 1463 "spell_shadow_detectlesserinvisibility";
    };
  };

  ClassMonk = {
    "1" = {
      "1" = mkTalent "Celerity" 115173 "ability_monk_quipunch";
      "2" = mkTalent "Tiger's Lust" 116841 "ability_monk_tigerslust";
      "3" = mkTalent "Momentum" 115174 "ability_monk_standingkick";
    };
    "2" = {
      "1" = mkTalent "Chi Wave" 115098 "ability_monk_chiwave";
      "2" = mkTalent "Zen Sphere" 124081 "ability_monk_forcesphere";
      "3" = mkTalent "Chi Burst" 123986 "spell_arcane_arcanetorrent";
    };
    "3" = {
      "1" = mkTalent "Power Strikes" 121817 "ability_monk_powerstrikes";
      "2" = mkTalent "Ascension" 115396 "ability_monk_ascension";
      "3" = mkTalent "Chi Brew" 115399 "ability_monk_chibrew";
    };
    "4" = {
      "1" = mkTalent "Ring of Peace" 116844 "spell_monk_ringofpeace";
      "2" = mkTalent "Charging Ox Wave" 119392 "ability_monk_chargingoxwave";
      "3" = mkTalent "Leg Sweep" 119381 "ability_monk_legsweep";
    };
    "5" = {
      "1" = mkTalent "Healing Elixirs" 122280 "ability_monk_jasmineforcetea";
      "2" = mkTalent "Dampen Harm" 122278 "ability_monk_dampenharm";
      "3" = mkTalent "Diffuse Magic" 122783 "spell_monk_diffusemagic";
    };
    "6" = {
      "1" = mkTalent "Rushing Jade Wind" 116847 "ability_monk_rushingjadewind";
      "2" = mkTalent "Invoke Xuen, the White Tiger" 123904 "ability_monk_summontigerstatue";
      "3" = mkTalent "Chi Torpedo" 115008 "ability_monk_quitornado";
    };
  };

  ClassPaladin = {
    "1" = {
      "1" = mkTalent "Speed of Light" 85499 "ability_paladin_speedoflight";
      "2" = mkTalent "Long Arm of the Law" 87172 "ability_paladin_longarmofthelaw";
      "3" = mkTalent "Pursuit of Justice" 26023 "ability_paladin_veneration";
    };
    "2" = {
      "1" = mkTalent "Fist of Justice" 105593 "spell_holy_fistofjustice";
      "2" = mkTalent "Repentance" 20066 "spell_holy_prayerofhealing";
      "3" = mkTalent "Evil is a Point of View" 110301 "ability_paladin_turnevil";
    };
    "3" = {
      "1" = mkTalent "Selfless Healer" 85804 "ability_paladin_gaurdedbythelight";
      "2" = mkTalent "Eternal Flame" 114163 "inv_torch_thrown";
      "3" = mkTalent "Sacred Shield" 20925 "ability_paladin_blessedmending";
    };
    "4" = {
      "1" = mkTalent "Hand of Purity" 114039 "spell_holy_sealofwisdom";
      "2" = mkTalent "Unbreakable Spirit" 114154 "spell_holy_unyieldingfaith";
      "3" = mkTalent "Clemency" 105622 "ability_paladin_clemency";
    };
    "5" = {
      "1" = mkTalent "Holy Avenger" 105809 "ability_paladin_holyavenger";
      "2" = mkTalent "Sanctified Wrath" 53376 "ability_paladin_sanctifiedwrath";
      "3" = mkTalent "Divine Purpose" 86172 "spell_holy_divinepurpose";
    };
    "6" = {
      "1" = mkTalent "Holy Prism" 114165 "spell_paladin_holyprism";
      "2" = mkTalent "Light's Hammer" 114158 "spell_paladin_lightshammer";
      "3" = mkTalent "Execution Sentence" 114157 "spell_paladin_executionsentence";
    };
  };

  ClassPriest = {
    "1" = {
      "1" = mkTalent "Void Tendrils" 108920 "spell_priest_voidtendrils";
      "2" = mkTalent "Psyfiend" 108921 "spell_priest_psyfiend";
      "3" = mkTalent "Dominate Mind" 605 "spell_shadow_shadowworddominate";
    };
    "2" = {
      "1" = mkTalent "Body and Soul" 64129 "spell_holy_symbolofhope";
      "2" = mkTalent "Angelic Feather" 121536 "ability_priest_angelicfeather";
      "3" = mkTalent "Phantasm" 108942 "ability_priest_phantasm";
    };
    "3" = {
      "1" = mkTalent "From Darkness, Comes Light" 109186 "spell_holy_surgeoflight";
      "2" = mkTalent "Mindbender" 123040 "spell_shadow_soulleech_3";
      "3" = mkTalent "Solace and Insanity" 139139 "ability_priest_flashoflight";
    };
    "4" = {
      "1" = mkTalent "Desperate Prayer" 19236 "spell_holy_testoffaith";
      "2" = mkTalent "Spectral Guise" 112833 "spell_priest_spectralguise";
      "3" = mkTalent "Angelic Bulwark" 108945 "ability_priest_angelicbulwark";
    };
    "5" = {
      "1" = mkTalent "Twist of Fate" 109142 "spell_shadow_mindtwisting";
      "2" = mkTalent "Power Infusion" 10060 "spell_holy_powerinfusion";
      "3" = mkTalent "Divine Insight" 109175 "spell_priest_burningwill";
    };
    "6" = {
      "1" = mkTalent "Cascade" 121135 "ability_priest_cascade";
      "2" = mkTalent "Divine Star" 110744 "spell_priest_divinestar";
      "3" = mkTalent "Halo" 120517 "ability_priest_halo";
    };
  };

  ClassRogue = {
    "1" = {
      "1" = mkTalent "Nightstalker" 14062 "ability_stealth";
      "2" = mkTalent "Subterfuge" 108208 "rogue_subterfuge";
      "3" = mkTalent "Shadow Focus" 108209 "rogue_shadowfocus";
    };
    "2" = {
      "1" = mkTalent "Deadly Throw" 26679 "inv_throwingknife_06";
      "2" = mkTalent "Nerve Strike" 108210 "rogue_nerve-_strike";
      "3" = mkTalent "Combat Readiness" 74001 "ability_rogue_combatreadiness";
    };
    "3" = {
      "1" = mkTalent "Cheat Death" 31230 "ability_rogue_cheatdeath";
      "2" = mkTalent "Leeching Poison" 108211 "rogue_leeching_poison";
      "3" = mkTalent "Elusiveness" 79008 "ability_rogue_turnthetables";
    };
    "4" = {
      "1" = mkTalent "Cloak and Dagger" 138106 "ability_rogue_unfairadvantage";
      "2" = mkTalent "Shadowstep" 36554 "ability_rogue_shadowstep";
      "3" = mkTalent "Burst of Speed" 108212 "rogue_burstofspeed";
    };
    "5" = {
      "1" = mkTalent "Prey on the Weak" 131511 "ability_rogue_preyontheweak";
      "2" = mkTalent "Paralytic Poison" 108215 "rogue_paralytic_poison";
      "3" = mkTalent "Dirty Tricks" 108216 "ability_rogue_dirtydeeds";
    };
    "6" = {
      "1" = mkTalent "Shuriken Toss" 114014 "inv_throwingknife_07";
      "2" = mkTalent "Marked for Death" 137619 "achievement_bg_killingblow_berserker";
      "3" = mkTalent "Anticipation" 114015 "ability_rogue_slaughterfromtheshadows";
    };
  };

  ClassShaman = {
    "1" = {
      "1" = mkTalent "Nature's Guardian" 30884 "spell_nature_natureguardian";
      "2" = mkTalent "Stone Bulwark Totem" 108270 "ability_shaman_stonebulwark";
      "3" = mkTalent "Astral Shift" 108271 "ability_shaman_astralshift";
    };
    "2" = {
      "1" = mkTalent "Frozen Power" 63374 "spell_fire_bluecano";
      "2" = mkTalent "Earthgrab Totem" 51485 "spell_nature_stranglevines";
      "3" = mkTalent "Windwalk Totem" 108273 "ability_shaman_windwalktotem";
    };
    "3" = {
      "1" = mkTalent "Call of the Elements" 108285 "ability_shaman_multitotemactivation";
      "2" = mkTalent "Totemic Persistence" 108284 "ability_shaman_totemcooldownrefund";
      "3" = mkTalent "Totemic Projection" 108287 "ability_shaman_totemrelocation";
    };
    "4" = {
      "1" = mkTalent "Elemental Mastery" 16166 "spell_nature_wispheal";
      "2" = mkTalent "Ancestral Swiftness" 16188 "spell_shaman_elementaloath";
      "3" = mkTalent "Echo of the Elements" 108283 "ability_shaman_echooftheelements";
    };
    "5" = {
      "1" = mkTalent "Rushing Streams" 147074 "inv_spear_04";
      "2" = mkTalent "Ancestral Guidance" 108281 "ability_shaman_ancestralguidance";
      "3" = mkTalent "Conductivity" 108282 "ability_shaman_fortifyingwaters";
    };
    "6" = {
      "1" = mkTalent "Unleashed Fury" 117012 "shaman_talent_unleashedfury";
      "2" = mkTalent "Primal Elementalist" 117013 "shaman_talent_primalelementalist";
      "3" = mkTalent "Elemental Blast" 117014 "shaman_talent_elementalblast";
    };
  };

  ClassWarlock = {
    "1" = {
      "1" = mkTalent "Dark Regeneration" 108359 "spell_warlock_darkregeneration";
      "2" = mkTalent "Soul Leech" 108370 "warlock_siphonlife";
      "3" = mkTalent "Harvest Life" 108371 "spell_warlock_harvestoflife";
    };
    "2" = {
      "1" = mkTalent "Demonic Breath" 47897 "ability_warlock_shadowflame";
      "2" = mkTalent "Mortal Coil" 6789 "ability_warlock_mortalcoil";
      "3" = mkTalent "Shadowfury" 30283 "ability_warlock_shadowfurytga";
    };
    "3" = {
      "1" = mkTalent "Soul Link" 108415 "ability_warlock_soullink";
      "2" = mkTalent "Sacrificial Pact" 108416 "warlock_sacrificial_pact";
      "3" = mkTalent "Dark Bargain" 110913 "ability_deathwing_bloodcorruption_death";
    };
    "4" = {
      "1" = mkTalent "Blood Horror" 111397 "ability_deathwing_bloodcorruption_earth";
      "2" = mkTalent "Burning Rush" 111400 "ability_deathwing_sealarmorbreachtga";
      "3" = mkTalent "Unbound Will" 108482 "warlock_spelldrain";
    };
    "5" = {
      "1" = mkTalent "Grimoire of Supremacy" 108499 "warlock_grimoireofcommand";
      "2" = mkTalent "Grimoire of Service" 108501 "warlock_grimoireofservice";
      "3" = mkTalent "Grimoire of Sacrifice" 108503 "warlock_grimoireofsacrifice";
    };
    "6" = {
      "1" = mkTalent "Archimonde's Darkness" 108505 "achievement_boss_archimonde-";
      "2" = mkTalent "Kil'jaeden's Cunning" 137587 "achievement_boss_kiljaedan";
      "3" = mkTalent "Mannoroth's Fury" 108508 "achievement_boss_magtheridon";
    };
  };

  ClassWarrior = {
    "1" = {
      "1" = mkTalent "Juggernaut" 103826 "ability_warrior_bullrush";
      "2" = mkTalent "Double Time" 103827 "inv_misc_horn_04";
      "3" = mkTalent "Warbringer" 103828 "ability_warrior_warbringer";
    };
    "2" = {
      "1" = mkTalent "Enraged Regeneration" 55694 "ability_warrior_focusedrage";
      "2" = mkTalent "Second Wind" 29838 "ability_hunter_harass";
      "3" = mkTalent "Impending Victory" 103840 "spell_impending_victory";
    };
    "3" = {
      "1" = mkTalent "Staggering Shout" 107566 "ability_bullrush";
      "2" = mkTalent "Piercing Howl" 12323 "spell_shadow_deathscream";
      "3" = mkTalent "Disrupting Shout" 102060 "warrior_disruptingshout";
    };
    "4" = {
      "1" = mkTalent "Bladestorm" 46924 "ability_warrior_bladestorm";
      "2" = mkTalent "Shockwave" 46968 "ability_warrior_shockwave";
      "3" = mkTalent "Dragon Roar" 118000 "ability_warrior_dragonroar";
    };
    "5" = {
      "1" = mkTalent "Mass Spell Reflection" 114028 "ability_warrior_shieldbreak";
      "2" = mkTalent "Safeguard" 114029 "ability_warrior_safeguard";
      "3" = mkTalent "Vigilance" 114030 "ability_warrior_vigilance";
    };
    "6" = {
      "1" = mkTalent "Avatar" 107574 "warrior_talent_icon_avatar";
      "2" = mkTalent "Bloodbath" 12292 "ability_warrior_bloodbath";
      "3" = mkTalent "Storm Bolt" 107570 "warrior_talent_icon_stormbolt";
    };
  };
}
