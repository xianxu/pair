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

---

## Re-review — 2026-09-16T16:58:54-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 265 — couch crashes on switch to brain: no admitted endpoint |
| repo | pair |
| issue file | workshop/issues/000265-couch-crashes-on-switch-to-brain-no-admitted-endpoint.md |
| boundary | whole-issue close |
| milestone | — |
| window | 220a965a21838bd2265967db5c9be7870816b64e..12a5bf45c12d543a68bb068740ce6c530bf7af68 |
| command | sdlc close --issue 265 |
| reviewer | claude |
| timestamp | 2026-09-16T16:58:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All seven prior findings are genuinely addressed, and I re-measured rather than trusted them: I reverted each claimed fix in turn and confirmed exactly one new test goes red with the right message — the `paintNow` classification, `onResize`'s classification, `resizeLayout`'s typing, `paintStripLocked`, `inheritSize`, and the restored #255 bypass (which the AST guard alone could not catch, and which `TestPanelDropsChildOnlyEventsWithoutAskingThePresenter` now catches by reading *which* arm ran out of the trace). I also re-ran the class enumeration independently: four producers, all routed through the one `noDestination` constructor; six production consumers (`deliverPresenterInput`, `paintNow`, `onResize`, `writeEvents`, `paintStripLocked`, `inheritSize`), all classifying with `errors.Is`. Nothing else in `terminal` can answer `ErrNoDestination`, so the answer-side sweep is complete. What stops SHIP is not the code: the *enumeration restatements* the family was opened over are still stale in five places — the plan's Core concepts table and Task 2, the issue's own Plan row, `atlas/couch.md`'s COUCH_TRACE event list, and a round-1 Log claim that a deferral was "recorded there" in `pair#273` when it is not — and the mechanical guard pins only one of the four answer-producing methods, which is the same gap that let this class escape planning and then round 1.

## 1. Strengths

- **`cmd/internal/terminal/destination.go:8-28`** — the sentinel plus one constructor, with the *reason* (a routing answer is not a terminal failure, and the panel is a legitimate no-destination state) written where the next reader hits it. The `View` travelling with the error is what made every mutation I ran diagnosable from the failure message alone.
- **`cmd/internal/couchtty/panel_input_routing_test.go:178`** — the best test in the diff. It distinguishes the panel-check arm from the classification arm by asserting the trace *detail* is `panel` and not `input`. That is an independently stated invariant, not a restatement of the implementation, and it is the only thing in the package that goes red when the #255 bypass is restored (measured: all other 16 assertions and the AST guard stay green).
- **`TestNoDecodedEventKindCanStopThePanelConsole`** walks `makeInputEvent`'s closed set (9 kinds) rather than the three the bug report named, and `TestFocusedActorStillReceivesChildOnlyEvents` is the mirror that stops "drop everything" from passing. Together they test the routing table, not the reported symptom.
- **Test doubles are the real seams** — `ttyio.NewFake()`, `ptychild.NewFakeChild`, `hostty.NewFakeHost`, and `focusReporting` seeding DECSET 1004 so the fake child's `\x1b[I` emission actually gates on the mode. The delivery assertion reads the fake's `Writes()`, i.e. observable state, not a call count (ARCH-MOCK).
- **`atlas/couch.md:404`** — `ErrBackpressure` explicitly excluded from the scheme, with the reason ("a capacity answer, still fatal"). I verified the consumer at `console.go:1161` still escalates it. Naming the look-alike that is *not* in the class is exactly what keeps the next sweep honest.

## 2. Critical findings

None. Every behavioral claim in the round-1 rework is backed by a test I independently proved red, and `go vet` plus the `terminal`, `couchtty`, `termcmd` and `artifactpath` packages are clean apart from the three documented environmental failures (`ptychild` spawn / `mkdir /tmp`: "operation not permitted").

## 3. Important findings

**I1 — the enumeration sweep stopped at `atlas/`; five other restatements are stale, one is false.**
**This is the 3rd finding in family `stale-enumeration-claim`.** Earlier rounds fixed instances (BR-6 fixed both atlas files). Do not fix these six sites one at a time — the rule is:

> When the enumeration of `ErrNoDestination` producers/consumers changes, every durable artifact that restates it is regenerated in the same commit, and a claim about another artifact is verified against that artifact before it is written. The enumeration is greppable: `grep -rn 'sites\|three\|four' atlas/ workshop/plans/000265* workshop/issues/000265*` plus the plan's Core concepts table.

Measured prevalence in this window — 7 restatements, 2 swept, 5 stale:

| artifact | state |
|---|---|
| `atlas/couch.md:379` | swept to four ✓ |
| `atlas/terminal.md:85` | swept ✓ |
| `atlas/couch.md:1105-1113` — the canonical COUCH_TRACE event list | **stale**: `no-destination` is a 7th event and is not listed. The prose section at :364-405 describes it; the enumeration a reader parses does not. |
| `workshop/plans/…-plan.md:57` — Core concepts *Integration points* table | **stale**: lists `Input / mouseInput / UpdateChrome` (3 of 4), and omits `Console.onResize`, `terminalMux.paintStripLocked`, `terminalMux.inheritSize`, `Console.traceDropped`, `traceNoDestination` — all new or modified by the diff. The review contract rates a table/code contradiction Critical; I am rating it Important because the code is measured correct at all six sites, so there is no behavioral consequence — but it is the one artifact a future reader will treat as the design of record. |
| `workshop/plans/…-plan.md:206` — Task 2 title "**both** refusal sites", Files list naming only `:470` and `:478` | **stale**; and the plan has **no `## Revisions` section at all**, after the plan gate added `UpdateChrome` and the close gate added `resizeLayout`, both `termcmd` repaint/resize sites, and three tests. AGENTS.md §1 requires an appended `## Revisions` entry rather than a silently-diverging plan. |
| `workshop/issues/000265….md:105` — Plan row 3 | **stale**: "(three, not two — `UpdateChrome` was found by the plan gate)". It is four. |
| `workshop/issues/000265….md:384` — "**This is a lead for `pair#273`** and is recorded there." | **false**. I grepped `000273-…md` for `installObservedThreadActor`, "without selecting", "focus=actor", "durable", "no-destination": no match. The BR-4 lead — the durable `(focus=actor, selected=nil)` state that `installObservedThreadActor` creates — is not in #273, which is the one artifact that needs it. |

Also in the same class, in code rather than prose: `termcmd/presentation.go:208-209` and `:361-362` assert "A tab can exist while nothing is admitted" / the presenter "can disagree" with pair term's tab model. I could not substantiate that for `pair term` today — `admitTab` is the only path that appends a tab and it calls `selectLocked` atomically (rolling the tab back on failure), `removeTab` reselects before dropping, `Retire` refuses while selected, and nothing in `termcmd` calls `Panel`; so `m.active >= 0` implies a selected endpoint. The test at `routing_answer_test.go:19-25` builds the state by hand for exactly that reason. Keep the guards — the class rule demands them — but say "defensive: unreachable through `admitTab` today; the guard exists so the class cannot reappear if `pair term` grows a panel", so the next reader does not go looking for the production path.

**I2 — the mechanical guard pins 1 of the 4 answer-producing methods; `termcmd` has none.**
**This is the 3rd finding in family `routing-answer-escalation`.** Round 1 fixed `resizeLayout` (BR-1) and `paintStripLocked`/`inheritSize` (BR-2) as instances. Do not add a seventh classification site and stop; the rule is:

> Every console/mux call to a presenter method that can answer `ErrNoDestination` must classify that answer. Enforce it on the *method set*, not on `Input` alone.

`input_door_guard_test.go` matches only `<x>.presenter.Input(...)`, so `UpdateChrome`, `Resize` and `ResizeLayout` — three quarters of the answer surface, and the two that round 1 caught — have no mechanical pin at all. A new `c.presenter.UpdateChrome(...)` call forwarding to `terminalError` compiles, passes the guard, and reproduces the crash on a path with no input in it. The cheap encoding: extend the existing AST walk to flag a call to any of `{Input, UpdateChrome, Resize, ResizeLayout}` on `*.presenter` whose enclosing function body does not reference `terminal.ErrNoDestination` (or is the allowlisted door). I checked it is satisfiable today in both packages without moving code — `paintNow`, `onResize`, `deliverPresenterInput`, `writeEvents`, `paintStripLocked` and `inheritSize` each already name the sentinel. `termcmd` has no guard test of any kind, and it is the package where the state is unreachable and therefore least likely to be caught by a failing test.

