---
id: 000665
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '42c592d0135be40ab072e997fc1ad993aefca01a' # card fields mirrored from issue-cards; edit via sdlc
---

# Adopt ariadne's Go lint catalog (pkg/golint) in pair

## Problem

ariadne#407 added pkg/golint, a stdlib-only Go lint catalog run as a repository test, so every landing enforces it. Its seed rules come from mistakes made in both repos: test-shrunk-deadline (ariadne#434), tmp-literal (#342), test-go-build (pair#602), test-tight-wallclock and test-fixed-sleep (the 10-15 flakes). pair has the same classes (pair#659, #662, #623 were wall-clock budgets) but no catalog.

## Spec

Adopt the catalog in pair: run it as a pair repository test over pair's tracked Go files, with the same suppression convention (`//lint:allow <rule> — <reason>`). Choose how pair gets the package (a module dependency on ariadne, or base-layer files via weave) and say why. Sweep current hits: fix the real ones, give the rest reasoned suppressions.

## Done when

- pair's suite runs the catalog over every tracked Go file and fails on a new finding.
- Every current hit is fixed or carries a reasoned suppression.
- The way pair consumes the package is documented in atlas.

## Plan

- [ ]

## Log

### 2026-10-18
