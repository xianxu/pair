# Upgrade-safe Couch startup and conversation recovery

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Execute the checked steps with TDD; SDLC owns milestone reviews. Work in pair:0, branch `000346-stale-temporary-store`.

**Goal:** Keep intact Couch stores usable after an OS upgrade, preserve cold-resume authority across device renumbering, and establish/recover valid Codex conversations previously rejected as `vscode` sources.

**Architecture:** Separate registry structure from availability required by GC. Reuse exact-target scanner validation for changed-device proofs. Reuse the watcher’s launch-boundary selection and causal-round matcher for explicit one-shot recovery; ordinary inventory must not discover and silently bind unknown conversations.

**Tech stack:** Go, macOS filesystem metadata, JSONL session ledgers/transcripts, existing stateful runtime doubles and portable fixture stores.

**Status:** SUPERSEDED by the 2026-09-29 probation/TTY-first agreement in issue #346. The implementation steps below are historical, not an executable checklist. Reconcile the architecture, entities, M2/M3 tasks and review boundaries before `change-code`. M1 registry safety remains required; the earlier review verdict applies only to the earlier revision.

## Core concepts

### Pure entities

| Entity | Lives in | Status |
| --- | --- | --- |
| StoreRegistry structural validity | `cmd/internal/storagegc/stores.go` | modified |
| Exact-conversation revalidation eligibility | `cmd/internal/sessioninventory/query.go` or small colocated helper | new |
| Codex root-source classification | `cmd/internal/sessioninventory/scan_codex.go` | modified |
| Current-launch recovery outcome | `cmd/internal/sessionwatch/recover.go` | new |

Registry structure means schema, absolute clean paths, uniqueness and ordering; it makes no claim about unavailable namespaces' references. Conversation identity is `(agent, native UUID)`, already established by the ledger. Revalidation eligibility means exact proof artifact keys and nonshrinking size; filesystem device/inode/generation are cache-continuity hints, not durable conversation identity. Eligibility never authorizes reuse of old scanner state; it authorizes full validation. Recovery outcome distinguishes no unique evidence, already established, candidate, committed and failed/indeterminate; only a current-launch transaction can publish it.

### Integration points

| Integration | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Registry registration and explicit missing-store abandonment | `cmd/internal/storagegc/stores.go` | modified | coordinated registry read/atomic write and filesystem availability |
| GC recovery command | `cmd/internal/gccmd/run.go` | modified | explicit operator choice and registry coordinator |
| Exact proof validation and catalog reuse | `cmd/internal/sessioninventory/query.go` | modified | native runtime and derived catalog |
| One-shot launch recovery | `cmd/internal/sessionwatch/recover.go`, `recovercli.go` | new | native scanner, Pair log, catalog, ledger transaction |
| Recovery-safe binding append | `cmd/internal/sessionledger/store.go` | modified | current launch and existing binding checked under ledger lock |
| Command dispatch | `cmd/internal/dispatcher/dispatcher.go` | modified | `pair session-repair` |
| Isolated process fixture | existing Couch process tests plus a shared fixture helper if needed | modified | HOME/XDG/Pair/Couch roots and inherited artifact variables |

Use `artifactpath` for every scoped path. Do not create a parallel transcript parser, matching algorithm, ledger encoder, or registry lock (ARCH-DRY, ARCH-PURE).

## Decisions and alternatives

1. **Registry:** selected-store registration survives unrelated unavailable registrations; GC remains blocked. Automatic pruning is rejected because missing does not mean empty. Explicit forgetting declares permanent abandonment and resets migration acknowledgment.
2. **Proofs (operator correction):** retain `(agent, native UUID)` as durable conversation identity. Device/inode/generation changes invalidate cached parsing and trigger exact-target full revalidation; they do not revoke the conversation. No volume UUID scheme or special device-number parser is needed. A filename locates a candidate but cannot authorize it without matching internal root identity. A copied/replaced transcript with the same validated conversation identity may be accepted; a different/disputed identity must be refused. Keep the existing nonshrinking proof-prefix requirement, so truncated/restored-shorter data remains a separate explicit recovery case.
3. **Unbound sessions:** recognize the valid `vscode` root source and offer explicit one-shot reconstruction from recorded evidence. Re-launching old watchers cannot repair parked sessions safely because their process-incarnation checks intentionally fail after reboot. Guessing by newest transcript, cwd, or legacy config is rejected.
4. **Scope:** empty slots remain empty. Pair:4 has no known transcript or sent round; report insufficient evidence rather than fabricate a binding. CLI empty-slot name rendering is a separate UI defect, outside this recovery change.

