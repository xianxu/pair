# Boundary Review — pair#146 (milestone M2)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 42b268852aea0407204cff2472821961aea388fa..418daccf9980afbd4b7cb86ffb68b9057a8dd36b |
| command | sdlc milestone-close --issue 146 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-23T08:39:37-07:00 |
| verdict | REWORK |

## Review

I have everything I need. Here is the review.

```verdict
verdict: REWORK
confidence: high
```

M2's headline claim — "`couch start` IS the console, and the reserved row survives real children" — is genuinely delivered and the row half is verified to a standard most milestones don't reach (a real vt emulator, a real pty child, a real operator stack). I re-ran the two load-bearing deletion checks and both hold. What blocks SHIP is that the *other* two fixes this window claims are unsound rather than incomplete, and I reproduced both with running code: (1) `Child.MidSequence()` describes the newest chunk the pump has **read**, not the chunk the console has **written**, so as soon as the console lags the pump — the burst case the guard exists for — it paints straight into the child's escape sequence again, byte-for-byte the corruption the commit says it fixed; and (2) the Interceptor parks a lone `ESC` indefinitely and then delivers it glued to the next keystroke as a meta prefix, in a console whose Spec names ESC as interrupt-in-Claude and mode-switch-in-nvim. A third: `couch start` where stdin isn't a terminal spawns and registers the child, then exits 1 with **no output at all**, because `hostty.MakeRaw`'s error is discarded. Underneath those, the milestone's central wiring is unpinned — I disabled `consoleRunner` entirely (`couch start` silently reverts to stdio) and the whole suite stayed green.

## 1. Strengths

- **The reserved-row ladder is real verification, not assertion.** `vtscreen_test.go` drives the actual `Console` against `charmbracelet/x/vt` and reads the rendered screen; `console_live_test.go` puts a real nvim on a real pty under it. I re-ran both claimed deletion checks: removing `case 'J'` from `screen.go:259` reddens 4 subtests of `TestScreenTreatsAnEraseAsRowDirty` plus `TestReservedRowComesBackAfterAChildClearsTheScreen`; removing the `Reserve` re-assert at `console.go:213` reddens `TestConsoleReassertsTheRegionWhenAChildDropsIt` and `TestReservedRowComesBackAfterAChildResetsMargins`. Both load-bearing.
- **`TerminalHandle` as a capability, with a runner that FAILS it.** `ptyrunner_test.go:24` asserts `ExecRunner`'s handle is *not* a `TerminalHandle`. A capability check no implementation can fail is vacuous; this one can't drift into that.
- **`Terminal()` returning the concrete `*ptychild.Child`** (`ptyrunner.go:40`) so `FakeRunner`'s double is the same type — production flow and test flow share the boundary exactly (ARCH-MOCK done right).
- **"One child, one notion of exited"** (`runner_fake.go:91,139`) with two tests. That divergence hung a test rather than failing it, which is the worst shape of fake drift, and it's now pinned on both the `SetExited` and `AutoExit` paths.
- **`launcher/args_test.go:322`** pins both directions of the `resume` positional guard and corrects a plan claim by measurement rather than argument. Exactly the right response to "I read the guard and stopped there".
- **The `lessons.md` entry on async assertions** (marker-behind-stimulus, and "ask what the assertion reports *before* the action runs") is the most transferable thing in this window, and it was applied in `vtscreen_test.go:198-210`.

## 2. Critical findings

**C1 — `MidSequence()` guards the wrong stream position; the splice it fixed is still reachable.** `console.go:181-195`, `console.go:234`, `ptychild/child.go:181-195`.

The pump feeds `Screen` **before** calling the sink (`child.go:113-120`), and the sink hands chunks to a 256-deep channel the console drains later. So when the console has written chunk *N* to the host and asks `MidSequence()`, the answer describes chunk *M ≥ N*. If *N* ends mid-sequence and *M* doesn't, the guard says "safe" and the paint lands between *N* and *N+1*. Reproduced in a scratch copy by holding the console inside `host.Write` (a slow tty, or plain scheduling) while the child completes the sequence:

```
\x1b[2J\x1b[38;2;76 \x1b[1;23r\x1b7\x1b[24;1H\x1b[2K[brain]\x1b8 ;82;88mCOLOURED
```

