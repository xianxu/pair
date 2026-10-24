# Fresh explicit-hub startup implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the session-warm startup integration. Steps use checkbox syntax.

**Goal:** Start a genuinely new configured fleet normally, with hub automation off, without mistaking an absent startup child for a failed attachment.

**Architecture:** Preserve the existing read-only startup observation before any new startup launch. Its pure classification decides whether configured hubs may start; existing lifecycle launch/adoption paths own the effects. Known shutdown and unknown evidence remain conservative. There is no speculative broker-registration change.

**Tech stack:** Go, existing Couch core/console, shared testcouch owner and native Claude/Codex fixtures.

## Evidence and alternatives

Main99846a5015 single-hub production form gesture admits the primary within2s. Exact two-hub PRIMARY WORKER1 run also admits the primary (observation0), while reproducing the extra hub's empty-result attachment error. Source traces start-hub -> StartInteractive returning zero StartResult,nil -> Started() unconditionally true -> adoption identity refusal. Pair:6's persistent refusal remains unclassified; its repeated-Enter helper is under correction, not evidence for a registry fix.

Selected: classify fresh once before effects, then use existing launch machinery. Rejected: keep fresh fleets parked and teach every harness to compensate (contradicts operator intent); globally make StartInteractive launch whenever absent (would auto-resume/start after a known shutdown); add registration retries (no reproduced registration defect).

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| StartupKind / ClassifyStartup | cmd/internal/couchcore/startup_policy.go | modified: fresh is proven zero history, not absence after history |
| StartResult.Started | cmd/internal/couchcore/ops.go | modified: empty result has no child; real handle preserves normal adoption |
| start-form admission | cmd/internal/couchtty/menu.go and menu_operations.go | existing owner; qualify repeated Enter before selecting any fix |
| Options.Hubs | cmd/internal/testcouch/fleet.go | modified: ordered explicit worker directories, same #830 contract |

One StartupObservation belongs to one console startup and is never recomputed after its own launches. Fresh requires zero retained records and no unreadable/unresolved evidence; known records absent mean full-shutdown, positive survivors mean normal, uncertainty means unknown. Existing parked/scope rules remain. Hubs use the same observation, not independent clocks or per-hub scans (ARCH-PURE, ARCH-DRY).

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| startup orchestration | cmd/internal/couchcmd/run.go, hubs.go | modified | initial primary attach and serial extra-hub lifecycle queue |
| StartFreshHub / start-hub dispatch | cmd/internal/couchcore/startup.go and operationdispatch.go | modified | one exact target resolution, existing ordinary Open / numbered Fresh path |
| shared fleet acceptance | cmd/couch/fresh_hub_registration_live_test.go | new | real private supervisor, native fixtures, inherited caller CLI |

Fresh launch permission is internal startup policy from the recorded observation; no automatic agent message is authorized by it. Launch the primary through the existing explicit open path after observation, before initial adoption; queue other configured hubs serially through the existing lifecycle workers with the same fresh permission. For normal/full-shutdown/unknown, retain surviving-only StartInteractive. An empty surviving-only result is a successful no-child result, never adopted or aborted. Automation initialization consumes the original observation, so newly launched hubs cannot turn it on.

## Transition and operator projection

| Observation/event | Permitted effect | Operator sees / recovery |
|---|---|---|
| Proven fresh | Start configured hubs once, primary selected; automation off | startup: fresh; hub automation off; say turn it on |
| Normal, selected hub survived | Reattach exact surviving sessions; per-hub automation follows existing survival rule | normal restart, missing hubs retain explicit recovery advice |
| Prior records absent | No cold launch or resume | full-shutdown; Resume from disk / Ctrl+Space Start |
| Unknown evidence | No speculative launch | unknown plus diagnostic and explicit recovery action |
| Surviving-only start returns empty | No adoption effect | empty hub stays available in switcher, no false identity-failure notice |
| Start form admitted, Enter arrives before completion | At most one launch for that form | existing operation progress; later Enter cannot create another start |
| Launch/attach failure or cancellation | Existing owned cleanup and row result; no retry | one result with existing recovery action |

