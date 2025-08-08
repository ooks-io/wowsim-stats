# reusable bash snippets
{lib, ...}: let
  inherit (lib) replaceStrings concatMapStringsSep;
  # function to generate shell code that finds the repository root
  findRepoRoot =
    /*
    bash
    */
    ''
      # find repository root by looking for flake.nix
      find_repo_root() {
        local current_dir="$PWD"
        while [[ "$current_dir" != "/" ]]; do
          if [[ -f "$current_dir/flake.nix" ]]; then
            echo "$current_dir"
            return 0
          fi
          current_dir="$(dirname "$current_dir")"
        done

        echo "Warning: Could not find repo root (flake.nix), using current directory" >&2
        echo "$PWD"
        return 1
      }

      repo_root=$(find_repo_root)
    '';

  # function to generate shell code that sets up web data directories
  setupWebDataDirs = {
    baseDir ? "web/public/data",
    subdirs ? [],
  }:
  /*
  bash
  */
  ''
    # setup web data directories
    web_data_dir="$repo_root/${baseDir}"
    mkdir -p "$web_data_dir"

    ${concatMapStringsSep "\n" (subdir: ''
        ${replaceStrings ["/"] ["_"] subdir}_dir="$web_data_dir/${subdir}"
        mkdir -p "$${replaceStrings ["/"] ["_"] subdir}_dir"
      '')
      subdirs}
  '';

  # function to generate shell code for common simulation result copying patterns
  copySimulationResults = {
    sourceFile,
    targetDir,
    targetFileName ? null,
  }: let
    finalTargetFileName =
      if targetFileName != null
      then targetFileName
      else "$(basename ${sourceFile})";
  in
    /*
    bash
    */
    ''
      # copy simulation results to target directory
      target_dir="${targetDir}"
      mkdir -p "$target_dir"
      cp "${sourceFile}" "$target_dir/${finalTargetFileName}"
      echo "Copied to: $target_dir/${finalTargetFileName}"
    '';

  # combined function for the most common pattern: find repo root + setup comparison dirs
  setupComparisonDirs = {
    comparisonType, # "race" or "trinkets"
    class,
    spec,
  }:
  /*
  bash
  */
  ''
    ${findRepoRoot}

    # setup comparison directory structure
    comparison_base_dir="$repo_root/web/public/data/comparison/${comparisonType}"
    comparison_dir="$comparison_base_dir/${class}/${spec}"
    mkdir -p "$comparison_dir"
  '';

  # combined function for the most common pattern: find repo root + setup rankings dirs
  setupRankingsDirs =
    /*
    bash
    */
    ''
      ${findRepoRoot}

      # Setup rankings directory structure
      web_data_dir="$repo_root/web/public/data"
      rankings_dir="$web_data_dir/rankings"

      mkdir -p "$web_data_dir"
      mkdir -p "$rankings_dir"
    '';

  # helper to generate DPS ranking display code
  displayDPSRankings = {
    title,
    jsonData,
    sortKey ? ".dps",
    filterCondition ? "select(.dps != null and .dps > 0)",
  }:
  /*
  bash
  */
  ''
    echo ""
    echo "${title}:"
    echo "======================================="
    echo '${jsonData}' | jq -r '
      .results | to_entries[] |
      ${filterCondition} |
      "\(.key): \(${sortKey} | floor) DPS"
    ' | sort -k2 -nr
  '';

  # argument parsing and environment variable handling
  parseArgsAndEnv =
    /*
    bash
    */
    ''
      # Check environment variable first, then parse command line args
      copyToWeb=true
      if [[ -n "''${DONT_COPY:-}" ]]; then
        copyToWeb=false
      fi

      for arg in "$@"; do
        case $arg in
          --dontCopy)
            copyToWeb=false
            shift
            ;;
          *)
            # unknown argument, ignore
            ;;
        esac
      done
    '';

  # conditional file output based on copyToWeb determination
  conditionalOutput = {
    structuredOutput,
    webSetupCode,
    webPath,
    webMessage,
  }:
  /*
  bash
  */
  ''
    if [[ "$copyToWeb" == "true" ]]; then
      ${webSetupCode}
      echo "$finalResult" | jq -c '.' > "${webPath}/${structuredOutput}.json"
      echo "${webMessage}: ${webPath}/${structuredOutput}.json"
    else
      echo "$finalResult" | jq -c '.' > "${structuredOutput}.json"
      echo "Results written to: ${structuredOutput}.json"
    fi
  '';
in {
  inherit displayDPSRankings setupRankingsDirs setupComparisonDirs copySimulationResults setupWebDataDirs findRepoRoot parseArgsAndEnv conditionalOutput;
}
