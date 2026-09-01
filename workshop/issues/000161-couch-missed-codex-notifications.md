---
id: 000161
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T13:35:54-07:00
---

# Couch misses Codex completion notifications

## Problem

During Couch dogfood testing, Codex completed work in Pair but Couch did not
surface a completion notification. No notification appeared in either the
status bar or the switcher.

The same joint failure can occur for Claude. One confirmed marker is:

```text
✻ Sautéed for 34s · done 1:39 PM
```

Claude marker mode currently requires the verb after `✻` to match ASCII-only
`[A-Za-z]+`. The accented `é` therefore prevents this otherwise valid marker
from reaching the shared notification sink. Codex remains a separate failure
mode because it currently relies on native OSC notification output.

A captured Codex completion renders as `─ Worked for 3m 57s ─…`. The same
capture also showed a copy prefixed by `> `. Investigation must distinguish the
live colored status line from quoted or replayed copies before granting either
form notification authority.

## Spec

### First slice: Claude Unicode marker

- In `endOfTurnByAgent["claude"]`, accept a marker verb made of one or more
  Unicode code points whose General Category is Letter (`L`). Precomposed
  `Sautéed` therefore matches; combining marks (`M`), digits, punctuation,
  whitespace, and an empty verb do not. Do not add Unicode normalization.
- Preserve the existing marker prefix and duration grammar. Optional trailing
  status/time text remains outside the parsed prefix.
- Preserve the current ownership boundary: only a newly finalized colored span
  can authorize this signal through `finalizeSpan -> emitOuter`. Quoted history,
  uncolored text, duplicate-span suppression, rate limiting, and delivery remain
  unchanged (`ARCH-DRY`, `ARCH-PURE`).

### Deferred: Codex completion

- Surface Codex completion notifications for Pair sessions in Couch's status
  bar and switcher.
- Keep the two surfaces consistent so a completed agent is not silently missed.
- This first slice does not modify Codex behavior or pre-decide whether the
  durable source is native OSC, transcript state, or a composed hook.

## Done when

### First-slice acceptance

- A production-path regression proves
  `✻ Sautéed for 34s · done 1:39 PM` reaches Claude's existing notification
  sink through `finalizeSpan -> emitOuter`.
- Existing ASCII markers retain behavior.
- Focused grammar tests prove an empty verb and any verb containing a non-`L`
  code point do not match.

### Later issue acceptance

- A Codex completion in Pair produces a visible notification in the Couch
  status bar.
- The same completion is visible in the Couch switcher.
- Automated coverage reproduces the missed Codex path and prevents regression.

## Plan

- [ ] Correct and regression-test Claude's Unicode marker-verb grammar first.

Deferred after the first slice:

- [ ] Reproduce and identify where the Codex completion event is lost.
- [ ] Add a failing test for the observed Pair/Codex completion path.
- [ ] Restore notification propagation to the status bar and switcher.
- [ ] Verify both surfaces from completion through display.

## Log

### 2026-09-01

Captured from initial Couch dogfood testing. Codex finished work in Pair, but
no notification appeared in the status bar or switcher. Investigation was
explicitly deferred to issue implementation.

Claimed the issue and entered planning. Inspection showed Claude uses a
finalized colored-span marker while Codex relies on native OSC. Captured
`✻ Sautéed for 34s · done 1:39 PM` as a concrete Claude failure: the ASCII-only
verb class rejects `é`. The operator approved a first slice that widens only
the verb to Unicode letters and leaves Codex discovery open.

Reviewed sibling `cmux` as prior art. cmux accepts OSC 9/99/777 and documents
Codex's `notify` command as one completion source, but its more robust agent
integration uses a PATH wrapper it controls plus PID-anchored transcript state
and optional injected hooks. Its design explicitly notes Codex has one legacy
`notify` slot that may already belong to another integration, so a later Pair
design must not silently overwrite it; viable alternatives are transcript
state, chaining the existing handler, or verified multi-hook injection.

Fresh-context spec review found that the Claude-first slice was not independently
finishable because Codex criteria still governed `Done when`, and that "Unicode
letters" and malformed input were underspecified. Split immediate and deferred
acceptance, named the pure grammar and production delivery seams, and defined the
verb as General Category `L` code points only.

Captured a concrete Codex stopped-state rendering: `─ Worked for 3m 57s ─…`,
with another observed copy prefixed by `> `. No current Codex text marker exists
in `pair-wrap`; Codex still takes the native OSC path. This evidence narrows the
later reproduction, but does not yet establish that rendered text is a stable or
authoritative completion protocol.

## Revisions

### 2026-09-01 — add confirmed Claude failure mode

Expanded the issue after dogfood evidence showed the two missing notification
surfaces are not Codex-only. Added a bounded first slice for Unicode Claude
marker verbs without changing the still-open Codex completion design.

### 2026-09-01 — separate first-slice acceptance after review

Separated the Claude patch from deferred Codex acceptance and plan ordering.
Specified the exact Unicode category boundary and the pure-parser and
production-delivery seams the regression tests must exercise.

### 2026-09-01 — add captured Codex stopped-state line

Recorded the observed `Worked for` rendering and its `> `-prefixed duplicate so
later Codex design tests the actual terminal stream and does not mistake quoted
or replayed text for a live completion.
