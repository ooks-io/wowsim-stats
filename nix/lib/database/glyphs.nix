let
  mkGlyph = name: spellId: icon: {
    inherit spellId icon;
    name = "Glyph of ${name}";
  };
in {
  # TODO: complete glyph list, currently only supporting those that exist in the current profiles
  ClassDeathKnight = {
    "43533" = mkGlyph "Anti-Magic Shell" 58623 "spell_shadow_antimagicshell";
    "43537" = mkGlyph "Chains of Ice" 58620 "spell_frost_chainsofice";
    "45800" = mkGlyph "Simulacrum" 63331 "spell_holy_consumemagic";
    "104048" = mkGlyph "Regenerative Magic" 146648 "spell_shadow_antimagicshell";
    "104047" = mkGlyph "Loud Horn" 146646 "inv_misc_horn_04";
    "43548" = mkGlyph "Pestilence" 58657 "spell_shadow_plaguecloud";
    "43554" = mkGlyph "Vampiric Blood" 58657 "spell_shadow_lifedrain";
    "43550" = mkGlyph "Army of the Dead" 58669 "spell_deathknight_armyofthedead";
    "45806" = mkGlyph "Tranquil Grip" 63335 "ability_rogue_envelopingshadows";
    "43673" = mkGlyph "Death Gate" 60200 "spell_arcane_teleportundercity";
    "43539" = mkGlyph "Death's Embrace" 58677 "spell_shadow_deathcoil";
  };
  ClassDruid = {
    "40914" = mkGlyph "Healing Touch" 54825 "spell_nature_healingtouch";
    "40906" = mkGlyph "Stampede" 114300 "spell_druid_stampedingroar_cat";
    "40909" = mkGlyph "Rebirth" 54733 "spell_nature_reincarnation";
  };
  ClassHunter = {
    "42909" = mkGlyph "Animal Bond" 20895 "classic_ability_druid_demoralizingroar";
    "42903" = mkGlyph "Deterrence" 56850 "ability_whirlwind";
    "42911" = mkGlyph "Pathfinding" 19560 "ability_hunter_pathfinding2";
    "42914" = mkGlyph "Aimed Shot" 126095 "inv_spear_07";
    "42899" = mkGlyph "Liberation" 132106 "achievement_bg_returnxflags_def_wsg";
  };
  ClassMage = {
    # major
    "44955" = mkGlyph "Arcane Power" 62210 "spell_nature_lightning";
    "42748" = mkGlyph "Rapid Displacement" 146659 "spell_arcane_blink";
    "42746" = mkGlyph "Cone of Cold" 115705 "spell_frost_glacier";
    "42739" = mkGlyph "Combustion" 56368 "spell_fire_sealoffire";
    "63539" = mkGlyph "Inferno Blast" 89926 "spell_mage_infernoblast";
    "42745" = mkGlyph "Splitting Ice" 56377 "spell_frost_frostblast";
    "45736" = mkGlyph "Water Elemental" 63090 "spell_frost_summonwaterelemental_2";
    # minor
    "42743" = mkGlyph "Momentum" 56384 "spell_arcane_blink";
    "63416" = mkGlyph "Rapid Teleportation" 46989 "spell_arcane_portaldalaran";
    "42735" = mkGlyph "Loose Mana" 56363 "inv_misc_gem_sapphire_02";
    "104104" = mkGlyph "Unbound Elemental" 146976 "spell_frost_summonwaterelemental";
    "45739" = mkGlyph "Mirror Image" 45739 "spell_magic_lesserinvisibilty";
  };
  ClassMonk = {
    # major
    "85697" = mkGlyph "Spinning Crane Kick" 120479 "ability_monk_cranekick_new";
    "87900" = mkGlyph "Touch of Karma" 125678 "ability_monk_touchofkarma";
    "85695" = mkGlyph "Zen Meditation" 120477 "ability_monk_zenmeditation";
    # minor
    "90715" = mkGlyph "Blackout Kick" 132005 "ability_monk_blackoutkick";
  };
  ClassPaladin = {
    # major
    "41097" = mkGlyph "Templar's Verdict" 54926 "spell_paladin_templarsverdict";
    "41092" = mkGlyph "Double Jeopardy" 54922 "spell_holy_righteousfury";
    "83107" = mkGlyph "Mass Exorcism" 122028 "spell_holy_righteousfury";
  };
  ClassPriest = {};
  ClassRogue = {
    # major
    "45761" = mkGlyph "Vendetta" 63249 "ability_rogue_deadliness";
    "42970" = mkGlyph "Hemorrhaging Veins" 146631 "spell_holy_sealofsacrifice";
  };
  ClassShaman = {
    # major
    "41539" = mkGlyph "Spiritwalker's Grace" 55446 "spell_shaman_spiritwalkersgrace";
    "71155" = mkGlyph "Lightning Shield" 101052 "spell_nature_lightningshield";
    "41529" = mkGlyph "Fire Elemental Totem" 55455 "spell_fire_elemental_totem";
    "41530" = mkGlyph "Fire Nova" 55450 "spell_shaman_firenova";
  };
  ClassWarlock = {
    # major
    "42472" = mkGlyph "Unstable Affliction" 56233 "spell_shadow_unstableaffliction_3";
    "42470" = mkGlyph "Soulstone" 56231 "inv_misc_orb_04";
    "45785" = mkGlyph "Life Tap" 63320 "spell_shadow_burningspirit";
    "42465" = mkGlyph "Imp Swarm" 104316 "ability_warlock_impoweredimp";
    # minor
    "43389" = mkGlyph "Unending Breath" 58079 "spell_shadow_demonbreath";
  };
  ClassWarrior = {
    # major
    "67482" = mkGlyph "Bull Rush" 94372 "achievement_character_tauren_male";
    "43399" = mkGlyph "Unending Rage" 58098 "ability_warrior_intensifyrage";
    "45792" = mkGlyph "Death From Above" 63325 "ability_heroicleap";
  };
}
