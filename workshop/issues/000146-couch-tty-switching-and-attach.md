---
id: 000146
status: working
deps: []
github_issue:
created: 2026-08-21
updated: 2026-08-22
estimate_hours: 10.32
started: 2026-08-22T12:14:19-07:00
---

# couch: tty switching and attach

Project: `workshop/projects/couch.md` — architecture and non-goals live there;
this issue is task 2.

## Problem

With a registry of named actors (`#145`), the operator still has no way to move
between them except terminal tabs, which know nothing about what a session is.
The switching experience is what determines whether couch gets used at all: if
getting back to a known place is ever slow or flaky, the operator reverts to tabs
and everything above it is dead weight.

## Spec

**A switcher, not a multiplexer.** One operator tty attached to one child at a
time, a key-sequence interceptor, and a per-child buffer replayed on attach so
the screen is not blank on landing. Explicitly NOT built: splits, layouts,
floating panes, simultaneous rendering, a plugin system. The failure mode to
avoid is reimplementing tmux badly — the complexity there lives in compositing
panes nobody is looking at.

**One keystroke home to the root actor, from anywhere, always.** The root actor
is whatever session couch launched in — usually brain, by convention rather than
mechanism; couch can start anywhere and nothing here knows about brain
specifically. This is the single most important property in the whole project:
if it is reliable the operator roams freely because getting back is free.

**`ctrl-space` moves up one level** — child → root actor, root actor → couch's
control panel. Bare key, acts immediately, no prefix keymap and no timing
window. Double-ESC was considered and rejected: ESC is already interrupt/cancel
in Claude Code and mode-switch in nvim, and a double-tap must either delay every
legitimate ESC or forward one it cannot retract. Richer navigation lives inside
couch's TUI with typeahead rather than in a chord table — one key to memorize,
then read a screen. Direct jumps (to actor N, to the latest notifier) are
deliberately deferred until the operator catches themselves wanting one.

**Switching is deterministic and LLM-free in the critical path.** Resolution of a
fuzzy reference sits *above* the switch (`#148`); the switch itself is a direct
call. A model turn inside this path reintroduces exactly the latency that sends
the operator back to tabs, so a direct route that skips resolution entirely —
hotkey home, a numbered list — must always exist.

**Detach and reattach without killing children.** A detached actor keeps running;
its child harness stays warm. Reuse what already exists rather than writing
terminal handling from scratch: `wrapcmd`'s terminal model over
`charmbracelet/x/vt` + `creack/pty`, and `scrollbackcmd`.

**couch does not composite — it reserves a row.** The child is given a terminal
one row shorter and couch owns the last row. The child never knows, so this is a
resize rather than compositing, and it works identically in the root actor and
while attached to any child. That row carries rolling notifications, so there is
exactly one place to look. Children that redraw on resize (nvim, zellij) handle
it natively.

Notification *detail* is not drawn there and not injected into the transcript as
system messages — transcript injection would burn the LLM's context every turn.
The row says something happened; `ctrl-space` and the advisor supply the rest.

**Agent children only.** couch does not host plain shells, log tails or test
runs; the operator leaves the window for those. The project's "single terminal
window" criterion means one window for *agent* work, and this is what keeps the
switcher from drifting into general child hosting.

Attachment is an **output routing decision**, not the actor's identity — messages
addressed to the operator route to the console when one is attached, and are
simply not rendered when none is.

## Done when

- couch supervises N sessions and switches the operator tty between them.
- `ctrl-space` reaches the root actor from inside every child, including one that
  is mid-output, and is measurably instant (no model turn, no network).
- A reserved status row is visible in the root actor and in every attached child,
  and the child renders correctly at the reduced height.
- An attached child that exits lands the operator in couch's TUI with which actor
  exited and why — never on a dead pane.
- Landing on a session shows recent context rather than a blank screen.
- Detach and reattach leave children running and warm.
- A numbered/direct switch path exists that requires no natural-language
  resolution.

## Plan

Design of record: `workshop/plans/000146-couch-tty-switching-and-attach-plan.md`.
Four review boundaries; the smoke steps stay where they were sequenced (risk
first) but are folded into the milestone whose risk they answer.

- [x] M1 — **shared pty-child core.** Extract `ptychild` (ring, replay
      query-strip, output scanner, pty child) out of `termcmd`'s multiplexer and
      migrate `pair term` onto it. Ships no couch behaviour; the migration is
      what validates the extraction (ARCH-DRY).
- [ ] M2 — **console over one child, with the reserved row.** `PtyRunner` behind
      the existing `Runner` seam (+ fake + live conformance), `couch start`
      becomes the console, `ctrl-space` interceptor, one-row-shorter child pty
      with a pinned scrolling region, and `Spawn` forced onto `pair resume
      <tag>` so a console restart reattaches instead of landing on a picker.
      **Smoke step 1** (one real `pair` + claude child, resize, nvim in and out,
      reattach across a `kill -9`) lands here.
