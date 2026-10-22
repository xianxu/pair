---
collaboration-mode: discussion
---

# Live reboot pane ownership diagnosis and correction plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Keep the native evidence capture and correction in this issue; the first capture is an investigation checkpoint, not permission to weaken an ownership validator.

**Goal:** Make public confirmed Fresh recovery complete for positively owned disposable Claude and Codex sessions, preserving exact stop authorization and all unknown/foreign/reused refusals.

**Architecture:** First retain the actual native observation at the retirement boundary and reproduce its causal sequence through the existing ownership probe. Then correct the ordering or proof handoff at the source of that failure. Reuse existing recovery authorization and lease machinery; a cached session name, a launch template, or a missing command never grants authority.

**Tech stack:** Go, existing Couch recovery state machine, procutil native identity/argv, Zellij, shared testcouch disposable fleet.

## Current evidence and uncertainty

The incident is recorded in issue #797's Log with its retained native log and SHA256. The stop offer names a registered wrapper, its Claude child and a Zellij session; confirmation fails inside the slot-tree fallback's final QuiesceNamed call.

Code proves this sequence:

1. Prepare and confirm recovery; preserve the prior screen/intent.
2. Optionally park the normal client helper.
3. Call any original terminal retirement.
4. TERM the additional registered wrapper tree and await its exits.
5. Re-prove session ownership from current live pane commands and quiesce it.

Steps2–4 can erase the evidence step5 requires. Zellij can leave exited panes held; command discovery then has no live argv. The existing fake separates process death from pane metadata, concealing this sequence.

The retained incident lacks pane JSON. Zellij also joins argv without quoting and can omit command enrichment after a100ms internal reply failure. These can yield the same generic refusal before any stop. The plan must distinguish the incident's missing/empty/ambiguous data, not infer it from its message.

## Core concepts

These are existing entities to inspect and reuse. No new production authority or state is approved by this diagnosis plan.

### Pure decisions

| Name | Lives in | Planned role |
| --- | --- | --- |
| ClassifySessionOwner / splitObservedCommand | cmd/internal/launcher/session_owner.go | Preserve fail-closed classification; distinguish missing evidence from ambiguous text in diagnostics if needed |
| RecoveryTerminalPlan | cmd/internal/launcher/recovery_terminal.go | Immutable exact server/descendant authorization data; candidate reuse, not a second cache |
| AdvanceRecoveryExecution | cmd/internal/couchcore/recovery_plan.go | Preserve screen/intent before retirement and launch only after confirmed retirement |

### Integration points

| Name | Lives in | Wraps / test seam |
| --- | --- | --- |
| withSlotTree | cmd/internal/couchcore/recovery_slot_tree.go | Registered wrapper, exact process signals and session quiescence |
| recoveryTerminalDriver.checkConfirmed | cmd/internal/launcher/recovery_terminal_confirmed.go | INTEGRATION validator; injected readiness reader defaults to bounded native metadata reads |
| executePreparedRecovery / parkPreparedRecovery | recovery_prepared.go / recovery_retirement_os.go | Confirmation, screen preservation, normal park and terminal retirement ordering |
| SessionOwnerProbe / ScopedThreadArtifactCollisionChecker | launcher/session_owner_os.go / couchcore/artifactcollision.go | Strict native pane replies, exact server generation and current binding |
| testcouch.Fleet | cmd/internal/testcouch | Isolated genuine agents, public CLI, identity-fenced owned cleanup |

The causal fake must couple process retirement to pane command disappearance. Reuse the real SessionOwnerProbe and strict reply decoder behind the existing fake native IO boundary; do not replace the final ownership decision with a scripted success.

## Investigation checkpoint before production changes

- [x] Retain a bounded disposable native capture using the shared harness. Record exact tool/version, public reboot offer, confirmation request/outcome, native pane JSON and server/wrapper generations before and at the failing retirement boundary. Retain the original failure's bounded NativeQueryEvidence before private-root cleanup. Credentials and unrelated process argv do not enter committed evidence.
- [x] Run one personality per Go invocation. Exercise Claude and Codex; retain setup, operation, output, write and cleanup bounds. All direct children and readers join; unknown cleanup prevents acceptance. Do not stage pair:0 or exercise the live fleet.
- [x] Build a deterministic replay through actual SessionOwnerProbe and recovery composition: valid owned wrapper evidence -> accepted confirmation -> park/retire -> held/exited pane missing command -> current refusal. Also distinguish pre-existing omission and ambiguous joined text.
- [x] Record which event first loses the witness and why confirmed native terminal attribution selected the slot-tree fallback. If the capture contradicts post-stop evidence loss, stop and revise the diagnosis rather than applying an ordering patch.

