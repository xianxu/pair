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

1. **`couch start` becomes the console. No new verb.** `couchcmd`'s dispatch table is asserted *identical* to `couchcore.Operations()` (`run_test.go`), so a console-only verb would need an exception to the invariant that keeps the operator's surface and the advisor's from drifting. `start` already blocks for the child's lifetime and already returns a `Handle` the CLI drives — "blocks and owns the tty" narrows that contract rather than inventing one. **The root actor is the first child couch starts**, and `start`'s path argument **defaults to `.`** — so `cd brain && couch start` makes brain home, which is the Spec's "whatever session couch launched in" delivered by convention, with nothing in couch knowing about brain (PQ-4).
   - **Home is chosen by which tree you start first, not by couch's cwd**, and the two coincide only because the default does. `couch start ../pair` from brain deliberately makes *pair* home; that is a legitimate invocation and it is what the M2 single-child smoke uses, but M3's smoke must run the real configuration — couch from brain with no path, pair added as a second child — or the project's headline property is verified against a stand-in.
   - **The launching shell does not come back until couch exits.** couch owns that tty for its lifetime and no key leaves couch — by design, since a switcher that can be escaped into an unmanaged shell is a fourth place to lose track of work. Stated because it is operator-visible: the terminal you type `couch` in is spent.

2. **`--no-console` is the escape hatch, and it announces itself.** It keeps today's `ExecRunner` path (inherit stdio, block, no pty). If the tty layer misbehaves the operator is never stranded — and per the escape-hatch rule the fallback prints a loud line saying the console is off, rather than silently degrading. This also keeps `ExecRunner` a live production path, so its live conformance check stays honest rather than pinning dead code.

3. **The pty is a capability on `Handle`, not a new signature on `Runner`.** `Runner.Start(dir, argv, env) (Handle, error)` is unchanged; a handle from `PtyRunner` additionally satisfies `TerminalHandle`. Rationale: two runners genuinely differ in what they can offer, and widening `Start` would churn every existing caller and fake for a capability only one of them has. The console type-asserts once, at its own boundary.

4. **The reserved row is a scrolling-region reservation, not compositing.** The child pty is `rows-1` tall; the host's scrolling region is pinned to `1..rows-1` (DECSTBM) so a child scrolling at *its* bottom line scrolls inside the region and cannot walk onto the reserved row. The row is painted with save-cursor / absolute-move / paint / restore-cursor so the child's cursor is never disturbed. **This is the design the Spec chose** ("couch does not composite — it reserves a row"); compositing every frame through a `vt.Emulator` is explicitly the rejected alternative.
   - **Known risk, and why it is scheduled early:** apps that set or reset margins themselves (`nvim` emits `\x1b[r` on exit) can drop the reservation. Mitigation is to re-assert the region whenever the row is painted, and to re-assert immediately when the stream scanner sees a margin reset or an alt-screen transition. If real children defeat this, the fallback is to drop the row to an on-demand overlay rather than to start compositing — recorded here so the fallback is a decision, not an improvisation.

5. **Landing repaints from the ring. One mechanism, for every child.** `pair term` already replays a raw buffer to land on a tab running `nvim`, daily, and it works — so the plan's first answer (branch on alt-screen, replay line-mode children and *nudge* alt-screen ones with a resize) invented a second mechanism to solve a problem the existing one has not shown.
   - **The nudge is a documented fallback, not the default.** If the M3 smoke lands garbled on zellij, add it — and accept its real cost, which the plan-quality gate named: `TIOCSWINSZ` raises `SIGWINCH` only when the winsize actually *differs*, so a nudge is a `rows-1 → rows-2 → rows-1` round trip and a visible double reflow of the whole workbench. That is a price worth paying to fix a broken landing and not worth paying speculatively.
   - `Screen.AltScreen` is still tracked — Decision 4 needs it to re-assert margins across an alt-screen transition. It simply stops being an attach-path branch.

6. **Both halves of the terminal plumbing are extracted from `termcmd`, and `pair term` migrates onto both in M1 (ARCH-DRY).** The first draft extracted only the *child* half and left `couchtty.Console` to re-implement the *host* half — `term.MakeRaw` (`termcmd/run.go:222`), `signal.Notify(SIGWINCH)` → `pty.GetsizeFull` (`:244`, `:975-983`), `term.Restore`, and the `\x1b[r` region reset `termcmd.restoreTerminal` already writes (`:1107-1109`). That last one would have put one escape sequence in two packages, against this plan's own one-constant rule, and it is why the first draft's "test the console with fakes, no real tty" and "test the signal path" tasks were unbuildable: there was no injectable host in the type inventory (PQ-2, ARCH-PURE/ARCH-MOCK).
   - **Two packages, one responsibility each.** `ptychild` owns a child on a pty (ring, replay strip, output scanner). `hostty` owns the operator's terminal (size, raw mode, resize notifications, and the terminal-control constants — DECSTBM, save/restore cursor, region reset). `couchtty` and `termcmd` are both clients of both.
   - Original rationale, unchanged: `termcmd.terminalMux` already is a switcher: pty-backed tabs, a 128KB replay ring, redraw-from-snapshot on switch, resize propagation, EOF-driven removal. Building couch's a second time is the duplication ARCH-DRY exists to stop.
   - **What is shared is structure; what stays is policy.** `cmd/internal/ansi`'s doc makes exactly this split, and it applies again: `pair term` cycles numbered tabs and exits when empty; couch switches named actors and falls back to a panel. Those policies stay with their callers.
   - **`stripTerminalQueries` moves and is shared** — both callers replay a raw buffer *to a real terminal*, so the deny-list is one policy with two sites, not two opposed policies (contrast `wrapcmd`, which strips `\x1b[>7u` while `termcmd` requires `\x1b[>1u` to survive; those stay apart).
   - **The migration is the test.** Extracted code with no second consumer is unvalidated new code; `termcmd`'s 1137 lines of existing tests are the regression net that proves the extraction is faithful. That is why M1 migrates rather than deferring it — and why M1 comes first even though it ships no couch behaviour.