- [ ] M3 — **many children and the panel.** Up-one-level focus, per-child ring
      replay (or a resize nudge for alt-screen children), typeahead + numbered
      direct switch, panel actions dispatching through `couchcore.Operations()`.
      **Smoke step 2** (two real children, switching, `ctrl-space` from a
      mid-output child) lands here.
- [ ] M4 — **exits, detach, and what the row says.** Child exit lands in the
      panel with actor + code, detach/reattach stays warm, notices over
      `couchcore.Enqueue`, terminal restored on every exit path including
      signals, atlas reconciled.

## Estimate

Derived after the plan cleared plan-quality (round 2, CLEAN), against the four
milestones in `workshop/plans/000146-couch-tty-switching-and-attach-plan.md`.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.15
item: pensive                 design=0.8  impl=0.08
item: greenfield-go-module    design=0.2  impl=0.32
item: greenfield-go-module    design=0.2  impl=0.2
item: cross-cutting-refactor  design=0.3  impl=0.2
item: real-api-discovery      design=0.0  impl=0.2
item: milestone-review        design=0.0  impl=0.2
item: greenfield-go-module    design=0.5  impl=0.32
item: smaller-go-module       design=0.1  impl=0.16
item: smaller-go-module       design=0.1  impl=0.16
item: greenfield-go-module    design=0.1  impl=0.2
item: smaller-go-module       design=0.1  impl=0.08
item: smaller-go-module       design=0.1  impl=0.16
item: real-api-discovery      design=0.0  impl=0.24
item: real-api-discovery      design=0.0  impl=0.24
item: milestone-review        design=0.0  impl=0.2
item: tui-screen              design=0.3  impl=0.28
item: smaller-go-module       design=0.1  impl=0.08
item: smaller-go-module       design=0.1  impl=0.2
item: smaller-go-module       design=0.0  impl=0.08
item: real-api-discovery      design=0.0  impl=0.24
item: milestone-review        design=0.0  impl=0.2
item: smaller-go-module       design=0.1  impl=0.12
item: smaller-go-module       design=0.1  impl=0.16
item: atlas-docs              design=0.1  impl=0.06
item: real-api-discovery      design=0.0  impl=0.16
item: milestone-review        design=0.0  impl=0.2
item: cross-cutting-refactor  design=0.0  impl=0.2
item: ux-rename-iteration     design=0.4  impl=0.1
item: ux-rename-iteration     design=0.4  impl=0.1
item: scope-pivot             design=0.3  impl=0.12
total: 10.32
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**What each item is**, in plan order — the mapping is the derivation, so it is
written down rather than left implied:

