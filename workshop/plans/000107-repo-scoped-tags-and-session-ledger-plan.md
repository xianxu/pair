# Repo-Scoped Tags and Session Ledger Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Pair tags repo-local work items with a per-tag session ledger, so tags, picker rows, sidecars, and live zellij sessions no longer collide across repos or conflate tag with agent.

**Architecture:** Add a pure scoped identity layer that derives hidden repo scope keys, display repo names, session names, and per-repo data paths from one model (ARCH-DRY, ARCH-PURE). Zellij session names stay human-readable as `pair-<repo>-<tag>` with a numeric suffix only when two distinct repo scopes need the same public name; the hidden scope key is stored in a global session-name index and scoped data directory, never in names/titles/picker labels. Then migrate launcher consumers to use that model before updating nvim/shell/zellij sidecar paths and grandfathering legacy flat data. The issue is not complete until live sessions, history, picker rows, ledger writes, config/agent caches, and existing flat data all derive from or migrate into the scoped model (ARCH-PURPOSE).

**Tech Stack:** Go launcher modules and tests, shell smoke tests, Lua/nvim sidecar consumers, zellij KDL layout, atlas docs.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `RepoScope` | `cmd/internal/launcher/scope.go` | new |
| `ScopedTag` | `cmd/internal/launcher/scope.go` | new |
| `ScopedPaths` | `cmd/internal/launcher/scoped_paths.go` | new |
| `SessionLedger` | `cmd/internal/launcher/ledger.go` | new |
| `SessionNameIndex` | `cmd/internal/launcher/session_index.go` | new |
| `ScopedSessionName` | `cmd/internal/launcher/sessionname.go` | modified |
| `ScopedSnapshot` | `cmd/internal/launcher/session.go` | modified |

