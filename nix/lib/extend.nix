{inputs, ...} @ args:
inputs.nixpkgs.lib.extend (self: super: {
  sim = import ./. {
    inherit (args) inputs self;
    lib = self;
  };
})
