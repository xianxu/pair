---
id: 000228
status: codecomplete
deps: []
github_issue:
target: workbench-latency
created: 2026-09-10
updated: 2026-09-11
estimate_hours: 1.98
started: 2026-09-10T21:58:26-07:00
actual_hours: 1.26
---

# A reattach asks every pair session for its clients

## Problem

**Proving that ONE thread is detached asks EVERY live pair session for its
clients, and a reattach does that five times.** Measured for `#206`:
`cmd/probes/reattachcost`, with full numbers and co-tenancy in #206's Log
(2026-09-10).

- `zellij attach` itself takes about 55 ms, and 8 at once barely cost more.
- A couch-shaped reattach (the snapshots couch actually takes, then the
  attach) takes **5.3–5.7 s per thread** on this host, with 8 live pair
  sessions of which 5 were detached. That is about 5.5 s for every manual
  reattach today.
- The call that costs is `zellij --session X action list-clients` against a
  *real* detached pair session: about 190–350 ms, and one took 971 ms. The
  same call takes about 38 ms against an attached session and about 53 ms
  against a fresh empty one, and `query-tab-names` against the same real
  detached session takes about 55 ms.

`launcher.ZellijSource.SnapshotContext` runs 2 `list-sessions` calls plus one
`list-clients` for every non-exited pair session on the host (its own comment:
"N is every pair session, not just the ones the caller cares about"). One couch
reattach takes 5 of those snapshots:

| # | site | what the caller actually needs |
|---|---|---|
| 1 | couchcore `ResumeContext` → `DetachedSessions` | is THIS thread's session live with zero clients? |
| 2 | couchcore `confirmStillDetached` → `DetachedSessions` | the same, re-proved before the child effect |
| 3 | launcher `createflow.go` loop top → `rt.Sessions()` (orphan-nvim sweep) | which sessions are live (not exited) |
| 4 | launcher `runOnce` → `rt.Sessions()` | what DecideLaunch reads (to be established in planning) |
| 5 | couchtty inventory refresh on completion | the whole inventory (every session, legitimately) |

Sites 1–4 pay O(all sessions) `list-clients` calls for a question about one
session, or for a question that needs no client count at all. A startup pass
over N detached threads (`#206`) is therefore O(N × S) of the expensive call:
about 80 s for 10 threads, and about 13 s for the first one.

This is `workbench-latency`'s defect shape: work per interaction scales with
what exists, not with what changed.

## Spec

The invariant, stated before the mechanism: **the client-count work a reattach
does scales with the threads it touches, not with the sessions that exist.**
Each snapshot site asks only what its decision needs:

- **Sites 1–2 (the detached proof)** ask only about the candidate thread's own
  session. The proof keeps its meaning: the session exists, is not exited, and
  has zero clients. It is not weakened to save a call.
- **Site 3** needs liveness only. If `list-sessions` answers that, it needs no
  client counts at all.
- **Site 4** is decided in planning, after reading what `DecideLaunch` and its
  helpers read for a forced-tag attach. It is not assumed.
- **Site 5** is out of scope. The inventory really does describe every
  session, and its refresh schedule already coalesces.

Planning decides the mechanism, for example a targeted snapshot over named
sessions beside the full one. It must keep one parser and one call pattern
(`ARCH-DRY`), not a second hand-rolled zellij client.

## Done when

- A couch reattach of one thread makes a number of `list-clients` calls
  independent of the number of live pair sessions. This is asserted as a count
  at the `ZellijSource` seam (a counting fake or shim), not as a timing.
- The detached proof's semantics are unchanged. Existing proof tests pass
  unmodified, and the targeted form has tests for exists / exited / attached /
  detached.
- Every snapshot site in the table above is either narrowed or has its full
  scan justified in the Log.
- `cmd/probes/reattachcost` ships, with a `make test-reattach-cost` target, and
  is re-run after the change with co-tenancy recorded. The couch-shaped phase's
  per-thread time is compared with #206's baseline.