That is the reported nvim corruption verbatim. `console_test.go:191-196`'s own comment describes this ordering and concludes the bug "does not reproduce" — the opposite is true; the shipped test avoids the window rather than covering it, which is also why it is mildly racy.

Second failure mode, same guard: `Screen.Pending()` is `len(s.pending)`, which is **0** while `skipping != skipNone`. Measured — after 70 KB of an unterminated OSC 52 fed in 4096-byte chunks, `Pending() == 0`, so `MidSequence()` reports false in the middle of a sequence. That is the clipboard case this repo already documents as the everyday trigger.

Fix sketch: the property is about the bytes the console has **emitted**, so compute it there — give each pane its own `ptychild.Screen` fed with each chunk as it is written to the host, and gate on that (still one scanner, still ARCH-DRY; `console_live_test.go:splicedPaint` already models it correctly in the harness). Fold `skipping != skipNone` into whatever "mid-sequence" means. And serialize every host write through one point: today `onChunk`, `watchResize` and `pumpStdin`→`onHotkey` all write to `c.host` with no ordering between them.

**C2 — a lone `ESC` keystroke is parked, then delivered as a meta prefix.** `keys.go:127-141`, `keys.go:88-95`.

`sequenceAt` classifies a bare `\x1b` as `seqPartial` (it is a genuine prefix of `\x1b[200~`), so `Feed` holds it and returns nothing. Measured: `Feed("\x1b")` → `before=""`; the next `Feed("i")` → `before="\x1bi"`, which a child reads as Alt+i, not ESC-then-i. End to end through the console, an ESC keystroke never reaches the child at all. Held bytes are also never flushed at teardown.