| Item | Covers | Why that primitive |
|---|---|---|
| `pensive` | this planning session: code survey, plan doc, two plan-gate rounds | design 0.8 takes the upper half of the 0.3–1 band — the survey spanned `couchcore`, `termcmd`, `wrapcmd` and `launcher`, and round 1 returned four Important findings. **Not** spec-discounted: no plan pre-resolved this work, it *is* the plan. |
| `greenfield-go-module` ×1 | M1 `ptychild` — `Ring`, `StripQueries`, `Screen`, `Child` | new package, single concern (a child on a pty). Design ×0.2-discounted: the plan fixes the contract and the test surface. |
| `greenfield-go-module` ×1 | M1 `hostty` — `Host`, `OSHost`, `FakeHost`, control constants | same shape, one concern (the operator's terminal), same discount. |
| `cross-cutting-refactor` | M1 migration of `pair term` onto **both** packages | multi-file, behaviour-preserving, with an existing suite as the net. Design 0.3 is not discounted to zero: where the seam falls in `runShell` is a live decision. |
| `greenfield-go-module` ×1 | M2 `couchtty` — `Console` + `Interceptor` | design 0.5 rather than the ×0.2 floor: DECSTBM's behaviour under real children is the one thing the plan cannot pre-resolve, and Decision 4 carries a named fallback that would cost redesign. |
| `smaller-go-module` ×2 | M2 `Reserve`/`RenderStatusRow` (Task 2.4); `PtyRunner`/`TerminalHandle` (Task 2.1) | well-specced extensions of shapes that exist — the `Runner` seam, `termcmd`'s escapes. |
| `greenfield-go-module` ×1 | M2 Task 2.2 — `FakeRunner`'s scripted in-memory terminal **plus** the live conformance pin against a real pty | split out of Task 2.1 on the second pass: a stateful behavioural fake with a real-vs-fake conformance check is not a mirror of an existing shape, it is the ARCH-MOCK work, across three files. |
| `smaller-go-module` ×1 | M2 Task 2.6a — `Spawn` forced onto `pair resume <tag>` | argv plus a derivation that reuses `launcher.DefaultTag`; small because the lever already exists. |
| `tui-screen` | M3 the panel — rows, typeahead, numbered pick | literally the primitive's description: a screen plus a state machine plus tests. |
| `smaller-go-module` ×2 | M3 `Focus`; N-children routing + replay in `Console` | pure model, then wiring onto seams M2 built. |
| `smaller-go-module` ×2 | M4 `Notice`/`Feed` + row content; exits + restore-on-signal | `Feed` delegates to `couchcore.Enqueue`, so it is an extension rather than new logic. |
| `atlas-docs` | M4 `atlas/couch.md` reconciliation | the atlas's "there is no pty yet" paragraphs are falsified by this issue. |
| `smaller-go-module` ×1 | M2 Task 2.6 — `NewCouchWith`, the `no-console` `FlagOnly` arg, `path` defaulting to `.`, displacing `couchcmd/run.go:169-178` | two files nothing else in this table claims. |
| `smaller-go-module` ×1 | M3 Task 3.4 — the panel-actions-are-a-subset-of-`Operations()` audit | design 0.0: the rule is already decided, this is the assertion. |
| `real-api-discovery` ×5 | one per operator smoke, plus the reattach/park experiments | **the closest primitive to what a smoke actually is** — a budget for discovering how an external system really behaves, the external systems here being Ghostty, zellij and nvim rather than an HTTP API. Task 1.5's `pair term` smoke (M1 migrates the daily driver; a repaint regression surfaces nowhere else); Task 2.7's rendering smoke incl. DECSTBM survival across alt-screen transitions; Task 2.7's **`kill -9` reattach + park-vs-kill determination** (a separate discovery — zellij's session lifecycle, not terminal rendering, and it ends in a correction to `workshop/projects/couch.md`); Task 3.5's real-configuration smoke, where Decision 5's replay-vs-nudge fallback is decided; Task 4.6's full-session smoke. |
| `ux-rename-iteration` ×2 | two iteration rounds on the status row, the panel and the navigation feel | v2.1's known-limitations section says TUI features take 3–5 rounds, not 1. Two is budgeted rather than five because the Spec pre-settled the navigation rule (one key, up one level) — the rounds left are how the row and panel *read*. |
| `scope-pivot` ×1 | Decision 4's disclosed DECSTBM fallback | expected-value budget for a **named, already-disclosed** risk, not a generic contingency: if the reserved row does not survive real children, the plan's own instruction is to take the fallback, which is a scope event. |
| `milestone-review` ×4 | the M1/M2/M3 boundaries plus the issue close | one per `sdlc milestone-close` / `sdlc close` — exactly the four boundaries the Plan commits to. At the band ceiling (0.2), because each one runs more than a review: whole-tree `go test`, `-race`, and at M2/M4 `make test-live` and the shell suites. |
| `cross-cutting-refactor` ×1 | fixing what the four boundary reviews hand back | a review gate returns findings — this issue's own plan-quality round 1 returned four Important ones — and ARCH-PURPOSE requires fixing the *class*, which is by definition a sweep across files. Budgeting the review while budgeting no rework is the gap the second estimate pass closed. |

**Read this as ship wall-clock, not calendar.** v3.1 writes `impl=` at 40% of the
v2 table because post-#118 actuals came in near half of v2's implementation
hours; the design column is unscaled.

**The number moved by decomposition, not by picking one.** Round 1 of this block
totalled 6.75 and the estimate-quality gate was right that it was thin: two of
three operator smokes were budgeted at zero, Task 2.6 had no item, and a TUI
issue carried no iteration rounds. Adding the items the work actually contains
took it to 9.33. The total was never the input — had the missing items summed to
less, the number would have gone down.

**Calibration signal, recorded now rather than argued at close.**
`calibration-ledger.tsv:376` has `pair#145` — the immediate predecessor, same
project, same operator, closed the same day — at **8.51h actual** with no
estimate recorded. `:357` has `pair#139` at **5.83 estimated → 22.37 actual
(ratio 0.26)** under this same v3.1 model, and `baseline-v3.1.md`'s open
question 3 already flags the under-estimation direction. #146 is materially
larger than #145 along every axis, so a total below 8.51 was not credible; 9.33
is barely above it, and if this repo's terminal work keeps landing near #139's
ratio the honest expectation is a miss on the high side. That is a v3.1
calibration input, not a reason to inflate the block — the ledger learns from
the gap, and hand-tuning the estimate to be right destroys exactly that signal.

