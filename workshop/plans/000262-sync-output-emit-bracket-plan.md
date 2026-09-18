# Synchronized-output emit bracket (#262 M1) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every frame the presenter paints reaches the parent terminal as one
synchronized-output (DECSET 2026) bracket. The terminal then draws only the
finished frame, never the whole-screen erase-and-redraw in between. That redraw
is the #262 flicker.

**Architecture:** The bracket is part of frame serialization, so it lives in the
two pure renderers, `Render` (`render.go`) and `HistoryRender.Emit`
(`history_render.go`). Each opens with `ESC[?2026h` and closes with `ESC[?2026l`
around the whole frame, alt-screen switches included. The presenter changes in
one place: `parentReleaseControls` closes sync, so a frame write that failed
after opening the bracket cannot leave the parent frozen past release. The cursor
epilogue both renderers duplicate becomes one helper beside the bracket
constants.

**Tech Stack:** Go; `cmd/internal/terminal`; `ttyio.Fake` as the stateful parent
double; the `@xterm/headless` oracle (`tests/terminal-oracle`) and the native
zellij oracle (`tests/terminal-oracle/discovery`) as independent terminal
models.

**Why (evidence):** see the 2026-09-17 `## Log` entries in
`workshop/issues/000262-diagnose-input-screen-flicker.md`. On 80×24, one
keystroke makes `Emit` write 2137 B (`ESC[23L`, 24× `EL2`, every row repainted),
against 2156 B for a first paint.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `syncBegin` / `syncEnd` | `cmd/internal/terminal/render.go` | new |
| `cursorEpilogue` | `cmd/internal/terminal/render.go` | new |
| `Render` | `cmd/internal/terminal/render.go` | modified |
| `HistoryRender.Emit` | `cmd/internal/terminal/history_render.go` | modified |
| `parentReleaseControls` | `cmd/internal/terminal/presenter.go` | modified |

- **`syncBegin` / `syncEnd`**: the DECSET/DECRST 2026 strings. They are used by
  both renderers, by `parentReleaseControls`, and by every bracket assertion.
  One constant means no site can open a bracket that another site's literal
  fails to close.
- **`cursorEpilogue(c Cursor) string`**: CUP to the cursor, DECSCUSR (shape and
  blink), and `ESC[?25h` when visible. Today this is duplicated verbatim at
  `render.go:80-92` and `history_render.go:403-415`. Extracting it gives M2's
  DECSCUSR decision one site, and M2 can widen it to `(prev, next Cursor)`
  locally.
- **`Render`** (panel frames): `syncBegin + <existing bytes> + syncEnd`. A no-op
  frame still returns `nil`.
- **`HistoryRender.Emit`** (child-pane frames, streamed in chunks of up to
  64 KiB): `syncBegin` is the first `add`, `syncEnd` the last, so one bracket
  spans every chunk. `!p.dirty` still writes nothing. On an alt-screen switch,
  `packet()` flushes the pending `syncBegin` as its own write ahead of
  `ESC[?1049h`/`l`. That is required: `Presenter.write` tracks alt-screen
  ownership by EXACT match on those packets (`presenter.go:196-201`), so they
  must stay whole.
