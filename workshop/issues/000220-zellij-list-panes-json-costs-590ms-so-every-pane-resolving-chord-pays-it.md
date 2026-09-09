---
id: 000220
status: working
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
started: 2026-09-09T13:39:07-07:00
---

# zellij list-panes --json costs 590ms, so every pane-resolving chord pays it

## Problem

Operator report: `Alt+Shift+←/→` (switch the right pane's tabs from the draft,
`#216`) has *"too much of a delay"* next to focusing the right pane and pressing
`Alt+←/→`.

That comparison is not the defect. The in-pane chord is in-process — bytes reach
`pair term`'s stdin and `handleTerminalChord` calls `mux.previousTab()` — so it
is ~0ms and nothing crossing a process boundary will match it. The defect is
what the crossing costs, and it is **not** the process boundary either.

### Measured on this machine, 2026-09-09

| call | median of 5 |
|---|---|
| `/usr/bin/true` | 6.0 ms |
| `bin/pair --version` (binary load) | 9.0 ms |
| `zellij action list-panes` (plain) | 22.6 ms |
| **`zellij action list-panes --json`** | **590.7 ms** |
| `zellij action list-panes --json --command --state --geometry` | 598.6 ms |
| `pair layout switch-terminal-tab` end-to-end | **631 ms** |
| `pair layout focus-terminal` end-to-end | **671 ms** |

**It is `--json` alone.** Adding `--command`, `--state` and `--geometry` costs
nothing measurable on top; dropping `--json` takes 590ms to 22ms. zellij
0.44.3's JSON pane serialization is 26x its plain listing.

`layoutcmd.OSRuntime.ListPanesJSON` runs
`zellij action list-panes --json --command --state --geometry`, so **every
pane-resolving action pays ~600ms**:

- `pair layout focus-terminal` — `Alt+k` from the draft or agent pane
- `pair layout toggle-focused` — `Alt+Shift+Enter`
- `pair layout switch-terminal-tab` — `Alt+Shift+←/→` (`#216`)

**This is pre-existing.** `Alt+k` has cost ~670ms since `layoutcmd` was written;
`#216` inherited it and put it on a chord pressed often enough to notice.

### Why the call is avoidable

`resolveRightTerminal` needs one thing: the right terminal's pane id. Both
callers use only `terminal.ID`. Two sidecars already answer that without zellij:

- `$PAIR_TERMINAL_PANES_PATH` — the TerminalPaneRegistry, `paneID pid` per line,
  self-registered by each `pair term`, deduped by pane id and filtered by pid
  liveness.
- `$PAIR_LAST_TERMINAL_PANE_PATH` — the recorded last-used split half.

The pane list is only needed for the tie-break `pickRightTerminal` applies when
the registry is ambiguous AND no half is recorded — zellij focus, then pane
order.

Second, smaller cost on the same path: `procutil.Alive` shells out to
`kill -0 <pid>` — one subprocess per registry line (~6ms each) for what is a
single syscall.

## Spec

**Resolve the right terminal from the sidecars when they can answer, and call
zellij only when they cannot.**

- Fast path, no `list-panes`: read the live registry ids. If the recorded
  last-terminal is among them, that is the answer. If there is exactly one live
  id, that is the answer — there is nothing to tie-break.
- Slow path, unchanged: two or more live ids and no recorded preference. Fall
  back to today's `list-panes --json` + `pickRightTerminal`, preserving the
  zellij-focus-then-pane-order tie-break exactly.
- The fast path must produce the SAME id the slow path would in the cases it
  handles, or `Alt+k` and `Alt+Shift+←/→` start disagreeing about which split
  half they mean — the property `#216` BR-10 exists to protect.
- `procutil.Alive` becomes a `syscall.Kill(pid, 0)` check. `EPERM` means the
  process exists but is not ours: alive. Every caller benefits.

**Not in scope:** matching the in-pane chord's latency. A cross-process delivery
cannot reach ~0ms without a resident control channel in `pair term`'s input
loop, which is a much larger change to a surface `#199`/`#216` showed is
delicate. The goal is to remove the 590ms that has no reason to be there.

## Done when

- `pair layout switch-terminal-tab` and `pair layout focus-terminal` no longer
  call `list-panes --json` when the registry can answer, measured before/after
  on a quiet host and recorded in the `## Log`.
- The fast path and the slow path are proven to agree on the same inputs.
- The slow path still runs, and is still correct, when the registry is ambiguous
  and no half is recorded.
- `procutil.Alive` performs no subprocess.
- No regression in `#216`'s tests or `#123`/`#199`'s focus behaviour.

## Plan

- [ ] `procutil.Alive` via `syscall.Kill(pid, 0)`, treating `EPERM` as alive;
      unit rows for live, dead, and not-ours.
- [ ] Fast-path resolution in `layoutcmd`, returning the pane id; slow path kept
      verbatim behind it.
- [ ] Agreement test: for inputs the fast path handles, assert it returns what
      `pickRightTerminal` would have.
- [ ] Ambiguity test: two live ids and no record still consults the pane list.
- [ ] Re-measure end-to-end and record the before/after in the `## Log`.

## Log

### 2026-09-09

Filed from operator report on `#216`. Measured before designing, per `#201`'s
lesson — the first hypothesis was the process boundary and the subprocess count,
and the measurement refuted both: the boundary costs ~31ms of the ~631ms, and a
single zellij flag costs the rest.
