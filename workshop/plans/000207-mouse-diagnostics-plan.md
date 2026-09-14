# Mouse diagnostic coverage implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) for execution. Steps use checkbox syntax.

**Goal:** Make the next mouse-mode recurrence attributable without changing terminal behavior.
**Architecture:** Extend the existing opt-in mouse tracer at production writer boundaries. Scanner fields explicitly describe beliefs, never measured terminal state. Keep the original issue recovery acceptance open beyond this diagnostic milestone.
**Tech Stack:** Go, couchtty, hostty, existing fake host and child PTYs.

## Core concepts

| Name | Lives in | Status |
|---|---|---|
| mouseTracer | cmd/internal/couchtty/mousetrace.go | modified |
| Console | cmd/internal/couchtty/console.go | modified |

`mouseTracer` remains formatting plus the existing file sink. `Console` owns
state snapshots and terminal writes; no new mouse-state authority is introduced.
Integration seams are existing Host.Write and traceFile, exercised with fake
host bytes and temporary trace files. No external service is introduced.

## Approved diagnostic contract

Every mouse event includes quoted active actor id and thread repo/tag identity,
and panel/actor surface. Rename misleading host-before to scanner-before.
Live child-mode changes record scanner old/new and host write byte/error result.
Takeover records pre-reset scanner, reset-to-none, replay-derived scanner, body
size, target identity, and emitted/short/error write result, including empty
replays and panel takeovers. It records every takeover even when modes match.
Startup and cleanup mouse writes also carry outcome and current identity.
Assertions record the snapshot before the attempt and distinguish emitted,
deferred (unsafe framing/cursor-save), short-write and error, with byte counts.
Use the existing writeOwn gate without changing its policy; an internal result
return may be added while existing callers ignore it. Never feed Couch's own
assertions into hostScan or issue terminal queries as part of this work.

## Architecture and operating envelope

ARCH-DRY: reuse mouseTracer, formatMouseModes, writeOwn and existing fake hosts.
ARCH-PURE: formatting is pure; snapshots and writes remain the thin Console shell.
ARCH-PURPOSE: enumerate live stream, takeover reset/replay, paint, startup,
cleanup, deferred writes and IO failure; test actual producer paths.
ARCH-MOCK: host captures bytes and supports short/error writes; child PTYs and
trace files use temporary roots. No live session mutation or restart.
ARCH-CONSTRAINTS: diagnostics remain opt-in, one bounded metadata record per
existing assertion/change or takeover, never raw replay bodies or per-byte logs.
ARCH-SECURE: quote identities and errors to prevent newline injection; no prompt,
transcript or child output content in the log. Existing file permission applies.
ARCH-ORDER: snapshot before writes, report observed return value afterward.
Existing concurrent writer ordering is not repaired here (#251 owns output
serialization); trace records are observations, not proof of terminal-global
ordering. Deferred writes record zero emitted bytes and no queued byte promise;
a later repaint produces a new attempt. No new persistent state or workers.
ARCH-FUNERAL: use existing opt-in trace file lifetime/close owner and append
policy; operator-selected diagnostic logs are explicitly temporary and removed
by operator after capturing the recurrence. Added per-record cost is bounded
metadata (identity, modes, counts); no new artifact family.

## M1 — producer diagnostic coverage

- [x] Add failing tests in cmd/internal/couchtty/mousetrace_test.go entering
  writeChild, takeOverScreen, paintNow, Run startup and release cleanup. Cover
  two thread identities, panel/empty replay, unsafe-frame defer, host error and
  short write. Assert logged outcomes match actual host byte acceptance.
- [x] Run `go test ./cmd/internal/couchtty -run MouseTrace -count=1`; confirm
  failures name the missing producer events/fields before implementation.
- [x] Implement snapshot/outcome formatting and hooks in console.go and
  mousetrace.go with no policy or emitted-byte changes.
- [x] Run focused tests then `go test ./cmd/internal/couchtty ./cmd/internal/couchcmd`
  and race-focused diagnostics. Update atlas/couch.md diagnostic documentation
  (or its existing diagnostic owner located by COUCH_MOUSE_TRACE search).
- [x] Commit, log verification and run `sdlc milestone-close --issue 207 --milestone M1`.
  Do not close #207, merge, deploy or restart live Couch.

## M2 — later causal repair

The original issue's recovery acceptance remains pending a captured recurrence
and approved causal design. No behavioral repair is authorized by this plan.

## Revisions

### 2026-09-14 — PQ-1 function strategies replace test inventory

The following function-level strategies supersede the case inventory in M1.
- `mouseWriteResult.detail`: direct tests over byte-count/error/deferred inputs;
  assert that only full acceptance with nil error emits the success outcome.
- `Console.mouseTraceContextLocked`: direct tests over hostile identity strings;
  quote every free-form field so tab/newline inputs cannot forge trace records.
- `writeChild` and `takeOverScreen`: feed mode-bearing streams through actual
  producers using the stateful host; compare snapshots and recorded result with
  captured accepted bytes, including injected partial/error writes. A blocking
  host seam pauses output while active identity changes, proving attribution
  uses the pre-write snapshot rather than post-write active state.
- `writeOwn` and `paintNow`: drive the real parser's framing/cursor-save gate;
  cross its boundary and compare emitted/deferred outcomes with host capture.
- `Run` and `release`: enter actual lifecycle with fake input/host; compare
  startup and cleanup event outcomes with captured writes and trace close.
- `mouseTracer.record`: preserve existing opt-in/nil/file tests; free-form
  diagnostic fields use quoted bounded values (256 bytes before quoting), so
  one producer record remains under 4 KiB. Typical records are under 1 KiB.

Record frequency remains one event per existing assertion/transition plus each
startup, takeover and cleanup, with no raw body logging. The existing log is
append-only without automated rotation: at 10 diagnostic events/s the typical
added metadata is below 10 KiB/s; a 15-minute focused capture budgets below
9 MiB. Operator cleanup after capture remains explicit, and trace is disabled
outside requested diagnosis. No promise of an automatic file-size cap is made.

### 2026-09-14 — implementation details

Free-form diagnostic fields use a stricter 128-byte pre-quote limit (ellipsis
marks truncation); the previously specified 256-byte maximum remains satisfied.
`mouseWriteResult`, `mouseTraceQuote`, `Console.mouseTraceContextLocked` and
`Console.traceMouseClicks` live in mousetrace.go and are covered by producer and
formatter tests. The atlas owner is atlas/couch.md. Existing non-diagnostic
writeOwn callers ignore its new observation result. No emission policy changed.

| Pure entity | Lives in | Status |
|---|---|---|
| mouseWriteResult | cmd/internal/couchtty/mousetrace.go | new |

`mouseWriteResult.detail` formats accepted/requested counts and outcome. The
context formatter and assertion integration remain methods of Console.

### 2026-09-14 — M1 review: explicit concept classifications

The reviewer returned SHIP with no blocking findings and one Minor classification
omission. This consolidated table supersedes the earlier split concept tables;
no implementation or ownership changes are required.

| Name | Kind | Lives in | Status |
|---|---|---|---|
| mouseTracer | INTEGRATION | cmd/internal/couchtty/mousetrace.go | modified |
| Console | INTEGRATION | cmd/internal/couchtty/console.go | modified |
| mouseWriteResult | PURE | cmd/internal/couchtty/mousetrace.go | new |