- **`parentReleaseControls`**: `syncEnd` goes right after the CAN/ST abort. CAN
  cancels a partially written `syncBegin`; `syncEnd` closes a complete one.
  Release runs whenever `parentTouched` is set, including after `p.fail`
  (`presenter.go:130-131`).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Presenter.write` over `ttyio.Writer` | `cmd/internal/terminal/presenter.go` | unchanged | the parent terminal |
| `ttyio.Fake` | `cmd/internal/ttyio/fake.go` | reused | the parent terminal, in tests |
| `recordingParent` (test-only) | `cmd/internal/terminal/presenter_test.go` | new | `ttyio.Fake`, keeping per-write boundaries |
| xterm-headless oracle | `tests/terminal-oracle/driver.cjs` | reused | an independent xterm model |
| native zellij oracle | `tests/terminal-oracle/discovery/zellij_oracle.py` | reused | a disposable real zellij |

- **Parent terminal (ARCH-MOCK).** The seam is `ttyio.Writer`. `ttyio.Fake` is
  the stateful double: accepted bytes plus scripted per-write outcomes.
  Production and tests share `Presenter.write`. `recordingParent` wraps a `Fake`
  to record where each write starts, so the failure sweep can cut any write of a
  multi-write frame.
- **What each oracle can prove.** `@xterm/headless` 5.5.0 does NOT implement mode
  2026; the package contains no occurrence of it. A green xterm run therefore
  proves the *ignoring-terminal* path: the bracket leaves the end state
  unchanged on a terminal that does not know the mode. That is exactly the
  unconditional-emit assumption. The native zellij oracle
  (`PAIR_TERMINAL_NATIVE=1`, sandbox off) is the automated model of `pair term`'s
  real parent, and is run once. The *honouring* path, Ghostty under couch, is
  covered only by the operator smoke (Task 5).

### Architecture principles applied

- **ARCH-PURE:** the bracket lives in the pure renderers, tested byte-for-byte
  without IO. The presenter change is one constant in its release sequence.
- **ARCH-DRY:** one set of bracket constants, and one cursor epilogue instead of
  two copies.
- **ARCH-ORDER:** the bracket holds no state between frames. It opens and closes
  inside one synchronous paint. The only cross-event state is at the parent:
  - *closed*: steady state;
  - *open*: inside a paint;
  - *open-orphaned*: a write failed after `syncBegin` was accepted, and the
    presenter is `Failed`.

  Transitions:
  - paint completes → *closed*;
  - write fails → *open-orphaned* (terminal, via `p.fail`);
  - release → *closed* (`syncEnd` in the release controls).

  The mishandled event is a failure at an arbitrary offset of an arbitrary
  write. Task 3 sweeps accepted-prefix offsets across EVERY write of a frame:
  single-write frames, alt-screen frames (`syncBegin` | `?1049h` | body), and
  multi-chunk frames. The fake scripts the cut, so any failing offset reproduces
  exactly.
- **ARCH-CONSTRAINTS:** this is the keystroke path.
  - Cost: +16 B per painted frame, plus one extra write only on alt-screen
    switches.
  - Sync-open time: bounded by one paint, with each write bounded by
    `WriteTimeout` (2s). The presenter fails rather than retrying.
  - An orphaned bracket is bounded by the terminal's own sync timeout until
    release.
  - Not gated on a capability query: terminals without mode 2026 ignore it, and
    a DECRQM confirm would be an async-stale belief (ariadne#232).
- **ARCH-SECURE:** N/A. It emits constant sequences and parses no new input.
- **ARCH-FUNERAL:** creates nothing durable (constants in a byte stream).
- **ARCH-PURPOSE:** this fixes the symptom where it lives: couch writes straight
  to Ghostty. Ingest-side sync already exists (#255 M2), so with M1 the pipeline
  is atomic end to end. The row diff reduces bytes, not flicker, and has a named
  trigger (#120 or a measurement). For Task 4, the class is *prose presenting
  the pre-#255 reserved-row machinery as live*. It is swept, not just the one
  sentence.

---

## Task 1: Bracket constants, shared cursor epilogue, bracketed `Render`

**Files:** `cmd/internal/terminal/render.go`; test `render_test.go`.

- [ ] **Red.** Add `assertOneBracket(t, label, wire)`: `wire` starts with
  `syncBegin`, ends with `syncEnd`, and contains exactly one of each. Add
  `TestRenderBracketsEveryFrameAndNeverANoop`. It asserts `assertOneBracket` for a
  first paint, a one-cell diff and a cursor-only change, and asserts that
  `Render(f, f)` returns `nil`. Run
  `go test ./cmd/internal/terminal/ -run TestRenderBrackets`; it fails to compile
  (`syncBegin` undefined).
- [ ] **Green.** Add the constants and `cursorEpilogue` (lifted verbatim from
  `render.go:80-92`, plus CUP). `Render` writes `syncBegin` before the preamble,
  and `cursorEpilogue(next.Cursor) + syncEnd` in place of `:80-92`.
- [ ] **Package green.** Run `go test ./cmd/internal/terminal/`. Update any
  byte-exact `Render` expectation to include the bracket. Never weaken one to
  `Contains`.
- [ ] **Commit:** `#262 M1: bracket panel frames in synchronized output`.

## Task 2: Bracketed `HistoryRender.Emit`

**Files:** `cmd/internal/terminal/history_render.go`; test
`history_render_test.go`.