## Constraints, ordering, and trust

- ARCH-PURPOSE: cover all three incident classes and both Couch labels (`binding lost`, `session gone`). Verify the final cold-resume decision, not only helper outputs.
- ARCH-CONSTRAINTS: interactive established queries read only proof-named bodies when full revalidation is needed. A second unchanged query reuses the authorized catalog and must read zero transcript-body bytes. Recovery is an explicit batch command, serial, one owner per invocation. Use existing scanner byte/record bounds; no full-body corpus scan. Bound candidate work at 256 post-boundary artifacts / 256 MiB total observed size and fail with an actionable diagnostic above the cap. These are initial conservative implementation budgets, validated against incident fixtures before acceptance.
- ARCH-SECURE: persisted registry, ledger, catalog, and transcript bytes remain untrusted. Strict parsing and existing safe path readers apply. Reject malformed native IDs, symlinked or missing artifacts, conflicting roots, parent-bearing `vscode`, and mismatched internal native identity. Metadata format changes cause revalidation, not UUID inference. No transcript text in reports or test fixtures copied from user state.
- ARCH-MOCK: existing native runtime and watcher doubles model file metadata changes, transcript replacement, append, and publication failures. Registry/process tests use temporary real stores; no test touches operator HOME or inherited explicit session paths.
- ARCH-ORDER: registry mutation remains inside coordinator lock. Missing-store abandonment resets migration in the same atomic write. Recovery captures the current launch, scans, revalidates the chosen proof, then compares launch and binding under the ledger lock. A new launch or different concurrent binding refuses; same-root publication is idempotent. Unknown write outcome is reconciled using existing ledger commit-outcome handling, never reported as success.
- ARCH-FUNERAL: no new durable artifact family. Registry, ledger, and catalog retain existing retention ownership. Recovery reports are stdout, not growing sidecars. Duplicate recovery does not append duplicate bindings. Temporary test roots are owned by test cleanup; external smoke fixtures use a trap and fully isolated roots.
- No running thread is restarted by a recovery command. In particular, leave brain:0 running. Retention leases protect the selected owner during repair without relying on a stale agent PID.

## Chunk 1: Registry startup and explicit abandonment (M1)

**Files:** `cmd/internal/storagegc/stores.go`, `stores_test.go`; `cmd/internal/couchcore/retention_test.go`; `cmd/internal/couchcmd` process/acceptance tests; `cmd/internal/storagegc/collector_test.go`; `cmd/internal/gccmd/run.go` and tests; `atlas/couch.md`, `atlas/index.md` as needed.

- [ ] Add failing regressions: register normal and auxiliary stores, delete auxiliary, construct/load/save the normal coordinated thread store and list it through Couch. Existing/new normal registration preserves the missing entry; unreadable/aliased selected stores and malformed registries still refuse.
- [ ] Add collector assertions that preview and apply cannot collect eligible payloads when any registered namespace is unavailable. Cover permission failures separately from absence with an injectable availability seam where platform privileges make chmod unreliable.
- [ ] Split structural validation from full availability validation. `RegisterStore` checks structure plus the selected existing canonical store; `ReadRegistry` and `CompleteMigration` retain full inventory validation. Audit all `loadRegistry` callers so availability requirements are explicit.
- [ ] Add `pair gc --forget-missing-store <exact-path>` and coordinator method. It is exclusive with register/complete/apply operations, requires an exact clean absolute registered path and confirmed ENOENT, rejects existing paths/symlinks/unknown paths/permission errors, preserves other entries, atomically sets `MigrationComplete=false`. Retry of an already removed entry reports no such registration rather than silently acknowledging a new inventory.
- [ ] Test GC remains blocked after forgetting until an independent `--complete-migration --store ...` acknowledges the remaining inventory. Document remount/restore as the alternative to permanent abandonment; never infer safe deletion from `/tmp` spelling.
- [ ] Add an isolated smoke recipe/helper setting HOME, XDG_DATA_HOME, PAIR_DATA_DIR and COUCH_STORE_DIR consistently and removing inherited explicit artifact-path overrides. Run a sentinel test proving the operator-shaped registry remains byte-identical. Record that the historical scratchpad invocation has not been identified.
- [ ] Run `go test ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/couch -count=1`, then the relevant packages with `-race`.
- [ ] Update atlas and issue Log, commit with `#346 M1`, then `sdlc milestone-close --issue 346 --milestone M1 --verified '<actual evidence>'`.

