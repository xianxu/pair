---
id: '000198'
status: done
started: 2026-09-06T12:06:08-07:00
created: 2026-09-06
updated: 2026-09-06
estimate_hours: 3.81
actual_hours: 2.23
---

# couch: let the operator choose layout3

## Problem

couch always launches `pair resume <tag> --layout2`. The operator wants
`--layout3` — pair's own right-hand user terminal — available under couch.

**The literal is one line** (`couchcore/launch_existing.go:50`), and the code
anticipated this request:

> The layout: pinned to layout2 by operator decision 2026-08-22. couch owns
> terminal switching now, so layout3's third pane — pair's own user terminal —
> is the layer couch replaces. Provisional ("for now"), which is why it is a
> literal here rather than a knob nobody has asked for.
> — `couchcore/couch.go:427`

So this is the sanctioned reversal of a provisional decision, not a fight with
the design. But the change is **not** one line, for four reasons the tree
already records.

**1. The pin is a decision with a rationale, recorded in five places.** The
literal at `launch_existing.go:50`, the rationale at `couch.go:427-431`, and
`atlas/couch.md` at `:201`, `:815`, `:880`, `:903-904` ("Layout stays pinned to
layout2: couch owns terminal switching, so layout3's third pane is the layer
couch replaces"). Editing the literal and leaving those is exactly the stale-map
failure — the atlas would teach a rule the binary no longer follows.

**2. The neighbouring path is destructive.** `launch_existing.go:46-49` (#179):

> `--layout2` is dropped [for a warm reattach] for the same reason: a running
> session already has its layout, and asking for a different one sends Pair down
> a conflict path that offers to **DELETE the live session** — destroying the
> agent this exists to preserve.

The existing `in.Warm` branch already sends no layout at all, so it is safe
today. Any change here must preserve that invariant exactly: the layout flag is
for a COLD start/resume boundary and must never reach a warm reattach. That is
what makes this a change to review rather than a literal to swap.

**3. Scope of the choice is a real fork, not a detail.** `StartArgs` is
persisted "so a revival reproduces the launch without the operator restating
it" (`couchcore/startargs.go`). So:

- **Per-thread** — layout belongs in `StartArgs`, and a revived thread comes
  back in the layout it was started with. Matches how agent and argv already
  work.
- **Namespace-global** — a `couch --layout3` flag applying to every thread it
  starts, held wherever couch keeps per-namespace preference.

These differ in observable behavior after a park/resume cycle, so the plan must
pick one deliberately. Per-thread is the shape the surrounding data already
has, and is the recommendation absent a reason otherwise.

**4. ~18 couch-side test references pin layout2**, several with comments
asserting the pin is intentional (`couch_test.go:1666`: "Layout pinned to
layout2 (operator decision 2026-08-22)"; `cmd/couch/main_test.go:154`: "want
**exactly** `pair resume <generated-couch-tag> --layout2`"):

| file | refs |
|---|---|
| `cmd/internal/couchcore/couch_test.go` | 5 |
| `cmd/internal/couchcmd/run_test.go` | 5 |
| `cmd/couch/main_test.go` | 3 |
| `cmd/internal/couchcore/runner_test.go` | 3 (generic argv fixtures, not pins) |
| `cmd/internal/couchcore/warmresume_test.go` | 1 (pins the #179 omission — must stay) |
| `cmd/internal/couchcore/resume_launch_test.go` | 1 |

Their **premises** need correcting, not their assertions flipped: "couch pins
layout2" becomes "couch sends the thread's chosen layout at a cold boundary and
none at a warm one". The `warmresume_test.go` case is the one that must keep
asserting exactly what it asserts today.

**Mechanism is confirmed to work.** `pair resume <tag> --layout3` parses:
`ParseArgs` runs `extractLayoutRequest` first (`launcher/args.go:51`), stripping
layout flags before the positional guard, and `launchArgsAcceptLayout` admits
them for resume. `couch.go:433-439` records this as measured, not reasoned —
including the correction of an earlier comment that claimed the opposite.

**Honest sizing:** a small feature — flag, plumbing, the persistence decision,
the test premises, and the doc sweep — not a one-line change.
