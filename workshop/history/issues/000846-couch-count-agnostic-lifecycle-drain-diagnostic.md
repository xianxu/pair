---
id: 000846
status: done
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: '6f72567c49e94f69af5bf1ac457295730c768126' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T15:07:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "bdeeb9f3", done: "6df9dd6c"}
actual_hours: 0.10
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

- [x] Change the format string and its exact pin.
- [x] Focused and package tests for cmd/internal/couchcmd; go vet.
- [x] sdlc close with the run as evidence; recorded suite and merge before the day's window closes.

## Log

### 2026-10-23
- 2026-10-23: closed — Focused drain pin green; full couchcmd package green (302.9s); go vet clean; recorded suite queued on 175209eef and running. One format string and its exact pin changed. No atlas: wording only, no new surface — the #802 drain passage already describes it semantically.; review verdict: SHIP

Filed and claimed at the TL's direction after #802 landed, to fit before the day's window closes. Kept the requested structure but used `slot(s)`: `for N slots` still renders `for 1 slots`, so the count-agnostic property needs the parenthesized plural. One producer line and its exact pin; no other message touched. Quick flow (inferred, no estimate).