## Chunk 2: Durable conversation identity and proof revalidation (M2)

**Files:** `cmd/internal/sessioninventory/query.go`, `query_test.go`; existing `runtime_os_test.go` / `filemeta_darwin.go` documentation; `cmd/internal/couchcore/parkedproducers_test.go` and cold resume tests; `atlas/couch.md`.

- [ ] Write a failing production-query regression: create proof for `dev:10:ino:20`, restart the query runtime with `dev:11:ino:20`, unchanged native ID and valid root body; require established status. Test both same-size and grown bodies. Delete derived catalog first so it cannot hide the failure.
- [ ] Add same-conversation positive cases for changed inode, changed generation, and unknown/changed metadata identity format; each must fully revalidate before success. Add negative cases for shrink, missing/symlinked target, different internal UUID, disputed root, source/parent contradiction, and mutation during reread. A metadata predicate or matching filename alone must never produce success.
- [ ] Extend the existing full-target fallback after incremental validation fails, removing filesystem-identity equality as durable authorization. Require the complete exact proof artifact set and nonshrinking size, then reread those artifacts from byte zero. Require the same agent/native UUID/root role/scanner schema and undisputed state. Preserve #328 behavior; do not add a platform-specific ID parser or rewrite old ledger identities.
- [ ] Permit catalog reuse for a fully validated same-conversation target only when artifact keys, authorized undisputed root/schema, size/parser lower bounds and current exact fingerprint agree. Catalog entries need not retain the filesystem identity of the original proof. A changed current fingerprint returns through validation. Test repeated unchanged query reads zero body bytes; loss/corruption of catalog forces reread and cannot grant authority. Do not rewrite the ledger to make a read succeed.
- [ ] Add Couch classification and resume-precondition regressions for both verified-park and no-receipt records. Both become cold-resumable with a validated surviving transcript; neither requires a live process/session. Contradictory transcripts remain refused.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/couchcore -count=1` and `-race`; mutation-check by removing the revalidation call so the boundary regression fails.
- [ ] Update atlas and issue Log, commit with `#346 M2`, then close M2 through SDLC with actual evidence.

## Chunk 3: Codex source support and existing-launch recovery (M3)

**Files:** `cmd/internal/sessioninventory/scan_codex.go`, `scan_codex_test.go`; `cmd/internal/sessionwatch/run.go`, `lifecycle.go`, new `recover.go`, `recover_test.go`, `recovercli.go`, `recovercli_test.go`; `cmd/internal/sessionledger/store.go`, `store_test.go`; `cmd/internal/dispatcher/dispatcher.go` and tests; artifact consumer registry if new path consumers require it; `README.md`, `atlas/couch.md`.