No new executor, queue, retry, store, protocol or agent-message producer. Existing lifecycle reservations continue through adoption; late completion keeps existing focus rules (ARCH-ORDER).

## Operating envelope and trust

Startup only: reuse one inventory/session observation and existing worker cap/serial hub queue. Startup latency uses #773's caller-owned budget; no timeout increases. Native qualification is bounded at8 minutes per command with private roots, one run per personality after deterministic tests. Real CLI caller lookup is checked separately from fixture readiness. Existing exact-generation cleanup owns descendants; evidence is retained outside removed private roots. No live fleet/pair:0 involvement and no real agent credentials. No persisted format changes (ARCH-EVOLVE); StartupKind is ephemeral output. Existing thread and diagnostic retention owns production residue; no new durable runtime artifact.

## Implementation

### Task1: pure startup and child-result boundaries

Files: startup_policy.go/startup_policy_test.go; ops.go and colocated adoption tests.

- [x] ClassifyStartup: generate incomplete and contradictory snapshot/presence combinations; an independent predicate forbids fresh whenever history or uncertainty exists, and permutation checks forbid input-order dependence. Confirm red on readable zero-history input before changing classification.
- [x] StartResult.Started: vary the record and handle independently, with handle possession as the independent adoption-authority oracle; confirm the empty result fails before the fix and inspect sibling Started implementations for the same contract.
- [x] Run targeted tests, confirm intended failures, implement minimal classifier and no-child correction; rerun startup/adoption tests.
- [x] Commit with #863 and model trailers.

### Task2: orchestrate only proven fresh launches

Files: couchcmd/run.go, hubs.go, automation.go consumers and hubs_startup_test.go; couchcore/operationdispatch.go.

- [x] runConsole/startExplicitHubs: use the existing stateful fake process/store boundary with held launch/adoption and cancellation; count effects against the immutable initial observation, independently forbidding cold effects without fresh evidence. startupAutomationOn: mutate subsequent live inventory while retaining the initial observation and require automation to stay off for fresh startup.
- [x] Observe before new launches; route permitted fresh primary/extra effects through existing explicit-open/lifecycle owners. Keep surviving-only paths conservative.
- [x] Update startup summary for fresh (not missing/resume-from-disk language for never-started hubs); keep real launch failure recovery advice.
- [x] Run core/cmd/tty startup and automation targeted tests; commit.

### Task3: qualify form duplicate admission

Files: couchtty/menu start tests / console lifecycle tests; production reducer only if a failing test proves a gap.

- [x] reduceStartKey/dispatchMenuOperation: hold effect completion and generate repeated Enter plus duplicate/late result events; independently count start effects per form identity and require at most one. Current source synchronously closes admitted form; do not add redundant state if the invariant already holds.
- [x] Ask pair:6's corrected native run to retain whether a second actual launch occurred. If reproduced, fix in this issue with one consumed-form identity until outcome; no new retry. If it needs another mechanism, stop and report to hub before expanding.
- [x] Record production versus harness classification with evidence.

### Task4: native acceptance and close

Files: shared testcouch Hubs option/tests; fresh_hub_registration_live_test.go; relevant atlas startup/hub docs and issue Log.

