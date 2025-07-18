{self, ...}: {
  perSystem = {
    pkgs,
    self',
    lib,
    ...
  }: let
    inherit (lib) getExe;
  in {
    packages.default = pkgs.stdenvNoCC.mkDerivation {
      pname = "my package";
      version = "0.1.0";
      src = "${self}/src";
      nativeBuildInputs = [];

      buildPhase = "";
      dontInstall = true;
    };
    apps.default = {
      type = "app";
      program = "${getExe self'.packages.default}";
    };
  };
}
