---
id: 000846
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: 'c3f5c51dc28fc011a1f2bae56dcee040bb7890d2' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: count-agnostic lifecycle drain diagnostic

## Problem

The drain diagnostic approved in #802 renders `Lifecycle: 1 slots' metadata unverified at restart drain` when one slot is unverified. The possessive plural reads as a typo at N=1. The TL approved a count-agnostic follow-up in the #802 landing report.

## Spec

Change the one string in `cmd/internal/couchcmd/lifecycle_notices.go` to `Lifecycle: metadata unverified for %d slot(s) at restart drain`. `slot(s)` is the count-agnostic form: the requested `for N slots` still renders `for 1 slots`, so this keeps the requested structure and drops the agreement problem. Update the exact-match pin in `lifecycle_notices_test.go`. No other producer, cap or copy behavior changes.

## Done when

- The drain diagnostic never renders a plural that disagrees with its count.
- The exact-match test pins the new string; the couchcmd package is green; no other message is touched.

## Plan

- [ ] Change the format string and its exact pin.
- [ ] Focused and package tests for cmd/internal/couchcmd; go vet.
- [ ] sdlc close with the run as evidence; recorded suite and merge before the day's window closes.

## Log

### 2026-10-23
