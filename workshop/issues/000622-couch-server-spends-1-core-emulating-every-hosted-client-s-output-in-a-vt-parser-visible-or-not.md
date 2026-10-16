---
id: 000622
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '265dd0f29173b5fbedd04bc26389f4c5795c2634' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.

## Spec

- Measure first: per hosted client, ingest bytes per second and CPU, at idle and during a busy turn; which consumers
  need the emulated screen (console view, peek, waiting/idle detection, resource monitor).
- Then the smallest change that keeps every consumer correct. Options to weigh: emulate only the slot(s) the console
  is showing and keep a cheap raw ring for the rest (replaying it when a slot is shown or peeked); or parse lazily
  (append raw bytes, emulate on demand). Idle/waiting detection must keep working for off-screen slots.
- No new background workers; bounded memory per client.

## Done when

- The Couch server's steady CPU with ~27 hosted clients drops by a measured amount (before/after `sample` and CPU%),
  with the console, `--peek` and waiting notices behaving as before (tests plus a live check).

## Plan

- [ ]

## Log

### 2026-10-16
