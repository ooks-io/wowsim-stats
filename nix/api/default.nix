let
  realm = import ./realm.nix;
in {
  flake.api = {
    inherit realm;
  };
  _module.args.api = {
    inherit realm;
  };
}