## Correction decision and approval boundary

Two viable shapes must be assessed against that capture:

1. **Quiesce while the positive witness still exists.** This is the smallest correction if the only destructive predecessor is the added slot-tree TERM loop. It must also cover the outer normal-park ordering. Moving a single inner call is insufficient if park already erased the witness. Keep normal park's preservation/continuation semantics.
2. **Reuse prepared exact terminal retirement across witness loss.** If normal park must precede terminal retirement, reuse the existing immutable server/descendant plan, lease, current-generation/survivor checks and allow-gone semantics. The initial proof must be positive; missing initial data still refuses. No new persisted permit, no name-only authorization and no expansion after confirmation.

Preferred rule: obtain the positive proof before destroying its witness, and recheck the exact accepted generation at each effect. Choose the smallest existing mechanism that enforces that rule for both normal-park and legacy fallback. Do not broaden ClassifySessionOwner to accept unknown panes. Do not treat Zellij's joined string as structured argv where boundaries matter; do not implement a cross-repository producer change without a separate approved scope.

- [x] Present the captured cause and the exact chosen correction to the hub; append the approved design as a revision. Run change-code and its plan-quality gate only after that design is concrete and approved. No production edit precedes this boundary.

## Implementation and verification after that approval

- [x] Add the failing causal regression first, including both applicable park/retirement orderings. Confirm it fails on current main, then implement the approved correction.
- [x] Cover same-session foreign owner, unknown/missing/ambiguous command, server replacement and PID reuse, newly arrived descendant, cancellation, expired/changed confirmation, stop applied but acknowledgment lost, independent exits and partial retirement. Assert no unauthorized signal and no launch before confirmed absence; never retry uncertain destructive input automatically.
- [x] Give missing actual-command evidence and parser ambiguity distinct pane-specific diagnostics without leaking raw arbitrary command text into the concise CLI error. Preserve native diagnostic evidence under the existing bounds.
- [x] Replay the captured native shape in the normal suite. Run public reboot plus its returned confirmation token against genuine disposable Claude and Codex workers; prove the new generation accepts public delivery, the old exact set is absent and cleanup is complete. No fixture-only crash substitutes for live reboot on this issue.
- [ ] Update the recovery atlas and enumerable consumer tests. Run relevant race/ownership tests plus catalog, Godoc, artifact inventory and reader audits. Publish the final head, run the declared affected check/full fallback, close through its mandatory review and merge on SHIP. Report the PR to the hub.

## Constraints, state and trust

This is an explicit recovery operation, not a sampling or keystroke hot path. Preserve existing RPC/query/retirement deadlines, confirmation lifetime, exact-set bounds and serial native test structure. Count the new observations/effect gates at N and2N if the selected correction adds traversal; require linear table/set work and no repeated host census per target.

Actions: inspect, request a new confirmation, acknowledge the current confirmation, resume preserved recovery after inspection.

| Observation / outcome | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Complete positive initial proof, same accepted set | confirmation required | acknowledge current offer | source may be retired after acknowledgment |
| Missing/ambiguous/foreign proof before effects | refused with specific diagnostic | inspect | none from this operation |
| Identity or ownership changes | refused; refresh confirmation | request new confirmation | earlier individually confirmed stops may remain applied |
| Unconfirmed stop or query | partial/unconfirmed, no launch | inspect | stopped source work; preserved evidence remains |
| Exact retirement confirmed | new generation started | inspect/deliver | old working turn intentionally ends |

ARCH-DRY: reuse the existing terminal plan, leases, strict parser and fleet harness. ARCH-PURE: classify evidence independently of IO; the shell executes declared recovery order. ARCH-PURPOSE: both public live reboot personalities and negative ownership gates are required; a diagnostic-only change does not finish the issue. ARCH-MOCK: couple fake native metadata to process state and compare it with native capture. ARCH-CONSTRAINTS: no deadline tuning or extra unbounded scans. ARCH-SECURE: argv, pane JSON and persisted bindings are observations, never trusted because they are local. ARCH-ORDER: distinguish initial proof, own applied stops and uncertain completion; preserve the original exact set and current effect vetoes. ARCH-FUNERAL: no new production durable family; bounded diagnostic captures stay in the test log/evidence record, fixture roots and processes have existing cleanup. ARCH-EVOLVE: no persisted format or field-meaning change is proposed; if the concrete correction needs one, return for approval.

