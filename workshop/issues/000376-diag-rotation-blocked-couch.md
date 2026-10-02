---
id: 000376
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '78fb262ffbff98602c9efe190d34e43a7ba7ad45' # card fields mirrored from issue-cards; edit via sdlc
---

# Diagnostic log rotation blocked while Couch runs

## Problem

Diagnostic logs written through `cmd/internal/diagnosticlog` never rotate while
Couch is running. Measured 2026-10-01 on the dev machine:
`~/.local/share/pair/repos/2e51fcf9799b1d8f/wrap-events-couch-6b111ea230c149dc.jsonl`
reached **550 MB** after 2.5 days (growing ~53 KB/s while codex streams output).
Its `.pair-diagnostics/state.json` still shows `current.start` 2026-09-29T14:45Z,
so it has never rotated, although the design rotates at 64 MB
(`MaxGenerationBytes`) or 24 h (`GenerationPeriod`), `writer.go:24`.

Side effects: `pair wrap` ~23% CPU and fseventsd ~47% CPU (every event is an
append plus a `state.json` republish), and unbounded disk growth.

Root cause chain (read from code, confirmed against live processes):

1. `Writer.Write` sees size/age over the limit and calls
   `scheduleMaintenance`, which runs `_ = Maintain(...)` (`writer.go:757`) —
   the error is discarded, so the refusal is invisible.
2. `Maintain` requires `VerifyWriters` (`proof.go:30`). Its `Runtimes` scan
   (`ps -axo pid=,comm=`) treats every `pair`/`couch`/`nvim`/`zellij` process
   as a potential writer unless it carries `PAIR_RETENTION_PROTOCOL=1` or a
   `PAIR_DATA_DIR` naming a different root. A process with an empty root counts
   as same-root → `ErrUnknownWriters`.
3. Live: 13 processes lack both variables — `couch` itself and every
   `pair --couch-session-v1 resume <tag>` client it spawns. So whenever Couch is
   up, every diagnostic log on the machine is pinned to its first generation.

The debug trace being on by default is fine (this is the pair/couch dev
machine); the defect is that the bounded-retention design is defeated.

## Spec

- Couch and the session clients it launches must be classified correctly by
  the retention proof: if they never write diagnostic files, they advertise
  `PAIR_RETENTION_PROTOCOL=1` (and their `PAIR_DATA_DIR` root); if any does
  write, it registers as a writer instead. Verify which before choosing —
  don't blanket-mark a process that actually writes (ARCH: root cause, not a
  wider exemption in `VerifyWriters`).
- A maintenance refusal must be observable: record it (once per cause, rate
  limited) where `pair doctor`/diagnostics surface it, instead of `_ =`.
- Keep the fail-closed safety property for genuinely unknown writers.

## Done when

- A test with a fake `Inspection` reporting a Couch process and
  `--couch-session-v1` clients (as launched by current code) lets rotation
  proceed for a file over `MaxGenerationBytes`.
- A test proves a maintenance refusal is surfaced, not discarded.
- Live check: with Couch running, a `wrap-events-*.jsonl` past 64 MB rotates
  (state.json `current.start` advances; old generation lands in the
  `.pair-diagnostics` store and is GC-collectable).

## Plan

- [ ] Audit whether couch / `--couch-session-v1` clients ever open diagnostic files
- [ ] Propagate the retention env (or register) from couch's launch path
- [ ] Surface `Maintain` errors
- [ ] Tests + live check above

## Log

### 2026-10-01

- Found while chasing high CPU on `couch-6b111ea230c149dc` (brain codex
  thread). Sibling issue #377 covers `session-watch`'s 120% CPU from the same
  session.
