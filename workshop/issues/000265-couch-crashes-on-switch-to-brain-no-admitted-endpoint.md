---
id: 000265
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-16
estimate_hours: 1.70
started: 2026-09-16T09:04:38-07:00
---

# couch crashes on switch to brain: no admitted endpoint

## Problem

Switching couch to the `brain` repo crashes couch.

Repro (from operator report):

- Trigger: switch to `brain` (coding harness not remembered — not the usual harness variant couch expects).
- Observed: `brain` appears in the tab bar, but the main viewport is fully blank. No content is rendered.
- Then: pressing a key (possibly `Esc`, uncertain) prints a raw escape sequence at the top-left corner instead of being handled.
- Then: couch exits with:

  ```
  couch: terminal: terminal: no admitted endpoint
  ```

Source of the inner error is `cmd/internal/terminal/presenter.go:494` inside `Presenter.Input`:

```go
if v.State != Ready || v.Admitted == "" || p.selected == nil {
    return errors.New("terminal: no admitted endpoint")
}
```

That path is the non-mouse input path — any key event when the presenter has no admitted endpoint. The double `terminal:` prefix in the fatal log indicates the error is wrapped once on the way out of the terminal package and again at the couch top-level.

Current behavior turns a blank/unadmitted pane into a fatal couch exit on the next keystroke. Expected behavior: the pane should either render (admit the endpoint) or degrade without taking down the whole couch; unhandled input on an unadmitted/blank view should be dropped or surface a non-fatal diagnostic, not crash.

Open questions to resolve during fix: what harness `brain` is actually running (and why its endpoint never reaches `Ready`/`Admitted`), why the viewport stays blank while the tab exists, and why the key event leaks as a raw escape sequence instead of being consumed.

## Spec

- Preserve the report verbatim in `Problem` (switch-to-brain → blank viewport → escape sequence leak → crash with `terminal: no admitted endpoint`).
- Trace the switch path for `brain`: endpoint creation/selection, `Presenter.Select` → `View` transition, and what leaves `v.Admitted == ""` / `v.State != Ready` / `p.selected == nil` after the switch. Identify why `brain`'s harness produces an endpoint that never admits.
- Make `Presenter.Input` (and any other `no admitted endpoint` throw site) non-fatal for the couch process: unadmitted input should be ignored or return a handled error without exiting couch. If the error is still surfaced, it must be non-fatal and actionable (which endpoint/view state, which harness).
- Fix the blank-viewport side: either ensure the `brain` endpoint is admitted and paints, or show an explicit empty/error state instead of a blank screen with a leaked escape sequence.
- Keep the fix in the terminal/presenter seam; do not add a `brain`-only workaround in the viewer layer.

## Done when

- Switching to `brain` no longer crashes couch; a key press (including `Esc`) on a blank/unadmitted pane does not exit the process.
- `brain` either renders its content after switch, or shows a coherent empty/error placeholder instead of a blank screen with raw escape sequences leaking to the top-left.
- The `terminal: no admitted endpoint` condition is handled without a fatal couch exit and, if logged, includes enough context to diagnose the endpoint/view state.
- Regression coverage exists for the input-on-unadmitted-endpoint path (key event when `Admitted == ""` / not `Ready` / `selected == nil` does not panic/exit).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Derivation, so the numbers can be checked rather than trusted:

- **Step 2 — primitives.** Seven, one per plan task plus the close boundary review. Task 1 (`destination.go`, a sentinel + one constructor) and Tasks 4–5 (an AST guard mirroring `menu_inventory_guard_test.go`; a three-line termcmd change plus its test) are all *smaller-go-module* — well-specced, mirror-or-extend. Task 2 touches the same refusal shape at three sites in one file, which is *cross-cutting-refactor*. Task 3 carries the door, the panel rule, the `paintNow` classification and three test families against a `(state, event)` model, so it is *tui-screen*. Task 6 is *atlas-docs*, and the close review is *milestone-review*.
- **Step 2.5 — library availability.** N/A, and stated rather than skipped: this is routing between two of our own packages. No library short-circuits it, so no design halving applies.
- **Step 3 — spec-quality discount ×0.2 on design.** Applied to every primitive. `workshop/plans/000265-…-plan.md` carries the exact code for each edit, file:line for every call site, full test bodies, and the ARCH-ORDER routing table — the design decisions are already made, and two plan-review passes plus the plan-quality gate resolved the three that were still open. Design hours here are the cost of *reading* that, not making it.
- **Step 4 — Method B.** Not used; every primitive matched the table.
- **Step 5 — familiarity ×1.0 on impl.** Familiar territory. The bug is reproduced, every symbol the plan names is verified to exist, and several of the tests have already been compiled and run against HEAD during review.
- **Step 6 — buffer +15%** on design, not +30%: the thorough-plan-doc case in v2.1's calibration.
- **v3.1 scaling.** Each `impl=` is 40% of the v2 primitive-table implementation hours, per v3.1's ship-wall-clock unit. Design hours are unscaled.

The largest risk to this number is not the code — it is `make test`, which must run with the retention-owner env scrubbed and a non-symlinked `TMPDIR` (see `workshop/lessons.md`). That is carried in the *milestone-review* impl hours.

**Revised after the estimate-quality gate (2026-09-16), 1.08 → 1.70.** Three of its four points are folded in; the fourth is recorded rather than adopted.

- *Operator verification was itemized nowhere.* Correct — the plan has a standalone `## Operator verification` section, mandatory per `#209`, and the Step 2 enumeration walked tasks, so a plan section fell through it. Added as `ux-rename-iteration`, the primitive that fits an operator round-trip.
- *`milestone-review` was carrying three things in 8.4 minutes* — the close review, the full `make test` with its env scrub, and any rework. The ledger says pair closes actually run 2 gate rounds (`#248`) to 5 (`#255`). Now two `milestone-review` items at the top of the band, because the primitive models one chunk each and two rounds is the observed floor here.
- *`tui-screen` was underweighted.* Task 3 spans Steps 1, 3, 3b and 3c, touches `terminal_input.go` and `console.go`, and adds three test families against the `(focus, event)` table — larger than Tasks 1, 2, 4 and 5 combined, yet it had `impl=0.20` against their 0.36. Raised to the upper-middle of its band on both columns.

Recorded, not adopted — **the nearest comparator is `pair#248`**, which carried this exact triple (`estimate=1.08, design=0.28, impl=0.76`) under the same model in the same couch/console subsystem with a thorough plan, and landed at **7.21h (ratio 0.15)**. Deduped pair rows run a median ≈0.94, so the fleet-wide "v3.1 runs ~2× high" cushion does not apply here. This estimate expects to land nearer 1.0 than 0.15, because the design dialogue that `#248` paid for during implementation has already been paid here — two plan-review passes and a plan-quality round closed every open question, and the churn is mostly test code whose content is written out in the plan. **If the close gate runs more than two rounds, expect this to drift toward `#248`'s ratio**; that is the number to check at close rather than explain away.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.02 impl=0.08
item: cross-cutting-refactor design=0.04 impl=0.08
item: tui-screen design=0.24 impl=0.32
item: smaller-go-module design=0.02 impl=0.10
item: smaller-go-module design=0.02 impl=0.10
item: atlas-docs design=0.02 impl=0.06
item: ux-rename-iteration design=0.06 impl=0.08
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.06
total: 1.70
```

## Plan

Rewritten 2026-09-16 to match the narrowed scope — see `## Revisions`. Rows are
the plan at `workshop/plans/000265-couch-crashes-on-switch-to-brain-no-admitted-endpoint-plan.md`.

