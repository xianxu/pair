---
id: 000846
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
severity: low
started: 2026-10-23T15:07:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
---

# Couch: count-agnostic lifecycle drain diagnostic

## Problem

The drain diagnostic approved in #802 renders `Lifecycle: 1 slots' metadata unverified at restart drain` when one slot is unverified. The possessive plural reads as a typo at N=1. The TL approved a count-agnostic follow-up in the #802 landing report.