**Step 2.5 (library availability) answered, for the one item where it bites.**
v2.1 requires the check on every `greenfield-go-module`. Three of the four have
design already ×0.2-discounted, so it is near-moot there. `couchtty`'s 0.5 is
deliberately undiscounted, and the check's answer is the plan's Tech Stack line:
**no TUI framework** — bubbletea/lipgloss would not short-circuit this, because
the console's job is to *pass bytes through* and reserve one row, not to render
a frame tree. pair writes raw escapes directly and couch must too. Design stands
undiscounted.

**`familiarity: 1.0` is kept, with the caveat named.** `termcmd` already does
pty, raw mode, `SIGWINCH` and replay, so the tree is familiar for M1 and most of
M3/M4. The scrolling-region reservation and the paste-aware interceptor are not,
and v3.1 applies familiarity to *impl* — which is where a DECSTBM surprise would
land. The block compensates on the design side (`couchtty design=0.5`,
undiscounted) plus the `scope-pivot` item, rather than bending a global
multiplier that would also lift the parts that genuinely are familiar.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project.

**Layering fork — SETTLED 2026-08-21, host `pair` whole.** The operator ran
`./bin/couch start ../pair` against `#145`'s spawn path and pair came up
correctly, so couch does **not** absorb zellij's role: the stack stays
couch → pair → zellij → claude+nvim, and a zellij inside a couch-owned pty is
just a child that redraws on SIGWINCH.

This issue's scope is therefore the narrow one: route one tty to one child at a
time, with no responsibility for what the child runs internally. Estimation is
unblocked.

### 2026-08-22
- 2026-08-22: closed M1 — Round 4. BR-1 fixed at its third and correct shape: the bound was never the rule -- past maxPending the scanner now stops BUFFERING but keeps FRAMING (streaming skip to the real terminator), so whole and chunked input agree at every length; deletion check reproduces the reviewers own whole=true/split=false measurement. BR-18 (which round 2 introduced) fixed with the enumeration it demanded: 4 of 7 post-terminal-state stimuli diverged across Host and Child, 1 fatally (FakeHost SetSize panicked on send-to-closed-channel); all now agree and BOTH conformance tests drive past the terminal transition rather than stopping at it. BR-6 consolidated into procutil.WaitCode after two rounds. Verified: go test ./cmd/... green; make test-race whole-tree green; make test-smoke 8/8 incl. nvim alt-screen round trip; make build; make test-term-pane-shortcuts green. All deletion checks confirmed mutate+compile+traverse.; review verdict: FIX-THEN-SHIP

Claimed and planned. Design of record:
`workshop/plans/000146-couch-tty-switching-and-attach-plan.md`; the eight loose
steps above are regrouped into four review boundaries there, unchanged in
content.

Three decisions the plan makes that this Spec did not settle, recorded here
because they narrow scope:

- **`couch start` becomes the console; no new verb.** The CLI's dispatch table
  is asserted identical to `couchcore.Operations()`, so a console-only verb
  would need an exception to the invariant that keeps the operator's surface and
  the advisor's from drifting. `--no-console` is the loud escape hatch back to
  #145's inherit-stdio behaviour.
- **The pty-child mechanics are extracted from `termcmd`, not written twice.**
  `pair term` already is a switcher (pty tabs, a 128KB replay ring,
  redraw-from-snapshot, resize propagation). `pair term` migrates onto the
  shared package in M1 -- its existing suite is the only regression net that can
  prove the extraction faithful.
- **Detach is console-scoped, because durability is zellij's.** couch's child
  is `pair` -> a zellij *client*; the work lives in the zellij *server* session,
  which survives detached when the client dies. So the fleet already outlives a
  terminal window one layer below couch and no daemon is on the path -- `#147`'s
  transport is not a prerequisite for the Done-when's "running and warm".

### 2026-08-22 -- reattach is zellij's, and Spawn must stop hitting the picker

The operator's read of the layering corrected the plan's first answer on detach:
couch hosts `pair`, which runs zellij, so a session is *already* reattachable
beyond a console's lifespan. The durability boundary is the zellij server, not
couch's process tree, and reasoning about a couch daemon was reasoning about the
wrong layer.

What that leaves `#146` owing is determinism on the way back IN, and there is a
real gap there today: `Spawn` runs `pair --layout2` with no tag, and
`launcher.DecideLaunch` with no tag and a detached session present returns
`ActionPick` -- an fzf picker inside couch's pty. A console restart currently
lands the operator on a picker rather than on their session.

Fix folded into M2: spawn `pair resume <tag>` with `tag =
launcher.DefaultTag(<worktree root>)`, which takes the `ForcedTag` branch
(attach if live or detached, create otherwise) and skips the name prompt.
`--layout2` is dropped -- `resume` refuses a third argv element outright, and an
omitted layout flag reuses the tag's recorded layout while forcing one on a live
tag makes pair ask before recreating the workbench.

