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

- [x] M1 — Vertical slice: shared Go emitter (`cmd/internal/adapt`), wire pair-wrap
      (Aspect 1 remap fired/bypass; Aspect 2 overlay fired + near-miss heuristic),
      `bin/pair` truncate-at-launch, atlas Telemetry-Signal lines for Aspects 1&2 +
      `## 3. Drift Telemetry` section, `pair-doctor` (doctor/ script + README), tests
      (emitter format + drift near-miss regression). Proves emit→file→doctor→fix loop.
- [ ] M2 — Horizontal fan-out: shell helper `bin/lib/adapt-log.sh`; Aspect 3
      (session-watch + resume), Aspect 4 (pair-slug), Aspect 5 (PTY filter),
      Aspect 7 (nvim prompt-search); atlas Telemetry-Signal lines for 3,4,5,7;
      extend doctor registry coverage; tests per emitter.
      Carried from M1 review (all minor, non-blocking): (a) **cross-emitter golden
      test** — one Go + one shell + one Lua line asserted byte-identical in field
      order, so the "same schema, three languages" contract can't silently drift;
      (b) a concurrent-writer test asserting multi-process `O_APPEND` stays
      line-atomic before the shell/Lua emitters land; (c) rename the byte
      `truncate([]byte,int)` in pair-wrap (collides with `adapt.truncate`) →
      e.g. `capBytes`; (d) add an `atlas/index.md` "see also" pointer to
      `doctor/README.md`. M2 emitters must reference the atlas outcome enum, not
      re-invent the strings.

## Log

### 2026-06-03
- 2026-06-03: closed M1 — make test green (go ./... + lua + queue + doctor); -race clean on cmd/internal/adapt + cmd/pair-wrap (pair-scrollback-render race pre-existing/excluded). doctor.sh verified end-to-end against synthetic logs: tallies + deduped near-miss findings + NO-DATA + malformed-line tolerance (doctor_test.sh). Fresh-eyes review done; C1 panic + I1 robustness fixed with regression tests.; review verdict: SHIP

Filed from a design discussion. User chose passive logging (idea 2) + atlas-as-registry
(idea 3, already largely done); deferred active probes. The differentiator vs the
user's initial framing: **near-miss logging** — without it a success-only log can't
detect drift because breakage manifests as silence. Plan approved:
`~/.claude/plans/tidy-stargazing-music.md`. Two milestones (M1 vertical slice proves
the loop; M2 fans out across components/languages).

**M1 landed.** Shared `cmd/internal/adapt` emitter; pair-wrap aspects 1 (return-remap
fired/bypass) + 2 (overlay-detect fired + near-miss via `promptShape`); `bin/pair`
truncate-at-launch + quit cleanup; `doctor/{doctor.sh,README.md}`; atlas §3 registry.
Decision: pair-doctor ships as a plain script+README under `doctor/` in the pair repo
(NOT a construct skill — `construct/local` symlinks into ariadne's base layer and
propagates to all downstream repos, wrong home for a pair-specific tool).

Fresh-eyes review (BASE 4d0d32d → HEAD): verdict **changes-requested**, all fixed
before close. C1 — `promptShape` panicked because `strings.ToLower` isn't
length-preserving (offset past `len(visible)`); fixed with length-preserving
`asciiFold` + clamped `snippetLine`, regression test `TestPromptShapeMultibyteNoPanic`.
I1 — `doctor.sh` died on a single malformed JSONL line; fixed with tolerant
`jq -R 'fromjson? // empty'` + `|| true`, regression `doctor/doctor_test.sh`
(`make test-doctor`). Also: removed duplicate `dataDir()` (use `adapt.DataDir()`),
gated near-miss strip on live recorder, dropped the false-positive-prone
"do you want to" shape, added emitPlainCR/Open/Close/truncate coverage. Two lessons
recorded. `make test` green; `-race` clean on adapt + pair-wrap (the
pair-scrollback-render race is pre-existing, unrelated, excluded from `test-race`).
