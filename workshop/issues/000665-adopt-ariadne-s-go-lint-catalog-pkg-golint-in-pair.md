---
id: 000665
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 7.61
collaboration-mode: collaboration
card_mirror: 'e43e70d350aabfb4392efbacce14d8a02d1a7e5a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T05:39:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

- [ ] M1 — Enforce the module catalog; resolve all78non-sleep findings; quarantine exactly103existing sleeps in one shrink-only tracked site list; qualify dependency consumers, review and land the guard.
- [ ] M2 — Remove sleep debt with explicit signals, prioritizing flake-history packages (#659/#662/#623/#485); review and land the completed sweep.

## Log

### 2026-10-18

### 2026-10-18 — Claimed and initial design

Claim and start-plan succeeded in pair:1 after the hub corrected the #636 dispatch; no changes/native runs were made for #636. start-plan refreshed the ariadne layer and required one clean re-invocation before branching. Peer MEMORY.md is absent; AGENTS.local.md was read. Chosen proposal: consume upstream golint through the existing module pin, avoiding duplicated catalog logic and a new protected weave export (ARCH-DRY). Read-only baseline found 181 findings in91files over2547tracked Go files; the durable plan records every affected path and forbids blanket migration suppressions. Module advancement also changes the existing runtime layergraph dependency, so compatibility qualification is explicit. Await hub approval before implementation; no code has changed.

## Revisions

### 2026-10-18 — R1: two approved milestones

Hub delivery2b583e46 approves module consumption and layergraph qualification, and splits enforcement/78non-sleep repairs (M1) from103sleep repairs (M2). M1's exact-site JSON allowlist is temporary debt that can only shrink against the main baseline; no inline blanket exemptions. Plan R1 records matching, ratchet regressions and retirement. Implementation may proceed through change-code. Whole issue remains incomplete until M2; milestone publication will use the supported SDLC path or be escalated if unavailable.

### 2026-10-18 — R2: plan-quality verification refinement

Plan reviewPQ-1 requested named validation functions with adversarial-input strategies. Added sleepSites/decodeDebt/checkDebt/checkShrink/scanTracked/debtBaseline and the repository gate, distinguishing pure transformations from real-Git integration tests. Supersedes the prose test-case matrix without changing the approved scope or ratchet mechanism. No implementation has begun.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; the source is flagged stale, so calibration is provisional. Existing upstream checker and Go/Git avoid new-library discovery. Design uses the thorough-spec discount, implementation uses the v3.1 40% factor, familiarity1.0 and15%design buffer. Scanner/debt is one greenfield test module; M1 sweep is six bounded fixture groups and M2 sleep repair is twenty fixture-family groups (not103independent features). The grouped cross-cutting rows below aggregate those primitive counts; no queue/full-suite idle time is included. Three review boundaries comprise two milestones and whole-issue close.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.15 impl=0.32
item: cross-cutting-refactor design=0.24 impl=1.20
item: cross-cutting-refactor design=0.80 impl=4.00
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.00 impl=0.60
design-buffer: 0.15
total: 7.61
```

### 2026-10-18 — Plan review cleared

Plan-quality round2 CLEAN disposed PQ-1; no open blocking findings. Estimate derived after that result. No implementation has begun; M1's catalog test and exact-site debt validation will start red.