## 4. Minor findings

- `cmd/internal/couchtty/trace.go:172-176` — `traceEvent`'s doc comment ("records one timing-trace event… caller must not hold `c.mu`") is now orphaned above `traceDropped`; `traceEvent` itself is undocumented. Move `traceDropped` below `traceEvent`, or re-separate the comments.
- `workshop/lessons.md:5119` and `:5145` — two bullets state the same lesson ("Enumerate the answer, not the call site" / "Enumerate the ANSWER, not the callers"), one from the plan gate and one from BR-1. Merge them; `lessons.md` is read at every session start and a duplicated rule reads as two rules (ARCH-DRY).
- `traceDropped` writes `err.Error()` into the trace detail, while the sibling `reattachDoneDetail` documents "records a code, never the error's text". Substantively fine (the payload is a static string plus `%q`-quoted endpoint IDs, which the format explicitly permits, and `%q` escapes any tab/newline so the TSV line cannot be broken) — but the convention now has two answers and neither cites the other. One sentence at `atlas/couch.md:1115`.

## 5. Test coverage notes

- Every claimed round-1 fix has a red oracle, verified by me and not taken on the commit message: `resizeLayout` typing → `presenter_test.go:860` (`resize-layout` row) and `routing_answer_test.go:70` (`resize` row); `onResize` classification → `panel_input_routing_test.go:150`; `paintNow` classification → `:143`; `paintStripLocked` → `routing_answer_test.go` (`repaint` row); `inheritSize` → (`resize` row); the panel check → `:178`.
- The one uncovered producer, `mouseInput`, is a *measured* absence and the measurement is correct: `Ready` is `ViewState`'s zero value (`view.go:8`), so an idle presenter passes `v.State != Ready` and the click resolves to `ParentPressMouse`. The comment at `presenter_test.go:828-837` states this; I confirmed it.
- Gap implied by I2: no test asserts the *rule* that a consumer must classify. Every current test asserts a specific site's behavior, so the next unclassified call site ships green.
- `termcmd`'s two tests exercise a state the production constructor cannot reach (see I1). They pin the code but report nothing about a reachable interleaving — worth saying so in the test comment so a future reader does not read green as coverage of production behavior.

## 6. Architecture notes

- **ARCH-DRY — pass.** One sentinel, one constructor, four producers. The six consumers each spell the `errors.Is` check inline with genuinely different control flow (continue / return / fall-through-and-resize-children / inverted guard), so that is not extractable duplication. Flagged instead: the `lessons.md` duplication above.
- **ARCH-PURE — pass.** `ErrNoDestination`/`noDestination` are pure and `destination_test.go` runs with no writer, host or endpoint — no mocks needed, so the PURE claim in the plan's table holds. The classification lives in the IO shell, which is where it belongs; `deliverChildInput`'s only decision is a one-line lock-guarded read of an already-pure predicate (`Focus.IsPanel`).
- **ARCH-PURPOSE — flag (I1, I2).** The shadow-sweep on the *code* is complete: I enumerated all four producers and all six consumers and confirmed each derives from the single source. What does not derive is the enforcement (3 of 4 methods hand-maintained, I2) and the restatements (I1). Those are the two remaining hand-maintained consumers of the model.
- **ARCH-MOCK — pass.** Production and test flow share the same seams (`ttyio`, `ptychild`, `hostty` fakes), and `terminalqualify` carries the live conformance cases for the real terminal. The one flag is the hand-built `terminalMux` noted in I1.
- **ARCH-CONSTRAINTS — pass.** Added cost on the keystroke path is one `errors.Is` plus, when tracing is on, one `TryLock`ed line append that drops rather than blocks. I checked the repaint paths for a per-frame writer: the status spinner is gated on pending reattach placeholders, and chunk handling does not repaint per chunk, so trace growth is operator-paced, not frame-paced.
- **ARCH-SECURE — pass.** No credentials; the trace payload is a static reason plus `%q`-quoted endpoint IDs, which are addresses the trace format permits, and the quoting prevents a payload from breaking the TSV framing. The one convention divergence is the Minor above.
- **ARCH-ORDER — pass on the tested surface; the residual is deferred.** The decoder's closed set × panel state is a real `(state, event)` table, and `:178` asserts an invariant independently of the transition implementation. The residual, already disposed as BR-4 so not re-raised: `(focus=actor, presenter selected=nil)` is still *representable, reachable and durable* — two state machines over one question with no reconciliation. This diff makes it survivable, not legal. Collapsing it (a select that follows the focus, or a focus that cannot outrun the selection) is the real ARCH-ORDER fix and belongs to `pair#273` — which is exactly why I1's last row matters: the lead is not actually recorded there.
- **ARCH-FUNERAL — pass.** No new artifact family. The one new writer to an existing family (the operator-named, 0600, operator-removed COUCH_TRACE file) grows per operator action, not per frame.