- [ ] Add a sanitized fixture using Codex `source:"vscode"`. Prove it is a root only without a parent; unknown source, internal source, malformed subagent, and vscode+parent remain rejected. Reference the official SessionSource enum; do not infer parenthood from cwd or filename.
- [ ] Add watcher regression through real launch boundary, Pair send, native progress and proof append: vscode now binds and config reflects the same root. Seed an old catalog containing rejected/unknown vscode state and require fresh target validation, so the refactor cannot retain obsolete rejection. No-progress, duplicate matching roots and pre-boundary transcripts cannot bind.
- [ ] Accept `vscode` in `codexRole`'s existing parent-free root branch. Factor the existing one-pass scan/round matching/proof preparation for reuse; keep the long-running watcher's process checks and cadence intact. Separate proof publication from its live watcher config callback; repair must not invoke the argv-writing config path.
- [ ] Add `pair session-repair --scope-key S --tag T --agent A [--root GLOBAL_PAIR_ROOT] [--apply]`. Default is preview. Resolve paths through `artifactpath.Address` from the global root, validate exact owner fields and supported agent, and do not inherit an unrelated thread's PAIR_TAG or explicit artifact paths. Existing root default follows HOME/XDG; `--root` provides a fully isolated fixture/operator override.
- [ ] For an unbound version-2 current launch, select only post-boundary targets, validate them, use the saved Pair log suffix and existing completed-round thresholds/ambiguity rules, and require a unique proof-bearing result. Preview prints owner, launch, native ID and diagnostic outcome only; no durable writes or authored text. Bound/conflicting/legacy launches are reported explicitly, not reassigned. Apply recomputes evidence rather than trusting an earlier preview.
- [ ] Add a recovery-specific ledger transaction alongside existing append APIs: under the same ledger lock, require unchanged owner/launch; refuse a different existing root/conflict; return existing same-root binding idempotently; otherwise append the fully validated proof. Preserve committed/uncommitted/indeterminate outcomes and reconciliation. Never weaken ordinary conflicting-evidence detection by globally changing append semantics.
- [ ] Before apply, revalidate the selected proof and current Pair suffix; refuse changed evidence. Publish catalog through the existing safe writer and require a retention lease for the addressed owner. Repair writes only ledger/catalog; preserve existing compatibility config byte-for-byte and leave missing config absent. Its CLI has no launch argv and must never erase saved options or refresh a later launch's config. Prove production Couch cold resume uses repaired ledger authority and the saved thread launch profile without a config rewrite. Handle a concurrent watcher, new launch, crash after append, failed catalog publication, and repeated apply in stateful tests.
- [ ] Exercise CLI dispatch and read-only preview, isolated roots, no evidence (pair:4 shape), ambiguous roots, malformed log, source contradiction, existing binding, stale launch, competing different binding, and retry after an indeterminate write. Use the same APIs as production, not a fake that unconditionally accepts bindings.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/dispatcher ./cmd/internal/couchcore -count=1` plus `-race`, then `go test ./cmd/... -count=1` with test-owned roots and the repository's normal build (`make build`).
- [ ] Update docs and Log; close M3 through SDLC. Use freshly built `couch --list` to verify the 15 device-renumber cases and preview the four audited Codex recoveries. Apply only uniquely validated recoveries through the supported command; do not stop brain:0 or fabricate a pair:4 conversation. Verify post-recovery inventory/cold authority and record residual insufficient-evidence outcomes.
- [ ] Close issue through `sdlc close --issue 346 --verified '<actual evidence>'`; publish the completed branch through SDLC PR/merge. Do not mark complete while a committed requirement is unmet.

## Verification notes

Run red tests before implementation, inspect failures, then run focused green suites before broad tests. Keep process tests isolated from inherited session artifact variables. A full live Mac reboot is not required to reproduce this bug: the stateful runtime changes device identity between persisted-proof creation and a fresh resolver instance. Use real metadata-only inspection of the affected transcripts as a conformance check, without changing their contents. Test reports must distinguish proof recovery from merely seeing a live session.

## Revisions

### 2026-09-29 — UUID is conversation identity

Operator pointed out that native transcript UUIDs already provide conversation identity; volume information is unnecessary for scoping/deduplication. Replaced the proposed device-only/same-inode exception with exact-conversation revalidation whenever filesystem continuity is lost. Device/inode/generation remain cache hints. Added positive replacement/copy cases with the same validated UUID and negative cases for contradictory internal identity. No implementation has started.

### 2026-09-29 — Fresh review: preserve saved arguments

Reviewer found that reusing `ObserveAndPersist` config refresh with no CLI argv could erase saved options or write stale config after a concurrent launch. Repair is now explicitly ledger/catalog-only; the live watcher retains its config behavior. Added byte-preservation and production cold-resume tests, plus a rejected-vscode old-catalog regression.

### 2026-09-29T09:59:21-07:00 — Superseded by agreed launch observation and TTY ownership

The operator approved the conceptual model now recorded in issue #346: retain a durable UUID resume target independently of file/parser proofs, launch under probation, and observe current-launch exchanges on both existing and new root transcripts. Strong evidence may replace requested A with actual D while retaining history; silence and early Alt+n remain supported. Unknown native source values are open-world, not blanket rejection. Automatically preserve raw TTY and its sidecar on quit and before same-tag reuse; use printable TTY for text features and isolate native telemetry. This supersedes the exact-body-revalidation, conflicting-internal-UUID refusal, closed source allowlist, new-file-only recovery and immutable-existing-binding assumptions in this historical plan. The issue's active Spec/Done when/Plan are the current contract. Detailed implementation planning and review remain pending; no code has been changed.