This is a deliberate slice of `#149`, not a collision: `#149` decides the tag IS
the space (durable, opaque, several per tree, names as an attribute layer) and
supersedes this derivation. `#146` needs only that re-entry is deterministic.

Also queued for the M2 smoke: `workshop/projects/couch.md` asserts "`couch stop`
is a kill, not a park." If `Stop` signals the zellij *client*, the session
detaches and the work survives -- a park. Whichever it is, the project record
gets corrected from an observation rather than left as an unverified invariant.

### 2026-08-22 -- M1 built: ptychild + hostty, with pair term migrated onto both

Two packages, extracted from `termcmd`'s multiplexer rather than written a
second time for couch:

- **`cmd/internal/ptychild`** -- the child half. `Ring` (bounded replay window),
  `StripQueries` (#127's deny-list, moved not copied), `Screen` (one scanner
  over child output), `Child` (pty + read pump), `NewFakeChild` (the stateful
  double).
- **`cmd/internal/hostty`** -- the host half. `Host` seam over the operator's
  terminal, `OSHost`, `FakeHost`, and the terminal-control constants. `\x1b[r`
  was about to exist in two packages; it exists in one.

**Three things the extraction fixed rather than merely moved:**

- ~~`Ring`'s trim now COPIES instead of re-slicing, fixing unbounded growth.~~
  **RETRACTED at the M1 boundary review (BR-4).** Re-slicing is also bounded --
  measured, it peaks *lower* than copying (cap 48 vs 64 over 2000 appends into a
  32-byte ring), because shrinking the remaining capacity forces the next append
  to reallocate. The copy is a clarity choice, not a bug fix. The deletion check
  I ran removed the trim ENTIRELY, which proves boundedness is pinned, not that
  copy-vs-re-slice is. Two of the three "bugs found" below survive; this one did
  not, and claiming it is exactly the aspiration-shaped artifact lie.
- `Screen` sees sequences SPLIT ACROSS READS. `updateMouseMode` scanned each
  chunk independently, so a mouse-mode sequence bisected by a pty read boundary
  was missed. Every one of its cases ported, plus the split-read case it could
  not pass.
- BEL is framed, not grepped. Every title change ends in BEL, so the status
  row's one activity signal would have fired on every nvim buffer switch.

**Tests edited, and why none is a behaviour change** (the plan's rule is that an
edited test IS one until justified):

- `TestUpdateMouseMode` -> ported to `ptychild.TestScreenMouseMode`, all cases.
- `TestRedrawSnapshotIsRaceFree` -> moved to `ptychild`, where the mutex now is.
  Asserting it in termcmd would test a call into a lock it does not own.
- Five `terminalTab` literals built `os.Pipe` scaffolding purely to populate a
  `pty` field that no longer exists; two that needed ring content now use
  `ptychild.NewFakeChild`.

**Verified:** whole tree `go test ./cmd/...` and `-race` green; `make build`;
`make test-term-pane-shortcuts` green; fuzz 595k execs on `FuzzScreenFeed` and
3.5M on `FuzzStripQueries`, no panics.

**Deletion checks run** (each mutation confirmed the named test goes red): ring
trim *(removal of the whole trim -- NOT copy-vs-re-slice, see BR-4)*, ring
snapshot aliasing, `StripQueries` neutered (termcmd's
`TestRedrawTabEmitsNoQueries` goes red -- so termcmd still pins the behaviour
through the new call), the `?1049` alt-screen case, the private-introducer
discrimination on `r`, BEL counted inside OSC, `pty.Setsize` removed, and
ring-updated-after-sink.

**Environment note:** the command sandbox blocks `pty.Start`, so every
pty-backed test in `ptychild`, `termcmd` and (from M2) `couchtty` must run
unsandboxed. A sandboxed green on those packages is not evidence.

**Automated half of Task 1.5's smoke: `probes/termsmoke`** drives the real
`./bin/pair` under a pty. All 8 steps pass against `22d0226`:

```
PASS  tab 1 runs a command                         (found "MARKER-ONE")
PASS  Alt+t opens a second tab                     (found "MARKER-TWO")
PASS  Alt+Left repaints tab 1 from its ring        (found "MARKER-ONE")
PASS  Alt+Right repaints tab 2 from its ring       (found "MARKER-TWO")
PASS  resize reaches the child                     (found "40 100")
PASS  still usable afterwards                      (found "STILL-ALIVE")
PASS  nvim enters the alt screen                   (found "\x1b[?1049h")
PASS  switching away and back repaints nvim        (found "ALT-SCREEN-MARKER")
```

