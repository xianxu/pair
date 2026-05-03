---
id: 000011
status: working
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# Outer-PTY passthrough for OSC notifications

## Problem

When pair runs inside an outer wrapper (e.g. cmux) that watches the agent's PTY stream for OSC 9 / OSC 777 escapes to surface "agent needs attention" indicators, zellij silently eats those escapes. Empirically verified: OSC 9 and OSC 777 from inside a zellij pane do NOT reach cmux's stream watcher; the same escapes from a plain shell inside cmux do.

Zellij 0.44 forwards BEL (verified via Apple Terminal — `printf '\a'` rings whether zellij is in the chain or not), but cmux watches OSC, not BEL. So the BEL passthrough doesn't help for cmux integration.

## Spec

Two-layer solution:

1. **Outer-TTY capture.** `bin/pair` calls `tty(1)` before invoking zellij (this path IS the outer PTY) and writes it to `$DATA_DIR/outer-tty-${PAIR_TAG}`. Refreshed on every attach because the outer PTY changes across detach/reattach while pane-shell env stays frozen at session-creation time (env-var approach would go stale).

2. **Two consumers** of the captured path:
   - **`bin/pair-wrap`** (Python): transparent PTY proxy around the agent. Watches the agent's output stream for BEL / OSC 9 / OSC 777, writes OSC 9 directly to the outer-TTY path on detection. Wired into `zellij/layouts/main.kdl` so the agent runs under it automatically. **Zero agent config required** — any agent that bells on idle lights up the outer wrapper.
   - **`bin/pair-notify`** (bash): hook helper for richer signals (semantic events, custom messages). Intended for Claude Code `Notification`/`Stop` hooks where the wrapper's coarse BEL signal isn't enough.

## Plan

- [x] `bin/pair`: capture `tty` after `PAIR_TAG` is finalized on both attach and create branches; write to `$DATA_DIR/outer-tty-${PAIR_TAG}`. Remove the file alongside `cleanup_quit_marker` on full quit.
- [x] `bin/pair-notify`: helper. Writes OSC 9 (or 777 with `--osc 777`) to the recorded outer TTY. No-op + non-fatal on missing file or stale path.
- [x] `bin/pair-wrap`: PTY proxy. Forwards stdin/stdout transparently with SIGWINCH propagation. Watches output for BEL / OSC 9 / OSC 777, emits OSC 9 to outer TTY (rate-limited 0.5s). Detection failures swallowed so the proxy never blocks the agent.
- [x] `zellij/layouts/main.kdl`: wrap the agent invocation in `pair-wrap`.
- [x] README: subsection covering both layers + sample claude hook config.
- [x] `atlas/architecture.md`: combined section on capture + both consumers.
- [x] Smoke test (out of pair): `pair-wrap echo` forwards output, exit codes propagate (0, 7, 127 for missing binary), BEL in wrapped output → OSC 9 written to fake outer-tty file.
- [ ] **End-to-end verification (user-driven):** detach current pair session, reattach via fresh `pair`, run normal claude tasks inside cmux, confirm cmux tab badges when claude waits for input. Open question: does claude emit BEL by default, or only with `bell` setting enabled?

## Log

### 2026-05-03

Filed after empirical investigation showed zellij filters OSC 9/777 (drops them as unrecognized escapes during virtual-screen reconstruction). BEL forwarding (added in zellij 0.44 per #776/PR #981) works but isn't useful for cmux which only watches OSC.

Initial design was hook-only (`pair-notify` called from claude `Notification` hook). User pushed back: cleaner answer is a transparent PTY wrapper that detects agent BEL/OSC and translates to outer-PTY OSC 9, removing the per-user hook config requirement. Pivoted to two-layer design — wrapper as the headline mechanism, `pair-notify` retained for hook-driven richer signals.

Implementation done. Out-of-pair smoke tests pass: `pair-wrap echo` forwards correctly, exit codes propagate, BEL inside wrapped output triggers OSC 9 to outer-tty path, native OSC 9 in wrapped output also triggers (BEL terminator path), no-PAIR_TAG case is a graceful no-op. End-to-end verification with cmux pending — needs a fresh pair attach since the in-flight session predates the layout change.
