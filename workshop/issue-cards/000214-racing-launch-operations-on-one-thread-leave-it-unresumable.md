---
id: 000214
status: codecomplete
created: 2026-09-08
updated: 2026-10-07
estimate_hours:
github_issue:
started: 2026-10-06T14:45:49-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 2.01
tracker:
    version: 1
    completion:
        token: close-9101551ff9ec
        repository: github.com/xianxu/pair
        reviewed_head: 7cd33d907cde6d81944994aeea2e17a5c26ae3b9
        evidence_commit: 4c72f7429b582683c10f16df084ae758a284e9c2
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
