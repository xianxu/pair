---
id: 000227
status: working
deps: []
github_issue:
created: 2026-09-10
updated: 2026-09-13
estimate_hours: 0.75
started: 2026-09-13T09:18:29-07:00
---

# right-pane chords pass through to a full-screen app

## Problem

The operator runs nvim with parley in the layout3 right pane. Parley binds `<M-t>` to
its outline; `pair term` binds `<M-t>` to "new terminal tab" and takes it first, so
parley never sees it.

The operator's first proposal was to move pair's new-tab chord to `<M-n>`. **That does
not work**, and the reason reframes the issue.

### Why `<M-n>` is unavailable

`workbenchshortcut.Decide` resolves **global** chords before role-scoped ones, for the
right terminal as well (`shortcut.go`):

```go
if in.Role == PaneRoleLeftAgent || in.Role == PaneRoleLeftDraft || in.Role == PaneRoleRightTerminal {
    if decision, ok := DecideGlobal(in.Chord); ok {
        return decision
    }
}
```

`<M-n>` is global — `ChordAltN → ActionRestartPair`, *"reload pair — kill and
re-launch the workbench"*. A new-tab binding on `<M-n>` inside the right-terminal block
would be unreachable: pressing it would raise the *Reload pair?* confirm instead. And
even a free chord would only move the collision to whatever else nvim binds there.

### The collision is wider than `<M-t>`

In the right-terminal role, `pair term` takes these before any child sees them:

- **handled** (turned into pane actions): `<M-t>`, `<M-w>`, `<M-r>`, `<M-S-d>`,
  `<M-k>`, `<M-S-Enter>`, `<M-Left>`, `<M-Right>`;
- **swallowed** (dropped): `<M-j>`, `<M-/>`, `<M-S-c>`, `<C-M-c>`.

So `<M-t>` is the collision the operator noticed. Every full-screen app in the right
pane loses the whole set.

## Spec

**When the active tab's child owns the screen, pane-local chords pass through to it;
`pair term` intercepts them only at a shell.**

### Definition of "full-screen app"

**The child has switched the terminal to the alternate screen buffer** —
`ESC[?1049h` on start, `ESC[?1049l` on exit (also `?1047`/`?47`). This is how nvim,
vim, `less`, `man`, `htop` and `lazygit` take over the window and restore the shell's
scrollback when they quit. It is the app *declaring* ownership of the screen, not a
heuristic, and it selects exactly the programs that expect to own the keyboard.

`ptychild.Screen` already tracks it, and — importantly — already with the `#196`
discipline, because `#209`'s repaint work needed it:

```go
// RepaintModes is what hostty.RepaintFor needs from this child, taken in one
// locked read so the two fields cannot disagree.
// … there is deliberately no exported AltScreenObserved to reach for (#209 BR-12).
func (c *Child) RepaintModes() (altScreen, observed bool)
```

**Use `RepaintModes()`, not the bare `AltScreen()`**, so the unknown case is decided
explicitly rather than read as "off".

### The three states

| observed | altScreen | meaning | chord handling |
|---|---|---|---|
| yes | on | full-screen app owns the pane | **pass through** |
| yes | off | returned to a shell | intercept (today) |
| **no** | — | child never used the alt screen | intercept (today) |

The unknown row maps to "intercept" for a reason specific to `pair term`: it spawns
every tab's child and observes each byte from birth. There is no fresh-Screen-for-a-
running-child path like couch's reattach (`#196`), so "not observed" genuinely means
"never entered the alt screen", which for a tab means a shell. It is also the
no-regression choice.

### Where the definition is imperfect — and why that is acceptable

- **Interactive apps that render inline**, without the alt screen: `fzf --height`,
  `less -X`, REPLs, some agent CLIs that preserve scrollback. `pair term` keeps
  intercepting — **today's behaviour**.
- **Alt-screen apps where the operator still wants pair's chord**: rare.

The failure modes are lopsided in the safe direction: where the definition is wrong,
the result is the status quo, never a new problem.

### Scope and the one decision

Apply passthrough to the **handled and swallowed** role-scoped chords above. **Global
chords are unaffected** — they are workbench-wide by design and resolved before any
role logic.

**Decide: is an always-available tab chord needed?** With passthrough, the operator
cannot open a tab while nvim has focus in the right pane; they leave nvim first. If
that is too limiting, add **one** escape chord that creates a tab regardless of
alt-screen state, chosen to avoid both the global table and common nvim bindings
(e.g. `<M-S-t>`). Default: **no escape chord** — add it only if the operator finds the
restriction biting, since each always-on chord is keyboard real estate taken from
every app.

### Where the decision is made

`Decide` takes a `ShortcutInput`; it has no view of the child. The alt-screen state
must reach it from `pair term`, which owns the active tab's `Child`. Prefer passing it
**in** as an input field over having `Decide` reach out — `Decide` is a pure mapping
from input to disposition today, and keeping it pure keeps it table-testable
(`ARCH-PURE`).

## Done when

- With nvim (or any `?1049` app) focused in the right pane, `<M-t>` reaches it — parley
  opens its outline.
- At a shell prompt in the right pane, `<M-t>` still opens a tab.
- Every handled and swallowed right-terminal chord above passes through under a
  full-screen app, and behaves as today at a shell — a table test over all three
  states for each chord.
- The unknown alt-screen state intercepts, asserted.
- `<M-k>` (focus-left) still returns to the left stack under a full-screen app —
  it does NOT pass through, so the operator is never trapped in nvim.
