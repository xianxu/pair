---
id: 000893
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '26ffad8f8d1cba55a283070a497a06b1e0765457' # card fields mirrored from issue-cards; edit via sdlc
---

# Affected policy: audited reads on reader certificates, then narrow each input population

## Problem

## Spec

pair#853 declared reader certificates (`tests/affected-readers/<test>.json`) as sdlc sidecars (ariadne#598): a certificate
change selects its test's package, and an optional `reads` member lets a change to a declared input add that test's
package to the selection. No real certificate declares `reads` yet.

Today every non-Go input a Go reader uses is covered by a `full` glob (`nvim/**`, `zellij/**`, `scripts/**`, `tests/**`
…) or a group, and `reads` only adds packages: it never removes an input's own `full` rule. So populating `reads` changes
no selection until a `full` rule is narrowed for the inputs those readers declare. This issue does both, input population
by population: audit each reader of a population and record its `reads`, then narrow that population's `full` rule only
once every reader of it declares its reads. `non_go_full_fallback` readers are the population that matters; there are
589 certificates in total, so the audit is per reader and the narrowing is incremental.

## Done when

- For at least one input population (e.g. `zellij/**`), every reader that reads it declares audited `reads`, its `full`
  rule is narrowed, and a change to one of its files selects exactly those readers' packages (qualified on the real
  planner in `tests/affected-config-test.py`).
- A population with any undeclared reader keeps its `full` rule (test).
- `make affected-readers` can record `reads` without hand-editing JSON, keeping unknown members.

## Plan

- [ ]

## Log