## 7. Plan revision recommendations

The plan needs a `## Revisions` section — it has none, and two gates have changed its scope:

1. **`### 2026-09-16 — plan-quality gate: a third refusal site`** — reason: the design enumerated the refusal sites by caller (`grep '.Input('`) and so named two. `UpdateChrome` (`presenter.go:783`) answers the same condition. Delta: Task 2's Files list gains `:783`; its title stops saying "both refusal sites"; the Core concepts *Integration points* table gains `Presenter.UpdateChrome` and `Console.paintNow`.
2. **`### 2026-09-16 — close boundary review round 1: a fourth site and two more consumers`** — reason: the same by-caller enumeration, re-applied. Delta: Task 2 gains `resizeLayout` (`presenter.go:634`), so the enumeration is **four**, not three; Task 5's scope widens from `writeEvents` to `paintStripLocked` and `inheritSize`; the Core concepts table gains `Console.onResize`, `terminalMux.paintStripLocked`, `terminalMux.inheritSize`, `Console.traceDropped` (new) and `traceNoDestination` (new); Task 3 gains the three mutation-proven tests (`TestChromeRepaintWithNoEndpointDoesNotStopTheConsole`, `TestResizeWithNoEndpointDoesNotStopTheConsole`, `TestPanelDropsChildOnlyEventsWithoutAskingThePresenter`) and records that `deliverPresenterInput` returns void; Task 1's ARCH-FUNERAL/`Out of scope` note records that `cmd/internal/artifactpath` holds an exhaustive production-source inventory that a new file must join.
3. **The issue's `## Plan` row 3** — change "(three, not two — `UpdateChrome` was found by the plan gate)" to four, naming `resizeLayout` and the gate that found it, so the issue and the atlas agree.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      resizeLayout typed; both consumers classify. Mutation-verified: reverting the typing reddens presenter_test resize-layout and termcmd resize; reverting onResize's classification reddens TestResizeWithNoEndpointDoesNotStopTheConsole.
  - id: BR-2
    disposition: addressed
    note: |
      paintStripLocked and inheritSize both classify; reverting each independently reddens exactly its row in TestRoutingAnswerDoesNotStopTheMuxOnRepaintOrResize.
  - id: BR-3
    disposition: addressed
    note: |
      Both halves now have a red oracle. Measured: deleting paintNow's errors.Is block reddens TestChromeRepaint...; restoring the pair#255 bypass reddens TestPanelDropsChildOnlyEventsWithoutAskingThePresenter via the trace detail, which the AST guard cannot see.
  - id: BR-4
    disposition: addressed
    note: |
      traceNoDestination added on all four drop paths, via the non-repainting channel this finding named; reachability confirmed by the two mutations and asserted for the panel arm. See new finding on the atlas trace-event list and the unrecorded pair#273 lead.
  - id: BR-5
    disposition: addressed
    note: |
      deliverPresenterInput is void; no caller can misread a latched failure as nil.
  - id: BR-6
    disposition: addressed
    note: |
      Both atlas files now say four and name all four sites, with the by-answer enumeration rule spelled out. The same claim in the plan and issue is still stale -- raised as the class, not re-raised here.
  - id: BR-7
    disposition: addressed
    note: |
      Absence is now measured and documented at presenter_test.go:828-837; I confirmed Ready is ViewState's zero value (view.go:8), so an idle presenter cannot reach mouseInput's guard.
