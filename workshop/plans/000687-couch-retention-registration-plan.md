---
collaboration-mode: discussion
---

# Couch diagnostic protocol registration implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Use superpowers-executing-plans for the warm registration/proof integration; bounded codec work may be delegated after approval. Steps use checkbox syntax.

**Goal:** Let diagnostic retention recognize the Couch supervisor's exact registered incarnation without weakening current-file or reader protection.

**Architecture:** Extend the existing coordinator runtime lease with an explicit diagnostic protocol version. Rooted diagnostic proof consumes that lease through one root-aware inspection adapter, shared by writer maintenance and GC sweeps. The runtime registry owner accepts and preserves additive unknown JSON members; no second registry or process-name exemption is introduced.

**Tech stack:** Go, existing storagegc coordinator/atomic JSON, diagnosticlog inspection/sweep, strictjson duplicate-key validation, portable temporary-root fixtures and owned child processes.

## Evidence and alternatives

Couch already registers its actual PID/birth and couch-runtime role in couchcore.New before exposing its coordinated thread store. That record dates to #239, so its presence alone does not promise the later diagnostic writer protocol. diagnosticlog.VerifyWriters currently recognizes a per-log writer or process-environment protocol flag; the supervisor has neither a per-thread owner nor a matching per-log registration for unrelated current logs. OSRuntime's resolved Pair root is an in-process selection, so guessing its exported environment or HOME would introduce another root resolver.

1. **Recommended:** publish a versioned diagnostic capability on the existing root runtime lease and consume it from the already-selected root. Old leases remain conservative until a new supervisor publishes participation. This has one durable owner and an exact identity join.
2. Treat every existing couch-runtime lease as diagnostic-aware. Smaller, but retrospectively grants a protocol that old producers never declared.
3. Set a global environment exemption or exempt the executable name. Neither binds the declaration to the registered incarnation/root; discard these alternatives.

No arbitrary descendant directories or host HOME are searched. No new backup, retention-age, capacity or process-control policy is added.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Runtime diagnostic capability | cmd/internal/storagegc/coordinator.go; runtimes.go | modified |
| Runtime registry wire preservation | cmd/internal/storagegc/runtime_codec.go | new |
| Exact lease participation predicate | cmd/internal/diagnosticlog/runtime_registry.go | new |

- **Runtime diagnostic capability:** optional integer diagnostic_protocol on ProcessRegistration, with version 1 meaning the current coordinated diagnostic writer contract. Missing/zero and unsupported future versions grant no authority. RegisterRuntime accepts an optional typed RuntimeRegistrationOptions value; existing callers retain their old signature/behavior. Couch explicitly requests version 1 using a shared CouchRuntimeRole constant. Matching existing records upgrade in place without changing their ID. Ordinary registration never clears an already-published capability; positively dead incarnations retain their existing pruning rule. Shared ProcessIdentity remains comparable and unchanged.
- **Wire preservation:** runtimeRegistry keeps its typed Processes projection plus private raw members keyed by registration ID. Custom JSON decode/encode retains unknown envelope, registration and nested process members, merging only known owned fields on rewrite. Public ReadRuntimes returns the validated projection; RegisterRuntime loads and updates the full registry so extensions are not lost. Retired dead rows may be removed whole. Colocated codec tests use byte strings only.
- **Participation predicate:** a registration qualifies only for the supported diagnostic version, Couch role and exact observed PID/birth. It consumes validated values and no IO. Unknown members never supply authority. Colocated table tests include mismatched PID, birth, role and unsupported capability.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Runtime lease read/update | cmd/internal/storagegc/runtimes.go | modified | existing bounded no-follow metadata read and root-locked atomic publication |
| Root-aware diagnostic inspection | cmd/internal/diagnosticlog/runtime_registry.go | new | existing Inspection/SweepSnapshot, coordinator read, ProcessProbe |
| Production proof composition | cmd/internal/diagnosticlog/registry.go; cmd/internal/gcruntime/runtime.go | modified | selected root, ordinary writer maintenance, explicit/scheduled GC |
| Supervisor publication | cmd/internal/couchcore/couch.go | modified | existing identity-fenced RegisterRuntime before coordinated store admission |

