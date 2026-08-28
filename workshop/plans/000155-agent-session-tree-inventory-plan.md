# Deterministic Agent Session-Tree Inventory Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give Pair one deterministic inventory of every supported agent's native root/subagent forest, and establish a Pair tag's root only after Pair observes one completed native user-to-agent round.

**Architecture:** A new `sessioninventory` package separates pure native facts, forest assembly, round matching, binding state, and stable rendering from one injected runtime boundary. Agent scanners emit facts rather than making selection decisions. Before input, Pair persists a content-free provisional launch baseline whose physical ledger ordinal delimits the new Pair-log/native suffix; one locked `sessionledger` store owns all append/fsync writes. The watcher persists a root only when one candidate uniquely completes a durably logged round; validated native parent edges then propagate that established binding to descendants without becoming binding evidence. Every existing native-session consumer migrates to this package, and a shadow-sweep test prevents independent glob/newest/`lsof` logic from returning (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`).

**Tech Stack:** Go standard library, existing Pair launcher/sessionwatch/transcript packages, platform `sqlite3` behind an injected seam for Agy facts, shell integration tests, Neovim Lua tests, sanitized native fixtures, and an opt-in no-LLM live conformance probe.

**Authoritative spec:** `workshop/issues/000155-agent-session-tree-inventory.md`, especially “Round-gated establishment contract.” Earlier send-journal and minted-incarnation passages are explicitly superseded and must not be implemented.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status | Introduced |
|------|----------|--------|------------|
| `NativeRecordFact` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `SessionNode` / `SessionForest` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `ParentEdge` / `EdgeProvenance` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `Diagnostic` | `cmd/internal/sessioninventory/model.go` | new | M1 |
| `Inventory` | `cmd/internal/sessioninventory/inventory.go` | new | M2 |
| `NativeEvent` | `cmd/internal/sessioninventory/event.go` | new | M2 |
| `TokenUsage` | `cmd/internal/sessioninventory/usage.go` | new | M1 |
| `RoundObservation` | `cmd/internal/sessioninventory/round.go` | new | M2 |
| `Binding` / `Candidate` / `Ambiguity` / `Evidence` | `cmd/internal/sessioninventory/binding.go` | new | M2 |
| `PairLedgerFact` / `PairLogFact` | `cmd/internal/sessioninventory/pairfacts.go` | new | M2 |
| `LedgerRecord` / `LaunchBaseline` | `cmd/internal/sessionledger/record.go` | new | M2 |
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

**`LedgerRecord` / `LaunchBaseline`** — append-only `launch` and `binding`
records. A launch record's physical source ordinal is its generation key and
stores only the Pair-log byte offset plus sorted native root/event watermarks;
a binding record references that ordinal and supplies the established root.
- **Relationships:** The newest valid launch ordinal is current for one
  `(scope,tag,agent)`. It supersedes older current state without deleting
  history. Only a binding joined to that latest launch is current.
- **DRY rationale:** Launcher, watcher, restart, and offline recovery share one
  lifecycle projection; timestamps never select a generation.
- **Future extensions:** New record kinds must preserve physical ordinals,
  including ordinals consumed by malformed lines.

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
| `LedgerStore` | `cmd/internal/sessionledger/store.go` | new | M2 | serialized append/fsync for every ledger writer |
| `SessionLogStore` | `cmd/internal/pairlog/store.go` | new | M2 | durable existing-markdown-log append before send |

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

**`LedgerStore`** — sole cross-process ledger writer. It encodes before taking
an exclusive lock, derives the next physical ordinal while locked, appends one
complete JSONL record, fsyncs, then unlocks.
- **Injected into:** launcher and watcher write adapters; readers stay in `sessioninventory`.
- **Failure semantics:** a write/fsync error fails the operation. A partial
  malformed tail consumes its ordinal; retry appends a new record. Binding
  append failure leaves the latest launch provisional for offline recovery.
- **Future extensions:** Compaction must take the same lock and preserve
  current launch/binding joins.

**`SessionLogStore`** — sole durable append path for the existing
`log-<tag>.md`. The streaming `pair session-log append` command reads the
authored body from stdin, resolves the scoped artifact, takes its lock, writes
and fsyncs an atomic replacement, then syncs the parent directory.
- **Injected into:** one `submit_operator_text` wrapper used by every existing
  operator-authored submission path (`send_and_clear` and
  `ship_buffer_and_reset`).
- **Failure semantics:** any append failure preserves the draft, displays the
  error, and prevents submission; therefore every completed native round has a
  durable Pair-log turn without introducing a send journal.
- **Generated prompts:** PairReview readiness, compaction, and PairDoctor use a
  separate `send_generated_prompt` wrapper, never enter the Pair log, and never
  qualify as operator-round evidence.
- **Enforcement:** no call site may invoke low-level `send_to_agent` outside
  those two wrappers; a Lua source test enumerates all calls.

## Non-goals

- No send journal, minted incarnation ID, delivery transaction, semantic/fuzzy
  match, or timestamp-authorized binding.
- No recoverable native ID before a completed round; the provisional ledger
  baseline is content-free delimiting metadata, not recovery state.
- No parent edge may create or strengthen a root candidate.
- No schema-v1 activity field; the internal activity query remains separate.

## Review Boundaries

- **M1 — deterministic forests:** pure model/order, runtime seam/fake, four scanners, fixtures, rendering core, and live conformance.
- **M2 — round-gated bindings:** shared event/log normalization, live and offline round matching, watcher persistence, and public CLI.
- **Final issue close — migration:** every Go, shell, launcher, and Neovim consumer uses the inventory; shadow sweep, atlas, and full verification pass.

Every commit command in this plan includes a body explaining why and ends with
the required `Co-Authored-By: OpenAI Codex <codex@openai.com>` trailer, even
when the abbreviated command below shows only its subject.

## Risky Function Test Strategies

Each implementation step follows RED (add the named function test and observe
the expected missing/wrong-behavior failure), GREEN (minimal implementation),
then the listed focused command. The strategy names the adversarial input class
and mechanical guard; executable fixtures/fuzz seeds own individual cases.

| Production function | Test strategy |
|---|---|
| `BuildForest` | fuzz arbitrary duplicated/conflicting/shuffled facts; fail closed on disputed edges and compare canonical bytes |
| `SortInventory` / `StableID` | property-test permutations, null times, and path aliases; enforce documented total tuples and canonical length-prefix hashing |
| `InventoryWithRuntime` | mutate stateful storage/SQLite/process/open-file facts across calls; all effects cross the one typed runtime |
| `ScanClaude` / `ScanCodex` / `ScanAgy` / `ScanMuse` | fuzz bounded sanitized record streams around each v1 allowlist; unknown/malformed shapes become diagnostics, never facts |
| `NormalizeNativeEvent` / `NormalizePairText` | shared Go/Lua golden plus malformed-record fuzzing; accepted source kinds and exact bytes are identical |
| `QualifyTurnSequence` | fuzz Unicode length/token boundaries and repeated/gapped sequences; only globally unique contiguous fingerprints qualify |
| `ResolveBindings` | property-test shuffled multi-tag/root graphs and precedence; equal conflicts stay ambiguous and parent edges only propagate |
| `ParseLedger` / `CurrentLaunch` | fuzz malformed/interleaved records; physical ordinals are retained and only the latest launch plus joined binding is current |
| `LedgerStore.Append` | subprocess concurrency and injected short-write/fsync failures on a real temp filesystem; lock prevents lost rows and malformed tails consume ordinals |
| `PersistSessionLog` | injected open/write/fsync/rename failures plus concurrent append; submission is enabled only after one durable atomic markdown append |
| `submit_operator_text` / `send_generated_prompt` | enumerate low-level send callers and inject append failure; authored text fails closed while generated control prompts remain non-evidence |
| `ObserveAndPersist` | mutate both baselines, PID identity, candidates, and crash points; stale generations cannot bind and post-progress recovery uses suffixes only |
| `RenderV1` / `RunCLI` | shuffled complete/partial inventories and failing writers; buffer before stdout and enforce byte goldens/result matrix |
| `SessionActivity` / `TokenUsageForRoot` | shuffle authorized artifacts/events and unsupported records; query only an established root and last accepted root usage |
| `transcript.Resolve` / `contextcmd.TranscriptPath` | vary inherited authority and binding states; native validation/path lookup must come only from inventory |
| `resolveLiveCodexTranscript` / `resolveTargetSession` | vary ambiguous/unbound inventories and stale config; never fall back to `lsof` or newest files |
| `HistorySource.latestLedgerEntry` / `resolveSessionID` | vary launch generations and legacy caches; only the current joined binding or exact environment authority wins |
| `ShadowSweep` | scan every governed Go/shell/Lua root with seeded forbidden forms; unclassified roots and independent parsers fail |

## Chunk 1: Deterministic Native Forests

### Task 1: Define the pure forest and total-order model

**Files:**
- Create: `cmd/internal/sessioninventory/model.go`
- Create: `cmd/internal/sessioninventory/order.go`
- Test: `cmd/internal/sessioninventory/model_test.go`
- Test: `cmd/internal/sessioninventory/order_test.go`

- [x] RED: add `TestBuildForest` and `TestSortInventory` from the strategy
      table; run `go test ./cmd/internal/sessioninventory -run
      'Test(BuildForest|SortInventory)' -count=1` and confirm the missing-model failure.
- [x] GREEN: implement `BuildForest`, `SortInventory`, `StableID`, the pure
      entities, and fail-closed edge diagnostics; rerun the focused tests.
- [x] Add seeded fuzz/property targets for those functions and run
      `go test ./cmd/internal/sessioninventory -run 'Test|Fuzz' -count=1`.
- [x] Commit: `git add cmd/internal/sessioninventory && git commit -m '#155 M1: define deterministic native forests'`.

### Task 2: Add the single runtime seam and stateful fake

**Files:**
- Create: `cmd/internal/sessioninventory/runtime.go`
- Create: `cmd/internal/sessioninventory/runtime_os.go`
- Create: `cmd/internal/sessioninventorytest/fake_runtime.go`
- Test: `cmd/internal/sessioninventorytest/fake_runtime_test.go`
- Create: `cmd/internal/sessioninventory/scan.go`
- Test: `cmd/internal/sessioninventory/scan_test.go`

- [x] RED: add `TestInventoryWithRuntime` from the strategy table against an
      importable stateful fake; run the focused package tests and confirm the
      missing-runtime failure.
- [x] GREEN: define the typed `Runtime` operations and implement
      `InventoryWithRuntime` plus `sessioninventorytest.FakeRuntime`; keep raw
      shell execution outside pure code and use external-package tests.
- [x] RED/GREEN: add `TestOSRuntimeBoundaries`, then implement `OSRuntime` with
      existing `procutil`, path validation, bounded reads, centralized storage
      roots, and a read-only platform `sqlite3` adapter.
- [x] Run `go test ./cmd/internal/sessioninventory -run 'TestInventoryRuntime' -count=1` and `go test ./cmd/internal/sessioninventorytest -count=1`; confirm both pass.
- [x] Commit: `git add cmd/internal/sessioninventory cmd/internal/sessioninventorytest && git commit -m '#155 M1: add stateful inventory runtime seam'`.

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

- [x] For each `ScanClaude`, `ScanCodex`, `ScanAgy`, and `ScanMuse` function,
      pin one sanitized v1 fixture corpus, add the strategy-table test/fuzz
      target, observe RED, implement the facts-only scanner, and rerun GREEN.
      Agy parent facts remain orphans until a populated sanitized fixture proves
      the relationship.
- [x] RED/GREEN: add `TestForestProjection` for canonical bytes, then implement
      the stable forest-only projection; do not create partial public schema-v1.
- [x] RED/GREEN: add `TestLiveNativeSessionShapeConformance` and implement the
      opt-in no-LLM probe with redacted output and the specified skip/fail rules.
- [x] Add `test-native-session-live` to `Makefile.local` and include it in
      `test-live`. The manual and scheduled workstation command is
      `make test-native-session-live`; no installed sample emits the documented
      skip diagnostic and succeeds, while recognized drift/unreadability/privacy
      leakage fails.
- [x] Run:
  - `go test ./cmd/internal/sessioninventory -count=1`
  - `go test ./cmd/internal/sessioninventorytest -count=1`
  - `go test ./cmd/internal/procutil -count=1`
  - `go test ./cmd/internal/artifactpath -run '^TestProductionArtifactReferencesAreExactlyClassified$' -count=1`
  - `go test ./cmd/internal/couchcore -run '^(TestIssue149M5DeclarationDispositionSourceSetMatchesMilestoneDiff|TestIssue149M5DeclarationDispositionSetIsClosed)$' -count=1`
  - `git diff --check`
- [x] Update `atlas/index.md`, `atlas/session-identity.md`, and
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
- Create: `cmd/internal/pairlog/store.go`
- Create: `cmd/internal/pairlog/store_test.go`
- Create: `cmd/internal/pairlog/runcli.go`
- Create: `cmd/internal/pairlog/runcli_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`

- [x] RED/GREEN: add `TestNormalizeNativeEvent` and the scanner strategy,
      implement only accepted event normalization, and rerun focused tests.
- [x] RED/GREEN: add the versioned shared `NormalizePairText` golden exercised
      by Go and Lua, extract both implementations, and route slug,
      continuation, matcher, and Neovim send behavior through them.
- [x] RED/GREEN: add `TestPersistSessionLog` plus subprocess concurrency and
      failure injection from the strategy table; implement locked atomic
      append/fsync in `pairlog` and the streaming `pair session-log append`
      route.
- [x] RED/GREEN: add the Lua caller-enumeration and failure tests for
      `submit_operator_text`/`send_generated_prompt`; route
      `send_and_clear` and `ship_buffer_and_reset` through the durable authored
      wrapper, and PairReview/compaction/PairDoctor through the generated
      wrapper. Make direct low-level send calls fail the source test.
- [x] Preserve bounded reads and structured malformed-record diagnostics; never inspect transcript content in conformance output.
- [x] Run:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/slugcmd ./cmd/internal/continuationcmd ./cmd/internal/pairlog ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
  - `make test-lua`
- [x] Commit: `git add cmd/internal/sessioninventory cmd/internal/slugcmd cmd/internal/continuationcmd cmd/internal/pairlog cmd/internal/dispatcher cmd/pair-go nvim && git commit -m '#155 M2: make submitted rounds durably observable'`.

### Task 5: Implement pure causal-round binding and crash recovery

**Files:**
- Create: `cmd/internal/sessioninventory/round.go`
- Create: `cmd/internal/sessioninventory/round_test.go`
- Create: `cmd/internal/sessioninventory/binding.go`
- Create: `cmd/internal/sessioninventory/binding_test.go`
- Create: `cmd/internal/sessionledger/record.go`
- Create: `cmd/internal/sessionledger/record_test.go`
- Create: `cmd/internal/sessionledger/store.go`
- Create: `cmd/internal/sessionledger/store_unix.go`
- Create: `cmd/internal/sessionledger/store_test.go`
- Create: `cmd/internal/sessionledger/store_subprocess_test.go`
- Modify: `cmd/internal/launcher/ledger.go`
- Test: `cmd/internal/launcher/ledger_test.go`

- [x] RED/GREEN: add `TestQualifyTurnSequence` from the strategy table, then
      implement the exact one/two-turn thresholds and completed-progress rule.
- [x] RED/GREEN: add `TestResolveBindings` from the strategy table, then
      implement global precedence, ambiguity intersection, stable IDs, and
      propagation-only parent inheritance.
- [x] RED/GREEN: add `TestParseLedger` and `TestCurrentLaunch`, then implement
      typed `launch`/`binding` records. The launch row stores content-free
      Pair-log/native watermarks; its physical ordinal is the generation key.
      A newer launch supersedes older current state, and a binding is current
      only when it references the latest launch ordinal.
- [x] RED/GREEN: add `TestLedgerStoreAppend` and its subprocess concurrency
      target, then implement the sole locked append/fsync writer and failure
      semantics. Replace launcher parsing/writes with shared records/store.
- [x] Implement offline recovery as `Pair/native suffix after the latest
      launch's stored watermarks → QualifyTurnSequence → ResolveBindings`.
      No timestamp, process order, or older launch can authorize that suffix.
- [x] Run:
  - `go test ./cmd/internal/sessioninventory -run 'Test(QualifyTurnSequence|ResolveBindings|OfflineRecovery|ParentPropagation)' -count=1`
  - `go test ./cmd/internal/sessionledger ./cmd/internal/launcher -run 'Test(Ledger|CurrentLaunch)' -count=1`
- [x] Commit: `git add cmd/internal/sessioninventory cmd/internal/sessionledger cmd/internal/launcher && git commit -m '#155 M2: establish durable launch generations'`.

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

- [x] RED/GREEN: add `TestObserveAndPersist` from the strategy table, then
      implement two-sided baseline capture, completed-round polling, ambiguity
      intersection, PID-before/after corroboration, and injected crash recovery.
- [x] Delete `discover`/`discoverByBirth` and every first/newest selection;
      process/open-file facts corroborate but never select a root.
- [x] Make `sessionwatch.Runtime` retain only scheduling/config-cache effects,
      inject `LedgerStore` for ledger writes, adapt one
      `sessioninventory.Runtime` for all native reads, and delete duplicate
      walk/read/process/open-file methods.
- [x] Route initial launch and in-pane `freshAgentInvocation` restart through
      synchronous `PrepareLaunch` for every supported agent. It captures
      watermarks and durably appends the provisional launch before the agent can
      accept input; the watcher receives that physical ordinal.
- [x] RED/GREEN: add `TestPrepareLaunchAuthority` against competing lifecycle
      generations and authority sources. Only exact live invocation authority or
      a current joined binding may expose a native ID; provisional metadata
      cannot authorize recovery.
- [x] Before appending a binding under the shared ledger lock, verify its launch
      ordinal is still latest. After durable binding append, atomically refresh
      config; cache failure reports `binding_stale` without weakening ledger
      authority.
- [x] Remove `LiveAgentSessionID` from launcher runtime/OS runtime. Make
      restart, markers, compaction, and lifecycle consume only an established
      ledger binding or intentionally start fresh while provisional.
- [x] Run:
  - `go test ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/launcher -count=1`
  - `go test ./cmd/internal/wrapcmd -count=1`
  - `bash tests/pair-session-watch-test.sh`
- [x] Commit: `git add cmd/internal/sessionwatch cmd/internal/sessionledger cmd/internal/launcher cmd/internal/wrapcmd tests/pair-session-watch-test.sh && git commit -m '#155 M2: gate watcher binding on native progress'`.

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

- [x] RED/GREEN: add `TestRenderV1` from the strategy table, then implement
      buffered stable human/schema-v1 rendering and privacy redaction.
- [x] RED/GREEN: add `TestRunCLI` from the strategy table, then implement the
      exact flag/result matrix and buffered dispatcher route.
- [x] Run:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
  - `git diff --check`
- [x] Update atlas for the establishment boundary, ambiguity, offline recovery,
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

- [ ] RED/GREEN: for transcript/context/title, add consumer tests against the
      established-binding query, then remove glob/first/path/native-parser
      fallbacks while preserving exact inherited environment authority.
- [ ] RED/GREEN: add `TestTokenUsageForRoot` from the strategy table, migrate
      context/title to it, and reduce `ctxmeter` to agent-neutral humanization.
- [ ] RED/GREEN: add slug/review consumer tests against ambiguous/unbound
      inventory results, then remove live `lsof` and duplicate parser fallbacks.
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

- [ ] RED/GREEN: add `TestSessionActivity` and `TestActivityCLI` from the
      strategy table, then implement established-binding activity lookup and
      its buffered internal transport without changing schema-v1.
- [ ] RED/GREEN: add launcher/opener consumer tests against established,
      provisional, and legacy projections, then remove native path formulas and
      direct config/history selectors.
- [ ] RED/GREEN: add Lua/shell `session_age_hint` parity tests, then route
      Neovim age/idle through the activity query and remove native
      `find`/`stat`/parser calls.
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

- [ ] RED/GREEN: add `TestShadowSweep` from the strategy table, then implement
      the governed Go/shell/Lua scan outside `sessioninventory`.
- [ ] Update artifactpath's manifest/coverage vocabulary for the deleted
      `codexsid` owner, shared ledger/config/native-session ownership, and every
      new resolved consumer; make unclassified source roots fail.
- [ ] Keep an explicit shadow allowlist only for inventory implementation and
      fixture/test assertions; remove every reported consumer (`ARCH-DRY`).
- [ ] Run focused verification:
  - `go test ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest ./cmd/internal/sessionledger ./cmd/internal/pairlog ./cmd/internal/sessionwatch ./cmd/internal/transcript ./cmd/internal/contextcmd ./cmd/internal/ctxmeter ./cmd/internal/titlepoller ./cmd/internal/slugcmd ./cmd/internal/reviewcmd ./cmd/internal/launcher ./cmd/internal/opener ./cmd/internal/changelogcmd ./cmd/internal/procutil ./cmd/pair-go -count=1`
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

## Revisions

### 2026-08-28 — plan-quality round 1

**Reason:** `sdlc change-code` identified four blocking classes: no durable
launch generation for offline suffix recovery, competing ledger writers,
best-effort Pair-log persistence before submission, and test prose expressed as
case inventories rather than production-function strategies.

**Delta:** add typed provisional launch/binding ledger records joined by physical
ordinal; introduce one locked/fsynced `LedgerStore` and durable
`SessionLogStore`; require synchronous baseline persistence before input and
durable markdown append before send; add non-goals; and replace enumerated test
cases with a named risky-function strategy table plus concise RED/GREEN steps
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-28 — plan-quality round 3

**Reason:** the gate found the durable-before-send rule covered normal draft
send but not the second operator-authored dirty-history submission path.

**Delta:** state and enforce the class rule: `send_and_clear` and
`ship_buffer_and_reset` use one fail-closed `submit_operator_text` wrapper;
generated PairReview, compaction, and PairDoctor control prompts use
`send_generated_prompt` and never become operator-round evidence; no other
direct `send_to_agent` call is allowed (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-28 — installed native-shape conformance

**Reason:** read-only inspection performed for M1 found two current native
shapes outside the pre-implementation prose: Codex child `session_meta.source`
is now string `"subagent"`, and Muse root `run_id` is run-scoped rather than the
root session UUID.

**Delta:** pin both legacy-object and current-string Codex child sources; require
Muse `run_id` equality only for child records while root identity remains
path-owned. Sanitized fixtures and the live conformance probe enforce these
bounded widenings; unknown sibling shapes still fail closed (`ARCH-PURPOSE`,
`ARCH-MOCK`).

### 2026-08-28 — corrected Codex source conformance

**Reason:** the first redacted aggregation collapsed the type of the Codex
`source` field and made an object key look like a string value. A second
type-preserving pass over the installed corpus showed that current children
still use an object source.

**Delta:** withdraw the string-`"subagent"` widening. Codex v1 accepts the
legacy empty/depth-only `subagent` objects and the current exact five-field
`thread_spawn` object. The nested and top-level parent IDs must agree; unknown
keys and string child sources remain near-misses (`ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-28 — M1 boundary-review contract closure

**Reason:** the first M1 boundary review found that focused scanner tests were
green while repository source contracts and several exact Spec tuples were not
yet executable. It also found that the concise implementation names did not
make their relationship to the Core Concepts table explicit.

**Delta:** `Fact`, `Node`, and `Forest` remain the concise implementation names
and now expose `NativeRecordFact`, `SessionNode`, and `SessionForest` aliases;
validated child nodes carry explicit `ParentEdge` and `EdgeProvenance` values
through forest projection. Move artifact-source classification and the
historical source-set disposition into M1. Expand M1 verification to cover the
single diagnostic registry and canonical ID/coalescing order, equal-time node
ordering, regular-file-only enumeration with partial-walk preservation, and
presence-aware token usage. These are boundary invariants, not final-migration
cleanup (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-28 — M1 Core Concepts inventory enforcement

**Reason:** a second `core-concepts-match-code` review found that manually
maintained entity paths can remain inconsistent after implementation aliases
are introduced.

**Delta:** record `NativeRecordFact` at its actual declaration in
`cmd/internal/sessioninventory/model.go`. Source declarations now carry a
bounded #155 concept marker, and an exhaustive contract compares every M1 Core
Concept name, pure/integration kind, status, milestone, and path in both
directions. A future M1 row or marked declaration cannot drift independently
(`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-08-28 — M2 boundary review: established authority closure

**Reason:** the first M2 review found that plain restart could still fall back
to a stale config session ID when the current typed launch was provisional.

**Delta:** include `launcher/markers.go` and its tests in M2 closure. A plain
restart resumes only the established ID carried by the ledger-derived marker;
an empty marker starts fresh, retains saved non-resume arguments, and drops the
compatibility config (`ARCH-PURPOSE`).

### 2026-08-28 — M2 boundary review: diagnostics, ordering, and framing

**Reason:** the review found repeated fail-open classes: one Muse event default,
Pair log/config read failures, raw empty-agent ordering, and delimiter-based
Pair-log framing that could not represent arbitrary authored Markdown.

**Delta:** every versioned event-adapter default and Pair-artifact read boundary
emits a registry-backed diagnostic; every nullable comparator component uses one
null-last projection with exhaustive equal-prefix tests; new Pair-log entries
carry a byte-counted versioned header while the parser retains legacy input
compatibility (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-28 — M2 boundary review: executable public CLI contract

**Reason:** the CLI was user-facing but absent from README, and the checked task
overstated result-matrix/golden coverage.

**Delta:** add README usage and provisional/established semantics; execute every
normal/conformance usage, partial, absent, fatal, privacy, render, and writer row
through injected renderers where failure is otherwise unreachable; pin complete
output bytes in checked-in goldens (`ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-28 — M2 boundary review: gate-owned closure and complete byte oracles

**Reason:** the rerun showed that manually staged Couch actual/closed metadata
violated the repository's pre-close invariant, and branch assertions did not
meet Task 7's stronger checked-in byte-golden promise.

**Delta:** issue, plan, and project closure metadata remain absent until the
successful `milestone-close` transaction writes them. One checked-in matrix
oracle now captures exact exit code, stdout, and stderr bytes for every normal,
conformance, usage, partial, fatal, privacy, serialization, and writer branch;
README coverage is executable from the public CLI's own package
(`ARCH-PURPOSE`).

### 2026-08-28 — M2 gate preflight correction

**Reason:** `sdlc milestone-close` rejected the absence-only interpretation of
the prior review recommendation: its project gate requires the checked Couch
row and load-bearing detail block, including measured actual and date, before
review dispatch.

**Delta:** move the Couch milestone checkbox, actual, date, and detail together
as one consistent preflight state. The repository contract rejects partial
project closure state; the successful binary transaction still owns issue and
plan closure state (`ARCH-PURPOSE`).

### 2026-08-28 — full-range M2 authority, framing, and schema closure

**Reason:** once the review window correctly covered all of M2, it found five
classes hidden by the earlier fix-only ranges: provisional launches still
selected config in the pure resolver; some Pair evidence rejection was silent;
Neovim retained the legacy delimiter grammar; unrelated open files vetoed
portable matching; and internal evidence fields leaked into schema v1.

**Delta:** a current launch suppresses config selection until a typed binding
joins it; every Pair read, owner mismatch, and scanner-unknown ledger/config ID
diagnoses; one tested Lua framing module reads and rewrites legacy plus
byte-counted entries; process corroboration is available only when an open file
maps to a scanner-authorized root; and an independent `evidenceV1` DTO emits
exact documented fields with required non-null arrays (`ARCH-DRY`, `ARCH-PURE`,
`ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-28 — classify the complete Pair evidence rejection boundary

**Reason:** the full-range review found a third instance of silent rejection:
syntactically valid versioned ledger rows naming an unsupported agent. Fixing
only that row shape would leave the same ambiguity at recognized sidecar names
and paths.

**Delta:** classify every recognized Pair evidence input exhaustively. Valid
evidence for a supported requested agent is processed; valid evidence for a
supported but unrequested agent is intentionally filtered; unsupported
versioned enum values, malformed recognized sidecar owners or paths, rejected
ownership, unknown native IDs, and failed reads produce registry-backed
diagnostics. One table-driven regression crosses these filter/reject boundaries
so adapters cannot recreate silent failure independently (`ARCH-DRY`,
`ARCH-PURPOSE`).
