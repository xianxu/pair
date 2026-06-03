---
id: 000045
status: working
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 5
---

# adaptation flight recorder and pair-doctor

## Problem

`pair` adapts each agent harness (claude/codex/agy) across 7 integration aspects
documented in `atlas/how-to-bring-up-a-new-harness-cli.md`. Harnesses update
constantly and silently break these adaptations — e.g. codex renames its picker
confirm string and the overlay detector goes quiet, leaking a stray newline into
the prompt (the #000042 class of bug). Nothing surfaces that drift today: the unit
tests freeze our *assumptions* about each harness and validate matchers against
frozen strings, so they pass forever even after the live harness moves.

## Spec

A **passive flight recorder**: the user runs `pair` normally, every adaptation
trigger leaves a structured trace, and when something feels off they run
`pair-doctor` (a script + README under `doctor/` in the pair repo) that reads the
trace and proposes fixes — without having to describe the symptom.

- **Sink:** one per-session append-only JSONL, `$PAIR_DATA_DIR/adapt-<tag>.jsonl`
  (fits the existing `<purpose>-<tag>` file convention). Flat schema, one line per
  event: `ts, comp, agent, aspect(1-7), signal, outcome(fired|bypass|near-miss|fail),
  detail(≤200 chars, local-only)`. `detail` is a single capped string (not nested)
  so shell/Lua emitters stay one-liners.
- **Always-on, multi-process:** each component appends with `O_APPEND|O_CREATE`;
  `bin/pair` truncates once at launch (owns per-session lifecycle) → no
  multi-writer truncate race, no rotation (events are low-frequency).
- **Key decision — log near-misses, not just successes.** A success-only log can't
  catch drift: when codex renames its picker, our detector goes silent and the
  *absence* is invisible. We must also record "harness emitted prompt-shaped output
  but no registered detector matched it" — that line hands the doctor the exact new
  string to match.
- **Registry = the atlas doc.** Each of the 7 aspects gets a `**Telemetry Signal:**`
  line; the doc the human edits IS what the doctor checks against.
- **Reader:** `pair-doctor` aggregates the JSONL on demand (no separate rollup file)
  → flags near-miss/fail → maps aspect→atlas section → proposes the concrete fix.

Scope: passive logging + atlas-as-registry. Active golden-scenario probes are
explicitly deferred. Full design: `~/.claude/plans/tidy-stargazing-music.md`.

## Done when

- A user running `pair codex` produces `adapt-<tag>.jsonl` lines for return-remap
  and overlay-detect during normal use.
- Mangling a codex overlay marker produces a `near-miss` line (regression the unit
  suite cannot currently catch), covered by a Go test.
- `doctor/doctor.sh` reads the trace, reports the near-miss, and (via doctor/README.md
  + atlas §3) maps it to the atlas aspect with a concrete fix.
- All 6 *runtime* aspects (1,2,3,4,5,7) emit signals from their owning component
  (Go/shell/Lua), all into the one shared-schema file; atlas documents each signal.
  Aspect 6 (agent settings) is static config with no runtime trigger — it has no
  emittable signal by nature, so the atlas notes it as "static, no telemetry".

## Plan

- [ ] M1 — Vertical slice: shared Go emitter (`cmd/internal/adapt`), wire pair-wrap
      (Aspect 1 remap fired/bypass; Aspect 2 overlay fired + near-miss heuristic),
      `bin/pair` truncate-at-launch, atlas Telemetry-Signal lines for Aspects 1&2 +
      `## 3. Drift Telemetry` section, `pair-doctor` (doctor/ script + README), tests
      (emitter format + drift near-miss regression). Proves emit→file→doctor→fix loop.
- [ ] M2 — Horizontal fan-out: shell helper `bin/lib/adapt-log.sh`; Aspect 3
      (session-watch + resume), Aspect 4 (pair-slug), Aspect 5 (PTY filter),
      Aspect 7 (nvim prompt-search); atlas Telemetry-Signal lines for 3,4,5,7;
      extend doctor registry coverage; tests per emitter.

## Log

### 2026-06-03

Filed from a design discussion. User chose passive logging (idea 2) + atlas-as-registry
(idea 3, already largely done); deferred active probes. The differentiator vs the
user's initial framing: **near-miss logging** — without it a success-only log can't
detect drift because breakage manifests as silence. Plan approved:
`~/.claude/plans/tidy-stargazing-music.md`. Two milestones (M1 vertical slice proves
the loop; M2 fans out across components/languages).
