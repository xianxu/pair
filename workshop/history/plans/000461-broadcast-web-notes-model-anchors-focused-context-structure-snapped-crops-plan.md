# broadcast-web notes model Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The real `notes.Model` of #457's seam: stable anchors, focused context and structure-snapped, clustered crops, pure over `recording.State`.

**Architecture:** A zero-value-usable `notes.Structural` in `cmd/internal/webcast/notes`, beside #457's `FakeModel`. Three pure parts in their own files: selectors (make and match a small CSS subset), landmarks (the container rule table), and crops (pad, snap per axis, cluster, clip, hint). It runs #457's `RunModelContract` plus model-specific tests on the fixtures (`stress`, `app`, `blog`, `segments`). No interface change; the seam stays #457's.

**Tech Stack:** Go, the `recording` package (reducer state, `Box`, `NoteAt`, `Builder`, fixtures).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Structural` | `cmd/internal/webcast/notes/structural.go` | new |
| `selectorFor` / `matchSelector` | `cmd/internal/webcast/notes/selector.go` | new |
| `landmark` (container rule table) | `cmd/internal/webcast/notes/landmark.go` | new |
| `crops` (pad, snap, cluster, clip) | `cmd/internal/webcast/notes/crop.go` | new |
| `liveNotes`, `trim` (shared with the fake) | `cmd/internal/webcast/notes/fake.go` | modified (fake's `Live` calls `liveNotes`) |

- **Structural** — implements `notes.Model`. Fields: `Margin` (CSS px around the notes; zero = 48). Everything else is a named constant.
  - **Relationships:** consumed by `query.Surface` (#462) and, later, the DOM submission (wave 3) through the `notes.Model` interface; holds no state.
  - **DRY rationale:** `Live` and the 200-character `trim` are the fake's rules; the real model calls the same helpers, so "notes on the current document" has one definition (`Seg >= DocSeg`).
- **Selector** — a stable CSS selector in a small subset, generated and matched by the same file so the two can't drift:
  - one step: `#id` (a stable, unique id), `[data-testid="v"]` (also `data-test`, `data-cy`; unique), else `tag:nth-of-type(k)`;
  - steps join with ` > ` from the nearest ancestor that has a stable id/test attribute, else from the tree root's element;
  - a node inside a shadow root or an iframe document: the host element's selector, ` >>> `, then the path inside (the deep combinator is ours, not CSS).
  - A stable id: matches `^[A-Za-z][\w-]*$`, has no run of 4+ digits (generated ids), and is unique in its tree.
  - Resolve by selector: candidates matching the selector in the top document; one whose trimmed text equals `at.Text` wins; else a single candidate; else unresolved (ambiguous). A selector outside the subset (the fixtures' `h1~Reports`) matches nothing: those notes resolve by node id.
  - **Future extensions:** an `aria-label` / `name` step when re-renders that keep no id prove common.
- **Landmark** — the nearest meaningful ancestor-or-self of an element (crossing shadow hosts and frame owners, stopping below `body`):
  - tags: `header nav main aside footer form section article dialog fieldset figure table tr ul ol dl li details menu search`;
  - `role`: `banner navigation main complementary contentinfo form search region dialog alertdialog article list listitem table row grid group toolbar menu menubar tablist tabpanel figure`; `aria-modal="true"`;
  - layout boxes, by class, for elements with element children only (a bordered `input` is not a card): a class token starting with `card panel modal dialog popover toolbar navbar sidebar drawer tile`, or a `rounded*` token together with `border`/`shadow*`.
  - None found: the parent element (body at most), marked **not meaningful**, so crops don't snap to it. Self counts only when self is a container (`ol`, `nav`), so a heading's landmark is its `header`.
  - **Future extensions:** the rule table is the place per-app conventions widen; `Hints` covers crops.
- **Context** — `{Node, Landmark, Text, Box}`: `Node` is the resolved element (by `Resolve`, so a re-rendered element still gives context), `Text` the landmark's whitespace-collapsed text in a 500-character window centred on the anchor's own text (so the anchor is always in it), `Box` the landmark's box.
- **Crops** — for a submission's anchors:
  1. each anchor's box: `at.Box` as given (a viewer-made note's box is the viewer's, #457 10-12), else `st.Boxes[node]`; neither → error.
  2. per anchor: the box padded by `Margin`; if its landmark is meaningful and has a box, snap **per axis**: x always widens to the landmark's horizontal extent (a header or nav goes full width); y widens to the landmark's vertical extent only when that is at most half the viewport height (400 px at 800), else the notes plus margin stay as they are (#455's pane rule: full width, height limited to the notes).
  3. cluster: merge any two crops whose gap on both axes is at most `2*Margin` (single linkage, repeated until stable), so far-apart notes get one crop each.
  4. clip to the page: x and y at least 0, right and bottom edges within the root element's box (viewport width when unknown).
  5. `hints.Crop(st.URL, crop)` per crop, when `hints != nil`.
  - **Future extensions:** region notes (`Note.Region`): `Crops` takes `[]NoteAt`, which has no region. Reported to the TL as a seam question; not built here.

### Integration points

None new. The model is pure over `recording.State`; the reducer and fixtures are #457's. The integration test surface is `RunModelContract` (the seam) plus fixture tests; the recorder's real recordings replace the fixtures when #458 lands (project integration guard).

### Lifecycle and size (ARCH-FUNERAL, ARCH-CONSTRAINTS)

Creates nothing durable: every value is computed per call and dies with it. Cost per call is O(nodes) for selector matching and the landmark walk; clustering is O(crops²) over a submission's few notes.

## Tasks

### Task 1: Selectors

**Files:** Create `notes/selector.go`, `notes/selector_test.go`.

- [x] Tests first: `#id` for a stable unique id; a generated id (`:r1:`, `row-12345`) and a duplicated id fall back to a path; `[data-testid=…]`; `tag:nth-of-type(k)` path from the nearest stable ancestor; a node in an open shadow root and in an iframe document (` >>> `); round trip: every element of the `blog` and `stress` fixtures at their last time matches its own selector uniquely; a non-subset selector matches nothing.
- [x] Implement `selectorFor(st, n) string` and `matchSelector(st, sel) []*DOMNode`.
- [x] `go test ./cmd/internal/webcast/notes`; commit `#461: selectors`.

### Task 2: Landmarks and context

**Files:** Create `notes/landmark.go`, `notes/landmark_test.go`.

- [x] Tests first, on the fixtures at each note's time: `blog-n1` (h1) → its `header`; `blog-n2` (img) → the `li`; `blog-n3` (ol) → itself; `stress-n2` (x-closed) and `stress-n1` → `section.side`; `app-n3` (h1 in body) → `body`, not meaningful. Builder cases: a role, `aria-modal`, a Tailwind card (`rounded-lg shadow`) container, and a bordered `input` that is not a card.
- [x] Context text window: a long landmark keeps the anchor's text inside the 500-character window.
- [x] Implement `landmark(n) (*DOMNode, bool)` and the window.
- [x] Commit `#461: landmarks and focused context`.

### Task 3: Crops

**Files:** Create `notes/crop.go`, `notes/crop_test.go`.

- [x] Tests first: `blog-n1` crop is full width (x 0, w 1280) and holds the header; `blog-n2` crop's x range is the `li`'s and its height is the image plus margins, not the `li`'s 1218 px; `blog-n1`+`blog-n3` give two crops, `blog-n1`+`blog-n2` two; two adjacent paragraphs give one; crops clip at x 0 and the page's right edge; a note with a given `at.Box` different from `st.Boxes` uses the given one; no box → error; a hint replaces each crop.
- [x] Implement.
- [x] Commit `#461: structure-snapped, clustered crops`.

### Task 4: The model, contract, fixtures, docs

**Files:** Create `notes/structural.go`, `notes/structural_test.go`; modify `notes/fake.go` (`liveNotes`), `atlas/webcast-format.md` (a "Notes model (#461)" section: anchor, context and crop shapes and rules; the #455-compatible note shape), and `atlas/index.md` if needed.

- [x] `RunModelContract(t, func() Model { return Structural{} })`.
- [x] Every fixture note resolves at its time; resolve by selector after its element is replaced by a re-render (same selector, new id); ambiguity → unresolved.
- [x] Commit `#461: Structural model; atlas`.
- [x] Full verification: `go test ./cmd/internal/webcast/...`, then the full `make -k test` per repo practice.

## Revisions

- 2026-10-12 — built. Clustering merges crops whose gap (on the axis they are furthest apart on) is less than one
  margin, not two margins on both axes: padded crops already carry a margin each, so notes within about three margins
  share a crop. Clipping widens the page to every note's own box, so a crop never cuts its note. Crops resolve each
  anchor first, so a re-rendered element still snaps to its landmark.
- 2026-10-12 — close review round 1 (BR-1): a test-attribute value containing `>` made a selector the matcher
  splits apart. Rule, not site: `uniqueStep` skips values with a quote or `>`, and `Anchor` drops any selector that
  doesn't round-trip through `matchSelector`. The matcher also steps into template content, and an anchor with no
  box of its own takes the layout box of the element it resolves to now.
