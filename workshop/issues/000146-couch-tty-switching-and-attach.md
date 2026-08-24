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
- [x] M2 — **console over one child, with the reserved row.** `PtyRunner` behind
      the existing `Runner` seam (+ fake + live conformance), `couch start`
      becomes the console, `ctrl-space` interceptor, one-row-shorter child pty
      with a pinned scrolling region, and `Spawn` forced onto `pair resume
      <tag> --layout2` so a console restart reattaches instead of landing on a
      picker. **Smoke step 1** (one real `pair` + claude child, resize, nvim in
      and out) lands here; the `kill -9` reattach moved to M3 — see the
      2026-08-23 carry note.
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

### 2026-08-22 -- M2 built: the console, and the reserved row VERIFIED not asserted

`couch start` is the console now. `PtyRunner` sits behind the existing `Runner`
seam as a CAPABILITY on the handle (`TerminalHandle`), so `--no-console` keeps
`ExecRunner` a live production path rather than dead code the conformance check
pins for nothing. `Terminal()` returns the concrete `*ptychild.Child` rather
than an interface, because `FakeRunner`'s double IS a `ptychild.NewFakeChild` --
the same type, so a console test takes the branch production takes.

**The reserved row is verified against a real terminal emulator.** This was the
milestone's headline risk and the one thing a pty cannot answer: a pty passes
escapes through uninterpreted, so no amount of byte-level assertion says whether
DECSTBM actually holds. `charmbracelet/x/vt` interprets them and pair already
depends on it for `wrapcmd`'s terminal model, so `vtscreen_test.go` drives the
real `Console` against a real emulator and reads the resulting SCREEN:

- 40 lines of scrolling child output leave the reserved row intact.
- A child that resets margins (`\x1b[r`, which nvim emits on exit) drops the
  reservation and the console puts it back -- verified by scrolling 40 more
  lines afterwards and finding the row still there.
- After release the bottom row is usable again, so the operator's shell does not
  inherit a fenced-off screen.

Deleting the `Reserve` call turns the first of those red with the exact symptom
("40 lines of child output overwrote the reserved row"). **Decision 4's fallback
is not needed at the design level** -- what remains for the operator smoke is
whether a real terminal (Ghostty) and a real `pair`/zellij/claude stack agree
with the emulator.

Two expectations were corrected rather than satisfied while writing those: a
trailing newline leaves the cursor on a fresh blank line, so "the last line of
output" is on `rows-2`, not `rows-1`. Asserting the exact row would have been
asserting the test's arithmetic; the assertions read the whole child area
instead.

**Found while wiring the CLI:** `FakeRunner` ended its handle while its terminal
double kept running, so the first console CLI test HUNG rather than failed --
the same fake-diverges-from-production class as BR-18, at a different seam. One
child, one notion of exited, now pinned by two tests.

`Spawn` forces `pair resume <tag>` (Decision 11), with the tag from the tree so
re-entry is deterministic. The pre-existing argv assertion was updated: that is
a deliberate behaviour change, not a test bending to code.

