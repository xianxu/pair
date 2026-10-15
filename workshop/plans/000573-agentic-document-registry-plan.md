# Agentic document registry implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Make a submitted markdown document use Couch's existing read/diff/write/status and owed-event protocol without modifying its frontmatter.

**Architecture:** A namespace-local registry binds canonical document paths to stable document identities and fallback owners. Registered documents keep agent-owned state and shadows under that identity; the operator's lock stays next to the document. Existing unregistered session-sync documents retain their current sidecars and behavior.

**Tech Stack:** Go, existing sessionsync ledger and durablefile publication, filesystem flock, Couch CLI, Parley caller.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| DocumentRegistration / RegistryState | cmd/internal/sessionsync/registry_model.go | new |
| RegistryEvent / transition | cmd/internal/sessionsync/registry_model.go | new |
| EffectiveOwner | cmd/internal/sessionsync/file.go | new |
| Registered submit matching | cmd/internal/sessionsync/owed.go | modified |
| Operator lock age | cmd/internal/sessionsync/file.go | new |

- One registration owns one current path, one stable ID, a fallback owner and historical path intervals. Registration time is the initial owed horizon for a new plain document. A repeated registration never moves this horizon or changes the baseline/folds. A conflicting fallback owner refuses; explicit couch-owner may override it. Legacy owner is interpreted only on session-sync documents.
- Registry transitions are the only authority for register/forget/rename. Missing + register creates; same binding + register is idempotent; different fallback owner refuses; registered + rename changes its current path, retaining the old path's submit interval; occupied destination refuses; replay of the same rename returns the existing identity. Forget is idempotent and never alters the document or its operator lock. A forgotten plain file refuses ordinary sync operations. An explicitly typed session-sync file still works after forgetting.
- Historical intervals preserve submits recorded before rename without attributing later submissions from a newly registered document at the old pathname. Normal owner deliveries and ledger gaps retain their existing semantics. Historical intervals persist until explicit forget; never infer that an old row has expired from wall-clock age. The fixed history cap refuses additional renames rather than evicting live evidence.
- Lock age uses Parley's RFC3339 time line and an injected clock. Unknown/malformed/future time remains a blocking operator lock with unknown age. Exactly 30 minutes retains the ordinary refusal; greater than 30 minutes adds the age and an instruction to ping the operator. No lock is expired or removed by Couch.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| RegistryStore | cmd/internal/sessionsync/registry.go | new | namespace-local JSON, stable state directories, flock, durablefile |
| Sync document resolver | cmd/internal/sessionsync/sync.go | modified | registry membership, document/lock reads, existing CAS writes |
| Registry CLI | cmd/internal/couchcmd/session_sync.go | modified | register/list/forget/rename, existing JSON envelope |
| Delivery recorder | cmd/internal/couchcmd/message_deliveries_test.go | modified test | real ledger plus in-process message delivery fixture |

- RegistryStore lives under the existing namespace's session-sync directory. A persistent global registry flock serializes metadata changes and registered read/write operations; lock ordering is registry first, then the existing document flock when importing legacy state. All commands reload membership under this guard, so a queued write cannot resurrect a forgotten or moved registration. Existing session-sync operations keep their legacy flock behavior. No background worker or external service is introduced.
- New registrations have a random stable ID; central shadow and state are prepared before registry publication. When registering an existing session-sync document, import its exact state/shadow under its legacy flock, preserving Since, folds, gap acceptance and LastWrite; leave its old companions untouched. An atomic registry publication admits the prepared identity. Unreferenced preparation directories are reclaimed by subsequent registry mutations under the same global guard. A rename only changes the atomic registry record, so it cannot partially move a shadow/state pair.
- A publication failure may have crossed rename before directory fsync failed. Repeating register/rename/forget revalidates the resulting record and confirms directory durability before returning success. It does not infer failure from a lost acknowledgment. No successful response precedes durable metadata publication.
- The filesystem is the stateful test backend: tests use isolated temporary namespace folders and real documents, sidecars and subprocess flocks. The existing in-process broker/runtime fixtures cover actual CLI and ledger integration. No network fake is needed.

## Caller contract (to agree with parley.nvim:2)

```
couch --session-sync register FILE --owner SLOT [--json]
couch --session-sync list [--json]
couch --session-sync lookup FILE [--json]
couch --session-sync forget FILE [--json]
couch --session-sync rename OLD NEW [--json]
```

