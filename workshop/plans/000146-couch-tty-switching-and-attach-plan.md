# couch: tty switching and attach — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One `couch` process owns the operator's tty and routes it to one agent child at a time — `ctrl-space` goes up one level, a reserved bottom row says what happened, a per-child buffer means landing is never a blank screen, and a child that exits lands the operator in the panel rather than on a dead pane.

**Architecture:** `couch start` stops inheriting stdio and becomes **the console**: it allocates a pty per child, puts the real tty in raw mode, and pumps bytes. Child output is passed through verbatim — couch **does not composite**; it reserves the last row by sizing children one row shorter and pinning the host's scrolling region above it. The pty-child mechanics (pty + replay ring + stream scanner) are extracted from `termcmd`'s existing multiplexer into a shared `ptychild` package, so `pair term` and `couch` share the *structure* and keep their own *policy* — the same split `cmd/internal/ansi` already makes (ARCH-DRY). Everything the panel can do dispatches through `couchcore.Operations()`; there is no second implementation for the operator's surface (ARCH-PURPOSE).

**Tech Stack:** Go 1.26, `creack/pty`, `golang.org/x/term`, `cmd/internal/ansi`. No TUI framework — pair writes raw escapes directly, and this follows that.

**Issue:** pair#146. **Project:** `workshop/projects/couch.md`. **Predecessor:** pair#145 (registry, spawn, seams).

## How to read this plan

Same convention as `000145-couch-spawn-and-registry-plan.md`, for the same reason: **contracts and test intent, not finished code.** Hand-written Go in markdown cannot be validated without executing it. Each task states the **contract**, **what each test must catch**, and the **deletion check** — the line you remove to prove the test is load-bearing (`workshop/lessons.md`: "A test that survives deleting the seam it names tests nothing").

Terminal code has its own standing moves, all of them lessons already paid for in this repo:

- **One scanner per package.** Framing a CSI is `cmd/internal/ansi`'s job. Do not write a third scanner (lessons: "Paired protocol terminators need one constant, not one per site").
- **Every pure byte-scanner gets a `Fuzz*`** asserting no-panic plus `len(out) <= len(in)`, seeded with malformed forms — not just valid sequences (lessons: "Any pure byte-scanner gets a fuzz test").
- **Buffer only real prefixes.** A complete-but-unsupported control is consumed, never held (lessons: "Escape decoders must distinguish prefixes from unknown complete controls"). Add split-boundary tests where the final byte arrives in the next read.
- **Terminal behaviour is not provable from unit tests.** Every milestone that changes what the operator sees ends with an operator smoke against a *real* `pair` + `claude` child. Fakes prove the wiring; the smoke proves the terminal.

---

## Decisions

1. **`couch start` becomes the console. No new verb.** `couchcmd`'s dispatch table is asserted *identical* to `couchcore.Operations()` (`run_test.go`), so a console-only verb would need an exception to the invariant that keeps the operator's surface and the advisor's from drifting. `start` already blocks for the child's lifetime and already returns a `Handle` the CLI drives — "blocks and owns the tty" narrows that contract rather than inventing one. **The root actor is the child of the first `couch start`**, which is exactly "whatever session couch launched in".

2. **`--no-console` is the escape hatch, and it announces itself.** It keeps today's `ExecRunner` path (inherit stdio, block, no pty). If the tty layer misbehaves the operator is never stranded — and per the escape-hatch rule the fallback prints a loud line saying the console is off, rather than silently degrading. This also keeps `ExecRunner` a live production path, so its live conformance check stays honest rather than pinning dead code.

3. **The pty is a capability on `Handle`, not a new signature on `Runner`.** `Runner.Start(dir, argv, env) (Handle, error)` is unchanged; a handle from `PtyRunner` additionally satisfies `TerminalHandle`. Rationale: two runners genuinely differ in what they can offer, and widening `Start` would churn every existing caller and fake for a capability only one of them has. The console type-asserts once, at its own boundary.

4. **The reserved row is a scrolling-region reservation, not compositing.** The child pty is `rows-1` tall; the host's scrolling region is pinned to `1..rows-1` (DECSTBM) so a child scrolling at *its* bottom line scrolls inside the region and cannot walk onto the reserved row. The row is painted with save-cursor / absolute-move / paint / restore-cursor so the child's cursor is never disturbed. **This is the design the Spec chose** ("couch does not composite — it reserves a row"); compositing every frame through a `vt.Emulator` is explicitly the rejected alternative.
   - **Known risk, and why it is scheduled early:** apps that set or reset margins themselves (`nvim` emits `\x1b[r` on exit) can drop the reservation. Mitigation is to re-assert the region whenever the row is painted, and to re-assert immediately when the stream scanner sees a margin reset or an alt-screen transition. If real children defeat this, the fallback is to drop the row to an on-demand overlay rather than to start compositing — recorded here so the fallback is a decision, not an improvisation.

