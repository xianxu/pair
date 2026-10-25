---
id: 000892
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '8b7b24f5c8ecfbe696b6cd97a3e53f4be8261b21' # card fields mirrored from issue-cards; edit via sdlc
---

# Adopt weave's regenerated merge-check.yml (SIGPIPE-safe pin check); new slots come up dirty without it

## Problem

## Spec

**Adopt weave's regenerated merge-check.yml in pair (local-gates change, operator-verified).** New pair slots (pair:10,
pair:11, 10-25) come up with `.github/workflows/merge-check.yml` modified by weave refresh: ariadne's newer seeded
template replaces the pin-verification `grep -Eq` lines with `grep -E … >/dev/null` ("Drain printf's full input: quiet
grep can SIGPIPE it under pipefail"). `sdlc start-plan` refuses the dirty tree, so each new slot must restore the file
by hand before it can start work, and pair's CI keeps the SIGPIPE-prone version.

Commit the regenerated workflow on its own branch so pair's committed copy matches weave's output again. It touches
the local gates, so the operator verifies the PR and applies `local-gates-verified`; agents never apply it.

## Done when

- pair's committed merge-check.yml equals weave refresh's output (a fresh slot comes up clean); CI green on the PR.
- The operator has verified and labelled the PR.

## Plan

- [ ]

## Log

### 2026-10-25
