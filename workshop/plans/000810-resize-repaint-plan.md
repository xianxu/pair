---
collaboration-mode: discussion
---

# Recover peer delivery after an unrepainted resize

> **For agentic workers:** Follow AGENTS.md §3. Use the executing-plans skill after TL approval and the change-code gate. SDLC owns the single close review. This is planning only until #791 lands; do not stack implementation on its unlanded branch.

**Goal:** A peer delivery waiting on resize-invalidated cursor evidence requests a harmless native redraw within a few seconds and resumes only after a newly observed, fully qualified empty composer.

**Architecture:** The existing stdin writer owns a bounded recovery transition, driven by its existing one-second peer poll. It can request one SIGWINCH per real resize through the owned child-group lifetime fence. A redraw request never grants input authority; the existing composer/frame/paste/operator/automatic-input checks decide every eventual paste.

**Tech stack:** Go, terminalModel, existing automatic-input/admission owner, ownedGroup, native SIGWINCH, shared testcouch stateful agent fixture, existing isolated native conformance harness.

## Evidence and dependency

TL authorization9a7378bb-68de-4a95-bb1c-129f827a79c1 accepts the controlled mechanism plus the operator's observations: the viewed slot stalls after multiline tab-bar changes, and a keystroke releases the batch. The historical recordings cannot establish precise resize-to-repaint elapsed time because resize events have no timestamps.

#791 supplies the sanitized native94x39→38→39 sequence, coherent latest-resize timing and the controlled silence test. With native idle paint, real geometry change, then withheld output, delivery remains specifically cursor-hidden at0/15/30/300 simulated seconds; feeding the retained native repaint immediately restores admission. This reproduces the mechanism, not historical causation. Static banner/footer recognition is already correct. #791 remains in the landing queue; implementation here waits for that parent to land, then brings main in normally. Planning branch starts independently at main651e30092e7ca245e5537ba1450b7ea7eb36ddae, without cherry-picking #791.

Existing anchors on this base:

- `terminal_model.go:133` PrepareResize and resolve invalidate cursor evidence for changed geometry; same-size synchronized events preserve evidence (#735). #791's pending SnapshotDiagnostic exposes the exact transaction alongside the screen.
- `wrap.go:1675` existing peer tick invokes dispatchPeer. No new polling goroutine is needed.
- `dim.go:197` signalChildWinch already requests SIGWINCH for focus-only redraws. Its current direct-PID fallback does not provide a lifetime fence for a new delayed effect; reuse the child-group mechanism through its actual owner.
- `owned_group.go:13` retains the direct child's unreaped leader until cleanup ends, fencing group-number reuse. New delayed redraw and StartStop/Wait must serialize on that owner rather than depend on a liveness scan.
- `testcouch/agentfixture/cmd/main.go:61` forwards native SIGWINCH dimensions; `agentfixture/run.go:209` paints on each size notification, including unchanged geometry. That is controlled fake behavior, not proof installed agents do it.
- Installed Codex's JS launcher forwards only INT/TERM/HUP, so native conformance must signal the owned process group, not just that launcher. Actual unchanged-geometry redraw support is unverified for Claude2.1.308 and Codex0.163.0.

## Alternatives and recommendation

1. **Recommended: one explicit owned-group SIGWINCH, then ordinary re-observation.** Reuses the existing redraw seam, keeps current geometry, sends no PTY input, and preserves every admission guard. Native support must pass isolated conformance before this is accepted as the recovery mechanism.
2. **Force a temporary geometry change then restore.** More intrusive: it changes layout and can still coalesce into one ignored event. This requires extra geometry ownership/order and is not included. If unchanged-size SIGWINCH fails native conformance, stop and return to TL with that measurement before choosing it.
3. **Reuse pre-resize cursor visibility.** Rejected: it would authorize input from stale evidence and violates the explicit requirement to keep the cursor check.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| PeerResizeRecovery / advancePeerResizeRecovery | cmd/internal/wrapcmd/peer_resize_recovery.go | new |
| PeerResizeRecoveryObservation / PeerResizeRecoveryDecision | cmd/internal/wrapcmd/peer_resize_recovery.go | new |
| ResizeInvalidationObservation | cmd/internal/wrapcmd/terminal_model.go | new pure value |

PeerResizeRecovery is one fixed-size value owned by the stdin writer, not one entry per delivery. Its key is a wrapper-lifetime cursor-invalidation sequence supplied by terminalModel; retain only the latest sequence/attempt phase. Existing #791 timing cannot safely act as a unique sequence across clock changes. Retain one ResizeInvalidationObservation (sequence, committed time, outcome) separately from latest-attempt diagnostics. Update it only when PrepareResize actually invalidates evidence (changed dimensions or same-size resynchronization after an abort), then resolve that same transaction on commit/abort. A synchronized same-size no-op never replaces it. Thus duplicate SIGWINCH cannot hide the relevant invalidation, reset its clock or renew its budget; an aborted change followed by a successful same-size retry gets a new committed invalidation.

The transition receives time, current resize key/timing, complete/visible flags, current pre-paste refusal and owner eligibility. After2 seconds of a committed cursor invalidation with no restored cursor, it emits at most one redraw request for that invalidation sequence. Polling at one second means the request is due by3 seconds. Prepared/aborted geometry, an absent/unsupported agent, active drive, partial/pending human input, picker/image, active turn, ended delivery or stopping child does not admit the effect. Clock rollback restarts the bounded wait without new attempt authority. A head change cannot request a second redraw for the same resize. A later valid native observation goes through normal admission; neither request nor success status invents visibility.

If the child has not repainted within2 more seconds, retain the precise cursor-hidden refusal and expose `redraw requested; no confirmed repaint` in existing local diagnostic/detail surfaces. It remains queued under existing delivery lifetime/deadline rules; do not loop signals or reset that lifetime. A later native repaint may still unblock it normally. A new cursor-invalidating resize supplies a new bounded opportunity.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| terminalModel resize observation | cmd/internal/wrapcmd/terminal_model.go | modified after parent lands | cursor-invalidation sequence/completion under existing mutex |
| dispatchPeer recovery handoff | cmd/internal/wrapcmd/peer_delivery.go; peer_resize_recovery.go | modified/new | existing stdin writer and queued peer poll |
| ownedGroup.SignalLive | cmd/internal/wrapcmd/owned_group.go | new | bounded SIGWINCH syscall while leader remains unreaped |
| signalChildWinch | cmd/internal/wrapcmd/dim.go | modified | common guarded redraw owner, reused by focus and recovery |
| agentfixture resize control | cmd/internal/testcouch/agentfixture/run.go; cmd/main.go | modified only as needed for deterministic suppression | controlled native redraw/no-redraw behavior |
| native redraw conformance | cmd/internal/wrapcmd/peer_live_test.go; harness_tty_live_test.go | modified | isolated real Claude/Codex PTYs and joined cleanup |

Add one mutex and stopping state to ownedGroup: SignalLive holds it through the bounded signal syscall; StartStop revokes admission under the same mutex before cleanup starts, and Wait reaps only after revocation. This unreaped leader is the lifetime resource. No PID discovery, host process census, new global lock or background retry worker. Existing test injection routes through the same request boundary and can return failure/ignored/delayed repaint; a signal error or uncertain response never becomes delivery success. Preserve the established focus-redraw behavior and exercise all its consumers.

Select the recovery request under the stdin owner's normal state turn and emit after admission/mailbox unlock. Recheck its current delivery/resize token before the request. A resize racing after that check may receive a harmless redundant redraw; it still gains no input authority, and the new geometry's ordinary cursor invalidation wins. Child stop and signal are serialized by ownedGroup, so no redraw can target a reaped/reused group. There is no async effect surviving the scope.

Fake state: PTY geometry, initial recognized idle frame, suppression of the first genuine resize paint, pending redraw request, chosen immediate/delayed/ignored/error response, operator draft/picker/working state, output-generation log and input bytes. Reuse the shared agentfixture/Fleet ownership machinery; do not build a private Couch or force a real fleet slot into this state.

## Visible states and recovery

Actions: wait for native repaint; inspect receipt/diagnostics; request authorized wrapper refresh.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Real resize, cursor invalidated, below2s | waiting for recognized composer: cursor hidden | wait for native repaint | delivery delay only |
| One redraw requested, response pending | redraw requested; still waiting for recognized composer | wait for native repaint | no input admitted |
| Native qualified empty repaint | ordinary delivery/confirmation | inspect receipt/diagnostics | existing uncertain-delivery outcomes remain |
| Repaint shows a draft, picker, work or incomplete frame | existing precise hold/wait | wait for native repaint | no draft loss or forced submission |
| Signal fails/ignored or2s response window expires | cursor hidden; no confirmed repaint | inspect receipt/diagnostics | automatic recovery unavailable; queue still safe |
| Aborted resize, or retry still incomplete | existing incomplete/hidden wait | wait for native repaint | recovery requires a committed invalidation |
| Wrapper/child stops or binding changes | existing exited/released receipt | request authorized wrapper refresh | this in-memory attempt state disappears |

Covered: resize-with-silence for Claude/Codex, ordinary delivery and Claude batches, no typed recovery input, no repeated signals for the same actual resize, late output and stop races. Not covered: historical causation, arbitrary non-resize recognition failures, rewriting native agent code, or automatic fallback geometry bouncing. Qoder runs the same safe-refusal tests and native check when credits are available; no support claim without evidence.

## Work and residue budgets

Let R be admitted real resizes, P queued peer polls, C copied screen cells and B emitted diagnostic bytes. Existing screen work remains O(C) per dispatch; new transition is O(1) per poll, one bounded signal per cursor-invalidating resize, no table/filesystem scans. Same-size events cannot increase signal count for the same actual-change sequence. Count at8/16 resizes,8/16 heads and repeated8/16 polls: requests <=R (count resynchronizing retries in R), zero on ineligible polls, no per-head multiplication. Generated interleavings independently assert once-per-resize and no request after owner stop. Diagnostic records continue using #791's whole64KiB cap and existing retention.

Creates no durable artifact family or background process: one fixed-size proxy recovery value and terminal sequence die with the wrapper; ownedGroup's handle dies after joined cleanup; existing diagnostic writer owns record rotation/GC. Sequence exhaustion refuses recovery without changing admission. Additional diagnostic fields are additive only; no persisted read-modify-write format is introduced (ARCH-EVOLVE).

## Implementation tasks (one atomic review boundary)

- [ ] After TL approval and parent landing, integrate current main, verify #791 is an ancestor, and run change-code. Derive an estimate only after plan-quality passes. Do not stack or cherry-pick parent code.
- [ ] Make the controlled resize-then-silence replay the behavioral red: a stateful agent suppresses first resize repaint, receives no operator bytes, honors the redraw request and reaches ordinary safe delivery within5 seconds. Run both Claude/Codex fake profiles and Claude batch readiness. Keep #791's silence-preserves-refusal control for agents that ignore requests.
- [ ] Prove the native premise before declaring recovery supported. Before implementing the recovery transition, add opt-in redraw-only conformance to the existing isolated harness for real installed Claude and Codex: initial confirmed idle and bounded quiet baseline, explicit unchanged-geometry process-group SIGWINCH, fresh complete paint within2s, qualified composer and no submitted prompt. Record versions, precise resize/request/paint times and sanitized bytes, then join process-group and PTY cleanup. Extend the shared disposable-Couch harness rather than writing a private fleet. If either real agent ignores unchanged-size SIGWINCH, stop and report the measured result before selecting another mechanism.
- [ ] Add the ownedGroup lifetime signal boundary first, keeping its unreaped-leader proof. Test signal/start-stop/wait interleavings with causally controlled native children; assert no signal after revocation/reap and preserve focus-redraw tests.
- [ ] Implement advancePeerResizeRecovery, cursor-invalidation sequence observation and the thin dispatch handoff. Keep all composer/cursor/frame/input/automatic admission decisions unchanged. Requests and diagnostic IO execute outside mailbox/admission locks; existing owner handles cancellation/stop and no new worker is spawned.
- [ ] Risky functions: advancePeerResizeRecovery → generated time/resize/head/input/exit sequences with independent signal-bound/no-input invariants; PrepareResize/SnapshotDiagnostic → concurrent actual/no-op/abort sequences with coherent keys and no budget resets; ownedGroup.SignalLive/StartStop/Wait → controlled before/after revocation races and native group identity/lifetime checks; dispatchPeer → stateful delayed/ignored/error repaint, draft/picker/work/partial-output controls with recorded input bytes; encodePeerWaitCapture → existing arbitrary-cell fuzz and whole-record cap with new fixed metadata.

- [ ] Run complete changed packages, targeted race/growth/sequence tests, existing automatic-input/#735/agent-drive/focus controls, source/reader inventory and lint. Update atlas/couch.md and couch-testing.md; existing index links already cover them. Native conformance is explicit and bounded, not added as an unbounded default suite dependency.
- [ ] Push and submit the declared recorded affected/full-suite check, then close through SDLC's mandatory fresh review. TL coordinates any real operator staging/refresh; no unilateral :0 stage or live-fleet signals. Land only after final recorded pass and the required native support evidence.

## Architectural accounting

ARCH-DRY: reuse the stdin owner, existing tick, signalChildWinch and shared native harness. ARCH-PURE: one deterministic eligibility transition, thin signal/terminal shell. ARCH-PURPOSE: bounded recovery after actual silence rather than only more diagnostics. ARCH-MOCK: controlled stateful repaint/no-repaint model plus real-agent conformance. ARCH-CONSTRAINTS:2s eligibility, one signal/cursor-invalidating resize,2s response observation, fixed memory and counted growth. ARCH-SECURE: no PTY recovery bytes, no invented cursor evidence, lifetime-fenced owned group, private captures. ARCH-ORDER: owner/stop fence, exact resize keys, late response cannot override newer geometry or normal input guards. ARCH-FUNERAL: no new durable family/worker, existing joined cleanup and diagnostic retention. ARCH-EVOLVE: additive observation metadata only.

## Approval status

Approved by TL4eb28919-f71c-41c0-ab9f-8fd610dfab6b. Native capability experiment may run now, before parent landing; production changes still wait for #791. No #810 production changes or native launches have run at this checkpoint. Signal support in installed agents remains an explicit unverified prerequisite; if it fails, bring a revised design instead of silently switching to geometry bouncing.


## Revisions

### 2026-10-22 — native SIGWINCH premise failed; implementation parked

Claude2.1.308 ignored one unchanged-size owned-group SIGWINCH after2s confirmed quiet: no output/repaint within2s, zero post-request PTY bytes, joined cleanup nil, controlled capability assertion FAIL8.298s. Exact times/private evidence and raw hash are in the issue Log. Stopped before Codex and before production code as required.

TL1e66b2d3-7981-4155-9929-76ad686f0bc1 supersedes this plan's implementation tasks: wait for #796's default collapse style and #791's capture to land, then revisit only if stalls recur with new capture evidence. The one-row geometry bounce remains a candidate requiring explicit TL approval. No automatic fallback or weakened cursor check is authorized. Keep the draft's ownership/test analysis as design history, not an executable approved recovery mechanism. Land #791 when its retained suite passes, then resume #807.
