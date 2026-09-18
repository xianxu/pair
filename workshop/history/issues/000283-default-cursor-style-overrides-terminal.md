---
id: 000283
status: done
deps: [pair#255]
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 1.06
started: 2026-09-18T07:14:08-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 0.60
---

# A child's default cursor style overrides the terminal's configured cursor

## Problem

A child that never sets a cursor style, or resets it with `ESC[0 q`, makes pair
send `ESC[1 q` (an explicit blinking block) on every frame. That OVERRIDES the
parent terminal's configured default: Ghostty's `cursor-style` and
`cursor-style-blink`, or any terminal's equivalent. Before #255, the child's
`ESC[0 q`, or its silence, reached the terminal as "use your default".

The default cannot survive the pipeline:

- the vendored DECSCUSR handler maps an absent parameter and `0` to `n = 1`,
  a blinking block (`third_party/vt/handlers.go:848-855`);
- the endpoint maps `Shape: int(cur.Style)+1` (`cmd/internal/terminal/endpoint.go:228`),
  so `Frame.Cursor.Shape` is never 0, although `frame.go:15` documents
  *"0/default"*;
- `cursorEpilogue` (`render.go`) therefore never emits `ESC[0 q`.

Found in #262 M2, while classifying the per-frame DECSCUSR. The flicker fix does
not depend on it.

## Spec

Carry "terminal default" through as a state distinct from an explicit block:

- the emulator records "never set, or reset to 0";
- the endpoint publishes Shape 0 for it;
- `cursorEpilogue` emits `ESC[0 q`.

RIS and a fresh endpoint start at the default. Explicit `ESC[1 q` / `ESC[2 q`
stay explicit. Needs a decision on how much of `third_party/vt` to touch, versus
tracking it at the endpoint through the `CursorStyle` callback. The callback
fires only on CHANGE, so a `0` sent while the cursor is already a blinking block
is invisible to it.

## Done when

- A child that never sets a style, and a child that sends `ESC[0 q`, both
  produce `ESC[0 q` at the parent. An explicit `ESC[1 q` still produces
  `ESC[1 q`. Tested at the endpoint → renderer seam.
- Operator smoke: with Ghostty `cursor-style = bar`, a plain shell under couch
  shows a bar.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.3 impl=0.14
item: smaller-go-module   design=0.1 impl=0.14
item: atlas-docs          design=0.05 impl=0.05
item: milestone-review    design=0.0 impl=0.14
design-buffer: 0.30
total: 1.06
```

- The first `smaller-go-module` covers the vt default, the endpoint mapping, `Frame.Validate` and `cursorEpilogue`. Its design is at the top of the range with no ×0.2 spec discount, because the vt-vs-endpoint decision was made inside this claim window. So v2.1's full +30% buffer applies, not the halved one.
- The second covers the endpoint → parent acceptance test and the qualifier literal renumbering.
- `impl=` values are v2 table midpoints ×0.4 (v3.1): 0.35→0.14 and 0.125→0.05. Familiarity is 1.0: #262 M2 worked in this same renderer and endpoint.
- The base case is one close-review round. The Ghostty smoke has no primitive in the vocabulary, so it is not itemized.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000283-default-cursor-style-overrides-terminal-plan.md`. Decision: fix it
in the pair-owned vt fork — `CursorDefault` becomes the zero `CursorStyle`, with the
explicit styles renumbered onto DECSCUSR shape families 1/2/3. The callback route
can't see RIS or a `0` sent over a blinking block.

- [x] vt records the default: `CursorDefault` zero member; DECSCUSR 0/absent → default; vt tests incl. callback on block→default
- [x] Endpoint publishes Shape 0 (Blink false); `Frame.Validate` rejects a blinking default; endpoint cursor table at every byte split
- [x] `cursorEpilogue` emits `ESC[0 q` for Shape 0; endpoint → presenter → parent acceptance over codes 0..6, absent, RIS
- [x] Qualifier literals renumbered; consumer sweep; PAIR_PATCHES + atlas; `make test` + `go test ./...`
- [x] Operator smoke: Ghostty `cursor-style = bar`, couch's panel (switcher) caret shows a bar; the pane path is covered by the endpoint → parent acceptance test (see Revisions)

## Log

### 2026-09-17

Filed from #262 M2 (see its `## Log`, "M2 input: caret blink, and a DECSCUSR
fidelity finding").

### 2026-09-18
- 2026-09-18: closed — Tests: vt go test + -race green (pair_cursor_style_test: DECSCUSR 0..6/absent/RIS/>6, callback on block->default). go test ./... -count=1 green on re-run; the first run had a load-sensitive wrapcmd TestNotificationBrokerBeforeExecAndCleanup flake that passed 10/10 isolated and 3/3 as a package. make -k test: only the known pre-existing test-changelog fails. Acceptance: TestChildCursorStyleReachesParentVerbatim (real endpoint -> Presenter.Select -> ttyio.Fake parent) shows default arriving as ESC[0 q and explicit 1..6 verbatim. Qualifier probe: 84 pass / 0 fail / 6 not-covered. Operator smoke in Ghostty 1.3.1 with cursor-style=bar: couch switcher filter caret shows a bar (was a steady block); operator judged passed; the smoke covered the panel caret, and the pane path is covered by the acceptance test.; review verdict: SHIP

**Design.** Fix it in vt, not at the endpoint. The `CursorStyle` callback fires
only on change, RIS bypasses it, and the endpoint has read `Emulator.Cursor()`
directly since #255 M2 BR8. `CursorDefault` is the zero `CursorStyle`, and the
explicit styles are renumbered onto DECSCUSR shape families 1/2/3. This was
chosen over a `StyleSet bool` for ARCH-ORDER: 8 representable states instead of
12, for 7 legal ones. It also makes the endpoint mapping the identity, and the
`+1` there was the site that lost the default. `Frame.Validate` refuses Shape 0
with Blink, so there is one representation of "default".

**change-code flow.** The first run inferred `quick`. The plan file was named
`000283-default-cursor-style-plan.md`, but inference looks for
`<issue-file-slug>-plan.md`. After renaming the plan, the flow was re-inferred as
`full`. Plan-quality passed in round 1 with 2 Minor notes. Estimate-quality
flagged two derivation errors, which were corrected (see Revisions).

**Implementation (TDD, each seam red → green):**
- vt `pair_cursor_style_test.go` covers every code 0..6, an absent parameter,
  RIS, an ignored code above 6, and a callback on block → default.
  `TestPairCursorStyleRejectsInvalid` now proves an invalid code leaves an
  explicit underline intact; a fresh value would coincide with the default.
- The endpoint cursor table replays each case at every byte split. It adds
  never-set, zero-after-blinking-block (the callback-invisible case),
  absent-after-underline, explicit block, and unknown-ignored. The
  reset/alternate-entry/reset-held literals had encoded the bug, and now expect
  the default.
- `TestChildCursorStyleReachesParentVerbatim` (`presenter_test.go`) is the
  Done-when seam: real endpoint → `Presenter.Select` → `ttyio.Fake` parent, the
  final DECSCUSR. It uses `presenterFixture`, whose `t.Cleanup` releases the
  presenter and closes the endpoint. That answers plan-gate PQ-2 (goroutine
  extent).
- Qualifier probe: 3 literals renumbered (restore 1→2, buffer 2→3, style 2→3),
  exactly the predicted set. Result: 84 pass, 0 fail, 6 not-covered
  (`qualified=false` is the tracked integration obligations, unchanged). The RIS
  literal `"0,true"` keeps its text, but 0 now means default. The case table
  has a legend.

**Consequence.** Couch's panel caret (`couchtty/console_menu.go:216`, `Cursor{Visible:
true}`) went from `ESC[2 q` (a steady block) to `ESC[0 q`, the operator's
configured cursor. This matches the documented "0/default" meaning. The xterm
oracle (@xterm/headless 5.5) maps 0 to 1 and cannot observe any of this, so the
Ghostty smoke is the only live conformance check.

**Verification.**
- vt: `go test ./...` and `-race` green.
- `go test ./... -count=1` green (5-var retention scrub).
- `make -k test`: the only failure is the known pre-existing `test-changelog`.
- The first full Go run had one failure, `wrapcmd`
  `TestNotificationBrokerBeforeExecAndCleanup` ("startup hook lost"). It did not
  recur: 10/10 isolated, 3/3 as a package, and the whole-suite re-run was green.
- The flake is unrelated to this diff. The endpoint is the only production writer
  of `Cursor.Blink`, and it always writes the canonical form; the console menu
  sets Shape 0 with Blink false. So the new `Validate` clause can't reject a real
  frame.
- The flake is a load-sensitive startup race: under full-suite load the child
  sends and exits before the broker flushes.

**Operator smoke (passed).** Ghostty 1.3.1 with `cursor-style = bar`, config
reloaded, couch PID 48712 built from `1949fdfd`.
- The switcher's `filter:` caret is a bar. Before, pair forced it to `ESC[2 q`,
  a steady block.
- The start-thread `path` caret sits inside the reverse-video selected row,
  where a thin bar is hard to see against the highlight. The operator accepted
  this as a visibility artifact, not a style failure.
- Ghostty 1.3.1's `setCursorStyle(.default)`
  (`src/termio/stream_handler.zig`) applies the configured style and turns blink
  on, and `changeConfig` re-applies it immediately when the cursor is at the
  default. That confirms `ESC[0 q` is the right wire for "use your cursor".
- A false start: `ghostty +show-config` at first showed no `cursor-style`,
  because the operator had reverted the line at that moment.
- Scope of the smoke: it covered couch's own panel (switcher) caret, not a
  plain-shell pane. The operator judged the change working and the smoke
  passed. The pane path (child → endpoint → presenter → parent) is covered by
  `TestChildCursorStyleReachesParentVerbatim`, which drives the production
  endpoint and presenter.

## Revisions

- 2026-09-18: The estimate went from 0.98 to 1.06 after change-code's estimate-quality notes.
  - The impl values now follow the stated midpoint ×0.4 rule; the earlier split-by-size numbers didn't match that note.
  - The design buffer went from 0.15 to 0.30, per v2.1 Step 6, because the ×0.2 spec discount wasn't applied.
  - The plan itself is unchanged.
- 2026-09-18 (close round 1, Minor CR findings): corrected the smoke row to what was
  observed.
  - The operator's Ghostty smoke checked couch's own panel caret (the switcher
    `filter:` line), not a plain-shell pane as the Done-when names. The operator
    judged it passed. The pane path is covered by
    `TestChildCursorStyleReachesParentVerbatim`, which drives the production
    endpoint and presenter.
  - The Done-when text is left as filed.
  - The plan's Task 4 predicted "68 executable cases". The qualifier actually
    measured 84 pass, 0 fail, 6 not-covered, the same as before this change
    apart from the 3 renumbered literals.
  - vt's default reports its zero-value blink. That is now documented at every
    surface exposing it: `Cursor.Steady`, the `CursorStyle` callback, and the
    qualifier legend.
