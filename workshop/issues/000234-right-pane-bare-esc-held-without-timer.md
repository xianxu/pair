---
id: 000234
status: codecomplete
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.79
started: 2026-09-12T15:41:35-07:00
actual_hours: 0.71
---

# A bare ESC in the right pane is held until the next keystroke: pair term treats it as an unfinished Alt chord and has no ambiguity timer outside rename

## Problem

Operator report, 2026-09-12: with nvim running inside the right pane
(`pair term`), leaving insert mode takes **two** presses of ESC.

Root cause, from the mux's stdin loop (`cmd/internal/termcmd/run.go`):

- Every legacy Alt chord pair recognizes begins with ESC (`\x1bx`, `\x1bj`,
  …, `workbenchshortcut/shortcut.go:357`). So a read that contains only
  `\x1b` satisfies `IsChordPrefix` (`shortcut.go:493`: shorter than a
  candidate and a prefix of it).
- The main loop then does exactly this (`run.go:540-543`):

  ```go
  if workbenchshortcut.IsChordPrefix(data) || isSGRMousePrefix(data) {
      held = append(held, data...)
      break
  }
  ```

  and waits for the **next read**. The only timer in that `select` is the
  rename one (`case <-timer.C(): applyRename(...)`), armed only inside a
  rename session — where a lone pending ESC gets `timer.Reset(50 *
  time.Millisecond)`. Outside rename, nothing ever releases `held`.

So a single ESC sits in `held` until the operator types something else. The
second ESC arrives, `held+data` is `\x1b\x1b`, which is no chord, and both
bytes go to nvim at once — normal mode, on the second press. That is the
symptom exactly.

**It is worse than a double-press.** The follower decides what the held ESC
becomes. ESC then `j` — the most common thing a vim user types after leaving
insert — is delivered as `\x1bj`, which is `ChordAltJ`: pair's focus-left
chord fires, and nvim sees neither the ESC nor the `j`. Same for ESC then
`k`, `t`, `w`, `r`, `x`, `/`. By construction from the chord table; not
reproduced, and worth reproducing first because it turns "annoying" into
"keystrokes go to the wrong pane".

