let
  mkRealm = id: region: name: {
    inherit id region name;
  };
in {
  # US Realms
  atiesh = mkRealm 4372 "us" "Atiesh";
  myzrael = mkRealm 4373 "us" "Myzrael";
  "old-blanchy" = mkRealm 4374 "us" "Old Blanchy";
  azuresong = mkRealm 4376 "us" "Azuresong";
  mankrik = mkRealm 4384 "us" "Mankrik";
  pagle = mkRealm 4385 "us" "Pagle";
  ashkandi = mkRealm 4387 "us" "Ashkandi";
  westfall = mkRealm 4388 "us" "Westfall";
  whitemane = mkRealm 4395 "us" "Whitemane";
  faerlina = mkRealm 4408 "us" "Faerlina";
  grobbulus = mkRealm 4647 "us" "Grobbulus";
  "bloodsail-buccaneers" = mkRealm 4648 "us" "Bloodsail Buccaneers";
  remulos = mkRealm 4667 "us" "Remulos";
  arugal = mkRealm 4669 "us" "Arugal";
  yojamba = mkRealm 4670 "us" "Yojamba";
  skyfury = mkRealm 4725 "us" "Skyfury";
  sulfuras = mkRealm 4726 "us" "Sulfuras";
  windseeker = mkRealm 4727 "us" "Windseeker";
  benediction = mkRealm 4728 "us" "Benediction";
  earthfury = mkRealm 4731 "us" "Earthfury";
  maladath = mkRealm 4738 "us" "Maladath";
  angerforge = mkRealm 4795 "us" "Angerforge";
  eranikus = mkRealm 4800 "us" "Eranikus";

  # EU Realms - Smart merging enabled to preserve historical data
  everlook = mkRealm 4440 "eu" "Everlook";
  auberdine = mkRealm 4441 "eu" "Auberdine";
  lakeshire = mkRealm 4442 "eu" "Lakeshire";
  chromie = mkRealm 4452 "eu" "Chromie";
  "pyrewood-village" = mkRealm 4453 "eu" "Pyrewood Village";
  "mirage-raceway" = mkRealm 4454 "eu" "Mirage Raceway";
  razorfen = mkRealm 4455 "eu" "Razorfen";
  "nethergarde-keep" = mkRealm 4456 "eu" "Nethergarde Keep";
  sulfuron = mkRealm 4464 "eu" "Sulfuron";
  golemagg = mkRealm 4465 "eu" "Golemagg";
  patchwerk = mkRealm 4466 "eu" "Patchwerk";
  firemaw = mkRealm 4467 "eu" "Firemaw";
  flamegor = mkRealm 4474 "eu" "Flamegor";
  gehennas = mkRealm 4476 "eu" "Gehennas";
  venoxis = mkRealm 4477 "eu" "Venoxis";
  "hydraxian-waterlords" = mkRealm 4678 "eu" "Hydraxian Waterlords";
  mograine = mkRealm 4701 "eu" "Mograine";
  amnennar = mkRealm 4703 "eu" "Amnennar";
  ashbringer = mkRealm 4742 "eu" "Ashbringer";
  transcendence = mkRealm 4745 "eu" "Transcendence";
  earthshaker = mkRealm 4749 "eu" "Earthshaker";
  giantstalker = mkRealm 4811 "eu" "Giantstalker";
  mandokir = mkRealm 4813 "eu" "Mandokir";
  thekal = mkRealm 4815 "eu" "Thekal";
  "jindo" = mkRealm 4816 "eu" "Jin'do";

  # kr
  "shimmering-flats" = mkRealm 4417 "kr" "Shimmering Flats";
  lokholar = mkRealm 4419 "kr" "Lokholar";
  iceblood = mkRealm 4420 "kr" "Iceblood";
  ragnaros = mkRealm 4421 "kr" "Ragnaros";
  frostmourne = mkRealm 4840 "kr" "Frostmourne";
}
