---
id: 000283
status: working
deps: [pair#255]
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 1.06
started: 2026-09-18T07:14:08-07:00
flow: {kind: full, provenance: inferred}
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

- [ ] vt records the default: `CursorDefault` zero member; DECSCUSR 0/absent → default; vt tests incl. callback on block→default
- [ ] Endpoint publishes Shape 0 (Blink false); `Frame.Validate` rejects a blinking default; endpoint cursor table at every byte split
- [ ] `cursorEpilogue` emits `ESC[0 q` for Shape 0; endpoint → presenter → parent acceptance over codes 0..6, absent, RIS
- [ ] Qualifier literals renumbered; consumer sweep; PAIR_PATCHES + atlas; `make test` + `go test ./...`
- [ ] Operator smoke: Ghostty `cursor-style = bar`, plain shell under couch shows a bar

## Log

### 2026-09-17

Filed from #262 M2 (see its `## Log`, "M2 input: caret blink, and a DECSCUSR
fidelity finding").

## Revisions

- 2026-09-18: The estimate went from 0.98 to 1.06 after change-code's estimate-quality notes.
  - The impl values now follow the stated midpoint ×0.4 rule; the earlier split-by-size numbers didn't match that note.
  - The design buffer went from 0.15 to 0.30, per v2.1 Step 6, because the ×0.2 spec discount wasn't applied.
  - The plan itself is unchanged.