- [x] Diagnostic Hubs argv input red then green (same #830 contract), exact two-hub main caller admission measured.
- [x] Convert diagnostic into durable Claude/Codex fresh two-hub acceptance: Fleet.Start returns with actual live callers, both hubs registered, no false hub-failed notice, automation off, no automatic hub input. Keep explicit known-shutdown gesture test where appropriate; no duplicate fleet owner.
- [x] Run one bounded native qualification per personality. On failure retain full wrapper and console facts, classify before another run.
- [x] Coordinate #830 integration; whoever lands second takes identical Hubs contract. Preserve #830 T6 skip until its owner verifies it.
- [x] Document fresh vs known shutdown, update atlas and issue Log, audit reader inventory, run make test-affected-inventory.
- [ ] Push; run sdlc local-check run --pre-close, close with actual evidence and --no-project; on SHIP merge main then sdlc merge --yes. Report real merge/archive state to hub. No live stage requested.

## Revisions

2026-10-24 — Hub563a3e8a: zero-history is fresh and hubs start, automation off; no blanket relaxation after known shutdown. Hub c1133e63 authorizes only the small shared Hubs harness option before #830 lands. Hub bc2786cd requires duplicate Enter production qualification if the peer run proves two launches. Measurement currently reproduces empty adoption, not persistent registration failure; no registry fix planned.

2026-10-24 — PQ-1: compress test inventories into named risky-function strategies, with generated contradictory evidence, independent adoption authority, held effects and at-most-one launch invariants. Architecture and selected behavior unchanged.

2026-10-24 — Native qualification confirmed Open intentionally cannot create a never-used managed slot (resume-no-survivor). Hub7cf1b120 approves the existing Fresh path for this zero-history-only permission. Use StartFresh for initial configured hubs; known-shutdown/unknown remain untouched. Original evidence still owns automation. Retained Zellij v0.45.1 RemoveClient unwrap panic is an external lifecycle fault, separately recorded, not a registry patch or timeout change.

2026-10-24 — Hub9a53d319 approves consolidation after the second path correction: core StartFreshHub resolves the exact target once with Open intent, uses StartFreshSlot only for a numbered target, otherwise shares the existing resolution-based ordinary open implementation. Both initial primary and extra-hub startup consume this helper. Real OS SlotCatalog-backed tests for both target kinds red on missing helper, then green; no relaxed generic StartInteractive semantics. Stop and re-plan if a third path correction is needed.

2026-10-24 — Implementation and native/deterministic qualification complete. The final checkbox is post-implementation gate procedure: final recorded check is queued, close review overlaps under tl-worker PERMISSIONS, and no landing precedes final-head recorded pass. No unimplemented product behavior is waived.

### 2026-10-24 — consolidated re-plan after close REWORK; awaiting approval

This Revision supersedes the completion assertion above. Task2's claimed held fresh launch/adoption coverage is withdrawn: its tests inspect prepared calls and invoke callbacks; the held real-console tests cover surviving sessions. BR-1 is accepted as Critical. BR-2 remains an Important acceptance gap, not a classified regression or flake. No production code changes until the hub approves this re-plan, followed by change-code's plan-quality gate.

**Contract:** Fresh requires no history in the active namespace's global/legacy storage or any configured hub's local store, independently of enrollment. Observation grants permission to attempt empty creation only. Existing store reservations and locks own mutation authority. Automatic startup cannot replace, archive or cold-resume a conversation. History or uncertain evidence keeps the affected hub parked.

**Alternatives:** Parking every fresh hub is simplest but contradicts the operator's requirement. A broader scan alone leaves the appearance race. General Fresh authorizes replacement; general Open can cold-resume newly appeared history. Instead, add empty-only checks to the existing creation commits, with no new executor, lock family or durable token (ARCH-DRY, ARCH-SECURE, ARCH-ORDER).

#### Revised concepts and boundaries

| Entity/seam | Files under cmd/internal/ | Change |
|---|---|---|
| History evidence: empty, retained, unknown | couchcore/startup_policy.go | Pure classification input; incomplete/default evidence cannot mean empty |
| Configured-target history observation | couchcore/startup_history.go (new), threadstore_preview.go, threadstore_layout.go | Read-only direct observation, independent of enrollment |
| Empty-only startup creation | couchcore/startup.go, slotrecovery.go, threadtag.go, threadstore.go | Narrow existing reservation/commit paths; no replacement or resume |

Observe each distinct configured hub using the existing read-only workspace/catalog identity boundary. Never call selectedSlot, reconciliation or enrollment merely to observe. Construct the verified numbered target's local ThreadStore directly. Reuse preview path validation and read locks; missing storage is empty only when the exact target identity is known. Inspect namespace current/manifest and global retained archives, including legacy records that ordinary routed lists filter out; inspect configured local thread.json and archives independently. Any retained entry denies fresh without decoding its full payload. Malformed/unsupported data, pending journal, unsafe path, contention, changed identity or read failure means unknown. Observation creates, migrates and repairs nothing.

Resolve each mutation target once. Numbered startup uses requireEmpty=true, never public StartFreshSlot's replacement mode. Carry that intent to the existing replaceSlotCurrent locked commit, where both current and archive absence are checked alongside exact physical identity before publication. Ordinary startup bypasses startInteractiveResolution's warm/cold selectors and uses create-only spawn. There is currently NO atomic empty-primary-scope API: AllocateThreadTag claims a unique address and CreateThread checks only that tag. Add a narrow guarded variant of that existing locked reservation publication that checks target-scope current and archived history. Reuse artifact claim/release, journal publication and CommitStartClaim; explicit operator Open/Fresh semantics stay unchanged.

Use one lock-assuming history reader shared by preview and commit, not recursive public ArchivedThreads calls. Global/legacy and local history use different locks: hold the existing namespace coordinator, acquire the exact local store using the existing nonblocking nested-lock protocol, and recheck relevant legacy plus local history before the empty-only commit. A busy nested store releases the coordinator and refuses this startup attempt; no retry loop. Do not infer atomic absence from two separately released reads. Validate this ordering against existing migration/enrollment writers before implementation; if it cannot fit the existing coordinator protocol, stop and report rather than add another lock mechanism.

Initial fleet evidence remains immutable, keeping automation off after our own launches. Mutation checks are target-scoped, so our new primary does not prohibit another originally empty hub. This is not an all-or-nothing multi-hub transaction: each target independently refuses newly discovered history, preserving its bytes. The safety resource is its existing store/coordinator lock held from the final empty check through journal publication, rather than a stale observation.

**Cost:** Once per startup, O(H+E) for H distinct stores and E inspected directory entries, never per tick/message. Observe each store once, stop when retained history is established, use bounded directory reads with existing context/lock deadlines. Archive presence requires zero archive payload reads. Count visits at 8 and 16 stores to reject worse-than-linear traversal. No new cache or watcher. Existing launch budgets/reservations/adoption and cleanup remain owners; no persistent format changes (ARCH-CONSTRAINTS, ARCH-FUNERAL, ARCH-EVOLVE).

#### Actions and projection

Actions: turn automation on; select Resume; inspect the named diagnostic and retry explicitly; select explicit Fresh.

| Internal state/outcome | Visible state | One recovery action | Possible loss |
|---|---|---|---|
| Complete empty proof and successful empty commit | fresh, automation off | turn automation on | none |
| Retained history, including unenrolled/archive-only | parked with history advice | select Resume | none; no automatic archive/replacement |
| Unsafe, unreadable, unsupported or busy observation | parked with named uncertainty | inspect diagnostic and retry explicitly | none |
| History appears before commit | startup refused: target history changed | inspect diagnostic and retry explicitly | no old bytes lost; no new launch |
| Owned launch/adoption fails or is cancelled | existing failed operation result | existing explicit retry | only failed new launch, through existing cleanup |

Fresh remains an explicit operator action when history exists. Positive normal-survivor attachment is unchanged. Foreign namespaces, unsupported-data recovery and the separate Zellij probe bug are not expanded here.

#### Remaining steps — one close boundary

- [x] **Red observation proof:** Real SlotCatalog-backed startup_history/startup_observation tests supply unenrolled current, archive-only, legacy global, malformed and journal history. Assert non-fresh and independent before/after filesystem equality, no enrollment or initialization. Generated pure evidence combinations cover unknown/retained dominance; count 8/16-store visits and zero archive payload reads.
- [x] **Red mutation proof:** Hold the existing commit seam, introduce current or archive history after observation, release, and assert refusal, identical old bytes, zero retirement/archive effects and zero launches for both ordinary and numbered targets. Include cross-store legacy/local appearance and lock-contention controls. Keep an explicit operator Fresh replacement control. A negative control using general replacement/Open must fail the oracle.
- [x] Implement only the shared evidence reader and empty-only commit boundaries above. Run targeted core/store startup tests and commit the minimal correction.
- [x] In couchcmd/fresh_startup_test.go drive actual Console.Run with existing stateful lifecycle/store seams, holding launch and adoption separately. Prove serial extra launches, immutable automation off, cancellation joins, no empty-result adoption and history refusal. Count real effects; prepared argv or direct completion callbacks do not satisfy this step.
- [x] **Classify BR-2:** Retain the review report's command and failure: StartupArguments/claude failed after initial startup during switch-agent adoption, 'attach terminal input has ended', with 'terminal listing omitted a live server'; Codex skipped. FreshHubCaller, EditedRestart and dirty/force passed. Run unchanged StartupArguments on pinned main and corrected branch sequentially, same native tools/substrate and shared owner, using existing selected-candidate/disposable review infrastructure rather than a second slot checkout convention. Capture console, wrapper, listing and operation receipts on failure. No timeout increase. A matching main failure is independence evidence; branch-only failure or unresolved attribution blocks acceptance and goes to the hub before another run. A passing branch retry alone is not flake classification. Do not equate this with the earlier Zellij metadata panic.
- [x] Qualify corrected fresh two-hub Claude/Codex behavior plus the required EditedRestart, StartupArguments and dirty/force acceptance through the shared owner. Scenario watchdog includes cleanup. Retain/classify failures before another invocation. Prior FreshHubCaller PASS39.45s is partial old-source evidence, not proof of this correction or the whole matrix.
- [x] Update atlas/README/embedded skill; audit named readers; make test-affected-inventory; push final head and run recorded pre-close once. Prior #863 queued lease1020585962 was cancelled after REWORK and its parent joined without tests; no passing recorded #863 check exists.
- [ ] Narrow re-close with both findings' evidence, hub-approved --no-actual for the exact resting window, and --no-project. SHIP plus recorded final-head pass precedes sdlc merge --yes. After landing file the already-directed separate Zellij startup-probe readiness issue; report real merge/archive state.

**Approval requested:** This consolidated plan covers complete history evidence, empty-only mutation on both target kinds, genuine held fresh orchestration, and BR-2 classification. It adds no replacement authority, retry policy or registry repair. Production work remains stopped until hub approval.

### 2026-10-24 — implemented evidence and approved close disposition

The approved consolidated safety correction is implemented: direct configured-store history observation and empty-only locked publication on both target kinds, with real Console.Run held launch/adoption tests. The Run tests live in fresh_startup_run_test.go (rather than adding them to fresh_startup_test.go); the original prepared-call tests remain distinct. Native fresh two-hub callers and dirty/force acceptance passed for Claude and Codex. The required matrix completed with a Zellij panic (#866) and one Codex EditedRestart idle refusal; a pinned-main Codex comparison and one authorized instrumented branch Codex comparison passed, without establishing the original refusal's cause. TL24a27ad0 permits close with that refusal independently filed as unclaimed #867 and the native limits explicitly retained. The old broad acceptance assertion is superseded by this disposition, not silently marked all-green. No further native run, timing bump or admission relaxation. Remaining checkboxes are final audit/check and close/landing workflow, not undelivered product behavior.

2026-10-24 — Audit passed and final pre-close lease-1206458293 is queued on b16d2c1ba5b22ae7ae24893b57c72ab096522d3f; the checked verification step means invocation, not a claimed passing result. Close/review/landing remain pending. #866 was filed before landing at TL request, superseding the old timing in the final workflow checkbox.

### 2026-10-24 — BR-3 terminal-disposition re-plan (approval requested)

BR-1 is addressed by the reviewer's own negative controls. BR-3 is accepted: numbered startup publishes its exact local creating record and native address claim before discovery/enrollment, but those errors return before the shared launch tail owns rollback. The general namespace rollback cannot reliably reach an unenrolled local record. This is the post-publication ownership boundary, not another history/classification exception (ARCH-ORDER, ARCH-FUNERAL, ARCH-DRY).

Selected boundary: immediately after successful empty-claim publication, the existing call owns one exact no-helper claim until it explicitly hands ownership to launchClaimedThread. Route every discovery, identity, enrollment or cancellation exit in that interval through one cleanup. Reuse ThreadStore.deleteStartResult's exact address/revision/nonce mutation on the already-validated local store, independent of enrollment. Generalize the existing private rollback implementation only enough to select that known store; its existing namespace callers retain their default. Release the native address only after the authoritative selected store positively reports the pristine record absent. A changed revision/nonce, uncertain journal/physical identity, storage error or failed native release joins the original cause and retains conservative occupancy; never infer cleanup from the namespace not yet knowing the local record. Cleanup is independent of the cancelled caller context, as the existing rollback is. No session/process stop is needed before the helper handoff because no helper was launched. Once transferred, the existing launch owner handles preparation, launch and adoption failures; no double cleanup.

Rejected: enroll before the empty claim (moves the already-reviewed mutation boundary and may migrate retained history); call the namespace-only rollback after each return (cannot reach unenrolled state and duplicates error tails); remove a local file/claim directly (bypasses nonce/revision and journal authority). No new lock, retry, durable state or replacement permission. Existing store/coordinator lock ordering remains unchanged.

| Phase / failure | Visible result | One recovery action | Possible loss |
|---|---|---|---|
| Empty claim published, discovery/identity/enrollment/cancellation fails; exact rollback confirmed | Existing failed-start row with original reason | explicitly retry startup | failed new reservation only; no prior conversation |
| Exact rollback or claim release cannot be confirmed | Existing failed-start row includes cleanup uncertainty | inspect/recover the named attempt | no uncertain record or claim removed |
| Ownership handed to launchClaimedThread | Existing lifecycle progress/result | existing lifecycle recovery | unchanged existing launch semantics |

Red first: keep the reviewer's injected discovery failure immediately after artifact Claim and assert the local record/creating transaction and native marker, zero children and no archive. Add discovery identity mismatch, enrollment refusal, cancellation before enrollment, and publication/rollback failures. All confirmed no-helper failures must leave no creating transaction or stranded native claim; failed/changed cleanup must preserve exact successor/foreign history and the original error. Prove one rollback per failed owner and zero early rollback after successful handoff. Use real SlotCatalog/local store and existing artifact/store hooks, plus an explicit successful enrollment/launch control. Run targeted startup/store/rollback tests, then final reader inventory and a new pre-close record (the old queued #863 record was cancelled without running tests). Re-close through the binary.

BR-2 remains a separate evidence disposition: no full-matrix pass or independence of the original attach-input-ended signature is claimed. The pinned-main Zellij panic and passing Codex comparisons are counter-measurements for the later observed occurrences, with #866/#867 retaining their unknowns. Follow the TL's dispute direction using those exact limits; the next reviewer must answer it. No more native runs are authorized here.

2026-10-24 — BR-3 implemented under TL0d3e6cf1 approval and CLEAN plan gate. enrollFreshStartupClaim owns the pre-handoff interval with deferred exact-store rollback; existing rollback selects the authoritative store internally, and namespace callers retain their defaults. Discovery red, full14.790s failure/uncertainty/handoff cohort,47.239s surrounding regression cohort and rollback-removal overlay control are recorded in the issue Log. BR-2 is formally disputed with limits stated. Final new-head pre-close record and review remain pending.

### 2026-10-24 — recorded-suite fixture fidelity correction

The complete corrected-code suite exposed six introduced stale test assertions after the approved startup/Started contracts: distinguish genuine retained-history shutdown from fresh startup in the installed CLI fixture; preserve both joined unknown diagnostics; expect start-hub's action and path arguments; and supply a child handle only to intended-child StartResult positives, retaining explicit no-child non-adoption controls. Original assertions and deadlines remain; no production change. The seventh failure is independent PID-marker publication issue #869, filed unclaimed with the unchanged targeted repeat; no registration or suppressed failure. All scripts/packages finished before this disposition. New-head verification and narrow re-close are required for these test fixes.
