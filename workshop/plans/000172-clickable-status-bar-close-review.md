# Boundary Review — pair#172 (whole-issue close)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..a2ab2d2d2740e718dce136c3a58b8b385458c8b0 |
| command | sdlc close --issue 172 |
| reviewer | claude |
| timestamp | 2026-09-06T00:42:26-07:00 |
| verdict | REWORK |

## Review

I've completed the review, verifying claims by mutation and probe in a scratch copy of the pinned tree.

```verdict
verdict: REWORK
confidence: high
```

The pure geometry half of this issue is genuinely well built — chip spans come out of the clipping pass, `RouteMouseReport` is a real table with a property test over the whole event space, and the `#127` prefix-parking class is pinned at every split boundary this time. What blocks SHIP is the terminal half and the artifacts around it. I reproduced, at the pinned head, two mode-policy defects BR-26 named and one it implied: a child that enables `?1000h` without `?1006h` now receives SGR reports it cannot parse because couch sets `?1006` at startup and never releases it — behavior drift from the "a child that did enable tracking still receives its own events unchanged" Done-when, introduced by this diff; and a bare `?1000l` from the child never triggers a repaint, so the take-back BR-16's fix was supposed to deliver never fires on its own (probe: `host.Written()` stays empty). Separately, three claimed fixes are not pinned by any test — reverting the `if len(effects) > 0` guard, the `enterOperationFor` call in the click arm, and the column-vs-rune accounting in `RenderStatusRow` each leaves the entire `couchtty` suite green — and the atlas paragraph the *same commit* added still teaches the model the code refutes. Fixing the mode policy is a design change with its own test matrix, not a one-liner, which is why this is REWORK rather than FIX-THEN-SHIP.

## 1. Strengths

- `cmd/internal/couchtty/mouse.go:56` — `RouteMouseReport`'s three-way disposition with the full table in the doc block, and `mouse_test.go:53` `TestNoReportReachesAChildThatNeverAskedForMouse` asserting the zero-bytes rule as a *property* over button × release × every row, not a case list. That is the right shape for a Done-when phrased as "asserted, not assumed".
- `cmd/internal/couchtty/reserve.go:174-177` — spans recorded from inside the same `appendText` that clips (`used` accumulates `textwidth.Width`, so they are true columns, correct for a mouse X). `TestAChipClippedToOneColumnIsStillClickable` *constructs* the one-column case rather than sweeping widths and asserting whatever it finds — the fix BR-4 asked for, done properly.
- `cmd/internal/couchtty/keys.go:282-303` — the mouse arm placed before the fixed-string table with the `MaxReport` bound, pinned by `TestMouseReportsAreWithheldAndSurviveEverySplit` (every split index) and `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard`.
- `cmd/internal/couchtty/core_concepts_contract_test.go:414-421` — the `assertDirectTest` self-skip is a real guard fix, and it paid immediately: it exposed a second vacuous assertion (`endsItsOwnChild`), which got its own test at `menu_action_sweep_test.go:116`. A guard fix that finds a second instance on its first run is the good outcome.
- `console_menu.go:187-192` — extents captured at the single `RenderMenuView` host call site, so they cannot go stale against the drawn panel. I checked every `showMenu` caller and `takeOverScreen` writes `HomeAndClear` first, so body line *i* is screen row *i+1* and the `Y-1` conversion is right.

## 2. Critical findings

