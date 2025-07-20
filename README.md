WIP. [wowsim-stats](https://ooks-io.github.io/wowsim-stats/)

This tool uses [wowsims](https://github.com/wowsims/mop).

Planned Features

- Leaderboard for multiple encounters
  - Long/Short Raid/Challenge mode
    - Single target
    - Multi target
    - Cleave
    - Mass AOE
- Trinket Comparison per spec
- 10/25 raid benchmarks

# Developing

- Install nix
- run `nix develop`

# Running simulations

- `nix run .#<simulation>`

Available simulations from [nix/apps.nix](./nix/apps.nix):

- `singleTargetRaidLong`
- `multiTargetRaidLong`
- `cleaveRaidLong`
- `massMultiTargetRaid`

Example: `nix run .#singleTargetRaidLong`

This will output a json file inside `web/public/data` that is consumed by the
web frontend.