- Register requires an existing regular markdown file and an exact slot address; canonicalize absolute paths through symlinks. It does not alter the document, take the operator lock or submit a message. The caller registers durably before sending its unchanged submitted: absolute-path message. Failed registration prevents submission.
- JSON extends the existing capitalized Code/Error envelope with Registration (ID, File, Owner, RegisteredOwner, RegisteredAt), Changed (boolean) or Registrations (array). Owner is the effective owner observed from the document/registry. Success is exit 0, Code ok. lookup is read-only and does not require recording: Code ok, Registration null means positively unregistered; nonzero/unavailable never means absent. list returns persisted identities and fallback owners without reading every document; lookup/register resolve effective owner for the selected document. Distinct exit-1 refusal codes include owner-conflict, target-conflict, not-registered, registry-invalid, registry-busy and registry-limit. registry-uncertain means publication may have occurred: retry the exact same verb/arguments to confirm, never rollback based on this response alone. Changed false identifies an idempotent replay/no-op. Existing operator-lock exit 3 and stale exit 6 remain unchanged. Ordinary status keeps its existing fields; additive lock-age fields are optional.
- Rename is metadata-only after the caller successfully saves/moves the file. NEW must exist; OLD may be absent or still exist after saveas. Resolve symlinked ancestors for missing OLD, rather than falling back to its lexical spelling; retain observed lexical spellings on registration if needed for a removed symlink. Parley owns the filesystem move, .side directory/backlinks and relocating/releasing its operator lock before exposing the new buffer; subsequent Couch writes observe NEW.lock. Couch never moves the source document, .side directory or the lock. The agent uses the same verb after its own filesystem move. Historical paths locate retained owed events, not writable aliases to an old source file.
- Forget works even if the document has been deleted; it removes registration and central state, never the document or operator lock. List works without a recording supervisor and does not require every registered path to exist. Existing legacy files need no registration.
- Explicit couch-owner wins. Ordinary business owner frontmatter on a plain markdown file is not routing authority. Whole-file writes of registered plain documents need not add frontmatter, but must preserve effective ownership; legacy session-sync whole-file writes retain their current type/owner gate.

## Operating envelope and invariants

- ARCH-DRY/PURE: reuse ledger, gap/fold, CAS and durable publication; model registry transitions and owner/age/submit decisions independently of IO. No second owed ledger.
- ARCH-PURPOSE: sweep read, diff, write, status, full-file preservation, CLI parsing/JSON/help, skill and docs; registration alone is insufficient.
- ARCH-MOCK: portable temporary filesystem backend plus existing real broker seam; no production user files in tests.
- ARCH-CONSTRAINTS: interactive local commands, normally tens of documents. Bound registry input/output to 1 MiB, 256 live registrations and 256 unexpired aliases per document; bound new registered document/shadow bytes to 4 MiB, with visible refusals before mutation. Existing legacy envelope remains unchanged. Bound registry lock acquisition to 2 seconds and diff subprocess execution to 2 seconds; timeouts refuse without stealing locks. No fan-out/background processing. These are initial engineering limits, not measured performance claims; tests cover each refusal.
- ARCH-SECURE: registry JSON, paths, lock bytes and markdown are untrusted persisted inputs. Strictly validate version, IDs, absolute normalized paths, unique ownership, timestamps and size/type before use; unknown version/corruption refuses rather than resetting authority. No credentials. Registry paths must be derived from validated IDs, never persisted arbitrary state paths.
- ARCH-ORDER: state changes go through transition; serialization spans lookup through effects. Test register/register conflict, register/write, write/forget, write/rename, rename retry, repeated filesystem publication failure and process death before/after registry publication. The critical ordering is forget racing a waiting write: the write must reload and refuse. Existing operator/edit race warning remains, rather than claiming flock synchronizes Parley.
- ARCH-FUNERAL: registry and central state persist until explicit forget (operator-approved forever-until-forgotten policy). At most 256 registrations, each shadow at most 4 MiB plus small state; reaching a cap refuses new growth. Forgotten/unreferenced identity directories are removed under registry serialization; the single coordination flock inode is never deleted. Historical aliases persist until forget, bounded by the per-registration alias cap; reaching it refuses further renames and explains recovery without removing evidence. No automatic removal of missing documents.

## Implementation tasks (one atomic close boundary)

### 1. Registry model and persistence

Files: new registry_model.go, registry.go and colocated tests under cmd/internal/sessionsync/; cmd/internal/artifactpath/manifest.go.

- [x] Write unit tests for RegistryState.transition and decodeRegistry using the adversarial strategies below; run focused tests and observe RED before implementation.
- [x] Implement RegistryState.transition, decodeRegistry and RegistryStore.withLock/save with durable publication and bounded flock; verify through the fault-injectable temporary-filesystem strategy below.
- [x] Add new production source files to artifactpath's exhaustive inventory. Run go test ./cmd/internal/sessionsync ./cmd/internal/artifactpath; commit registry core.

### 2. Shared sync operations

Files: sessionsync/sync.go, file.go, owed.go and existing/new tests.