The root-aware adapter delegates OpenFiles unchanged. Its runtime observation joins the native snapshot to the selected root's registry, then positively rechecks only matching process incarnations through ProcessProbe before recognizing participation. Missing leases leave runtimes unrecognized; malformed/unreadable metadata or unknown process identity remain inspection uncertainty. No read takes the root mutex or initializes directories, preserving the existing root-then-log lock order. Outside a sweep, every invocation observes fresh evidence. A SweepSnapshot wraps the underlying snapshot and memoizes lease enrichment within that one sweep only; there is no lifetime cache.

EnvironmentOptions uses this adapter for rooted ordinary writer proof. NewWithInspection composes the same adapter for both diagnostic proof functions, retaining the caller's stateful native fake and the existing shared sweep context. Unrooted standalone DefaultProof remains conservative; it cannot manufacture a root. The session collector's existing root-runtime logic is unchanged.

## State, trust and evolution

Actions: inspect preview, retry preview/collection after a transient failure, and have the operator restart onto the new binary.

| Internal state / event | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Old/missing capability or no exact lease | existing unrecognized-runtime block | operator restart onto new binary | none |
| Matching version-1 lease and stable live birth | normal age/size policy | inspect preview | normal eligible diagnostic generations may expire |
| Same PID, different birth; wrong role/root | existing block/inspection uncertainty | inspect preview after runtime settles | none |
| Registry unreadable, malformed or replaced with unsafe inode | inspection uncertainty | retry after metadata repair/runtime settles | none |
| Exact payload has an open holder, even the registered Couch | existing held-file block | retry after holder releases | none |
| Cancellation or native inventory failure | existing yield/error, cursor not advanced as success | retry collection | none |

Publication transitions reuse the existing coordinator transaction: absent -> publish; same identity/role -> retain ID and optionally upgrade capability; live/unknown foreign entry -> retain; positively dead entry -> prune; canceled/failed persist -> existing atomic outcome/reconciliation. No additional background owner or new state machine is created.

ARCH-SECURE/ORDER: a capability is positive protocol evidence, not proof that a file has no readers. Exact holders are checked first and remain authoritative blockers. Recheck PID birth after reading metadata; a late identity change cannot inherit a stale lease. Context cancellation reaches all observations and is checked before returning participation.

