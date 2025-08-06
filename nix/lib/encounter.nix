let
  mkEncounter = {
    duration ? 300,
    durationVariation ? 60,
    targets ? [],
    useHealth ? false,
  }: {
    apiVersion = 1;
    inherit useHealth duration durationVariation targets;

    # default execute config
    executeProportion20 = 0.2;
    executeProportion25 = 0.25;
    executeProportion35 = 0.35;
    executeProportion45 = 0.45;
    executeProportion90 = 0.9;
  };
in {inherit mkEncounter;}