## Revisions

### 2026-10-22 — Diagnostic stage approved

Authenticated ops:0 approval9d722e7b-5b6a-4b7e-8907-d5d045742c03 authorizes the bounded throwaway Claude/Codex capture and causal replay first. Retain the refusal's actual NativeQueryEvidence before cleanup removes the private store; retain that diagnostic improvement regardless of the selected correction. No production ownership or ordering change is authorized yet. After capture, bring the concrete correction choice to the hub and append its approved design before changing it.

The initial diagnostic stage changes tests and the existing harness evidence renderer only. Reuse the persisted bounded LifecycleFailure.Zellij* fields; test that the pre-cleanup renderer emits the actual command, version, exit, byte counts, truncation and raw stdout/stderr prefixes, never synthesizing a valid JSON reply from a truncated prefix. The producer's4KiB per-stream cap and existing max-four failure selection remain unchanged. Native runs use the shared testcouch fleet and its disposable stores; no runtime/fleet stage.

### 2026-10-22 — Native cause established; concrete correction proposed

Both native captures on725e24c574b2907d79121071ddff26bbe2ddd685 reproduced the same state. The exact agent is gone. Pane0 is exited=true, exit_status=0, is_held=true and has no pane_command. The terminal pane and a correctly owned live draft remain. The retained native stdout is complete (Claude3817 bytes, Codex3816 bytes), exit0, no stderr/truncation; the strict owner refusal is correct for that observation. Captured replies now replay verbatim through the shared fixture catalog and normal strict parser/classifier. Do not make the classifier accept these replies.

Normal park is not an uncovered reason to move the entire outer coordinator: Pair's RunCleanup first requires QuiesceSession; failed park returns before slot-tree retirement. Successful park already removes the exact session. The original confirmed terminal retirement likewise confirms its accepted servers absent. The faulty case is the slot-tree fallback's own added TERM loop, which leaves its held pane before quiescence.

**Proposed production correction (pending hub approval):**

1. In withSlotTree.Retire retain original retirement first, then call named session quiescence **before** signalling or awaiting the added wrapper/children. Only on successful quiescence run the existing immutable exact-PID stop loop and waits. Normal park, preservation, confirmation and all signal/timeout choices stay as they are. Failed/unknown quiescence returns before this loop; a source independently stopped by park/original retirement is already absent and needs no new ownership proof.
2. Carry the positive owner observation's exact server through deletion. Today ScopedThreadArtifactCollisionChecker revalidates the observed server, then DeleteSessionContext independently chooses its initial allowed server from a later census. Avoid that re-admission: add a local DeleteOwnedSessionContext(ctx, SessionOwnerObservation) path on OSRuntime using existing quiesceAuthorizedSession with that one exact server. Validate positive owner state and full exact server identity before entering it. New or replaced servers refuse; disappearance may complete normally; each native signal still rereads identity/argv. Share the existing timeout/poll/ops selection so no second policy copy appears.
3. Use one scoped owned-quiescence helper for all three positive-owner deletion sites in artifactcollision.go (Quiesce, QuiesceNamed, TriggerBoundQuit). A deleter that cannot consume the exact owner proof refuses explicitly; there is no fallback to name-only deletion. Upgrade existing local stateful test deleters. This is a Go-internal capability, not a wire change or persisted permit.
4. Keep bounded pane-JSON retention. Refine the concise pure classifier diagnostic to distinguish missing/empty actual command from ambiguous command text and identify its pane, without copying arbitrary argv into the concise error. No classifying permission changes.

| Entity | Kind / path | Change |
| --- | --- | --- |
| withSlotTree retirement sequence | integration / couchcore/recovery_slot_tree.go | quiescence precedes its own added signals; no new coordinator state |
| owned-quiescence helper | integration / couchcore/artifactcollision.go | one current positive binding feeds one exact-server deletion path for all three consumers |
| DeleteOwnedSessionContext | integration / launcher/lifecycle_os.go | explicit exact positive owner input; existing immutable authorization/quiescence loop |
| ClassifySessionOwner diagnostics | pure / launcher/session_owner.go | specific refusal text only; same unknown/foreign/owned decisions |