5. **Landing repaints from a ring; alt-screen children get a resize nudge instead.** Replaying a raw buffer is right for line-mode children and is what `pair term` already does. For a child in the alt screen (zellij, nvim) the buffer is a stream of partial redraws, so replay is the wrong tool: those children redraw natively on `SIGWINCH`, which the Spec already anticipates. The console therefore *knows* whether a child is on the alt screen — from the same scanner that tracks margins and mouse mode — and picks: replay, or nudge.

6. **`ptychild` is extracted from `termcmd` and `pair term` migrates onto it in M1 (ARCH-DRY).** `termcmd.terminalMux` already is a switcher: pty-backed tabs, a 128KB replay ring, redraw-from-snapshot on switch, resize propagation, EOF-driven removal. Building couch's a second time is the duplication ARCH-DRY exists to stop.
   - **What is shared is structure; what stays is policy.** `cmd/internal/ansi`'s doc makes exactly this split, and it applies again: `pair term` cycles numbered tabs and exits when empty; couch switches named actors and falls back to a panel. Those policies stay with their callers.
   - **`stripTerminalQueries` moves and is shared** — both callers replay a raw buffer *to a real terminal*, so the deny-list is one policy with two sites, not two opposed policies (contrast `wrapcmd`, which strips `\x1b[>7u` while `termcmd` requires `\x1b[>1u` to survive; those stay apart).
   - **The migration is the test.** Extracted code with no second consumer is unvalidated new code; `termcmd`'s 1137 lines of existing tests are the regression net that proves the extraction is faithful. That is why M1 migrates rather than deferring it — and why M1 comes first even though it ships no couch behaviour.

7. **Detach is console-scoped. A daemon is `#147`.** Within a console, detaching from a child (switching away, or going to the panel) leaves it running and warm: its pty stays open, its ring keeps filling, reattach replays. Children do **not** survive the console process — they are its children, and closing the terminal takes them down. Making a fleet outlive its console needs a daemon plus pty handoff over a socket, which is the transport `#147` builds; doing it here would drag `#147` into `#146` and delay the switcher the operator would actually use daily. **Flagged for operator confirmation** — see Open questions.

8. **The status row carries the one real activity signal available today: `BEL`.** With no transport (`#147`) the row could only report attach/exit, which makes it decorative — and a row that never says anything useful is dead weight that still costs a terminal row. A child's `\x07` is a genuine "the agent wants you" signal, it is already in the byte stream the scanner reads, and it costs one field. Anything richer (per-actor mailbox depth, git staleness) is `#147`/`#148` and is deliberately absent.

9. **Notices reuse `couchcore.Enqueue`, they do not re-implement it.** The row's rolling feed wants exactly Enqueue's policy: collapse by kind (a second bell from the same actor replaces the first), bounded, never drop control (an exit is control). Keyed as `bell:<ActorID>` / `exit:<ActorID>` so collapse is per-actor rather than global (ARCH-DRY).

10. **`ctrl-space` (`0x00`) is intercepted before the child sees it.** The Spec settled the key; what this plan owes is the audit the repo's own lesson demands ("Never disable an input layer without auditing the escape hatches it provides"). `zellij/config.kdl` binds no Space chord, so nothing in the workbench loses a path. The audit for `claude` and `nvim` is a step in M2, and its result is recorded in the issue `## Log` — including, if something does ride on it, how a literal `ctrl-space` reaches a child.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Ring` | `cmd/internal/ptychild/ring.go` | new |
| `StripQueries` + query deny-list | `cmd/internal/ptychild/replay.go` | new (moved from `termcmd/queries.go`) |
| `Screen` | `cmd/internal/ptychild/screen.go` | new |
| `updateMouseMode` | `cmd/internal/termcmd/run.go` | deleted (folded into `Screen`) |
| `Focus` / `Up` / `Home` | `cmd/internal/couchtty/focus.go` | new |
| `PanelModel` / `Filter` / `Pick` | `cmd/internal/couchtty/panel.go` | new |
| `StatusModel` / `RenderStatusRow` | `cmd/internal/couchtty/statusrow.go` | new |
| `Intercept` | `cmd/internal/couchtty/keys.go` | new |
| `Reserve` / `Release` / `PaintRow` | `cmd/internal/couchtty/reserve.go` | new |
| `Notice` / `Feed` | `cmd/internal/couchtty/notice.go` | new |