findings:
  - id: new
    severity: Important
    family: stale-enumeration-claim
    title: |
      Third in family: the enumeration sweep stopped at atlas; five restatements stale, one cross-artifact claim false
    detail: |
      Do NOT fix these sites one at a time. Rule: when the ErrNoDestination
      producer/consumer enumeration changes, every durable artifact restating it is
      regenerated in the same commit, and a claim about another artifact is verified
      against that artifact before it is written. Measured prevalence, 7
      restatements / 2 swept / 5 stale: atlas/couch.md:1105-1113 omits the new
      no-destination trace event from the canonical COUCH_TRACE list; plan:57 Core
      concepts table lists 3 of 4 producers and none of onResize,
      paintStripLocked, inheritSize, traceDropped, traceNoDestination; plan:206
      Task 2 still says "both refusal sites" and the plan has no "## Revisions"
      section at all despite two gates changing its scope (AGENTS.md section 1);
      issue:105 Plan row still says "three, not two"; issue:384 claims the BR-4
      lead "is recorded there" in pair#273 -- grepped, it is not. Same rule in
      code: termcmd/presentation.go:208 and :361 assert a tab can exist while
      nothing is admitted, which I could not substantiate (admitTab selects
      atomically, removeTab reselects first, Retire refuses while selected,
      nothing calls Panel) -- keep the guards, correct the claim to "defensive,
      unreachable today".
  - id: new
    severity: Important
    family: routing-answer-escalation
    title: |
      Third in family: the AST guard pins 1 of the 4 answer-producing methods, and termcmd has no guard at all
    detail: |
      Do NOT add a seventh classification site and stop. Rule: every console/mux
      call to a presenter method that can answer ErrNoDestination must classify
      it, enforced on the method set rather than on Input alone.
      input_door_guard_test.go matches only <x>.presenter.Input, so UpdateChrome,
      Resize and ResizeLayout -- three quarters of the answer surface, and the
      exact two that round 1 caught -- have no mechanical pin. A new
      c.presenter.UpdateChrome forwarding to terminalError compiles, passes the
      guard, and reproduces the crash with no input involved. Cheap encoding:
      extend the existing AST walk to flag a call to any of {Input, UpdateChrome,
      Resize, ResizeLayout} on *.presenter whose enclosing function does not
      reference terminal.ErrNoDestination. I verified this is satisfiable today in
      both packages with no code movement.
  - id: new
    severity: Minor
    family: orphaned-doc-comment
    title: |
      traceEvent's doc comment is now orphaned onto traceDropped
    detail: |
      trace.go:172-176 -- the inserted traceDropped sits between traceEvent's
      comment and traceEvent, so traceDropped carries a two-paragraph comment
      whose first half describes a different function and traceEvent is
      undocumented. Move traceDropped below traceEvent.
  - id: new
    severity: Minor
    family: duplicated-lesson
    title: |
      lessons.md ships the same lesson twice in the same section
    detail: |
      workshop/lessons.md:5119 "Enumerate the answer, not the call site" and
      :5145 "Enumerate the ANSWER, not the callers" are the same rule, recorded
      once from the plan gate and once from BR-1. lessons.md is read at session
      start; a duplicated rule reads as two rules (ARCH-DRY). Merge, keeping
      BR-1's measurement that the same mistake escaped two gates.
  - id: new
    severity: Minor
    family: stale-enumeration-claim
    title: |
      traceDropped writes the error text where the sibling helper documents "a code, never the error's text"
    detail: |
      Substantively safe -- the payload is a static reason plus %q-quoted endpoint
      IDs, which the trace format permits as addresses, and %q escapes tab and
      newline so the TSV framing cannot break. But reattachDoneDetail's comment
      states the opposite convention and neither cites the other. One sentence at
      atlas/couch.md:1115 settling which applies.
```
