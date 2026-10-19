---
collaboration-mode: discussion
---

# Diagnostic log retention implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use subagent-driven development for bounded independent tasks, or executing-plans for session-warm work. This is one atomic close boundary; task numbers are not milestones.

**Goal:** Bound optional diagnostic residue with a configurable seven-day window and a 64 MiB generation cap, independently of backup and session retention, and report the first safe live reclamation.

**Architecture:** Reuse diagnosticlog's stable per-log lock, exact identity/last-write certificates, process inspection and durable deletion intent. One age/oversize decision replaces the backup fence and dormant count-based admission/eviction. Diagnostic-only collection must not acknowledge Couch migration, initialize session clocks, or delete session/history payloads.

**Tech stack:** Go, portable directory fixtures, injected clock and stateful process inspection; existing Pair GC CLI/scheduler.

**Status:** Full-flow change-code admitted implementation at a21488b4effef56ede5357236754a4111812f807. Implementation and the first approved live sweep are complete; final recorded suite, close and landing remain. Later hub-approved proof refinements are recorded under Revisions.

## Evidence and approaches

The issue Log records the read-only inventory, oversized legacy provenance and passing current append-limit controls. The 4.93 GB example last changed before the October 15 hard-limit fix. Existing tests intentionally preserve sparse 4 GiB legacy generations. New append admission is already bounded; inherited oversized files are the uncovered cleanup policy.

1. **Recommended: age-based eviction plus eviction of proven oversized legacy generations.** Default seven days; no three-generation count override. Evict a whole oversized legacy generation at any age through the existing deletion intent. This can lose recent oversized debug evidence, but requires no bulk payload rewrite or new compaction transaction.
2. **Fence off, keep old count policy:** smaller patch, but three generations can erase most of a busy day's diagnostics. Rejected because it does not mean approximately seven days.
3. **Tail-compaction of oversized generations:** preserves their newest bytes, but copies payload and requires a new crash-recoverable replacement/metadata protocol. Avoid this machinery unless the hub requires preserving young oversized tails.

No automatic deletion of files with unknown or live open holders is authorized. Registered modern writers without an open holder remain coordinated by the existing lock; a live holder is not silently treated as safe just because its registration is known. The first sweep reports those blockers explicitly. Newly admitted data always remains capped; a blocked legacy file is not silently described as compliant.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Diagnostic age/oversize policy and retention-day parser | cmd/internal/diagnosticlog/policy.go, collect.go | modified |
| Effective diagnostic options and retained reason | cmd/internal/diagnosticlog/writer.go | modified |
| Dormant count/debt selection | cmd/internal/diagnosticlog/policy.go | deleted as active policy |

The policy owns one-to-many generations per diagnostic log. It derives age from certified last write; current paths, immutable segments and legacy paths use the same decision. Persisted v2 MaxGenerations remains readable for compatibility but confers no new deletion or append-blocking authority. Keep the existing persisted byte authority and tighten-only behavior. No new diagnostic state machine or payload family is introduced.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Per-log admission, collection and deletion replay | cmd/internal/diagnosticlog/writer.go, pages.go, legacy.go, collect.go | modified | files, stable log lock, durable intent |
| Explicit and scheduled diagnostic traversal | cmd/internal/gcruntime/runtime.go, schedule.go | modified | root coordination, registry and exact-family inventory |
| Public diagnostic-only preview/apply | cmd/internal/gccmd/run.go | modified | CLI, metadata report |
| Wrapper trace measurement | cmd/internal/wrapcmd/diagnostic_turn_budget_test.go | new test | real proxy and owned buffered log |

Use the existing portable file backend. Process/OpenFiles/Runtimes conformance stays behind diagnosticlog.Inspection; regressions use a stateful inventory that changes holders, birth identity and protocol knowledge between preview and apply. Preserve real OS conformance tests. No cloud or backup API is added.

## Contract and visible states

Actions: preview, apply diagnostics, correct configuration, retry after a busy writer releases, or ask the owner to upgrade/relaunch a legacy writer. No restart is performed by this change.