**What that does NOT cover, and why M1 still needs the operator:** the probe
asserts that the right BYTES are replayed. It cannot judge whether the screen
LOOKS right -- a repaint that leaves a stale row, doubles a prompt, or puts the
cursor in the wrong cell passes every one of those assertions. That is the
visual regression the daily driver would show and unit tests cannot.

### 2026-08-22 -- M1 differential: migration is behaviour-preserving

Operator smoke found no crash but no visible tabs either, which prompted a
differential rather than an argument. Built `pair` at `7187b22` (the last commit
before any M1 code) into a temp worktree and ran `probes/termsmoke` against both
binaries.

**Byte-identical results, startup output included** -- all 8 steps pass on each,
and both emit `"\x1b[1;1H\x1b[J\x1b[?1034hsh-3.2$ "` on startup. The migration
does not change what `pair term` does on this path.

**The missing tab bar is not a regression and not a bug.** `pair term` renders
its tab strip as the *zellij pane title* (`renamePane` -> `setPaneTitle` ->
`zellij action rename-pane "[terminal 1] work"`). Run standalone there is no
pane to rename, the action fails, and the error is deliberately swallowed. The
older in-terminal inverse-video strip was removed on purpose and
`run_test.go:690` pins that it stays gone. So standalone `pair term` has never
had a tab indicator, before or after this change.

The fault was in the smoke INSTRUCTION, which sent the operator to the one mode
where the affordance being tested is invisible. Lesson recorded in
`workshop/lessons.md`.

What the operator did confirm, which is the part M1 needed: two tabs exist, the
switch works, and each tab's visual state comes back on landing -- i.e. the
extracted ring and the replay path behave.

### 2026-08-22 -- M1 boundary review round 1: FIX-THEN-SHIP, 5 Important

Every finding reproduced before being fixed; all five held. Fixed at the class,
not the site:

- **BR-1 `chunking-invariance` -- a real bug, and the everyday trigger is OSC 52.**
  `maxPending` was 256, so a clipboard write (kilobytes, always crossing a
  4096-byte pty read) was abandoned mid-sequence and its terminating BEL was then
  counted by the plain-run scan: a false page on every copy. Reproduced at 300 /
  4096 / 9000 bytes -- whole `bell=false`, split `bell=true`. Two fixes, because
  the bound and the abandon-path were both wrong: the guard is now 64 KiB (a real
  prefix must be able to be as long as the protocol allows -- the bound is a
  memory guard, not a plausibility judgement), and an abandoned run is now
  DISCARDED rather than rescanned as text. **The class fix is the fuzzer:** it
  asserted chunk-invariance for `AltScreen` and `Mouse` but not for the two
  latches, which is exactly why 595k execs missed this. It now covers both;
  3.7M execs green.
- **BR-2 `signal-goroutine-outlives-close` -- I re-opened a bug this repo already
  recorded.** `defer host.Close()` sat next to `NewOSHost`, so LIFO ran it AFTER
  `mux.closeAll()`, leaving the resize watcher live while child ptys closed:
  `Setsize -> ptmx.Fd()` racing `ptmx.Close()`, the scribecmd use-after-close in
  `lessons.md`. Pre-migration ordering was correct; the migration lost it.
  Re-registered after `closeAll`, with the reason in a comment. Related leak
  fixed with it: `Resized()` was never closed, so `for range host.Resized()`
  never returned -- now closed by the watcher (its only sender), pinned for BOTH
  hosts by `TestCloseReleasesResizedConsumers`.
- **BR-3 `fake-diverges-from-production`.** `NewFakeChild`'s doc claimed
  `Wait` returns immediately and `Done` starts true; measured, `Wait` blocks and
  `Done` is false -- the opposite. A test written from the doc would HANG in
  M2/M3 rather than fail. The doc was wrong, not the code: a fresh fake is
  *running*, like a real child. Class fix is the missing ARCH-MOCK piece --
  `TestFakeChildConformsToRealChildLifecycle` drives fake and real through one
  shared scenario.
- **BR-4 `fix-not-pinned-by-failing-test` -- a claim of mine that was false.**
  See the retraction above. Re-slicing is also bounded (peaks *lower* than
  copying: cap 48 vs 64), so the "unbounded growth" the comment asserted does not
  exist. Comment corrected, Log retracted, copy kept as a stated clarity choice.
- **BR-5 `plan-table-drift`.** The plan said `termcmd.restoreTerminal` is
  deleted; it exists and writes `hostty.ResetRegion`. Corrected via a plan
  `## Revisions` entry rather than a silent edit.

