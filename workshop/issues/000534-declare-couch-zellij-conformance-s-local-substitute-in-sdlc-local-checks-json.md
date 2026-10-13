---
id: 000534
status: open
deps: [ariadne#345]
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'c6082b3a8384a000538bfa1e77c7a00f94f527cd' # card fields mirrored from issue-cards; edit via sdlc
---

# declare couch-zellij-conformance's local substitute in .sdlc/local-checks.json

## Problem

ariadne#345 (landed 10-13) lets `sdlc merge` accept a recorded local run in place of a CI job that never got a runner, and treats recorded local runs as first-class evidence. pair hasn't declared its mapping yet, so pair#523-style waits on GitHub's macOS runners still block landings.

## Spec

Declare in pair's `.sdlc/local-checks.json`: the CI job `couch-zellij-conformance / conformance` is substituted by `make -f Makefile.local test-couch-zellij-live test-native-terminal-ci` (argv form, a sensible timeout). Add AGENTS.local and README lines: this target substitutes for the macOS job when no runner is available, and is worth running before close on changes to Couch's session/Zellij code. Touches `.sdlc/*`, so the PR needs the operator's `local-gates-verified` label (local-gates-guard). Pair slots need the new sdlc build (weave refresh) to use it.

## Done when

- The declaration validates under sdlc; a merge whose macOS job had no runner passes on a recorded local run; docs say when to run it; the operator labels the PR.

## Plan

- [ ]

## Log

### 2026-10-13
