---
id: 000595
status: codecomplete
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-16
estimate_hours: 2.15
card_mirror: '81d596abbcf451638398f67d1ad661ee00256da2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T19:45:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.73
---

# Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store

## Problem

pair#556 (with #528's observer and #584's fixes) guards "no agent conversation has two live writers" by forensics over
historical state: a scan of every OS process, argv, wrapper readiness files, parent chains and birth times, launch
ledgers decoded for conversation IDs, `lsof` of held transcripts, observed twice and compared, plus a reservation
store. About 1,700 non-test lines and ~45 refusal reasons; #584 needed eight design deltas to make it work on this
machine (unreadable root processes, 18 of 21 older wrappers without agent_identity, unbound launches, a 21 MB ledger, a
15 s decode, a 5 s budget under load). The operator and TL agree it's over-engineered (ops QA
`2026-10-15-37-whats-in-pair-0.md`; fresh-context review). #556's plan never weighed a lock that dies with its owner.

## Spec

A Pair wrapper holds a nonblocking exclusive flock for each exact conversation ID it knows until it exits. Known IDs lock before native startup; Fresh unknown IDs lock when the existing Pair watcher reports the current launch's binding. No native fd inheritance, native ID bootstrap or new discovery/forensics. Holder metadata supplies a label only; held means refuse with the holder slot and offer Fresh.

The native agent runs in its own wrapper-owned process group. Every controlled wrapper exit terminates that group before reaping the leader and releasing locks. Wrapper SIGKILL can leave an orphan engine after the lock drops: that accepted gap belongs to #583's reaper and #587's unattributed report, not this admission guard. Agents outside Pair are outside its guarantee.

One lock build: delete the global native owner scan, legacy/unbound fallbacks and durable admission reservation store in the same change; no launch takes native admission. There is no cutover gate (TL e93b109d): a resume whose conversation lock another live wrapper holds refuses at native start with `conversation X is in use by <holder>; Start fresh`, the holder named as Couch names a slot (`pair:3`), else its Pair tag, else `standalone`. That is the same refusal as Couch's registry `in use by` (#597), observed by the kernel. Wrappers started before this build hold no lock until their slot is relaunched; the gap closes as slots relaunch. Startup, Continue and Fresh remain available. Preserve/move the narrow confirmed stop-set attribution helpers.

## Done when

- Real kernel and wrapper tests show one winner for two wrappers on one ID, a diagnostic holder label, no losing-writer metadata damage, and lock release on wrapper exit including SIGKILL.
- Controlled normal/error/signal/restart exits stop only the owned agent group before reap/lock release; tests explicitly document the accepted SIGKILL orphan gap rather than promising inherited-agent protection.
- Current-launch unknown-to-known IDs acquire through existing discovery; stale/duplicate events and a late collision are handled without a new native integration or historical scan.
- Startup/Continue/Fresh are unaffected; a resume of a conversation another live wrapper holds refuses naming the holder; a binding notice after stop or with an unusable ID never fails the wrapper; controlled restart still re-execs.
- The forensic and reservation code is gone, shared consumers and stop-set safety remain, and close reports actual net production/test LOC deletion. Build, required recorded suite and SHIP review pass.
- Operator live maintenance acceptance: restart over old live slots, migrate them once at a quiet time, resume a parked scratch conversation, and refuse a held scratch conversation with its slot named. This remains pending until performed; any landing before it requires the TL's explicit direction.

## Plan

Durable implementation plan: workshop/plans/000595-conversation-flock-plan.md (one atomic close boundary).

