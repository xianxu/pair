---
id: 000198
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
started: 2026-09-06T12:06:08-07:00
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

## Spec

The operator can start a couch thread in layout3, and it stays layout3 across a
park/resume cycle.

- Layout is **per-thread**, carried in `StartArgs` alongside agent and argv,
  defaulting to layout2 (unchanged behavior for every existing thread and every
  persisted record).
- The flag reaches the argv **only at a cold start/resume boundary**. A warm
  reattach continues to send no layout flag at all — the #179 invariant is
  preserved verbatim, not re-derived.
- Old persisted records with no layout field read as layout2.

Open for the plan: the operator-facing surface — a `couch` CLI flag at start, a
menu affordance on the thread, or both. Whichever is chosen, it sets the
persisted per-thread value rather than a global mode.

**Worth deciding explicitly, not assuming:** the pin's rationale was that couch
*replaces* layout3's third pane. Running layout3 under couch means the operator
has both couch's terminal switching and pair's right terminal at once. That is
presumably the point of the request — but it is the substance of the reversal,
so record the answer in the issue rather than letting the code imply it.

## Done when

- A thread started with layout3 launches `pair resume <tag> --layout3`, and the
  operator gets pair's right terminal inside couch.
- That thread comes back as layout3 after a park/resume cycle; a thread started
  without the flag is layout2.
- A **warm reattach sends no layout flag**, pinned by the existing
  `warmresume_test.go` case still passing unmodified.
- Persisted records written before this change load as layout2 with no error.
- Test premises corrected across the couch-side files above; no test still
  asserts couch pins layout2 as policy.
- `atlas/couch.md` updated at all four sites (`:201`, `:815`, `:880`,
  `:903-904`) and the rationale comment at `couch.go:427` replaced with the new
  rule — the 2026-08-22 pin is recorded as reversed, with the reason.

## Plan

- [ ] Decide the operator surface (CLI flag / menu / both) and confirm
      per-thread persistence.
- [ ] Add the layout field to `StartArgs` with a layout2-default read for
      records that predate it; test the old-record path.
- [ ] Thread it to `launchTrackedThread`'s cold-boundary argv, leaving the
      `in.Warm` branch untouched.
- [ ] Correct the test premises; verify `warmresume_test.go` passes unmodified.
- [ ] Sweep `couch.go:427` and the four `atlas/couch.md` sites.
- [ ] Manual: start a layout3 thread under couch, park it, resume it, confirm
      the layout survives and the live session is never offered for deletion.

## Log

### 2026-09-06

Filed from an operator request, with the question "I think it's just one line
change?" — the honest answer is that the literal is one line and the change is
not, for the four reasons in Problem. The comment at `couch.go:427` explicitly
left room for this knob, so the design does not resist it; what makes it more
than a swap is the destructive warm-reattach neighbour (#179), the persistence
fork, and eighteen test references plus five doc sites that encode the old pin
as policy.

**The three connected pieces.** This issue is the flag; `#199` builds the tab
strip `pair term` draws in its own pane; `#200` makes that strip clickable on
one shared mouse arbitration. Only this one touches couch — `#199`/`#200` are
`termcmd` + `hostty` work — so **neither gates couch-lite's close**.

### The reversal, on the record

`#198`'s Spec asked for the reason rather than letting the code imply it. Two
independent reasons, and they point the same way.

**1. The pin's rationale did not survive the rescope.** The 2026-08-22 pin reads
"couch owns terminal switching now, so layout3's third pane — pair's own user
terminal — is the layer couch replaces". That was an actor-cluster-era claim,
made when couch was specified to own everything on the host
(`workshop/projects/couch.md`, 2026-08-21). `#170` rescoped couch to
**couch-lite** on 2026-09-02: a switcher over live coding sessions, with
Admission — fleet capacity and incumbency, its cross-repo provider dependency,
its stateful fake and its live conformance target — deleted whole. couch-lite
switches *agent sessions*; it does not hand the operator a shell at their cwd.
So the layer couch was said to replace is one couch-lite never took over. The
reversal is a correction the rescope already implied, not a change of taste.

**2. The operator's three original reasons have each moved** (2026-09-06):

| original reason for layout2 | what changed |
|---|---|
| keep it simple initially | couch-lite works well and is close to done (2026-09-03 scope event: one remaining issue) |
| unsure whether the right pane should host a web browser too, cmux-style | dropped — the pane is terminal-only, and constant access to a terminal at the same cwd is wanted |
| reservations about the right pane's quality: zellij tabs, the title mechanism, mouse | `#199`/`#200` — couch's status bar generalizes into a tab bar the pane manages itself |

The third reason is the one that was a genuine blocker, and `#172` is why it is
cheap to answer now: it turned a reserved interactive row from an idea into
working code, with its mouse-mode hazards found and fixed across four rounds
(BR-16 → BR-22 → BR-26 → BR-33/`#196`).

**Relationship to #194** (`open`, "Toggle a terminal pane in layout2"): both
answer "the operator wants a terminal", from opposite directions. #194 argues
layout3's permanent 50%-width third pane is the wrong posture and adds an
on-demand terminal to layout2 instead; this issue makes that permanent pane
available under couch. Not contradictory — different postures for different
sittings — but whoever picks up the second of the two should check whether the
first changed the want. Recorded rather than resolved: that is the operator's
call.
