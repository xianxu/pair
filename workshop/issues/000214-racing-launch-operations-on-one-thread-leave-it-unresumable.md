---
id: 000214
status: working
deps: []
github_issue:
created: 2026-09-08
updated: 2026-10-06
estimate_hours:
card_mirror: 'bb67527b27e87913062af2b669ebd1fe9791a39f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T14:45:49-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "93b62408", done: "ed20fee6"}
---

# racing launch operations on one thread leave it unresumable

## Problem

The operator pressed **resume-from-parked while a relaunch was already running**. The thread is now
`unusable: binding lost — repairable` and cannot be resumed from couch.

Reproduced from the artifacts, not inferred. Thread
`e108517d46ab4575/couch-a1e4585a90fbd35c` (`pair`, `layout3`, record revision 126):

| time | event |
|---|---|
| 07:50:51.409 | park recorded — `park-a641baffabc79a6a`, pid 60832 |
| 07:50:52.553 | launch #1 → `binding`, **launch_ordinal 26**, `root_native_id 9a99ff57-…` |
| 07:51:05.611 | launch #2 → `binding`, **launch_ordinal 29**, same `root_native_id` |
| 07:51:24.435 | launch #3 → `launch` recorded, **no binding ever follows** |

Three launches in 32 seconds against one parked thread, each re-binding the *same* native session.
Launch ordinals jumped 26 → 29, so 27 and 28 were consumed elsewhere in the race. The record moved
from revision 117 (the park closing) to 126.

**Nothing crashed and nothing is corrupt — the history became ambiguous.**
`ParkedResumeObservation`'s contract is that the projector accepts it *"only when it is the sole
observation for the address"*, and `parkedResumeProofMatches`
(`couchcore/actionableinventory.go:370-376`) returns false on `len(observations) != 1`. After two
bindings and a dangling third launch for one tag+agent there is no unique answer to "which
incarnation is this thread", so the resume proof cannot be re-derived and the row projects as
`ReasonBindingLost`.

Note what this means: **every one of the three launches succeeded.** This is not an error path. Valid
operations, correctly executed, composed into an unusable state.

### Why nothing stopped it

`couchtty/console.go:1509`:

```go
key := fmt.Sprintf("menu\x00%d\x00%s", effect.Attempt, effect.Operation)
```

The operation-queue dedup key is **attempt + operation name**. It contains **no thread address**.

Two consequences, and the second is the defect here:

1. The same operation cannot be double-submitted — which is what the key was built for.
2. **Different operations on the same thread are different keys and both admit.** `relaunch` and
   `resume` are distinct names, so the queue accepts both against one thread and nothing downstream
   objects.

So there is no "one operation in flight per thread" invariant. What has been *providing* that
property in practice is the single `operationQueue.Run` worker (`console.go:538`) serialising
everything — but serialising two launch operations does not make them safe, it only stops them
overlapping in time. Here they ran in sequence and still produced three launches.

**The blast radius is wider than resume+relaunch.** Any two launch-producing operations on one
thread compose the same way — `resume`, `relaunch`, and the `switch-agent` that `#184` is adding.
`#184` makes a third one, which is a reason to fix this first.

## Spec

> **Scope revised 2026-10-06** — the concurrency half (per-thread admission
> guard, refusal message) moved to `#205`. Current scope is
> "Scope after 2026-10-06" below; the text between here and there is the
> original record, kept for its reasoning. See Revisions.

**Uniqueness of the live incarnation must be enforced where it is required, not assumed.**

Two candidate layers; the plan picks after reading how `advanceSuccessfulStart` and
`CommitStartClaim` already interact, since one of them may already be the right home:

1. **Admit one launch per thread** — key the operation queue by `(thread address, operation class)`
   rather than by operation name, where `resume`/`relaunch`/`switch-agent` share a class. Refuses the
   second gesture at the point the operator makes it, which is where a refusal is comprehensible.
2. **Refuse the second launch at the store** — `CommitStartClaim` already writes under a revision
   CAS and its comment states the atomicity requirement. A launch that finds the thread already
   launched at a newer revision should fail rather than append a second binding.

Prefer both: (1) is the good error message, (2) is the invariant. (1) alone leaves the race open to
any future caller that does not go through the menu.

**And the diagnosis must survive.** `ReasonBindingLost` currently says "repairable" without saying
what to repair. Given the cause is *multiplicity*, the row should distinguish "no binding" from
"several bindings", because the second is recoverable by choosing the newest and the first is not.