ESC is interrupt in Claude Code and mode-switch in nvim — the Spec cites both as the reason double-ESC was rejected. Honest caveat: once zellij's Kitty protocol is up, the terminal likely sends ESC as CSI-u `\x1b[27u`, which falls through `seqNone` and is forwarded — which is plausibly why the smoke didn't see it. It still bites before the child enables KKP (pair's config picker is exactly that window), after it disables it, and for M3's panel.

Fix sketch: never park a partial that is itself a complete key — hold only from two bytes (`\x1b[`) on, or flush the held run when a read ends without extending the match. Add the negative test; `FuzzInterceptorFeed` cannot find this because it only bounds output length, not retention. Note `workbenchshortcut.FindChord` (`shortcut.go:342`) — the shape this deliberately copies — never holds anything.

**C3 — `couch start` off a tty spawns and registers the child, then exits 1 in silence.** `console.go:103-106`, `couchcmd/run.go:157-170`.

`consoleRunner` type-asserts `stdin`/`stdout` to `*os.File` and never asks whether they are terminals — while `dimCodes` at `run.go:305` in the same file already uses `term.IsTerminal`. With stdin a pipe (any script, cron, or an agent shell — including `#148`'s advisor), `host.Size()` fails so `Console` keeps `size{0,0}` → `ChildSize()` = 0×0 → the child gets a **0-row pty**; then `MakeRaw()` fails and `Run` discards the error and returns 1. Measured through the real CLI entrypoint: `exit code = 1, stderr = ""`. The actor record persists, the child is hung up when the process exits, and nothing mentions `--no-console`.

Fix sketch: in `consoleRunner`, require `term.IsTerminal` on both fds and otherwise take the announced `ExecRunner` path; and have `Run` report the `MakeRaw`/`Size` errors rather than collapsing them into `1`.

## 3. Important findings

**I1 — `fix-not-pinned-by-failing-test`, 3rd on this issue.** Earlier rounds fixed instances. Do NOT fix these two — state the rule and sweep it. Rule, extending round 2's ("a fix is defended only by a test the default suite executes and that goes red without it"): **the mutation you must run is the removal of the *wiring*, not of a helper — every behaviour a boundary claims, including its central one, needs a default-suite test that goes red when that behaviour is deleted.** Measured prevalence in this window, both green under mutation: (a) forcing `consoleRunner` to always return `(nil, ExecRunner{})` — i.e. `couch start` is no longer the console at all — leaves `./cmd/internal/couchcmd` and `./cmd/internal/couchtty` **ok**; `run_test.go:34-40`'s comment claims the branch "is observable in the rendered output", and it is not, because only the `--no-console` side asserts anything; (b) deleting the `path == "" → "."` default in `ops.go:78-81` (Decision 1's "`cd brain && couch start` is what makes brain home") leaves the tree green. The console branch is *executed* by existing CLI tests but *asserted* by none.

**I2 — `probe-hygiene`, 3rd on this issue.** Do NOT just add a target for `zellijpark`. The rule was already stated in round 2 — "a committed probe is a first-class artifact: a make target, an `atlas/` entry, and self-cleanup" — and the very next probe skipped two of the three. `probes/zellijpark` has self-cleanup but `make test-smoke` (`Makefile.local:50-51`) still runs only `termsmoke`, and `atlas/index.md:17-21` still names only `termsmoke`, while zellijpark's output is quoted as the evidence that corrected `workshop/projects/couch.md`. The class fix is to make the target enumerate the directory (`go run ./probes/<each>`) so adding a probe cannot skip it, and to make the atlas entry describe the convention rather than list members.

**I3 — `plan-table-drift`, 3rd on this issue (4th counting `#145` BR-41).** Do NOT correct these three rows. The rule from round 2 was "stop maintaining a second copy of a code shape in prose"; this round shows the same rule is owed by *decisions and task contracts*, not just tables: **a plan statement that asserts external behaviour must be measured before it is written, and a boundary that reverses a Decision writes the `## Revisions` entry in the same window.** Measured drift, none of it in Revisions: (a) Decision 11 and Task 2.6a still say `--layout2` is **removed** and that "`resume` refuses any third argv element (`launcher/args.go:104`)" — false, and the code + `couch_test.go:588` now pin the opposite; (b) `StatusModel`/`RenderStatusRow` are declared at `couchtty/statusrow.go`, which does not exist (they are in `reserve.go`); (c) `couchcore.TerminalHandle` is declared at `couchcore/runner.go`; it is in `ptyrunner.go`, and Task 2.1's contract still specifies `Terminal() Terminal` as an interface where the code deliberately returns the concrete `*ptychild.Child`. Task 1.3:199 and Task 2.5:296 still say `MarginsDirty` — flagged as a residual at M1 round 4 and now consumed by an implementer.

**I4 — `dead-field-and-leaked-consumer`, 2nd on this issue.** Do NOT delete just one. Rule: **a fake's surface, and a model field, must be exercised by the flow it exists for; a member added for symmetry with production and set at zero call sites is decoration that reads as coverage.** Measured, three members added this window with no writer anywhere in the tree: `FakeRunner.Sink` (`runner_fake.go:48` — production uses `PtyRunner`, and `testRT.NewCouchWith` discards the runner, so nothing ever sets it), `FakeRunner.Emit` (`runner_fake.go:228` — written for the snapshot-content conformance design that was then abandoned), and `StatusActor.Bell`, which `paintNow` hardcodes to `false` (`console.go:207`) so the `*` marker `reserve_test.go:58` pins can never appear in production.

**I5 — Task 2.3's ctrl-space audit was not performed, or not recorded.** The plan makes it a deliverable with an explicit clause: "check `claude` and `nvim` for a `ctrl-space` binding … Record the result in the issue `## Log`. If something does ride on it, say in the Log how a literal `ctrl-space` reaches a child." The Log mentions "the ctrl-space audit" only retrospectively (`:750`); its result appears nowhere. This is not ceremony: in Vim, `<C-Space>` maps to `<C-@>`, which in insert mode is `i_CTRL-@` (insert-last-text-and-stop-insert), and couch now shadows the key in both encodings with no documented escape hatch.

**I6 — `Deliver` drops child output silently, justified by a recovery that does not exist at this boundary.** `console.go:76-83`. The comment says a full buffer "drops the chunk from the LIVE path only -- the ring still has it, so the next repaint is correct". `grep Replay\|Snapshot cmd/internal/couchtty/*.go` (non-test) is **empty**: M2 has no repaint-from-ring, so a dropped chunk is gone from the screen permanently, and if it was mid-sequence the host terminal is left corrupted with nothing to resync it. Either block with a bounded wait, or latch a "this pane lost bytes" flag that forces a full repaint once M3 lands the ring replay — and correct the comment now.

**I7 — the forced tag is deterministic but not unique, so two trees can resume one session.** `couch.go:107`. `launcher.DefaultTag` is `NormalizeDisplayComponent(filepath.Base(path))` (`tag.go:25-31`), so `~/workspace/pair` and any other tree whose basename is `pair` — a git worktree, a clone under a different parent, `foo.bar` vs `foo_bar` — derive the same tag. `couch start` on the second tree then attaches to the first tree's zellij session while the registry records an actor against tree B: couch's one-agent-per-**tree** invariant and the session the operator lands in stop corresponding. `TestSpawnDerivesTheTagFromTheTreeNotTheCwd` pins derivation; nothing pins uniqueness. Cheap mitigation until `#149`: suffix a short hash of the full tree path.

**I8 — docs lag the surface this window added.** Two sites: `README.md:260-265` still describes couch as "a supervisor that registers agent sessions one-per-worktree and can spawn them" — it now takes over the operator's terminal for its lifetime, intercepts `ctrl-space`, reserves a row, and has a new `--no-console` flag; that is user-facing surface a reader types. And `atlas/architecture.md:458-460` still describes `Screen` as reporting "region-lost edges (DECSTBM, RIS, or an alt-screen transition)" — the rename to row-dirty and the addition of **erase**, which is the entire discovery of this milestone, did not reach it, nor did `MidSequence` (new public surface on `ptychild.Child` that the console's correctness depends on). `atlas/couch.md` itself is updated well.

## 4. Minor findings

- `ChildRows(0) == 0` while its doc says "It never returns zero"; only `ChildRows(1)` is tested (`reserve.go:21`, `reserve_test.go:31`). *2nd in `uncovered-negative-assertion`* — rule: an invariant asserted in a doc comment needs a test at its boundary case, or the comment is the lie. Reachable via C3.
- `Console.Run` never calls `c.Stop()` or `host.Close()`, so `watchResize` and `OSHost`'s SIGWINCH registration outlive the console. *2nd in `signal-goroutine-outlives-close`* — rule: teardown is explicit and ordered, not emergent from process exit. Latent while one console per process; `ptyrunner.go:88`'s own comment anticipates `#147` putting more than one in.
- `classify`'s `case 'J'` comment says "ED, every form", but `\x1b[?2J` (DECSED) returns early at the private-introducer branch (`screen.go:238`).
- `Screen.Pending`'s doc says "Exported for the tests that pin the bound"; it now has a production consumer (`MidSequence`).
- `Run`'s `select` between `c.chunks` and `exited` is a coin flip on exit, so the child's final output is likely dropped before `release()` clears the row.
- `conformance_live_test.go:245-249` reads `doneBeforeExit` *after* the write that makes the child exit, and then `Fatalf`s if it is true — racy by construction.
- `vtscreen_test.go` redefines `min` (shadowing the Go builtin), and `waitFor`/`waitLong`/`waitUntilTrue` are three near-identical polling helpers across two packages (ARCH-DRY, test-side).
- `ops.go:65` says the default is applied "at the CLI"; it is applied in `couchcore`'s `Invoke`. `atlas/couch.md:32` reads "hands the child its own stdio and block".

## 5. Test coverage notes

- Pinned well: the ED→row-dirty fix, the region re-assertion, the fake/real exit unification, both `resume` argv properties, and the Kitty CSI-u interception including the negative (`TestInterceptorForwardsOtherKittyChordsUntouched`), which is the test that matters most there.
- Not pinned: the console wiring itself and the `path` default (I1); `runConsole`'s non-`TerminalHandle` fallback branch; any assertion that `couch start` *without* `--no-console` behaves differently from with it.
- Wrong-window tests: `TestConsoleNeverInjectsInsideAChildEscapeSequence` covers the caught-up case and is written to avoid the lagging one (C1). The suite has no test where the console is behind the pump — which is the only state the guard exists for.
- Missing from the plan's own list: `FuzzInterceptorFeed` was specified to assert that "feeding a stream one byte at a time reaches the same state as feeding it whole"; the shipped fuzz only bounds output length. That property is what would have found C2.
- Environment, unchanged from M1: pty-backed tests fail loudly in the agent shell (`ptychild`, `hostty`, `termcmd`, `couchcore`'s pty tests); `keyscmd` fails on `mktemp` perms. `go vet ./cmd/...` clean; `./cmd/internal/couchtty -race -count=2` clean.

## 6. Architectural notes

- **ARCH-DRY — pass, with one flag.** `MidSequence` reusing `Screen` instead of a second parser is the right instinct (its *position* is the bug, not its reuse); `reserve.go` composes `hostty` constants rather than spelling escapes; `Spawn` reuses `launcher.DefaultTag`; declining to merge the workbench chord table is correctly argued. Flag: `consoleRunner` re-decides "am I on a terminal" by an unchecked type assertion while `term.IsTerminal` sits 140 lines below it in the same file (cited in C3).
- **ARCH-PURE — pass.** `keys.go` and `reserve.go` are pure and their tests need no IO; `Console` is a thin shell over `hostty.Host`. Residual policy in the shell: the notice strings at `console.go:246,313` and the hardcoded `Bell: false` at `:207` are model decisions living in the IO layer, which is why `StatusActor.Bell` is dead (I4).
- **ARCH-PURPOSE — flag.** The purpose ("`couch start` IS the console") is in the code but not enforced anywhere: deleting the wiring is invisible to the suite (I1). The ctrl-space deliverable is the easy subset — the encoding was fixed after the smoke, but the audit that the plan made part of the same task was not recorded (I5), and the interceptor now shadows a key it also parks (C2).
- **ARCH-MOCK — flag.** The fake growing a terminal *of the same type*, plus the new lifecycle-predicate conformance test (contract predicates, not hand-fed content), is the strongest ARCH-MOCK work in this project so far. But the seam does not hold at the CLI: `testRT.NewCouchWith` deliberately discards the injected runner, so production wiring and test wiring do not share that boundary — which is precisely how I1's mutation survives — and `FakeRunner.Sink`/`Emit` exist unexercised (I4). Before M3 multiplies children, the console needs a fake-driven path from `RunWithRuntime` down, which means a host seam in `consoleRunner` too.
- **Forward risk for M3:** `onChunk` calls `TakeRowDirty()` for every child but acts only `if rowDirty && isActive` (`console.go:243-249`) — the latch is consumed and discarded for inactive children, so an inactive child's dirty row is lost by the time the operator switches to it. Cheap to fix now, invisible with one child.

## 7. Plan revision recommendations

Append one `## Revisions` entry covering all of these (per I3, the point is the rule, not the individual rows):

1. **Decision 11 / Task 2.6a — `--layout2` is reinstated by operator decision (2026-08-22), and the stated reason for dropping it was false.** `resume` refuses stray *positionals* only; `ParseArgs` runs `extractLayoutRequest` before the guard. Both properties are now pinned in `launcher/args_test.go`. Delete Task 2.6a's test (b) ("assert `--layout2` is gone") — it now asserts the opposite of the code.
2. **Core concepts / Integration tables** — `StatusModel`/`RenderStatusRow` live in `couchtty/reserve.go`, not `statusrow.go`; `TerminalHandle` lives in `couchcore/ptyrunner.go`, not `runner.go`. Preferably apply the round-2 treatment: drop the restated paths and point at the package.
3. **Task 2.1's contract** — `Terminal()` returns the concrete `*ptychild.Child`, not a `Terminal` interface, and the reason (the fake *is* that type) is a decision worth keeping in the plan rather than only in the Log.
4. **Task 1.3:199 and Task 2.5:296** — `MarginsDirty` → `TakeRowDirty`, and the signal now includes ED. This was flagged as a residual at M1 round 4 and has since misdirected an implementer once.
5. **Task 2.7** — record that the `kill -9` reattach and the park-vs-kill determination *through the full `couch stop` path* are carried to M3 (the Log says so; the plan and the acceptance mapping still assign them to 2.7).
6. **Task 2.3** — the `FuzzInterceptorFeed` chunk-invariance property was specified and not implemented; either implement it or record why the length bound replaced it.

```findings
findings:
  - id: new
    severity: Critical
    family: guard-reads-wrong-stream-position
    title: |
      MidSequence() describes the newest chunk READ, not the chunk WRITTEN, so the row paint still splices into the child's escape sequences
    detail: |
      ptychild's pump feeds Screen before calling the sink (child.go:113-120) and the console
      drains a 256-deep channel later, so when console.go:182 asks MidSequence() about the chunk
      it just wrote, the answer describes a later chunk. Reproduced in a scratch copy by holding
      the console inside host.Write while the child completes the sequence -- emitted stream was
      "\x1b[2J\x1b[38;2;76" + "\x1b[1;23r\x1b7\x1b[24;1H\x1b[2K[brain]\x1b8" + ";82;88mCOLOURED",
      byte-for-byte the nvim corruption this commit claims to fix. console_test.go:191-196
      describes this exact ordering and concludes the bug "does not reproduce"; the shipped test
      avoids the window rather than covering it. Second mode, measured: Screen.Pending() is
      len(pending), which is 0 while skipping != skipNone, so MidSequence() returns false in the
      middle of an over-64KiB OSC 52 -- the everyday clipboard case this repo already documents.
      Fix: compute pending-ness over the bytes the console has EMITTED (a per-pane ptychild.Screen
      fed as chunks are written, which is what console_live_test.go's splicedPaint already does),
      fold in the skipping state, and serialise host writes -- onChunk, watchResize and onHotkey
      all write to c.host today with no ordering between them.
  - id: new
    severity: Critical
    family: prefix-parks-a-complete-key
    title: |
      a lone ESC keystroke is held indefinitely by the Interceptor and then delivered glued to the next key as a meta prefix
    detail: |
      sequenceAt (keys.go:127-141) classifies a bare 0x1b as seqPartial because it is a real prefix
      of "\x1b[200~", so Feed holds it (keys.go:93) and returns nothing. Measured: Feed("\x1b") ->
      before=""; the following Feed("i") -> before="\x1bi", which a child reads as Alt+i. Driven
      end to end through the console, an ESC keystroke never reaches the child at all, and held
      bytes are never flushed at teardown. ESC is interrupt in Claude Code and mode-switch in nvim
      -- the Spec cites both. Caveat stated honestly: once zellij's Kitty protocol is up the
      terminal likely sends "\x1b[27u", which falls through as seqNone and is forwarded, which is
      plausibly why the smoke did not see it; the legacy encoding still governs before the child
      enables KKP (pair's config picker), after it disables it, and for M3's panel. Fix: hold only
      from two bytes on, or flush a held run when a read ends without extending the match, and add
      the negative test -- FuzzInterceptorFeed bounds output length only, so it cannot find this.
      workbenchshortcut.FindChord, the shape this copies, holds nothing.
  - id: new
    severity: Critical
    family: swallowed-seam-error
    title: |
      couch start off a tty spawns and registers the child, gives it a 0-row pty, then exits 1 with no output
    detail: |
      consoleRunner (run.go:157-170) type-asserts stdin/stdout to *os.File and never asks whether
      they are terminals, while dimCodes at run.go:305 in the same file already uses
      term.IsTerminal. With stdin a pipe -- any script, cron, or agent shell, including #148's
      advisor -- host.Size() fails so Console keeps size{0,0}, ChildSize() returns 0x0 and the
      child is started on a zero-row pty; then MakeRaw() fails and console.go:103-106 discards the
      wrapped error and returns 1. Measured through RunWithRuntime: exit code 1, stderr "". The
      actor record persists, the child is hung up at process exit, and nothing mentions
      --no-console. Fix: gate the console on term.IsTerminal for both fds and fall back to the
      announced ExecRunner path; report the MakeRaw/Size errors instead of collapsing them to 1.
  - id: new
    severity: Important
    family: fix-not-pinned-by-failing-test
    title: |
      the milestone's central wiring is unpinned -- disabling the console entirely leaves the whole suite green
    detail: |
      3rd in this family on this issue. Do NOT fix these two instances -- the rule, extending round
      2's, is that the mutation you must run is removal of the WIRING, not of a helper: every
      behaviour a boundary claims, including its central one, needs a default-suite test that goes
      red when that behaviour is deleted. Measured, both green under mutation: (a) forcing
      consoleRunner to always return (nil, ExecRunner) -- couch start is no longer the console --
      leaves ./cmd/internal/couchcmd and ./cmd/internal/couchtty ok, though run_test.go:34-40
      claims the branch "is observable in the rendered output"; only the --no-console side asserts
      anything. (b) deleting the path == "" -> "." default in ops.go:78-81, which is Decision 1's
      "cd brain and couch start is what makes brain home", leaves the tree green.
  - id: new
    severity: Important
    family: probe-hygiene
    title: |
      probes/zellijpark ships with no make target and no atlas entry, one round after that rule was written
    detail: |
      3rd in this family. Do NOT just add a target for zellijpark. The rule was already stated in
      round 2 -- a committed probe is a first-class artifact: a make target, an atlas/ entry, and
      self-cleanup -- and the very next probe skipped two of three. make test-smoke
      (Makefile.local:50-51) runs only termsmoke; atlas/index.md:17-21 names only termsmoke; and
      zellijpark's output is quoted as the evidence that corrected workshop/projects/couch.md's
      park-vs-kill claim. Class fix: make the target enumerate probes/ so a new probe cannot skip
      it, and state the convention in the atlas entry rather than listing members.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      the plan's Decision 11 and three Core-concepts rows now contradict the code, with no Revisions entry
    detail: |
      3rd in this family on this issue (4th counting pair#145 BR-41). Do NOT correct the rows. The
      round-2 rule was "stop maintaining a second copy of a code shape in prose"; this round shows
      it is owed by DECISIONS and TASK CONTRACTS too -- a plan statement asserting external
      behaviour must be measured before it is written, and a boundary that reverses a Decision
      writes the Revisions entry in the same window. Measured drift, none of it recorded: Decision
      11 and Task 2.6a still say --layout2 is removed and that resume "refuses any third argv
      element (launcher/args.go:104)", which is false and which the code plus couch_test.go:588
      now pin the opposite of; StatusModel/RenderStatusRow are declared at couchtty/statusrow.go,
      a file that does not exist (they are in reserve.go); couchcore.TerminalHandle is declared at
      couchcore/runner.go but lives in ptyrunner.go, and Task 2.1 still specifies Terminal() as an
      interface where the code deliberately returns *ptychild.Child. Task 1.3:199 and Task 2.5:296
      still say MarginsDirty, flagged as a residual at M1 round 4 and since consumed.
  - id: new
    severity: Important
    family: dead-field-and-leaked-consumer
    title: |
      three members added this window have zero writers -- FakeRunner.Sink, FakeRunner.Emit, and StatusActor.Bell
    detail: |
      2nd in this family. Do NOT delete just one. The rule: a fake's surface, and a model field,
      must be exercised by the flow it exists for; a member added for symmetry with production and
      set at zero call sites is decoration that reads as coverage. Measured: FakeRunner.Sink
      (runner_fake.go:48) is never assigned anywhere in the tree -- production uses PtyRunner and
      testRT.NewCouchWith discards the runner; FakeRunner.Emit (runner_fake.go:228) has no callers,
      left over from the snapshot-content conformance design the file's own comment says was
      abandoned; StatusActor.Bell is hardcoded false at console.go:207, so the "*" marker
      reserve_test.go:58 pins can never appear in production.
  - id: new
    severity: Important
    family: undelivered-plan-step
    title: |
      Task 2.3's ctrl-space audit of claude and nvim was never recorded in the issue Log
    detail: |
      The plan makes it a deliverable with an explicit clause -- check claude and nvim for a
      ctrl-space binding, record the result in the issue Log, and if something rides on it say how
      a literal ctrl-space reaches a child. The Log mentions "the ctrl-space audit" only
      retrospectively at :750; its result appears nowhere. Not ceremony: in Vim, <C-Space> maps to
      <C-@>, which in insert mode is i_CTRL-@, and couch now shadows the key in both encodings
      with no documented escape hatch. An unrecorded audit is an unperformed audit.
  - id: new
    severity: Important
    family: unrecoverable-silent-drop
    title: |
      Deliver drops child output on a full buffer, justified by a repaint-from-ring that does not exist at this boundary
    detail: |
      console.go:76-83's default case drops the chunk and the comment says "the ring still has it,
      so the next repaint is correct". grep for Replay/Snapshot over couchtty's non-test files is
      empty: M2 has no repaint-from-ring, so a dropped chunk is gone from the screen permanently,
      and a chunk dropped mid-sequence leaves the host terminal corrupted with nothing to resync
      it. Either block with a bounded wait, or latch a "this pane lost bytes" flag that forces a
      full repaint once M3 lands the replay -- and correct the comment now, since it is a forward
      reference presented as a present-tense guarantee.
  - id: new
    severity: Important
    family: derived-id-not-unique
    title: |
      the forced resume tag is the tree's basename, so two different trees resume one zellij session
    detail: |
      couch.go:107 uses launcher.DefaultTag, which is NormalizeDisplayComponent(filepath.Base(path))
      (tag.go:25-31). Any two trees sharing a basename -- a git worktree, a clone under a different
      parent, foo.bar vs foo_bar which both normalise to foo_bar -- derive the same tag, so couch
      start on the second attaches to the first's session while the registry records an actor
      against the second tree. That breaks the correspondence between couch's one-agent-per-TREE
      key and the session the operator lands in. TestSpawnDerivesTheTagFromTheTreeNotTheCwd pins
      derivation; nothing pins uniqueness. Cheap mitigation until #149: suffix a short hash of the
      full tree path.
  - id: new
    severity: Important
    family: docs-lag-the-surface
    title: |
      README still describes couch as a spawner, and atlas/architecture.md still describes Screen as reporting region-lost
    detail: |
      README.md:260-265 says couch "registers agent sessions one-per-worktree and can spawn them";
      couch start now owns the operator's terminal for its lifetime, intercepts ctrl-space,
      reserves a row, and has a new --no-console flag -- user-facing surface a reader types.
      atlas/architecture.md:458-460 still describes Screen as reporting "region-lost edges
      (DECSTBM, RIS, or an alt-screen transition)", omitting the ERASE case that is the entire
      discovery of this milestone, and does not mention MidSequence, new public surface on
      ptychild.Child that the console's correctness depends on. atlas/couch.md itself is updated
      well; these two are the same window's other doc sites.
  - id: new
    severity: Minor
    family: uncovered-negative-assertion
    title: |
      ChildRows(0) returns 0 while its doc says "It never returns zero", and no test covers the boundary case
    detail: |
      2nd in this family. reserve.go:21-26 returns hostRows for hostRows <= 1, so ChildRows(0) is
      0; reserve_test.go:31 tests only ChildRows(1). The rule: an invariant asserted in a doc
      comment needs a test at its boundary case, or the comment is the lie. Reachable in
      production via the non-tty path (see the swallowed-seam-error finding), where it produces a
      zero-row pty.
  - id: new
    severity: Minor
    family: signal-goroutine-outlives-close
    title: |
      Console.Run never calls Stop() or host.Close(), so the resize watcher and the SIGWINCH registration outlive the console
    detail: |
      2nd in this family. console.go:102-130 defers restore() and release() but nothing stops the
      console's own goroutines or closes the host. The rule from M1's BR-2 applies unchanged:
      teardown is explicit and ordered, not emergent -- here from process exit rather than from
      defer LIFO. Latent while there is one console per process; ptyrunner.go:88's own comment
      anticipates #147 putting more than one in.
  - id: new
    severity: Minor
    family: stale-comment-reference
    title: |
      several comments overstate or misplace what the code does
    detail: |
      2nd in this family; low value individually, listed together. screen.go's case 'J' says "ED,
      every form" but \x1b[?2J (DECSED) returns early at the private-introducer branch;
      Screen.Pending's doc says "Exported for the tests that pin the bound" and now has a
      production consumer; ops.go:65 says the path default is applied "at the CLI" when it is
      applied in couchcore's Invoke; atlas/couch.md:32 reads "hands the child its own stdio and
      block".
  - id: new
    severity: Minor
    family: test-harness-races
    title: |
      the live conformance scenario and Run's exit select are both racy by construction
    detail: |
      conformance_live_test.go:245-249 reads doneBeforeExit AFTER the write that makes the child
      exit and then Fatalfs if it is true. console.go's select between c.chunks and exited is a
      coin flip on exit, so the child's final output is likely dropped before release() clears the
      row. Also test-side: vtscreen_test.go redefines min, shadowing the builtin, and
      waitFor/waitLong/waitUntilTrue are three near-identical polling helpers across two packages.
```
