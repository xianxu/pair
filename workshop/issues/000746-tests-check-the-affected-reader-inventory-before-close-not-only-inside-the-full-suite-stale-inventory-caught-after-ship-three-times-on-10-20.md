---
id: 000746
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a83e52cdd6504b21e0293f2d037a192a88e665c9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:55:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "becd7f63", done: "d732f560"}
---

# tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20)

## Problem

pair#725's affected-reader inventory (`tests/affected-go-readers.json`) is checked only inside the recorded full
suite, so slots discover a stale inventory after their close review is SHIP, at the most expensive point. On 10-20
it cost three items a suite slot each plus a re-close: pair#727 (sank train 190 for 1 h 45 m), pair#726 (one
fixture file) and pair#706 (eight test files).

## Spec

- A fast `make test-affected-inventory` (seconds) that runs only the reader-inventory guard.
- Run it where it's cheap: in the close gate's preflight (or the affected check's first step) and listed in
  tl-worker's "before close" steps, so a stale inventory is caught before review, not in the suite.
- The failure message names each file and the exact command that updates its disposition and hash.

## Done when

- The target exists and fails fast on a stale entry; the close preflight (or tl-worker's checklist) runs it; a test
  pins the message.

## Plan

- [x] Add fast check and explicit per-file record modes around the existing audit; pin stale diagnostics and repair with private tracked fixtures.
- [x] Add the Make target, affected-runner first step and local before-close instruction; verify fail-before-work ordering.
- [x] Run the fast guard and recorded affected check; close and landing are subsequent SDLC gates.

Design: ARCH-DRY reuses audit/tracked_readers/SOURCE_GUARDS without a new scanner.
The check scans tracked test sources once and starts no Go tools. A named record
operation retains other JSON fields and refuses untracked readers and invalid
classification; no mass update. Existing full guard tests and strict source-walk
negative controls remain. No new persistent file family or schema.

## Log

### 2026-10-20
- 2026-10-20: closed — BR-1 both historical non-scanner dispositions to required source-reader classification reproduced red; current source-scan requirement now overrides old disposition. Five private Git CLI tests pass2.015s including executing each emitted repair command, scanner/preflight refusals, removal and unknown/duplicate-key controls. make test-affected-inventory passed immediately before this close. Prior full suite on44c4e7ee passed13m21s; fresh recorded affected check on pushed a4f0b6ae is pending and must pass before landing, review overlaps per tl-worker. Twelve runner tests passed, exact inventory-first ordering retained. README/atlas/local before-close instructions updated; TL owns project, no staging for test tooling. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

### 2026-10-20 — implementation and red/green evidence

Fast-mode red: --check was rejected (exit2); affected ordering red: an injected
inventory failure was ignored and selection exited0. Shared check/record/remove
modes now reuse the existing audit, preserving source scanner enforcement and
unknown inventory fields; duplicate keys refuse. Per-file recording requires an
explicit disposition and refuses untracked readers or omitted fixed preflight.
Four private-Git CLI tests pass (2.294s); all twelve stateful runner tests pass
(29.760s), preserving exact selected argv with the new inventory-first call.
The real make test-affected-inventory passed in2.922s, with no Go/Make invocation
from check mode. README/atlas and Pair's generated-instruction source require
this target before every close, including narrow re-close. No close bypass or
schema change. Recorded affected verification follows on the pushed head.

### 2026-10-20 — recorded verification

Recorded affected check fell back to the full suite because the fresh graph
lacks generated runtime assets (the independently assigned #738). Attempt
3ce9ad3e006d17e6cfd338a8749fde5e on pushed44c4e7ee9f9606767dcf69f587118a9ee0057650
passed exit0 in13m21s after542s queue wait; log hash22f10d196178. All scripts,
editor checks and Go packages passed, including twelve runner tests and twelve
inventory tests with the seeded source-guard negative control. The existing
upstream affected-config qualification explicitly skipped for unavailable source;
no new skip or waiver. make test-affected-inventory passed again immediately
before close. The final checklist records pre-close verification; close and
landing remain the binary's subsequent gates, not a premature landing claim.

## Revisions

### 2026-10-20 — BR-1 current classification before historical disposition

The first close returned REWORK: both fixture-to-source-reader transitions
printed the historical disposition, which the record validator rightly refused.
Private Git regressions reproduced each red. Generated commands now prefer
required fixed_scanner classification over the historical disposition; each
regression executes the printed command and then passes the check, with all
reader tests already registered. No guard is relaxed. Five CLI tests pass;
make test-affected-inventory passes again. The prior recorded full-suite pass
covers the preceding head only; a new recorded check is required on the fix.
Review may overlap that queued check per tl-worker; landing still requires its
pass. This corrects the existing exact-command contract, not new scope.
