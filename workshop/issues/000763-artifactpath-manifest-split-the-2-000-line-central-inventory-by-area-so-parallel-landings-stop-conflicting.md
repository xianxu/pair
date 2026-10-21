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