- [x] Write failures for Sync.resolveDocument, EffectiveOwner, OwedForDocument and Sync.forget using the strategies below.
- [x] Route registered operations through the registry resolver and stable metadata; retain unchanged legacy route. Extend owed matching with time-scoped path history. Keep fold publication after successful document/shadow writes.
- [x] Test OperatorLockAge and file.guard with the injected-clock strategy below; add structured/text age without weakening refusal.
- [x] Exercise Sync.withDocument through controlled two-process barriers and independent state invariants as specified below. Run focused tests and race tests; commit integration.

### 3. Public CLI and documentation

Files: couchcmd/session_sync.go, session_sync_test.go, message_deliveries_test.go, run.go, skills/couch/SKILL.md; atlas/couch.md; workshop/lessons.md if review produces a rule.

- [x] Add failing tests for parseSessionSyncCLI and runSessionSyncCLI using the protocol and malformed-input strategies below; implement the five new verbs through Sync methods, retaining existing envelope/exit semantics.
- [x] Publish the agreed command/output/error contract to parley.nvim:2; include exact test invocation and fixture semantics. Do not change Parley sources.
- [x] Document first-submit registration, forget, metadata-only rename/lock responsibility, legacy compatibility and aged-lock ping rule in CLI help, Couch skill and atlas.
- [x] Run go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd ./cmd/internal/artifactpath and focused races, then make build. Record output and exact HEAD.

### 4. Verify, review, land

- [x] Commit/push through sdlc pr; run required full suite via sdlc local-check run --name suite on the pushed head. Preserve failures and fix demonstrated regressions; no bypass of CI/local-gates.
- [ ] Arrange the required operator live check through the hub before landing: a plain right-pane note submits, locks on next edit, rejects Couch write, unlocks on submit, and remains owed until write --fold. Stage/build only as the hub authorizes; clear stage after use.
- [ ] Update issue Log/atlas and run sdlc close --issue 573 --verified with exact evidence; the gate owns the mandatory fresh review. Fix Important/Critical findings or dispute with evidence.
- [ ] On SHIP, integrate main, repeat required checks if code changed, sdlc merge --yes; report landed PR and stop. TL owns project updates.

## Approval and coordination

Pending TL plan approval and parley.nvim:2 contract agreement. Operator has approved the project/slice; no implementation starts until the hub approves this concrete plan.

## Revisions

### 2026-10-15 11:02 PDT — TL approval and lifetime capacity

TL approved commit 07265d36b87ef30a549ff1a93a7900fe924d5360 with this authoritative delta: registry capacity is **4096 live registrations / 16 MiB registry**, replacing the draft's 256 / 1 MiB everywhere above. On overflow, refuse register with the exact command `couch --session-sync forget FILE` as recovery advice; never evict. Other bounds are approved. Lifetime metadata cost is at most 4096 bounded document shadows plus small per-document state, until explicit forget; no missing-path cleanup. Public verbs wait for parley.nvim:2 contract agreement.

Live verification delta: **land #573 independently without :0 staging**; backward compatibility is verified by tests. TL rebuilds pair:0 after landing. The operator's plain-note end-to-end live check happens after parley.nvim#332 is staged, with one Couch restart plus nvim restart, owned by the TL. This replaces Task 4's prelanding staging/live requirement for this item; no simulated test will be called operator acceptance.


### 2026-10-15 11:04 PDT — plan-quality round 1 corrections

PQ-1 (ARCH-ORDER): Forget is an authority handoff, not deletion-first cleanup. For a currently typed session-sync file, under the registry guard and its current canonical legacy flock, export the latest central shadow and State (including historical submit intervals in an additive State.SubmitPaths field) to the current path's legacy companions. Publish both durably before removing the registry entry. The registry remains authoritative after a crash or failure between exports; every new Sync command selects its route under the registry guard, so no new-process legacy reader can observe the intermediate pair as authority. A restart/retry repeats export from the retained central state. Only durable registry removal permits central cleanup. This includes renamed files: export at the current path, never the original path. Missing documents or existing plain documents have no legacy route and can be forgotten without export. Re-registration of a typed file imports State.SubmitPaths into its registry history, preserving the same owed interpretation. Mixed old/new CLI versions concurrently writing the same document are outside the supported operating envelope; unchanged legacy file format/read compatibility is retained.

PQ-2 (ARCH-PURPOSE/FUNERAL): ledger.go trims only above 256 KiB, so rows below that threshold can remain indefinitely. There is **no age-based historical-path pruning**. Retain at most 256 path intervals until explicit forget, and refuse further rename growth with visible advice. Export history when handing a legacy document back, so forget cannot implicitly erase a still-owed pre-rename submit. This supersedes all draft references to expired/unexpired history and pruning, without changing ledger retention policy.

PQ-3: The task prose now points to these named risk strategies rather than duplicating a case inventory. Each target gets colocated tests, RED before implementation, then targeted/race verification.

