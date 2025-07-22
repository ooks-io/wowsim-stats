<h1> <a href="https://wowsimstats.com">wowsimstats</a></h1>

Powered by [wowsims](https://github.com/wowsims/mop) and
[nix](https://nixos.org/guides/how-nix-works/).

## The how and why

The wowsimcli tool consumes simulation inputs as RaidSimRequest in the protojson
format. These input files can be exported from the
[WowSims Web UI](https://wowsims.com). The core of this project is a workflow
that automates simulation at scale. We use nix to programmatically compose the
required protojson input files. A script then orchestrates the entire process:

1. **Generate Inputs**: Based on the simulation you choose (eg
   dps-p1-raid-single-long), nix generates the corresponding protojson input
   files for every specialisation.
2. **Execute Simulations**: The script invokes wowsimcli for each generated
   file, running the simulation and producing a raw JSON output.
3. **Aggregate Data**: The script collects the output from all the individual
   runs and aggregates the data into a single, clean JSON file.

This approach allows us to easily generate thousands of simulation combinations
while also leveraging the powerful caching and parallelization features of the
nix build system. If a simulation's input hasn't changed, nix can use a cached
result instead of re-running it, saving significant time.

Right now the project is still in a _proof of concept_ stage, and requires more
work.

## Developing

- Install [nix (the package manager)](https://nixos.org/download/)
- run `nix develop`

This will enter a nix development shell with all the required dependencies in
your PATH.

- `wowsimcli`: The command-line tool used to run the simulations.

- `nodejs`: Web frontend.

## Running and updating simulations

To run simulations run:

- `nix run .#<simulation>`

Simulations are organized into the following format:

Race comparison benchmarks:
`race-<class>-<spec>-<phase>-<encounterType>-<targets>-<duration>`

DPS rankings: `dps-<phase>-<encounterType>-<targets>-<duration>`

`<class>-<spec>`:

- All DPS class/spec options are currently implemented, except for feral
- example: `druid-balance`

`<phase>`:

- Only `p1` is available currently, `preRaid` coming soon.
- example `p1`

`<encounterType>`:

- Only `raid` is available currently (dungeon and specific boss simulations are
  planned for future updates).
- example `raid`

`targets`:

- `single` 1 Target
- `cleave` 2 Target
- `three` 3 target
- `ten` 10 target
- example `single`

`duration`:

- `long`: 300 seconds, with a 60s variance (`300s ±60s`)
- `short`: 120 seconds, with a 30s variance (`120s ±30s`)
- `burst`: 30 seconds, with a 10s variance (`30s ±10s`)

**Example simulations**

`nix run .#dps-p1-raid-single-long` This will run all dps specs in P1 bis gear,
against 1 default raid boss for 300s 60s +/-, for 10,000 iterations each.

`nix run .#race-paladin-retribution-p1-raid-cleave-burst` This will run
simulations for all Retribution's playable races, against 2 default raid boss
targets, for 30s 10s +/-, for 10,000 iterations each.

## Simulation Output

Running a simulation script generates a JSON file with the aggregated data in
your present working directory. These files are also copied to web/public/data/
to be used by the web front-end. The ability to specify an output directory is
planned for a future update.

To refresh all data used by the website, run the following command:

`nix run .#allSimulations`

This command runs every simulation and updates the corresponding data files in
web/public/data/.
