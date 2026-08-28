# Deterministic Agent Session-Tree Inventory Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give Pair one deterministic inventory of every supported agent's native root/subagent forest, and establish a Pair tag's root only after Pair observes one completed native user-to-agent round.

**Architecture:** A new `sessioninventory` package separates pure native facts, forest assembly, round matching, binding state, and stable rendering from one injected runtime boundary. Agent scanners emit facts rather than making selection decisions. The watcher records a launch baseline, monitors exact Pair-log/native rounds, and persists a root only when one candidate uniquely completes the round; validated native parent edges then propagate that established binding to descendants without becoming binding evidence. Every existing native-session consumer migrates to this package, and a shadow-sweep test prevents independent glob/newest/`lsof` logic from returning (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`).

**Tech Stack:** Go standard library, existing Pair launcher/sessionwatch/transcript packages, platform `sqlite3` behind an injected seam for Agy facts, shell integration tests, Neovim Lua tests, sanitized native fixtures, and an opt-in no-LLM live conformance probe.

**Authoritative spec:** `workshop/issues/000155-agent-session-tree-inventory.md`, especially “Round-gated establishment contract.” Earlier send-journal and minted-incarnation passages are explicitly superseded and must not be implemented.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status | Introduced |
|------|----------|--------|------------|
| `NativeRecordFact` | `cmd/internal/sessioninventory/scan.go` | new | M1 |
| `SessionNode` / `SessionForest` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `ParentEdge` / `EdgeProvenance` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `Diagnostic` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `Inventory` | `cmd/internal/sessioninventory/inventory.go` | new | M2 |
| `NativeEvent` | `cmd/internal/sessioninventory/event.go` | new | M2 |
| `TokenUsage` | `cmd/internal/sessioninventory/usage.go` | new | M1 |
| `RoundObservation` | `cmd/internal/sessioninventory/round.go` | new | M2 |
| `Binding` / `Candidate` / `Ambiguity` / `Evidence` | `cmd/internal/sessioninventory/binding.go` | new | M2 |
| `PairLedgerFact` / `PairLogFact` | `cmd/internal/sessioninventory/pairfacts.go` | new | M2 |
| `SessionActivity` | `cmd/internal/sessioninventory/activity.go` | new | final |

**`NativeRecordFact`** — bounded, scanner-authorized facts extracted from one
native artifact without selection or correlation.
- **Relationships:** Facts assemble into nodes, edges, timestamps, events, and diagnostics.
- **DRY rationale:** Agent-specific format knowledge stays in scanners while forest assembly remains agent-neutral.
- **Future extensions:** A new native schema version adds a fact adapter and fixtures.

**`NativeEvent`** — normalized user, assistant, tool-call, tool-result, and terminal-error events emitted by all four native scanners.
- **Relationships:** N:1 with a `SessionNode`; ordered events form candidate rounds.
- **DRY rationale:** Slugging, watcher correlation, offline recovery, and conformance must not maintain separate transcript parsers.
- **Future extensions:** Additional event kinds can be added at the scanner boundary without changing forest identity.

**`TokenUsage`** — last accepted root-context token occupancy emitted as a
scanner fact, excluding sidechains, synthetic records, null usage, and agents
without a supported source.
- **Relationships:** Belongs to a scanner-authorized root and feeds context/title projections.
- **DRY rationale:** Removes Claude/Codex JSONL parsing from `ctxmeter`.
- **Future extensions:** New agent usage records widen only versioned scanner fixtures.

**`SessionNode` / `SessionForest`** — deterministic native root and descendant topology, including stable IDs, nullable parent, artifacts, timestamps with source, and edge provenance.
- **Relationships:** One forest root has zero or more descendants; an edge is accepted only after agent-specific validation.
- **DRY rationale:** All consumers need the same full topology and must not independently select first/newest roots.
- **Future extensions:** Resume capabilities can be projected from nodes without changing discovery.

**`ParentEdge` / `EdgeProvenance`** — validated native child-to-parent relation
with the scanner facts that authorized it.
- **Relationships:** Builds forest topology in M1; in M2 it may propagate an already-established root binding to descendants while preserving both binding and edge provenance.
- **DRY rationale:** Parent validation is represented once and cannot accidentally enter the correlation evidence lattice.
- **Future extensions:** Additional parent schemas remain fail-closed until a fixture authorizes them.

**`Diagnostic`** — coded fail-safe explanation attached to partial scanner or
correlation results.
- **Relationships:** M1 forests and M2 inventories retain diagnostics rather than dropping malformed facts.
- **DRY rationale:** Scanners, CLI, conformance, and consumers share one registry.
- **Future extensions:** New codes require explicit severity and ordering.

**`Inventory`** — complete, stably ordered M2 snapshot of forests, bindings,
ambiguities, and diagnostics.
- **Relationships:** Wraps M1 forest results after M2 correlation types exist.
- **DRY rationale:** Human output, JSON, conformance, and consumers share one result contract.
- **Future extensions:** Schema versions may add fields while preserving v1 ordering and nullable semantics.

**`RoundObservation`** — exact normalized Pair user text plus subsequent native progress in one root.
- **Relationships:** A completed observation proposes a root candidate for a provisional binding; later observations intersect an ambiguous candidate set.
- **DRY rationale:** Live establishment and offline post-progress recovery use the identical pure matcher.
- **Future extensions:** New agent progress records only extend normalization, not establishment semantics.

**`Binding`** — provisional, established, ambiguous, or unbound relation between `(scope, tag, agent)` and a nullable native root.
- **Relationships:** One established root propagates to descendants through validated parent edges; parentage is never independent evidence and cannot resolve ambiguity.
- **DRY rationale:** Watcher, config, ledger, CLI, restart, and recovery need one lifecycle and one stable `binding_id`.
- **Future extensions:** Additional durable evidence kinds can be introduced only through explicit schema evolution.

**`Candidate` / `Ambiguity` / `Evidence`** — deterministic explanation of roots
that do or do not qualify and why, using only ledger, live-round,
offline-round, or config evidence.
- **Relationships:** Candidates are intersected by later completed rounds; ambiguity retains all equal qualifying roots.
- **DRY rationale:** CLI diagnostics and watcher decisions expose the same candidate set.
- **Future extensions:** New evidence requires an explicit precedence and schema change.

**`PairLedgerFact` / `PairLogFact`** — normalized durable Pair facts used by the pure matcher.
- **Relationships:** Ledger facts restore established bindings; ordered log facts support only the narrow post-progress/pre-ledger offline reconstruction window.
- **DRY rationale:** Launcher and watcher currently duplicate ledger shapes while slug/continuation paths duplicate Pair-log normalization.
- **Future extensions:** A future ledger version can be adapted once at this boundary.

**`SessionActivity`** — inventory-owned internal projection of authorized root
creation and last-activity times, with timestamp source.
- **Relationships:** Derived only after inventory authorizes a root artifact;
  consumed by titlepoller and Neovim age/idle hints without native path formulas.
- **DRY rationale:** Preserves existing age/idle behavior while keeping native
  path resolution and stat ownership inside the inventory boundary.
- **Future extensions:** May enter a later public schema version; schema-v1 stays unchanged.

### Integration Points

| Name | Lives in | Status | Introduced | Wraps |
|------|----------|--------|------------|-------|
| `Runtime` | `cmd/internal/sessioninventory/runtime.go` | new | M1 | all filesystem, SQLite, process, ledger, and config reads |
| `OSRuntime` | `cmd/internal/sessioninventory/runtime_os.go` | new | M1 | host filesystem, `procutil`, platform `sqlite3` |
| `FakeRuntime` | `cmd/internal/sessioninventorytest/fake_runtime.go` | new | M1 | mutable portable external state |
| `WatcherInventory` | `cmd/internal/sessionwatch/sessionwatch.go` | modified | M2 | baseline, polling, round establishment, ledger persistence |
| `session-inventory` CLI | `cmd/internal/sessioninventory/runcli.go` | new | M2 | stable human/JSON/conformance rendering |
| `activity query` | `cmd/internal/sessioninventory/activitycli.go` | new | final | authorized age/idle projection for editor UI |

**`Runtime`** — the only boundary allowed to enumerate/read/stat native storage, obtain typed SQLite facts, inspect descendants/open files/PID identity, or read Pair state.
- **Injected into:** scanner, watcher, inventory, and conformance entry points.
- **Future extensions:** Agent-specific APIs can be added as typed facts without leaking effects into pure logic.

**`OSRuntime`** — production implementation. It reuses `procutil`, rejects symlink escapes, bounds record reads, and invokes `sqlite3` read-only for Agy behind the seam.
- **Injected into:** production CLI and watcher.
- **Future extensions:** A native SQLite library can replace the command without changing callers.

**`FakeRuntime`** — importable stateful in-memory model of storage, SQLite,
process, open-file, ledger/config-read, failure, and mutation state, including
unordered results and PID reuse.
- **Injected into:** production inventory entry points in tests.
- **Future extensions:** The sibling `sessioninventorytest` package is reused by
  watcher/launcher tests; their adapter-only write/scheduling fakes stay local.

**`WatcherInventory`** — starts at launch, captures a baseline, records provisional state, and monitors exact completed rounds. It never chooses first/newest.
- **Injected into:** launcher create/restart flow.
- **Dependency direction:** `sessionwatch.Runtime` keeps watcher scheduling and
  Pair ledger/config writes, embeds/adapts one `sessioninventory.Runtime` for
  reads, and deletes duplicate native walk/read/process/open-file methods.
- **Future extensions:** Watch notifications can expose progress without changing establishment.

**`session-inventory` CLI** — `pair session-inventory [--agent ...] [--scope current|all] [--json] [--conformance]` with schema-v1, redacted conformance, coded diagnostics, and specified exit statuses.
- **Injected into:** Pair's Go dispatcher.
- **Future extensions:** Machine consumers can depend on versioned JSON rather than native storage.

**`activity query`** — internal `pair session-inventory --activity --agent
<agent>` mode resolving current `(scope,tag,agent)` from Pair environment,
requiring its established binding, and returning only authorized
`created_at`/`last_activity_at` timestamps as JSON; it does not alter schema-v1
inventory output.
- **Injected into:** Neovim age/idle hint; titlepoller calls the same Go query directly.
- **Future extensions:** May become a versioned public projection if another consumer appears.

## Review Boundaries

- **M1 — deterministic forests:** pure model/order, runtime seam/fake, four scanners, fixtures, rendering core, and live conformance.
- **M2 — round-gated bindings:** shared event/log normalization, live and offline round matching, watcher persistence, and public CLI.
- **Final issue close — migration:** every Go, shell, launcher, and Neovim consumer uses the inventory; shadow sweep, atlas, and full verification pass.

Every commit command in this plan includes a body explaining why and ends with
the required `Co-Authored-By: OpenAI Codex <codex@openai.com>` trailer, even
when the abbreviated command below shows only its subject.

## Chunk 1: Deterministic Native Forests

### Task 1: Define the pure forest and total-order model

**Files:**
- Create: `cmd/internal/sessioninventory/model.go`
- Create: `cmd/internal/sessioninventory/order.go`
- Test: `cmd/internal/sessioninventory/model_test.go`
- Test: `cmd/internal/sessioninventory/order_test.go`

- [ ] Write failing table tests for roots, descendants, missing parents, conflicting parents, duplicate native IDs, symlink-escape diagnostics, missing timestamps, and shuffled input.
- [ ] Assert byte-identical ordered model output for every permutation, with null-last timestamps and tuple comparators ending in stable IDs/paths.
- [ ] Run `go test ./cmd/internal/sessioninventory -run 'Test(BuildForest|StableOrder)' -count=1` and confirm it fails because the package/model does not exist.
- [ ] Implement only the pure structs, stable-ID helpers, edge validation, diagnostics, and total-order functions needed by the tests.
- [ ] Keep disputed nodes unbound and emit `parent_conflict` or `duplicate_conflict` instead of choosing.
- [ ] Re-run the focused tests and confirm they pass.
- [ ] Commit: `git add cmd/internal/sessioninventory && git commit -m '#155 M1: define deterministic native forests'`.

### Task 2: Add the single runtime seam and stateful fake

**Files:**
- Create: `cmd/internal/sessioninventory/runtime.go`
- Create: `cmd/internal/sessioninventory/runtime_os.go`
- Create: `cmd/internal/sessioninventorytest/fake_runtime.go`
- Test: `cmd/internal/sessioninventorytest/fake_runtime_test.go`
- Create: `cmd/internal/sessioninventory/scan.go`
- Test: `cmd/internal/sessioninventory/scan_test.go`

- [ ] Write failing tests that drive the production inventory entry point with unordered walks, unreadable records, an SQLite fact set, changed PID identity, descendants, and concurrent open-file evidence.
- [ ] Define typed operations for `Walk`, bounded `ReadRecord`, `Stat`, `SQLiteFacts`, `Descendants`, `OpenFiles`, `ProcessIdentity`, and Pair artifact reads; do not expose raw shell execution to pure code.
- [ ] Implement importable `sessioninventorytest.FakeRuntime` as mutable
      external state, not function-call expectations (`ARCH-MOCK`); use
      external-package inventory tests to avoid an import cycle.
- [ ] Implement `OSRuntime` using existing `procutil` and a read-only platform `sqlite3` adapter; validate paths before reads and centralize storage-root knowledge.
- [ ] Run `go test ./cmd/internal/sessioninventory -run 'TestInventoryRuntime' -count=1` and `go test ./cmd/internal/sessioninventorytest -count=1`; confirm both pass.
- [ ] Commit: `git add cmd/internal/sessioninventory cmd/internal/sessioninventorytest && git commit -m '#155 M1: add stateful inventory runtime seam'`.

### Task 3: Pin and scan all four native shapes

**Files:**
- Create: `cmd/internal/sessioninventory/scan_claude.go`
- Create: `cmd/internal/sessioninventory/scan_codex.go`
- Create: `cmd/internal/sessioninventory/scan_agy.go`
- Create: `cmd/internal/sessioninventory/scan_muse.go`
- Create: `cmd/internal/sessioninventory/usage.go`
- Create: `cmd/internal/sessioninventory/usage_test.go`
- Create: `cmd/internal/sessioninventory/forest_projection.go`
- Test: `cmd/internal/sessioninventory/scan_claude_test.go`
- Test: `cmd/internal/sessioninventory/scan_codex_test.go`
- Test: `cmd/internal/sessioninventory/scan_agy_test.go`
- Test: `cmd/internal/sessioninventory/scan_muse_test.go`
- Test: `cmd/internal/sessioninventory/forest_projection_test.go`
- Test: `cmd/internal/sessioninventory/conformance_live_test.go`
- Modify: `Makefile.local`
- Fixtures: `cmd/internal/sessioninventory/testdata/native/{claude,codex,agy,muse}/v1/`

- [ ] Pin the Claude root/subagent/user/progress/malformed fixtures; add
      `TestScanClaudeV1` including accepted root-only usage and rejected
      sidechain/synthetic usage, run red, implement `scan_claude.go`, rerun green.
- [ ] Pin the Codex root/`parent_thread_id`/user/progress/malformed fixtures;
      add `TestScanCodexV1` including accepted `token_count` and null-info
      rejection, run red, implement `scan_codex.go`, rerun green.
- [ ] Pin the Agy root/transcript/near-miss-orphan/malformed SQLite facts; add
      `TestScanAgyV1`, run red, implement `scan_agy.go`, rerun green. Treat the
      checked-in v1 child as an orphan diagnostic unless an actual sanitized
      populated parent fixture proves the relationship; widen only then.
- [ ] Pin the Muse root/subagent/user/progress/malformed fixtures; add
      `TestScanMuseV1`, run red, implement `scan_muse.go`, rerun green.
- [ ] Add internal forest-projection goldens and shuffled-input tests, run red,
      then implement the stable forest-only projection used by scanner tests.
      Do not create a partial public schema-v1 before M2 binding types exist.
- [ ] Add the opt-in `PAIR_LIVE_NATIVE_SESSIONS=1` no-LLM conformance test; redact content, cwd, home paths, and raw IDs.
- [ ] Add `test-native-session-live` to `Makefile.local` and include it in
      `test-live`. The manual and scheduled workstation command is
      `make test-native-session-live`; no installed sample emits the documented
      skip diagnostic and succeeds, while recognized drift/unreadability/privacy
      leakage fails.
- [ ] Run:
  - `go test ./cmd/internal/sessioninventory -count=1`
  - `go test ./cmd/internal/sessioninventorytest -count=1`
  - `go test ./cmd/internal/procutil -count=1`
  - `git diff --check`
- [ ] Update `atlas/index.md`, `atlas/session-identity.md`, and
      `atlas/architecture.md`, then commit code, tests, atlas, and any
      pre-boundary design log while M1 remains unchecked.
- [ ] Run `sdlc milestone-close --issue 155 --milestone M1 --verified 'all four sanitized scanner fixtures produce complete deterministic forests; shuffled forest projections, runtime failure cases, and redacted conformance pass'`. Let the binary tick M1, update Couch, measure time, and write the close log.
- [ ] Resolve every Critical/Important finding using the printed post-verdict
      protocol; commit fixes plus binary-written artifacts with the exact
      `Review-Verdict` and `Review-Window` trailers.

## Chunk 2: Round-Gated Establishment

### Task 4: Share native events and Pair-log normalization

**Files:**
- Create: `cmd/internal/sessioninventory/event.go`
- Create: `cmd/internal/sessioninventory/event_test.go`
- Create: `cmd/internal/sessioninventory/pairfacts.go`
- Create: `cmd/internal/sessioninventory/pairfacts_test.go`
- Modify: `cmd/internal/slugcmd/slug.go`
- Modify: `cmd/internal/slugcmd/slug_test.go`
- Modify: `cmd/internal/continuationcmd/draft.go`
- Modify: `cmd/internal/continuationcmd/draft_test.go`
- Create: `cmd/internal/sessioninventory/testdata/normalization/v1.json`
- Create: `nvim/normalization.lua`
- Create: `nvim/normalization_test.lua`
- Modify: `nvim/init.lua`

- [ ] Add `TestNormalizeNativeEvents` with one fixture case per accepted
      user/assistant/tool-call/tool-result/terminal-error shape; run it and
      observe missing normalization.
- [ ] Implement only `NativeEvent` normalization and rerun until green.
- [ ] Add the versioned shared normalization golden with CRLF, trailing
      horizontal whitespace, outer blank space, sticky-comment framing,
      internal whitespace, Unicode, empty, generic, and duplicate cases.
- [ ] Add `TestNormalizePairTextGolden` and `nvim/normalization_test.lua`
      against that same fixture; run Go plus `make test-lua` and observe both
      fail before extraction.
- [ ] Extract Pair text normalization for Go and Lua, route `nvim/init.lua`
      send behavior and continuation/slug/matcher callers through it, and rerun
      both suites until byte-identical.
- [ ] Preserve bounded reads and structured malformed-record diagnostics; never inspect transcript content in conformance output.
- [ ] Run:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/slugcmd ./cmd/internal/continuationcmd -count=1`
  - `make test-lua`
- [ ] Commit: `git add cmd/internal/sessioninventory cmd/internal/slugcmd cmd/internal/continuationcmd nvim/init.lua nvim/normalization.lua nvim/normalization_test.lua && git commit -m '#155 M2: share session event normalization'`.

### Task 5: Implement pure causal-round binding and crash recovery

**Files:**
- Create: `cmd/internal/sessioninventory/round.go`
- Create: `cmd/internal/sessioninventory/round_test.go`
- Create: `cmd/internal/sessioninventory/binding.go`
- Create: `cmd/internal/sessioninventory/binding_test.go`
- Modify: `cmd/internal/launcher/ledger.go`
- Test: `cmd/internal/launcher/ledger_test.go`

- [ ] Add `TestSingleTurnQualification`: one normalized turn authorizes only at
      32 UTF-8 bytes, five Unicode word tokens, and a fingerprint unique across
      all remaining Pair segments and native roots; run it and observe failure.
- [ ] Implement the single-turn threshold/global uniqueness rule; rerun green.
- [ ] Add `TestTwoTurnQualification`: two contiguous filtered operator turns
      authorize only when each has eight bytes, at least one has three words,
      and the ordered pair is globally unique; include gaps, partial prefixes,
      duplicate Pair segments, and duplicate roots.
- [ ] Implement the minimal two-turn matcher; rerun green.
- [ ] Add `TestRoundRequiresProgress` covering no round, user-only, assistant,
      tool invocation, tool result, and terminal response/error; implement the
      accepted progress transition and rerun green.
- [ ] Add `TestGlobalBindingConflicts` covering simultaneous multi-tag/root
      matches, one root claimed by competing tags, equal candidates,
      deterministic ambiguity, and later-round candidate intersection.
- [ ] Implement deterministic global assignment/intersection without chronology
      tie-breaking; rerun green.
- [ ] Add `TestBindingPrecedence` for valid ledger > unique live round > unique
      offline round > sole validated legacy config, including stale/malformed
      config and scanner-unauthorized roots.
- [ ] Implement precedence plus `binding_stale`/`binding_conflict` diagnostics;
      rerun green.
- [ ] Write boundary tests for shutdown before progress (provisional/disposable), persistence after progress (established), and crash after progress but before ledger write (offline exact-round reconstruction).
- [ ] Write tests proving validated parent edges propagate an established binding to descendants while carrying both provenances, and cannot create candidates, strengthen evidence, or resolve ambiguity.
- [ ] Keep native matching contiguous after filtering to allowlisted
      operator-authored turns, followed by progress in the same root. Use the
      final issue contract's exact normalization/fingerprint thresholds;
      chronology may order/debug but never authorize.
- [ ] Derive `binding_id` from `(scope_key, tag, agent, root_node_id-or-empty)` and emit only `ledger`, `live_round`, `offline_round`, and `config` correlation evidence.
- [ ] Move/alias launcher ledger parsing through the shared typed facts so watcher and recovery cannot diverge.
- [ ] Run:
  - `go test ./cmd/internal/sessioninventory -run 'Test(SingleTurn|TwoTurn|Round|GlobalBinding|Binding|OfflineRecovery|ParentPropagation)' -count=1`
  - `go test ./cmd/internal/launcher -run TestLedger -count=1`
- [ ] Commit: `git add cmd/internal/sessioninventory cmd/internal/launcher && git commit -m '#155 M2: establish bindings after a completed round'`.

### Task 6: Replace watcher heuristics with provisional-to-established monitoring

**Files:**
- Modify: `cmd/internal/sessionwatch/sessionwatch.go`
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/runtime.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch_test.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/restart.go`
- Modify: `cmd/internal/launcher/restart_test.go`
- Modify: `cmd/internal/launcher/markers.go`
- Modify: `cmd/internal/launcher/markers_test.go`
- Modify: `cmd/internal/launcher/lifecycle_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/agent_restart_test.go`
- Test: `tests/pair-session-watch-test.sh`

- [ ] Add `TestLaunchBaselineExcludesOldRounds`: snapshot both native event
      positions and Pair-log byte/entry ordinal; old matching rounds on either
      side must not establish a fresh launch. Run and observe failure.
- [ ] Implement the two-sided baseline and rerun green.
- [ ] Add `TestWatcherProvisionalUntilProgress` for no progress, one unique
      completed round, ambiguity across polls, and later-round intersection;
      run red, implement the transitions, then rerun green.
- [ ] Add crash tests at the exact pre-progress, post-progress/pre-ledger, and
      post-ledger boundaries; inject each failure, implement idempotent offline
      recovery, and rerun green.
- [ ] Delete `discover`/`discoverByBirth` and all watcher first/newest selection. Process/open-file facts may corroborate candidates where available but may not select a root.
- [ ] Add `TestProcessEvidenceCorroboratesOnly` with PID identity stable before
      and after the descendant/open-file snapshot, PID reuse, conflicting open
      files, and no usable open-file evidence; the last case must still bind
      through a globally unique exact round.
- [ ] Make `sessionwatch.Runtime` retain only scheduling and Pair
      ledger/config writes, adapt one `sessioninventory.Runtime` for all native
      reads, and delete duplicate walk/read/process/open-file methods.
- [ ] Route initial launch and in-pane `freshAgentInvocation` restart through
      the watcher for every supported agent.
- [ ] Add launcher/wrap tests proving a freshly minted Claude
      `--session-id` is only launch input and a provisional scanner candidate:
      it may be passed through the live process's `PAIR_SESSION_ID` as exact
      launch authority, but is not written to config or a non-empty durable
      ledger binding before a completed round. Explicit resume of an
      already-established ID remains direct launch authority.
- [ ] Add a crash test proving the provisional process environment disappears
      with the process and cannot authorize offline recovery without a
      completed round.
- [ ] Make launch metadata rows carry an empty native ID until establishment.
      After a unique round, durably append the authoritative ledger binding,
      then atomically refresh the config cache.
- [ ] Inject config-write failure after ledger append and assert the ledger
      binding remains established while `binding_stale` is reported; do not
      claim cross-file atomicity.
- [ ] Remove `LiveAgentSessionID` from launcher runtime/OS runtime. Make
      restart, markers, compaction, and lifecycle consume only an established
      ledger binding or intentionally start fresh while provisional.
- [ ] Run:
  - `go test ./cmd/internal/sessionwatch ./cmd/internal/launcher -count=1`
  - `go test ./cmd/internal/wrapcmd -count=1`
  - `bash tests/pair-session-watch-test.sh`
- [ ] Commit: `git add cmd/internal/sessionwatch cmd/internal/launcher cmd/internal/wrapcmd tests/pair-session-watch-test.sh && git commit -m '#155 M2: gate watcher binding on native progress'`.

### Task 7: Expose stable inventory and conformance CLI

**Files:**
- Create: `cmd/internal/sessioninventory/runcli.go`
- Create: `cmd/internal/sessioninventory/runcli_test.go`
- Create: `cmd/internal/sessioninventory/inventory.go`
- Create: `cmd/internal/sessioninventory/render.go`
- Create: `cmd/internal/sessioninventory/render_test.go`
- Create: `cmd/internal/sessioninventory/testdata/golden/`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main.go`
- Test: `cmd/pair-go/main_test.go`

- [ ] Add schema-v1 human/JSON goldens now that binding types exist, including
      every required array/null and shuffled ordering; run red, implement
      buffered `render.go`, rerun green.
- [ ] Add `TestRunCLIResultMatrix` for all agents/current/all scopes, partial
      diagnostics, invalid agent, total scan failure, render failure, and
      conformance skip/fail privacy; run red, implement exact flags/exits, rerun.
- [ ] Add dispatcher tests proving `session-inventory` is a buffered Go command
      family rather than launcher argv; run red, register/route it, rerun green.
- [ ] Verify render failure leaves stdout empty and conformance never exposes
      content, cwd, absolute home paths, or raw IDs.
- [ ] Run:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
  - `git diff --check`
- [ ] Update atlas for the establishment boundary, ambiguity, offline recovery,
      and CLI, then commit implementation/tests/atlas and any pre-boundary
      design log while M2 remains unchecked.
- [ ] Run `sdlc milestone-close --issue 155 --milestone M2 --verified 'single/two-turn thresholds, ambiguity intersection, both crash boundaries, provisional launch/restart, watcher persistence, CLI exits, and privacy goldens pass'`. Let the binary tick M2, update Couch, measure time, and write the close log.
- [ ] Resolve every Critical/Important finding using the printed post-verdict
      protocol; commit fixes plus binary-written artifacts with exact review
      trailers before migration.

## Chunk 3: Complete Consumer Migration

### Task 8: Migrate transcript, slug, context, title, and review consumers

**Files:**
- Modify: `cmd/internal/transcript/transcript.go`
- Modify: `cmd/internal/transcript/transcript_test.go`
- Delete: `cmd/internal/codexsid/codexsid.go`
- Delete: `cmd/internal/codexsid/codexsid_test.go`
- Modify: `cmd/internal/contextcmd/contextcmd.go`
- Modify: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/internal/ctxmeter/ctxmeter.go`
- Modify: `cmd/internal/ctxmeter/ctxmeter_test.go`
- Modify: `cmd/internal/titlepoller/titlepoller.go`
- Modify: `cmd/internal/titlepoller/runtime.go`
- Modify: `cmd/internal/titlepoller/run.go`
- Modify: `cmd/internal/titlepoller/run_test.go`
- Modify: `cmd/internal/titlepoller/titlepoller_test.go`
- Modify: `cmd/internal/slugcmd/slugcmd.go`
- Modify: `cmd/internal/slugcmd/slugcmd_test.go`
- Modify: `cmd/internal/reviewcmd/reviewcmd.go`
- Modify: `cmd/internal/reviewcmd/runtime.go`
- Modify: `cmd/internal/reviewcmd/run.go`
- Modify: `cmd/internal/reviewcmd/run_test.go`
- Modify: `cmd/internal/reviewcmd/reviewcmd_test.go`

- [ ] Add transcript tests proving exact inherited `PAIR_SESSION_ID` remains
      direct authority while Codex root validation/path lookup uses inventory;
      run red, replace glob/first parsing, rerun green.
- [ ] Add context/title tests for established, provisional, ambiguous, and
      unbound roots; run red, route transcript/activity queries through
      inventory, rerun green.
- [ ] Replace `ctxmeter.ContextTokens` with inventory `TokenUsage` lookup,
      retain only agent-neutral humanization in `ctxmeter`, and add regression
      tests for Claude/Codex last-usage semantics before deleting its native
      JSONL parser.
- [ ] Add slug/review tests that reject ambiguous roots and eliminate live
      `lsof`/duplicated parser fallbacks; run red, migrate, rerun green.
- [ ] Run `rg -n 'cmd/internal/codexsid' cmd`, remove the last imports, delete
      `codexsid` and its tests, then rerun affected packages.
- [ ] Verify ambiguous/unbound results remain explicit and are never silently converted to newest/first.
- [ ] Run:
  - `go test ./cmd/internal/transcript ./cmd/internal/contextcmd ./cmd/internal/ctxmeter ./cmd/internal/titlepoller ./cmd/internal/slugcmd ./cmd/internal/reviewcmd -count=1`
- [ ] Commit: `git add -A cmd/internal && git commit -m '#155: migrate transcript and review consumers'`.

### Task 9: Migrate remaining launcher, opener, and Neovim consumers

**Files:**
- Create: `cmd/internal/sessioninventory/activity.go`
- Create: `cmd/internal/sessioninventory/activity_test.go`
- Create: `cmd/internal/sessioninventory/activitycli.go`
- Create: `cmd/internal/sessioninventory/activitycli_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/internal/launcher/config.go`
- Modify: `cmd/internal/launcher/config_test.go`
- Modify: `cmd/internal/launcher/history.go`
- Modify: `cmd/internal/launcher/history_test.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/opener/opener.go`
- Modify: `cmd/internal/opener/opener_test.go`
- Modify: `cmd/internal/opener/run.go`
- Modify: `cmd/internal/opener/run_test.go`
- Modify: `nvim/init.lua`
- Modify: `tests/review-toggle-test.sh`
- Modify: `tests/changelog-session-key-test.sh`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [ ] Add `TestSessionActivity` for authorized root creation/mtime, missing and
      ambiguous roots, timestamp source, and shuffled artifacts; run red,
      implement the inventory-owned query, rerun green.
- [ ] Add `TestActivityCLI` for current `(scope,tag,agent)` resolution,
      established output, provisional/ambiguous/unbound refusal, and buffered
      JSON transport; wire `--activity --agent <agent>` through the existing
      dispatcher family and rerun green without changing schema-v1 output.
- [ ] Add launcher tests for config validation, session existence, config
      picker, and history recovery using established inventory bindings; run
      red, remove native path formulas/selectors, rerun green.
- [ ] Add opener tests for environment authority, config fallback, provisional
      state, and changelog keying; run red, route fallback through inventory,
      rerun green.
- [ ] Add Lua/shell tests that obtain age/idle through the buffered activity
      query and prove Neovim no longer runs native-storage `find`, `stat`, or
      transcript parsing; route `session_age_hint` through that query and rerun
      `make test-lua` plus the shell tests.
- [ ] Run:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/launcher ./cmd/internal/opener -count=1`
  - `make test-lua`
  - `bash tests/review-toggle-test.sh`
  - `bash tests/changelog-session-key-test.sh`
  - `bash tests/term-pane-shortcuts-test.sh`
- [ ] Commit: `git add cmd/internal nvim tests && git commit -m '#155: migrate lifecycle and editor consumers'`.

### Task 10: Enforce the shadow sweep and close

**Files:**
- Create: `cmd/internal/sessioninventory/shadow_test.go`
- Modify: `cmd/internal/artifactpath/manifest.go`
- Modify: `cmd/internal/artifactpath/coverage_test.go`
- Modify: `atlas/index.md`
- Modify: `atlas/session-identity.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `atlas/review-workbench.md`
- Modify: `workshop/issues/000155-agent-session-tree-inventory.md`
- Modify: `workshop/projects/couch.md`

- [ ] Write a failing repository scan that enumerates every governed Go, shell, and Lua source root and rejects native-session glob/walk/`find`/`lsof`, first/newest selection, path formula, or independent native parser outside `sessioninventory`.
- [ ] Update artifactpath's manifest/coverage vocabulary for the deleted
      `codexsid` owner, shared ledger/config/native-session ownership, and every
      new resolved consumer; make unclassified source roots fail.
- [ ] Use an explicit shadow allowlist only for inventory implementation and
      fixture/test assertions.
- [ ] Remove every reported shadow consumer and run the scan until clean (`ARCH-DRY`).
- [ ] Run focused verification:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest ./cmd/internal/sessionwatch ./cmd/internal/transcript ./cmd/internal/contextcmd ./cmd/internal/ctxmeter ./cmd/internal/titlepoller ./cmd/internal/slugcmd ./cmd/internal/reviewcmd ./cmd/internal/launcher ./cmd/internal/opener ./cmd/internal/changelogcmd ./cmd/internal/procutil ./cmd/pair-go -count=1`
  - `go test ./cmd/internal/artifactpath ./cmd/internal/dispatcher -count=1`
  - `bash tests/pair-session-watch-test.sh`
  - `bash tests/review-toggle-test.sh`
  - `bash tests/changelog-session-key-test.sh`
- [ ] Run repository verification:
  - `go test ./... -count=1`
  - `make test-lua`
  - `bash tests/term-pane-shortcuts-test.sh`
  - `zellij --config-dir zellij setup --check`
  - `git diff --check`
- [ ] If representative native samples exist, run `PAIR_LIVE_NATIVE_SESSIONS=1 go test ./cmd/internal/sessioninventory -run TestLiveNativeSessionShapeConformance -count=1 -v` and record only the redacted result.
- [ ] Tick only the plain final migration checkbox; sweep the six named atlas
      files and record any legitimate conformance skip. Do not write issue
      status/actual/close-log or Couch actual/closed state by hand.
- [ ] Commit final implementation/artifacts with a `#155` subject and required `Co-Authored-By` trailer.
- [ ] Preview measured time with `sdlc actual --issue 155`.
- [ ] Close with `sdlc close --issue 155 --verified '<focused, full-suite, shell/Lua, zellij, diff, conformance evidence>'`; the binary owns issue status/actual/log and Couch actual/closed mutations.
- [ ] Resolve every Critical/Important finding via the printed post-verdict
      protocol, commit fixes plus binary-written artifacts and exact review
      trailers, then publish with `sdlc push`.