- Global chords (`<M-n>` restart, from-anywhere tab switch, etc.) are unchanged
  in every state.
- `Decide` stays pure and untouched; the passthrough gate lives in the pump
  (the one place all chord-dispatch paths funnel through), keyed on a pure
  `RightTerminalChordPassesThrough` predicate.
- The escape-chord decision is recorded (see Revisions): none added; switching
  stays global and focus-left survives.
- `atlas/` records the rule, and `pair keys` / help reflects that right-pane chords are
  conditional.

Durable plan: `workshop/plans/000227-right-pane-chords-pass-through-to-a-full-screen-app-plan.md`.

## Plan

- [x] `IsGlobalChord` + `RightTerminalChordPassesThrough` (pure predicates, `M-k` excluded)
- [x] `activeChildOwnsScreen()` on the mux (from `RepaintModes()`); `activeChildOwnsScreen` on the `ptyWriter` interface + `fakeMux`
- [x] Pump gate: forward a pass-through chord's raw bytes to a full-screen child; dispatch as today otherwise
- [x] Table test: each chord × {fullscreen, shell}; the tri-state accessor test covers unknown; the ESC-then-j, focus-left, and global regressions
- [x] Escape-chord decision recorded (none; switching + focus-left survive) — see Revisions
- [x] Update help text (keyhelp `groupTerminal` heading) and `atlas/`
- [ ] Manual: parley `<M-t>` in right-pane nvim; `<M-k>` back to the agent; ESC+`j` reaches nvim; `<M-t>` at the shell

## Revisions

### 2026-09-13 — pump gate, not a ShortcutInput field (plan-quality round 1)

**Reason.** The Spec sketched adding an alt-screen field to `ShortcutInput` and
having `Decide` return pass-through. That covers only the chords dispatched
through `Decide`; the tab chords (`M-t`/`M-w`/`M-r`/`M-S-d`/`M-Left`/`M-Right`)
go through `handleTerminalChord`, bypassing `Decide` — the very chords the
operator reported. **Delta:** the gate lives in the pump (the one place all
dispatch paths funnel through), keyed on the pure `RightTerminalChordPassesThrough`
predicate; `Decide` stays pure and untouched (ARCH-PURE).

### 2026-09-13 — M-k excluded from passthrough (plan-quality PQ-1)

**Reason.** The gate first passed through every non-global role chord. The
plan-quality review found `M-k` (focus-left) is the ONLY keyboard bridge from
the right terminal back to the left stack, with no global equivalent (zellij's
MoveFocus defaults are unbound). Passing it through would trap an operator with
nvim focused on the right. **Delta:** `RightTerminalChordPassesThrough` excludes
`ChordAltK`, so `M-k` always fires focus-left; a regression test asserts it.

### 2026-09-13 — escape-chord decision: none

Switching tabs survives via the global from-anywhere `M-S-←/→`, and focus-left
survives via the `M-k` exclusion. Only tab create (`M-t`) / close (`M-w`)
require leaving the full-screen app. Per #227's default, no always-available
escape chord is added until that restriction is felt.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Design at ×0.2 (the plan resolves the gate, the predicate and the M-k exclusion); impl at 40% of v2; +15% buffer.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.02 impl=0.08
item: smaller-go-module  design=0.06 impl=0.16
item: smaller-go-module  design=0.04 impl=0.16
item: atlas-docs         design=0.02 impl=0.05
item: milestone-review   design=0.00 impl=0.14
design-buffer: 0.15
total: 0.75
```

- `IsGlobalChord` + `RightTerminalChordPassesThrough` (pure) — 0.02 / 0.08
- `activeChildOwnsScreen` + pump gate + `ptyWriter` method — 0.06 / 0.16
- table test + 3 regressions + harness seam — 0.04 / 0.16
- keyhelp heading + atlas — 0.02 / 0.05
- close review — 0.00 / 0.14

## Log

### 2026-09-10

Operator request to free `<M-t>` for parley by moving pair's new-tab chord to `<M-n>`.
Checked the dispatch before agreeing: `Decide` resolves global chords first for the
right terminal, and `<M-n>` is the global restart-pair chord, so the move would have
shipped an unreachable binding that raises *Reload pair?* instead.

Reframed from "move one chord" to "stop stealing chords from full-screen apps", which
fixes `<M-t>` and the eleven other right-pane chords at once with no moves. The
alt-screen definition turned out to be already implemented with the `#196` tri-state,
because `#209` needed it — so the observed/unknown distinction the design depends on
exists and is enforced as a locked pair.

### 2026-09-13 (close)

- **Live verification (BR-4).** `probes/escsmoke` drives a real `pair term`
  under a pty with a real `nvim --clean` (which enters the alt screen) and asks
  nvim over its RPC socket. On this build: "Alt+j passes through to the
  full-screen nvim (cursor moves down)" PASSES and "Alt+k does NOT pass through
  (cursor unmoved)" PASSES — the mechanism works with a real editor. On an
  origin/main control build (no #227) the Alt+j step FAILS (cursor stays on
  line 1, Alt+j swallowed) — the delta. The operator's in-workbench parley
  check (`M-t` opens parley's outline; `M-k` returns to the agent; `M-t` at a
  shell still opens a tab) is the one remaining manual step, unticked because
  it needs the running workbench.
- BR-2: the `pair keys` terminal heading now names the two exceptions
  (`Alt+k`, `Shift+Alt+←/→`) rather than claiming universal passthrough.
- BR-3: README's layout-3 section now describes the conditional passthrough.