- **Ring** — a bounded byte buffer with a snapshot. `Append([]byte)`, `Snapshot() []byte` (an independent copy). Cap 128KB, lifted from `termcmd.appendBuffer`.
  - **Relationships:** 1:1 with `ptychild.Child`.
  - **DRY rationale:** removes the buffer-trim policy from `termcmd` so one place owns "how much scrollback a detached child keeps".
  - **Future extensions:** a byte cap is a proxy for "enough to land on". If landing proves thin, this widens to a line- or screen-aware cap without any caller changing.

- **StripQueries** — the replay deny-list from `termcmd/queries.go` (#127), moved verbatim with its tests. Removes capability queries from a *replayed* buffer so the repaint cannot re-ask the host terminal and have the answer land in another child's stdin.
  - **DRY rationale:** couch's repaint-on-attach is the same operation `redrawTab` performs. Without the move, couch either re-earns #127's bug or copies its table.
  - **Future extensions:** stays a best-effort deny-list; a missed query degrades to the old behaviour, exactly as documented today.

- **Screen** — the single scanner over a child's output stream. Consumes bytes, maintains `AltScreen bool`, `Mouse bool`, `MarginsDirty bool`, `Bell bool` (latched, cleared by the reader). Framing goes through `ansi.TerminatorScan`; it does **not** frame CSIs itself.
  - **Relationships:** 1:1 with `Child`.
  - **DRY rationale:** `termcmd.updateMouseMode` is today's half of this and gets folded in — one scanner per package, per the paired-terminator lesson.
  - **Future extensions:** title (OSC 0/2) and OSC 777 notifications are the natural next fields; the console's status row is already the place they would surface.

- **Focus** — `FocusPanel` or `FocusActor(ActorID)`, plus `Up(cur, root) Focus`: a non-root child goes home to the root actor; the root actor goes to the panel; the panel stays. Pure; the whole navigation rule is one function.
  - **DRY rationale:** first occurrence, but the rule is stated in three places (project, issue, atlas) and must have exactly one implementation.
  - **Future extensions:** direct jumps ("to actor N", "to the latest notifier") are deliberately deferred by the Spec; they widen `Up` into a `Move(cur, intent)` without touching the console.

- **PanelModel / Filter / Pick** — the panel as data: rows built from `couchcore.TreeSummary`, a typeahead filter over name + description + repo, and `Pick(query, digit)` resolving a keystroke to a row. Pure — the panel renders from this, and `#148`'s advisor can call the same resolution.

- **StatusModel / RenderStatusRow** — `(width int, m StatusModel) string`: the actor chips, the active marker, activity markers, and the newest notice, truncated to width. Pure, so the row is unit-testable without a terminal.

- **Intercept** — `(in []byte) (forward []byte, hits int)`: splits `0x00` out of a stdin chunk and returns what the child should still receive. Pure. `ctrl-space` is a single byte, so there is no framing state — and that is *why* the Spec chose a bare key over a chord.

- **Reserve / Release / PaintRow** — the escape sequences as pure string builders: `Reserve(rows)` → region + parking, `PaintRow(rows, text)` → save / move / clear / paint / restore, `Release()` → region reset. One constant per sequence, per the paired-terminator lesson.

- **Notice / Feed** — `Notice{Kind, Body, Control}` and a feed that delegates to `couchcore.Enqueue`. `Feed` holds the capacity and the key convention (`bell:<id>`, `exit:<id>`); the policy stays in Enqueue.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ptychild.Child` | `cmd/internal/ptychild/child.go` | new | `creack/pty` + `os/exec` |
| `couchcore.TerminalHandle` | `cmd/internal/couchcore/runner.go` | modified | pty capability on a `Handle` |
| `couchcore.PtyRunner` | `cmd/internal/couchcore/ptyrunner.go` | new | `ptychild.Child` behind `Runner` |
| `FakeRunner` terminal double | `cmd/internal/couchcore/runner_fake.go` | modified | in-memory stand-in for a pty |
| `couchtty.Console` | `cmd/internal/couchtty/console.go` | new | raw mode, `SIGWINCH`, real stdio |
| `termcmd.terminalTab` | `cmd/internal/termcmd/run.go` | modified | now holds a `ptychild.Child` |
| `couchcmd` wiring | `cmd/internal/couchcmd/run.go` | modified | picks `PtyRunner` vs `ExecRunner` |
| live conformance | `cmd/internal/couchcore/conformance_live_test.go` | modified | `PtyRunner` vs `FakeRunner` |

- **ptychild.Child** — one process on a pty: `Start`, `Write`, `Resize(rows, cols)`, `Snapshot`, `AltScreen`, `Bell`, `Wait`, `Close`. Owns the read pump that feeds `Ring` and `Screen`.
  - **Injected into:** `termcmd.terminalMux` and `couchtty.Console`, both of which keep their own switching policy.
  - **Future extensions:** a `Tee(io.Writer)` for on-disk scrollback (pair already tees `scrollback-<tag>-<agent>.raw`; couch would reuse that file rather than invent a second).

- **couchcore.PtyRunner** — `Runner` whose handles are pty-backed. Constructed with an initial winsize supplier so the first frame is already the right size rather than 80x24-then-resize.
  - **Injected into:** `couchcore.Couch` through the existing seam. Nothing in the domain learns about ptys.

- **FakeRunner terminal double** — the fake's children gain an in-memory terminal: writes are recorded and echoed per a scripted behaviour, resizes are recorded, and exit closes the read side (EOF). ARCH-MOCK: the fake models behaviour across calls, and the live check compares it against a real pty rather than asserting whatever each produces separately.

- **couchtty.Console** — the thin IO shell: `term.MakeRaw`, `SIGWINCH`, the stdin pump, the per-child output pumps, and terminal restoration. It holds no policy — every decision it makes is a call into a pure function above.

---

## Milestones

Four review boundaries, each a real stopping point. Value is front-loaded after M2; risk is answered in M1–M2.

## Chunk 1: M1 — the shared pty-child core

Ships no couch behaviour. It exists so that couch's console and `pair term` are one mechanism, and so `ptychild` arrives already validated by an existing suite.

### Task 1.1 — `Ring`

**Files:** Create `cmd/internal/ptychild/ring.go`, `cmd/internal/ptychild/ring_test.go`.

**Contract:** `Append` never grows past the cap; `Snapshot` returns a copy the caller may retain while `Append` continues.

- [ ] **Tests must catch:** (a) a buffer that grows unbounded — append past the cap, assert length; (b) **aliasing** — take a snapshot, append more, assert the snapshot is unchanged (today's `bufferSnapshotLocked` copies for exactly this reason, and the copy is invisible to a length assertion); (c) an append larger than the cap keeps the *tail*, not the head.
- [ ] **Deletion check:** remove the trim in `Append` → (a) goes red. Change `Snapshot` to return the slice directly → (b) goes red.
- [ ] Commit.

### Task 1.2 — `StripQueries` moves

**Files:** Create `cmd/internal/ptychild/replay.go` (+ `replay_test.go`); delete `cmd/internal/termcmd/queries.go` (+ its test) after moving both.

**Contract:** byte-identical behaviour to today. The doc comment moves with it — it is the record of *why* replay is filtered and the live path is not.

- [ ] Move the file and its tests verbatim, rename the package, export `StripQueries`.
- [ ] **Add `FuzzStripQueries`** — no panic, `len(out) <= len(in)`, seeded with the overlapping-prefix forms #127's review found (`\x1b]4;?`, a bare `\x1b[`, a CSI with no final byte). This is the repo's standing rule for byte-scanners and the original bug was exactly this shape.
- [ ] **Deletion check:** `termcmd`'s existing replay test must still pin the behaviour through the new call. If it passes with `StripQueries` replaced by `func(b []byte) []byte { return b }`, it was never pinning it — fix the test.
- [ ] Commit.

### Task 1.3 — `Screen`

**Files:** Create `cmd/internal/ptychild/screen.go`, `screen_test.go`. Delete `updateMouseMode` from `cmd/internal/termcmd/run.go`.

**Contract:** `Feed([]byte)` maintains `AltScreen`, `Mouse`, `MarginsDirty`, `Bell`. Framing uses `ansi.TerminatorScan` — no new CSI scanner.

- [ ] **Tests must catch:** (a) mouse-mode set/reset across `1000/1002/1003/1006` — port `termcmd`'s existing cases so the migration cannot silently lose them; (b) alt-screen enter/leave via `?1049`, `?1047`, `?47`; (c) `\x1b[r` and `\x1b[1;24r` both marking margins dirty, and `\x1b[3;4H` *not* doing so (a final byte is not enough — the introducer discriminates); (d) **split boundaries**: the same sequence delivered one byte per `Feed` reaches the same state; (e) a malformed complete control (`\x1b[@z`) is consumed, not held, and the following `z` is not swallowed.
- [ ] **Add `FuzzScreenFeed`** — no panic, and feeding a byte stream in one chunk equals feeding it split at every boundary.
- [ ] **Deletion check:** remove the `?1049` case → (b) red. Remove the introducer discrimination → (c) red.
- [ ] Commit.

### Task 1.4 — `Child`

**Files:** Create `cmd/internal/ptychild/child.go`, `child_test.go`.

**Contract:** `Start(dir, argv, env, size)` → a child on a pty with a running read pump; `Write`, `Resize`, `Snapshot`, `AltScreen`, `TakeBell`, `Wait() int`, `Close`. The pump feeds `Ring` and `Screen` and forwards each chunk to an injected sink (the console writes it to stdout only when that child is active).

- [ ] **Tests must catch (real child, `sh -c`, in-package integration):** (a) `Write` reaches the child — echo something back and read it from the snapshot; (b) `Resize` is *observed by the child* — the child prints `stty size` on `SIGWINCH`; a test that only asserts the ioctl returned nil proves nothing; (c) child exit closes the pump and `Wait` returns the code; (d) a `\x07` in the child's output latches `Bell` and `TakeBell` clears it.
- [ ] **Deletion check:** drop the `pty.Setsize` call → (b) red.
- [ ] Commit.

### Task 1.5 — migrate `pair term`

**Files:** Modify `cmd/internal/termcmd/run.go` (`terminalTab`, `newTab`, `readPTY`, `appendBuffer`, `resizeAll`, `redrawTab`, `closeAll`, `removeTab`, `appMouseMode`).

**Contract:** *no behaviour change.* `terminalTab` holds a `ptychild.Child`; the mux keeps every policy it has today — numbered tabs, rename, zellij pane titles, exit-when-empty.

- [ ] Replace the tab's `cmd`/`pty`/`buffer`/`mouse` fields with a `*ptychild.Child`; route `readPTY`'s chunks through the child's sink into the existing `output` channel so `copyActiveOutput` is untouched.
- [ ] **Tests must catch:** the existing `run_test.go` suite is the net. Run `go test ./cmd/internal/termcmd/ -count=1` and `make test-term-pane-shortcuts`. Any test that needed editing to pass is a **behaviour change** — stop and justify it in the plan's Revisions rather than editing the test.
- [ ] **Operator smoke:** `pair` → right terminal → open two tabs, switch, resize the window, run `nvim` in one and switch away and back. This is the daily driver; unit tests do not cover the repaint.
- [ ] Commit.

### Task 1.6 — close M1

- [ ] `go build ./... && go test ./cmd/... -count=1` and `go test ./cmd/... -race -count=1` (whole tree, not just the touched package).
- [ ] `sdlc milestone-close --issue 146 --milestone M1`; fix Critical/Important before crossing; record the verdict in `## Log`.

## Chunk 2: M2 — the console over one child, with the reserved row

The milestone that answers both terminal risks: does `pair` run correctly in a couch-owned pty, and does a reserved row survive real children.

### Task 2.1 — `TerminalHandle` and `PtyRunner`

**Files:** Modify `cmd/internal/couchcore/runner.go`; create `cmd/internal/couchcore/ptyrunner.go` (+ test).

**Contract:** `Runner.Start` is unchanged. `TerminalHandle` adds `Terminal() Terminal`, where `Terminal` is `io.Writer` + `Resize(rows, cols uint16) error` + `Snapshot() []byte` + `AltScreen() bool` + `Close() error`. `PtyRunner` wraps `ptychild.Child` and satisfies both.

- [ ] **Tests must catch:** (a) `ExecRunner`'s handle does **not** satisfy `TerminalHandle` — the capability check is meaningful only if one runner fails it; (b) `PtyRunner`'s does; (c) `PtyRunner` honours its initial size supplier at spawn (assert via the child, as in 1.4b).
- [ ] **Deletion check:** make `execHandle` satisfy the interface with stubs → (a) red.
- [ ] Commit.

### Task 2.2 — the fake grows a terminal, and conformance pins it

**Files:** Modify `cmd/internal/couchcore/runner_fake.go`, `conformance_live_test.go`.

**Contract (ARCH-MOCK):** `FakeChild` records writes and resizes and can be scripted to emit output; its handle satisfies `TerminalHandle`; exit closes the read side.

- [ ] **The live check compares implementations against one shared scenario** — write, resize, emit, exit — asserting the same observable sequence from both. A check that drives the fake by hand to the value it then asserts tests nothing; this is a named lesson, not a style note.
- [ ] **Tests must catch:** the drift that actually matters — a real pty delivers `SIGWINCH` and a fake that silently accepts a resize would hide a broken `Resize`; a real pty returns EOF on child exit and a fake that blocks forever would hide a hung pump.
- [ ] Gated on `PAIR_LIVE_COUCH=1`, no build tag (so it still compiles under `go test ./cmd/...`), reachable via `make test-live` — a gated-only pin nothing runs is not a pin.
- [ ] Commit.

### Task 2.3 — `Intercept`, and the ctrl-space audit

**Files:** Create `cmd/internal/couchtty/keys.go` (+ test).

- [ ] **Tests must catch:** (a) `0x00` is removed from the forwarded bytes and counted; (b) a chunk of several keystrokes containing one `0x00` forwards the rest **in order**; (c) `0x00` inside an escape sequence's payload is *not* treated as a hotkey — seed the case even if today's terminals do not produce it, because the failure mode is a swallowed keystroke.
- [ ] **Audit step (lesson: never disable an input layer without auditing what rides on it):** check `claude` and `nvim` for a `ctrl-space` binding (`zellij/config.kdl` already confirmed clear). Record the result in the issue `## Log`. If something does ride on it, say in the Log how a literal `ctrl-space` reaches a child — do not silently shadow it.
- [ ] Commit.

### Task 2.4 — `Reserve` and `RenderStatusRow`

**Files:** Create `cmd/internal/couchtty/reserve.go`, `statusrow.go` (+ tests).

**Contract:** pure string builders. `Reserve(rows)` sets the region to `1..rows-1`; `PaintRow(rows, text)` is save / absolute-move / clear-line / text / restore; `Release()` resets the region. Each sequence is one named constant.

- [ ] **Tests must catch:** (a) the region is `rows-1`, not `rows` — an off-by-one here is the whole bug; (b) `PaintRow` restores the cursor (assert the save/restore pair brackets the paint — without it the child's cursor lands on the status row); (c) `RenderStatusRow` truncates to width without splitting an escape sequence; (d) the active actor is marked distinctly from an actor with pending activity.
- [ ] **Deletion check:** drop the restore from `PaintRow` → (b) red.
- [ ] Commit.

### Task 2.5 — `Console`

**Files:** Create `cmd/internal/couchtty/console.go` (+ test).

**Contract:** the thin IO shell. `MakeRaw` on the real tty, `SIGWINCH` → measure the host and `Resize` the child to `rows-1`, stdin pump through `Intercept`, child output written to stdout only when that child is active, `Release` + `term.Restore` on every exit path.

- [ ] Re-assert `Reserve` whenever the row is painted, and immediately when `Screen` reports `MarginsDirty` or an alt-screen transition (Decision 4).
- [ ] **Tests must catch (fakes, no real tty):** (a) a child resized to `rows-1`, never `rows`; (b) an intercepted `ctrl-space` is **not** forwarded to the child; (c) restoration runs when the child exits *and* when the console is torn down mid-stream — a restore that only happens on the happy path leaves the operator's terminal with a broken scroll region.
- [ ] **Deletion check:** remove the `-1` → (a) red. Remove the deferred `Release` → (c) red.
- [ ] Commit.

### Task 2.6 — wire `couch start`

**Files:** Modify `cmd/internal/couchcmd/run.go`, `cmd/internal/couchcore/ops.go`.

**Contract:** `Runtime` gains `NewCouchWith(Runner)`; `NewCouch()` stays `ExecRunner` for `--no-console` and foreign-shell use. `start` declares a `no-console` arg — `FlagOnly`, because it bypasses the console the same way `same-tree` bypasses the guard, and a stray positional word must not be able to set it.

- [ ] **Tests must catch:** (a) the declared-operations audit still passes (the arg is declared, not smuggled); (b) `couch start x --no-console` takes the `ExecRunner` path and prints the loud fallback line; (c) `couch start x no-console` does **not** — the guard-bypass-never-binds-positionally rule has a test in this repo already; mirror it.
- [ ] Commit.

### Task 2.7 — operator smoke (the milestone's real verification)

- [ ] `make build` then `./bin/couch start ../pair` from `brain`.
- [ ] Confirm, and record each in the issue `## Log` with what was observed: pair + zellij + claude come up inside the pty; the layout is correct at `rows-1`; resizing the terminal reflows the child; the reserved row stays visible while claude streams output; `nvim` opens **and exits** without eating the row (the margin-reset case from Decision 4); `ctrl-space` is intercepted and does not reach the child; quitting restores the terminal (`echo $LINES`, scroll region reset, no raw-mode residue).
- [ ] If the row does not survive, take the Decision 4 fallback and record it as a Revision — do not start compositing.

### Task 2.8 — close M2

- [ ] Whole-tree `go test ./cmd/... -count=1` and `-race`; `make test-live`.
- [ ] `sdlc milestone-close --issue 146 --milestone M2`.

## Chunk 3: M3 — many children, and the panel

### Task 3.1 — `Focus`

**Files:** Create `cmd/internal/couchtty/focus.go` (+ test).

- [ ] **Tests must catch:** (a) a non-root child goes to the **root actor**, not the panel — the single most important property in the project, and the easy wrong implementation is "up = panel"; (b) the root actor goes to the panel; (c) the panel stays on the panel; (d) `Up` from a child whose root actor has **died** does not land on a dead actor — it goes to the panel.
- [ ] **Deletion check:** collapse (a) into (b) → (a) red.
- [ ] Commit.

### Task 3.2 — `PanelModel`

**Files:** Create `cmd/internal/couchtty/panel.go` (+ test).

**Contract:** rows from `couchcore.TreeSummary` — so parked trees stay listed, dimmed, exactly as `couch list` already renders them. `Filter(query)` matches name, description and repo (the same three fields `couchcore.LookupTrees` matches; the panel must not resolve on fewer). `Pick(digit)` selects the Nth **displayed** row.

- [ ] **Tests must catch:** (a) filtering matches the agent-published description, not just the operator's name; (b) `Pick(2)` after filtering picks the second *filtered* row, not the second underlying one; (c) a parked tree (no live actor) is listed; (d) ordering is stable across refreshes — a list that reorders under the operator's fingers makes numbered selection a hazard.
- [ ] **Deletion check:** drop description matching → (a) red.
- [ ] Commit.

### Task 3.3 — N children in the console

**Files:** Modify `cmd/internal/couchtty/console.go`.

**Contract:** the console holds a map of `ActorID` → child. Only the active child's chunks reach stdout; every child's chunks reach its own `Ring` and `Screen`. Attach = `Reserve`, then **replay** (`StripQueries(Snapshot())` after a clear) for a line-mode child, or a **resize nudge** for an alt-screen child (Decision 5), then repaint the row.

- [ ] **Tests must catch:** (a) an inactive child's output does not reach stdout but does reach its ring — the bug this prevents is a switcher that loses everything said while you were away; (b) an alt-screen child gets the nudge and **not** the replay, and a line-mode child the reverse; (c) attach repaints the status row *after* the child's repaint, so the row is not overwritten by the landing.
- [ ] **Deletion check:** always replay → (b) red.
- [ ] Commit.

### Task 3.4 — the panel dispatches through `Operations()`

**Files:** Modify `cmd/internal/couchtty/panel.go`, `console.go`.

**Contract:** `start`, `stop`, `name`, `describe` from the panel call `couchcore.Operations()` — the same table the CLI and (in `#148`) the advisor use. **No second implementation of an operator action.**

- [ ] **Tests must catch:** the panel's action set is a **subset of** `couchcore.OperationNames()`, asserted by name. The existing CLI audit proves the same thing for the CLI; without this one the panel is free to grow a private verb, which is precisely the drift the ops table exists to stop.
- [ ] **Deletion check:** add a panel-only action → the audit goes red.
- [ ] Commit.

### Task 3.5 — operator smoke: two real children

- [ ] From the root actor, start a second child on another peer repo via the panel.
- [ ] Confirm and log: switching between them is instant with no model turn; `ctrl-space` from the *second* child lands on the root actor; `ctrl-space` again reaches the panel; typeahead finds a child by its agent-published description; a digit jumps to it; **`ctrl-space` works while a child is mid-output** (start a long stream first — this is the Done-when clause most likely to fail, because a blocked stdout pump would stall the interceptor).
- [ ] Commit + `sdlc milestone-close --issue 146 --milestone M3`.

## Chunk 4: M4 — exits, detach, and what the row says

### Task 4.1 — a child that exits never leaves a dead pane

**Files:** Modify `cmd/internal/couchtty/console.go`; create `cmd/internal/couchtty/notice.go` (+ test).

**Contract:** on child exit — focus the panel, emit `exit:<id>` as a **control** notice carrying the actor and the exit code, and unregister through `couchcore` so the tree is freed (`Couch.Forget`, the path `PruneDead` already models).

- [ ] **Tests must catch:** (a) exit while that child is **active** focuses the panel; (b) exit while it is **inactive** does not steal focus but does record the notice — a switcher that yanks the operator out of the child they are typing in is worse than the dead pane; (c) the notice names the actor and the code; (d) the registry entry is gone afterwards.
- [ ] **Deletion check:** drop the Forget call → (d) red.
- [ ] Commit.

### Task 4.2 — `Feed` over `couchcore.Enqueue`, and the row says something

**Files:** Modify `cmd/internal/couchtty/notice.go`, `statusrow.go`, `console.go`.

- [ ] **Tests must catch:** (a) two bells from the *same* actor collapse to one entry; (b) bells from *different* actors do **not** collapse (the key is per-actor — a global `bell` kind would merge the fleet into one notice); (c) an exit notice is never dropped under capacity pressure; (d) the row marks an actor with a pending bell distinctly from the active one.
- [ ] **Deletion check:** key notices as bare `bell` → (b) red.
- [ ] Commit.

### Task 4.3 — detach and reattach stay warm

**Files:** Modify `cmd/internal/couchtty/console.go` (+ test).

- [ ] **Tests must catch:** (a) after switching away, the child's process is still alive and its ring is still growing; (b) reattaching replays what accumulated; (c) going to the panel and back is the same path as switching between children — one mechanism, not two.
- [ ] Record in the issue `## Log`: children do **not** survive the console (Decision 7), and `#147` owns the daemon that would change that.
- [ ] Commit.

### Task 4.4 — restore the terminal on every exit path

**Files:** Modify `cmd/internal/couchtty/console.go`.

- [ ] Region reset, cursor restored, raw mode restored, alt screen left — on normal quit, on last-child exit, and on `SIGTERM`/`SIGHUP` to couch itself.
- [ ] **Tests must catch:** the signal path specifically. A `defer` covers the happy path and does not run on a signal; a console that leaves the operator's terminal with a pinned scroll region after a `kill` is the worst failure this milestone can ship.
- [ ] Commit.

### Task 4.5 — docs and the map

**Files:** Modify `atlas/couch.md`; verify `couch --help` renders the new arg.

- [ ] Rewrite the atlas's **"There is no pty yet"** and **"Planned, not built"** paragraphs — they are current-state claims that this issue falsifies, and the atlas holds only current state.
- [ ] Add the console and the reserved row to `atlas/couch.md`, and describe `ptychild` as shared with `pair term` (name the second consumer, or the next reader re-derives it).
- [ ] Do **not** enumerate the operation set in prose — the atlas already records why that drifts.
- [ ] Commit.

### Task 4.6 — close

- [ ] Whole-tree tests, `-race`, `make test-live`, `make test` for the shell suites that touch `pair term`.
- [ ] Final operator smoke: a full session — start, roam, get paged by the row, come home, exit.
- [ ] `sdlc close --issue 146 --verified '<evidence>'` (let it measure `--actual`; do not hand-type hours).

---

## Acceptance mapping

| Done-when (issue #146) | Where it is met | How it is proven |
|---|---|---|
| couch supervises N sessions and switches the tty | 3.3 | unit (routing) + 3.5 smoke |
| `ctrl-space` reaches the root actor from every child, mid-output, instantly | 3.1, 2.3 | unit (focus) + 3.5 smoke (mid-output) |
| reserved row visible in root and attached child; child renders at reduced height | 2.4, 2.5 | unit (off-by-one, restore) + 2.7 smoke |
| an exited child lands the operator in the TUI with which actor and why | 4.1 | unit (active/inactive, code) |
| landing shows recent context, not a blank screen | 3.3 | unit (replay vs nudge) + 3.5 smoke |
| detach/reattach leave children running and warm | 4.3 | unit + logged scope note (Decision 7) |
| a numbered/direct switch path with no natural-language resolution | 3.2 | unit (`Pick` after filter) + 3.5 smoke |

## Verification before close

- `go build ./...`; `go test ./cmd/... -count=1`; `go test ./cmd/... -race -count=1` (whole tree — a race in the pumps will not show in one package).
- `make test-live` (`PAIR_LIVE_COUCH=1`) — the fake-vs-real pty conformance.
- `make test-term-pane-shortcuts` and the `pair term` smoke from 1.5 — the migration's regression net.
- The operator smokes from 2.7, 3.5 and 4.6, each logged with what was observed rather than "verified".
- `atlas/couch.md` reconciled to what exists (4.5).

## Open questions for the operator

1. **Detach scope (Decision 7).** Console-scoped detach — children die with the console; surviving it needs `#147`'s daemon. Confirm that reads the Done-when correctly, or `#146` grows a daemon.
2. **Migrating `pair term` in M1 (Decision 6).** It is the daily driver and the extraction's only regression net. Confirm the appetite, or `ptychild` ships unshared and the migration becomes its own issue — which is the duplication ARCH-DRY would flag.
3. **The status row's content (Decision 8).** `BEL` is the only real activity signal before `#147`. Confirm that is worth a permanent terminal row now, or the row ships showing only actors and attach/exit.

## Revisions

_(Append here: timestamp + reason + delta. Do not overwrite.)_