- [x] Reproduce / narrow: log the `View` state (`State`, `Admitted`, `Selected`) with the panel focused; confirm which guard fails at `presenter.go:470`. *(Done 2026-09-16: `state=0 selected="" admitted=""`; `Admitted == ""` is the failing term, and it is the panel's correct state.)*
- [x] Identify why the endpoint is unadmitted. *(Done: it is not a `brain` harness fault. `Presenter.Panel` clears `selected` by design; three event kinds skip the panel check and ask anyway. Root cause is routing, not admission.)*
- [x] Add the typed no-destination answer in `cmd/internal/terminal` and return it from every refusal site — **four**: `Input`, `mouseInput`, `UpdateChrome` (plan gate), `resizeLayout` (close gate, BR-1). The set is now derived by AST in `TestNoDestinationProducersAreEnumerated`, so a fifth cannot appear unenumerated.
- [x] Route every couch input through one panel-aware door so unadmitted input is non-fatal, and the panel check governs key release, focus and blur. `paintNow` classifies the same answer.
- [x] Pin the door with an AST guard, proven red against a deliberate violation.
- [x] Sweep the class: `termcmd/presentation.go` has the same escalation via `stopLocked`.
- [x] Add regression coverage for input on an unadmitted endpoint (all three kinds), the mirror that a focused actor still receives them, and the decoder's full closed set (9 kinds).
- [x] Atlas + lessons; operator smoke test passed (2026-09-16).

Moved out of scope (see `## Revisions`): "why paint is blank" → `pair#273`;
"fix or surface the blank-viewport case" → `pair#273`.

## Log

### 2026-09-15

- Recorded from operator report: switch to `brain` → tab appears but viewport blank → key/Esc leaks escape sequence at top-left → `couch: terminal: terminal: no admitted endpoint` and exit. Source pinned to `cmd/internal/terminal/presenter.go:494`. Issue created as `000265`.

### 2026-09-16 — state-machine inspection (startup lifecycle)

New symptom from the operator: couch also cannot **start** in `brain` — tab bar
appears, viewport blank, couch sits there; `ctrl+space` then exits with
`couch: terminal: terminal: no admitted endpoint`.

**The crash is fully explained and reproduced.** Two distinct defects, one
proven, one diagnosed from on-disk state.

#### 1. Input routing bypasses the panel check for three event kinds (PROVEN)

`Presenter.Panel` is *defined* to leave no admitted endpoint: it sets
`p.selected = nil`, and `SelectView` with `EndpointID == ""` leaves
`v.Selected == ""`, so `PresentView` lands `State=Ready, Admitted="",
selected=nil`. That triple is the panel's normal, healthy state — and it is
exactly the guard at `presenter.go:494`. So "no admitted endpoint" is not an
error condition; it is what the panel looks like.

`couchtty/terminal_input.go:41` routes three event kinds to
`presenter.Input` **without consulting `c.focus.IsPanel()`**, unlike every
other key path in the same function:

```go
case uv.KeyReleaseEvent, uv.FocusEvent, uv.BlurEvent:
    c.terminalError(c.presenter.Input(c.lifetime, event.Event))
    return
```

Couch asks the parent terminal for both families in its first mode delta
(`parentModeDelta`'s `!known` branch writes `\x1b[?1004h` — focus reporting —
and `\x1b[>3u` — kitty flags 1|2, i.e. *report event types*, which is key
release). So couch requests exactly the events that kill it. `makeInputEvent`
confirms these are the only three non-mouse kinds that reach that arm; every
other kind is either `Reply` (dropped) or goes through `route`.

Reproduced deterministically (scratch test, 3/3 subtests):

```
view after showMenu: state=0 selected="" admitted=""
REPRO: console stopped; terminalFailure = terminal: no admitted endpoint   [key-release]
REPRO: console stopped; terminalFailure = terminal: no admitted endpoint   [focus]
REPRO: console stopped; terminalFailure = terminal: no admitted endpoint   [blur]
```

The doubled prefix in the operator's message is confirmed as the fatal path:
`Console.teardown` prints `"couch: terminal: %v\n"` over the presenter's own
`"terminal: no admitted endpoint"`.

Introduced by `f32bb4cf` (`#255 M3: migrate Couch and Pair to owned terminal
state`) — `git log -S` shows that arm has exactly one commit. This is a #255
regression, not a pre-existing condition.

**Severity is a second, separable defect.** `Console.terminalError` latches
*any* error as `terminalFailure` and calls `Stop()`. An input-routing
mismatch is not a terminal-ownership failure: the presenter already separates
physical failure (`p.fail` → `Failed` state, `Failed()` channel) from a
returned error, and the console collapses the two. `mouseInput`'s
`terminal: no presented mouse destination` is the same class, reached from
`routeMouseEvent`'s unconditional release forward.

Not brain-specific: any repo where the panel is focused when one of those three
events arrives. `brain` hits it because nothing there lands the operator on an
admitted endpoint (see below), so the panel is where couch sits.

#### 2. A park wedged in `awaiting_completion` (the never-arriving event)

Operator's question — *what event is couch waiting for in brain?* — resolves to
the **park completion** for `brain·couch-e1a31510b7033d08`. Its record
(`threadstore/records/2e51fcf9799b1d8f/couch-e1a31510b7033d08.json`, rev 293,
last written 2026-09-15 22:53) still carries:

```json
"park": { "phase": "awaiting_completion",
          "identity": { "pid": 64734, ... },
          "attempts": [ { "number": 1, "timed_out": true,
                          "failure": { "code": "timeout",
                                       "diagnostic": "matching completion was not observed" } } ] }
```

pid 64734 is dead (verified). `ClassifyThread`
(`actionableinventory.go:265`) branches `if record.Park != nil { return
ThreadBusy, "" }` — total, unconditional, with **no expiry and no
owner-liveness check**. So the thread is permanently `ThreadBusy`
("parking in progress" in `couch --list`), and `ThreadBusy` is invisible to
every startup predicate:

- `SelectResumableRoot` ranks only `Detached`/`Parked` → never resumed
- `PathHoldsUsableThread` matches only `Live`/`Detached`/`Parked` → does not hold the path
- `reattachCandidate` wants `Detached`, or `Unusable` **with `ReasonUnknown`** → the pass skips it

Net effect: every `couch` in brain spawns a *fresh* thread instead of returning
to the operator's work, and leaves another record behind. `couch --list`
currently shows brain with 3 × `stale — helper ownership unresolved` plus the
wedged busy row; the three stale records are dated 09:02, 09:29 and 10:00 today
— one per start attempt. Compare: a park that timed out is the one lifecycle
state with no recovery path, whereas `recovery.go:84` does handle
`ReasonStaleIncarnation` and `ReasonSessionGone`.

#### Still open

Which of the three events fired in the operator's run is not yet pinned, and the
blank-viewport/hang between startup and `ctrl+space` is not yet attributed —
it could be the freshly-spawned `muse` child simply not painting, which is a
different layer. One instrumented start settles both:

```
COUCH_INPUT_TRACE=/tmp/couch-input.jsonl COUCH_TRACE=/tmp/couch-trace.jsonl couch
```

Neither open question changes fix 1: all three kinds are corrected by the same
routing fix, and the fatality question is independent of which one arrived.

## Revisions

### 2026-09-16 — narrowed to the routing half

Reason: `## Spec` and `## Done when` were written from the operator report,
before the state-machine inspection. They bundle three symptoms that the
diagnosis has since separated into three different defects in three different
layers. Left as written, this issue cannot be closed honestly: Done-when #2 and
Spec bullet 4 would be unmet, and `## Plan` rows 2 and 4 would stay unticked
against the `plan-unchecked` close gate. Caught by the plan-document review.

Delta — this issue now owns the **input-routing crash only**:

- **Spec bullet 4** ("Fix the blank-viewport side: either ensure the `brain`
  endpoint is admitted and paints, or show an explicit empty/error state") is
  **removed from scope**, together with **Done-when #2** (the `brain`
  renders-or-placeholder clause). Those describe a symptom that is not yet
  attributed and has no reproduction. `pair` carries no resumable thread either
  and couch works there, so the fresh-spawn path is not broken in general; one
  `COUCH_INPUT_TRACE`/`COUCH_TRACE` start in `brain` is what will attribute it.
  It gets its own issue once it does — recorded here so it is deferred, not
  dropped (ARCH-PURPOSE).
- The **raw escape sequence at the top-left** from the original report is part of
  that same unattributed half and moves with it.
- **Spec bullet 2** ("Trace the switch path for `brain`… identify why `brain`'s
  harness produces an endpoint that never admits") is **answered, not deferred**,
  and the answer changes its premise: nothing about `brain`'s harness fails to
  admit. The panel's `State=Ready, Admitted="", selected=nil` is the *correct*
  and healthy shape of a console showing its own screen. The defect is that
  three event kinds skip the panel check and ask the presenter anyway, and that
  the console escalates its correct answer to a fatal exit. See the
  2026-09-16 `## Log` entry.
- **Spec bullet 5** ("keep the fix in the terminal/presenter seam; do not add a
  `brain`-only workaround") **stands and is satisfied**: the fix is in the
  routing seam and in the presenter's answer type, with nothing `brain`-specific
  anywhere.

Unchanged and still in scope: Spec bullets 1 and 3, and Done-when #1, #3 and #4.
Scope **added** since the original wording, under ARCH-PURPOSE's class rule:
`termcmd/presentation.go` has the identical escalation via `stopLocked`, so the
fix sweeps both callers rather than the one couch site the report named.

The park-wedge half of the `brain` diagnosis is `pair#271`.

### 2026-09-16 — `## Plan` rows rewritten

Reason: the earlier Revisions entry narrowed `## Spec` and `## Done when` and
then left the `## Plan` checkboxes describing the old scope — the exact
`plan-unchecked` close-gate problem it named as its own motivation. Caught by the
plan-document review's second pass.

Delta: rows 2 and 4 named the blank-viewport half and could not be honestly
ticked under the narrowed scope; row 4 is removed to `pair#273` and row 2's
"and why paint is blank" clause with it. Rows 1 and 2 are marked done, because
the state-machine inspection answered both — row 2's premise turned out to be
wrong, which is recorded inline rather than silently ticked. The remaining rows
now mirror the durable plan's tasks one-to-one. Also corrected the stale
`presenter.go:494` citation in row 1 and in `## Problem`: the guard is at
`presenter.go:470` today, and `mouseInput`'s at `:478`.

### 2026-09-16 (implementation)

Tasks 1–5 landed; Task 6 done bar the operator smoke test.

- `terminal/destination.go` — `ErrNoDestination` + `noDestination`, the one
  constructor, carrying the `View` so a surfaced refusal names its state.
- Three refusal sites converted: `Input` (`presenter.go:470`), `mouseInput`
  (`:478`), `UpdateChrome` (`:783`).
- `couchtty`: `deliverPresenterInput` is the one door and classifies the answer;
  `deliverChildInput` adds the panel check, and the three bypassing kinds now go
  through it. `paintNow` stops escalating `UpdateChrome`'s refusal — the same
  crash reachable with no input at all, through the window `showMenu` opens
  between `presenter.Panel` and the focus flip.
- `termcmd/presentation.go` swept (same escalation via `stopLocked`).
- 14 new couchtty assertions + the AST guard + 2 terminal + 1 termcmd. The guard
  was proven red against a planted `c.presenter.Input(...)` before being trusted.

Two things the run surfaced that the plan did not predict:

1. `cmd/internal/artifactpath` holds an **exhaustive production-source
   inventory**, so a new file fails `TestProductionArtifactReferencesAreExactlyClassified`
   until it is registered. `destination.go` added to `NonArtifactSources`.
2. `TestConsoleRunRootEscapeClearsFilterThenReplaysActor` is **flaky and
   pre-existing** — ~1-in-40, more likely under `make test` load. Proven
   pre-existing by reverting this issue's two couchtty production files to the
   base commit and reproducing it identically. Recorded in `lessons.md`; wants
   its own issue.

Verification: `make test` with the retention-owner env scrubbed and a
non-symlinked `TMPDIR` — 209 packages ok, the only failure being the flake
above. Two couchtty tests and one termcmd test need the sandbox off
(`ptychild` spawn, `mkdir /tmp`), which is environmental and documented.

### 2026-09-16 — operator smoke test PASSED

The gesture that produced this issue no longer exits couch. Operator ran the
`#265` build in `brain` (the spawn path, so couch comes up over a blank pane —
`pair#273`), opened the switcher, and **switched the thread's harness from
`muse` to `claude`**. Before this change, `ctrl+space` on a blank pane exited
with `couch: terminal: terminal: no admitted endpoint`; now the switcher paints
and a full switcher operation completes.

That is the real verification. The 14 new couchtty assertions prove the console
does not latch a failure; only the operator's terminal proves the event that
was killing couch is actually gone, and the dogfood rule
(`memory: feedback_pair_dogfood_and_agnostic`, violated once in `#209`) is that
the operator says so, not the tests.

Incidental, and a lead for `pair#273`: the harness switch to `claude` succeeded
from a blank `muse` pane. If the pane became usable afterwards, the blank
viewport may be `muse`-specific rather than generic to couch's spawn path —
which would contradict this issue's earlier reasoning that every fresh spawn
blanks. Worth confirming before `#273` is designed.

### 2026-09-16 — close boundary review round 1: REWORK, addressed

Four blocking findings, all real, all reproduced by the reviewer before being
raised. The two Criticals are the same failure of mine twice: **I enumerated the
callers instead of the answer.**

- **BR-1 (Critical) — `resizeLayout` is a fourth refusal site.**
  `presenter.go:634` returned a plain `errors.New("terminal: resize requires
  endpoint")` for the same `p.selected == nil` condition, and `Console.onResize`
  handed it to `terminalError` — so *resizing the terminal window* still exited
  couch. `termcmd.inheritSize` had it too. The plan's own Task 2 acceptance
  said "re-run the real enumeration… a fourth would belong in Task 2"; I wrote
  that instruction and did not execute it. Converted, classified in both
  consumers, and `onResize` now falls through to resize the children rather than
  returning.
- **BR-2 (Critical) — `pair term`'s repaint path.** Task 5 swept `writeEvents`
  only. `paintStripLocked` guards on `activeTabLocked()` — pair term's own tab
  model, the exact disagreement the shipped test encodes — then fed
  `UpdateChrome`'s (already typed) answer to `stopLocked`. Fixed with
  `inheritSize`, and both now have a test row.
- **BR-3 (Important) — two halves of the fix had no red oracle.** Measured by
  the reviewer: deleting the `paintNow` classification left the package green,
  and restoring the `#255` bypass kept all 16 assertions green *and* passed the
  AST guard, because the door is the allowlisted callee either way. Three new
  tests, each **mutation-proven red** against its own fix:
  `TestChromeRepaintWithNoEndpointDoesNotStopTheConsole`,
  `TestResizeWithNoEndpointDoesNotStopTheConsole`, and
  `TestPanelDropsChildOnlyEventsWithoutAskingThePresenter` — which distinguishes
  the two arms at the trace, because the panel check drops *before* the
  presenter is asked (`panel`) while a bypass is classified *at* it (`input`).
- **BR-4 (Important) — every non-fatal path was silent, in a durable state.**
  The reviewer's insight is the important part: `installObservedThreadActor`
  sets `c.focus` to an actor **without selecting it** when `c.active` is empty,
  and nothing later selects. So `(focus=actor, selected=nil)` persists, and the
  operator sits with a blank viewport, no chrome and dead keys — with my change,
  now guaranteed silent. Added the `no-destination` trace event; `traceEvent` is
  the non-repainting channel my own notice analysis was looking for and did not
  find. **This is a lead for `pair#273`** and is recorded there.

Minors: `deliverPresenterInput` returned an error nothing consumed and reported
fatal errors as nil — now void; both atlas files said "three sites" — now four,
with the enumeration rule spelled out; `mouseInput` has no test row, and that is
now a *measured* absence rather than an omission (see the comment on
`TestPresenterRefusalsWithoutADestinationAreClassifiable`: `Ready` is
`ViewState`'s zero value, so an idle presenter never reaches that guard).

Verification after rework: `make test`, same env scrubs — **210 packages, exit
0, zero failures**. The pre-existing
`TestConsoleRunRootEscapeClearsFilterThenReplaysActor` flake did not recur.

### 2026-09-16 — close boundary review round 2: FIX-THEN-SHIP, ledger addressed

Verdict FIX-THEN-SHIP with two open Important findings, both tagged **"third in
family"**. The gate's message was the useful part: *"Not converging: fix rules,
not instances."* Both were fixed as rules.

- **BR-9 — the AST guard pinned 1 of 4 answer-producing methods, and `termcmd`
  had no guard at all.** Round 1 fixed `UpdateChrome` and `resizeLayout`; the
  guard I shipped watched only `Presenter.Input`, so a new
  `c.presenter.UpdateChrome` forwarding to `terminalError` would compile, pass
  the guard, and reproduce the crash. Replaced with
  `terminal/no_destination_contract_test.go`, which **derives** the producer set
  from source (`TestNoDestinationProducersAreEnumerated`) and checks every
  consumer call in both `couchtty` and `termcmd`
  (`TestConsumersOfNoDestinationMethodsClassifyIt`). Mutation-proven red three
  ways: dropping a producer from the mapping, unclassifying couch's `paintNow`,
  and unclassifying termcmd's `paintStripLocked` — the last two each named the
  exact file:line and function.
- **BR-8 — the enumeration sweep stopped at the atlas; 5 of 7 restatements
  stale, one cross-artifact claim false.** All corrected: the `no-destination`
  trace event is now in the atlas's canonical `COUCH_TRACE` list and pinned by
  `TestAtlasNamesEveryTraceEvent` (proven red by renaming the constant); the
  plan's Core-concepts table lists all four producers and all six consumers; the
  plan gained the `## Revisions` section AGENTS.md §1 requires, recording both
  scope growths; this issue's Plan row says four, not three. The false claim —
  that BR-4's lead "is recorded" in `pair#273` — was false, and is now true:
  `pair#273` carries it, with a cheaper distinguishing test than the trace
  capture.
- Also per BR-8, two `termcmd` comments asserted a hazard I could not
  substantiate. Verified: `pair term` never calls `presenter.Panel`, `admitTab`
  selects atomically, `removeTab` reselects before retiring, and `Retire`
  refuses while selected — so an admitted tab always has an endpoint. The guards
  stay; the comments now say **defensive and unreachable today**, with why they
  are kept.

Verification: `make test`, same env scrubs — **210 packages, exit 0, zero
failures.**

### 2026-09-16 — close boundary review round 3

One blocking finding, and it was a bug this issue introduced rather than one it
missed.

- **BR-13 — `onResize`'s no-destination arm left the focused child at its old
  size, and my comment claimed the opposite.** The loop under the presenter call
  skips the focused child *because the presenter's apply callback normally
  resizes it*. When the presenter refuses, that callback never runs — so the one
  pane the operator is looking at kept its attach-time geometry. Fixed by
  tracking `presenterResized` rather than re-deriving the skip from
  `!panel && selected != nil`. `TestResizeWithNoEndpointStillResizesTheFocusedChild`
  asserts the focused child agrees with `ChildSize()`, and is mutation-proven
  red against the old condition.

Also addressed, though the gate recorded it as non-blocking, because it was the
**4th finding in the `routing-answer-escalation` family** and the gate's standing
note is "fix rules, not instances": `consumerPackages` was a typed list of two,
so a third package acquiring a presenter would be unchecked, and the matcher saw
only `X.presenter.Method(...)` — blind to a presenter held in a local, which is
exactly how `terminalqualify` holds one. Both halves are now derived:
`discoverConsumerPackages` reads the tree for anything referencing
`terminal.NewPresenter` or `*terminal.Presenter`, and `presenterValued` tracks
locals assigned from `NewPresenter`. `terminalqualify` is now an **explicit
exemption with a reason** rather than an invisible gap — removing that exemption
turns the test red on all six of its sites, which is how I verified the local
detection works.

Verification: `make test`, same env scrubs — **210 packages, exit 0, zero
failures.**