7. **Detach is console-scoped, because durability is zellij's, not couch's.** couch's child is `pair` → a zellij **client**; the work (claude, nvim) lives in the zellij **server** session. Killing the console kills the client, and the session survives *detached* — so a fleet already outlives any terminal window one layer below couch, with no daemon involved. Within a console, switching away from a child leaves it running and warm: pty open, ring filling, replay on return.
   - **What `#146` owes is determinism on the way back in**, not durability. See Decision 11 — today a console restart lands on an fzf picker rather than on the session.
   - A daemon (couch's own supervisor plus pty handoff over a socket) is `#147`'s transport and is **not** required by the Done-when's "running and warm". Confirmed by the operator, 2026-08-22.
   - **A claim in the project file this milestone should settle:** `workshop/projects/couch.md` records "`couch stop` is a kill, not a park." If `Stop` SIGTERMs the pair/zellij *client*, the session detaches and the work survives — which is a park. Verified in the M2 smoke (Task 2.7) and the project record corrected either way.

8. **The status row carries the one real activity signal available today: `BEL`.** With no transport (`#147`) the row could only report attach/exit, which makes it decorative — and a row that never says anything useful is dead weight that still costs a terminal row. A child's `\x07` is a genuine "the agent wants you" signal, it is already in the byte stream the scanner reads, and it costs one field. Anything richer (per-actor mailbox depth, git staleness) is `#147`/`#148` and is deliberately absent.

9. **Notices reuse `couchcore.Enqueue`, they do not re-implement it.** The row's rolling feed wants exactly Enqueue's policy: collapse by kind (a second bell from the same actor replaces the first), bounded, never drop control (an exit is control). Keyed as `bell:<ActorID>` / `exit:<ActorID>` so collapse is per-actor rather than global (ARCH-DRY).

10. **`ctrl-space` (`0x00`) is intercepted before the child sees it, and the interceptor returns a SPLIT, not a filtered buffer.** The first draft's `(in []byte) (forward []byte, hits int)` concatenated the bytes either side of the hotkey — but in `x<ctrl-space>y`, `x` belongs to the child the operator is leaving and `y` to the one they land on, and one buffer cannot say that (PQ-1). The repo already has the right shape: `workbenchshortcut.FindChord` returns `(before, chord, raw, rest, ok)` (`shortcut.go:342-352`). couch reuses that **shape**, not that table — the chord set is workbench policy, and merging opposed tables is the bug rather than the cleanup (`cmd/internal/ansi`'s doc makes the same split).
    - **Bracketed paste is the one place this needs state.** A keyboard cannot put `0x00` inside an escape sequence, but a paste can carry arbitrary bytes — and a pasted NUL that silently switches actors *and eats a byte* is a data-loss bug the operator would never diagnose. So the interceptor suspends between `\x1b[200~` and `\x1b[201~`. That is real framing state, and it inherits the repo's rule: buffer only a genuine prefix, consume a complete-but-unsupported control, and test the boundary where the marker splits across two reads. `ansi.Frame`'s `Incomplete` status is what distinguishes the two; the two markers are one constant pair, not one per site. The Spec settled the key; what this plan owes is the audit the repo's own lesson demands ("Never disable an input layer without auditing the escape hatches it provides"). `zellij/config.kdl` binds no Space chord, so nothing in the workbench loses a path. The audit for `claude` and `nvim` is a step in M2, and its result is recorded in the issue `## Log` — including, if something does ride on it, how a literal `ctrl-space` reaches a child.

11. **`Spawn` forces a tag: `pair resume <tag>`, with `tag = launcher.DefaultTag(<worktree root>)`.** `resume` takes `DecideLaunch`'s `ForcedTag` branch — attach when the session is live or detached, create otherwise — and skips the name prompt (`launcher/decision.go:33-37`, `help.go:15`). Today `Spawn` runs `pair --layout2` with **no** tag, and `DecideLaunch` with no tag and a detached session present returns `ActionPick`: an fzf picker inside couch's pty, waiting on the operator. That is what the first minute after a console restart looks like right now.
    - **`--layout2` is dropped, and that is required rather than incidental.** `resume` refuses any third argv element (`launcher/args.go:104`), so `pair resume <tag> --layout2` is a usage error. It is also the right default: an omitted layout flag reuses the tag's recorded layout, a new tag already defaults to layout2, and forcing a layout on a *live* tag makes pair ask before recreating the whole workbench — a prompt the operator would meet inside couch's pty.
    - **The derivation is reused, not re-invented.** `launcher.DefaultTag(path)` is exported and already computes pair's create-flow default from a path (ARCH-DRY).
    - **This is a deliberate slice of `#149`, not a collision with it.** `#149` decides that the tag *is* the space — durable, opaque, several per tree, names as a mutable attribute layer — and supersedes this derivation. What `#146` needs is only that going back in is deterministic; recorded here so the overlap is chosen rather than discovered at `#149`'s plan.

