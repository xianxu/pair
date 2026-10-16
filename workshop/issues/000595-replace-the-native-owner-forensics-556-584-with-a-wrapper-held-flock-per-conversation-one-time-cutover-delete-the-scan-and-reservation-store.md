---
id: 000595
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '5c10e8776f82ba6f238e5611dd2e084dfc5dff8b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T19:45:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

- **One mechanism:** each pair wrapper, before exec'ing a native agent on conversation ID X, takes
  `flock(LOCK_EX|LOCK_NB)` on `<cache>/native-locks/<agent>-<X>.lock` and passes the fd to the agent process (not
  close-on-exec), so the lock lives exactly as long as any holder. A fresh conversation takes its lock when its ID is
  first known.
- **Admission** (resume from disk, relaunch, Codex resume): one non-blocking lock attempt. Free → proceed; held → refuse
  with "in use by <slot>" (the lock file records the holder's slot for the message only, never for the decision) and
  offer Fresh. Two outcomes.
- **One-time cutover** for wrappers started before the lock: they hold no lock. The cutover is an explicit,
  operator-confirmed relaunch of each old slot (an override flag for that relaunch only, logged), after which every live
  writer holds a lock. No permanent legacy code path.
- **Delete** the forensic scan (`recovery_native_*`, process-table/argv/ledger/lsof ownership proof), the reservation
  store (`native_admission*`), and #584's legacy and unbound fallbacks, with their tests. Keep only what other
  features still need (check each deletion's callers).
- Agents started outside pair are not covered (they aren't today either); document it.

## Done when

- Tests: two wrappers on one ID → second refused with the holder named; holder exits (including SIGKILL) → lock free;
  a forked/orphaned agent child keeps the lock; the cutover override relaunch works once and is logged.
- The forensic and reservation code is gone (net negative LOC, stated in the close).
- Live: resume-from-disk of a parked slot works on this machine; resuming a conversation held by a live slot is refused
  with that slot named.

## Plan

- [ ] Resolve measured fd inheritance and fresh unknown-ID requirements with the TL/operator; approve the durable plan before code.
- [ ] M1 — introduce the lock/native handoff and temporary confirmed cutover; separate review and operator acceptance.
- [ ] M2 — after migration, delete forensic/reservation authority and temporary override; final conformance, live acceptance and SHIP.

## Log

### 2026-10-15

- Claimed after #594 landed (47a485ce46ffb685ce951cfb16428fef2c2f1bcd); start-plan prepares a fresh main issue branch. No stacked development or implementation.
- Read-only caller inventory found that source stop-set attribution still consumes recoveryNativeWrappers/recoveryWrapperReady, so those narrow duties must be moved rather than blindly deleted. Other shared JSON/ledger and native inventory consumers must remain.
- Scratch Darwin flock measurements passed: second open including same-process reopen contends errno 35; dup and inherited child retain the lock after original close/wrapper SIGKILL; last-holder SIGKILL and reap frees it. Full scripts/results: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-595-flock.XreBFC/.
- Exact installed Codex shim spawn shape (stdio inherit) drops fd3 in its fake native child. After Node SIGKILL the fake engine remained alive while a contender acquired: a measured contradiction of merely passing ExtraFiles to the shim. Also, Codex/Agy/Muse Fresh IDs arrive after exec via the detached watcher; a newly acquired descriptor cannot retroactively be inherited. No real coding agent/slot was launched or signalled. Reported early to the hub (f0ae6753); durable draft workshop/plans/000595-conversation-flock-plan.md records the choices and a proposed two-boundary migration. PLAN ONLY pending decision.

## Revisions

### 2026-10-15 — measured constraints before implementation

The original strict orphan-child and late Fresh-ID requirements remain in force. This draft does not silently replace them with a wrapper-only promise. The TL/operator must choose a proven native integration or explicitly revise the guarantee before change-code. The proposed temporary migration build and final deletion are separate acceptance boundaries; final code retains one mechanism and no legacy fallback. Stable inode residue and exact cleanup timing are named in the draft for approval.