This is the ambiguity every ESC-prefix decoder has, and pair has already
solved it twice: couch's two input framers share `escapeAmbiguity = 35 *
time.Millisecond` (`couchtty/keys.go:49`, "the one deadline used by both
terminal-input framers to distinguish an ESC key from the first byte of a
split escape sequence"), and termcmd's own rename decoder uses 50 ms. The
main path of `pair term` is the one framer without it. `workshop/lessons.md`
already carries the rule (§"Escape decoders must distinguish prefixes…", and
the read-boundary lesson: "a bare ESC that might be the prefix of a following
CSI: read boundaries carry no semantic meaning, so resolve the ambiguity
explicitly").

Why it does not bite in the draft pane or the agent pane: the draft's nvim
receives keys through zellij with the kitty protocol pushed, where ESC is
`\x1b[27u` and unambiguous; the agent pane goes through pair-wrap, which has
its own decoder. Only `pair term`'s stdin path forwards raw legacy bytes with
a hold and no deadline.

## Spec

Give the main loop the same deadline the other three framers have.

1. **One ambiguity deadline, one owner.** Lift `escapeAmbiguity` (35 ms) out
   of `couchtty` into `workbenchshortcut` — the package that owns the chord
   table is the right owner of "how long a chord prefix may stay open" — and
   use it from couch's two framers, termcmd's rename decoder (replacing the
   local 50 ms), and the main loop. Four sites, one constant (ARCH-DRY).
2. **Main-loop timer.** When `held` becomes non-empty and is a bare ESC (or
   any chord prefix), arm the timer; on expiry, `mux.writeActive(held)` and
   clear it. New input before expiry appends and re-evaluates as today; a
   complete chord or a non-prefix flushes and stops the timer. Mirror the
   rename decoder's shape — it is the same problem three functions up.
3. **Tests generated from the chord table**, per the lessons rule: for every
   chord sequence, the split where only the first byte arrives, then (a) the
   rest arrives before the deadline → chord fires, (b) nothing arrives →
   bytes forwarded as typed after the deadline, (c) a non-chord byte arrives →
   both forwarded, no chord. Plus the specific regressions: `ESC`,`ESC` →
   two ESCs to the child; `ESC` … 35 ms … `j` → ESC then `j` to the child,
   not `ChordAltJ`.

Out of scope: making the right pane's child speak the kitty protocol (would
remove the ambiguity at the source, but it is a much larger change to how
`pair term` hosts its children, and the deadline is what every other framer
in this repo does).

## Done when

- In the right pane, one ESC leaves insert mode in nvim.
- ESC followed by `j`/`k` after the deadline reaches nvim as two keys; the
  Alt chords still fire when typed as chords. (`ESC`,`j` typed inside the
  deadline still decodes as a chord — that residual is #227's.)
- `escapeAmbiguity` has one definition, used by all four framers.
- The generated split tests pass; the two regressions above are explicit.

## Plan

Durable plan: `workshop/plans/000234-right-pane-bare-esc-held-without-timer-plan.md`.

- [x] Reproduce ESC-then-`j` in the right pane (does focus jump? record it) — reproduced at the pump (see Log); live pane pending
- [x] Move `escapeAmbiguity` to `workbenchshortcut`; re-point couch's framers and termcmd's rename decoder
- [x] Arm/expire the timer in the main loop; flush `held` on expiry
- [x] Table-generated split tests + the two explicit regressions
- [x] Manual: nvim in the right pane — single ESC, ESC+j, Alt+j — done against real nvim under a pty by `probes/escsmoke` (6/6 on this branch, 3/6 on an origin/main control binary; see Log); in-zellij-pane confirmation requested from the operator

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Design at ×0.2 (the plan pre-resolves the timer ownership, the constant's home, and the test oracle); impl at 40% of the v2 ranges; +15% buffer for a thorough plan doc.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.03 impl=0.08
item: smaller-go-module  design=0.06 impl=0.16
item: smaller-go-module  design=0.04 impl=0.20
item: atlas-docs         design=0.02 impl=0.04
item: milestone-review   design=0.00 impl=0.14
design-buffer: 0.15
total: 0.79
```

- lift `EscapeAmbiguity` + re-point three sites (mirror of couch's arm) — 0.03 / 0.08
- main-loop arm + expiry branch + timer-type rename — 0.06 / 0.16
- generated split tests, two regressions, fake hook, deterministic split test — 0.04 / 0.20
- atlas paragraph — 0.02 / 0.04
- close review — 0.00 / 0.14

## Log

### 2026-09-12
- 2026-09-12: closed — Pump-level regressions red on main then green: TestBareEscapeIsForwardedAfterTheDeadline, TestEscapeThenJAfterTheDeadlineIsTwoKeysNotAltJ, TestTheRealDeadlineForwardsABareEscape (real timer), TestATornMousePrefixMeetsTheSameDeadline, TestEveryChordSplitAtEveryByteResolvesAgainstTheDeadline (546 cases from the chord table; the deadline-first case gates EOF behind the observed write so a deleted expiry branch goes red). go test -race x2 clean. Full make test green outside the sandbox. LIVE: probes/escsmoke runs the real pair term under a pty with real nvim and asks nvim over RPC — control binary from origin/main reproduces the report (one ESC: mode()=i; ESC,120ms,j: still i, cursor unmoved; 3/6 fail), this branch 6/6 pass incl. Alt+j-as-one-write still consumed. Operator asked to confirm in the zellij pane with a fresh split; that is the only step not run here.; review verdict: SHIP

- Filed from the brain advisor session on the operator's report. Mechanism
  read from `run.go:540-543` and the `select` at `:461`; the rename path's
  50 ms timer and couch's 35 ms `escapeAmbiguity` are the in-repo precedents.
  The ESC-then-`j` misroute is derived from the chord table, not observed —
  first Plan step is to observe it.

- Reproduced at the pump before changing code (`TestEscapeThenJAfterTheDeadlineIsTwoKeysNotAltJ` red on `main`): `ESC` in one read, `j` in the next → the child received nothing; the pair was decoded as `ChordAltJ` and swallowed. In the live pane Alt+j is swallowed in the right-terminal role, so the visible symptom is `j` lost, not a focus jump. The generated table (`TestEveryChordSplitAtEveryByteResolvesAgainstTheDeadline`, 546 cases) was red on every "armed the deadline" assertion — nothing armed one.
- Shipped in 6d191022: `workbenchshortcut.EscapeAmbiguity` (35 ms) replaces couch's private constant and the rename arm's 50 ms literal (ARCH-DRY); the pump's timer is now the single escape-ambiguity deadline, owned by whichever side owns the pending bytes — rename session or plain `held` — with the arm/stop at the tail of each read handler (ARCH-ORDER). Plain arms for any held prefix, not only a lone ESC; couch arms only for the lone ESC, and the difference is deliberate (local pty, torn sequences re-join in microseconds).
- Tests: the two explicit regressions, the chord-table generated split tests (every sequence × every split × {tail before deadline, deadline first, non-chord follower}), one real-timer check that `pumpStdin`'s own timer actually ticks, and the existing split-chord tests moved onto the fake timer so they cannot race the 35 ms wall clock. `TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit` deleted as superseded by the generated table. `go test -race` clean ×3 on the pump tests; full `make test` green outside the sandbox (pty children are blocked inside it).
- Residual, recorded in Done-when: `ESC`,`j` typed inside 35 ms still decodes as a chord. #227's alt-screen passthrough is what closes it for full-screen children, which is why the two issues are related and #227 is next.
- Manual step (nvim in the right pane: single ESC, ESC+j, Alt+j, Alt+t) needs the operator's live session with the rebuilt binary — a new split (Alt+Shift+d) execs a fresh `pair term`; existing tabs keep the old process.
- Close round 1 came back FIX-THEN-SHIP. BR-3: the `RenameTimer` interface had NOT been renamed — BSD sed has no `\b`, so the substitution was a silent no-op and only the concrete types moved; renamed for real and verified by grep (lesson added). BR-4: no live evidence for the pane-level Done-when. Added `probes/escsmoke`: it runs the real `pair term` under a pty with a real `nvim --clean --listen` child and asks nvim over its RPC socket (`--remote-expr mode()` / `line('.')`), so the oracle is the editor, not a screen scrape. Against a control binary built from origin/main it reproduces the report exactly — one ESC then nothing: `mode() = "i"`; ESC, 120 ms, `j`: still `"i"`, cursor unmoved (swallowed as Alt+j) — 3/6 steps fail. Against this branch 6/6 pass, including Alt+j typed as one write still being consumed. Minor findings folded in: generated case (b) now gates EOF behind the observed write so it proves the expiry branch (the reviewer showed a scratch revert of that branch left it green); an explicit torn-SGR-mouse-prefix test pins the "any held prefix" decision; the latency comment now says the two timeouts add (35 ms + nvim's 50 ms ≈ 85 ms worst case) rather than nest.