12. **Resolution has one implementation, and the panel injects it.** The first draft had `PanelModel.Filter` match "name, description and repo — the same three fields `couchcore.LookupTrees` matches". That was wrong on the facts and wrong on the principle: `NamingTable.Lookup` matches **Name and Description** only (`naming.go:44-57`), `LookupTrees` adds the agent-published description via `Describe` (`couch.go:196-220`), and **repo is matched nowhere** — path resolution lives a layer up in `ResolveRef`, behind an `ActorID` exact-match branch (`couch.go:228-250`; the exact branch at `:231-235`, the path fallback at `:237-241`). A restated filter would either grow a match the CLI does not have or duplicate two-thirds of a rule that exists, and it would falsify the claim that `#148`'s advisor calls the same resolution (PQ-3).
    - **Shape:** `PanelModel.Filter(query string, resolve func(string) []Worktree)`. The model stays pure and unit-testable with a stub resolver; production passes `couch.LookupTrees`. One rule, three callers (CLI, panel, advisor), no restatement.
    - This is the same guard Task 3.4 applies to *actions*, applied to *resolution*. The panel is not allowed a private verb; it is not allowed a private match rule either.

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
| `Interceptor` | `cmd/internal/couchtty/keys.go` | new |
| `Console` | `cmd/internal/couchtty/console.go` | new (thin IO shell; see the source for its shape) |
| `Reserve` / `Release` / `PaintRow` | `cmd/internal/couchtty/reserve.go` | new |
| terminal-control constants (DECSTBM, cursor save/restore, region reset) | `cmd/internal/hostty/control.go` | new (`\x1b[r` moved from `termcmd/run.go`) |
| `termcmd.restoreTerminal` | `cmd/internal/termcmd/run.go` | modified (now writes `hostty.ResetRegion`; the method stays, the constant moved) |
| `Notice` / `Feed` | `cmd/internal/couchtty/notice.go` | new |

- **Ring** — a bounded byte buffer with a snapshot. `Append([]byte)`, `Snapshot() []byte` (an independent copy). Cap 128KB, lifted from `termcmd.appendBuffer`.
  - **Relationships:** 1:1 with `ptychild.Child`.
  - **DRY rationale:** removes the buffer-trim policy from `termcmd` so one place owns "how much scrollback a detached child keeps".
  - **Future extensions:** a byte cap is a proxy for "enough to land on". If landing proves thin, this widens to a line- or screen-aware cap without any caller changing.

