---
name: zellij-birth-race
description: Count how many new zellij sessions die at birth — bare zellij (poke) or Pair's cold create (launch), optionally under a list-sessions hammer.
---

# Counting sessions that die at birth (pair#287)

zellij 0.45.1 panics (`zellij-server/src/lib.rs:1462`, `Option::unwrap()` on
`None` in `ServerInstruction::RemoveClient`) when a connection accepted
*before* the first real client initialized the session closes. `zellij
list-sessions` connects to every session socket, so any `list-sessions`
running while a new server is coming up can kill that server. The operator
sees a blank pane that echoes typed keys.

```bash
go build -o "$SCRATCH/birthrace" ./probes/zellijbirthrace
"$SCRATCH/birthrace" poke -n 5                          # the zellij bug, no Pair
"$SCRATCH/birthrace" launch -n 10 [-pair PATH]          # Pair cold creates
"$SCRATCH/birthrace" launch -n 10 -hammer 10ms          # ...under a list-sessions loop
```

With no arguments it only describes itself and exits 0. That keeps `make
test-smoke`, which runs every probe bare, from killing zellij servers or
starting sessions. It is an instrument, and this file is its runbook.

Run it with the **harness sandbox off**. zellij's sockets and log live under
the real `$TMPDIR` (the sandboxed shell's `$TMPDIR` differs), and ptys are
refused inside the sandbox.

`-pair` must point at a binary named `pair`. The binary dispatches on its own
name, so a copy named `pair-before` runs as `pair-go` and every trial comes
back inconclusive. To measure an older build, put it at `<dir>/pair`.

## What each mode proves

- **poke:** zellij alone. It starts a bare session and connects to its
  socket, then hangs up, the moment the socket appears.
- **launch:** Pair end to end. It runs `pair resume <new-tag> --layout3` in a
  scratch git repo. A trial counts as `born` when the agent pane's sidecar
  appears (Pair's own birth evidence) and the session is still listed and not
  EXITED 500 ms later. It counts as `died` when zellij's log gains a panic
  line.
- **-hammer DUR:** runs `zellij list-sessions --short` back to back, DUR
  apart, from each trial's start until birth or death. 10 ms is what Couch's
  cold-resume registration poll did before #287.
- **-exit-wait DUR** (default 20 s; pair#288): after a death, the trial waits
  this long for Pair's launcher to exit. It notes when the launcher exited
  and whether a zellij client was left behind, matched by the trial's tag
  and `--new-session-with-layout` in `ps`.
  - A launcher still running after DUR counts as `hung` if the death came
    before the pane was born; that is what the launcher's birth watch ends.
  - It counts as `hung-after-birth` if the death came after the pane was
    born, which the watch does not cover.
  - A fixed launcher exits one of two ways: within about 1–2 s when its
    client exits on its own, or within about 10–11 s when the client hangs
    and the watch ends it.

A trial that shows neither birth nor panic is `inconclusive`. That is a
precondition failure, not a verdict (pair#208). The note after the verdict
says why.

## Running it from inside a Pair pane

The launcher refuses nested launches by walking the process ancestry for
zellij, so scrubbing the environment is not enough. The probe works around
this: `launch` re-execs itself through `sh -c '… &'`, the shell exits, and the
child reparents to launchd. The child writes a report file and the parent
prints it.

## What it leaves behind

Nothing. Each launch trial works in:

- a scratch repo,
- an isolated `XDG_DATA_HOME`, so none of the operator's Pair state is
  touched,
- a bin dir holding a fake `claude` that only sleeps.

Teardown deletes the zellij session under the name Pair recorded for the
trial's tag. It then kills every process whose argv carries that tag (the
launcher, the title poller, the session watcher, a dead server's orphaned
client) and removes the three directories. `CMUX_*` is scrubbed, so the
probe never retitles the operator's cmux workspace.

The zellij log is shared, so a panic from an unrelated session starting on
the same machine during a trial would be counted against it.

## Results (macOS, zellij 0.45.1, 2026-09-18)

| Build | Mode | Died | Hung |
|---|---|---|---|
| — | `poke -n 5` | 3/5 | — |
| main before #287 | `launch -n 10` | 4/10 | — |
| main before #287 | `launch -n 10 -hammer 10ms` | 10/10 | — |
| #287 | `launch -n 20` | 0/20 | — |
| #287 | `launch -n 10 -hammer 10ms` | 9/10 | — |
| main before #288 | `launch -n 20 -hammer 10ms` | 20/20 | 20/20 |
| #288 | `launch -n 20 -hammer 10ms` | 20/20 | 0/20 (each exited 10.5–11.2 s after start, no client left) |
| #288 | `launch -n 10` | 0/10 | — |

The 2026-09-18 runs predate `-exit-wait`, so their Hung column is empty.

The third row is what Couch's cold-resume registration poll did: every cold
resume under it died. #287 gates both the title poller and that poll on the
pane's birth, so Pair's own launch path no longer probes during a birth
(0/20).

An *external* prober still kills new sessions (the `-hammer` rows). That is
the residual risk: any machine-wide `list-sessions`, such as another
thread's 60 s poller or Couch's menu refresh, can land in a new server's
window. At their cadence the odds are low, and only upstream zellij can cure
it.

#288 makes such a death end the launcher instead of hanging it. The `Hung`
column counts launchers still running 20 s after a death before birth: what
Pair failed to notice. Re-run the hammer rows on every zellij version
change, because the launcher's fake models 0.45.1's hung client (it writes
0 bytes, isn't listed, and exits on SIGTERM).
