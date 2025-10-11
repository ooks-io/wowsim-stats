# Post–Realm-Merge Plan

This plan outlines the changes needed to robustly handle Blizzard realm consolidations/renames while preserving player identity and best runs in our rebuild-from-scratch pipeline.

## Goals
- Preserve best runs for a character across realm/name changes (identity = Blizzard character ID).
- Resolve a player’s “current” realm/name at build time, without relying on prior builds.
- Avoid 404s after realm renames (e.g., `arugal` → `arugal-au`).
- Keep dungeon leaderboards historically accurate, and make player pages/leaderboards stable and linkable.

## Backend (Data + Fetch)
- Realm source of truth
  - Fetch realm/connected-realm lists per region (us/eu/kr/tw) at build start and build a slug→(id,name,region) map.
  - Derive a rename/alias map (e.g., `arugal` → `arugal-au`) from differences vs. prior constants or by connected-realm membership.
  - Replace/augment hardcoded lists with fetched data in the build path (keep constants as fallback only).

- Player identity resolution (canonical identity per build)
  - While ingesting runs, collect per `player_id` all observed identities: (region, realm_slug, name, last_seen_ts).
  - Select canonical identity:
    - Primary: latest profile identity (if profile API succeeded).
    - Fallback: most recent identity from runs (max `completed_timestamp`).
  - Keep a small ordered candidate list for profile-fallback attempts.

- Profile fetch fallback logic
  - Before calling the Profile API, normalize realm slug via the rename map (e.g., `remulos` → `remulos-au`).
  - Fallback order for a single player (stop on first success):
    1) Canonical identity from `players` (name + realm_slug), after rename-normalization.
    2) Connected-realm sweep: try the same name against all slugs in the same `connected_realm_id` (e.g., Pagle group, Mirage Raceway group).
    3) Last-run realm heuristic: use the realm_slug of the player’s most recent run and retry the profile.
    4) Optional: try other identities observed in this build (distinct (realm_slug,name) pairs ordered by most recent run timestamp).
  - Persist the resolved identity for this build on first success (update `players.name` and `players.realm_id`).

- Realm lookup robustness
  - Use (region, slug) uniqueness for `realms` and lookups (done).
  - For unknown slugs during run ingestion, insert a scoped placeholder row to prevent batch aborts (done).

- DB helpers to implement
  - `GetPlayerCurrentIdentity(playerID int) (region string, realmSlug string, name string, err error)`
    - Join `players` → `realms` by `players.realm_id`; return `r.region, r.slug, p.name`.
  - `GetConnectedRealmSlugs(region, realmSlug string) ([]string, error)`
    - Look up this realm’s `connected_realm_id` and return all `slug` where `region = ? AND connected_realm_id = ?`.
  - `GetLastRunRealmForPlayer(playerID int) (region string, realmSlug string, ts int64, err error)`
    - `SELECT rr.region, rr.slug, cr.completed_timestamp FROM run_members rm JOIN challenge_runs cr ON cr.id = rm.run_id JOIN realms rr ON rr.id = cr.realm_id WHERE rm.player_id = ? ORDER BY cr.completed_timestamp DESC LIMIT 1`.
  - Tiny helper: `NormalizeRealmSlug(region, slug) string` (apply static rename map, e.g., US OCE → `-au`).

- Ranking semantics (decide and implement)
  - Dungeon leaderboards: keep using the run’s realm (historical accuracy).
  - Player leaderboards (realm scope):
    - Option A: current realm (canonical identity).
    - Option B: realm of best run per dungeon (where PRs were set).
  - We can compute both if desired; choose which to display by default.

## Generator (Static API)
- Canonical identity usage
  - Emit player pages/leaderboards using the canonical identity chosen above.
  - Add a permanent ID-based route: `/player/id/{player_id}.json` (and an Astro route if desired) for stable linking.

- Redirect stubs (no server required)
  - For any non-canonical identities observed this build, write a small JSON `{ "redirect_to": "/player/{region}/{realm}/{name}" }` at the old path.
  - Frontend will follow `redirect_to` on load; preserves old links/search index.

- Class/spec resiliency
  - When Profile API fails, derive `class_name`/`active_spec_name` from `main_spec_id` in generated JSON (implemented).

## Frontend
- Player profile page
  - Follow `redirect_to` if present in the loaded JSON and navigate to the canonical path.
  - If profile API data is missing, render a “last seen” badge based on latest run timestamp and identity.
  - Keep class/spec coloring via `main_spec_id` fallback (implemented).

- Player leaderboards
  - Ensure link color derived from `class_name` or fallback via `main_spec_id` (implemented).

- Realm selectors/lists
  - Ensure US OCE realms use `-au` slugs in UI (updated) and allow dynamic population from the fetched realm map in the future.

## Ops/Build
- Environment & rate limits
  - Confirm `BLIZZARD_API_TOKEN` availability; document recommended concurrency/timeouts.
  - Cap profile-fallback attempts per player to avoid excessive retries.

- Diagnostics
  - Log a summary of renamed slugs detected each build.
  - Emit a lightweight report of players whose profile was unresolved (for monitoring).

## Validation
- Test cases
  - Player with runs only on old slug (e.g., `arugal`) and profile disabled → uses latest run identity and renders with spec-derived class color.
  - Player with 404 then fallback to `-au` slug → profile resolves, identity updates.
  - Players who moved realms and never ran again → keep best runs via `player_id`; page exists under last-seen identity and ID route; redirect stubs from any alternate identities in this build.

## Incremental Rollout
1) Keep identity by `player_id` (already in place) and (region,slug) realm uniqueness (done).
2) Normalize OCE slugs and TW support (added) and fetch live realm lists at build start.
3) Implement canonical identity selection and profile-fallback logic.
4) Add redirect stubs + frontend redirect handling.
5) Decide realm ranking semantics and adjust generator SQL accordingly.
6) Optional: Add ID-based route and “last seen” UX.

## Risks & Limitations
- Classic APIs do not support character search; if a player changes both realm and name and never appears in new runs, we cannot discover the new identity. We will:
  - Preserve their best runs under `player_id`.
  - Serve the page under last-seen identity (and ID route).
  - Use redirects for identities observed in the same build.
