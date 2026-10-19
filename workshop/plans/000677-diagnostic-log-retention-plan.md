---
collaboration-mode: discussion
---

# Diagnostic log retention implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use subagent-driven development for bounded independent tasks, or executing-plans for session-warm work. This is one atomic close boundary; task numbers are not milestones.

**Goal:** Bound optional diagnostic residue with a configurable seven-day window and a 64 MiB generation cap, independently of backup and session retention, and report the first safe live reclamation.

**Architecture:** Reuse diagnosticlog's stable per-log lock, exact identity/last-write certificates, process inspection and durable deletion intent. One age/oversize decision replaces the backup fence and dormant count-based admission/eviction. Diagnostic-only collection must not acknowledge Couch migration, initialize session clocks, or delete session/history payloads.

**Tech stack:** Go, portable directory fixtures, injected clock and stateful process inspection; existing Pair GC CLI/scheduler.

**Status:** Proposed to the hub; no implementation until approval and change-code. Full flow is expected because the policy, CLI scope and collector integration exceed the quick shell's code limit.

## Evidence and approaches

The issue Log records the read-only inventory, oversized legacy provenance and passing current append-limit controls. The 4.93 GB example last changed before the October 15 hard-limit fix. Existing tests intentionally preserve sparse 4 GiB legacy generations. New append admission is already bounded; inherited oversized files are the uncovered cleanup policy.

1. **Recommended: age-based eviction plus eviction of proven oversized legacy generations.** Default seven days; no three-generation count override. Evict a whole oversized legacy generation at any age through the existing deletion intent. This can lose recent oversized debug evidence, but requires no bulk payload rewrite or new compaction transaction.
2. **Fence off, keep old count policy:** smaller patch, but three generations can erase most of a busy day's diagnostics. Rejected because it does not mean approximately seven days.
3. **Tail-compaction of oversized generations:** preserves their newest bytes, but copies payload and requires a new crash-recoverable replacement/metadata protocol. Avoid this machinery unless the hub requires preserving young oversized tails.

No automatic deletion of unknown or live legacy-writer files is authorized. The first sweep reports those blockers explicitly. Newly admitted data always remains capped; a blocked legacy file is not silently described as compliant.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Diagnostic age/oversize policy and retention-day parser | cmd/internal/diagnosticlog/policy.go, collect.go | modified |
| Effective diagnostic options and retained reason | cmd/internal/diagnosticlog/writer.go | modified |
| Dormant count/debt selection | cmd/internal/diagnosticlog/policy.go | deleted as active policy |
| Diagnostic-only CLI scope | cmd/internal/gccmd/run.go | modified |

The policy owns one-to-many generations per diagnostic log. It derives age from certified last write; current paths, immutable segments and legacy paths use the same decision. Persisted v2 MaxGenerations remains readable for compatibility but confers no new deletion or append-blocking authority. Keep the existing persisted byte authority and tighten-only behavior. No new diagnostic state machine or payload family is introduced.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Per-log admission, collection and deletion replay | cmd/internal/diagnosticlog/writer.go, pages.go, legacy.go, collect.go | modified | files, stable log lock, durable intent |
| Explicit and scheduled diagnostic traversal | cmd/internal/gcruntime/runtime.go, schedule.go | modified | root coordination, registry and exact-family inventory |
| Public diagnostic-only preview/apply | cmd/internal/gccmd/run.go | modified | CLI, metadata report |
| Wrapper trace measurement | cmd/internal/wrapcmd/wrap_events_test.go, diagnostic_buffer_test.go | modified tests | real proxy and owned buffered log |

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

- [ ] Add failing pure parser/selection tests for defaults/ranges/overflow, exact seven-day boundary, custom window, young oversized data and invalid/future evidence. Run targeted tests and retain the expected red against today's backup fence.
- [ ] Replace the sparse 4 GiB preservation expectation with explicit oversized eviction through the real journal; no GiB payload allocation. Keep a fresh bounded generation and unrelated file as survival controls.
- [ ] Update policy to the rule above and remove active count policy. Keep legacy persisted limits readable and keep new-write whole-record cap tests for synchronous/asynchronous maintenance, multiple handles and absent proof.
- [ ] Exercise current, closed, quarantined and legacy paths through one policy; preserve uncertainty rather than granting deletion authority to incomplete metadata.
- [ ] Replay deletion at intent/payload/metadata/parent/state boundaries, including cancellation and identity replacement. Resume twice; ensure exact-once removal/reporting and no neighbor deletion. Run diagnosticlog tests, then commit.