**A production data race, found by the whole-tree `-race` run** (the target that
had no runnable directory until M1's boundary review): `ptyHandle.ID()` returned
`strconv.Itoa(h.pid)`, and the pump can call the sink -- which tags each chunk
with `ID()` -- before `ptychild.Start` has even returned and `h.pid` is
assigned. Not merely a detector complaint: in production the first chunks of
every session would have been tagged with a zero id. The id is now minted before
the child exists and never derived from the pid. `ExecRunner` can use the pid
because nothing reads ITS id from another goroutine; this one is read from the
pump.

Worth noting how it surfaced: it was already committed and every non-race run
was green. `make test-race` earning a runnable target in M1 is what caught it.

### 2026-08-22 -- M2 smoke round 1: the row vanished, and the emulator tests had missed why

Operator ran `couch start ../pair` and reported `[pair]` appearing on the bottom
row and disappearing about a second later, with pair's saved-config picker
drawing over it. Two separate defects, both real.

**1. The reserved row is erased by any full-screen child's startup clear.**
DECSTBM restricts **scrolling**, not **erasing**. Every full-screen app clears
the display when it starts, and that takes the reserved row with it while the
region stays perfectly intact. The vt tests missed it because a scrolling child
never clears -- which is exactly the gap between "verified against an emulator"
and "verified against the real stack", and worth remembering as the LIMIT of
what M2's emulator suite proved.

Fixed by treating an erase as row-dirty. The concept was renamed with it:
`TakeRegionLost` -> `TakeRowDirty`, because "region lost" was the mistake that
hid this -- the console does not care WHY the row is gone, only that it is.
Erase-in-line is deliberately excluded (a child cannot address a row it does not
have, and repainting on every cleared line would be constant churn). Deletion
check: removing `J` from the dirty set turns four subtests red.

**The tests written to reproduce it passed first, on all five cases, in
0.01s** -- they polled "is the row there?" immediately after feeding the clear,
and saw the row still standing from before the async chunk landed. A green suite
reporting a live bug as fixed. The second shape (wait to observe the row vanish)
was flaky the other way, since a fast repair may never expose the damaged state
-- RIS, already handled, started failing for the wrong reason. The tests now
establish ordering with a marker behind the stimulus. Lesson recorded.

**2. `pair resume <tag>` still prompts -- but at a DIFFERENT picker than the one
Decision 11 closed.** Not `DecideLaunch`'s fzf session list; this is
`runConfigPicker` (`launcher/createflow.go:646`), the saved-config restore
prompt: "use saved params + session / use saved params / use new params".

Decision 11 was therefore incomplete, though narrower than it looks. `resume`
does avoid the name prompt and the session picker, and once a session is live
every later `couch start` ATTACHES and prompts nothing. This fires only on a
COLD start of a tag that has a saved config. It is skipped only when argv
already pins an explicit resume (`extractExplicitResume`), which needs the agent
session id -- something couch does not have and `#149` is the issue that would
give it one. No non-interactive override exists in pair today.

**Not fixed pending an operator call** -- see the open question below.

**Operator call 2026-08-22: leave the config picker.** It fires once per cold
start, and choosing fresh-vs-resume at that moment is defensible rather than
merely tolerable. Determinism there needs the agent session id, which is
`#149`'s to provide -- noted on that issue so it is not carried only here.
`#146` closes Decision 11 at what it actually delivers: no name prompt, no
session picker, and an attach with no prompt at all once a session is live.

### 2026-08-22 -- park-vs-kill settled by measurement, and Decision 7 confirmed

`probes/zellijpark` (committed) creates a throwaway zellij session, kills its
client, and looks:

```
== killing the CLIENT with SIGTERM (what `couch stop` sends)
   session present=true  => PARK: the session outlived the client
== killing the CLIENT with SIGKILL (what a crashed console leaves behind)
   session present=true  => PARK: the session outlived the client
```

**`workshop/projects/couch.md` was wrong** and is corrected there: `couch stop`
is a park. pair installs no SIGTERM handler and reaches `DeleteSession` only
from explicit quit/restart/layout paths, so signalling pair does not take its
session with it.

**Decision 7 is confirmed at its foundation.** The work outlives the console
under both a clean stop and a crash, so couch needs no daemon -- only
deterministic re-entry, which Decision 11 provides. That was the load-bearing
assumption behind not pulling `#147`'s transport into `#146`, and it is now
measured rather than reasoned.

**Scope of the measurement, stated honestly:** this exercises the zellij layer
-- a client killed directly. `couch stop` signals `pair`, whose zellij client is
its child; pair dying orphans that client, and closing couch's pty then hangs it
up. Both routes end at a client death, which is what was measured, but the full
path through pair is still an operator-smoke item rather than a measured one.

### 2026-08-22 -- M2 smoke round 2: ctrl-space reached nvim, and a claim of mine was wrong

**1. ctrl-space was not intercepted -- the terminal does not send NUL.**
Operator reported it landing in draft nvim. zellij explicitly enables the Kitty
keyboard protocol, so the terminal stops sending the legacy NUL for ctrl-space
and sends CSI-u: `\x1b[32;5u` (space is codepoint 32; ctrl is modifier bitmask
4, encoded as 4+1). An interceptor that knew only `0x00` forwarded it straight
to the child.

The evidence was already in the tree: pair's own chord table carries BOTH
encodings for every chord (`workbenchshortcut/shortcut.go:294-312`,
`{"\x1bj", ChordAltJ}, {"\x1b[106;3u", ChordAltJ}`). Decision 10 leaned on
ctrl-space being "a bare key, not a chord, so there is no framing state" -- true
of the legacy encoding and false of the one actually in use. The ctrl-space
audit checked what BINDS the key and never asked what ENCODES it.

Fixed with a sequence table carrying both forms. Exact strings, matching how
`workbenchshortcut` does it: a tolerant CSI-u parser would also have to decide
what `\x1b[32;5:3u` (key RELEASE) means, and guessing there is how a switcher
fires twice per keypress. The test that matters most is the negative one --
couch must NOT eat the workbench's own chords, which arrive in the same shape.
Deletion check: matching any CSI-u turns it red on `\x1b[106;3u` (Alt+j).

**2. `--layout2` is back, and my reason for dropping it was wrong.**
Operator decision: pin layout2 for now, since couch owns terminal switching and
layout3's third pane is the layer couch replaces.

Decision 11 had claimed `resume` refuses any third argv element, making the flag
impossible. Only POSITIONALS are refused. `ParseArgs` runs
`extractLayoutRequest` FIRST (`args.go:51`), which strips layout flags before
the guard sees them, and `launchArgsAcceptLayout` admits them for resume because
its `Command` is `""`. Measured: `resume mytag --layout2` parses to
`{tag, layout2}`; `resume mytag stray` is the thing that errors.

I read the positional guard and stopped there. Both properties are now pinned in
`launcher/args_test.go`, where a change to either would otherwise break couch
silently.

### 2026-08-22 -- M2 smoke round 3: layout2 and ctrl-space confirmed by the operator

Operator confirms against the real stack (Ghostty -> couch -> pair -> zellij ->
claude+nvim):

- **`--layout2` works.** couch's children come up two-pane, as pinned.
- **`ctrl-space` works.** The Kitty CSI-u encoding is intercepted; it no longer
  reaches draft nvim.

Recorded as exactly what was said. Still owed for M2's close, and NOT inferred
from the above: whether the reserved row survives pair coming up, claude
streaming, and nvim in-and-out (the ED fix, which is what broke in round 1);
whether the workbench's own chords still reach the child; whether quitting
leaves a clean terminal; and the `kill -9` reattach.

### 2026-08-23 -- a real child under the console found a corruption bug
- 2026-08-23: closed M2 — Round 4. All 3 Criticals disposed at round 3. The four remaining Importants fixed at the class: BR-24 consoleRunnerFor pins the WIRING without a pty (forcing it to decline now goes red in-sandbox) and the path default is pinned by EFFECT -- which surfaced that the explicit default was dead weight since filepath.Abs(empty) returns cwd, so Spawn refuses an empty path and the default is load-bearing. BR-26 all five named sites actually changed (Decision 11s false resume claim, Task 2.6as inverted test, statusrow.go which does not exist, TerminalHandles location and interface-vs-concrete contract, MarginsDirty at two sites) with a Revisions entry that does not overclaim. BR-36 Task 2.7 recorded item by item, separating operator-confirmed from automated, naming what is carried to M3 and why, and explicitly NOT claiming the row-while-claude-streams case. BR-38 fixed as an enumeration: readme_test.go derives from couchcore.Operations() and every FlagOnly arg, and immediately caught two gaps I had not thought to write. Verified: go test ./cmd/... green; make test-race DATA RACE clean; make test-live green; make test-smoke green via the probes/*/ enumeration; make build. Operator smoke on the real stack passed 2026-08-23.; review verdict: FIX-THEN-SHIP

Rather than wait on the remaining smoke items, I put a REAL pty child under the
real `Console` with a real terminal emulator reading the screen
(`console_live_test.go`, gated on `PAIR_LIVE_COUCH=1`). Every fake-child test
was green; a fake only emits what the test hands it whole, which is precisely
the gap.

**The bug: the console spliced its status row into the middle of the child's
escape sequence.** Observed emission with nvim as the child:

```
\x1b7\x1b[12;1H\x1b[2K[brain]\x1b8;82;88m
```

That is a row paint written into the middle of nvim's
`\x1b[38;2;76;82;88m`. A pty read boundary falls wherever the kernel puts it,
and the console wrote its own bytes into the gap -- corrupting the child's
colours and losing the row. This is very likely part of what the operator saw in
round 1, alongside the ED fix.

**Fixed by asking the child whether it is safe to write.** `Screen` already
frames sequences to track alt-screen and erase state, so it knows when the
stream ends mid-sequence; that is now `Child.MidSequence()` and the console
defers its paint, paying the debt on the next chunk that lands on a boundary
(ARCH-DRY -- no second parser). Deletion check: removing the deferral turns
`TestConsoleNeverInjectsInsideAChildEscapeSequence` red.

**Which test does what, stated because it matters:** the LIVE test found the
class but does not pin it -- reverting the fix leaves it green, since whether a
read boundary lands inside a sequence is kernel timing. The deterministic guard
is the unit test, which CONSTRUCTS the boundary. A live test treated as a
regression guard would be a pin that cannot fail.

**Two harness bugs of my own, worth recording as limits on what these prove:**

- The vt harness deadlocked the moment a real child appeared: nobody drained the
  emulator's reply pipe, so it wedged on nvim's capability queries. `wrapcmd`
  already does that drain (`terminal_model.go:91-94`) -- the harness learned it
  by hanging.
- The harness does not faithfully render an ALT-SCREEN app; nvim's own content
  comes back truncated. So the nvim test asserts on the byte STREAM, and
  "does the rendered screen look right with a real full-screen app" remains an
  operator smoke item rather than something I claim.

Also fixed: the first version of the unit test could not reproduce the bug at
all, because both `Feed` calls completed before the console processed either --
the same async-ordering flaw as the round-1 tests, caught this time by the
deletion check rather than shipped.

### 2026-08-23 -- M2 smoke PASSED against the real stack

Operator, on Ghostty -> couch -> pair -> zellij -> claude+nvim:

- **The reserved row stays.** `[pair]` remains on the bottom row through pair
  coming up, and everything pair-related works normally inside couch's pty.
  This is the milestone's headline property and the thing that broke in smoke
  round 1; it is now confirmed where the emulator could not speak.
- **`ctrl-space` is intercepted by couch** and renders its line after `[pair]`,
  rather than reaching draft nvim.
- **`--layout2`** confirmed earlier the same session.

**Decision 4 is settled: no fallback needed.** The reserved row survives a real
full-screen stack, so couch does not composite and does not need the on-demand
overlay the plan held in reserve.

What the smoke did NOT separately exercise, and is therefore carried to M3
rather than claimed here: the `kill -9` reattach (both halves are covered --
`probes/zellijpark` measured the zellij session surviving SIGTERM and SIGKILL to
its client, and `TestSpawnProducesTheSameTagForTheSameTree` pins the tag
determinism -- but their composition is untested); and quitting couch leaving a
clean terminal, which has unit coverage on both the child-exit and teardown
paths plus a vt check that the bottom row is usable after release.

### 2026-08-23 -- ctrl-space audit, recorded properly (Task 2.3)

The plan required this in the `## Log` and it went into a commit message
instead, which is not where anyone looks (M2 BR-28). Recording it, and what it
missed.

**What binds ctrl-space, checked 2026-08-22:** nothing.
`zellij/config.kdl` binds no Space chord; `nvim/*.lua` has no `<C-Space>`,
`<C-@>` or `<Nul>` mapping; `keyhelp/catalog.go` lists no Space entry;
`wrapcmd`'s input handling has no `0x00` case. So no escape hatch rides on it
and couch may claim it.

**What the audit did NOT ask, and should have:** what ENCODES the key. zellij
enables the Kitty keyboard protocol, so a real session sends
`\x1b[32;5u` rather than `0x00` -- and couch's interceptor knew only the legacy
byte, so ctrl-space reached draft nvim. The operator caught it in smoke.
"Nothing binds it" and "we will receive it" are different questions, and the
audit answered only the first. Both encodings are handled now, with the negative
case pinned: couch must not swallow the workbench's own chords, which arrive in
the same CSI-u shape.

### 2026-08-23 -- M2 boundary review round 1: REWORK, 3 Critical

The verdict was right and the Criticals were real. **BR-21 falsified the fix
this milestone's headline commit claimed to make**, which is the one worth
reading twice.

- **BR-21 `MidSequence` asked the wrong stream.** The console asked the CHILD
  whether it was mid-sequence -- but ptychild's pump feeds its `Screen` before
  calling the sink, and the console drains a 256-deep channel later, so the
  answer described a chunk the child had since read, not the chunk being
  written. The reviewer reproduced the original nvim corruption byte-for-byte.
  A second mode too: `Pending()` reads 0 while an over-long sequence is being
  SKIPPED rather than held, so the check reported "safe" mid-sequence.
  Fixed by framing the stream the console WRITES (`Console.hostScan`), which is
  race-free by construction -- one writer. `Child.MidSequence` is deleted rather
  than left to invite the same mistake, and `Screen.MidSequence` now covers the
  skip case with `Pending`'s doc saying it answers a different question.
  **And the test was worse than the bug.** My version fed chunk 1, waited for
  the console to process it, then fed chunk 2 -- the reviewer's phrase was
  "avoids the window rather than covering it", and that is exactly right: it
  synchronised away the very skew production has. Both modes now have tests that
  reproduce the production ordering, each verified red on revert.
- **BR-22 a lone ESC was held indefinitely.** ESC prefixes both paste markers
  and the CSI-u hotkey, so "hold every real prefix" buffered a pressed ESC until
  the next keystroke and then delivered them glued -- which a terminal reads as
  Alt+<key>. ESC is interrupt in claude and mode-switch in nvim; it would have
  appeared dead and then done the wrong thing. The discriminator is that a
  keystroke arrives as its own read. Residual stated in the code: a sequence
  split immediately after its ESC is forwarded rather than held.
- **BR-23 `couch start` off a tty was silently broken.** It spawned and
  registered the child, gave it a ZERO-ROW pty, then exited 1 with no output --
  so a scripted or piped invocation left a registered actor nobody could use.
  Now a missing terminal falls back to the stdio path, loudly.
- **BR-24 the milestone's central wiring was unpinned:** disabling the console
  entirely left the whole suite green, because every CLI test drives a non-tty
  stdout and none took the console branch. `consoleRunner` is now driven
  directly against a REAL pty, and the disable-it mutation turns it red.
- **BR-27 `StatusActor.Bell` had no writer** -- the row could never say WHICH
  actor wanted attention, which is Decision 8's entire justification for
  spending a permanent terminal row. Wired, with an inactive-vs-active
  distinction. `FakeRunner.Sink`/`Emit` were genuinely dead and are deleted; the
  fake's child is a real `*ptychild.Child`, so `Feed` already did what `Emit`
  did.
- **BR-29 `Deliver` dropped output on a full buffer**, justified by a
  repaint-from-ring that does not exist until M3 -- so a drop was silent,
  permanent loss. It blocks now, yielding to stop so teardown cannot deadlock.
- **BR-25/26/28/31** documentation and hygiene: `probes/zellijpark` gets the
  make target and atlas entry the rule written one round earlier demanded; the
  plan's Decision 11 is corrected and the tables stop restating code shapes; the
  ctrl-space audit is in the Log where it belongs; README and
  `atlas/architecture.md` reconciled.

**BR-30 is disputed, with a measurement.** The claim was that two trees sharing
a basename resume one zellij session. `AssignSessionName` was driven with two
distinct roots ending in `pair`: `ComposeSessionName` does collide, but the
collision ladder separates them -- `📁pair` and `📁pair-2`, keyed on the
distinct scope keys. couch passes each child its own tree as cwd, so the scopes
differ. If there is a hole it is in an EMPTY session index, not in the derivation.

### 2026-08-23 -- M2 boundary review round 2: REWORK again, and the pattern in it

Six findings came back `not-addressed`. Every one was a genuine partial fix, and
the shape they share is worth more than the individual repairs: **I fixed the
instance the finding named and stopped at the edge of the class.**

- **BR-21.** I moved the tracking to the stream the console WRITES -- correct --
  and left two of three writers unguarded. `applyLayout` (on SIGWINCH) and the
  hotkey path still wrote from their own goroutines. Now `Run` is the ONLY
  goroutine that touches the host; everything else sends it events. That is the
  class: not "guard this write" but "there is one writer".
  A second shape surfaced while fixing it, and it is the subtler one: feeding
  the console's OWN escapes into the framing scanner let it frame our
  `\x1b[1;23r` together with the child's pending `\x1b[38;2;76` as a single
  complete sequence -- so it reported "safe" exactly when it was not. The
  scanner is fed child bytes only. Consulting a state your own writes mutate is
  not consulting anything.
- **BR-22.** My discriminator keyed on the CHUNK length, so a sole `\x1b` read
  was exempt but `abc\x1b` then `i` still glued into Alt+i, and `\x1b\x1b`
  held one. It keys on the PARTIAL length now; four cases pinned.
- **BR-23.** The fallback was in and pinned, but `Console.Run` still returned a
  bare 1 on a MakeRaw failure. It says why now.
- **BR-24, and this one is the worst.** My new pins needed a real pty and so
  `t.Skipf` in the sandbox this issue's own Log documents as its environment --
  meaning the disable-the-console mutation stayed green and the pin proved
  nothing. **That is the third time a gated-only pin has been written on this
  issue**, after M1's BR-15 and the lesson already in `workshop/lessons.md`
  titled "A gated-only pin is not a pin". The decision is a pure function
  (`WantsConsole`) now and is pinned unconditionally.
- **BR-25.** I added a second hardcoded `go run` line where the class fix asked
  for an enumeration. `make test-smoke` now iterates `probes/*/`, so a new probe
  is covered by existing.
- **BR-26.** The Revisions entry asserted a table sweep that had not happened.
  Claiming a sweep is worse than skipping one -- it tells the next reader the
  drift is dealt with. Actually swept.
- **BR-37** deserves naming separately: `atlas/couch.md` still told the reader
  the console asks `Child.MidSequence()`, a method that round had DELETED, on
  the stream the review had just proved wrong. Stale docs are ordinary; a doc
  that teaches the exact mistake a review caught is worse than none.

Lesson recorded in `workshop/lessons.md` covering the whole three-attempt arc:
one writer, frame at the point of writing, feed the scanner only the other
party's bytes -- and a test that synchronises producer and consumer cannot see a
skew bug.

### 2026-08-23 -- Task 2.7's smoke, item by item

Recorded by name, because "the smoke passed" is not a record of what was
exercised (BR-36).

**Confirmed by the operator on the real stack** (Ghostty -> couch -> pair ->
zellij -> claude+nvim), 2026-08-22/23:

- pair + zellij + claude come up inside couch's pty, and everything
  pair-related works normally there.
- The reserved row stays through pair coming up. This is the item that FAILED in
  smoke round 1 and drove the ED fix.
- `ctrl-space` is intercepted by couch and renders its line after `[pair]`,
  rather than reaching draft nvim. This item also failed in round 1 and drove
  the Kitty-encoding fix.
- `--layout2` is what the children come up with.

**Covered by automated verification rather than by the operator**, and named so
the difference is visible:

- The child renders at `rows-1` and reflows on resize -- unit-pinned
  (`TestConsoleSizesTheChildOneRowShort`,
  `TestConsolePropagatesAHostResizeToTheChild`) and exercised against a real pty
  child.
- The row survives a scrolling child, a margin reset, and every ED form --
  verified against a real terminal emulator reading the SCREEN, plus a real pty
  child for the scrolling case.
- Quitting restores the terminal -- unit-pinned on the child-exit AND teardown
  paths, plus a vt check that the bottom row is usable again after release.

**Carried to M3, deliberately, with the reason:**

- **The `kill -9` reattach.** Both halves are measured separately -- the zellij
  session surviving SIGTERM and SIGKILL to its client (`probes/zellijpark`), and
  the tag determinism (`TestSpawnProducesTheSameTagForTheSameTree`) -- but their
  COMPOSITION is untested. It needs a second couch process, which is M3's shape
  anyway.
- **`nvim` in-and-out under the real stack.** The margin-reset case is
  emulator-verified and the row survived the operator's session, but nvim
  specifically was not driven in-and-out by hand. M3's smoke has the operator in
  a full session again.

**Not observed and not claimed:** the row while claude STREAMS a long response.
The operator confirmed the row stays through startup and normal use; a long
streaming response was not called out, and the automated scrolling coverage is
the nearest evidence rather than the same thing.

### 2026-08-23 -- M2 boundary review round 3: FIX-THEN-SHIP, and the doc findings turn into enumerations

Verdict moved to FIX-THEN-SHIP; all three Criticals disposed. Four Importants
remained, and the two that had already come back twice are the interesting ones.

- **BR-24, third time.** I had pinned `WantsConsole` but not that
  `consoleRunner` USES it -- forcing it to decline still left the suite green.
  Split into `consoleRunnerFor(..., hasTerminal bool, ...)` so the WIRING is
  pinned without a pty. And `TestStartDefaultsItsPathToCwd` asserted on
  `ArgSpec.Required` rather than on the default's effect, so deleting the default
  changed nothing.
  Fixing that properly surfaced something real: **the explicit `.` default was
  dead weight**, because `filepath.Abs("")` already returns the cwd. Two
  mechanisms producing one result means neither is pinned, so `Spawn` now refuses
  an empty path outright and the default is load-bearing.
- **BR-26, third time.** The round-2 entry claimed a table sweep that touched
  only prose bullets; **0 of the 5 named sites had changed**. All five fixed
  individually this time -- Decision 11's false `resume` claim, Task 2.6a's test
  that asserted `--layout2`'s ABSENCE, `statusrow.go` (a file that does not
  exist), `TerminalHandle`'s declared location and interface-vs-concrete
  contract, and `MarginsDirty` at two sites. The rule the review extracted is
  worth keeping: **a plan statement asserting external behaviour must be measured
  before it is written**, and a boundary that reverses a Decision writes the
  Revisions entry in the SAME window.
- **BR-36.** Task 2.7's smoke is now recorded item by item, separating what the
  OPERATOR confirmed from what automated verification covers, plus what is
  carried to M3 with the reason -- and one item explicitly NOT claimed (the row
  while claude streams a long response was never called out as observed).
- **BR-38.** The atlas identifier sweep had no counterpart for README's typed
  surface, so the class recurred at the one documented site the sweep did not
  cover. `couch start` now claims ctrl-space from every child and its argument
  is optional, and the README said neither -- an operator whose ctrl-space stops
  reaching their editor had no documented explanation.
  Fixed as an ENUMERATION rather than two lines: `readme_test.go` derives from
  `couchcore.Operations()` and from every `FlagOnly` argument, so a new operation
  or a new bypass is documented by existing. It immediately caught two gaps I had
  not thought to write -- `couch describe` and the `--same-tree` bypass.

### 2026-08-23 -- M2 boundary review round 4: gate passed, three fixes bundled in

No open blocking findings after four rounds. Three findings were recorded past
the round cap rather than accepted, so they were fixed before the close commit
per the FIX-THEN-SHIP protocol -- a cap is the gate declining to keep spending
rounds, not a judgment that a finding is wrong.

- **An inactive pane's row damage was thrown away.** `onChunk` consumed
  `TakeRowDirty` for every pane but acted only for the active one, so a
  background child's erase vanished and attaching to it later would have landed
  on a screen with no status row -- a bug that would first appear in M3, where
  attaching is the whole feature. The latch is per-pane now, mirroring how the
  bell already worked.
- **`Console` was filed under "Pure entities"** by my own sweep, while the row
  beside it called it a thin IO shell. It is an integration point and is filed
  as one.
- **My README exemption named a home it had not checked.** The enumeration
  exempted `publish-description` as agent-facing with a comment pointing at
  `atlas/couch.md` -- which did not document it either. The atlas now describes
  it, and a second test enforces the exemption's other half: an exempted
  operation must be documented in the atlas, and the exemption list may not name
  an operation couch no longer declares. An exemption that names another home
  has to check that home.

**One recurring miss of mine worth recording separately:** the test for the
inactive-pane latch failed on its first run because it asserted immediately
after `Feed`, before the console's loop had processed the chunk. `Feed` is
synchronous, the console is not. That is the third time this session I have
written an assertion that races the consumer -- twice it produced a false PASS
(a live bug reported fixed), and here a false FAIL. It is already in
`workshop/lessons.md`; what it needs is applying, not recording again.

### 2026-08-23 -- M3 built: couch is a switcher

`Focus` + `PanelModel` + N children in the console + the panel dispatching
through `Operations()`. `ctrl-space` now goes somewhere: child -> root actor ->
panel, with liveness consulted so it never lands on a dead actor.

**Design points worth keeping:**

- **`Focus` carries an explicit kind.** Without it `FocusActor("")` compares
  EQUAL to `FocusPanel()`, so a bug producing an empty id would silently render
  the wrong screen and look deliberate. The zero value is still the panel, which
  is the right default for a console with nothing attached.
- **A screen TAKEOVER is a different write from an interleaved paint.** M2's
  mid-sequence gate is correct for a paint inserted into a continuing stream;
  a switch landing or the panel opening REPLACES that stream's screen, so
  deferring would strand the operator on the previous child. `takeOverScreen`
  resets the framing state for the same reason. The M2 splice test caught this
  distinction by failing on my first cut -- the guard I built then is what
  flagged the new code.
- **The panel owns the keyboard while it is up**, and a background child's
  output stops painting -- otherwise a streaming child paints over couch's own
  screen and keys aimed at the panel reach a child.
- **A digit is a direct switch**: no typeahead, no resolution, no model turn.
  The Spec requires a route that always exists, and this is it.
- **`Filter` keeps the MODEL's order, not the resolver's.** A lookup may return
  any order; numbered selection is only safe if rows do not move under the
  operator's fingers.

**Two tests were fixed after deletion checks failed to FIRE**, which is the
useful half of running them:

- `Filter` in the resolver's order left every ordering test green, because the
  fixtures happened to agree. The test that catches it reverses the resolver.
- The resolver wiring test called `wireResolver` directly, so it pinned the
  FUNCTION and not that anything calls it -- the same shape as M2's BR-24. The
  wiring moved onto the path that actually runs a console, and is now driven
  through it.

**The async-marker trap hit twice more** (five times across M2/M3): a wait
condition polling something the PRODUCER sets synchronously is true before the
consumer has run. `lessons.md` now leads with the question that catches it --
*could this be true before the code under test ran?* -- because the rule alone
was not enough to stop me repeating it.

Still owed for M3: Task 3.5's operator smoke.

### 2026-08-23 -- M3 smoke round 1: the panel was not usable, and one gap was a claim I never built

Operator opened the panel and found it inert: arrows and Escape did nothing,
a mouse move filled the filter with `[<;0;M[<;;M...` until the list read
"(nothing running)" with no way back, and there was no way to start a second
child at all. Four bugs and a scope gap, all mine.

- **Mouse reports were typed into the filter.** `panelKey` took any printable
  byte as typeahead -- and every byte of `\x1b[<0;12;4M` after the ESC is
  printable. New `DecodePanelKeys` frames sequences through
  `cmd/internal/ansi` and DROPS the ones the panel does not use, rather than
  letting them decay into text. Two framing details it had to learn: `ansi.Frame`
  puts `O` in the two-byte class, so `\x1bOA` leaked its `A` as a rune until
  SS3 was handled first; and a bare ESC reports Incomplete, so the Escape KEY
  needed the same "a keystroke arrives as its own read" discriminator the
  Interceptor uses.
- **Escape did nothing.** It now backs out: clears the filter if there is one,
  otherwise returns to the actor. A picker with no way out is a trap.
- **Arrows did nothing.** The panel had no cursor at all -- so no highlight,
  and no way to tell what Enter would do. `PanelModel` carries one, clamped
  rather than wrapped, and preserved across filtering.
- **No notification in the panel.** The bell showed only on the status row,
  competing for one line. The panel is the place to LOOK, so it marks the actor.
- **`start` was declared and never wired -- and my audit passed anyway.**
  `PanelActions()` returned four names; the audit asserted each is a declared
  `couchcore` operation, which a list that does nothing satisfies. All four are
  now reachable (`s`/`x`/`n`/`d`, with a prompt for the ones needing an
  argument) and dispatch through the injected `Operations()` table, and the
  audit checks REACHABILITY as well as declaration.

Two lessons recorded: a capability audit that checks declaration passes on a
list that does nothing; and framing input is not optional once you accept
keystrokes.

**What the operator asked for that is now built:** a panel you can arrow
through, type-ahead to filter, jump into by number, and that shows which actor
wants you. What is deliberately still absent: mouse selection (couch drops
mouse reports rather than acting on them) -- worth revisiting only if the
operator wants it.

### 2026-08-23 -- M3 smoke round 2: Escape was dead, for the reason ctrl-space was

Operator: "after ctrl-space, esc doesn't get back to previous screen".

Same root cause as M2's ctrl-space bug, which I fixed for ONE key. zellij
enables the Kitty keyboard protocol, so a real session's Escape arrives as
`\x1b[27u` -- and the panel's Escape, Enter and arrows were all decoded only in
their legacy forms. My tests fed the legacy bytes, so they passed.

Fixed generally rather than per-key: `decodeCSIu` reads the protocol's
`CSI <codepoint> [;<mods>] u` and maps by CODEPOINT, so a key nobody enumerated
still decodes. Modified printables are refused -- ctrl+a must not insert an `a`.
Arrows accept parameters, since a modifier does not stop an arrow being an
arrow. Both encodings are pinned end to end through the console, and dropping
CSI-u decoding turns 12 assertions red.

Lesson recorded: a key-encoding fix must cover every key the surface consumes,
because a per-key fix guarantees the next key reports the same bug.

### 2026-08-23 -- the tree-occupied refusal named an action couch cannot perform

Operator hit `couch start` in brain and got the one-agent-per-tree refusal. The
GUARD was correct -- a couch from the earlier smoke was still alive, and a
pruning test confirms a dead incumbent does not refuse. The ADVICE was not:

```
  -> switch to it, or --same-tree (this repo runs one agent at a time)
```

"switch to it" names a remedy couch has no verb for: attaching to a session
another couch process hosts needs `pair#147`'s transport. An operator who tries
to follow it finds nothing, and reaches for `--same-tree` -- the one option that
BYPASSES the guard. A refusal that pushes the operator toward the escape hatch is
worse than no advice.

It now offers commands that exist (`couch stop <ref>`, `couch start <ref>
--same-tree`), says plainly that attaching needs `#147`, and a test asserts every
`-> couch <verb>` it prints is a DECLARED operation -- so the advice cannot drift
from the verb set the way it just did.

### 2026-08-23 -- M3 smoke round 3: starting worked below the panel but never joined it

Operator confirmation: `ctrl-space`, Escape, and switching during child output
work. The panel still showed only one actor after `s` started another, and
typeahead returned no match for both `brain` and `pair`.

The two symptoms were one boundary failure. Panel actions erased the value
returned by `couchcore.Operations()`, so `start` registered and spawned a child
but its `StartResult` never reached `Console.Attach`. At the same time,
`rebuildPanel` put the console-local child id in `PanelRow.Tree`; production
typeahead delegates to `Couch.LookupTrees`, which correctly returns real
worktrees, so the keys could never match.

Fixed the class (ARCH-PURPOSE): operation results now cross the injected
dispatcher and a returned terminal child joins the live console; panel rows
carry worktree identity for matching separately from child identity for
switching and bell state. The typeahead regression was proven RED against a
real-worktree resolver, the panel-start regression was proven RED at the
operation-result boundary, and the bell join was found by the same identity
shadow-sweep. Targeted `couchtty` + `couchcmd` suites are green. Task 3.5 remains
open pending the repeated real two-actor smoke.

### 2026-08-23 -- M3 smoke round 4: the panel displayed a label its resolver did not know

Operator typed `pair`, exactly the fallback label on screen, and still got
`(no match)`. The prior fix made the row carry the real worktree, but the shared
`LookupTrees` rule searched only operator names/descriptions and the
agent-published description. `PanelModel` independently displayed
`Worktree.Repo()` when no operator name existed. The plan-quality gate had even
documented "Repo is not matched"; the later fallback-label decision failed to
revise that contract.

Fixed at the shared source (ARCH-DRY): `LookupTrees` now matches repo basename
as well. The regression test models `/w/pair` with no explicit name and was
observed RED (`LookupTrees(pair) = []`) before the change, then GREEN. This is
the user-visible class (ARCH-PURPOSE): text rendered as the panel's identifying
label must be typeable back into its typeahead. Task 3.5 remains open for the
real rerun.

### 2026-08-23 -- M3 operator smoke passed

Operator confirmed the repeated real-stack smoke after `4e0a1ad`: the second
actor appears in the panel, repo-label typeahead resolves it, and the complete
M3 smoke now passes. Earlier rounds separately confirmed `ctrl-space` and
Escape, deterministic switching during child output, and the panel's keyboard
navigation. This supplies Task 3.5's missing external behavior evidence; M3 is
ready for its SDLC-owned boundary review.