None newly raised. (BR-26's `?1006` half is behavior drift from a stated Done-when and would be Critical if raised fresh; it is disposed under its existing id rather than re-raised.)

## 3. Important findings

See the `findings` block. In brief: BR-5, BR-17, BR-26, BR-27 and BR-28 all remain open at the pinned head, each verified rather than read off a commit message; one new finding on the chip-span coordinate *unit* being unpinned.

## 4. Minor findings

- BR-9 — `go doc ChipSpan` still prints the whole `RenderStatusRow` rationale; `go doc RenderStatusRow` prints nothing. The "fix" added a meta-paragraph about the problem *inside the block that has the problem*.
- BR-29 — `mouseinput.go:51`'s `s == ""` is still unreachable.
- BR-30 — `console_mouse_test.go:146` still checks only `w[0] == 0x1b`.
- BR-31 — `manifest.go:549,557` still out of alphabetical order.

## 5. Test coverage notes

Verified by mutation in a scratch copy (`./cmd/internal/couchtty/` full package; the only pre-existing failure is `TestNotificationPTYConformance`, environmental — `ptychild: operation not permitted`):

| mutation | result |
|---|---|
| `index := len(lines)` → `start + i` in `renderRootMenuFrame` | **RED** (`TestPointToActorSpansEveryLineOfAnActor`) — BR-5's named mutation is now caught |
| `clampExtents` body → `return extents` | GREEN — and a panic probe confirms neither branch fires anywhere in the suite; the code now says so honestly |
| scroll-branch panic probe (`start > 0`) under the three extent tests | never entered — the 12-actor fixture selects actor 0, so it clips at the bottom but never scrolls |
| `if len(effects) > 0` guard removed (BR-17) | GREEN |
| `enterOperationFor(thread)` → `"switch"` in the click arm | GREEN |
| `used += textwidth.Width(clipped)` → `len([]rune(clipped))` | GREEN |

The last three are the fix-completeness rule: a fix is complete only when a test fails without it.

## 6. Architectural notes

- **ARCH-DRY — pass with one flag.** `enterOperationFor`, the single `mouseinput` parser, and spans-from-the-clipping-pass are all the right consolidation. The flag is that the *restatements* (atlas §356-361, the plan's disposition table) are hand-maintained copies of the model that have now diverged from it — BR-27 and BR-28.
- **ARCH-PURE — pass.** `ChipSpan`/`ActorExtent`/`ColumnToActor`/`PointToActor`/`RouteMouseReport` all run with no IO; `Console.onMouse` is the thin shell and does the one 1-based→0-based conversion.
- **ARCH-PURPOSE — flag.** Shadow-sweep on the plan-as-source: it is a restatement that no longer derives from the tree in at least six places with no `## Revisions` entry (BR-28), and the Done-when's "scrolled list" case is the deferred subset (BR-5).
- **ARCH-MOCK — flag, and it is the root of BR-26.** `ptychild.Screen` is the fake modelling the terminal's mouse state, and it models four DECSET modes as one bool. There is no live conformance check for the claim the whole policy rests on ("1000/1002/1003 replace one another; 1006 is orthogonal") — that belief lives only in a code comment. The fake and the dependency have already drifted; the `?1002h`+`?1006l` case is where.
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002/1003` is declared and enforced by the constant; `MaxReport` bounds the held buffer; `paintNow`'s new `childWantsMouse()` call is outside the lock it re-takes (no deadlock, confirmed by the passing suite).
- **ARCH-SECURE — pass.** Labels are `sanitize`d before both the render and the spans; `Parse` rejects non-numeric and out-of-range fields rather than guessing.
- **ARCH-ORDER — flag, the main one.** BR-26 is an ordering finding, not a mode finding: the take-back has no *event* of its own. `Screen`'s DECSET handler sets `s.mouse` without `rowDirty`, and `onChunk` repaints only on `RowDirty`/`Bell`/attention, so the `(child releases mode) → (couch re-asserts)` cell has no trigger. `TestMouseModeTransitions` cannot see this because it calls `repaint()` by hand. That is the "test that can only observe one interleaving" case: it confirms the ordering the author supplied.

## 7. Plan revision recommendations

`workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md` needs one `## Revisions` entry covering, at minimum: `:41-56` the disposition table has no `couchOwnsScreen` dimension and its "anything, elsewhere | yes | forward verbatim" row is contradicted by the switcher case; `:155` `FindSGR` shipped as `Find`; `:161` the stated three-argument `RouteMouseReport` signature shipped with four; `:165`/`:394`/`:396` describe `seqMouse` as live work that `:118` already records as deleted; `:396` `mouseinput.IsSGRPrefix` shipped as `IsPrefix`; `:182` `DisableMouseClicks` has status `new` and exists nowhere in the tree; `:186-192` states `FeedHit` returns the event alongside the hit, where what shipped is the `Interceptor.Mouse()` side-channel; and all 42 checkboxes are unticked including M1's Tasks 1-5. Also stale in the same class: `core_concepts_contract_test.go:61-62`'s comment says "The M2/M3 rows carry status `planned` and are invisible here" immediately above the M2 rows it lists, and the file still speaks of M2/M3 after the milestones collapsed to one.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Clamp half answered honestly (probe confirms unreachable); the scrolled
      case is still entered by no extent test -- panic probe on start>0.
  - id: BR-9
    disposition: not-addressed
    note: |
      go doc ChipSpan still prints the RenderStatusRow rationale; go doc
      RenderStatusRow prints nothing. Verified by running go doc.
  - id: BR-17
    disposition: not-addressed
    note: |
      enterOperationFor is shared (good), but both the resume branch and the
      len(effects)>0 guard survive reversion green; the notice divergence remains.
  - id: BR-25
    disposition: addressed
    note: |
      The named bullet and every rename residue are gone; the only remaining
      MouseClickTracking hits are the Revisions entries recording the rename.
      The class is carried by BR-28.
  - id: BR-26
    disposition: not-addressed
    note: |
      Untouched since round 8 (only a docs commit landed after). Both mechanisms
      reproduced at head; see the detail on the re-raise rule below.
  - id: BR-27
    disposition: not-addressed
    note: |
      atlas/couch.md:358-361 unchanged, and the diff shows the same commit ADDED
      that paragraph -- it is not stale prose, it is a new wrong claim.
  - id: BR-28
    disposition: not-addressed
    note: |
      Only :118 (seqMouse deleted) changed; five named sites plus two more remain
      and there is still no Revisions entry.
  - id: BR-29
    disposition: not-addressed
    note: |
      mouseinput.go:51 unchanged.
  - id: BR-30
    disposition: not-addressed
    note: |
      console_mouse_test.go:146 still w[0] == 0x1b.
  - id: BR-31
    disposition: not-addressed
    note: |
      manifest.go:549,557 unchanged.
findings:
  - id: new
    severity: Important
    family: unpinned-exported-shape
    title: |
      ChipSpan's documented column unit is unpinned -- swapping textwidth.Width for a rune count leaves the whole couchtty suite green
    detail: |
      This is the 4th finding in family `unpinned-exported-shape`. Earlier rounds
      fixed instances (BR-6 a missing test file, BR-13 a run shape, BR-30 a weak
      assertion). Do NOT fix this instance alone.

      The rule that covers all of them: a documented property of an exported shape
      is pinned only by a fixture that can DISCRIMINATE it. reserve.go:159
      accumulates `used += textwidth.Width(clipped)`, and ChipSpan's own doc calls
      Start/End "the column range" -- correct, because an SGR X is a terminal
      column. But every chip fixture is ASCII, where columns, runes and bytes all
      coincide, so replacing that line with `len([]rune(clipped))` leaves
      `./cmd/internal/couchtty/` fully green (verified by mutation). The same
      fixture-cannot-discriminate shape is why BR-30's first-byte check passes and
      why BR-5's clipped case needed constructing rather than sweeping. The
      enumeration this implies, and which should be swept in one round: for each
      exported geometry contract on this feature, name the property, then name the
      fixture value that violates it -- a CJK or emoji label for the column unit
      (labels are UNTRUSTED text and truncate()'s own doc calls out emoji), a
      scrolled window for the drawn-row base (BR-5), a report behind other bytes
      for the zero-bytes rule (BR-30). Measured prevalence: 3 of 3 geometry
      properties I mutated are unpinned by the shipped fixtures.
```

---

## Re-review — 2026-09-06T08:03:15-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..511bf9cac38b82151465358a324acee08eb400c1 |
| command | sdlc close --issue 172 |
| reviewer | claude |
| timestamp | 2026-09-06T08:03:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pure layer of this feature is in good shape — chip spans come out of the clipping pass, extents come off the `actorStart` boundary the scroll window already uses, the SGR parser is one package with a bound, and three of the fixes this round claims are pinned by tests I confirmed go red under mutation. What blocks SHIP is the mode-ownership half. The per-paint `?1000;1006h` re-assert introduced in this window (nothing wrote mouse modes before `c15030df`) decides whether to overwrite the child's tracking from a per-`Child` observation that starts empty and is never re-derived — and the operator has already filed `pair#196` (`deps: ["#172"]`) against exactly that: drag selection in the agent pane loses its live highlight under couch after some reattachments, the demotion symptom `console.go:1035-1042` documents in its own comment. That falsifies the Done-when this issue wrote for itself, from code this window added, and it is deferred to a follow-up rather than fixed. Separately, the Done-when's "on a scrolled or clipped list" extent case is still unpinned — I added `+ start` to the extent base at `menu_render.go:467` and the whole `couchtty` suite stayed green, which is BR-5's second half surviving its fix round.

## 1. Strengths

- **The transition table became the test list, and it discriminates.** `console_mouse_test.go:321-388` enumerates child-enables-1000/1002/1003, child-disables, child-exits, no-child, plus the two-child switch at `:372`. `TestCouchDoesNotDemoteAChildsTrackingMode` (`:298`) covers the case a keyboard smoke test cannot reach. Driving the exit through `onExit` rather than deleting the map entry (`:349-351`) is the right instinct — it tests a state production reaches.
- **Chip spans are pinned as display columns.** `reserve_mouse_test.go:170` with a CJK label; I replaced `textwidth.Width(clipped)` with `len([]rune(clipped))` at `reserve.go:159` and it went red. `TestAChipClippedToOneColumnIsStillClickable` (`:109`) constructs the width instead of sweeping and hoping.
- **`enterOperationFor` (`menu.go:604`) is one authority**, consumed by both `reduceRootKey:471` and the click arm — and the Manual-on-refusal guard is real: removing `if len(effects) > 0` at `menu.go:361` reddens `TestAClickRefusedMidOperationDoesNotMarkSomeoneElsesWork`.
- **`assertDirectTest` now excludes itself** (`core_concepts_contract_test.go:417-422`). A guard that greps its own directory and lists every symbol verbatim was asserting nothing; unmasking it is worth more than the rows it added.
- **The parser promotion is finished on the DRY axis.** `sgrMouseSize` (`rename_input.go:184`) derives the report length from `ParsePrefix` rather than restating the framing rule, and `MaxReport` bounds the Interceptor's hold (`keys.go:296`) with `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard` behind it — #127's dead keyboard cannot come back through this path.

## 2. Critical findings

**`console.go:1043` — couch overwrites a still-tracking child whenever its per-`Child` observation is empty, and nothing re-derives it.** *This is the 5th finding in family `unspecified-event-policy`.* Per the escalation protocol I am not asking for another site fix. The rule that covers BR-16, BR-22, BR-26 and this one: **couch decides whether to write a terminal-global mode from a value it can only ever have observed, so the decision needs a state model — what the child is believed to hold, what invalidates that belief, and what event re-evaluates it — not a bool read at paint time.** Today `Screen.mouse` is set only by `classify` seeing a DECSET in that `Child`'s output (`screen.go:428-431`), it is never queried, and a new pane for a still-running thread starts from `Screen{}` with `mouse=false` while `replay` can only re-derive it if the original DECSET is still inside `DefaultRingBytes`. `paintNow` then writes `?1000;1006h` over a child holding `?1002`, permanently, because nothing re-evaluates. Measured prevalence: 3 write sites (`console.go:530` startup, `:1044` per-paint, and the `MouseForward` path's encoding gate at `:1534`), 1 governed by the observation; 0 events that re-evaluate the belief — `onChunk` repaints only on `RowDirty`/`Bell`/attention and `classify` sets `rowDirty` for alt-screen alone, so a bare `?1000l` leaves couch's own clicks off until an unrelated repaint; and `TestMouseModeTransitions` cannot see any of this because it calls `con.repaint()` by hand. `pair#196`, filed by the operator with `deps: ["#172"]`, is this reaching production: "Standalone `pair`: correct … under `couch`, after some reattachment: broken", which is the Done-when "a child that did enable tracking still receives its own events unchanged" failing on code this window added.

## 3. Important findings

**`workshop/issues/000172-clickable-status-bar.md` `## Log` — the one Done-when with no automatic test is recorded as an assertion, not a measurement.** The plan's Task 12 Step 2 says in terms: "Record the result in `## Log` as a measurement, not as 'it worked'." The Log says "Operator smoke-tested twice on the real stack." It does not say which of the named steps ran (drag-select in the draft, wheel-scroll the agent pane, click a chip) or what each did, so the record cannot distinguish "nvim scroll verified" from "clicking worked". That matters concretely here: the case it was meant to cover is the one `pair#196` reports broken, and nvim/zellij set `?1006` so the mainstream child passes while the failing configuration is invisible. Fix: write the three steps and their observed results into `## Log`, and state which child was attached (whether it held `?1002` and whether it held `?1006`).

## 4. Minor findings

- `screen.go:107` — narrowing `Mouse()` to tracking-only also changes `termcmd.appMouseMode()` (`run.go:819`), which now sends the wheel to `zellij scroll-up` for a child that had emitted `?1006h` alone. *This is the 2nd finding in family `unnamed-seam-change`* — the rule: changing the semantics of an exported observation requires enumerating its call sites in the same commit and stating the effect at each. Measured prevalence: 2 consumers, 1 named. No live defect (a child holding `1006` without tracking receives nothing either way), and no termcmd test pins either direction.
- `menu.go:352` — a click on a non-actionable row, or on a chip whose thread is not yet in `state.Inventory`, returns silently; Enter sets `errorMenuNotice` at `:468-470` with the comment "Silence is what the operator reports as a bug". *This is the 2nd finding in family `parallel-handler-restates-decision`* — the rule: where the click arm and the key arm diverge, the divergence is declared (as the Manual flag is), not left as an omission.
- `menu_render.go:180-196` — `clampExtents` is documented unreachable and kept. The disposition is honest and I am not asking for a manufactured test; noting it so the next reader does not read it as covered behaviour.

## 5. Test coverage notes

- **Mutation-verified red** (so these fixes are real): the Manual-on-refusal guard, the display-column unit, the SGR-encoding forward gate. All three reddened exactly one test each.
- **Mutation-verified green** (so this is not covered): `index := len(lines) + start` at `menu_render.go:467` leaves `./cmd/internal/couchtty/` fully green. I instrumented `renderRootMenuFrame` and confirmed `start > 0` is entered by exactly one test in the package (`TestRenderMenuKeepsSelectedRowVisibleAndBounded`), which asserts nothing about extents. `TestExtentsNeverPointPastTheDrawnMenu` exercises the row-budget *clip* (`end < len(rows)`) but never the *scroll*, because its selected actor is `inventory[0]`. Constructing it is one line: select a late actor so the window starts below zero.
- No test can observe the ordering in the Critical above: `TestMouseModeTransitions` drives `repaint()` directly, so it samples one interleaving and reports no coverage of "what event re-evaluates the belief". A seam that lets a test drive a child DECRST and assert a repaint followed is what would close it.
- The suite is otherwise green in this checkout; the only failures are the pty-spawn ones this environment cannot run (`TestNotificationPTYConformance`, `ptychild/child_test.go`, `hostty`, `wrapcmd` TTY capture) — environmental, not this diff.

## 6. Architectural notes

- **ARCH-DRY — flag.** `enterOperationFor` and the single parser are correct. But the disposition is now decided in two places: `RouteMouseReport` (`mouse.go:53`) returns `MouseForward` and its doc comment presents itself as the complete table, while `onMouse` (`console.go:1534`) can turn that into a drop. The encoding dimension belongs in the router's signature and table, not in the shell.
- **ARCH-PURE — pass.** `ChipSpan`/`ColumnToActor`/`ActorExtent`/`PointToActor`/`RouteMouseReport` are all pure and tested with no IO; `onMouse` is the thin seam. The one leak is the encoding policy above.
- **ARCH-PURPOSE — flag.** Deferring `pair#196` to a follow-up defers the mode-ownership half that Spec §4 and the Done-when made the point of the issue, not a separable extension.
- **ARCH-MOCK — pass.** `ptychild.NewFakeChild` and `hostty.FakeHost` sit on the same seam production uses, `TestFakeChildConformsToRealChildLifecycle` is the conformance check, and the mouse path is driven through real bytes rather than by calling `onMouse`. The real-terminal half is manual, which is declared — see the Important above about how it was recorded.
- **ARCH-CONSTRAINTS — pass.** `?1000` and never `?1002`/`?1003` is the right envelope choice and the reason is written down at `control.go:51-58`. Nothing unbounded: the Interceptor's hold has `MaxReport`, the extents loop is O(rows).
- **ARCH-SECURE — pass.** Reports are parsed into a typed `Event` at the boundary and refused rather than clamped; `ColumnToActor`/`PointToActor` are total, so negative and oversized coordinates map to nobody; labels are sanitised before they reach a span; the bounded hold means a pasted `\x1b[<` cannot park the keyboard.
- **ARCH-ORDER — flag.** This is where the Critical lives. `Screen` now carries two independent bools (`mouse`, `sgrMouse`) and couch carries no state at all for "do I currently hold the terminal's mouse mode" — it is recomputed per paint. The legal states and the events that move between them are not written anywhere a test can drive: process still running behind a new pane, ring eviction of the original DECSET, a bare `?1000l` with no repaint following. Naming which of cancel/queue/preempt/ignore governs each is the missing enumeration.

## 7. Plan revision recommendations

- `## Revisions` entry: **"The disposition table is not complete."** `:47-56` still has no `couchOwnsScreen` dimension, so the two switcher rows the code carries are absent and the `anything, elsewhere | yes | forward verbatim` row is contradicted by both the switcher branch (`mouse.go:63-68`) and the encoding gate (`console.go:1534`). Re-derive the table from `RouteMouseReport` plus `onMouse`.
- Same entry: `:159-160` states `RouteMouseReport` takes `(event, hostRows, childWantsMouse)`; four parameters shipped.
- Same entry: all 41 checkboxes are unticked, including Tasks 1-5 and 10-12 whose work demonstrably landed. A plan ticked nowhere cannot be the record of what shipped.
- Consider recording BR-28's own rule as delivered-or-not: three named-symbol sites were corrected and the three structural ones (table, signature, checkboxes) were not, which is the instance rather than the class the rule asked for.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Clamp half honestly dispositioned; the SCROLLED half is still unpinned -- `index := len(lines) + start` at menu_render.go:467 leaves the whole couchtty suite green, and start>0 is entered by no extent test.
  - id: BR-9
    disposition: not-addressed
    note: |
      `go doc ChipSpan` still prints RenderStatusRow's untrusted-text rationale; the fix added a paragraph explaining the misattribution instead of moving it, and RenderStatusRow (reserve.go:138) now has no doc comment at all.
  - id: BR-17
    disposition: addressed
    note: |
      enterOperationFor consolidates the rule and the Manual-on-refusal guard is mutation-verified red; the notice divergence named in the detail is re-raised separately as a Minor under its family.
  - id: BR-26
    disposition: not-addressed
    note: |
      Tracking/encoding split shipped and pinned; the encoding is still never stood back from (the child is silently sent nothing instead), the startup write at console.go:530 is still ungated, and no event re-evaluates the belief -- see the new Critical for the class.
  - id: BR-27
    disposition: addressed
    note: |
      atlas/couch.md:358-368 rewrites the refuted paragraph and quotes it only to deny it; grep confirms no surviving assertion of the additive/idempotent model.
  - id: BR-28
    disposition: not-addressed
    note: |
      Three named-symbol sites fixed (DisableMouseClicks, seqMouse, IsSGRPrefix/FindSGR); the disposition table's missing couchOwnsScreen dimension, the three-argument RouteMouseReport signature at :159, and all 41 unticked checkboxes remain.
  - id: BR-29
    disposition: not-addressed
    note: |
      mouseinput.go:51 still reads `!strings.HasPrefix(s, "\x1b[<") || s == ""`.
  - id: BR-30
    disposition: not-addressed
    note: |
      console_mouse_test.go:145-149 unchanged, and the same first-byte-only assertion was copied into the new TestAChildThatDidNotAskForSGRIsNotSentSGR at :420-424.
  - id: BR-31
    disposition: not-addressed
    note: |
      manifest.go:549 and :557 unchanged -- mouseinput.go still sits inside the couchcore block and couchtty/mouse.go still precedes couchtty/menu.go.
  - id: BR-32
    disposition: not-addressed
    note: |
      The column-unit instance is fixed and mutation-verified, but the enumeration the finding itself wrote was not swept: the scrolled-window drawn-row base (BR-5) and the report-behind-other-bytes rule (BR-30) are both still unpinned.
findings:
  - id: new
    severity: Critical
    family: unspecified-event-policy
    title: |
      couch overwrites a still-tracking child's mouse mode whenever its per-Child observation is empty, and nothing re-evaluates the belief
    detail: |
      This is the 5th finding in family `unspecified-event-policy`. Earlier rounds fixed instances (BR-16 the missing re-assert, BR-22 the write, BR-26 the observation). Do NOT fix a fifth site. The rule that covers all of them: couch decides whether to write a terminal-GLOBAL mode from a value it can only ever have observed, so that decision needs a state model -- what the child is believed to hold, what invalidates the belief, and what event re-evaluates it -- not a bool read at paint time.
      Verified at the pinned head: `Screen.mouse` is set only by classify seeing a DECSET in that Child's output (screen.go:428-431), is never queried, and a new pane for a still-running thread starts from `Screen{}` with mouse=false while replay can only re-derive it if the original DECSET is still inside DefaultRingBytes. paintNow (console.go:1043) then writes `?1000;1006h` over a child holding `?1002`, permanently, because nothing re-evaluates: onChunk repaints only on RowDirty/Bell/attention and classify sets rowDirty for alt-screen alone, so a bare `?1000l` also leaves couch's own clicks off indefinitely. The operator filed pair#196 (deps ["#172"]) reporting exactly this in production -- agent-pane drag selection loses its live highlight under couch after some reattachments, correct in standalone pair -- which is the Done-when "a child that did enable tracking still receives its own events unchanged" failing on code this window introduced (EnableMouseClicks does not exist before the base commit).
      Measured prevalence: 3 write sites, 1 governed by the observation; 0 events that re-evaluate it; 6 transition cells pinned, 0 pinning the trigger, because TestMouseModeTransitions calls repaint() by hand.
  - id: new
    severity: Important
    family: manual-verification-unrecorded
    title: |
      The one Done-when with no automatic test is recorded as "it worked", which the plan's own step forbade
    detail: |
      Plan Task 12 Step 2 says "Record the result in `## Log` as a measurement, not as 'it worked'." The Log records "Operator smoke-tested twice on the real stack" -- it does not say which of the three named steps ran (drag-select in the draft, wheel-scroll the agent pane, click a chip) or what each observed, so the record cannot distinguish a verified nvim scroll from a verified chip click. That is load-bearing here: nvim and zellij both set ?1006, so the mainstream child passes while the configuration pair#196 reports broken is invisible to the smoke test. Write the steps, their observed results, and which modes the attached child was holding.
  - id: new
    severity: Minor
    family: unnamed-seam-change
    title: |
      Narrowing Screen.Mouse() to tracking-only silently changes termcmd's wheel policy, and the diff names only couchtty
    detail: |
      This is the 2nd finding in family `unnamed-seam-change`. Do NOT fix this instance alone. The rule: changing the semantics of an exported observation requires enumerating its call sites in the same commit and stating the effect at each. termcmd.appMouseMode (run.go:819) reads Child.Mouse(), so a child that had emitted `?1006h` alone now sends the wheel to `zellij scroll-up` instead of forwarding it. Measured prevalence: 2 consumers, 1 named; no termcmd test pins either direction. No live defect -- such a child receives no reports either way -- but the change was landed as a couchtty fix.
  - id: new
    severity: Minor
    family: parallel-handler-restates-decision
    title: |
      A click on a non-actionable row, or on a chip not yet in the inventory, is silent where Enter explains
    detail: |
      This is the 2nd finding in family `parallel-handler-restates-decision`. Do NOT fix this instance alone. The rule: where the click arm and the key arm diverge, the divergence is DECLARED -- as the Manual flag is, with its reason in a comment -- rather than left as an omission. menu.go:352 returns `next, nil` for both `!ok` and `!menuThreadActionable`, while reduceRootKey:468-470 sets an errorMenuNotice whose own comment says "Silence is what the operator reports as a bug". The inventory-miss case is reachable only from a chip click, since the switcher can only select rows the inventory already holds.
```

---

## Re-review — 2026-09-06T08:22:39-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..647d069071fa8c302c30db8f369fb8a844bbe0a1 |
| command | sdlc close --issue 172 |
| reviewer | claude |
| timestamp | 2026-09-06T08:22:39-07:00 |
| verdict | unknown |

## Review

Failed to authenticate. API Error: 401 OAuth access token has been revoked.

---

## Re-review — 2026-09-06T09:04:17-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..43ba894cde939eee949944f30e35884ec1f9abe5 |
| command | sdlc close --issue 172 |
| reviewer | claude |
| timestamp | 2026-09-06T09:04:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The geometry half of this issue is genuinely good — chip spans come out of the clipping pass, extents come out of the drawn rows after the notice shift, both maps are total, and the click reaches Enter's own `enterOperationFor` rule rather than a restatement. What blocks the boundary is the terminal half, and specifically the finding this round was supposed to close. BR-33's fix gave `Screen` a third state and gated **one** of couch's two writers: `console.go:530` still writes `?1000;1006h` unconditionally at `Run()` start, on the exact production ordering (`dispatchInitialAttach` runs *before* `console.Run()`, `couchcmd/run.go:399-401`), and `TestCouchEnablesItsOwnMouseTracking` + `TestTeardownDisablesMouseTracking` **assert** that write happens under two children whose mode is unknown — I verified by mutation that gating it reddens exactly those two tests, while `TestMouseModeTransitions`' "nothing observed yet: couch stands back" row passes only because it calls `host.Reset()` first. Worse, the belief is read with three states by the write (`couchMayOwnTheMouse`, `:1527`) and two by the forward (`wantsMouse := child.child.Mouse()`, `:1555`), so an unknown-mode child — the reattach case the fix exists for — now receives **zero** mouse bytes rather than a demoted stream (measured through the real input path). Two Done-when bullets are unmet, and BR-5's scrolled case is still unpinned (adding the scroll offset `start` to the extent base leaves the whole `couchtty` suite green; the only test that reaches `start > 0` never looks at extents).

## 1. Strengths

- `reserve.go:161-177` — spans recorded inside the same `appendText` that clips, so a dropped chip contributes no span and a clipped one contributes the columns it drew. `TestAChipClippedToOneColumnIsStillClickable` and `TestChipSpansAreDisplayColumnsNotRunes` construct the cases rather than sweeping; I confirmed the CJK fixture discriminates the column unit, which closes BR-32 properly.
- `menu_render.go:157-165` — extents re-based in `RenderMenuView` after the notice insert, with the reason stated. Mutating the base to the row index reddens `TestPointToActorSpansEveryLineOfAnActor`, so the drawn-row base is load-bearing.
- `menu.go:597-607` — `enterOperationFor` extracted so the click and Enter share one authority instead of a restatement. This is the right shape for the "same path as Return" requirement.
- `core_concepts_contract_test.go:414-422` — excluding the contract file from its own glob (BR-23) is a real fix: the guard now asserts something, and it immediately found three untested types.
- `keys.go:283-308` — the bounded hold (`MaxReport`) is directly pinned by `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard`, which is the #127 hazard the plan flagged.
- `workshop/lessons.md` — eleven substantive entries, including "a belief you can only have OBSERVED needs a third state". AGENTS.md §4 is satisfied.

## 2. Critical findings

**C1 — `console.go:530`: the startup writer is ungated, and two tests pin the ungated behaviour.** (BR-33/BR-26 remaining half; family `unspecified-event-policy`.)
`Run()` writes `hostty.EnableMouseClicks` before `applyLayout`/`paintNow`, with panes already installed by `dispatchInitialAttach`. Measured at head: a probe through `newMouseFixture` reports `startup wrote EnableMouseClicks under an UNKNOWN child: true`. Gating it with `couchMayOwnTheMouse()` reddens `TestCouchEnablesItsOwnMouseTracking` (`console_mouse_test.go:250`) and `TestTeardownDisablesMouseTracking` (`:236`) — i.e. the suite currently asserts the write the fix's own doc comment forbids. Fix sketch: gate `:530` on the same predicate, and give both tests a fixture whose active child has *declared* it holds no tracking (`\x1b[?1000h\x1b[?1000l`, the pattern `TestSwitchingToAChildWithoutTrackingReturnsTheTerminalToCouch` already uses) so they pin the rule instead of its violation.

**C2 — `console.go:1555`: the forward decision reads a two-state belief, so an unknown-mode child receives zero mouse bytes.** (family `unspecified-event-policy`.)
`wantsMouse := child.child.Mouse()` is false for both "declared none" and "never observed". With C1 leaving the terminal in `?1000;1006`, a reattached agent that really holds `?1002` gets **nothing** — strictly worse than the press-only demotion pair#196 reported, and a direct contradiction of "a child that did enable tracking still receives its own events unchanged". Verified: a click at `(7,9)` with an unknown-mode child leaves the child's writes as `"x"` alone. The same two-state read makes an *observed* `?1002h`-without-`?1006h` child (plain vim with `ttymouse=xterm2`, older TUIs) receive nothing at all, permanently — `onMouse:1558` swallows it because couch's own ungated `?1006` write changed the encoding underneath it. Fix sketch: the belief needs one accessor with three values consumed by every site — write, forward, and encoding — and the unknown case has to answer "forward, because couch never took the terminal" rather than "swallow".

## 3. Important findings

**I1 — no event re-evaluates the belief.** (BR-26's fourth half.) `classify` sets `rowDirty` for alt-screen only (`screen.go:433-448`), and `onChunk` repaints on `RowDirty`/`Bell`/attention (`console.go:1119-1140`). A child's bare `?1000l` therefore leaves couch's own clicks off until some unrelated paint. `TestMouseModeTransitions` cannot see this because it calls `con.repaint()` by hand — a test that observes one interleaving the author chose (`ARCH-ORDER`). The transition table needs a trigger column, and `Screen` needs a latched edge for a mouse DECSET the way it has one for the row.

**I2 — the plan artifact still disagrees with the tree in three of BR-28's six places, with no further `## Revisions` entry.** `plan:160` states `(event, hostRows, childWantsMouse)` where four parameters shipped; the "complete disposition table" (`plan:41-56`) still has no `couchOwnsScreen` dimension, still omits the two switcher rows the code carries, and its "anything, elsewhere | yes | forward verbatim" row is now contradicted *twice* (by the switcher case and by the `SGRMouse()` gate); all 41 checkboxes are unticked, none ticked. A fourth site has appeared since: the plan promises `FeedHit` "returns the decoded event alongside the hit", while the code carries it through `Interceptor.mouse` → `Console.mouseHit` mutable fields (`keys.go:236`, `console.go:136`).

**I3 — `console.go:1504-1506` carries a doc block for `childWantsMouse`, a method that does not exist in `couchtty`, glued to the front of `couchMayOwnTheMouse`'s doc.** Third instance of the `orphaned-doc-comment` family; see §7 for the rule rather than the instance.

## 4. Minor findings

- `atlas/couch.md:360` says couch re-asserts "ONLY while no child holds tracking", omitting the unknown state that is the whole point of `7f68fecd`; the same wording is in `workshop/history/issues/000166-*.md`'s new entry.
- `mouseinput.go:51` — `s == ""` is unreachable after the `HasPrefix` check (BR-29, carried over verbatim from `termcmd`).
- `console_mouse_test.go:146` and now `:442` both check only `w[0] == 0x1b`; the weak assertion has been copied into the new encoding test rather than strengthened (BR-30).
- `artifactpath/manifest.go:549,557` — `mouseinput/mouseinput.go` sits inside the `couchcore` block and `couchtty/mouse.go` before `couchtty/menu.go` (BR-31).
- `termcmd/run.go:815-819` — `appMouseMode` still reads `Child.Mouse()` after its semantics narrowed to tracking-only; neither direction is pinned (BR-35).
- `README.md:391` — the new clickable paragraph is inserted mid-thought between the Alt+x sentence and the `Alt+n` paragraph it belongs after.

## 5. Test coverage notes

Mutation results at the pinned head (`./cmd/internal/couchtty/`, only the sandbox-blocked `TestNotificationPTYConformance` failing at baseline):

| mutation | result |
|---|---|
| gate `console.go:530` on `couchMayOwnTheMouse()` | **2 tests red** — they pin the ungated write |
| `index := len(lines) + start` (scroll offset) in `renderRootMenuFrame` | **green** — no extent fixture scrolls |
| `clampExtents` → `return extents` | **green** — documented as unreachable, honestly |
| `index := start + mi` (row-index base) | red ✓ |
| `panic` when `start > 0` | only `TestRenderMenuKeepsSelectedRowVisibleAndBounded` trips it, and it never reads extents |

`TestExtentsNeverPointPastTheDrawnMenu` clips but does not scroll, because its selection is `inventory[0]`. Selecting the *last* actor in that fixture gives `start > 0` for free and closes BR-5's remaining half.

## 6. Architectural notes

- **ARCH-DRY** — pass on the parser promotion and `enterOperationFor`; flag on dispositions: `RouteMouseReport` is documented as the complete table, but the encoding gate at `console.go:1558` is a fourth disposition decided outside it, so the table is no longer the single source it claims to be.
- **ARCH-PURE** — pass on the geometry; flag on the payload path (I2): the decoded event crosses two mutable fields instead of being a return value.
- **ARCH-PURPOSE** — **flag.** BR-33 named the class ("the belief needs a state model — what invalidates it, what re-evaluates it") and the round fixed the site the finding pointed at. C1, C2 and I1 are the unswept siblings of that same enumeration.
- **ARCH-MOCK** — pass. `FakeChild`/`FakeHost` sit at the same seam production uses, and `TestFakeChildConformsToRealChildLifecycle` is the conformance check.
- **ARCH-CONSTRAINTS** — pass. `?1000` not `?1002` for couch's own use is justified by report rate; the hold is bounded by `MaxReport`.
- **ARCH-SECURE** — pass. Reports are parsed into a typed `Event` at the boundary and refused rather than clamped; labels are sanitised before they reach the row.
- **ARCH-ORDER** — **flag**, and it is the through-line: `(mouse, sgrMouse, mouseObserved)` is 3 bools for what the atlas says is one mutually-exclusive tracking state plus an encoding, with no written enumeration of legal combinations; no event re-evaluates it (I1); and `TestMouseModeTransitions` drives the transition by calling `repaint()` itself, so a green run is a sample of size one that cannot report the missing trigger.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `000172-mouse-support-status-bar-and-switcher-plan.md` covering, in one pass rather than site by site:

- **`RouteMouseReport`'s shipped signature and table.** Rewrite "The complete disposition table" (`:41-56`) with the `couchOwnsScreen` dimension and the two switcher rows, correct the "anything, elsewhere | yes | forward verbatim" row for both the switcher case and the `SGRMouse()` gate, and fix `:160`'s three-argument signature to the four that shipped.
- **The `FeedHit` payload.** State that the decoded event travels through `Interceptor.Mouse()` and `Console.mouseHit`, not as a return value, and why.
- **Checkbox state.** All 41 boxes are unticked against work the tree carries; tick what shipped or say in the entry that the plan is archived as-designed.
- **The mouse-belief state model.** The plan has no `(state, event) -> (state, effects)` enumeration for the belief, which is what C1/C2/I1 all fall out of. Write the states (`tracking` / `none` / `unknown`), the writers (`Run`, `paintNow`), the consumers (write, forward, encoding), and the events that re-evaluate it — the enumeration pair#196's own Done-when asked for and which is still the missing artifact.