| Function | Adversarial input class and mechanical guard |
|----------|--------------------------------------------|
| RegistryState.transition | Generated legal/illegal event sequences over multiple path occupants and owner bindings; assert independent identity, no-eviction, horizon and effect invariants. |
| decodeRegistry | Fuzz persisted bytes seeded with truncated/unknown-version/invalid-ID/path/time shapes; typed validation must refuse and never manufacture an empty authority. |
| EffectiveOwner | Table/property tests over unrelated frontmatter keys and explicit/legacy metadata; only declared routing fields can affect the result. |
| OwedForDocument | Generate interval/row permutations with reused pathnames; retained owned events survive rename, and events outside the document's ownership intervals never become its submits. |
| OperatorLockAge / file.guard | Inject a fixed clock and malformed/arbitrary lock time text; boundary comparisons may add advice but cannot permit a write for an extant operator lock. |
| RegistryStore.withLock/save | Temporary filesystem plus injected publisher/directory-sync failures at durability boundaries; retry must confirm the same identity/effect, bounded waits must not steal locks. |
| Sync.resolveDocument / Sync.withDocument | Real temp files, symlink spellings and controlled cross-process barriers; resolve metadata under the shared guard, and delayed mutations must recheck current authority. |
| Sync.forget | Inject failure/crash between each metadata export and registry-removal boundary; restart must use retained central authority until both legacy companions/history are durable. |
| parseSessionSyncCLI / runSessionSyncCLI | Fuzz argv/value boundaries plus stateful temp namespace/runtime/broker; successful envelopes must match committed state and failed/unrecognized probes must never look like confirmed absence. |

Parley accepted the concrete contract, including rename returning effective Registration plus Changed. An unsupported older binary may emit nonzero usage text rather than JSON; only exit 0 + Code ok + explicit Registration:null proves absence. Nonzero, malformed/missing JSON or a missing Registration field means unavailable. Freeze the shared caller contract in atlas/couch-agentic-documents.md, linked from atlas/index.md, and send its commit/path to parley.nvim:2 before public-verb implementation.

### 2026-10-15 11:21 PDT — admitted-submit attribution across delayed delivery

TL explicitly approved this scoped delta after implementation exposed a real ordering gap: Entry.At is when delivery completes, so a submit admitted before a rename can finish afterward and be incorrectly filtered by pathname intervals. Add immutable broker-owned Message.AcceptedAt, stamped at direct admission/enqueue and retained by queue promotion. The recorder persists it as Entry.AddressedAt. Only interval matching uses AddressedAt (falling back to At for older rows); the existing delivery horizon, status display and retention continue to use At. No public command/Parley contract change. A stable submit document ID is outside this bounded fix; the caller still registers and submits before its subsequent rename transaction.

| Pure entity | Lives in | Status |
|-------------|----------|--------|
| Message.AcceptedAt | cmd/internal/couchmessage/model.go | modified |
| Entry.AddressedAt | cmd/internal/sessionsync/ledger.go | modified |
| OwedForDocument | cmd/internal/sessionsync/owed.go | modified |

| Integration point | Lives in | Status | Wraps |
|-------------------|----------|--------|-------|
| Broker admission/enqueue | cmd/internal/couchmessage/broker.go | modified | broker clock and immutable message |
| Delivery recorder | cmd/internal/couchcmd/message_deliveries.go | modified | completion-time ledger append |

Test strategy: TestBrokerAcceptanceTimeSurvivesQueuedPromotion advances an atomic fake clock through real broker admission/promotion; acceptance stays fixed while the deadline changes. TestQueuedSubmitRemainsOwedAfterRenameAndOldPathReuse uses the real message service/ledger and an endpoint barrier: enqueue before rename, reuse OLD for a new registration, then complete delivery; the original document must owe the submit and its replacement must not. Observed RED before the recorder/matcher change, then GREEN. ARCH-ORDER distinguishes address-time identity from completion-time delivery evidence; neither timestamp substitutes for the other.

### 2026-10-15 11:49 PDT — implementation progress checkpoint

Tasks 1–3 are complete; their RED/GREEN, fault-injection, cross-process and race evidence is recorded in the issue Log. Registry core and integration shipped together in the atomic implementation commit rather than separate module commits. The agreed caller contract and effective-owner/rename semantics are frozen in atlas/couch-agentic-documents.md. Task 4 remains pending the required queued full suite, close review and landing. Its live-check row is superseded by the TL-approved postlanding combined check above; no :0 staging or operator acceptance is claimed.

### 2026-10-15 — verification complete, close pending

The required suite passed on 5aea67ee68e0b2103d40305fd28934f6002fde3d in 9m52s; the issue Log retains both the original failure and its test-only monotonic-time correction. All server checks passed on the corrected head. Task 4 full-suite row is complete; close review and landing remain pending. The live-check row remains superseded by the approved postlanding schedule, owned by TL.
