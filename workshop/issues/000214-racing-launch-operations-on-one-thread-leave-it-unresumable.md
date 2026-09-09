---
id: 000214
status: open
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
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

## Done when

- Two launch-producing operations cannot be admitted against one thread; the second is **refused**
  (not queued) with a message naming what is already running on that thread.
- The refusal is per (thread, operation class); two *different* threads are unaffected, so the guard
  does not serialise the fleet.
- A refused second gesture leaves the thread **resumable** — asserted by a test that fires
  resume-then-relaunch and then resumes successfully.
- A test reproduces the three-launch history and asserts the projector's reason distinguishes
  multiple bindings from none.
- The existing thread is either repaired in place or archived with its transcript pointer recorded;
  the decision is stated with its reason.
- `#184`'s `switch-agent` inherits the guard rather than adding a third racer.

## Plan

- [ ] Decide the layer(s) per Spec; read `CommitStartClaim` / `advanceSuccessfulStart` first.
- [ ] Key the queue by thread + operation class; refuse with a naming message.
- [ ] Enforce single-incarnation at the store CAS.
- [ ] Split the `binding lost` reason into none-versus-several.
- [ ] Test: resume-then-relaunch admits one, thread stays resumable; three-launch history projects
      the multiplicity reason.
- [ ] Resolve the stuck thread; record which and why.

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
