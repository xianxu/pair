# couch: say why a terminal stall ended the session, and stop repainting the whole screen (pair#409) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:**
- When couch exits because the outer terminal stopped accepting output, the next
  start says why.
- On the scrollback path, a small screen change costs bytes in proportion to the
  rows that changed, not a full-screen repaint.

**Architecture:** The two halves are independent, one milestone each.

- **M1:** `crashreport` gains a third kind of ending, a *recorded exit*. A one-line
  reason is written into this run's crash file, which already exists and is
  already reported once on the next start. couchcmd writes the line when the
  console ended on a parent-output `WriteFailure`. couchtty stays free of
  crashreport: it only exposes why it ended.
- **M2:** `HistoryRender.Emit` gains #262's cheaper fast path. When nothing pushed
  history and nothing reset, and every changed row is unwrapped in both frames
  (and does not soft-wrap into the next row), it repaints only those rows with
  `CUP` + `EL2` + cells. Everything else keeps the full rebuild.

**Tech Stack:** Go (`crashreport`, `couchcmd`, `couchtty`, `terminal`), the xterm-headless
oracle (`tests/terminal-oracle`).

---

## Context and decisions

- Issue: `workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md`.
  Per its Revision of 2026-10-07, the exit stays; it must stop being silent.
- #262's row-diff design (history:
  `workshop/history/issues/000262-diagnose-input-screen-flicker.md`, 2026-09-17).
  The full design repaints wrap chains, taken as the union of the chains in both
  frames, with a confined `IL`. Its stated cheaper first step is to diff only rows
  unwrapped in both frames and rebuild for everything else. The issue accepts
  either. **This plan builds the cheaper step:**
  - it covers the measured cost, an idle spinner or elapsed-time counter, which
    sits on an unwrapped row and adds no history;
  - it leaves the soft-wrap provenance rules (zellij keeps a row's wrap flag
    across `EL2`; only `IL` yields a pristine row) untouched, because every row it
    repaints is unwrapped in both frames and has no outgoing soft link.
  The full chain diff stays recorded for later, with its trigger: a measured cost
  from wrapped rows.
- **Trusting `previous` at row level** rests on the presenter being the parent's
  sole writer, which #262 verified (`hostty.Reservation` has no production
  caller). `Render` already trusts `previous` at cell level.
