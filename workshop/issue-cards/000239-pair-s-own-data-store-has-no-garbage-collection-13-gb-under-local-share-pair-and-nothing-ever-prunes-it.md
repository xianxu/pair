---
id: '000239'
status: done
started: 2026-09-13T16:20:21-07:00
created: 2026-09-12
updated: 2026-09-15
estimate_hours: 6.57
actual_hours: N/A
---

# Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it

## Problem

Measured on the operator's machine, 2026-09-12. `~/.local/share/pair` is
**13 GB**. No code path in the tree deletes anything under it except the
exact per-pane sidecars removed on Alt+x; every other family grows until the
disk does.

| family | bytes | notes |
|---|---|---|
| `wrap-events-*.jsonl` | 12.7 GB | 163 files; one is 4.8 GB (`repos/e108517d46ab4575/wrap-events-pair.jsonl`); 345 MB is older than 60 days |
| `scrollback-*.raw` | 767 MB | live scrollback rings; one is 203 MB |
| `parked-scrollback-*.raw` | 543 MB | one per park; parks that were later resumed keep theirs |
| `ledger-*.jsonl` | 73 MB | 143 ledgers; 99% of the bytes are launch boundary snapshots (#238) |
| `draft-`, `log-`, `config-`, `agent-ready-`, `thread-claim-` … | small | per-thread sidecars, hundreds of files |

By contrast the agent's own store, `~/.claude/projects`, is bounded: Claude
Code sweeps transcripts after 30 days and the oldest file on this machine
is 29 days old (650 MB, 2,629 files). Pair records more than the agent does
about the same sessions and keeps it forever.

This is the first instance of the principle filed as ariadne#224
(`ARCH-FUNERAL`): every artifact a program creates needs a designed end — a
time-bound sweep, or an explicit growth bound with its consequences stated.
Pair's data families have neither.