- **StripQueries** — the replay deny-list from `termcmd/queries.go` (#127), moved verbatim with its tests. Removes capability queries from a *replayed* buffer so the repaint cannot re-ask the host terminal and have the answer land in another child's stdin.
  - **DRY rationale:** couch's repaint-on-attach is the same operation `redrawTab` performs. Without the move, couch either re-earns #127's bug or copies its table.
  - **Future extensions:** stays a best-effort deny-list; a missed query degrades to the old behaviour, exactly as documented today.

- **Screen** — the single scanner over a child's output stream. It answers the
  questions the console asks of a child: is it on the alternate screen, does it
  want mouse reporting, has it done something that can drop the reserved row,
  has it rung the bell. Framing goes through `ansi.TerminatorScan`; it does
  **not** frame CSIs itself.
  - **The field list deliberately lives in the code, not here.** Two rounds of
    review caught this table drifting from the shapes it restated
    (`restoreTerminal`, then these accessors), which is the same failure mode
    `atlas/couch.md` records for enumerating couch's operation set in prose: a
    hand-maintained restatement is a second source that drifts. Read
    `ptychild/screen.go`.
  - **DRY rationale:** `termcmd.updateMouseMode` is today's half of this and gets
    folded in — one scanner per package, per the paired-terminator lesson.
  - **Future extensions:** title (OSC 0/2) and OSC 777 notifications are the
    natural next answers; the console's status row is already where they surface.

- **Focus** — `FocusPanel` or `FocusActor(ActorID)`, plus `Up(cur, root) Focus`: a non-root child goes home to the root actor; the root actor goes to the panel; the panel stays. Pure; the whole navigation rule is one function.
  - **DRY rationale:** first occurrence, but the rule is stated in three places (project, issue, atlas) and must have exactly one implementation.
  - **Future extensions:** direct jumps ("to actor N", "to the latest notifier") are deliberately deferred by the Spec; they widen `Up` into a `Move(cur, intent)` without touching the console.

- **PanelModel / Filter / Pick** — the panel as data: rows built from `couchcore.TreeSummary`, and `Pick(digit)` resolving a keystroke to a displayed row. `Filter(query, resolve func(string) []Worktree)` **injects** the match rule rather than restating it; production passes `couch.LookupTrees` (Decision 12). Pure, so a stub resolver tests it and `#148`'s advisor genuinely shares the resolution rather than being claimed to.

- **StatusModel / RenderStatusRow** — `(width int, m StatusModel) string`: the actor chips, the active marker, activity markers, and the newest notice, truncated to width. Pure, so the row is unit-testable without a terminal.

- **Interceptor** — `Feed(in []byte) (before []byte, hit bool, rest []byte)`: the bytes for the *current* focus, whether the hotkey fired, and the bytes for the focus landed on. Holds one piece of state — whether a bracketed paste is open — and a partial-marker buffer for the split-read case (Decision 10).
  - **DRY rationale:** the return shape is `workbenchshortcut.FindChord`'s, deliberately. If a third site ever needs "find a key in a stream and split around it", that is the moment to extract one scanner rather than write a third.
  - **Future extensions:** a second hotkey (the Spec defers direct jumps) widens `hit bool` to a small enum without changing any caller's shape.

- **Reserve / Release / PaintRow** — the escape sequences as pure string builders: `Reserve(rows)` → region + parking, `PaintRow(rows, text)` → save / move / clear / paint / restore, `Release()` → region reset. One constant per sequence, per the paired-terminator lesson.

- **Notice / Feed** — `Notice{Kind, Body, Control}` and a feed that delegates to `couchcore.Enqueue`. `Feed` holds the capacity and the key convention (`bell:<id>`, `exit:<id>`); the policy stays in Enqueue.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ptychild.Child` | `cmd/internal/ptychild/child.go` | new | `creack/pty` + `os/exec` |
| `couchcore.TerminalHandle` | `cmd/internal/couchcore/runner.go` | modified | pty capability on a `Handle` |
| `couchcore.PtyRunner` | `cmd/internal/couchcore/ptyrunner.go` | new | `ptychild.Child` behind `Runner` |
| `FakeRunner` terminal double | `cmd/internal/couchcore/runner_fake.go` | modified | in-memory stand-in for a pty |
| `hostty.Host` | `cmd/internal/hostty/host.go` | new | the operator's terminal: size, raw mode, resize signal |
| `hostty.OSHost` / `hostty.FakeHost` | `cmd/internal/hostty/os.go`, `fake.go` | new | `x/term`, `creack/pty` sizing, `SIGWINCH` |
| `couchtty.Console` | `cmd/internal/couchtty/console.go` | new | drives `hostty.Host` + N `ptychild.Child` |
| `termcmd` host half | `cmd/internal/termcmd/run.go` | modified | `runShell`'s raw/`SIGWINCH`/restore move behind `hostty.Host` |
| `termcmd.terminalTab` | `cmd/internal/termcmd/run.go` | modified | now holds a `ptychild.Child` |
| `couchcmd` wiring | `cmd/internal/couchcmd/run.go` | modified | picks `PtyRunner` vs `ExecRunner` |
| live conformance | `cmd/internal/couchcore/conformance_live_test.go` | modified | `PtyRunner` vs `FakeRunner` |

- **ptychild.Child** — one process on a pty: `Start`, `Write`, `Resize(rows, cols)`, `Snapshot`, `AltScreen`, `Bell`, `Wait`, `Close`. Owns the read pump that feeds `Ring` and `Screen`.
  - **Injected into:** `termcmd.terminalMux` and `couchtty.Console`, both of which keep their own switching policy.
  - **Future extensions:** a `Tee(io.Writer)` for on-disk scrollback (pair already tees `scrollback-<tag>-<agent>.raw`; couch would reuse that file rather than invent a second).

- **couchcore.PtyRunner** — `Runner` whose handles are pty-backed. Constructed with an initial winsize supplier so the first frame is already the right size rather than 80x24-then-resize.
  - **Injected into:** `couchcore.Couch` through the existing seam. Nothing in the domain learns about ptys.

- **FakeRunner terminal double** — the fake's children gain an in-memory terminal: writes are recorded and echoed per a scripted behaviour, resizes are recorded, and exit closes the read side (EOF). ARCH-MOCK: the fake models behaviour across calls, and the live check compares it against a real pty rather than asserting whatever each produces separately.

- **hostty.Host** — the seam over the operator's own terminal: `Size() (rows, cols)`, `MakeRaw() (restore, error)`, `Resized() <-chan struct{}`, and `io.Writer` to the screen. `OSHost` wraps `x/term` + `pty.GetsizeFull` + `signal.Notify(SIGWINCH)`; `FakeHost` is scriptable — a settable size, a resize channel a test can fire, and a captured output buffer.
  - **Injected into:** `couchtty.Console` and `termcmd.runShell`. This is what makes "test the console with no real tty" and "test the signal path" writable at all (PQ-2).
  - **Future extensions:** a remote host (`#120`'s terminal stream) is the same interface over a socket rather than a tty — worth noting, not worth building.

- **couchtty.Console** — the thin IO shell: it drives `hostty.Host` and the per-child pumps and holds **no policy**. Every decision it makes is a call into a pure function above.

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

### Task 1.4a — `hostty`, the host half

**Files:** Create `cmd/internal/hostty/host.go`, `os.go`, `fake.go`, `control.go` (+ tests). Modify `cmd/internal/termcmd/run.go` (`runShell`'s raw-mode block, the `SIGWINCH` goroutine, `captureSize`, `restoreTerminal`).

**Contract:** `Host` is `Size()`, `MakeRaw() (restore, error)`, `Resized() <-chan struct{}`, `io.Writer`. `control.go` holds the terminal-control constants — DECSTBM set/reset, cursor save/restore — **one constant per sequence**, which is the whole reason `\x1b[r` may not stay in `termcmd`.

- [ ] **Tests must catch:** (a) `FakeHost` can report a size, fire a resize, and capture writes — if it cannot, no console test in M2/M4 can be written, which is the finding this task answers; (b) `restore` is idempotent — a console that restores on both the child-exit path and a deferred teardown must not double-restore into a broken state; (c) `Resized()` delivers a *coalesced* signal, not one per syscall — a burst during a window drag must not queue N resizes.
- [ ] **Deletion check:** make `FakeHost.Resized()` a nil channel → (a) red in M2's console test, which is where it matters.
- [ ] Migrate `termcmd`: `runShell` takes a `Host`; `restoreTerminal` keeps its place in the mux and writes `hostty.ResetRegion` instead of a literal. `pair term`'s existing suite is the net, same rule as Task 1.5 — a test that needed editing is a behaviour change, not a fix.
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

### Task 2.3 — `Interceptor`, and the ctrl-space audit

**Files:** Create `cmd/internal/couchtty/keys.go` (+ test).

**Contract:** `Feed(in []byte) (before []byte, hit bool, rest []byte)` — bytes for the current focus, whether `0x00` fired, bytes for the focus landed on. Suspended inside a bracketed paste (Decision 10).

- [ ] **Tests must catch:** (a) `x\x00y` returns `before="x"`, `hit`, `rest="y"` — the split point is the contract, and a concatenated buffer would send `y` to the child being left; (b) two hotkeys in one chunk fire twice with the middle segment routed to the intermediate focus; (c) a `0x00` **inside a bracketed paste** is forwarded, not intercepted — the silent data-loss case; (d) a paste marker split across two `Feed` calls is still recognised, and a **complete-but-unsupported** control is consumed rather than held (the repo's prefix-vs-complete rule); (e) no hotkey → `before` is the whole input and `rest` is empty, so the caller has exactly one place to look.
- [ ] **Add `FuzzInterceptorFeed`** — no panic; `len(before)+len(rest) <= len(in)`; and feeding a stream one byte at a time reaches the same state as feeding it whole.
- [ ] **Deletion check:** drop the paste suspension → (c) red. Return `append(before, rest...)` as one buffer → (a) red.
- [ ] **Audit step (lesson: never disable an input layer without auditing what rides on it):** check `claude` and `nvim` for a `ctrl-space` binding (`zellij/config.kdl` already confirmed clear). Record the result in the issue `## Log`. If something does ride on it, say in the Log how a literal `ctrl-space` reaches a child — do not silently shadow it.
- [ ] Commit.

### Task 2.4 — `Reserve` and `RenderStatusRow`

**Files:** Create `cmd/internal/couchtty/reserve.go`, `statusrow.go` (+ tests).

**Contract:** pure string builders **over `hostty/control.go`'s constants** — this file composes sequences, it does not spell them. `Reserve(rows)` sets the region to `1..rows-1`; `PaintRow(rows, text)` is save / absolute-move / clear-line / text / restore; `Release()` resets the region.

- [ ] **Tests must catch:** (a) the region is `rows-1`, not `rows` — an off-by-one here is the whole bug; (b) `PaintRow` restores the cursor (assert the save/restore pair brackets the paint — without it the child's cursor lands on the status row); (c) `RenderStatusRow` truncates to width without splitting an escape sequence; (d) the active actor is marked distinctly from an actor with pending activity.
- [ ] **Deletion check:** drop the restore from `PaintRow` → (b) red.
- [ ] Commit.

### Task 2.5 — `Console`

**Files:** Create `cmd/internal/couchtty/console.go` (+ test).

**Contract:** the thin IO shell, driving `hostty.Host` (never `x/term` or `signal` directly). `MakeRaw`, `Resized()` → measure and `Resize` the child to `rows-1`, stdin pump through `Interceptor`, child output written to the host only when that child is active, `Release` + `restore` on every exit path.

- [ ] Re-assert `Reserve` whenever the row is painted, and immediately when `Screen` reports `MarginsDirty` or an alt-screen transition (Decision 4).
- [ ] **Tests must catch (driven by `hostty.FakeHost` + `FakeRunner`, no real tty):** (a) a child resized to `rows-1`, never `rows`; (b) an intercepted `ctrl-space` is **not** forwarded to the child; (c) restoration runs when the child exits *and* when the console is torn down mid-stream — a restore that only happens on the happy path leaves the operator's terminal with a broken scroll region; (d) firing `FakeHost`'s resize channel propagates to the child, so the `SIGWINCH` path is covered by a test rather than by the smoke alone.
- [ ] **Deletion check:** remove the `-1` → (a) red. Remove the deferred `Release` → (c) red.
- [ ] Commit.

### Task 2.6 — wire `couch start`

**Files:** Modify `cmd/internal/couchcmd/run.go`, `cmd/internal/couchcore/ops.go`.

**Contract:** `Runtime` gains `NewCouchWith(Runner)`; `NewCouch()` stays `ExecRunner` for `--no-console` and foreign-shell use. `start` declares a `no-console` arg — `FlagOnly`, because it bypasses the console the same way `same-tree` bypasses the guard, and a stray positional word must not be able to set it. `start`'s `path` arg becomes optional, defaulting to `.` (Decision 1).

**What the console displaces, named precisely:** `render`'s `StartResult` branch (`couchcmd/run.go:171-178`) today prints `started <id> on <tree> (pid N)` and then blocks on `Handle.Wait()`. That branch becomes: if the handle is a `TerminalHandle`, construct a `couchtty.Console` over `hostty.OSHost{}` and run it (the console owns the exit code); otherwise keep today's print-and-wait. **`couchcmd` constructs and drives the `Console`** — `couchcore` never learns that a terminal exists.

- [ ] **Tests must catch:** (a) the declared-operations audit still passes (the arg is declared, not smuggled); (b) `couch start x --no-console` takes the `ExecRunner` path and prints the loud fallback line; (c) `couch start x no-console` does **not** — the guard-bypass-never-binds-positionally rule has a test in this repo already; mirror it.
- [ ] Commit.

### Task 2.6a — `Spawn` forces a tag, so going back in is deterministic

**Files:** Modify `cmd/internal/couchcore/couch.go` (the `argv` construction in `Spawn`); test in `couch_test.go`.

**Contract:** `argv = ["pair", "resume", launcher.DefaultTag(<worktree root>)]` plus `args.ExtraArgs`. `--layout2` is **removed** (Decision 11).

- [ ] **Tests must catch:** (a) the argv is `resume <tag>` and the tag derives from the **worktree root**, not from `args.Cwd` — a spawn from `kbench/competition/arc-agi-3/` must resume `kbench`'s tag, since that is the tree couch keyed on; (b) `--layout2` is gone — assert its absence, because leaving it in is a *usage error at runtime* that no unit test would otherwise see; (c) the same tree spawned twice produces the same tag (determinism is the whole point).
- [ ] **Deletion check:** derive the tag from `args.Cwd` instead of the tree → (a) red.
- [ ] Commit.

### Task 2.7 — operator smoke (the milestone's real verification)

- [ ] `make build` then `./bin/couch start ../pair` from `brain`.
- [ ] Confirm, and record each in the issue `## Log` with what was observed: pair + zellij + claude come up inside the pty; the layout is correct at `rows-1`; resizing the terminal reflows the child; the reserved row stays visible while claude streams output; `nvim` opens **and exits** without eating the row (the margin-reset case from Decision 4); `ctrl-space` is intercepted and does not reach the child; quitting restores the terminal (`echo $LINES`, scroll region reset, no raw-mode residue).
- [ ] **Reattach across a console death (Decision 7/11).** `kill -9` the couch process, then re-run `couch start` on the same tree. Confirm and log: the same zellij session comes back with claude still mid-thread — **not** an fzf picker, and **not** a second session. This is the property that makes a daemon unnecessary; if it does not hold, Decision 7 is wrong and the daemon question reopens before M3.
- [ ] **Settle park-vs-kill.** From a second shell, `couch stop <ref>` while the console runs. Record what actually happens to the zellij session: gone (kill) or detached-and-resumable (park). Correct `workshop/projects/couch.md`'s "`couch stop` is a kill, not a park" line to match, in the same commit.
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

**Contract:** rows from `couchcore.TreeSummary` — so parked trees stay listed, dimmed, exactly as `couch list` already renders them. `Filter(query, resolve func(string) []Worktree)` **injects** the match rule and keeps the rows it returns (Decision 12); production passes `couch.LookupTrees`. `Pick(digit)` selects the Nth **displayed** row.

- [ ] **Tests must catch:** (a) `Filter` returns exactly the rows the injected resolver named — with a stub resolver, so the test pins *delegation* rather than re-testing `LookupTrees`; (b) `Pick(2)` after filtering picks the second *filtered* row, not the second underlying one; (c) a parked tree (no live actor) is listed; (d) ordering is stable across refreshes — a list that reorders under the operator's fingers makes numbered selection a hazard.
- [ ] **Deletion check:** have `Filter` do its own `strings.Contains` on `Name` and ignore the resolver → (a) red. That is the exact regression Decision 12 exists to prevent, so the test must fail on it.
- [ ] **Wiring check (one line, in the console):** production passes `couch.LookupTrees` — assert it, or the injection is a seam nothing uses.
- [ ] Commit.

### Task 3.3 — N children in the console

**Files:** Modify `cmd/internal/couchtty/console.go`.

**Contract:** the console holds a map of `ActorID` → child. Only the active child's chunks reach the host; every child's chunks reach its own `Ring` and `Screen`. Attach = `Reserve`, then **replay** — `StripQueries(Snapshot())` after a clear, for every child alike (Decision 5) — then repaint the row.

- [ ] **Tests must catch:** (a) an inactive child's output does not reach the host but does reach its ring — the bug this prevents is a switcher that loses everything said while you were away; (b) the replayed bytes are `StripQueries`'d — a raw replay re-asks the host terminal and the answer lands in the *newly active* child's stdin, which is #127's bug arriving at a new site; (c) attach repaints the status row *after* the child's repaint, so the row is not overwritten by the landing.
- [ ] **Deletion check:** replay `Snapshot()` unstripped → (b) red.
- [ ] Commit.

### Task 3.4 — the panel dispatches through `Operations()`

**Files:** Modify `cmd/internal/couchtty/panel.go`, `console.go`.

**Contract:** `start`, `stop`, `name`, `describe` from the panel call `couchcore.Operations()` — the same table the CLI and (in `#148`) the advisor use. **No second implementation of an operator action.**

- [ ] **Tests must catch:** the panel's action set is a **subset of** `couchcore.OperationNames()`, asserted by name. The existing CLI audit proves the same thing for the CLI; without this one the panel is free to grow a private verb, which is precisely the drift the ops table exists to stop.
- [ ] **Deletion check:** add a panel-only action → the audit goes red.
- [ ] Commit.

### Task 3.5 — operator smoke: two real children, in the real configuration

**Run couch from `brain` with no path** (`cd ~/workspace/brain && couch start`), so the root actor is genuinely brain and "home" is the session `#148` will make the advisor — not the pair-as-root stand-in M2 used (Decision 1, PQ-4).

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
- [ ] Record in the issue `## Log` what the layering actually delivers: couch's child is a zellij *client*, so the console's death costs the view and not the work; warmth beyond the console belongs to zellij's server session plus the forced tag from Task 2.6a, and `#147`'s daemon is not on the path to it.
- [ ] Commit.

### Task 4.4 — restore the terminal on every exit path

**Files:** Modify `cmd/internal/couchtty/console.go`.

- [ ] Region reset, cursor restored, raw mode restored, alt screen left — on normal quit, on last-child exit, and on `SIGTERM`/`SIGHUP` to couch itself.
- [ ] **Tests must catch:** the signal path specifically, driven through `hostty.FakeHost` (which is why Task 1.4a exists). A `defer` covers the happy path and does not run on a signal; a console that leaves the operator's terminal with a pinned scroll region after a `kill` is the worst failure this milestone can ship.
- [ ] Commit.

### Task 4.5 — docs and the map

**Files:** Modify `atlas/couch.md`; verify `couch --help` renders the new arg.

- [ ] Rewrite the atlas's **"There is no pty yet"** and **"Planned, not built"** paragraphs — they are current-state claims that this issue falsifies, and the atlas holds only current state.
- [ ] Add the console and the reserved row to `atlas/couch.md`, and describe `ptychild` **and `hostty`** as shared with `pair term` — name the second consumer in both cases, or the next reader re-derives it. `pair term` is now a client of two extracted packages; `atlas/` must say so, since a reader of `termcmd` alone would not guess it.
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
| detach/reattach leave children running and warm | 4.3, 2.6a | unit (warm across a switch) + 2.7 smoke (warm across a console death) |
| a numbered/direct switch path with no natural-language resolution | 3.2 | unit (`Pick` after filter) + 3.5 smoke |

## Verification before close

- `go build ./...`; `go test ./cmd/... -count=1`; `go test ./cmd/... -race -count=1` (whole tree — a race in the pumps will not show in one package).
- `make test-live` (`PAIR_LIVE_COUCH=1`) — the fake-vs-real pty conformance.
- `make test-term-pane-shortcuts` and the `pair term` smoke from 1.5 — the regression net for **both** migrations (child half in 1.5, host half in 1.4a).
- The operator smokes from 2.7, 3.5 and 4.6, each logged with what was observed rather than "verified".
- `atlas/couch.md` reconciled to what exists (4.5).

## Settled by the operator — 2026-08-22

1. **Detach scope:** console-scoped (Decision 7), with the observation that made it cheap — zellij's server session already outlives the console, so couch needed determinism on re-entry rather than a daemon. Folded in as Decision 11 + Task 2.6a.
2. **`pair term` migration:** extract and migrate in M1 (Decision 6). Its suite is the regression net.
3. **Status row content:** include the BEL activity signal (Decision 8).

## Revisions

_(Append here: timestamp + reason + delta. Do not overwrite.)_

### 2026-08-22 — reattach reframed; three scope calls settled

**Reason:** the operator pointed out that couch hosts `pair`, which runs zellij — so a session is already reattachable beyond a console's lifespan, and the plan's Decision 7 was reasoning about the wrong durability boundary.

**Delta:** Decision 7 rewritten (durability is the zellij server's, not couch's; a daemon is not on the path). New Decision 11 and Task 2.6a: `Spawn` forces `pair resume <tag>` and drops `--layout2`, so a console restart reattaches instead of landing on an fzf picker. Task 2.7 grows two smoke items — reattach across a `kill -9`, and settling whether `couch stop` parks or kills (the project file currently asserts "kill"). Task 4.3 and the acceptance mapping updated to match. Open questions replaced by the operator's answers.

### 2026-08-22 — plan-quality round 1: four blocking findings, fixed at the class

**Reason:** `sdlc change-code`'s plan-quality gate raised PQ-1..PQ-4 (Important) and PQ-5 (Minor). Every factual correction it made was checked against the source and was right. Dispositions, each aimed at the class rather than the named site:

- **PQ-1 `stream-split-contract` — addressed.** `Intercept` becomes `Interceptor.Feed(in) (before, hit, rest)`, adopting `workbenchshortcut.FindChord`'s return shape so the split point is expressible. The class the finding pointed at is bigger than the signature: a *stateless* interceptor could not honour the plan's own split-boundary rule either, so Decision 10 now names bracketed paste as the one place state is real, and Task 2.3 tests the split-read marker and the pasted-NUL data-loss case.
- **PQ-2 `io-seam-unnamed` — addressed.** The class is "one half of the terminal plumbing was extracted and the other left duplicated". Decision 6 now extracts **both**: new `hostty` package (Host seam + `OSHost`/`FakeHost` + the terminal-control constants, `\x1b[r` among them) with `pair term`'s host half migrated onto it in new Task 1.4a. That is what makes Task 2.5's fake-driven tests and Task 4.4's signal-path test writable. Task 2.6 now names `couchcmd/run.go:171-178` as what the console displaces, and says `couchcmd` constructs and drives the `Console`.
- **PQ-3 `resolution-single-source` — addressed.** The stated field list was wrong (`LookupTrees` matches name + operator description + agent description; repo is matched nowhere). New Decision 12: `Filter` **injects** the resolver rather than restating the rule, production passes `couch.LookupTrees`, and Task 3.2's deletion check now fails on a re-implemented `strings.Contains`. Generalised beyond the panel: the guard Task 3.4 applies to actions now applies to resolution too.
- **PQ-4 `home-actor-contract` — addressed.** Decision 1 now states the definition (root actor = first child; `start`'s path defaults to `.`), the ordering it implies (`cd brain && couch start` is what makes brain home), and the launching shell's fate (spent for couch's lifetime; no key leaves couch). Task 3.5's smoke moves to that real configuration instead of verifying the project's headline property against pair-as-root.
- **PQ-5 `resize-nudge-mechanism` — addressed by removal.** `TIOCSWINSZ` only raises `SIGWINCH` on an actual size change, so the nudge cost a `rows-1 → rows-2 → rows-1` double reflow. Rather than accept that, Decision 5 drops the branch: `pair term` already replays a raw buffer to land on an `nvim` tab daily, so replay is the one mechanism for every child and the nudge is a documented fallback if M3's smoke lands garbled. Task 3.3's alt-screen test is replaced by one pinning that the replay is `StripQueries`'d — #127's bug arriving at a new site is the real hazard on that path.

### 2026-08-22 — M1 boundary review: table corrected, one claim retracted

**Reason:** `sdlc milestone-close --milestone M1` returned FIX-THEN-SHIP with five
Important findings. Two of them are about this document rather than the code.

**Delta:**

- **BR-5 `plan-table-drift`.** The Core-concepts table and Task 1.4a both said
  `termcmd.restoreTerminal` is *deleted*. It is not — the method survives at
  `termcmd/run.go` and writes `hostty.ResetRegion`. The behaviour the row was
  about (one escape constant, one site) *is* delivered, so this was a
  table-accuracy defect rather than missing work. Both rows now say `modified`
  and describe what actually happened. Recorded here rather than silently
  edited, per the artifact rule: a table that quietly rewrites itself to match
  the code teaches the next reader nothing.
- **BR-4 `fix-not-pinned-by-failing-test`** is the one worth remembering. M1's
  Log claimed the `Ring` copy-vs-re-slice change fixed unbounded growth and that
  a deletion check pinned it. Neither holds: reverting to the re-slice leaves
  `TestRingDoesNotGrowWithoutBound` green, and measured, re-slicing peaks *lower*
  than copying (cap 48 vs 64) because it forces the next append to reallocate.
  The deletion check I actually ran removed the trim entirely — a different
  mutation, proving a different thing. The code comment and the issue Log are
  corrected; the copy stays as a clarity choice, stated as one.

### 2026-08-22 — M1 boundary review round 2: the Screen row stops restating shapes

**Reason:** `plan-table-drift` came back a second time on this issue (third
counting `pair#145`'s BR-41) — the Core-concepts entry for `Screen` declared
`MarginsDirty` and `Bell`, while the code has `regionLost` and `bell` behind
`Take*` readers.

**Delta:** renaming two words would have been the instance fix and the family
would have returned a third time. The row now describes what `Screen` *answers*
and points at the source for the shapes, which is the rule `atlas/couch.md`
already applies to couch's operation set: stop maintaining a second copy of a
code shape in prose. The same treatment stands ready for any other row that
starts enumerating identifiers.

### 2026-08-23 — M2 boundary review: Decision 11 corrected, and the tables stop restating shapes

**Reason:** the M2 review found this document contradicting the code in four
places (BR-26), one round after the same family was closed for M1 by making the
`Screen` row stop enumerating identifiers.

**Delta:**

- **Decision 11's central claim was wrong and is corrected.** It said `resume`
  refuses any third argv element, and dropped `--layout2` on that basis. Only
  POSITIONALS are refused: `ParseArgs` runs `extractLayoutRequest` first, so
  layout flags never reach the guard. Measured, and now pinned in
  `launcher/args_test.go`. `--layout2` is back, by operator decision — couch
  owns terminal switching, so layout3's third pane is the layer couch replaces.
- **Decision 11 is also narrower than it claimed.** It said the forced tag
  removes pair's prompts. It removes the NAME prompt and `DecideLaunch`'s session
  picker; `runConfigPicker`'s saved-config prompt still fires on a cold start of
  a tag with a saved config. Left deliberately (operator, 2026-08-22) and
  recorded on `pair#149`, which owns the identity model that would let couch skip
  it.
- **Decision 5's "one mechanism" held, and Decision 4's fallback was not
  needed** — confirmed by operator smoke on the real stack 2026-08-23.
- **The `Screen` row's treatment is extended to the rest of the table.** Rows
  now name what an entity ANSWERS and point at the source rather than
  restating field lists, which is what kept this family recurring. The register
  of what shipped is the code plus the issue `## Log`; this document records the
  DECISIONS and stops competing with the source for the shapes.