Minors also fixed: `make test-race` targeted `./cmd/pair-wrap/`, a directory
deleted at the Go entrypoint switch -- so the `-race` half of Task 1.6 had no
runnable target at all (now `./cmd/...`); `OSHost`'s SIGWINCH coalescing is
tested against real signals rather than only the fake's; the DECRSTR negative
(`\x1b[?1049r` is not a margin change) has a case, and its deletion check now
fires; `probes/termsmoke` cleans up its child on the `os.Exit` path; `splitAny`
stopped allocating per scanned byte; `OSHost.state` (write-only) deleted.

**Deletion-check discipline failed twice while fixing BR-4**, both recorded in
`lessons.md`: a mutation whose `\x1b` became a real ESC byte matched nothing and
"passed" without running, and a mutation that applied removed a `return` the
test input never reached. A deletion check owes three things -- mutate,
compile, traverse -- and only the first is usually performed.

**Verified after the fix round:** `go test ./cmd/...` green; `make test-race`
(whole tree, newly runnable) green; `make build`; `make test-term-pane-shortcuts`
green; `probes/termsmoke` 8/8 against the rebuilt binary; `FuzzScreenFeed` 3.7M
execs with latch invariance asserted.

### 2026-08-22 -- M1 boundary review round 2: 14 disposed, 3 open, rules fixed

The gate's own summary was the useful part: *"3 repeat families. Not converging:
fix rules, not instances."* All three are now rule fixes.

- **BR-1 `chunking-invariance` -- correctly NOT accepted as addressed.** Raising
  the bound cured the everyday OSC 52 case but left chunk-invariance broken
  ABOVE the bound: whole input discarded the entire run, split input discarded
  the first `maxPending` bytes and then rescanned the remainder as text, where a
  BEL still counted. The bound was never the rule. The rule is **resync to the
  next ESC** after abandoning a sequence, which whole and split follow
  identically. Pinned by a 4096-byte-chunked test at `maxPending+5000`; reverting
  the resync turns it red.
- **BR-15 `pin-that-skips-itself`.** Three of the previous round's fixes were
  defended by tests that `t.Skipf` when `pty.Open` fails -- in the sandboxed
  shell this issue's own Log documents as its environment. `go test
  ./cmd/internal/hostty/` reported **ok with 3 of 9 silently skipped**. Two
  needed no pty at all and are now driven against a temp file (zero skips in the
  package). The one that genuinely needs a terminal FAILS loudly instead,
  matching how `ptychild` already handles the identical condition -- one
  milestone should not ship two handlings of one constraint.
  **And BR-2 had no test anywhere**, because defer ordering inside a
  tty-requiring function cannot be asserted. Fixed structurally: the ordering is
  now an explicit `teardown(host, closeChildren)` rather than emergent from LIFO
  registration, so it is both harder to invert and *testable* --
  `TestTeardownStopsTheWatcherBeforeClosingChildren` goes red on a swap.
- **BR-16 `probe-hygiene`.** `probes/` was new top-level surface with no make
  target and no atlas entry, while its 8/8 output was quoted as M1 evidence. Now
  `make test-smoke`, plus an `atlas/index.md` entry stating what earns a place
  there -- the plan schedules three more smokes (Tasks 2.7, 3.5, 4.6), so the
  convention wants to exist before the second one.
- **`plan-table-drift`, 2nd on this issue (3rd counting `pair#145` BR-41).** The
  plan's `Screen` row restated field names that had since changed. Renaming two
  words would have been the instance fix and the family would return. The row now
  describes what `Screen` ANSWERS and points at the source -- the rule
  `atlas/couch.md` already applies to couch's operation set.

**Deletion checks this round each confirmed mutate + compile + traverse**, per
the lesson the previous round earned: resync removal turns the new invariance
test red; swapping `teardown`'s two statements turns the ordering test red.

**Verified:** `go test ./cmd/...` green with zero skips in `hostty`; `make
test-race` (whole tree) green; `make test-smoke` 8/8.

### 2026-08-22 -- M1 boundary review round 3: BR-1's third shape, and a bug the round-2 fix introduced

- **BR-1, third attempt, and the previous two are worth keeping on the record
  because each was a smaller instance of the same mistake.** Raising the bound
  (round 1) fixed the everyday case. Discarding to the next ESC (round 2)
  restored invariance for UNTERMINATED runs -- and broke it for terminated ones:
  a 70 KiB OSC fed whole frames fine because its terminator is in the buffer,
  while the same bytes in 4096-byte chunks blew the bound, were abandoned, and
  **dropped a real BEL that followed**. The reviewer measured it as "the
  4096-byte production path now DROPS a real bell where whole-feed rings it",
  which is a *worse* failure than the one being fixed -- a missed page rather
  than a false one.
  The bound was never the rule. The rule is **stop BUFFERING, keep FRAMING**:
  past `maxPending` the scanner switches to a streaming skip that consumes to the
  sequence's real terminator without holding it. Memory stays O(1) and the stream
  stays in sync at every length. Deletion check reproduces the reviewer's exact
  measurement (`whole=true split=false`).
  One expectation was corrected rather than satisfied: the old bounded-pending
  test demanded that a sequence AFTER an unterminated CSI still be recognised.
  Whole-feed does not do that either -- `\x1b[` is a final byte to `ansi`'s
  introducer-independent scan -- so the test now asserts **equivalence**, which
  is the invariant, instead of a recovery the framing never promised.