| Internal state/failure | Visible result | Recovery | Possible loss |
|---|---|---|---|
| Certified generation within window and cap | retained with effective age window | preview later | none |
| Certified expired or oversized generation; writer proof succeeds | eligible, then collected with reason/bytes | apply diagnostics | entire selected diagnostic generation, including young oversized legacy data |
| Busy lock or unknown/live legacy writer | blocked; no unlink | retry after release or upgrade/relaunch through its owner | none |
| Identity/metadata changed or malformed | blocked with reason | inspect/correct via owner; preview again | none |
| Interrupted deletion with exact durable intent | recovery pending/replayed under same log lock | apply diagnostics again | only the already-authorized generation; no duplicate byte claim |
| Invalid diagnostic-day setting | explicit configuration error, no collection | correct configuration | none |
| Couch migration unacknowledged | session collection remains blocked; diagnostics independently evaluated | diagnostic-only apply | diagnostic loss only as above; no session/history mutation |

The log lock is held across validation, policy admission and durable deletion effects. Root-before-log order remains unchanged. A stale preview is never unlink authority. Cancellation/death can leave only existing resumable deletion intents; unrelated files and stable lock inodes remain untouched.

## Policy and configuration

- Default seven days; proposed PAIR_DIAGNOSTIC_DAYS accepts integer 1..365. Empty means seven. Invalid, overflow, zero and negative values return an explicit error before optional collection starts.
- Parse once at the GC service boundary; the CLI and scheduled worker consume the same typed duration. Test constructors inject it without ambient HOME/config. A bad value fails closed rather than falling back to a surprising window.
- Replace hardcoded RetentionPeriod uses in selection/reasons/reporting with the effective service policy; preserve a named default for callers/tests.
- For a certified generation and complete writer proof: eligible if last-write age is at least the window OR its size exceeds its persisted byte cap. Unknown/future clocks and invalid identity remain blocked, including oversized cases.
- New records exceeding available room continue to return ErrLimit before any append. Rotation retains the shared cap and existing async maintenance; process scans never move onto per-byte callbacks.
- Remove active count-debt/trimming calls in Write, rotateGeneration, maintain and previewPageLocked. Do not scan a complete 64-generation candidate inventory on every page. Existing page cursors handle any number of age-eligible generations within their fixed budgets.
- Use the existing recoverDeletion for all eviction; do not copy or truncate multi-GiB payloads. Update legacy adoption so a proven oversized legacy file can enter the journal without waiting seven days.

## Task 1 — policy and inherited-size regression

**Files:** diagnosticlog/policy.go, writer.go, collect.go, pages.go, legacy.go; policy_test.go, hard_limit_test.go, legacy_test.go, backup_fence_test.go, deletion_replay_test.go, pages_test.go.

- [x] Add the parser/selection regressions using the named-function strategies below; run targeted tests and retain the expected red against today's backup fence.
- [x] Replace the sparse 4 GiB preservation expectation with explicit oversized eviction through the real journal; no GiB payload allocation. Keep a fresh bounded generation and unrelated file as survival controls.
- [x] Update policy to the rule above and remove active count policy. Keep legacy persisted limits readable and keep new-write whole-record cap tests for synchronous/asynchronous maintenance, multiple handles and absent proof.
- [x] Exercise current, closed, quarantined and legacy paths through one policy; preserve uncertainty rather than granting deletion authority to incomplete metadata.
- [x] Apply the deletion replay strategy below through the existing fault seam; run diagnosticlog tests and commit the verified policy change.

**Named-function test contract:**

| Risky function | Adversarial strategy and mechanical guard |
|---|---|
| ParseRetentionDays(string) | Fuzz hostile scalar grammar against the allowed integer-domain predicate; invalid input returns an error before IO, empty input derives the named default. |
| DecideGeneration(now,lastWrite,size,cap,period) | Generate chronological/size tuples against the independently stated age-or-size predicate; invalid or future evidence never grants eligibility. This is the named pure selector reused by all diagnostic paths. |
| ResolveDiagnosticLimits | Compose conflicting writer requests across persisted versions; effective byte authority only tightens, and dormant count fields never authorize eviction. |
| Open / Writer.Write / maintain | Drive stateful multi-handle writes across cap/maintenance admission; actual file sizes never exceed admitted authority and failed proof produces no over-cap append. |
| previewPageLocked / previewLegacyCurrent | Mutate certified metadata/inodes or holder state through the real file/Inspection seam between observations; changed or unknown evidence stays visibly blocked. |
| VerifyWriters | Vary open-holder identity and protocol knowledge in the stateful inventory; any live open holder or unknown writer blocks destructive collection, including registered holders. |
| collectManagedPage / recoverDeletion | Inject death/cancellation at the existing durable effect seam, then replay with replaced/unchanged identities; only the exact authorized generation can disappear and repeated recovery cannot claim its bytes twice. |