No new stored or process-lifetime state is introduced. The existing bound caller/thread gates remain lifetime owners. The positive observation is call-local and passed directly to the effect owner, never cached. Native server replacement cannot be serialized by a Go lock; the accepted generation is immutable and every signal must re-prove it, so replacement is a veto rather than a new target.

Regression sequence: convert the current characterization into the desired successful recovery assertion and first observe it fail. Add a quiescence failure/unknown matrix proving no added wrapper signal, a successful normal-park/absent-session path, and original-retirement error/absent paths. Replace the native server between owner revalidation and deletion and during the quiescence loop: neither the original-gone nor replacement PID may gain new signal authority; test independent exits and cancellation. Keep all original stop-set, hub, generation, foreign owner and no-launch-on-uncertainty tests.

After the concrete plan is approved, rerun change-code if design/flow requires it, implement TDD, then convert the native diagnostic run to successful reboot+delivery acceptance for Claude and Codex. No additional public command or deadline is introduced. The unrelated lossy joined-argv producer hazard remains a separately scoped discovery; this correction never reconstructs permission from missing or malformed text.

### 2026-10-22 — Concrete production correction approved

Authenticated ops:0 approval a5fddddd-f74f-4737-99de-8bf35d5658b5 approves the complete concrete revision above: original retirement first, named quiescence before added TERM/waits, unchanged outer park, exact observed server passed to quiesceAuthorizedSession through DeleteOwnedSessionContext, all three scoped deletion consumers sharing it, and explicit refusal for unsupported deleters. Required proof: replacement-between-check-and-effect and unknown-no-wrapper-signal regressions, then successful native reboot plus delivery. No other state/protocol, deadline or parser permission change is authorized.

### 2026-10-22 — BR-1 core-concept classification repair

The initial pure-decisions row grouped RecoveryTerminalPlan data with checkConfirmed. That label was inaccurate: checkConfirmed calls readRecoveryReady, whose default is osfs.ReadRegularMetadata, and confirmedRecoveryFixture injects the reader. Split immutable plan data (PURE) from the validator (INTEGRATION) in the tables above. No implementation or IO seam changes. Swept all core-concept rows: the other pure classifier/argument parser is IO-free, while lifecycle, probe, scoped checker and disposable fleet remain integration entities. This corrects the documented classification rather than changing the approved mechanism.

### 2026-10-22 — BR-2 startup discovery separated by TL disposition

Authenticated ops:0 disposition5a670f4e-1ef7-49da-99ba-835f86d2814b keeps this correction scoped to ownership and quiescence order. Separate unclaimed #806 retains the reviewer Codex replacement-before-first-wrapper disconnect, the exact-head successful diagnostic, both log hashes and #773's potentially related startup evidence. The initiating trigger remains unclassified; Server-not-ready and independent global queries are hypotheses, not a causal verdict or registered flake. Re-close this item with the preserved failure and measured exact-head success; no startup mechanism, bound or parser permission changes are introduced here.

### 2026-10-22 — Correct pre-retirement capture-change refusal

Per TL513c014e-0e54-4c8a-986a-76a5008a2f19, one focused trace distinguishes a changed capture from a guard misfire. The exact refusal branch requires a fresh size/UTC-mtime/device/inode tuple differing from the directly captured identity, and the existing unchanged/real-write regression passes. The r2 Codex run kept its old agent alive and started no replacement. The retained log lacks the before/after tuples, so exact file/bytes/redraw cause remains unclaimed. Record this as correct refusal behavior, not a defect or flake; no additional native run or implementation change. The positive unchanged-evidence acceptance already completed on both agents; the negative changed-evidence refusal preserves its existing contract.

### 2026-10-22 — Self-found test isolation repair after SHIP

The direct caller regression passed a private checker DataDir but omitted private HOME for TriggerBoundQuit's compatibility quit marker. Isolate HOME and defend direct in-process quit-trigger test calls with the existing Pair lint catalog, sweeping refusal-only callers too. This is executable-test isolation only; production ownership/quiescence and native acceptance assertions remain unchanged. Cancel the now-obsolete queued source check, publish the corrected test head, run a fresh recorded check, and narrow re-close before merge.

### 2026-10-22 — Full review required for lint-policy addition

The fix-review confirms the HOME repairs and rule tests but treats the new executable lint policy as expanded scope outside its narrow authorization. Run full sdlc close on the expanded head; this supersedes the prior revision's narrow re-close instruction. Production/native acceptance code is unchanged, and the already-queued corrected-source suite remains valid. No gate bypass or another suite restart.
