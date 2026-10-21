---
id: 000763
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cf2587baaae908f681098c1374cefe8d7d6a3397' # card fields mirrored from issue-cards; edit via sdlc
---

# artifactpath manifest: split the 2,000-line central inventory by area so parallel landings stop conflicting

## Problem

## Spec

`cmd/internal/artifactpath/manifest.go` is a single 1,969-line inventory that nearly every change touching an
artifact path must edit (28 main commits in two days). Parallel slots keep conflicting on it at landing:
- pair#731 (pair:6) needed a second integration that only unioned independent term/webcast reader entries;
- pair#755 (pair:4), 10-21: after both suites and both reviews passed, merge refused a reader-inventory conflict
  twice as main advanced during its queue wait (bc79cbae, then bade54de). Each resolution costs a fresh full suite
  plus a fix review (~40 min queue + 10 min suite).

Make the inventory conflict-free by construction, keeping its exhaustiveness check:
- Split the entries so each area owns its own file (e.g. `manifest_webcast.go`, `manifest_term.go`, …, or one
  file per family), each contributing to the same registry; the source-coverage test still requires every token use
  to be declared exactly once and fails on duplicates across files.
- Keep entries sorted within a file, and document in the package comment where a new entry goes.
- No behaviour change: the same families, allowances and counts; the coverage test passes unchanged in meaning.

## Done when

- No single file holds all entries; two changes adding entries for different areas touch different files.
- The coverage test proves the union is identical to today's manifest (a one-time equivalence test against the
  pre-split list, removable afterwards), and fails on a duplicate declared in two files.
- atlas notes where inventory entries live.

## Plan

- [ ]

## Log

### 2026-10-21

## Revisions
- 10-21 01:50 (TL ops:0): **wrong file named at filing.** pair:4 reports the actual conflicts are in
  `tests/affected-go-readers.json` (473 lines, 26 main commits in two days); `manifest.go` auto-merged both times.
  That file maps each dynamic-reader test file to a content hash, one line per file in a single JSON object, so any
  two branches touching neighbouring test files collide on adjacent lines, and every resolution costs a full suite
  plus a fix review. **Re-scoped:** make the reader audit conflict-free by construction: one small file per reader
  (e.g. `tests/affected-readers/<path>.sha256`) or another layout where independent changes never touch adjacent
  lines, with a regenerate target (`make affected-readers`) so resolving is mechanical. Same audit semantics: a
  changed reader still requires re-audit. Keep `manifest.go` as is unless it also shows real conflicts.
  Done when: two branches changing different reader tests merge without conflict (test it with git), the audit check
  still fails on a stale or missing hash, and atlas says where the entries live.