## Task 2 — independent diagnostic traversal and CLI scope

**Files:** gcruntime/runtime.go, schedule.go, runtime_test.go, schedule_test.go, maintenance_budget_test.go; gccmd/run.go, run_test.go/apply_acceptance_test.go; artifactpath manifest if a new source is warranted.

- [x] Add failing portable-root integrations from the entry-point strategies below. Diagnostic-only preview creates nothing; apply affects only certified diagnostics and leaves session clocks/store acknowledgments unchanged.
- [x] Factor diagnostic preview/apply ownership from the existing Service paths. Reuse DiagnosticInventory and registry discovery; remove only diagnostics' migration gates in explicit pages and scheduled batches. Preserve session collection/migration guards and root-before-log locking.
- [x] Add --diagnostics-only for preview/apply; refuse combinations with migration/register/forget-store operations. No new operator command is required for history/session retention. Include effective diagnostic days and separate collected/blocked bytes in JSON/text metadata reports.
- [x] A pre-migration scheduled session phase must yield to diagnostics rather than terminate the maintenance cycle. Preserve cancellation, nonblocking busy handling and cursor progress.
- [x] Run gcruntime/gccmd targeted tests and retention integration, then commit.

**Named-entry-point test contract:**

| Risky function | Adversarial strategy and mechanical guard |
|---|---|
| gcruntime.New / NewWithInspection | Inject malformed environment policy or portable root configuration; invalid policy fails before collection and test construction never reads ambient production settings. |
| Service.PreviewDiagnostics | Feed exact-family discovery with unrelated/unknown entries and unacknowledged session migration; before/after filesystem inventories are identical. |
| Service.ApplyDiagnostics | Change holder/identity evidence after preview through stateful Inspection; only freshly certified eligible diagnostics disappear, with no session/history clock or payload mutation. |
| Service.diagnosticPages | Generate bounded page sequences with partial progress and cancellation; cursors advance or report an explicit blocker, and removed-row accounting excludes prior replay. |
| Service.Batch | Drive durable cursor replay through pre-migration session work and diagnostic phases; diagnostic progress remains reachable without evaluating/persisting session owners on a diagnostic page. |
| gccmd.runWithService | Generate incompatible flag families and service outcomes; reject mixed diagnostic/migration operations before side effects, and successful output reports the effective policy and independent categories. |

**Mechanical bounds:** At N=80 and 2N=160 real small generations, sweep bounded pages and assert exact removed counts/logical bytes, zero payload-copy writes, metadata write bytes/operations proportional to removed generations through ObserveIO, and writer proof calls proportional to pages (not records). Seed more than64 young generations to prove the retired count inventory no longer blocks rotation or prematurely evicts recent data. Existing directory traversal remains capped at128*(limit+1) names per page. At8/16 unrelated session owners, diagnostic-only pages must evaluate/persist zero session clocks, using existing owner probe/persist counters. No full host census is added.

## Task 3 — detail-level measurement, docs and live verification

**Files:** wrapcmd/diagnostic_turn_budget_test.go; README.md; atlas/storage-retention.md, storage-retention-io.md; issue Log. Update help from the same effective policy.

