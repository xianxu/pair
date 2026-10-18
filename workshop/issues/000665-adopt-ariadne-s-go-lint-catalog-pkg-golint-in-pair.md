---
id: 000665
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a84812deb32f791df25f6e33a25ccf76380dd846' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T05:39:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

Durable plan: [Go lint catalog](../plans/000665-go-lint-catalog-plan.md).

- [ ] Adopt a pinned upstream catalog through the existing module dependency, add its repository integration test, resolve all findings, qualify dependency consumers, document, review and land.

## Log

### 2026-10-18

### 2026-10-18 — Claimed and initial design

Claim and start-plan succeeded in pair:1 after the hub corrected the #636 dispatch; no changes/native runs were made for #636. start-plan refreshed the ariadne layer and required one clean re-invocation before branching. Peer MEMORY.md is absent; AGENTS.local.md was read. Chosen proposal: consume upstream golint through the existing module pin, avoiding duplicated catalog logic and a new protected weave export (ARCH-DRY). Read-only baseline found 181 findings in91files over2547tracked Go files; the durable plan records every affected path and forbids blanket migration suppressions. Module advancement also changes the existing runtime layergraph dependency, so compatibility qualification is explicit. Await hub approval before implementation; no code has changed.
