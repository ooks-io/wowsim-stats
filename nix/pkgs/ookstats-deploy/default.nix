{
  writeShellApplication,
  ookstats,
  awscli2,
  coreutils,
  jq,
  nodejs,
  nodePackages,
  sqlite,
  gum,
}:
writeShellApplication {
  name = "ookstats-deploy";

  runtimeInputs = [
    ookstats
    awscli2
    coreutils
    jq
    nodejs
    nodePackages.vercel
    sqlite
    gum
  ];

  text =
    /*
    bash
    */
    ''

      # configuration

      # how many entries per leaderboard page
      PAGE_SIZE=25

      # how many concurrent connections
      CONCURRENCY=20

      # player search api shard size
      SHARD_SIZE=5000

      # where to output the static API
      API_OUT=web/public

      REGIONS=us,tw,kr,eu

      DB_PATH="db/current.db"
      HASH_PATH="db/current.db.sha256"
      LOCK_PATH="db/db.lock"
      BACKUP_PREFIX="db/backups/"
      LOCAL_DB="local.db"
      LOCK_TIMEOUT=7200  # 2 hours in seconds

      # log helper function
      log() {
        local type="$1"
        local prefix="$2"
        local message="$3"

        if [ "$type" = "verbose" ]; then
          if [ -n "$verbose" ]; then
            gum log --structured --level debug "$message" --prefix "$prefix"
          fi
          return
        fi
        if [ "$type" = "abort" ]; then
          gum log --structured --level error "$message" --prefix "$prefix"
          exit 1
        else
          gum log --structured --level "$type" "$message" --prefix "$prefix"
        fi
      }


      # configuration

      # parse arguments
      DRY_RUN=false
      ENVIRONMENT="production"

      for arg in "$@"; do
        case $arg in
          --dry-run)
            DRY_RUN=true
            shift
            ;;
          --preview)
            ENVIRONMENT="preview"
            shift
            ;;
          --production|--prod)
            ENVIRONMENT="production"
            shift
            ;;
          *)
            echo "Unknown argument: $arg"
            echo "Usage: ookstats-deploy [--dry-run] [--preview|--production]"
            exit 1
            ;;
        esac
      done

      verifyEnv() {
        log info "Environment" "Deploying to: $ENVIRONMENT"

        if [ "$DRY_RUN" = true ]; then
          log info "Dry Run" "DRY RUN MODE - No uploads will be performed"
        fi

        # verify required environment variables
        if [ -z "''${BLIZZARD_API_TOKEN:-}" ]; then
          log abort "Environment" "BLIZZARD_API_TOKEN not set"
        fi

        if [ "$DRY_RUN" = false ]; then
          if [ -z "''${AWS_ENDPOINT_URL:-}" ]; then
            log abort "Environment" "AWS_ENDPOINT_URL not set"
          fi
          if [ -z "''${AWS_BUCKET:-}" ]; then
            log abort "Environment" "AWS_BUCKET not set"
          fi
          if [ -z "''${AWS_ACCESS_KEY_ID:-}" ]; then
            log abort "Environment" "AWS_ACCESS_KEY_ID not set"
          fi
          if [ -z "''${AWS_SECRET_ACCESS_KEY:-}" ]; then
            log abort "Environment" "AWS_SECRET_ACCESS_KEY not set"
          fi
          if [ -z "''${VERCEL_TOKEN:-}" ]; then
            log abort "Environment" "VERCEL_TOKEN not set"
          fi
          if [ -z "''${VERCEL_ORG_ID:-}" ]; then
            log abort "Environment" "VERCEL_ORG_ID not set"
          fi
          if [ -z "''${VERCEL_PROJECT_ID:-}" ]; then
            log abort "Environment" "VERCEL_PROJECT_ID not set"
          fi
        fi
      }

      # s3 helper function
      s3() {
        aws s3 "$@"
      }

      # cleanup function for trap
      cleanup() {
        local exit_code=$?
        if [ "$DRY_RUN" = false ] && [ -f ".lock_acquired" ]; then
          log info "Lock File" "Releasing lock"
          s3 rm "s3://$AWS_BUCKET/$LOCK_PATH" 2>/dev/null || true
          rm -f .lock_acquired
        fi
        exit $exit_code
      }
      trap cleanup EXIT INT TERM

      # === STEP 1: LOCK ACQUISITION ===
      getLock() {
        log info "Lock File" "Acquiring lock..."

        if [ "$DRY_RUN" = false ]; then
          # check if lock exists
          if s3 ls "s3://$AWS_BUCKET/$LOCK_PATH" >/dev/null 2>&1; then
            log info "Lock File" "Lock file exists, checking age..."

            # download and check lock timestamp
            s3 cp "s3://$AWS_BUCKET/$LOCK_PATH" .lock_check || true
            if [ -f .lock_check ]; then
              lock_time=$(cat .lock_check)
              current_time=$(date +%s)
              age=$((current_time - lock_time))

              if [ "$age" -lt "$LOCK_TIMEOUT" ]; then
                log error "Lock File" "Another build is running (lock age: $age seconds)"
                log error "Lock File" "Lock was acquired at: $(date -d @"$lock_time" 2>/dev/null || date -r "$lock_time")"
                rm -f .lock_check
                exit 1
              else
                log warn "Lock File" "Stale lock detected (age: $age seconds), overriding..."
                rm -f .lock_check
              fi
            fi
          fi

          # create lock
          current_timestamp=$(date +%s)
          echo "$current_timestamp" > .lock_temp
          s3 cp .lock_temp "s3://$AWS_BUCKET/$LOCK_PATH"
          rm -f .lock_temp
          touch .lock_acquired
          log info "Lock File" "Lock acquired at: $(date)"
        else
          log info "Dry Run" "Skipping lock acquisition"
        fi
      }

      # === STEP 2: DATABASE DOWNLOAD ===
      downloadDB() {
        log info "Database" "Downloading database..."

        if [ "$DRY_RUN" = false ]; then
          # download database if exists
          if s3 ls "s3://$AWS_BUCKET/$DB_PATH" >/dev/null 2>&1; then
            log info "Database" "Downloading from R2..."
            s3 cp "s3://$AWS_BUCKET/$DB_PATH" "$LOCAL_DB"

            # download and verify hash
            if s3 ls "s3://$AWS_BUCKET/$HASH_PATH" >/dev/null 2>&1; then
              s3 cp "s3://$AWS_BUCKET/$HASH_PATH" "$LOCAL_DB.sha256"

              log info "Database" "Verifying integrity..."
              if ! sha256sum -c "$LOCAL_DB.sha256"; then
                log abort "Database" "Hash mismatch - corrupted download"
              fi
              log info "Database" "Integrity verified"
            else
              log warn "Database" "No hash file found, skipping verification"
            fi

            db_size=$(stat -c%s "$LOCAL_DB" 2>/dev/null || stat -f%z "$LOCAL_DB")
            log info "Database" "Downloaded size: $((db_size / 1024 / 1024)) MB"
          else
            log info "Database" "No existing database found, starting fresh"
            rm -f "$LOCAL_DB"
          fi
        else
          log info "Dry Run" "Using existing local database or starting fresh"
        fi

        # record pre-build database size
        if [ -f "$LOCAL_DB" ]; then
          PRE_BUILD_SIZE=$(stat -c%s "$LOCAL_DB" 2>/dev/null || stat -f%z "$LOCAL_DB")
        else
          PRE_BUILD_SIZE=0
        fi
      }

      # === STEP 3: BACKUP CREATION ===

      backupDB() {
        log info "Backup" "Creating backup..."

        if [ "$DRY_RUN" = false ] && [ -f "$LOCAL_DB" ]; then
          # cleanup old backups FIRST (>24h)
          log info "Backup" "Cleaning up old backups..."
          cutoff_timestamp=$(date -u -d '24 hours ago' +%s 2>/dev/null || date -u -v-24H +%s)

          # check if backup directory exists
          if s3 ls "s3://$AWS_BUCKET/$BACKUP_PREFIX" >/dev/null 2>&1; then
            s3 ls "s3://$AWS_BUCKET/$BACKUP_PREFIX" | awk '{print $4}' | while read -r backup_file; do
              if [ -n "$backup_file" ]; then
                # extract timestamp from filename: 2025-12-04T14:00:00Z.db
                file_timestamp=$(echo "$backup_file" | sed 's/\.db$//' | xargs -I {} date -d {} +%s 2>/dev/null || echo "0")

                if [ "$file_timestamp" -gt 0 ] && [ "$file_timestamp" -lt "$cutoff_timestamp" ]; then
                  log info "Backup" "Deleting old backup: $backup_file"
                  s3 rm "s3://$AWS_BUCKET/$BACKUP_PREFIX$backup_file" || true
                fi
              fi
            done
          else
            log info "Backup" "No existing backups found"
          fi

          # now create new backup
          backup_timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
          backup_path="$BACKUP_PREFIX$backup_timestamp.db"

          log info "Backup" "Uploading to $backup_path"
          s3 cp "$LOCAL_DB" "s3://$AWS_BUCKET/$backup_path"

          log info "Backup" "Backup created successfully"
        else
          log info "Dry Run" "Skipping backup (dry run or no database)"
        fi
      }

      # === STEP 4: BUILD PIPELINE ===

      buildPipeline() {
        log info "Build" "Running ookstats build"

        export OOKSTATS_DB="$LOCAL_DB"

        ookstats build \
          --out "$API_OUT" \
          --regions "$REGIONS" \
          --page-size "$PAGE_SIZE" \
          --shard-size "$SHARD_SIZE" \
          --concurrency "$CONCURRENCY"

        log info "Build" "Build completed successfully"
      }

      # === STEP 5: DATABASE INTEGRITY CHECK ===

      dbIntegrityCheck() {
        log info "Integrity" "Checking database integrity"

        if [ -f "$LOCAL_DB" ]; then
          POST_BUILD_SIZE=$(stat -c%s "$LOCAL_DB" 2>/dev/null || stat -f%z "$LOCAL_DB")

          log info "Integrity" "Pre-build size:  $((PRE_BUILD_SIZE / 1024 / 1024)) MB"
          log info "Integrity" "Post-build size: $((POST_BUILD_SIZE / 1024 / 1024)) MB"

          # check for significant shrinkage
          if [ "$PRE_BUILD_SIZE" -gt 0 ]; then
            size_ratio=$((POST_BUILD_SIZE * 100 / PRE_BUILD_SIZE))
            if [ "$size_ratio" -lt 90 ]; then
              log abort "Integrity" "Database shrunk by >10% - possible data loss (Ratio: $size_ratio%)"
            fi
          fi

          # basic sanity check - count players
          player_count=$(sqlite3 "$LOCAL_DB" "SELECT COUNT(*) FROM players WHERE is_valid = 1" 2>/dev/null || echo "0")
          log info "Integrity" "Valid players in database: $player_count"

          if [ "$player_count" -eq 0 ] && [ "$PRE_BUILD_SIZE" -gt 0 ]; then
            log warn "Integrity" "No valid players found in database"
          fi
        else
          log abort "Integrity" "Database file not found after build"
        fi
      }

      # === STEP 6: HASH & UPLOAD DATABASE ===
      uploadDB() {
        log info "Upload" "Uploading database to R2..."

        if [ "$DRY_RUN" = false ]; then
          log info "Upload" "Calculating database hash..."
          sha256sum "$LOCAL_DB" > "$LOCAL_DB.sha256"

          log info "Upload" "Uploading database..."
          s3 cp "$LOCAL_DB" "s3://$AWS_BUCKET/$DB_PATH"

          log info "Upload" "Uploading hash..."
          s3 cp "$LOCAL_DB.sha256" "s3://$AWS_BUCKET/$HASH_PATH"

          # verify upload
          if ! s3 ls "s3://$AWS_BUCKET/$DB_PATH" >/dev/null 2>&1; then
            log abort "Upload" "Database upload verification failed"
          fi

          log info "Upload" "Database uploaded successfully"
        else
          log info "Dry Run" "Skipping database upload"
          sha256sum "$LOCAL_DB" > "$LOCAL_DB.sha256"
          log info "Dry Run" "Hash calculated (not uploaded):"
          cat "$LOCAL_DB.sha256"
        fi
      }

      # VERCEL DEPLOYMENT
      vercelDeploy() {
        log info "Vercel" "Deploying to Vercel ($ENVIRONMENT)"

        if [ "$DRY_RUN" = false ]; then
          cd web

          log info "Vercel" "Installing dependencies..."
          npm ci

          log info "Vercel" "Pulling configuration..."
          vercel pull --yes --environment="$ENVIRONMENT" --token="$VERCEL_TOKEN"

          if [ "$ENVIRONMENT" = "production" ]; then
            log info "Vercel" "Building for production..."
            vercel build --prod --token="$VERCEL_TOKEN"

            log info "Vercel" "Deploying to production..."
            deployment_url=$(vercel deploy --prebuilt --prod --archive=tgz --token="$VERCEL_TOKEN")
          else
            log info "Vercel" "Building for preview..."
            vercel build --token="$VERCEL_TOKEN"

            log info "Vercel" "Deploying to preview..."
            deployment_url=$(vercel deploy --prebuilt --archive=tgz --token="$VERCEL_TOKEN")
          fi

          log info "Vercel" "Deployment complete: $deployment_url"

          cd ..
        else
          log info "Dry Run" "Skipping Vercel deployment"
          log info "Dry Run" "Would deploy $API_OUT to $ENVIRONMENT"
        fi
      }

      # === STEP 8: LOCK RELEASE ===
      # handled by cleanup trap

      finish() {
        log info "Deploy" "Deployment Complete"
        if [ "$DRY_RUN" = true ]; then
          log info "Dry Run" "This was a dry run - no uploads or deployments were performed"
        fi
      }

      main() {
        verifyEnv
        getLock
        downloadDB
        backupDB
        buildPipeline
        dbIntegrityCheck
        uploadDB
        vercelDeploy
        finish
      }

      main
    '';
}
