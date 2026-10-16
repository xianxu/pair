---
id: 000595
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 2.15
card_mirror: '320b7af3c737be1c9a5638b1e06bcd689f040f4b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T19:45:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

One maintenance cutover, one lock build: delete the global native owner scan, legacy/unbound fallbacks and durable admission reservation store in the same change. Ordinary disk Resume refuses while the current live-wrapper registry/readiness inventory contains any wrapper without the lock capability: `relaunch older slots first: X, Y`. Startup, Continue and Fresh remain available. The existing confirmed Relaunch of the exact live old slot is the migration action, still taking the new flock after source retirement; no new override or dual-guard build. Preserve/move the narrow confirmed stop-set attribution helpers.

## Done when

- Real kernel and wrapper tests show one winner for two wrappers on one ID, a diagnostic holder label, no losing-writer metadata damage, and lock release on wrapper exit including SIGKILL.
- Controlled normal/error/signal/restart exits stop only the owned agent group before reap/lock release; tests explicitly document the accepted SIGKILL orphan gap rather than promising inherited-agent protection.
- Current-launch unknown-to-known IDs acquire through existing discovery; stale/duplicate events and a late collision are handled without a new native integration or historical scan.
- Legacy capability tests show startup/Continue/Fresh unaffected, ordinary disk Resume names sorted live old slots, confirmed Relaunch migrates them, and Resume works once they are gone. Dead/stale readiness alone is not a blocker.
- The forensic and reservation code is gone, shared consumers and stop-set safety remain, and close reports actual net production/test LOC deletion. Build, required recorded suite and SHIP review pass.
- Operator live maintenance acceptance: restart over old live slots, migrate them once at a quiet time, resume a parked scratch conversation, and refuse a held scratch conversation with its slot named. This remains pending until performed; any landing before it requires the TL's explicit direction.

## Plan

Durable implementation plan: workshop/plans/000595-conversation-flock-plan.md (one atomic close boundary).

- [x] Obtain TL approval of the wrapper-lifetime/capability-cutover revision before change-code.
- [ ] Implement and verify the small lock, existing binding notification and owned-group cleanup; remove forensic/reservation authority in the same build, preserving moved source stop-set attribution.
- [ ] Run full verification and SHIP review; coordinate the single operator maintenance acceptance; report actual net deletion and land as directed.

## Log

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

## Revisions

### 2026-10-15 — measured constraints before implementation

The original strict orphan-child and late Fresh-ID requirements remain in force. This draft does not silently replace them with a wrapper-only promise. The TL/operator must choose a proven native integration or explicitly revise the guarantee before change-code. The proposed temporary migration build and final deletion are separate acceptance boundaries; final code retains one mechanism and no legacy fallback. Stable inode residue and exact cleanup timing are named in the draft for approval.

### 2026-10-15 19:58 PDT — TL scope decision replaces the original promise

TL cb293d05 explicitly narrows inherited-child/orphan protection to wrapper lifetime and chooses a maintenance cutover. This supersedes the initial fd-to-native guarantee, orphan-child-keeps-lock acceptance, temporary override and proposed dual-guard milestones. Controlled exits kill the wrapper-owned group; SIGKILL/orphan recovery is #583/#587. The active Spec, Done when and durable plan now agree. Resume-only legacy capability gating leaves the existing confirmed Relaunch available to perform migration, avoiding a circular all-slots refusal; there is no new override flag. No code changed.

Measured deletion baseline: 1,777 production + 2,766 test lines in dedicated old blocks, less 114 production lines moved/retained for stop-set safety, gives 1,663 replaceable production + 2,766 test lines. Allow 350–650 new production and 500–900 test lines: approximately 1,000–1,300 net production lines and 2,900–3,600 production-plus-test lines deleted, before extra bridge removals. This is a planning range, not a final diff claim. The revised plan is committed and sent before code.

### 2026-10-15 20:08 PDT — approval and plan-quality clarification

TL approved 2833b837 via submitted receipt 659dde9c, including confirmed Relaunch as the migration path. change-code full-flow plan review requested named transition/test seams and explicit discovery-channel failure semantics (PQ-1/2/3); these are now documented without expanding the wrapper-knows-ID promise. Unknown discovery loss is visible and retains unknown status; known-ID contention/IO failure stops the owned group. No implementation before the gate passes.

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