- [ ] **Red.**
  - Update `TestHistoryRendererAltTransitionsAreWholePackets`, for both enter and
    leave: `packets[0] == syncBegin`, `packets[1]` is the WHOLE `?1049h`/`l`
    packet, and the last packet ends with `syncEnd`. The leave half keeps its
    `\x1b[3J` check.
  - Add `TestHistoryEmitBracketsEveryFrame`. It asserts `assertOneBracket` on
    reset, a history append (the `source+"\r\nNEXT"` shape from
    `TestHistoryWireIndependentOracle`) and a steady one-cell edit, and that an
    idle frame emits zero bytes.
  - Add `TestHistoryEmitBracketSpansChunks`: an alt-screen endpoint of 200×100,
    each cell a distinct SGR colour, enough that the body spans ≥3 chunks, with
    the test asserting that fixture property itself. The joined output holds one
    bracket, and the final chunk ends with `syncEnd`.
- [ ] **Green.** `e.add(syncBegin)` immediately after
  `e := historyEmitter{write: write}`, before the alt packets. Replace
  `:403-415` with `e.add(cursorEpilogue(p.next.Cursor))`, then `e.add(syncEnd)`
  before the final `e.flush()`.
- [ ] **Oracles.**
  - Run `PAIR_TERMINAL_ORACLE=1 go test ./cmd/internal/terminal/`: every xterm
    oracle suite is green. This proves the end state is unchanged on a terminal
    that ignores 2026, per the Integration points note.
  - Once, with the sandbox off:
    `PAIR_TERMINAL_NATIVE=1 go test ./cmd/internal/terminal/ -run 'Native'`,
    the native zellij oracle for `pair term`'s parent.
  - Record both results in `## Log`. If zellij cannot run here, record that
    rather than skipping silently.
- [ ] **Commit:** `#262 M1: bracket child-pane frames in synchronized output`.

## Task 3: Release closes an orphaned bracket, for a cut in any write

**Files:** `cmd/internal/terminal/presenter.go:263-277`; test
`presenter_test.go`.

- [ ] **Red.**
  - `recordingParent`: implements `ttyio.Writer` and delegates to a `ttyio.Fake`.
    It appends each call's input length to a slice, which lets a probe learn a
    frame's write boundaries.
  - `assertStreamBracketed(t, label, stream)` scans the parent stream and
    requires:
    - no nested `syncBegin`;
    - every frame preamble (`ESC[?25l`) falls inside a bracket;
    - the stream does not END with sync open.

    A redundant `syncEnd`, such as release's, is allowed.
  - `TestPresenterStreamIsBracketedAcrossPaintsAndRelease`: select, feed, present,
    flush, release. It asserts ≥2 brackets and `assertStreamBracketed`.
  - `TestPresenterReleaseClosesSyncAfterAnyCutWrite`, table-driven over three
    frame shapes:
    - an 8×5 normal first paint (one frame write);
    - an 8×5 alt-screen first paint (`syncBegin` | `?1049h` | body);
    - a 200×100 alt-screen first paint (multiple 64 KiB body chunks).

    Per shape, a probe presenter on `recordingParent` records the write lengths.
    The first write is the mode delta, because an empty write never reaches the
    writer (`transport.go:144`); assert that it is. Cut offsets:
    - every offset for the two small shapes;
    - for the large shape, every write boundary ±2, the first and last 32 bytes,
      and a stride of 4093 elsewhere.

    For each offset, a fresh presenter:
    - fully accepts every write before the one containing the cut;
    - gives the cut write `{Limit: k, Err}`, or `{ZeroProgress: true, Err}` when
      `k == 0`;
    - requires `Select` to fail, then releases.

    Assert `assertStreamBracketed` on the parent bytes. Run it: the cases past a
    complete `syncBegin` fail with "stream ends with synchronized output open".
- [ ] **Green.** In `parentReleaseControls`, add `controls += syncEnd` right after
  `"\x18\x1b\\"`. In its doc comment, add *synchronized output* to what Render
  owns.
- [ ] **Package green.** Run `go test ./cmd/internal/terminal/`, updating any
  byte-exact release expectation.
- [ ] **Commit:** `#262 M1: release closes a bracket a failed frame left open`.

## Task 4: Retire prose that presents the pre-#255 reserved-row machinery as live

The class: code comments and atlas prose describing `hostty.Reservation`'s
painters, `ptychild.Screen`'s paint gating, or the deleted
`paneWriter`/`writeOwn`/`flushOwed` door as the production path. #255 M3 moved
all chrome to `terminal.Presenter.UpdateChrome`.

