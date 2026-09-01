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

## Spec

- Surface Codex completion notifications for Pair sessions in Couch's status
  bar and switcher.
- Keep the two surfaces consistent so a completed agent is not silently missed.
- As a first bounded slice, accept one or more Unicode letters in Claude's
  marker verb while preserving the existing marker prefix and duration grammar.
  Optional trailing status/time text remains outside the parsed prefix.
- Preserve the current ownership boundary: only a newly finalized colored span
  in Claude marker mode can authorize this signal. Quoted history, uncolored
  text, duplicate-span suppression, rate limiting, normalization, and delivery
  remain unchanged (`ARCH-DRY`, `ARCH-PURE`).
- Do not broaden the Claude verb to arbitrary non-space punctuation. This
  grammar correction does not modify Codex behavior or pre-decide a broader
  semantic turn-state detector.

## Done when

- A Codex completion in Pair produces a visible notification in the Couch
  status bar.
- The same completion is visible in the Couch switcher.
- Automated coverage reproduces the missed-notification path and prevents a
  regression.
- A production-path regression proves
  `✻ Sautéed for 34s · done 1:39 PM` reaches Claude's existing notification
  sink, while ASCII markers retain behavior and malformed spans gain no
  notification authority.

## Plan

- [ ] Reproduce and identify where the Codex completion event is lost.
- [ ] Add a failing test for the observed Pair/Codex completion path.
- [ ] Restore notification propagation to the status bar and switcher.
- [ ] Verify both surfaces from completion through display.
- [ ] Correct and regression-test Claude's Unicode marker-verb grammar first.

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

## Revisions

### 2026-09-01 — add confirmed Claude failure mode

Expanded the issue after dogfood evidence showed the two missing notification
surfaces are not Codex-only. Added a bounded first slice for Unicode Claude
marker verbs without changing the still-open Codex completion design.