**Function-level adversarial strategy:** ParseRetentionDays fuzzes hostile scalar input into typed duration/error. The age/oversize decision is tested from independently stated predicates across boundary clocks and sizes. previewPageLocked/previewLegacyCurrent consume replaced identities and ambiguous proof at their real file seam. recoverDeletion uses existing fault checkpoints and changed metadata/inodes, not mocked unlink calls.

## Task 2 — independent diagnostic traversal and CLI scope

**Files:** gcruntime/runtime.go, schedule.go, runtime_test.go, schedule_test.go, maintenance_budget_test.go; gccmd/run.go, run_test.go/apply_acceptance_test.go; artifactpath manifest if a new source is warranted.

- [ ] Add a failing portable-root integration with migration_complete=false, expired/oversized diagnostics, old authored session/history files, unknown neighbors and a stateful inspection inventory. Diagnostic-only preview creates nothing; apply removes only eligible diagnostics and does not initialize or update session clocks/store acknowledgments.
- [ ] Factor diagnostic preview/apply ownership from the existing Service paths. Reuse DiagnosticInventory and registry discovery; remove only diagnostics' migration gates in explicit pages and scheduled batches. Preserve session collection/migration guards and root-before-log locking.
- [ ] Add --diagnostics-only for preview/apply; refuse combinations with migration/register/forget-store operations. No new operator command is required for history/session retention. Include effective diagnostic days and separate collected/blocked bytes in JSON/text metadata reports.
- [ ] A pre-migration scheduled session phase must yield to diagnostics rather than terminate the maintenance cycle. Preserve cancellation, nonblocking busy handling and cursor progress.
- [ ] Run gcruntime/gccmd targeted tests and retention integration, then commit.

**Mechanical bounds:** At N=80 and 2N=160 real small generations, sweep bounded pages and assert exact removed counts/logical bytes, zero payload-copy writes, metadata write bytes/operations proportional to removed generations through ObserveIO, and writer proof calls proportional to pages (not records). Seed more than64 young generations to prove the retired count inventory no longer blocks rotation or prematurely evicts recent data. Existing directory traversal remains capped at128*(limit+1) names per page. At8/16 unrelated session owners, diagnostic-only pages must evaluate/persist zero session clocks, using existing owner probe/persist counters. No full host census is added.

## Task 3 — detail-level measurement, docs and live verification

**Files:** wrapcmd/wrap_events_test.go, diagnostic_buffer_test.go; README.md; atlas/storage-retention.md, storage-retention-io.md; issue Log. Update help from the same effective policy.

- [ ] Replay explicitly delimited owned turns through the production proxy with fixed output chunks, controlled flush barriers and a real diagnostic sink. Measure baseline bytes per turn at N/2N turns, per-chunk event counts, payload/metadata bytes and drop counters. Do not infer turns from stdin reads or idle alerts in the real sample.
- [ ] Proposed detail decision: retain current hashed per-chunk evidence within bounded lifetime unless the owned-turn measurement demonstrates it overwhelms the envelope. Record the measured decision and PAIR_WRAP_EVENTS=0 opt-out. Any new default summary/drop policy is a scope revision returned to the hub before code, not silently folded in.
- [ ] Document default/range/invalid-value behavior, diagnostic-only preview/apply, reasons, deliberate loss of oversized legacy generations and retained unknown/live-writer blockers in root README and atlas.
- [ ] Owned conformance: create aged and young logs with a running modern writer and a held legacy FD. Candidate CLI must collect only eligible/proven files, leave legacy-held files blocked, then collect after explicit release; no operator UI or native lane.
- [ ] After hub approval of this live arrangement, build this slot's candidate and run diagnostic-only metadata preview against the real Pair root, then diagnostic-only apply bounded by an owned process deadline. No pair:0 staging or migration acknowledgment. The operator has authorized debug trimming without backup; no authored/history payloads are selected.
- [ ] Record before/after diagnostic device+inode/size inventory and exact collected-row bytes, plus partial-progress/blocked reasons if the bound expires. Do not equate mtime estimates with certified eligibility or net concurrent growth with reclaimed bytes. No backup is created and no multi-GiB payload is copied.

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
