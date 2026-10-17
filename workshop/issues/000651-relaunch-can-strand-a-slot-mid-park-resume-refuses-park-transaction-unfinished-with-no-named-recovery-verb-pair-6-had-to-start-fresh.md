---
id: 000651
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f89b2484b85403f651081300d7516d9fabb75e75' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T13:48:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Relaunch can strand a slot mid-park: resume refuses 'park transaction unfinished' with no named recovery verb (pair:6 had to start fresh)

## Problem
A failed relaunch can retain an open park after its Pair/terminal has gone. Resume then refuses with “recover that park or Start fresh”, but park recovery is not a public CLI verb and no retry/recover menu action is offered. Even direct controller Recover cannot resolve an absent terminal with missing or failed completion.


## Spec
Preserve the conversation through interrupted relaunch without bypassing exact process identity, terminal ownership, store revision or completion authority. Before request publication is attempted, cancellation/failure can roll back the newly begun park. After publication may have occurred, retain the transaction and reconcile its exact attempts: never invent a successful park or undo unknown external effects.

Expose a confirmed, idempotent recover-park operation through the hub CLI and the thread action menu. It completes a matching successful park, drives an existing live attempt when safe, or aborts an interrupted dead-owner transaction and retires only its exact positively dead incarnation. This last outcome preserves native binding/history and explicitly reports incomplete cleanup/scrollback; it does not manufacture VerifiedPark or mean filesystem effects were reversed. Later resume uses its normal native-conversation checks.

Every unfinished-park refusal and partial relaunch receipt names the recovery verb. All paths classify the durable state after a partial write; a failed pre-begin commit must not claim Pair stopped, and a finalized park must not claim an open transaction.


2026-10-17 ~11:50: `couch --relaunch pair:6 --confirm`, a Codex slot on tag 1-pair-55 whose binding had just been confirmed, didn't finish. Its output ended with a recovery menu ("abandon — give up on it and leave the thread occupied"), and the slot had no live wrapper afterwards. `couch --resume pair:6` then refused: "resume-parking: a park transaction is unfinished; recover that park or Start fresh". `couch --help` names no verb for recovering a park, and the resume candidates listed only "Resume from disk (confirmed)", which refuses, and "Start fresh". `scrollback-1-pair-55-codex.raw` was missing, apparently moved by the half-done park. The hub had to discard the conversation with a fresh reboot. That was cheap here because pair:6 was idle between items, but it would lose real work mid-item.

Earlier the same day, pair:4's reboot failed similarly at zellij session ownership ("ownership unresolved"), which may be pair#640's reply misrouting.

Wanted:
- A relaunch either completes or rolls back its park, so the slot stays resumable from disk.
- An interrupted park is recoverable by a named verb, shown in the refusal: finish it, or restore the scrollback and binding, then resume.
- The relaunch's failure output names the step that failed (park, kill, spawn) and the recovery command.

## Done when
- Isolated cancellation/fault and journal-restart tests cover every durable park boundary plus all six cleanup stages; each outcome retains independently checked ownership and conversation invariants.
- Pre-publication rollback leaves the original live incarnation intact. Matching success finalizes once; dead-owner interruption can be retired without fresh conversation or forged cleanup success. Unknown/live competing owners remain protected.
- A real executable, confirmed recover-park command and matching menu action are available from each applicable refusal; repeated recovery is harmless and never resumes/submits automatically.
- Required suite, mandatory close review, documentation and TL-coordinated live relaunch/interruption/recovery acceptance pass before landing (unless TL explicitly revises live scheduling).


- A test interrupts a relaunch at each park step; afterwards `couch --resume` succeeds, or the refusal names a recovery verb that, once run, makes it succeed.
- The relaunch failure text names the failed step and the recovery command.

## Plan
See [durable implementation plan](../plans/000651-interrupted-park-recovery-plan.md). Await TL approval before change-code or production edits.

- [ ] Promote the isolated interruption matrix to regressions and implement bounded rollback/recovery over the existing park authority.
- [ ] Expose recover-park through confirmed CLI/menu/receipts and make all interrupted-park advice actionable.
- [ ] Run targeted/race, fake/native conformance and required suite; coordinate live acceptance, close review and landing.


- [ ]

## Log

### 2026-10-17

### 2026-10-17 — diagnosis before implementation
Claimed and started from main. start-plan refreshed the stale ariadne tool build and explicitly requested an unchanged second invocation; the second invocation created the canonical branch successfully. No gate bypass.

Read-only audit traced Relaunch → ParkExpected → BeginPark → PublishRequest → request acknowledgement → quit trigger → completion/death → FinalizePark → Resume. Existing controller modes are not a public CLI: park is PresentationTUI and slot admission rejects park. Menu does not expose recovery modes. Existing startup reconciliation inspects/publishes but deliberately does not trigger another quit.

Diagnostic source base from Git: 6df6562411120a83c21262ffc95dfe6e7fa496d6. Temporary same-package probe used existing real ThreadStore plus stateful Pair lifecycle/terminal/process fakes and store hooks. Twelve boundary cases and six cleanup-stage faults pass as reproductions: before begin; canceled after begin; before/after request publication; before request acknowledgement; before/after trigger; completion before child death; dead child without completion; before finalization; interrupted finalization journal; after finalization cleanup; each RunCleanup stage. Missing completion with absent terminal, and failed cleanup with dead owner, both retain open Park and Recover refuses; resume reproduces ResumeParking with no executable advice. Before-begin failure and journal-finalized/canceled returns misleadingly say the park is open. Recover completes the other exact-evidence cases. These are controlled fake/IO failures and context cancellation, not native process-kill evidence or proof of the original pair:6 ordering.

Retained directory: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair651-repro.6hqyct. Command: go test ./cmd/internal/couchcore -run '^TestProbe651' -count=1 -v; exit0,8.858s. probe.log SHA256 70667c6c8714d99f23ad0137e60337e50a7672ac830d0ce38407e62b24cc70bf; probe651_test.go SHA256 33ede8c19255653279f4270e931e689192437453b308456c388837778d9ebbd0. evidence.json records source head/command. Temporary probe removed from checkout after retention; no production code changed.

Verified pair:6 receipt033e6b3d: its previous recovery screen is unavailable and it never received the original failed relaunch receipt. It can substantiate only TL's earlier “stopped partway through parking scrollback” report, not a prior tag/phase or exact ordering. Do not infer original identity from its new tag.

ARCH-ORDER/SECURE shape the proposed distinction between unpublished rollback, matching completion and dead-owner retirement; ARCH-DRY reuses existing worker, park transition, exact ownership and prepared resume rather than adding a parallel launch or destructive Fresh fallback. TL approval remains pending.