### Recovery for the thread that is already in this state

Not lost. The conversation is intact at
`~/.claude/projects/-Users-xianxu-workspace-pair/9a99ff57-cb0f-4590-8ea7-d684f2ff5a3f.jsonl`
(last written 07:50, matching the park), with the parked scrollback at
`parked-scrollback-couch-a1e4585a90fbd35c-20260908T075050.raw`. `claude --resume 9a99ff57-…` in the
repo returns the session outside couch. **Do not hand-edit the threadstore** — it is CAS-versioned at
revision 126 and couch owns it.

Whether couch can repair the record in place — pick the newest binding, drop the dangling launch — is
the useful question this issue should answer, since the alternative is archiving a thread whose
agent transcript is perfectly fine.

### Operator decision, 2026-09-08 — scope the guard to one-at-a-time

**Exactly one resume-class operation may be in progress per thread; a later
request is REJECTED.** Not queued, not coalesced — refused, so the operator
learns immediately that the gesture did not take.

This deliberately forecloses the harder question. Concurrency *within a single
repo* is a real future direction (several threads at one path is already legal —
`atlas/couch.md`: *"two threads in one tree at different subdirectories remain
legal"*), and the guard may be relaxed then. **For now: one.** Choosing the
narrow rule now is what makes this small enough to land ahead of `#184`, which
would otherwise add `switch-agent` as a third racer.

So Spec option 1 is the chosen shape, and option 2 (store-level CAS refusal) is
the belt to its braces rather than an alternative:

- Key the operation queue by **(thread address, operation class)** where
  `resume` / `relaunch` / `switch-agent` share one class.
- The second gesture is refused with a message naming what is already running on
  that thread — a refusal the operator can act on beats a silent second launch.
- `CommitStartClaim`'s revision CAS stays the invariant of last resort, so a
  caller that does not go through the menu still cannot double-launch.

### Working-tree note for whoever takes this

`couchtty/console.go:1509` holds the dedup key and was **uncommitted and being
actively edited** by the `#199` session on 2026-09-08. Coordinate before editing
it; that file has already cost `#199` four defects in one day.

### Scope after 2026-10-06

The re-read (Log, 2026-10-06) found the race this issue was named for already
blocked inside one console: `dispatchMenuOperation` refuses a second operator
operation and `CommitStartClaim` refuses a second occupant. The incident's real
mechanism is **newest-launch shadowing**: `sessionledger` `CurrentLaunch` counts
only the newest launch ordinal, and a launch that never binds hides every
earlier binding. Two things remain, neither about concurrency:

1. **Name the binding failure.** `bindingResumeDiagnostic` (`couchcore/resume.go`)
   already tells ambiguous, unbound and provisional apart; the evidence pass
   drops the code and the projector returns a bare `ReasonBindingLost`
   (`actionableinventory.go:601`). Carry the code through so the row says which
   one, and what (if anything) repairs it.
2. **Launches couch never claimed.** The in-pane agent restart
   (`pair agent restart` → SIGUSR2 → `wrapcmd` `freshAgentInvocation`) appends a
   ledger `launch` row with no store claim. Until its agent binds, that launch
   shadows the thread's established binding — the most plausible source of the
   2026-09-08 launch #3. Fix it on couch's side, in how it reads the ledger
   (for example, an unbound newer launch whose process is gone does not shadow
   an earlier binding). Pair must keep working without couch, so the wrapper
   gains no couch call (layer direction).

The stuck 2026-09-08 thread is out of scope: a month on, it has been archived
or recovered by hand (`claude --resume 9a99ff57-…`).

## Done when

- A thread whose resume proof fails names the failure when the resolution
  proves it, with or without a park receipt (operator decision D2, applied to
  proven evidence on 2026-10-07; see Revisions):
  - ambiguous reads "two conversations claim it — reboot";
  - unbound, only with a proven-absent fresh file and nothing behind it, reads
    "no turn taken yet — reboot";
  - provisional, only on an incomplete storage listing, reads "conversation
    not confirmed — retry".

  An unproven refusal keeps `binding-lost` (with a receipt) or `session-gone`
  (`provenBindingRefusal`, table-tested). Every new reason is produced by a
  test shape and passes the defining-word guard. The slot actions and the
  start-reuse notice treat the four binding reasons as one class.
- The named reasons are produced through the real resolver contract
  (resolution together with a typed refusal), by test.
- A resolver IO failure projects `unusable/unknown`, never a binding verdict
  (test).
- After an in-pane fresh restart whose agent never took a turn, the thread
  resumes the conversation that restart replaced (operator decision D1). The
  owner query falls back to the previous established generation exactly when a
  complete listing proves the chosen file absent. A test reproduces the
  2026-09-08 ledger shape. Couch's proof and pair's resume launch agree
  because both read the same query, and pair's restart path is unchanged.
- The Pair wrapper makes no couch call (layer direction).

## Plan

Durable plan: `workshop/plans/000214-racing-launch-operations-on-one-thread-leave-it-unresumable-plan.md`.

- [x] `sessionledger.PreviousEstablished` (pure) and its table test.
- [x] The owner query falls back on `FreshRequired`, with the 2026-09-08 shape test.
- [x] Named binding-failure reasons, the IO-error fix, and the sweep over every
      reason switch.
- [x] Atlas, full verification, close.

## Log

### 2026-09-08

Operator report, diagnosed from the ledger and threadstore rather than reproduced live. The timeline
above is from `ledger-couch-a1e4585a90fbd35c.jsonl`; the record is
`couch/threadstore/records/e108517d46ab4575/couch-a1e4585a90fbd35c.json`.

**This corrects `#205`.** That issue's Spec argues a bounded worker pool is safe because
`operationQueue`'s per-key dedup already guarantees "no two operations in flight on one thread" and
so the property does not depend on the single worker. **The key does not contain the thread**, so
that guarantee does not exist — the single worker is the only thing serialising these today.
Parallelising the queue as `#205` specifies would make this race substantially easier to hit, so
`#205` should depend on this issue rather than the reverse.

### 2026-10-06: re-read against current code (claimed in pair:4)

Claimed so `#205` (now also covering the startup reattach pass) can land on it.
An exploration of current code, with the key claims re-read by hand, finds the
premise of this issue mostly stale:

- **Inside one console the operator cannot overlap two operations, and could not
  on 2026-09-08 either.** `dispatchMenuOperation` (`couchtty/menu.go:1785-1788`)
  silently drops any operator operation while `InFlight` is set. The guard is
  global, not per thread.
- **The store already refuses a second occupant.** `CommitStartClaim`
  (`couchcore/threadstore.go:564-592`) refuses when `Incarnations` is non-empty
  or a park is open, under the revision CAS. All four claim sites go through it.
  Spec option 2 largely exists.
- **The "multiplicity" diagnosis is wrong.** The evidence pass appends at most one
  `ParkedResumeObservation` per record (`actionableinventory.go:930-938`), so
  `len != 1` cannot fire on two bindings. The real rule is
  `sessionledger/record.go` `CurrentLaunch`: only the newest launch counts, so
  dangling launch #3 shadowed the earlier bindings (the "#168 shape"). The
  projector drops `bindingResumeDiagnostic`'s ambiguous/unbound/provisional code
  and returns a bare `ReasonBindingLost` (`:601`). That is where the split
  belongs.
- **Launch paths that bypass `InFlight`:** the reattach pass (warm-only, holds
  but never takes the slot), continuations, remote socket `resume|reboot`, and
  the in-pane agent restart (`pair agent restart` → SIGUSR2 → a ledger `launch`
  row with no store claim). The last one is the most plausible source of the
  unbound launch #3.
- **What `#205` actually needs from here:** the queue key still has no thread
  (`console.go:1661`), and the single worker (`console.go:629`) is what orders
  pass, remote, continuation and operator jobs on one thread. With a pool, only
  the store CAS separates them; relaunch's park→resume window would then lose to
  a concurrent resume (`ParkedNotResumed`), which fails safe but confuses. A
  per-(thread, launch-class) admission guard where all four enqueue paths meet
  is the real prerequisite.

Stale line refs: `console.go:1509`→`:1661`, `:538`→`:629`,
`actionableinventory.go:370-376`→`:668-674`.

### 2026-10-07: design reading (after #205 landed)

**Item 1, naming the failure:**
- The evidence pass drops `bindingResumeDiagnostic`'s code
  (`actionableinventory.go:965`), and `ThreadEvidence` has no field for it.
- `ClassifyThread` says `binding-lost` only when a park receipt exists
  (`:613-618`). Otherwise a binding failure is labelled `session-gone`
  (`:626`), which can be archived.
- **Separate bug:** a resolver IO error still marks the proof resolved (`:964`),
  and a zero resolution reads as "unbound". Relaunch already guards against
  this (`relaunch.go:123-130`); the evidence pass does not.
- Wording already exists in `bindingRefusalDiagnostic` (`resume.go:363`) and
  `ResumeRebootAdvice` (`resume_route.go:65`): unbound and ambiguous mean
  reboot, provisional means retry after a turn.
- Every table that grows with a new reason is listed by the exploration: the
  label, the defining-word test, the menu notice, `SwitchableState`, the slot
  resume offer and the lost-slot notice.

**Item 2, shadowing:**
- `sessionledger.CurrentLaunch` (`record.go:556`) keeps only the newest
  launch's bindings. Its writers (`store.go`) need that meaning, so the reader
  is the place to change.
- The in-pane restart (`wrap.go:2462`) records a chosen-id launch with **no
  process identity**, so "is that launch's process gone" cannot be decided from
  the ledger.
- The evidence that decides it: a complete listing that proves the chosen
  session file absent (`FreshRequired`). That means the fresh agent never took
  a turn.
- Couch reads the ledger only when no session is present and nothing is live
  (`:959`).
- This is the #168 shape (punted, "reopen if a new thread loses its binding
  this way"), and the 2026-09-08 incident qualifies.
- No test covers bindings followed by a newer unbound launch.

## Revisions

### 2026-10-06: concurrency half moved to `#205`

**Reason.** The re-read against current code (Log, 2026-10-06) showed the
named race already blocked within one console, and that the per-thread guard is
only needed once `#205` replaces the single queue worker with a pool. The
operator chose to move that half into `#205`, where removing the old ordering
and adding its replacement land together.

**Delta.**
- Moved to `#205`: the per-(thread, launch-class) admission guard, the refusal
  message naming what is running, and the "different threads unaffected" and
  "refused gesture leaves the thread resumable" Done-when items.
- Dropped: store-level CAS refusal (already exists in `CommitStartClaim`); the
  "several bindings" split (the mechanism was newest-launch shadowing, not
  multiplicity); repairing the 2026-09-08 thread.
- Kept and reframed: naming the binding failure; added the unclaimed in-pane
  launch as the likely root cause.
- `## Done when` and `## Plan` were rewritten for the new scope; the original
  Spec text is kept above the new subsection.
- The card title still describes the old scope; retitling waits for the
  operator.


### 2026-10-07: operator decisions D1 and D2; design settled

**Reason.** The design reading (Log) left two calls to the operator: what
resume should do after an unturned fresh restart, and how far the named
reasons should reach.

**Delta.**
- D1: resume the old conversation.
- D2: name binding failures with or without a park receipt.
- Done-when and Plan were rewritten for the settled design, which lives in the
  durable plan.

### 2026-10-07: D2 applied to what the evidence proves

**Reason.** In implementation, the full couch suites showed that
`ResumeBindingUnbound` is also the code for "no launch record at all". Mapping
it straight to `no-turn` relabeled ordinary ended sessions, which exceeds what
D2 asked for.

**Delta.** A named reason now needs proof from the resolution
(`provenBindingRefusal`):
- ambiguous: always proven;
- unbound: only with `FreshRequired`, meaning a complete listing proved the
  fresh file absent and no earlier conversation exists;
- provisional: only on an incomplete listing.

Anything unproven keeps `binding-lost` with a receipt and `session-gone`
without one. D2's intent (name what is known, with or without a receipt) is
unchanged. The slot actions and the start-reuse notice treat all four binding
reasons as one class (`IsBindingFailure`).


### 2026-10-07: close review REWORK (BR-1..BR-3) addressed

**Reason.** The full close review found three blocking problems:
- **BR-1:** the evidence pass dropped the resolution on a typed refusal, so
  `no-turn` and `unconfirmed` could never fire in production.
- **BR-2:** the named reasons were tested only from hand-built evidence.
- **BR-3:** "retry after a turn" did not match the incomplete-listing proof.

**Delta.**
- The refusal path reads the resolution, per the real contract.
- `TestNamedBindingReasonsAreProducedThroughTheResolver` produces every named
  reason through the resolver seam. It turns red under the BR-1 mutation.
- The `unconfirmed` label is now "conversation not confirmed — retry".
- The switcher notice uses `Label()` as its single source.
- The 2026-09-08 shape is tested at query level too.