- **BR-18 `fake-diverges-from-production`, and I introduced it in round 2.**
  `FakeHost.Close` closed `resized` under the lock while `SetSize` sent outside
  it and never consulted `closed`, so a post-Close `SetSize` **panicked** --
  in the double M2's console tests are built on, which crashes a run instead of
  failing it. The enumeration the finding demanded: post-terminal-state, `Host`
  is {resize, Write, Size, MakeRaw} and `Child` is {Write, Resize, Signal}; 4 of
  those 7 diverged, 1 fatally. All now agree, and the class fix is that BOTH
  conformance tests drive past the terminal transition rather than stopping at
  it -- which is exactly where a fake diverges most easily and where round 2's
  otherwise well-shaped conformance test stopped.
- **BR-6 `needless-indirection`, carried two rounds, now consolidated.**
  `waitCode` was byte-identical in `couchcore` and `ptychild`, plus a one-line
  `errors.As` wrapper in each: three packages holding one decision about what a
  child's exit code means. Now `procutil.WaitCode`, with both copies and both
  wrappers deleted.

**Verified:** `go test ./cmd/...` green; `make test-race` (whole tree) green;
`make test-smoke` 8/8. Every deletion check this round confirmed mutate +
compile + traverse, and two reproduced the reviewer's own measurements before
being fixed.

### 2026-08-22 -- M1 boundary review round 4: SHIP, with three fixes bundled in

The gate passed the boundary (round cap 3 reached, 6 disposed) and the
mechanical close ran. Per the FIX-THEN-SHIP protocol these three landed BEFORE
the close commit rather than after, so the reviewed anchor is HEAD.

- **BR-1's last residual, and it was in my own comment.** `skipTerminator` says
  it holds a trailing ESC "so ST is not split in half" -- and then dropped it,
  because the caller took `buf[n:]` and returned without saving. A chunk
  boundary falling inside a two-byte ST therefore swallowed the NEXT real bell.
  The reviewer measured it at **1 of 70,550 cut positions**: precisely the
  residual a fuzzer finds and a reader does not, in code whose comment claimed
  the opposite. Now the unconsumed remainder is held in `pending`; the deletion
  check reproduces `whole=true split=false`.
- **`framing-omits-sequence-class`.** `frame`'s `default: return 2, true` covered
  only ESC-c style two-byte escapes, so DCS / APC / PM / SOS payloads were
  scanned as plain text -- `\x1bP+q616263\x07\x1b\\` rang a false bell, and a
  tmux passthrough `\x1bPtmux;\x1b[?1049h\x1b\\` set alt-screen from INSIDE a
  sequence. 4 of the 5 string-terminated classes leaked; OSC was the one covered.
  Reachability is low today, but `TakeBell`'s doc and `atlas/architecture.md`
  both state "BEL is only counted outside a sequence" as a property, and **an
  invariant has to hold over every class it is claimed over** -- otherwise the
  doc is the lie, not the code.
- **`needless-indirection`, 2nd in family.** `Child.Replay` documented itself as
  what a repaint should write and had **zero production callers**, while
  `redrawTab` hand-composed the same expression. Two places holding one decision
  about what a repaint may contain -- and M3 Task 3.3 spells it out a third time
  for couch's attach path. `redrawTab` now takes an already-stripped replay;
  `replaySnapshotLocked` (renamed from `bufferSnapshotLocked`) is the one site
  that calls `Replay`. The redraw test moved onto the production path, since
  hand-feeding raw bytes would assert a composition production no longer does.

**A process failure worth recording over the fix itself:** three scripted edits
in this round silently did not apply -- one pattern did not match, one script
raised before its `write()` so nothing in it landed -- and each time the suite
stayed green and I reported the edit as done. The deletion check for the third
is what exposed it, by *not* failing. `lessons.md` now extends the
mutate/compile/traverse rule to ordinary edits: an edit is applied because you
checked, not because you wrote it.

**Verified at this commit:** `go test ./cmd/...` green; `make test-race` (whole
tree) green; `make build`; `make test-term-pane-shortcuts` green; `make
test-smoke` 8/8 incl. the nvim alt-screen round trip. All four deletion checks
this round confirmed mutate + compile + traverse.