- **Not in scope:**
  - surviving the stall (the operator's decision);
  - the restore write that also times out on exit, which leaves the terminal
    broken: it is a symptom of the same stall, and the recorded reason now
    explains it;
  - the 647 MB `wrap-events` log noted in the issue Log.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Exited` (crashreport `Kind`) | `cmd/internal/crashreport/crashreport.go` | new |
| `Classify` / `Summary` | `cmd/internal/crashreport/crashreport.go` | modified |
| `ExitReason(err) (string, bool)` | `cmd/internal/terminal/transport.go` | new |
| `changedPlainRows(previous, next Frame) ([]int, bool)` | `cmd/internal/terminal/history_render.go` | new |

- **Exited.** A crash file whose content begins with the exit marker
  (`couch-exit: `). It is the run's own recorded reason, not a runtime panic.
  `Classify` stays a pure function of the listing plus each non-empty file's first
  line, which `list` reads, bounded to 256 bytes. `Summary` folds every exit into
  the one status sentence: `previous couch exited: <reason>`, with a crash and an
  abrupt end reported alongside as today.
  - **DRY:** it reuses the per-run file, the rename-on-report rule and the
    sweep/retention of #397. It adds no second channel.
- **ExitReason(err).** If `err` holds a parent-output `*WriteFailure`, returns
  "terminal stopped accepting output for 5s (wrote A of N bytes: <cause>)". Pure;
  the single place the wording lives.
- **changedPlainRows(previous, next).** Returns the rows whose cells or metadata
  differ, and ok=false unless:
  - the geometry, endpoint and alt state are the same;
  - every changed row is unwrapped in both frames;
  - the row below each changed row is unwrapped in both frames (so it has no
    outgoing soft link);
  - row 0, if changed, is unwrapped in both frames, so it does not continue from
    history.
  Pure, with table tests.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `crashreport.RecordExit(reason)` | `crashreport.go` | new | the active capture's file |
| `Console.TerminalFailure() error` | `cmd/internal/couchtty/console.go` | new | the console's recorded terminal failure |
| couchcmd exit hook | `cmd/internal/couchcmd/run.go` (`runConsole`) | modified | crashreport, console |
| `HistoryRender.Emit` fast path | `cmd/internal/terminal/history_render.go` | modified | parent writer |

- **RecordExit** writes `couch-exit: <reason>\n` to the active capture file and
  syncs it. With no active capture it does nothing. `Finish` keeps a non-empty file
  as today, so the reason survives the normal return.
- **couchcmd's hook.** After `console.Run()` returns,
  `if reason, ok := terminal.ExitReason(console.TerminalFailure()); ok {
  crashreport.RecordExit(reason) }`. couchtty does not import crashreport.
- **Emit's fast path:** `p.diffRows` is set in `RenderWithHistory` when
  - `!reset && !enterAlt && !leaveAlt`;
  - nothing new went to history (`installed.Cursor == history.Cursor`, so no
    `p.rows`);
  - `changedPlainRows` returns ok.
  `Emit` then writes, for each changed row: `CUP(0,y)`, `EL2`, the cells and
  `blankTail(soft=false)`. The prologue (sync bracket, modes re-assert) and the
  cursor epilogue are the same as the full path. If either fails, the full rebuild
  runs.

## Tasks

### M1 — the exit says why

- [ ] **1.1** `crashreport`:
  - `Exited` kind and `Report.Reason`;
  - `list` reads the first ≤256 bytes of non-empty `.log` files;
  - `Classify` and `Summary`;
  - `RecordExit`.
  Tests:
  - `TestRecordedExitIsReportedOnceWithItsReason`: install, `RecordExit`, `Finish`,
    then a new `Install` with the pid dead reports kind `Exited` with the reason;
    a third install reports nothing;
  - a panic file stays `Crashed`, an empty file stays `Abrupt`;
  - `Summary` combines exits with crashes.
- [ ] **1.2** `terminal.ExitReason` with table tests: parent output, child input,
  a wrapped error, nil.
- [ ] **1.3** `Console.TerminalFailure()`, and the couchcmd hook in `runConsole`.
  Test `TestPresenterStallRecordsTheExitReason`:
  - a fake parent writer that blocks past a shortened `WriteTimeout` (the existing
    stall harness in `presenter_stall_test.go` and the console fakes);
  - the console ends, and the crash file holds the reason;
  - the next `Install` reports it, separately from a Go panic.
- [ ] **1.4** Docs: the atlas crash-report section (`atlas/couch.md` near "crashreport")
  and the README if it lists startup notices.
- [ ] **M1 — milestone close** (`sdlc milestone-close --milestone M1`).

### M2 — repaint only the rows that changed

- [ ] **2.1** `changedPlainRows` with table tests:
  - one changed unwrapped row;
  - changed row 0 (unwrapped);
  - a changed row whose next row is wrapped (refused);
  - a changed wrapped row (refused);
  - a changed row-metadata-only difference;
  - a geometry or alt change (refused);
  - no change (empty list, ok).
- [ ] **2.2** The `Emit` fast path. Tests:
  - `TestHistoryEmitSpinnerTickWritesOnlyTheChangedRow`: on a 191×53 frame, one
    changed cell on an unwrapped row. Record the bytes before (full rebuild, about
    17 KB) and after (a few hundred bytes) in the test log, and assert under 1 KB.
  - `TestHistoryRowDiffMatchesFullRebuildOracle`: frame sequences (spinner ticks,
    a status line changing length, a cleared row, a wide character, styled
    cells), each emitted with the fast path and with a forced full rebuild,
    then fed to the xterm-headless oracle. Viewport cells, wrap flags and history
    must be identical. The test reuses the harness of
    `TestHistoryWireIndependentOracle`.
  - The existing history, presenter and oracle tests still pass.
- [ ] **2.3** Measure against the recorded case: replay a spinner-only sequence at
  the issue's size and record the bytes per idle minute before and after in the
  Log (target: down by more than 10×).
- [ ] **2.4** Docs: the atlas terminal or presenter section, noting that the
  history path diffs unwrapped rows.
- [ ] **2.5** Full suite unsandboxed (`make -k test`, scratchpad-TMPDIR
  `test-changelog`, `go test ./...`), compared with main.
- [ ] **2.6** Live check (operator): run couch with `COUCH_CAPTURE_DIR`, in the
  background behind a browser with a spinner running. Compare idle host-write
  bytes per minute with the issue's capture. If couch still exits, the next start
  must name the reason.
- [ ] **M2 — close** (`sdlc close`).

## Plan quality notes

- **ARCH-DRY.** One crash file per run, reported once (#397). One wording function
  (`ExitReason`). The fast path reuses the emitter's `cells` and `blankTail`.
- **ARCH-PURE.** `Classify`, `Summary`, `ExitReason` and `changedPlainRows` are
  pure, and the IO stays in `list`, `RecordExit` and `Emit`.
- **ARCH-PURPOSE.** The fast path is gated on the exact condition under which it
  is provably equal to the full rebuild (no wrap provenance involved). The oracle
  test enforces that equality.
- **ARCH-CONSTRAINTS.** The 5 s deadline is unchanged. The byte cost of an idle
  spinner tick goes from about 17 KB to a few hundred bytes.

## Revisions

### 2026-10-07 (a) — plan-quality round 1

- **PQ-1: the M1 test does not shorten `WriteTimeout`.** It is a const
  (`profile.go:23`), and making the write budget injectable just for a test would
  widen the presenter seam for no product reason. Task 1.3 becomes two proofs:
  - **Stall → typed failure.** This is already proven by `presenter_stall_test.go`
    with the real 5 s budget. That test stays the evidence that a stalled parent
    becomes `WriteFailure{Op: "parent output"}`; it is not duplicated.
  - **Typed failure → recorded and reported.** New in package `couchcmd`:
    `TestConsoleTerminalFailureIsRecordedAsTheExitReason`. A console whose
    `TerminalFailure()` holds a parent-output `*WriteFailure` (wrapped, as teardown
    joins it) is passed through the same `recordConsoleExit` helper `runConsole`
    calls. The crash file holds the reason, and the next `Install` (with the pid
    dead) reports `Exited` with it, while a panic file in the same directory stays
    `Crashed`. No timing is involved.
- **PQ-2: one generative check replaces the hand-listed cases.** Tasks 2.1 and 2.2
  become one strategy:
  - `TestHistoryRowDiffEqualsFullRebuild` builds seeded random frame pairs from
    the existing `historyFixture`. The same scrollback goes into both frames, and
    the mutations are:
    - a line's text changed or cleared, or its length changed across `cols`
      (wrap-flag flips);
    - wide characters at the right edge;
    - full-width rows;
    - styled and background cells;
    - the cursor moved.
  - Each pair is emitted twice: the fast path after a full paint of the first
    frame, and a forced full rebuild. Both go to the xterm-headless oracle
    (`runHistoryOracle`), which must report identical viewport lines, wrap flags
    and history.
  - It runs at least 60 seeds; a failure prints the seed and both wires.
  - Pairs that the gate refuses (`changedPlainRows` not ok) must produce
    byte-identical output to today's full rebuild. That proves the refusal falls
    back, so the generator also exercises the refusal branch.
  - `changedPlainRows` keeps a small table test only for its refusal reasons,
    because the property test proves equality but not that the gate is as wide as
    intended.
  - The byte assertion (one changed cell on 191×53 → under 1 KB) stays as its own
    test.
- **Minors.**
  - `ExitReason` derives the duration from `WriteTimeout`; no literal "5s".
  - The issue's "no message" is corrected: teardown does print
    `couch: terminal: <err>` to stderr (`console.go:982`), but onto the terminal
    that just stopped accepting output, so it is lost. The fix stands.
  - **ARCH-ORDER:** a presenter failure is terminal; no partial-paint recovery path
    later consumes `previous`, so the row diff never trusts a half-written frame.
    The presenter commits `previous` only after every chunk succeeds.
  - **ARCH-FUNERAL:** the recorded exit lives in #397's per-run file, with its
    rename-on-report and `pair gc` sweep, so nothing new is created and nothing
    new needs collecting.

### 2026-10-07 (b) — as built: M1 wording and the M2 gate (M1 review advisories, M2 evidence)

- **M1 as built.**
  - `ExitReason` reads "terminal stopped accepting output (a write waited up to
    5s; wrote A of N bytes)". A caller's shorter deadline can end a write first,
    so "for 5s" would overclaim.
  - The `Exited` notice names its file.
  - `Console.TerminalFailure` returns teardown's own classification (BR-1).
  - Tasks 1.1–1.4 are done (M1 closed SHIP, 673162c8).
- **M2: the gate is narrower than #262's wording.** #262 also required that a
  changed row not soft-wrap into the row below. The generative test shows that
  condition guards nothing:
  - with it dropped, 48 of 80 seeds take the row diff instead of 26;
  - every one of them equals the full rebuild under both the xterm oracle and the
    native zellij oracle (`PAIR_TERMINAL_NATIVE=1`).
  Repainting row `y` never touches row `y+1`'s own wrap flag, which is where the
  soft link lives. So the gate is "every changed row is unwrapped in both
  frames", with nothing reset and nothing new for history.
  `TestChangedPlainRowsGate` pins its width:
  - a cursor-only change and a plain-row edit are taken;
  - so is the head of a wrapped line;
  - a wrapped row, a wrap-flag flip, an endpoint switch and a resize are refused.
- **Oracle setup.** `tests/terminal-oracle` needed `npm ci` in this checkout;
  without it the oracle tests skip silently. `PAIR_TERMINAL_ORACLE=1` turns that
  skip into a failure.