- [x] Replay explicitly delimited owned turns through the production proxy with fixed output chunks, controlled flush barriers and a real diagnostic sink. Measure baseline bytes per turn at N/2N turns, per-chunk event counts, payload/metadata bytes and drop counters. Do not infer turns from stdin reads or idle alerts in the real sample.
- [x] Proposed detail decision: retain current hashed per-chunk evidence within bounded lifetime unless the owned-turn measurement demonstrates it overwhelms the envelope. Record the measured decision and PAIR_WRAP_EVENTS=0 opt-out. Any new default summary/drop policy is a scope revision returned to the hub before code, not silently folded in.
- [x] Document default/range/invalid-value behavior, diagnostic-only preview/apply, reasons, deliberate loss of oversized legacy generations and retained unknown/live-writer blockers in root README and atlas.
- [x] Owned conformance: create aged and young logs with a running modern writer and a held legacy FD. Candidate CLI must collect only eligible/proven files, leave legacy-held files blocked, then collect after explicit release; no operator UI or native lane.
- [x] Build this slot's candidate and run diagnostic-only metadata preview against the real Pair root. Send the hub file and byte counts per eligible/retained/blocked category and the exact blocked reasons BEFORE any live apply. Then run the approved diagnostic-only apply bounded by an owned process deadline. No pair:0 staging or migration acknowledgment. The operator has authorized debug trimming without backup; no authored/history payloads are selected.
- [x] Record before/after diagnostic device+inode/size inventory and exact collected-row bytes, plus partial-progress/blocked reasons if the bound expires. Do not equate mtime estimates with certified eligibility or net concurrent growth with reclaimed bytes. No backup is created and no multi-GiB payload is copied.

## Task 4 — verification and landing

- [ ] Run diagnosticlog, gcruntime, gccmd, wrapcmd and artifactpath/testpair guards; retain meaningful red/green results. Run retention shell/Lua integration and the real diagnostic-only checks above.
- [ ] Open early draft for CI overlap; run one recorded suite on final pushed code. Use the fleet flake rule; never blind retry or weaken deadlines.
- [ ] Close through the mandatory fresh review, fix the finding class, and enter sdlc merge promptly on SHIP. Let merge own main integration and its authoritative suite; do not chase main manually.
- [ ] Verify Git/GitHub/tracker landing, report actual reclaimed/remaining diagnostic bytes and any legacy blocker. #663 remains parked pending operator traces; relate measured results without claiming they explain all device writes.

## Architectural lenses

- **ARCH-DRY:** one policy for current/closed/legacy/quarantined generations; reuse deletion intent, exact-family inventory, inspection and existing root/log locks.
- **ARCH-PURE:** typed day parser and eligibility selection are pure; file/cursor effects remain in the thin existing IO owners.
- **ARCH-PURPOSE:** all diagnostic consumers, explicit/scheduled traversal and inherited oversized data are included; session migration cannot silently defeat the first sweep.
- **ARCH-MOCK:** portable directories plus a stateful Inspection double cover holder changes; owned-process and real-root conformance check the consumed OS seam. No new external service.
- **ARCH-CONSTRAINTS:** optional maintenance remains off terminal callbacks,100-generation pages, bounded directory names and linear counted effects; new payloads64MiB, buffers256KiB, no bulk rewrite, one owner per existing worker.
- **ARCH-SECURE:** classify exact diagnostic families before deletion; persisted state/process replies are untrusted and revalidated under locks; no credentials or transcript bodies enter measurements. Unknown/malformed evidence retains rather than guesses.
- **ARCH-ORDER:** preview is advisory, deletion uses the same lock and durable intent as restart replay; stateful tests force holder/identity change and death/cancellation between effects. No unmanaged background helper survives its owner.
- **ARCH-FUNERAL:** generation creators are unchanged; last consumer is short-term debugging; age/oversize sweeps remove payload/metadata and retire registry records while permanent lock inodes stay stable. No backup fence or dormant count cliff remains.

This is a policy change to a previously deliberate backup/legacy-preservation contract, not evidence of a new cap-bypass syntax class. Existing hard-cap regressions remain; no unrelated cross-repo lint rule is invented before a mechanical cause is demonstrated.

## Revisions

### 2026-10-18 — hub approval and live preview gate

The hub approved the full-flow design, default/range/fail-closed configuration, whole eviction of young oversized legacy generations, bytes-per-turn measurement and proposed detail decision. Its explicit live gate requires sending real preview files/bytes by category and blocked reasons before live apply. Preserve live/unknown open-holder blocking even when the process is registered; no slot restart, focus change or :0 staging is authorized or needed here.

### 2026-10-18 — PQ-1 test-contract refinement