- **RepoScope** — hidden repo scope key plus display repo name and absolute repo path.
  - **Relationships:** 1:N with `ScopedTag`; one repo scope owns many local tags.
  - **DRY rationale:** Replaces basename-prefix filtering in `history.go`, bare `pair-<tag>` zellij names in `decision.go`, and flat sidecar paths in `createflow.go` with one source of repo identity.
  - **Future extensions:** Repo default agent (#83) can attach to `RepoScope` without changing tag semantics.

- **ScopedTag** — display tag plus owning `RepoScope`.
  - **Relationships:** N:1 with `RepoScope`; 1:N with `SessionLedger` entries.
  - **DRY rationale:** Keeps "what the user typed" separate from the hidden collision-safe filesystem key.
  - **Future extensions:** Peer-repo touches can record a secondary cwd on ledger entries without moving the tag home.

- **ScopedPaths** — pure path derivation for draft/log/queue/config/agent/pane/scrollback/changelog/adapt/outer-tty sidecars under the repo scope dir.
  - **Relationships:** 1:1 with `ScopedTag` for tag-level paths; 1:1 with `(ScopedTag, agent)` for agent-level paths.
  - **DRY rationale:** Removes scattered string formatting such as `draft-"+tag+".md`, `config-"+tag+"-"+agent+".json`, and zellij layout fallback expressions.
  - **Future extensions:** Can grow a compatibility reader list while keeping writers scoped-only.

- **SessionLedger** — append-only JSONL ledger for a tag's sessions.
  - **Relationships:** 1:N with `ScopedTag`; each entry records one agent session launch or discovery.
  - **DRY rationale:** Becomes source of truth for last agent/params/session, while `agent-<tag>` and `config-<tag>-<agent>.json` remain derived compatibility caches.
  - **Future extensions:** Retention policy can compact old rows while preserving last-per-agent entries.

- **SessionNameIndex** — global append-only/public-name registry mapping `session_name -> {scope_key, repo_root, repo_name, tag, assigned_at, last_seen}`.
  - **Relationships:** N:1 with `RepoScope`; one scope can own many active or remembered public names.
  - **DRY rationale:** Solves same-name repo collisions without putting the hidden scope key in zellij's user-visible session namespace.
  - **Future extensions:** Can become the source for `pair list --all-repos` and stale-name repair.

- **ScopedSessionName** — zellij-safe, globally unique public session name. Format is `pair-<repo-component>-<tag-component>` for the first scope and `pair-<repo-component>-<tag-component>-N` for later same-public-name collisions, where `N` is the lowest stable positive integer assigned by `SessionNameIndex`. If zellij rejects that readable base for #54 length, the builder deterministically shortens the repo/tag components, never replacing them with a hash.
  - **Relationships:** 1:1 with a live `ScopedTag` attempt; the index maps the public name back to a scope and tag.
  - **DRY rationale:** Replaces direct `sessionName(tag)` calls and keeps #54 length probing centralized while honoring the no-hash-in-UI constraint.
  - **Future extensions:** If zellij exposes structured metadata later, the numeric suffix can remain only as compatibility, with metadata carrying the stable key.

- **ScopedSnapshot** — launch decision snapshot filtered to the current repo scope, with live rows annotated by tag, session name, and last agent.
  - **Relationships:** 1:1 with each startup decision; contains current-scope live sessions and history rows.
  - **DRY rationale:** Prevents each consumer from rediscovering repo filtering independently.
  - **Future extensions:** Supports a future "all repos" list by adding an explicit broader query instead of weakening default scope.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ScopeResolver` | `cmd/internal/launcher/osruntime.go` | new | cwd/git filesystem |
| `SessionNameStore` | `cmd/internal/launcher/osruntime.go` | new | global session-name index JSONL |
| `ScopedHistorySource` | `cmd/internal/launcher/history.go` | modified | scoped data dir + legacy flat data dir |
| `LedgerStore` | `cmd/internal/launcher/ledger.go` | new | JSONL files + atomic writes |
| `ScopedOSRuntime` | `cmd/internal/launcher/osruntime.go` | modified | zellij, fzf, filesystem, process env |
| `ScopedPaneEnv` | `zellij/layouts/main.kdl` and `nvim/init.lua` | modified | inherited `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_SCOPE_DIR` |
| `LegacyGrandfathering` | `cmd/internal/launcher/migrate.go` | new | existing flat sidecars and live unscoped sessions |

- **ScopeResolver** — resolves repo root, repo display name, hidden key, and scope directory once per launch.
  - **Injected into:** `RunLaunch` through `Env`/runtime setup; pure derivation is unit-tested with fake cwd/root values.
  - **Future extensions:** Alternate root detection for non-git directories.

- **SessionNameStore** — reads/appends the global public-name index under the global Pair data dir.
  - **Injected into:** session-name assignment, live-session filtering, picker rows, list, restart, and legacy attach.
  - **Future extensions:** Can add pruning of stale names after zellij sessions and scoped ledgers disappear.

- **ScopedHistorySource** — lists historical tags from the current scope dir, with compatibility reads from flat data when no scoped copy exists.
  - **Injected into:** `OSRuntime.ScanHistory`.
  - **Future extensions:** A migration command can reuse the compatibility enumeration.

- **LedgerStore** — appends and reads ledger entries, then emits derived caches.
  - **Injected into:** create flow, session watcher integration, `InferAgent`, config picker, and bare `pair` continuation selection.
  - **Future extensions:** Retention policy and repair command.

- **ScopedOSRuntime** — owns concrete zellij and filesystem effects after pure identity is resolved.
  - **Injected into:** `RunLaunch`, list, rename, restart, cleanup.
  - **Future extensions:** `pair list --all-repos`.

- **ScopedPaneEnv** — makes panes consume the scoped data dir rather than reconstructing flat `~/.local/share/pair` paths.
  - **Injected into:** zellij environment only; nvim reads the same env.
  - **Future extensions:** Additional pane helpers should consume `PAIR_DATA_DIR`/`PAIR_SCOPE_DIR`, not derive paths.

- **LegacyGrandfathering** — moves or aliases old flat sidecars into the current repo scope without deleting source data until successful.
  - **Injected into:** first scoped launch, history scan, and attach/resume of live old sessions.
  - **Future extensions:** Manual `pair migrate` diagnostic.

## Chunk 1: Scoped Identity and Path Model

### Task 1: Add pure repo scope derivation

**Files:**
- Create: `cmd/internal/launcher/scope.go`
- Test: `cmd/internal/launcher/scope_test.go`
- Modify: `cmd/internal/launcher/tag.go`

- [x] **Step 1: Write failing tests for scope derivation**

Cover:
- same basename at different absolute paths gets different hidden keys.
- display repo name is the basename, normalized only where filenames/session names need it.
- hidden key never appears in `DisplayRepo`, `DisplayTag`, or picker text helpers.
- path hashing uses the cleaned absolute repo path, not cwd basename.

Run: `go test ./cmd/internal/launcher -run 'TestRepoScope|TestDefaultTag'`
Expected: FAIL because `RepoScope` does not exist.

- [x] **Step 2: Implement minimal pure model**

Add `RepoScope{Root, DisplayName, Key}` and helpers:
- `ResolveRepoScope(root string) (RepoScope, error)` for pure normalized input.
- `NormalizeDisplayComponent` only for filesystem/session-name unsafe contexts.
- Keep `DefaultTag(cwd)` behavior until create flow is migrated; tests should document the transition.

- [x] **Step 3: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestRepoScope|TestDefaultTag'`
Expected: PASS.

### Task 2: Centralize scoped sidecar paths

**Files:**
- Create: `cmd/internal/launcher/scoped_paths.go`
- Test: `cmd/internal/launcher/scoped_paths_test.go`
- Modify: `cmd/internal/launcher/config.go`

- [x] **Step 1: Write failing path tests**

Cover every existing sidecar family found in the shadow sweep:
- `draft`, `log`, `queue`
- `config`, `agent`, `agent-pid`
- `pane`, `scrollback` raw/ansi/viewport/events
- `adapt`, `outer-tty`, `agent-output`, `agent-picks`
- `changelog`, `nvim-pid`

Run: `go test ./cmd/internal/launcher -run 'TestScopedPaths|TestCanonicalConfigPath'`
Expected: FAIL for missing `ScopedPaths`.

- [x] **Step 2: Implement `ScopedPaths` and bridge config helpers**

Add a constructor taking `(globalDataDir, RepoScope, tag)` and returning `ScopeDir = <globalDataDir>/repos/<scope-key>` or equivalent hidden subdir. Keep `CanonicalConfigPath` as a wrapper over the new path helper where callers have only `(dataDir, tag, agent)` until all call sites are migrated.

- [x] **Step 3: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestScopedPaths|TestCanonicalConfigPath'`
Expected: PASS.

### Task 3: Centralize scoped session names

**Files:**
- Modify: `cmd/internal/launcher/decision.go`
- Create or modify: `cmd/internal/launcher/sessionname.go`
- Create: `cmd/internal/launcher/session_index.go`
- Test: `cmd/internal/launcher/decision_test.go`
- Test: `cmd/internal/launcher/session_index_test.go`

- [x] **Step 1: Write failing tests for scoped session naming**

Cover:
- same repo/tag keeps readable `pair-<repo>-<tag>` when unclaimed.
- two same-name repos with different scope keys get stable `pair-<repo>-<tag>` and `pair-<repo>-<tag>-2` assignments.
- the same scope/tag reuses its prior public session name when zellij still has it or the index marks it current.
- over-long names are shortened by trimming the longer component first, preserving at least 4 normalized characters of repo and tag plus the numeric suffix; if no candidate passes `ProbeSessionName`, launch fails with "repo/tag too long for zellij socket path; choose a shorter tag" before writing sidecars.
- hidden hash/key is not present in picker labels or pane titles.
- hidden hash/key is present only in the index record and scoped data-dir path.
- generated names still pass through `ProbeSessionName`.

Run: `go test ./cmd/internal/launcher -run 'TestScopedSessionName|TestSessionNameIndex|TestDecideLaunch'`
Expected: FAIL for missing scoped session-name API.

- [x] **Step 2: Implement public-name assignment**

Implement pure helpers:
- `PublicSessionBase(scope RepoScope, tag string) string` -> `pair-<repo-component>-<tag-component>`.
- `BuildSessionNameCandidates(scope RepoScope, tag string, suffix int) []string` -> full readable candidate first, then deterministic shorter candidates. Shortening rule: reserve `len("pair--") + len(optional "-N")`, repeatedly remove one rune from the longer of repo component and tag component until both are at the 4-rune floor, and never emit a candidate containing the scope key/hash.
- `AssignSessionName(index SessionNameIndex, live []Session, scope RepoScope, tag string) (name string, updated SessionNameIndex)`.
- Assignment rule: reuse the existing index binding for the same `(scope_key, tag)` when possible; otherwise choose the lowest `-N` public suffix whose live/index owner is absent or the same scope/tag. For each suffix, probe candidates from `BuildSessionNameCandidates` in order and reserve the first accepted name. Never embed the scope key in the returned name. If every candidate for suffixes 1..100 is rejected or occupied, return a typed `SessionNameExhausted` error and abort before creating sidecars.

- [x] **Step 3: Modify decisions to carry session name separately from tag**

Keep `LaunchDecision.Tag` as the repo-local tag. Make every decision use the assigned public session name instead of `sessionName(tag)`. Do not infer repo scope inside pure decisions; pass a pure naming context and session-name index snapshot.

- [x] **Step 4: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestScopedSessionName|TestSessionNameIndex|TestDecideLaunch'`
Expected: PASS.

## Chunk 2: Ledger and Launcher Source of Truth

### Task 4: Add session ledger pure parsing and retention

**Files:**
- Create: `cmd/internal/launcher/ledger.go`
- Test: `cmd/internal/launcher/ledger_test.go`

- [x] **Step 1: Write failing ledger tests**

Cover:
- append/read JSONL entries with `agent`, `args`, `session_id`, `started`, `last_active`, `repo_root`, `repo_name`.
- latest entry wins for bare `pair` continuation.
- latest per agent is retained when compacting.
- malformed rows are skipped without losing valid rows.

Run: `go test ./cmd/internal/launcher -run TestSessionLedger`
Expected: FAIL for missing ledger implementation.

- [x] **Step 2: Implement pure ledger types and selectors**

Add pure helpers:
- `ParseLedger(raw string) []LedgerEntry`
- `BuildLedgerLine(entry LedgerEntry) (string, error)`
- `LatestLedgerEntry(entries []LedgerEntry) (LedgerEntry, bool)`
- `LatestLedgerEntryForAgent(entries []LedgerEntry, agent string) (LedgerEntry, bool)`
- `CompactLedger(entries []LedgerEntry, keepRecent int) []LedgerEntry`

- [x] **Step 3: Verify green**

Run: `go test ./cmd/internal/launcher -run TestSessionLedger`
Expected: PASS.

### Task 5: Make ledger drive agent/config inference

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Test: `cmd/internal/launcher/createflow_test.go`
- Test: `cmd/internal/launcher/osruntime_test.go`

- [x] **Step 1: Write failing tests for ledger-backed inference**

Cover:
- `InferAgent(tag)` prefers ledger latest entry, falls back to `agent-<tag>`, then config filename.
- `runCreate` appends a ledger row and still writes derived `agent-<tag>` and config cache.
- bare `pair` with scoped history continues latest current-repo tag and inherits the ledger agent/params.

Run: `go test ./cmd/internal/launcher -run 'Test.*Ledger|TestRunLaunch.*Latest'`
Expected: FAIL because inference ignores ledger.

- [x] **Step 2: Add runtime store methods**

Extend the runtime seam narrowly, for example:
- `ReadLedger(tag string) ([]LedgerEntry, error)`
- `AppendLedger(tag string, entry LedgerEntry) error`
- `DerivedConfigPath(tag, agent string) string` if needed to avoid raw string formatting.

Update fake runtime and `OSRuntime`.

- [x] **Step 3: Wire create flow**

When launching:
- resolve scoped paths before creating draft/config.
- append a ledger row at launch with known args and session id if known.
- when async session watcher later captures the id, update by appending a newer row rather than mutating old rows.
- continue writing compatibility caches from the ledger/source launch data.

- [x] **Step 4: Verify green**

Run: `go test ./cmd/internal/launcher -run 'Test.*Ledger|TestRunLaunch.*Latest|TestRunLaunchPickInferredAgentMustNotInheritCliArgs'`
Expected: PASS.

## Chunk 3: Repo-Scoped Picker, History, List, Rename, Cleanup

### Task 6: Filter live and historical rows to current repo

**Files:**
- Modify: `cmd/internal/launcher/session.go`
- Modify: `cmd/internal/launcher/history.go`
- Modify: `cmd/internal/launcher/zellijparse.go`
- Modify: `cmd/internal/launcher/pick.go`
- Test: `cmd/internal/launcher/history_test.go`
- Test: `cmd/internal/launcher/pick_test.go`
- Test: `cmd/internal/launcher/zellijparse_test.go`

- [ ] **Step 1: Write failing repo-filter tests**

Cover:
- history scan lists only current scope dir and no longer uses basename prefix.
- detached sessions from another repo do not trigger the picker.
- picker rows annotate agent, repo display name, and tag.
- no hidden scope key appears in picker plain or colored rows.
- live session mapping reads `SessionNameIndex` first and scoped pane metadata second; unindexed sessions are treated as legacy candidates, never as current-scope proof by name alone.

Run: `go test ./cmd/internal/launcher -run 'TestHistory|TestBuildPickRows|TestPairSessionNames'`
Expected: FAIL with current flat filtering.

- [ ] **Step 2: Implement scoped snapshots**

Build a snapshot with this exact mapping order:
1. If `SessionNameIndex` binds the zellij session name to the current `scope_key`, include it as current-scope live.
2. If no index entry exists, read scoped pane metadata written at launch (`scope_key`, `repo_root`, `tag`, `agent`, `session_name`) and include only when `scope_key` or cleaned `repo_root` matches current scope.
3. If neither exists, classify as `LegacyUnscoped`; do not include it in the normal current-scope picker. Task 9 defines the explicit recovery path.

- [ ] **Step 3: Update picker rows**

Use labels such as `pair <repo>/<tag>  <agent>  (...)` while returning plain text keys that map to `ScopedTag`. Keep fzf ANSI handling unchanged.

- [ ] **Step 4: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestHistory|TestBuildPickRows|TestPairSessionNames'`
Expected: PASS.

### Task 7: Update list, rename, restart, and cleanup for scoped paths

**Files:**
- Modify: `cmd/internal/launcher/list.go`
- Modify: `cmd/internal/launcher/rename.go`
- Modify: `cmd/internal/launcher/restart.go`
- Modify: `cmd/internal/launcher/lifecycle.go`
- Test: `cmd/internal/launcher/rename_test.go`
- Test: `cmd/internal/launcher/restart_test.go`
- Test: `cmd/internal/launcher/lifecycle_test.go`

- [ ] **Step 1: Write failing behavior tests**

Cover:
- `pair list` shows current repo rows by default and keeps agent/status columns.
- rename moves scoped sidecars atomically and rolls back within one scope dir.
- restart markers carry scoped session names but repo-local tags.
- cleanup removes scoped tag artifacts without touching another repo's same tag.

Run: `go test ./cmd/internal/launcher -run 'TestRunList|TestRename|TestRestart|TestCleanup|TestPark'`
Expected: FAIL where flat paths are still assumed.

- [ ] **Step 2: Migrate each consumer to `ScopedPaths`**

Replace hand-built paths in these files with `ScopedPaths`. Keep the runtime interfaces small; avoid passing a large path bag into helpers that only need one path family.

- [ ] **Step 3: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestRunList|TestRename|TestRestart|TestCleanup|TestPark'`
Expected: PASS.

## Chunk 4: Pane Consumers and Legacy Grandfathering

### Task 8: Update zellij and nvim consumers to scoped data

**Files:**
- Modify: `zellij/layouts/main.kdl`
- Modify: `nvim/init.lua`
- Modify as needed: `bin/pair-notify`, `bin/lib/adapt-log.sh`, `bin/pair-changelog-open`, `bin/pair-scrollback-open`
- Test: existing shell/Lua tests under `tests/`, plus targeted new tests if a helper path is not covered.

- [ ] **Step 1: Write failing smoke tests for scoped `PAIR_DATA_DIR`**

Add tests proving:
- zellij layout uses inherited `PAIR_DATA_DIR` instead of recomputing global flat data dir.
- nvim draft/log/queue and changelog/session-key helpers work when `PAIR_DATA_DIR` points at a scoped subdir.
- shell helpers append/read under the scoped dir.

Run the smallest applicable tests first, for example:
- `bash tests/changelog-session-key-test.sh`
- `bash tests/queue-send-test.sh`
- `bash tests/scrollback-open-test.sh`
- `bash tests/changelog-open-test.sh`

Expected: FAIL for any helper still deriving the old global path.

- [ ] **Step 2: Update consumers to treat `PAIR_DATA_DIR` as authoritative**

Do not recompute `${XDG_DATA_HOME:-$HOME/.local/share}/pair` in panes or helpers when `PAIR_DATA_DIR` is already exported. If a helper needs the global root, pass a separate env such as `PAIR_GLOBAL_DATA_DIR`.

- [ ] **Step 3: Verify green**

Run the same shell/Lua tests.
Expected: PASS.

### Task 9: Grandfather flat sidecars and live old sessions

**Files:**
- Create: `cmd/internal/launcher/migrate.go`
- Test: `cmd/internal/launcher/migrate_test.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/history.go`
- Modify: `cmd/internal/launcher/osruntime.go`

- [ ] **Step 1: Write failing grandfathering tests**

Cover:
- flat `draft-work.md`, `log-work.md`, `config-work-claude.json`, `queue-work/`, and scrollback files are copied or moved into the current scope on first use without deleting data before successful write.
- live old `pair-work` in the same repo can be attached/resumed.
- flat artifacts from another repo with same basename are not silently claimed unless pane metadata or transcript cwd evidence proves they belong to current repo.
- ambiguous flat artifacts with no cwd evidence are listed as `legacy unscoped` recovery rows and are copied into the current scope only after the user explicitly selects that row.

Run: `go test ./cmd/internal/launcher -run 'TestGrandfather|TestMigrate'`
Expected: FAIL for missing migration.

- [ ] **Step 2: Implement conservative migration**

Implement these ownership rules:
1. **Proven current repo:** pane metadata `cwd`/`cwd_display`, scoped pane metadata, or transcript path proves the repo root. Copy into the current scope automatically on first current-scope launch/resume.
2. **Proven other repo:** leave untouched and exclude from current-scope history/picker.
3. **Ambiguous legacy:** leave flat files untouched; show a separate picker row labeled `legacy unscoped <tag> (manual import)` only when `tag == DefaultTag(currentRepoRoot)` or `strings.HasPrefix(tag, DefaultTag(currentRepoRoot)+"-")`. Selecting it copies into the current scope and writes a ledger row with `legacy_import: true`.

Prefer copy-then-atomic-rename patterns for files Pair can race on. Record a migration marker in the scope dir after success. Never delete a flat source unless the operation is explicitly a move for a lifecycle cleanup and the scoped copy exists.

- [ ] **Step 3: Verify green**

Run: `go test ./cmd/internal/launcher -run 'TestGrandfather|TestMigrate|TestHistory'`
Expected: PASS.

## Chunk 5: Acceptance, Docs, and Gate Prep

### Task 10: End-to-end regression tests

**Files:**
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `tests/pair-rename.sh` if scoped shell coverage is needed
- Modify: `Makefile` only if a new test script is added

- [ ] **Step 1: Add acceptance-level tests**

Cover Done-when directly:
- two repos each have `work` with independent sidecars and live session names.
- picker current-repo filtering and agent annotation.
- explicit `pair codex -- <codex args>` never launches another agent.
- bare `pair` continues latest current-repo ledger entry with agent + params.
- legacy flat data remains recoverable.

Run: `go test ./cmd/internal/launcher`
Expected: FAIL until previous tasks are fully wired.

- [ ] **Step 2: Verify full local suite**

Run:
- `go test ./...`
- `make test`

Expected: PASS.

### Task 11: Update atlas and issue artifacts

**Files:**
- Modify: `atlas/index.md`
- Modify or create: an atlas page describing Pair identity/session storage, likely `atlas/session-identity.md`
- Modify: `workshop/issues/000107-repo-scoped-tags-and-session-ledger.md`

- [ ] **Step 1: Update docs**

Document:
- repo scope vs display tag vs agent vs session id.
- scoped data-dir layout.
- ledger source-of-truth and derived caches.
- compatibility/grandfathering behavior.

- [ ] **Step 2: Check plan boxes and log verification**

Update #107 `## Plan` checkboxes as tasks complete. Add `## Log` notes with commands and any ARCH-* decisions.

- [ ] **Step 3: Run final verification for close**

Run:
- `git diff --check`
- `go test ./...`
- `make test`
- any live/manual smoke required by the final diff, especially `zellij setup --check --config-dir zellij` if `main.kdl` or config changes.

Expected: PASS. Use this evidence in `sdlc close --issue 107 --verified '<evidence>'`.
