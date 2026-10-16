---
id: 000595
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '19bbc3c2ed479981ea27ab52720c19e3aa09b684' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
