{inputs, ...} @ args:
inputs.nixpkgs.lib.extend (self: _super: {
  sim = import ./. {
    inherit (args) inputs self;
    lib = self;
  };
})
