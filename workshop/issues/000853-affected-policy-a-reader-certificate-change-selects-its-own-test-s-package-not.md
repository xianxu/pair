---
id: 000853
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '5379b5a4b0e7fcd7a7e861263b7550ae02b4bba3' # card fields mirrored from issue-cards; edit via sdlc
---

# Affected policy: a reader-certificate change selects its own test's package, not the full suite

## Problem

## Spec

**pair's affected-test policy falls back to the full suite whenever a reader certificate changes (found on the first
day of affected boarding, 10-23).** Several changes today ran a full suite under `--pre-close` only because they
touched a file under `tests/affected-readers/…` (the per-test reader certificates): pair#825 (its
`request_lifetime_test.go.json`), pair#679 and pair#739. Any test fix that changes what a test reads updates its
certificate, so most test changes get the full suite, which defeats the point of boarding on the affected check.

Fix: in pair's affected policy (`.sdlc/affected.json` and its selector), map a reader-certificate change to the
package (or test group) whose certificate it is, plus whatever that certificate's reads select, instead of selecting
everything. Keep the full-suite fallback for changes to the policy itself, the selector, or shared harness code.

## Done when

- A change to one test's reader certificate selects that test's package (and the reads it declares), not the full
  suite (test with a real certificate change, e.g. pair#825's).
- Changes to the policy, selector or shared harness still select the full suite (test).

## Plan

- [ ]

## Log

### 2026-10-23