- `#206` is unblocked.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec           design=0.25 impl=0.05
item: milestone-review     design=0.00 impl=0.20
item: milestone-review     design=0.00 impl=0.10
item: smaller-go-module    design=0.05 impl=0.16
item: smaller-go-module    design=0.05 impl=0.12
item: smaller-go-module    design=0.05 impl=0.16
item: smaller-go-module    design=0.05 impl=0.20
item: smaller-go-module    design=0.05 impl=0.12
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 1.98
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The rows, in order:

1. The in-window design already spent: the site analysis, the plan, and two
   plan-gate rounds. Anchored on the measured 0.39h at change-code (#183's
   judge asked for this row).
2. Plan review round 1 (it found the missed snapshot).
3. Plan review round 2.
4. Plan Task 1 (filtered snapshot, `SessionLive`, shared stub).
5. Task 2 (`launchShape`).
6. Task 3 (launcher liveness and the name rule).
7. Task 4 (couchcore narrowing plus the sandboxed end-to-end count test).
8. Task 5 (probe before/after, make target, guards).
9. Docs.
10. The close review.

The design buffer is +15% because a thorough plan doc exists. Frequency, per
`#201`: this runs once per reattach, which today is a gesture and in #206
becomes a startup pass of N. That is why the promise is a count.
(`sdlc estimate-source` reports the calibration doc `[stale]`, #127.)

## Plan

- [x] Baseline: done (#206 Log, 2026-09-10).
- [x] Establish what each of sites 1–4 reads from its snapshot; record it.
- [x] Design the narrowing: `workshop/plans/000228-a-reattach-asks-every-pair-session-for-its-clients-plan.md`.
- [x] Implement with the count-based tests.
- [x] Re-measure with the probe; record before/after with co-tenancy.

## Log

- 2026-09-11: closed — Operator real-stack smoke after make install + couch restart: detach then reattach of a thread is "way much faster". Counted at the seam: warm ResumeContext = 2 list-clients + 6 list-sessions at S=3 and S=22; forced-tag pair resume = 0 full snapshots, 2 liveness, 0 name probes. Mutation sweep 12/12 killed as named (apply-asserted, named failures only, tree identical to pre-sweep snapshot; incl. two sandbox-guard rows). Probe before/after same co-tenancy (N=8): old pattern 7.7-8.7 s/thread, new 272-282 ms/thread, zellij action p95 24 vs 25 ms. Unsandboxed make test exit 0, 197 ok on 0979584e.; review verdict: FIX-THEN-SHIP

### 2026-09-10

Split out of `#206` at the operator's direction, after #206's Plan step 1
measurement showed the reattach cost is couch's and pair's proof snapshots, not
`zellij attach`. `#206` depends on this and is blocked on it.

Claimed. What each snapshot site reads, from the code, which is Plan step 2:

- **Sites 1–2.** `DetachedSessions` resolves the candidates' session names
  from the index *before* it snapshots (`artifactcollision.go`), and
  `ProjectDetachedSessions` reads state only for binding names. So asking only
  those names gives the same answer.
- **Site 3.** `liveTagsForSweep` never reads `Session.State`; names are enough.
- **Site 4.** `DecideLaunch`'s forced-tag branch reads only
  `sessionBlocksReuse`, which is attached-or-detached, meaning not exited.
  `hasDetached` and `pick.go` read attach state, and they are reached only by
  bare `pair`, never by `pair resume <tag>`. Every other state read in the
  launcher is exited-versus-not (`legacy_live.go`, `session_index.go`), apart
  from `list.go`, which is the separate `pair list`.
- **The zellij CLI has no cheaper client probe:** `list-sessions` in 0.45.1
  carries no client information.

The design is in the durable plan:
- one filtered snapshot routine, with `SnapshotSessionsContext` and
  `LivenessContext`;
- a new `SessionLive` state;
- `decisionNeedsAttachState`, owned by `DecideLaunch`;
- the checker gains a `Zellij` field;
- counted invariants at the `ZellijSource.Path` stub seam.

After the change, a reattach makes `list-clients` twice, independent of the
session count.

### 2026-09-10: implemented (`8dfaee41` … `0979584e`)

**The seven sites, with what each reads and its cost before and after** (the
Done-when's table, recounted by the plan review):

| # | site | reads | before | after |
|---|---|---|---|---|
| 1 | couchcore `ResumeContext` → `DetachedSessions` | this thread's session: live and zero clients | 2 ls + S lc | 2 ls + 1 lc |
| 2 | couchcore `confirmStillDetached` | the same, re-proved | 2 ls + S lc | 2 ls + 1 lc |
| 3 | couchcore `awaitResumeRegistration` → `PairSession` | not exited | 2 ls + S lc per poll | 2 ls per poll |
| 4 | launcher orphan-nvim sweep | names only | 2 ls + S lc | 2 ls |
| 5 | launcher `runOnce`, forced tag | exited-versus-not | 2 ls + S lc | 2 ls |
| 6 | launcher name-acceptance probe | "is this length accepted" | 1 lc | 0 when the name is live |
| 7 | couchtty inventory refresh (async) | the whole candidate set | 2 ls + S lc | 2 ls + C lc |

`pair list`'s `ListSessions` stays a full scan, justified because each row
renders "attached (N clients)". It is commented as the one justified full scan
among the three `list-clients` producers.

**Counted at the seam.** A warm `ResumeContext` through the real checker makes
**2 `list-clients` + 6 `list-sessions` at S=3 and at S=22**
(`TestWarmResumeAsksTwoSessionsForClientsWhateverTheHostHas`). A forced-tag
`pair resume` takes 0 full snapshots, 2 liveness snapshots and 0 name probes.

**Mutation sweep.** Script `mutate228.py`: apply-asserted, named failures only,
tree identical to a pre-sweep snapshot. **12 of 12 killed as named**, including
two sandbox guards:
- a tested path asking zellij `kill-session` fails the stub-log assertion;
- dropping the PATH shim fails the new `LookPath` assertion. That is the guard
  for the guard: with correct code, no route execs a bare zellij, so a dropped
  shim was otherwise unobservable.

On its first run, row 12 failed on a compile error (an unused import), and the
sweep refused to count that as a kill, as designed.

**Probe, before and after under one co-tenancy.** `PAIR_PROBE_N=8 make
test-reattach-cost`: load 1.98, 8 agent processes, 26 zellij sessions listed,
12 cores. S=17 in the old pattern: host 8 live plus probe 9.

| phase | per thread | pass of 8 | `zellij action` p95 |
|---|---|---|---|
| raw attach | 54–64 ms | 475 ms | 22 ms |
| **old pattern** (5 full snapshots + name probe + attach) | **7.7–8.7 s** | 66 s | 24 ms |
| **new pattern** (2 targeted + 3 liveness + attach) | **272–282 ms** | 2.2 s | 25 ms |

Caveat, recorded with the numbers: the probe's own sessions answer
`list-clients` in about 53 ms, where real detached pair sessions take about
250 ms.
- **Old column:** it understates the real before-cost.
- **New column, on real sessions:** add about 2 × 200 ms, so roughly 0.7 s per
  reattach.
- **The promise is the count, not either timing.**

**Suite.** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`, unsandboxed: exit 0,
197 packages `ok`. The whole existing suite passes unmodified.

### 2026-09-11: close review, fixed before the close commit

**Verdict FIX-THEN-SHIP.** The review's findings block was malformed, so the
gate recorded 0 findings and converged. The prose findings bind anyway; each
is dispositioned here.

**Important (`ARCH-PURPOSE`, `ARCH-ORDER`): the `SessionLive` guard reached one
reader of attach state, not the class.** `DecideLaunch` refused a snapshot
that never asked; couch's `ProjectDetachedSessions` did not. Fed a liveness
snapshot, it would read every thread "not detached", and couch would present
that as a proof. The review named two readers. Enumerating every comparison
against `SessionDetached` and `SessionAttached` found a third: the picker,
guarded only because its one caller runs after `DecideLaunch`.

- **One rule.** `launcher.RequireAttachState(sessions)` refuses any
  `SessionLive` row, and all three readers apply it first:
  - `DecideLaunch`'s bare branch (its inline loop is gone);
  - the picker's entry, `resolvePickWithPolicy`, which aborts with exit 1 and
    names the missing attach state;
  - `ProjectDetachedSessions`, which now returns an error that
    `DetachedSessions` propagates.
- **One red-first test per reader.** `TestDecideLaunchRefusesAttachStateItWasNotGiven`,
  `TestThePickerRefusesAttachStateItWasNotGiven`,
  `TestProjectDetachedSessionsRefusesAttachStateItWasNotGiven`.
- **Mutations, 5 of 5 killed as named.** The rule returning nil fails the
  launcher and couchcore tests. Each call site handed `nil` instead of its
  sessions fails that site's own test.

**Minors fixed.**
- `atlas/architecture.md`: the three-forms line named `pair list` as a user of
  the full form. It is a separate scan (`OSRuntime.ListSessions`). The line now
  names every full-form caller, including `pair rename`'s gate and the create
  flow's re-check, and names the `RequireAttachState` rule.
- `liveTagsForSweep`'s doc comment now says it reads names only. Before, only
  its caller said so.
- `ProbeSessionName` points at the short-circuit that skips it for a live name.
- **Live conformance.** `TestSessionDetachLive` (`PAIR_LIVE_COUCH=1`) now
  checks the named and liveness forms against real zellij 0.45.1, at both the
  attached and the detached stage. It passes. Two mutations each fail it: the
  named form keeping every session, and the liveness form asking every session.

**Not taken, with reasons.**
- **`LivenessContext` runs a redundant `list-sessions --short`.** The reviewer
  says `--no-formatting` carries the names too, so every form could run one
  `list-sessions`, not two. Unverified here. It changes the parser's input for
  all three forms, and it is outside this issue's promise, which is a
  `list-clients` count. The obvious place for it is #229, which measures the
  path that pays it.
- **The duplicate-row claim cannot be falsified with `StubZellij`**, which takes
  a map and so cannot emit two rows under one name. The claim holds by
  inspection of `snapshot`'s per-line filter. Testing it needs a raw-lines stub.
- **`SnapshotSessionsContext` with no names** still runs two `list-sessions`.
  That is unreachable today, since `DetachedSessions` returns before calling it.
- **The probe derives its marker from the session-name shape.** That is
  probe-only code.

**Architectural notes for later work.**
- **`SessionState` now mixes two kinds of fact.** `attached`, `detached` and
  `exited` are observations; `live` means the observation was not made.
  `RequireAttachState` is a runtime stop-gap. The structural fix is two types,
  a liveness row and a classified row, so passing a liveness snapshot to a
  reader of attach state fails to compile. Worth doing before a fourth reader
  or a fourth form appears.
- **#191 is stale.** Its motivation was couch's blocking startup inventory
  paying a full `list-clients` fan-out, which this issue removed. Its real
  remainder is bare `pair`'s picker and `pair list`. A note is in its Log.
- **`launcher.Run` (`run.go`) is reachable only from tests.** It calls
  `DecideLaunch` outside `runOnce`'s snapshot choice. It belongs with #192.

**Suite after the fixes.** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`,
unsandboxed: exit 0, 197 packages `ok`.

## Revisions

### 2026-09-10 — the site table was two short (plan review round 1)

**Reason.** A fresh-context plan review applied the plan in a scratch clone and
counted the reattach path end to end. It found two sites this issue's table
missed:

- **A sixth full snapshot on the critical path.** Couch's
  `awaitResumeRegistration` polls `PairSession`, which reads only "not exited".
  The reviewer measured 22 `list-clients` calls with 22 live sessions.
- **A standalone `list-clients`.** The launcher's session-name acceptance probe
  (`ProbeSessionName`) runs one on the thread's own session.

**Delta.** The authoritative table is now the seven-site one in the plan
(`workshop/plans/000228-…-plan.md`, "Sites after the change"). Done-when
bullet 3 ("every snapshot site in the table above") refers to that table. The
invariant is unchanged; the enumeration was incomplete. That is why the plan
adds an end-to-end count test over the whole couchcore path, not just
per-function tests.