- [x] Obtain TL approval of the wrapper-lifetime/capability-cutover revision before change-code.
- [x] Implement and verify the small lock, existing binding notification and owned-group cleanup; remove forensic/reservation authority in the same build, preserving moved source stop-set attribution.
- [x] Run full verification and SHIP review; report actual net deletion (prod -1156, tests -2541 vs main).
- [ ] Operator maintenance acceptance, then land as the TL directs (held for #589).

## Log



- 2026-10-16: closed — Post-close fix for CI couch-zellij-conformance (PR #369): TestNativeConsoleWrapperZellij/wrapped-zellij left notify/lock under its fixture because the wrapper (pane child) was still finishing cleanup ~60 ms after the zellij server exited (measured with a probe; wrapper reparented to PID 1, then exited). The test now joins the wrapper by its pidfile, bounded 5 s, before removing the fixture. make -f Makefile.local test-native-terminal-ci green 3/3 locally; main passes the job; no product change. no-plan-check: the open item is the operator maintenance acceptance, pending by design. (fixed-then-ship); review verdict: SHIP
- 2026-10-16: closed — Recorded suite green on 7a7b6086 (round-1 fixes). Round-2 fixes on c1a5e8b9: per-platform live-member probe (darwin kill probe, Linux /proc non-zombie members; TestProcStatLiveInGroup; linux vet clean), slot label resolved in the background and held locks relabelled (TestSetHolderRelabelsHeldLocks), plan states the approved design. Touched packages green: nativelock, wrapcmd, launcher, artifactpath. Suite runs on c1a5e8b9 failed only on load-sensitive tests unrelated to this change (test-review nvim branch-mismatch flake; TestPublicWarmAttach zellij list-sessions timeout at load avg 30), each passing in isolation; suite rerun before merge. Net vs main: prod -1095ish, tests -2418ish (recount at merge). no-plan-check: the open item is the operator maintenance acceptance, pending by design.; review verdict: FIX-THEN-SHIP
### 2026-10-15

- Claimed after #594 landed (47a485ce46ffb685ce951cfb16428fef2c2f1bcd); start-plan prepares a fresh main issue branch. No stacked development or implementation.
- Read-only caller inventory found that source stop-set attribution still consumes recoveryNativeWrappers/recoveryWrapperReady, so those narrow duties must be moved rather than blindly deleted. Other shared JSON/ledger and native inventory consumers must remain.
- Scratch Darwin flock measurements passed: second open including same-process reopen contends errno 35; dup and inherited child retain the lock after original close/wrapper SIGKILL; last-holder SIGKILL and reap frees it. Full scripts/results: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-595-flock.XreBFC/.
- Exact installed Codex shim spawn shape (stdio inherit) drops fd3 in its fake native child. After Node SIGKILL the fake engine remained alive while a contender acquired: a measured contradiction of merely passing ExtraFiles to the shim. Also, Codex/Agy/Muse Fresh IDs arrive after exec via the detached watcher; a newly acquired descriptor cannot retroactively be inherited. No real coding agent/slot was launched or signalled. Reported early to the hub (f0ae6753); durable draft workshop/plans/000595-conversation-flock-plan.md records the choices and a proposed two-boundary migration. PLAN ONLY pending decision.


- TL approved 2833b837 in receipt 659dde9c, verified submitted to this incarnation. change-code passed at 52ba9463 after plan-quality CLEAN (PQ-1..5 addressed) and estimate-quality INFO (2.15h provisional; integration/verification allowances optimistic). No code before gate.
- Implementation checkpoint: secure one-attempt nativelock + private pure Advance; wrapper handles held through joined own-group cleanup; bounded watcher pipe reports only committed current-launch bindings; readiness capability + host-wide listed-live inventory replaces global forensic admission. Old scan/store and writable-lsof-only helpers/tests are removed; exact confirmed stop-set attribution moved, selectors retained.
- Confirmed Relaunch's existing ParkExpected precedes ordinary ResumeContext, so the exact post-park source agent/ID travels in a private context and is copied into the existing persisted/validated RecoveryIntent.Relaunch marker for that cold start only. There is no user override. New/fork --session-id destinations are now retained by the shared selector parser, distinct from the fork source.
- Early read-only review found three real integration gaps: explicit new/fork IDs discarded before lock, notification failure interrupting config publication, and inherited session IDs breaking non-native wrap. Each reproduced RED; wrapper/selector/watcher regressions now GREEN. The notification remains best effort and cannot interrupt existing discovery persistence/lifecycle work.
- Primitive race/fuzz and real scratch subprocess contention, normal/SIGKILL release passed; 1,277 fuzz executions. APFS lock residue measured 6 metadata bytes / 4,096 allocated bytes / one stable inode. No real agent or live slot mutated.
- Targeted integration command across artifactpath/sessionwatch/wrapcmd/couchcore/launcher passed (2026-10-15 20:20): wrapper lock/pipe, legacy barriers both gates, exact migration context, notification failure, successful Relaunch profile, exhaustive source inventory and dead-symbol checks. Full verification, build, mandatory close review and operator maintenance acceptance remain pending.

- Checkpoint follow-through: make build passed (slot2 only; no stage). Focused race tests passed across nativelock/wrapcmd/sessionwatch/launcher/couchcore, including real lock subprocess release, pure lifecycle, owned group, pipe and migration. One test initially read cmd.Process.Pid after Wait (Go may set it to -1); it now retains the originally proved group ID. The migration fake clock now uses tagged-intent CreatedAt rather than absent inferred Context time. Broader recovery package run remains in flight; earlier launcher fixture failures in that already-running binary predate these fixes. Current counted Go diff: 1,127 production added / 2,052 removed (net -925); 1,032 tests added / 3,268 removed (net -2,236), provisional until final close. Atlas now describes the single lock and maintenance path; stale scan/store/lsof authority prose removed.

- Merged origin/main including #587 (2156c014): two content conflicts only. Artifact manifest retains all resource-sampler entries plus native-lock sources, deleting the obsolete admission entry. Atlas retains #587 resource monitoring and #595 replacement admission text; no sampler behavior changed. Broader recovery run passed couchcore (108.973s), wrapper (38.005s), watcher; launcher tagged fixture clock failure was fixed and its focused rerun passed.

- Urgent interruption (TL 368e95f0): #584 currently blocks real parked slots after tonight restart; checkpoint #595 and take a separate hotfix from main, then return. #595 recorded full suite has NOT started. Indirect Codex selector/ambient-ID regression reproduced RED and fix applied; latest combined race run reports OwnedGroupStopsDescendantsBeforeReap EPERM on the post-cleanup group probe (intermittent; earlier race run passed). Investigate honestly after hotfix, do not call it passed. No stage/live changes.

### 2026-10-15 — clean handoff to Claude (operator c6097dd3)

Stopped implementation on request. Pre-handoff checkpoint eb6528a5d23af2b898cb7f7e5e2517ca7dba630b contains the approved plan plus substantial UNFINISHED implementation; not SHIP, no recorded suite, no stage, no live maintenance acceptance. Plan workshop/plans/000595-conversation-flock-plan.md approved at2833b837, clarified plan gate CLEAN; implementation design checkpoint52ba9463. Main including #587 was merged at771e79785; two documentation/inventory conflicts resolved by preserving sampler and lock entries.

Inheritance measurements are preserved in the plan and Log: local scratch directory /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-595-flock.XreBFC/ contains measure.py, extra.py, spawn-proof.py plus JSONL results. Separate opens contend (errno35), dup/inherited child retains until last fd exits, holder death releases. Installed Codex Node shim spawn with stdio inherit drops fd3 in its fake native child: after Node SIGKILL, fake engine lives and a contender acquires. This disproved an inherited-child guarantee; TL explicitly approved wrapper lifetime only. No real agent or working slot was started/stopped. One lock measured 6 metadata bytes /4096 allocated /one permanent inode on APFS.

Retained implementation: nativelock primitive/state, exact new/fork destination selector field, wrapper lock/own-group cleanup, existing watcher binding pipe, readiness capability, maintenance check; global scan/reservation source/test deletion and retained stop-set helpers moved separately. Existing unknown-Fresh discovery remains best effort and visible. Partial net count was -925 production Go /-2236 test lines; recount final diff at close. Build passed before final small fixes; focused integration/source inventory/race checks passed, but latest combined race run intermittently reports OwnedGroupStopsDescendantsBeforeReap EPERM on post-cleanup group probe. Do not hide it: the earlier PID-after-Wait mistake was corrected to retained pgid, and a later run still failed; determine actual group observation before asserting conformance. Last indirect Codex alias/ambient-ID change is committed but that combined run failed on the group test.

Early read-only review found and fixed three RED/GREEN bugs: explicit --session-id/new-fork destinations must lock, discovery notification failure must preserve config/lifecycle work, and inherited session IDs must not affect non-native wrapped commands. Cold Relaunch additionally required an internal persisted RecoveryIntent.Relaunch marker, set only after successful exact-source park and agent/ID match, validated by existing recovery authority; this is in the checkpoint. Mandatory close review has NOT run.

Successor: reconcile with the operator's later #597 redirect (f4341d43: own group teardown, registry-only same-ID live collision, no global scans on Couch recovery, Fresh always reachable). #597 is handed off separately with its Fresh marker fix and findings; do not stack blindly or restore the removed forensic guard. Operator/TL own staging/maintenance; pair:0 untouched. No work remains uncommitted after handoff and both issues are explicitly released through sdlc, not marked complete.

## Revisions

### 2026-10-15 — measured constraints before implementation

The original strict orphan-child and late Fresh-ID requirements remain in force. This draft does not silently replace them with a wrapper-only promise. The TL/operator must choose a proven native integration or explicitly revise the guarantee before change-code. The proposed temporary migration build and final deletion are separate acceptance boundaries; final code retains one mechanism and no legacy fallback. Stable inode residue and exact cleanup timing are named in the draft for approval.

### 2026-10-15 19:58 PDT — TL scope decision replaces the original promise

TL cb293d05 explicitly narrows inherited-child/orphan protection to wrapper lifetime and chooses a maintenance cutover. This supersedes the initial fd-to-native guarantee, orphan-child-keeps-lock acceptance, temporary override and proposed dual-guard milestones. Controlled exits kill the wrapper-owned group; SIGKILL/orphan recovery is #583/#587. The active Spec, Done when and durable plan now agree. Resume-only legacy capability gating leaves the existing confirmed Relaunch available to perform migration, avoiding a circular all-slots refusal; there is no new override flag. No code changed.

Measured deletion baseline: 1,777 production + 2,766 test lines in dedicated old blocks, less 114 production lines moved/retained for stop-set safety, gives 1,663 replaceable production + 2,766 test lines. Allow 350–650 new production and 500–900 test lines: approximately 1,000–1,300 net production lines and 2,900–3,600 production-plus-test lines deleted, before extra bridge removals. This is a planning range, not a final diff claim. The revised plan is committed and sent before code.

### 2026-10-15 20:08 PDT — approval and plan-quality clarification

TL approved 2833b837 via submitted receipt 659dde9c, including confirmed Relaunch as the migration path. change-code full-flow plan review requested named transition/test seams and explicit discovery-channel failure semantics (PQ-1/2/3); these are now documented without expanding the wrapper-knows-ID promise. Unknown discovery loss is visible and retains unknown status; known-ID contention/IO failure stops the owned group. No implementation before the gate passes.

### 2026-10-16 — re-plan approved (TL e93b109d); close round 1 BR-1

Delta to Spec and Done when: the capability cutover gate ("relaunch older slots first"), its RecoveryIntent.Relaunch marker and readiness capability are dropped; no launch takes native admission (superseding #597's Couch-only skip); the lock refusal names the holder as Couch names a slot and counts as the same "in use" refusal, not a new one. Accepted transitional gap: pre-#595 wrappers hold no lock until relaunched. The durable plan's gate rows are marked dropped in its Revisions.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration marked stale by estimate-source, so provisional.

- Issue/spec: base design 1.0, thorough-spec x0.2; implementation 0.2 x0.4.
- Lock package: greenfield base design 1.5, existing x/sys/unix flock and file IO halve design, thorough-spec x0.2; base implementation 0.6 x0.4. No novel native integration remains.
- Three smaller modules: binding notification, owned-group cleanup, capability check; each base design 0.3 x0.2 and implementation 0.5 x0.4, using existing watcher, PTY and inventory seams.
- Cross-cutting deletion and stop-set move: base design 0.5 x0.2, implementation 0.5 x0.4; same repository only.
- Atlas/docs: base design 0.1 x0.2, implementation 0.15 x0.4.
- Single mandatory boundary review: design 0.1 x0.2, implementation 0.5 x0.4. Familiar codebase x1.0, thorough plan buffer +15% design. Required suite/review execution fits these integration/review primitives; operator maintenance timing is external and remains separately reported.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.15 impl=0.24
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.15
```

### 2026-10-16 — re-plan against landed #597/#600/#605/#607 (pair:2, Claude; sent to TL d0e96fbc, awaiting OK)

Keep the wrapper-held per-conversation flock, owned-group cleanup, watcher binding pipe, all scan/reservation/admission deletions and the moved stop-set helpers. Drop the capability cutover gate ("relaunch older slots first") and the RecoveryIntent.Relaunch marker: on Couch paths they would be a fourth refusal against #597's design, and for standalone Pair they block every resume until all slots relaunch. The accepted transitional gap: pre-#595 wrappers hold no lock until relaunched. Launcher reserves no admission for anyone; #597's couchHosted bypass and ResumeRequired/Recovery skip go with it. Open question to the TL: a Couch resume whose lock is held by another live wrapper refuses at native start as the same "in use" refusal. Merge main (14 conflicts) before building. The intermittent TestOwnedGroupStopsDescendantsBeforeReap EPERM under -race stays open until measured.

### 2026-10-16 — close review round 1 (REWORK) dispositions

- BR-1 (Critical), accepted: issue Spec/Done when now state the approved re-plan (Revision above); the plan's gate rows are marked dropped with a Revision; atlas Owners no longer names the deleted source.
- BR-2 (Important), accepted: a binding notice after stop began, or with an ID that cannot name a lock, is logged and dropped (`errNoticeIgnored`), so a SIGUSR2 restart still re-execs. TestRestartThenLateBindingNoticeStillRestarts RED without the fix ("discovery after stop").
- BR-3 (Important), accepted: owned-group cleanup polls during the grace and ends once no live member remains (darwin EPERM = only zombies, ESRCH = empty). TestOwnedGroupFinishesEarlyWhenTheGroupExits RED without (5.01s of a 5s grace).
- BR-4 (Important), accepted: the holder label resolves lazily on the first lock, once, behind an injected function (TestHolderLabelResolvesOnlyOnFirstLock); non-native wraps never resolve it.
- BR-5 (Important), accepted: wrapper-level tests drive notice handling and controlled signals through `handleBindingNotice` and `stopForSignal` (late collision stops the owned group and names the holder; unusable notice leaves the agent running; restart then late notice still restarts).
- Minor, accepted: an invalid holder label falls back to "unlabelled" instead of refusing the lock.
- Minor, deferred as follow-ups for the TL: the watcher-start-failure notice wording, and one shared rule for the three ID/label validators.

### 2026-10-16 — close review round 2 (REWORK) dispositions

- BR-3 (Important), accepted as a rule: "does the owned group still have a running member" is now per platform. Darwin keeps the kill probe (EPERM = only zombies); Linux reads /proc for non-zombie members of the group, because kill(-pgid, 0) succeeds there on our unreaped zombie leader; other platforms stay conservative. The stat parser is pure and tested (TestProcStatLiveInGroup).
- Minor (external probe on startup/I/O paths), accepted as a rule: locks are taken immediately with the fallback label (Pair tag, else "standalone"); the slot lookup runs once in the background from wrapper start and relabels held locks in place (`Handle.Relabel`, TestSetHolderRelabelsHeldLocks). Nothing waits for it.
- Minor (plan still describes the gate), accepted: Architecture line, rollout section, test row and checklist now state the approved design; the superseded design stays in Revisions.

### 2026-10-16 — close round 3 FIX-THEN-SHIP fixes

- Relabel truncates before writing: a contender reading in between sees an empty label (shown as the generic holder), never a torn mix.
- Discovery-unavailable notice only for a supported native agent whose conversation is still unbound, and now also when the watcher cannot be made or the executable is unknown.
- Plan: the remaining stale gate lines (deletion list, recovery test row, estimate, landing item) state the approved design.
- Deferred to the TL as a follow-up: one shared rule for NativeLockFilename, validHolder and validBindingNotice.

### 2026-10-16 — CI conformance failure at landing (PR #369)

couch-zellij-conformance failed on TestNativeConsoleWrapperZellij/wrapped-zellij: "remove private native fixture: directory not empty"; main passes it. Reproduced locally (make -f Makefile.local test-native-terminal-ci). Measured: the leftover is notify/lock, recreated by the wrapper's notification broker while the test already removed the fixture; at cleanup start the wrapper (reparented to PID 1 after its zellij server exited) was still alive and exited ~60 ms later. #595's owned-group cleanup lengthened the wrapper's exit by tens of ms, exposing a teardown race: the test joined the zellij server, never the wrapper it launched. Fix in the test: join the wrapper by its pidfile (bounded 5 s) before removing the fixture. 3/3 local CI-target runs green.
