# Boundary Review — pair#265 (whole-issue close)

| field | value |
|-------|-------|
| issue | 265 — couch crashes on switch to brain: no admitted endpoint |
| repo | pair |
| issue file | workshop/issues/000265-couch-crashes-on-switch-to-brain-no-admitted-endpoint.md |
| boundary | whole-issue close |
| milestone | — |
| window | 220a965a21838bd2265967db5c9be7870816b64e..4da8b09b3790d6c7def96ac0995ab193c6b0c76f |
| command | sdlc close --issue 265 |
| reviewer | claude |
| timestamp | 2026-09-16T16:36:48-07:00 |
| verdict | REWORK |

## Review

I inspected the pinned range, read the issue + plan, and reproduced two remaining crash paths with scratch tests (created, run, deleted — `git status --porcelain cmd/` verified clean after each; the working tree is untouched).

```verdict
verdict: REWORK
confidence: high
```

The structural idea is right and well executed: `ErrNoDestination` is the correct third value, the one-door pattern is clean, and the AST guard genuinely goes red (I planted a violation and confirmed it names `routeInputEvent`). The reported crash is gone, and the operator smoke test is real evidence. What blocks SHIP is the class sweep the plan itself made a checkable step — "every refusal in `cmd/internal/terminal` that reports absence of an endpoint … confirm it is exactly these three. A fourth would belong in Task 2." There is a fourth (`resizeLayout`'s `resize requires endpoint`), and the *consumer* enumeration was applied to couchtty's repaint path but not to `pair term`'s. Both remaining sites are reachable and fatal, and I reproduced both: a terminal resize in the exact state the operator reported still exits couch, and a repaint still stops `pair term` on the **already-typed** `ErrNoDestination`. That is the instance-versus-class distinction (ARCH-PURPOSE), not a new defect class.

## 1. Strengths

- `cmd/internal/terminal/destination.go:20` — the sentinel + single constructor is the right shape, and the doc comment explains *why* the panel's empty state is healthy rather than just what the code does. Pure, IO-free, colocated test.
- `cmd/internal/couchtty/input_door_guard_test.go:33` — verified red: planting `_ = c.presenter.Input(c.lifetime, nil)` in `routeInputEvent` fails with `terminal_input.go:52:7 in routeInputEvent`. A guard that has been red is worth having.
- `panel_input_routing_test.go:96` — `TestNoDecodedEventKindCanStopThePanelConsole` drives the decoder's full closed set rather than the three kinds the report named. That is the right oracle shape (ARCH-PURPOSE applied to tests).
- `TestFocusedActorStillReceivesChildOnlyEvents` with the `\x1b[?1004h` seed — the plan's analysis that an unseeded fake writes nothing either way is correct, and the mirror test is what stops an over-dropping fix.
- ARCH-MOCK: production and test share the same `hostty.NewFakeHost` / `ptychild.NewFakeChild` boundary; no new double introduced.
- `atlas/couch.md` and `atlas/terminal.md` are specific and load-bearing, including the `ErrBackpressure` non-goal. Docs gate satisfied for atlas; README correctly untouched (no new user-facing surface).

## 2. Critical findings

**C1 — `resizeLayout` is a fourth refusal site; it is untyped and still fatal.**
`cmd/internal/terminal/presenter.go:634` returns `errors.New("terminal: resize requires endpoint")` when `p.selected == nil` — the same answer as the three converted sites. `Console.onResize` (`cmd/internal/couchtty/console.go:1222-1228`) reads `panel := c.focus.IsPanel()` and calls `presenter.Resize` whenever focus is *not* panel, then hands the error to `c.terminalError` → couch exits.

Reproduced (scratch, since removed):
```
SCRATCH REPRO: resize latched a terminal failure: terminal: resize requires endpoint
```
The state is not a microsecond race. `installObservedThreadActor` (`console.go:419`) sets `c.focus = FocusActor(handleID)` when `c.active == ""` and the attach is foreground, **without** selecting in the presenter; `finishOperation` then reads `panelFocused` after that write (`console.go:1795`, `:1828`) so it does not `showMenu`, and only `resume`/`recover-*` take the `forceSwitch` path (`console.go:1816`). So `(focus=actor, selected=nil)` persists — which is also a plausible account of the blank viewport in the original report and in pair#273. In that state this diff made keystrokes and chrome paints safe but left window resize fatal.

Fix sketch: `return noDestination("resize requires endpoint", p.View())` at `presenter.go:634`, and classify in `Console.onResize` the way `paintNow` now does (skip; the resize that follows the next select is authoritative). Same treatment for `termcmd` `inheritSize` (`presentation.go:394-398`). Then correct `atlas/terminal.md` and `atlas/couch.md`, which currently assert "Three sites answer with it."

**C2 — `pair term`'s repaint path still escalates `ErrNoDestination`.**
Task 5 swept `writeEvents` only. `terminalMux.paintStripLocked` (`cmd/internal/termcmd/presentation.go:353-363`) guards on `m.activeTabLocked() == nil` — `pair term`'s own tab model, which is exactly the "two models can disagree" premise the shipped `TestRoutingAnswerDoesNotStopTheMux` encodes — then calls `UpdateChrome` and hands any error to `stopLocked`. Built from the *same* unadmitted mux the shipped test constructs, a repaint kills the mux:
```
SCRATCH REPRO: a repaint latched a mux failure:
terminal: no destination for input: chrome requires selected endpoint (state=0 selected="" admitted="")
```
The plan's ARCH-PURPOSE section names both enumerations ("by input" and "by repaint") and applies the repaint one only to couchtty's `paintNow`. Fix sketch: `if errors.Is(err, terminal.ErrNoDestination) { return }` in `paintStripLocked`, plus a test alongside `TestRoutingAnswerDoesNotStopTheMux` driving `m.paintStrip()` on the identical fixture.

## 3. Important findings

**I1 — Two behavior changes in this diff have no test that fails without them.**
Measured, both restored afterwards:
- Removing the `errors.Is(err, terminal.ErrNoDestination)` block from `console.go:1116-1123` (Task 3 Step 3b) leaves `go test ./cmd/internal/couchtty/` with only the two pre-existing environmental failures (`TestNotificationPTYConformance`, `TestCouchProductionSoak`). Nothing goes red.
- Restoring the `#255` bypass — `case uv.KeyReleaseEvent, uv.FocusEvent, uv.BlurEvent: _ = c.deliverPresenterInput(event.Event)`, i.e. the door without the panel check — keeps all 16 new couchtty assertions green **and** passes the AST guard, because `deliverPresenterInput` is the allowlisted callee.

So the shipped tests pin the *classification* half of Task 3 and nothing pins the *panel-rule* half, which is the half the issue's title describes. Fix sketch: a `paintNow` test driving `(focus=actor, selected=nil)` (the C1 fixture reaches it in four lines), and either an oracle for the drop (count presenter round-trips, or assert `deliverChildInput` returns before the door) or widen the guard so routing arms in `routeInputEvent` must call `deliverChildInput`, not `deliverPresenterInput`.

**I2 — The new non-fatal paths emit no diagnostic at all.**
`paintNow` returns silently, `deliverChildInput` drops silently, `deliverPresenterInput` discards. Because the `(focus=actor, selected=nil)` state is durable rather than transient (see C1), an operator in it sees a blank viewport, no chrome, and no keystroke effect, with zero signal — the pair#273 symptom, now guaranteed silent on this path. The plan's rejection of notices is correct (re-entrancy through `publishNotice`), but `c.traceEvent` is already the non-repainting channel and costs nothing on the happy path. Fix sketch: one trace event at the classification point in `deliverPresenterInput` / `paintNow`.

## 4. Minor findings

- `terminal_input.go:20-26` — `deliverPresenterInput`'s return value has zero consumers: all three call sites discard it, and it returns `nil` even when a genuinely fatal error was just latched, so the signature promises a distinction no caller uses. Either make it `func(uv.Event)` or have `deliverChildInput` use the result.
- `presenter_test.go:830` — covers `Input` and `UpdateChrome`; `mouseInput`'s converted line is exercised only indirectly through the couchtty mouse subtests. A third table row would be one line.
- `atlas/couch.md` / `atlas/terminal.md` both state "Three sites answer with it" — an enumeration claim that goes stale the moment C1 lands.

## 5. Test coverage notes

- `go build ./...` clean; `go vet` on the three touched packages clean; `./cmd/internal/terminal/` and `./cmd/internal/artifactpath/` pass.
- `couchtty` and `termcmd` fail only on `ptychild`-spawn / `mkdir /tmp` tests, which the issue log already documents as environmental.
- Guard test verified red against a planted violation.
- Gaps are I1 (two untested behavior changes) plus the missing coverage implied by C1/C2 — note that the resize and repaint paths are *absent* from the plan's `(focus, event)` table, which is precisely why they were missed; the table enumerates input kinds only.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (C1): `resizeLayout` restates the same fact the "ONE constructor" was introduced to own.
- **ARCH-PURE** — pass: `noDestination` is a pure `(string, View) -> error`, tested with no writer, host or endpoint.
- **ARCH-PURPOSE** — flag (C1, C2): the class is "a caller that escalates a presenter routing answer to a fatal failure"; the sweep covered couch fully and `pair term` only on the input axis, and never re-ran the *producer* enumeration the plan asked for before starting.
- **ARCH-MOCK** — pass.
- **ARCH-CONSTRAINTS** — pass: one extra mutex read plus an `errors.Is` per event; panel events now skip a presenter FIFO round-trip. The plan's rejection of a per-refusal notice on cost grounds holds.
- **ARCH-SECURE** — pass on input/credentials (typed `uv.Event` at the boundary; error text carries only couch-minted endpoint ids). Noted under I2: the degradation is invisible rather than visible.
- **ARCH-ORDER** — partial. The door correctly re-reads `c.focus` under `c.mu` and the flip-to-panel interleaving is analysed. The deeper issue this diff exposes but does not resolve: `c.focus` and `p.selected` are two authorities for one fact, with `(actor, nil)` a legal-but-meaningless combination reachable from `console.go:419`. That is the boolean-constellation shape — worth collapsing into one authority (or making `focus=actor` structurally imply a selected endpoint) in a follow-up, since every site that reads one and calls the other is a candidate for this same bug.
- **ARCH-FUNERAL** — pass: package-level sentinel, per-event allocation on the refusal path only, nothing durable.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000265-…-plan.md`:

- The ARCH-PURPOSE producer enumeration ("confirm it is exactly these three: `Input`, `mouseInput`, `UpdateChrome`") is **incomplete**: `resizeLayout` (`presenter.go:634`) is a fourth site reporting absence of an endpoint, escalated fatally by `couchtty.onResize` and `termcmd.inheritSize`. Record it as belonging to Task 2.
- Task 5's sweep of `termcmd` covered the input axis only; `paintStripLocked` (`presentation.go:353`) escalates the same typed answer. The "by repaint" enumeration must be applied to `termcmd` as it was to couchtty.
- The ARCH-ORDER `(focus, event)` table should gain rows for the two non-input events that reach the presenter — **repaint** and **resize** — since their absence is what made both misses invisible to the plan review.
- Task 3 Step 3b and the panel-check half of Step 3 shipped without red oracles; record that and name the tests that would supply them.

```findings
findings:
  - id: new
    severity: Critical
    family: routing-answer-escalation
    title: |
      resizeLayout is a fourth no-endpoint refusal: untyped, and a terminal resize still exits couch
    detail: |
      presenter.go:634 returns a plain errors.New("terminal: resize requires endpoint") for the
      same p.selected == nil condition the three converted sites report, and Console.onResize
      (console.go:1222-1228) hands it to c.terminalError. Reproduced with a scratch test from the
      durable (focus=actor, selected=nil) state that installObservedThreadActor (console.go:419)
      creates when c.active == "" on a foreground attach: "resize latched a terminal failure:
      terminal: resize requires endpoint". termcmd.inheritSize (presentation.go:394-398) has the
      same escalation. Convert to noDestination and classify in both consumers; the plan's own
      Task 2 acceptance says a fourth site belongs there, and both atlas files assert "three sites".
  - id: new
    severity: Critical
    family: routing-answer-escalation
    title: |
      pair term's repaint path still stops the mux on the already-typed ErrNoDestination
    detail: |
      Task 5 swept only writeEvents. terminalMux.paintStripLocked (termcmd/presentation.go:353-363)
      guards on activeTabLocked() -- pair term's own tab model, the exact disagreement the shipped
      TestRoutingAnswerDoesNotStopTheMux encodes -- then feeds any UpdateChrome error to stopLocked.
      Reproduced on the identical fixture that test builds: "a repaint latched a mux failure:
      terminal: no destination for input: chrome requires selected endpoint". The plan names both a
      by-input and a by-repaint enumeration and applied the second to couchtty only.
  - id: new
    severity: Important
    family: fix-without-red-oracle
    title: |
      The paintNow classification and the panel-check half of Task 3 have no test that fails without them
    detail: |
      Measured and restored. Deleting the errors.Is block at console.go:1116-1123 leaves the couchtty
      package green apart from the two known environmental failures. Restoring the #255 bypass
      (key release / focus / blur calling deliverPresenterInput directly, no panel check) keeps all
      16 new assertions green AND passes the AST guard, because the door is the allowlisted callee.
      So only the classification half is pinned; the routing half named in the issue title is not.
  - id: new
    severity: Important
    family: unsignalled-degradation
    title: |
      Every new non-fatal path is silent, and the state it degrades in is durable rather than transient
    detail: |
      paintNow returns, deliverChildInput drops, deliverPresenterInput discards -- no trace, no log.
      Because (focus=actor, selected=nil) persists until the operator switches (console.go:419 never
      selects and finishOperation only forceSwitches for resume/recover), an operator sits with a
      blank viewport, no chrome and dead keys with zero signal -- the pair#273 symptom, now
      guaranteed silent. c.traceEvent is already the non-repainting channel the plan's notice
      analysis was looking for.
  - id: new
    severity: Minor
    family: dead-return-value
    title: |
      deliverPresenterInput's error return has no consumers and misreports fatal errors as nil
    detail: |
      All three call sites discard it, and it returns nil after terminalError has latched a genuine
      failure, so the signature advertises a distinction nothing uses.
  - id: new
    severity: Minor
    family: stale-enumeration-claim
    title: |
      atlas/couch.md and atlas/terminal.md both assert "three sites answer with it"
    detail: |
      That count is already wrong given resizeLayout, and will need updating with the C1 fix.
  - id: new
    severity: Minor
    family: fix-without-red-oracle
    title: |
      presenter_test.go covers Input and UpdateChrome but not mouseInput's converted refusal
    detail: |
      mouseInput's noDestination line is exercised only indirectly through couchtty's mouse
      subtests; a third table row in TestPresenterRefusalsWithoutADestinationAreClassifiable is
      one line.
```
