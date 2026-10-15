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
- Historical intervals preserve submits recorded before rename without attributing later submissions from a newly registered document at the old pathname. Normal owner deliveries and ledger gaps retain their existing semantics. Prune expired historical intervals only beyond the existing 14-day ledger retention horizon; never silently evict a live registration.
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
- ARCH-FUNERAL: registry and central state persist until explicit forget (operator-approved forever-until-forgotten policy). At most 256 registrations, each shadow at most 4 MiB plus small state; reaching a cap refuses new growth. Forgotten/unreferenced identity directories are removed under registry serialization; the single coordination flock inode is never deleted. Historical aliases expire with the existing ledger retention horizon; per-registration alias cap refuses further renames until pruning/forget. No automatic removal of missing documents.

## Implementation tasks (one atomic close boundary)

### 1. Registry model and persistence

Files: new registry_model.go, registry.go and colocated tests under cmd/internal/sessionsync/; cmd/internal/artifactpath/manifest.go.

- [ ] Write pure transition tests for idempotency, conflicts, rename history/time intervals, forget, validation and growth limits. Run focused tests and observe RED before implementation.
- [ ] Implement the model, typed parsing and portable store with durable atomic writes and persistent bounded flock. Add filesystem tests for failed/uncertain publication, restart reload, corruption and abandoned-preparation cleanup.
- [ ] Add new production source files to artifactpath's exhaustive inventory. Run go test ./cmd/internal/sessionsync ./cmd/internal/artifactpath; commit registry core.

### 2. Shared sync operations

Files: sessionsync/sync.go, file.go, owed.go and existing/new tests.

- [ ] Write failures covering frontmatter-free read/diff/write/status; first registration horizon; register retry retaining unread edits/folds; explicit-owner precedence; ordinary owner ignored; legacy migration; whole-file ownership guard; symlink aliases sharing metadata; rename retaining old submits without taking new occupants' submits.
- [ ] Route registered operations through the registry resolver and stable metadata; retain unchanged legacy route. Extend owed matching with time-scoped path history. Keep fold publication after successful document/shadow writes.
- [ ] Write fake-clock tests at 30 minutes, just over, malformed/future lock time and legacy agent lock; add structured/text age without weakening refusal.
- [ ] Add controlled two-process barriers for write versus rename/forget and same canonical path CAS writers. Run focused tests and race tests; commit integration.

### 3. Public CLI and documentation

Files: couchcmd/session_sync.go, session_sync_test.go, message_deliveries_test.go, run.go, skills/couch/SKILL.md; atlas/couch.md; workshop/lessons.md if review produces a rule.

- [ ] Add failing parser and runtime tests for all four new verbs, JSON payloads/refusals, no-frontmatter submit delivery/owed/fold, and effective owner. Implement CLI through Sync methods, retaining existing envelope/exit semantics.
- [ ] Publish the agreed command/output/error contract to parley.nvim:2; include exact test invocation and fixture semantics. Do not change Parley sources.
- [ ] Document first-submit registration, forget, metadata-only rename/lock responsibility, legacy compatibility and aged-lock ping rule in CLI help, Couch skill and atlas.
- [ ] Run go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd ./cmd/internal/artifactpath and focused races, then make build. Record output and exact HEAD.

### 4. Verify, review, land

- [ ] Commit/push through sdlc pr; run required full suite via sdlc local-check run --name suite on the pushed head. Preserve failures and fix demonstrated regressions; no bypass of CI/local-gates.
- [ ] Arrange the required operator live check through the hub before landing: a plain right-pane note submits, locks on next edit, rejects Couch write, unlocks on submit, and remains owed until write --fold. Stage/build only as the hub authorizes; clear stage after use.
- [ ] Update issue Log/atlas and run sdlc close --issue 573 --verified with exact evidence; the gate owns the mandatory fresh review. Fix Important/Critical findings or dispute with evidence.
- [ ] On SHIP, integrate main, repeat required checks if code changed, sdlc merge --yes; report landed PR and stop. TL owns project updates.

## Approval and coordination

Pending TL plan approval and parley.nvim:2 contract agreement. Operator has approved the project/slice; no implementation starts until the hub approves this concrete plan.

## Revisions

### 2026-10-15 11:02 PDT — TL approval and lifetime capacity

TL approved commit 07265d36b87ef30a549ff1a93a7900fe924d5360 with this authoritative delta: registry capacity is **4096 live registrations / 16 MiB registry**, replacing the draft's 256 / 1 MiB everywhere above. On overflow, refuse register with the exact command `couch --session-sync forget FILE` as recovery advice; never evict. Other bounds are approved. Lifetime metadata cost is at most 4096 bounded document shadows plus small per-document state, until explicit forget; no missing-path cleanup. Public verbs wait for parley.nvim:2 contract agreement.

Live verification delta: **land #573 independently without :0 staging**; backward compatibility is verified by tests. TL rebuilds pair:0 after landing. The operator's plain-note end-to-end live check happens after parley.nvim#332 is staged, with one Couch restart plus nvim restart, owned by the TL. This replaces Task 4's prelanding staging/live requirement for this item; no simulated test will be called operator acceptance.