- [ ] **Enumerate, then fix every site.** Sites known at plan time:
  1. `cmd/internal/hostty/reserve.go:13-18`. Two false sentences: *"shared by two
     consumers — couch … `pair term` …"*, whose only caller is
     `cmd/probes/couchnestedrows`, and *"`\x1b[r` lives here and only here"*,
     which is also emitted at `terminal/render.go:35`, `history_render.go:313`,
     `:359` and `presenter.go:275`. Rewrite both:
     - the painters now serve that probe;
     - the presenter is the parent's sole production writer, and its renderers
       and release controls own the region reset (#262).

     `control.go:24-27` is true as written; leave it.
  2. `atlas/architecture.md` ≈`:538-610`: the console-write gating paragraph
     (`SafeToPaint`), the row-dirty debt, the `paneWriter` compile-error door, and
     the `Screen` scanner paragraph. For each, grep its named symbols in
     non-test production code. Where the mechanism is gone, condense the
     paragraphs into one short note:
     - since #255 M3, both hosts compose chrome through the presenter
       (→ `atlas/terminal.md`);
     - the pre-#255 lessons live in #199's history.

     Keep any paragraph still true, such as the strip-mutation test if
     `stripmutation_test.go` exists.
  3. The #262 issue itself: the `## Spec` sentence and the `## Done when` bullet
     cite `control.go:27`. Retarget both to `reserve.go:13-18`. Log entries are
     history and are left as written, since the Revisions note records the
     correction.

  Then sweep for siblings not listed:

  ```
  grep -rn 'SafeToPaint\|TakeRowDirty\|ReserveAndPaint\|paneWriter\|only here' atlas cmd
  ```

  Fix each hit that presents the old path as live, and record the sweep's final
  empty-of-stale result in `## Log`.
- [ ] `go build ./... && go test ./cmd/internal/hostty/` passes.
- [ ] **Commit:** `#262 M1: retire prose presenting pre-#255 reserved-row painters as live`.

## Task 5: Verify, atlas, operator smoke

- [ ] **Full suite.**
  - Scrub the retention-owner env group (memory: `make test` env leak). Run
    `make test > $TMPDIR/mt262.log 2>&1` and read the tail.
  - The pty-child packages (`couchtty`, `termcmd`) need the sandbox off.
  - `parley_harness_golden` 7/7 is a known pre-existing failure.
  - Record the result.
- [ ] **Atlas.** In `atlas/terminal.md`'s `Presenter` paragraph, add one sentence:
  every presented frame is one synchronized-output (2026) bracket, opened and
  closed by the renderers, and release closes a bracket a failed write left open
  (#262). `atlas/couch.md:358` ("synchronized output … are virtual terminal
  state") is about ingest and remains true.
- [ ] **Install and hand to the operator.** Install the binary couch runs. Ask
  the operator to smoke:
  1. Under couch, typing in the draft beside a static agent pane: is the global
     flicker gone?
  2. Under couch, a quiet draft while the agent works: same question.
  3. `pair term` under plain zellij: any in-pane flicker? Record what that says
     about zellij and 2026.
  4. Does the caret blink regularly while typing? This is M2's DECSCUSR input.

  Record the answers in `## Log`. M1 is not done until the operator confirms.
- [ ] **Close the milestone:** `sdlc milestone-close --issue 262 --milestone M1`.

## Revisions

### 2026-09-17 — plan-quality round 1

- **PQ-1:** Task 4 is retargeted from `control.go:27`, which is true as written,
  to `reserve.go:13-18`. It is widened to the class, which includes the atlas
  paragraphs describing the deleted door, and the issue's Spec and Done-when
  citations are corrected.
- **PQ-2:** the plan now states that the xterm oracle does not implement 2026
  and so proves only the ignoring path, and adds one native zellij oracle run.
- **PQ-3:** the failure sweep now cuts every write of three frame shapes
  (single-write, alt-screen, multi-chunk) via `recordingParent`.
- **PQ-4:** verbatim test bodies and the reproduced implementation are replaced
  by per-test specifications.

### 2026-09-17 — Task 4 enumeration widened during implementation

- The plan-quality round-2 disposition named two more sites:
  `couchtty/reserve.go:12-18` and `cmd/probes/couchnestedrows/main.go:18`.
- The sweep then found two more:
  - `ptychild.Screen`'s console-facing docs (the type has no production
    consumer) got a status note;
  - `atlas/couch.md`'s teardown sentence claimed the reserved row is cleared,
    which is false.
- Deleting the machinery itself went to pair#281 rather than into this plan.