Replace parser/recovery case inventories with named-function adversarial strategies. The class sweep covers pure selection, writer admission/proof, file preview/deletion and each diagnostic service/scheduler/CLI entry point; the N/2N counted acceptance envelope remains. This is a test-contract clarification with no policy or live-arrangement change.

### 2026-10-18 — stale declaration isolation (hub-approved)

The real preview failed before reporting because a structurally valid registry entry's parent repository directory no longer exists. Hub approval receiptb95da9f0-1f04-40c2-bc24-e802a659fa47: validate declaration version/absolute clean path/directory/lock shape first, then allow a missing-parent declaration to be enumerated solely as a blocked row. Per-path canonicalization continues refusing it; never recreate its parent, delete its declaration or touch payload. Malformed declarations still fail the inventory closed. Add paired missing-parent/malformed regressions and a service preview that includes both the blocked path and other valid rows. Native runtime inventory uncertainty remains blocked and must be counted in the real preview.

### 2026-10-18 — one native inventory per sweep (hub-directed)

Submitted hub receipt017891ce-c02f-4b4b-a264-18dd71b240d2 directs fixing repeated native observation, rather than merely extending the120s preview. Each explicit diagnostic sweep and each bounded scheduled batch captures one immutable native open-file inventory across its candidate paths and one process inventory. Per-path proof evaluates that shared observation under the existing exact identity/log lock; new sweeps recapture it, so preview never authorizes apply. Native observation failure is cached as uncertainty, never as empty inventory. Ordinary writers outside a sweep retain their existing live proof. N/2N service budgets must count one native snapshot regardless of candidates, alongside zero session-clock effects. Add stateful preview/apply snapshots, missing/incomplete/failed observation controls and parser conformance. Cancel the now-obsolete queued branch suite before pushing the code revision.

### 2026-10-18 — native acquisition after exact-command measurement

Hub approval6cb78cc3-b658-4c66-80b8-052634118f70 follows exact-command evidence:379filename arguments took13.409s to report no holders, while one machine-wide listing took1.910s/2,769,742bytes with exit0 and no warnings. Use the unfiltered machine listing once, bounded16MiB stdout/64KiB stderr, and filter in memory to exact current and validated segment paths; resolve process births only for matches. Keep the existing2s command bound and fail closed on any warning/partial/overflow. A loaded-host preview may be repeated in a quieter moment per the hub, never by raising the deadline. Added current/closed-holder parser RED before acquisition change, native conformance and constant-cost guards.

### 2026-10-18 — scope transient clients by their existing owner

Hub receipts7bdae765-38b7-41cd-acc4-66dd39e1bd16 and a473cb31-2893-4985-9112-3552cf411a35 identify PID51802 as a live brain Couch client, not a stale process to stop. The existing proof equated missing PAIR_RETENTION_PROTOCOL with unknown ownership and blocked all logs when PAIR_DATA_DIR was absent. Use already-published COUCH_PAIR_DATA_DIR/COUCH_THREAD_SCOPE/COUCH_THREAD_TAG to derive the validated composite owner; PAIR_DATA_DIR/PAIR_TAG cover selected ordinary wrappers. Conflicts/missing owner evidence remain ambiguous. Known clients block their exact diagnostic families and any current/segment payloads they hold, leaving unrelated owners to normal policy. Missing protocol means relaunch-to-register, not old creation time; name the slot/tag/PID in that blocker. The exact live-client/no-PAIR_DATA_DIR regression failed against the previous fallback before implementing this refinement.

### 2026-10-18 — immutable readers separate from plausible current writers

Hub direction33f17a20-36e5-4cdc-8369-014af0decbc4 replaces the all-log fallback for ownerless transient runtimes. Writers append only the current path; closed generation payloads are not reopened for append. A complete native open-file observation therefore protects each immutable payload independently. Add a reader-only proof for closed generation preview/deletion/replay, retaining full writer ownership proof for current paths and pending rotation. Keep per-file identity/mtime and durable intent checks unchanged; actual closed readers still block. The per-sweep listing records current and segment holders separately (not one aggregate that would block old segments because the current file has a reader). The exact ownerless-current/unheld-expired-segment integration failed before the change, then passed0.510s; it also tests a late segment holder blocking apply, and release allowing only the expired segment's7bytes while the current file survives.