ARCH-EVOLVE (hub requirement; ariadne#454 is still in flight in the refreshed base): unknown JSON members survive every surviving-record rewrite at all three depths. Reuse strictjson.RejectDuplicateKeys for one unambiguous JSON value. Keep schema version, required shape, bounded size, strict birth, ID/identity uniqueness, role and target validation. Wrong known-field types, duplicate keys, trailing data and incompatible envelope versions remain errors. Unsupported capability versions are retained but never recognized. Do not relax readStateJSON globally or add extension maps to shared ProcessIdentity.

ARCH-DRY/PURE/MOCK: coordinator owns the wire contract and mutation, the adapter owns lease consumption, and the recognition predicate stays pure. Reuse processWorld, FakeProcessProbe and real temporary coordinator files, extending them only for controllable lease rewrites/identity changes. No function-call-count mock stands in for the stateful lease interaction.

ARCH-CONSTRAINTS: this is startup/maintenance IO, not append/keystroke work. Existing 1 MiB registry and 256-entry bounds remain. For P native runtimes, R lease records and N diagnostic paths, build one identity index and do O(P+R+N) work per sweep. Read the selected registry once, acquire native process/open-file snapshots once, and probe at most one birth per matching leased runtime. N=8/16 tests assert constant reads/native acquisitions/matching probes as N doubles; a later sweep must observe lease removal/replacement. No cap or timeout increases. Network/credentials are not involved.

ARCH-FUNERAL: reuse runtimes.json, its 256-entry bound and positively-dead pruning; no sidecar, daemon, new timer or durable cache. In-memory preserved raw fields die with the loaded registry/sweep.

## Implementation and verification

### Task 1: Preserve the runtime registry across versions

Files: modify runtimes.go/coordinator.go and runtimes_test.go; create runtime_codec.go/runtime_codec_test.go in cmd/internal/storagegc.

- [x] runtimeRegistry.UnmarshalJSON/MarshalJSON: seed and fuzz additive and ambiguous JSON across all three object depths; reject malformed known structure and preserve surviving opaque members through semantic round trips.
- [x] Coordinator.ReadRuntimes/RegisterRuntime: stateful real-file rewrites with controlled liveness and capability upgrades; assert stable IDs, bounded validated reads, unknown-member retention, and no authority or writes on rejected metadata. Replace the old additive-member refusal.
- [x] Implement private runtime wire preservation and full-registry update, then the optional typed diagnostic capability request. Keep old caller behavior and no-op registration writes unchanged.
- [x] Run go test ./cmd/internal/storagegc -run 'Test.*Runtime|TestRuntimeRegistry' -count=1; capture RED/GREEN and commit.

### Task 2: Consume the exact lease through existing proof

Files: create cmd/internal/diagnosticlog/runtime_registry.go/runtime_registry_test.go; modify registry.go and cmd/internal/gcruntime/runtime.go; extend process_inventory_test.go/sweep_test.go and add cmd/internal/gcruntime/runtime_registration_test.go.

- [x] matchesRuntimeRegistration: table-driven exact-identity/capability mutations; only a supported Couch lease for the observed incarnation qualifies.
- [x] runtimeInspection.Runtimes/Snapshot/OpenFiles: stateful portable lease files plus controllable native inventory and process probe; adversarial lease/identity changes and cancellation must fail closed while exact holders remain authoritative.
- [x] runtimeInspection.Snapshot: counted N=8/16 sweeps assert one registry read and native snapshot pair, bounded matching identity checks, and refreshed evidence next sweep.
- [x] EnvironmentOptions and gcruntime.NewWithInspection: run ordinary maintenance and explicit/scheduled collection against the same stateful supervisor-shaped fixture; assert only eligible unheld diagnostics change, with no metadata writes during preview.
- [x] Run targeted diagnosticlog/gcruntime tests and commit only after GREEN.

### Task 3: Publish from Couch and qualify the boundary

Files: modify cmd/internal/couchcore/couch.go/runtime_retention_test.go and cmd/internal/artifactpath/manifest.go; update atlas/storage-retention.md and atlas/index.md only if needed.

- [x] Extend TestCoordinatedCouchRegistersActualRootRuntime to require the supported diagnostic capability and retained ID/idempotency. Reuse the constructor's owned temp roots and verify no owner activity is initialized.
- [x] Have Couch request the capability through the existing RegisterRuntime transaction before admitting the coordinated store. Add both new production files to artifactpath's exhaustive source classification.
- [ ] Run affected full packages plus go test ./cmd/internal/termcmd -run '^TestNoDeclarationCarriesTwoStackedGodocs$' -count=1 and the artifact ownership guard before a recorded suite.
- [x] Use an owned child/isolated root to verify real PID/birth lease recognition, reuse/replacement rejection and held-FD protection; bound and join every owned child. No foreground/native lane interaction.
- [x] Record a read-only real diagnostic preview against the existing running supervisor. Before restart, its old capability may correctly remain blocked; report that as rollout state, not manufactured success.

### Final gates and live arrangement

After approval, run sdlc change-code --issue 687 --flow full before implementation; the wire codec and shared observation adapter are expected to exceed 100 added production lines. Open an early draft PR for CI overlap, run the recorded suite on pushed code, close review, then promptly enter sdlc merge --yes. No live apply, process intervention or pair:0 staging is proposed. Ask the hub to accept the owned conformance/read-only preview now and verify the real supervisor's version-1 lease and updated preview after its next operator-coordinated restart. Any further real deletion requires a separate preview and explicit apply approval.

The recorded suite/CI/merge are workflow gates, not unchecked implementation Plan rows to waive at close. Report the verified landing and restart steps to the hub; the hub owns the project file.

## Revisions

### 2026-10-18 — pre-approval path and flow qualification

Confirmed the existing atlas owner is atlas/storage-retention.md. Pin the full flow because preserving the private wire format and consuming its identity-fenced observation together exceed the quick shell; this keeps the architectural plan review explicit. No behavior or scope change.

### 2026-10-18 — PQ-1 function-level test contract

Compressed prose case inventories into named codec, registry, predicate, adapter and composition functions with adversarial strategies and mechanical guards. Retained integration acceptance and N/2N budgets. No behavior, trust boundary or live arrangement changed.