### Final consolidated proof model and verification note

The third proof refinement is recorded as a consolidated design rather than a new timeout/probe workaround. One stable per-log lock protects certificate validation and durable deletion; it cannot retrospectively cover readers or pre-protocol writers that never acquired it, so bounded native observation remains necessary at that boundary. One sweep observes open files/processes; each later sweep/apply recaptures. Current generations use writer ownership plus exact holders. Closed immutable payloads use exact holders only. Unknown native output blocks; known Couch owner fields scope transient current writers; genuinely ambiguous current ownership stays blocked. No lifetime absence proof is inferred from an idle timestamp.

Task3 owned conformance exposed intentional current-file blocking behind the live supervisor after a legacy FD release. Portable stateful controls cover eligible-current release; native current/closed FD controls cover real reader appearance/removal; the real approved18-generation sweep supplies positive native closed collection. No current-file proof was bypassed to manufacture a positive result. The first sweep's exact12,734,567,104bytes, before/after inodes and volume df deltas are in the issue Log. #687 owns supervisor recognition separately. Task4 gates remain outstanding.

### 2026-10-18 — close-review concept and source-ownership corrections

BR-3 corrects the concept map: diagnostic-only CLI scope belongs to the existing public preview/apply integration row because it parses arguments and invokes filesystem-facing services; it is not a pure entity. The measured proxy-turn fixture lives in the new diagnostic_turn_budget_test.go, rather than changes to wrap_events_test.go or diagnostic_buffer_test.go. The tables and Task3 file list now identify those actual owners without changing the approved behavior.

BR-5 closes production source ownership for the native-proof refinement. diagnosticlog/proof.go is an artifactpath resolved consumer for adapt, wrap-events and fullscreen-diagnostics, using existing scoped resolver bindings. diagnosticlog/sweep.go joins the exhaustive non-artifact source inventory because it observes native processes and open-file evidence without constructing Pair artifact names. The known-violations baseline is unchanged; the source guard must reject missing classification and then pass with these declarations.

### 2026-10-18 — close-review complete apply reports

BR-4 preserves the collector's removed-only return contract while exposing retained/blocked managed decisions through the report-only Options.ObserveCollection callback. The explicit service records those decisions and appends completed removals before propagating a later page error; eligible candidates that did not finish deletion are never reported as removals or counted as collected bytes. A mixed managed fixture checks removed, recent and held rows together. A second-delete-intent fault checks that the first completed deletion and its exact bytes survive the error report while the interrupted deletion is not claimed.

### 2026-10-18 — close review replay and traversal correction

BR-1 moves exact-target reader authorization into shared recoverDeletion, so Open, Write, maintain and explicit collectors cannot replay an old intent using proof of a different current file. Legacy Proof-only callers inspect the actual rotated payload. Four entry-point regressions cover held and released targets. Counted fallback proof checks now include one per deleted payload plus bounded page overhead (90/180 at80/160); the shared native snapshot remains constant at8/16. BR-2 preserves a valid missing-parent declaration as an observable preview blocker and advances only that path through the real scheduler; malformed inventory and global uncertainty continue to fail closed. No policy, data scope or new persistent state is introduced.

### 2026-10-18 — complete scheduler blocker rule after BR-6

The second review cleared BR-1 through BR-5 but reproduced the same traversal family with a changed append certificate. A missing-parent-only exception did not implement the full approved blocker rule. The collection boundary now distinguishes complete per-path decisions from failure to acquire global evidence: native Snapshot/OpenFiles/Runtimes acquisition errors carry ErrInspectionUnavailable while retaining the original error cause; a complete observation of a live holder remains a local blocker. Scheduler collection advances past all local certificate, recovery, malformed per-log or filesystem blockers without deleting or repairing them. The existing declaration/path and preview preserve their visible blocker. Cancellation/deadlines, busy coordination and incomplete native inspection propagate; malformed registry/global inventory and coordinator errors occur outside that local collection boundary and still stop/yield. Real scheduler tests enumerate changed reservation, malformed log state, held pending deletion, local intent fault and stale missing parent before later eligible data, plus global inspection controls. No new persistent state or permissive deletion authority is introduced.
