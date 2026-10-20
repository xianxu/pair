---
id: 000734
status: wontfix
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '212d8c172ae6e67fdce25582afe5a79266de9a49' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T12:21:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Declare opt-in drain acceptance local check

## Problem

#722's approved re-plan needs a named opt-in disposable-Couch acceptance matrix recorded like a suite. The recorder reads declarations only from main. Adding that declaration changes protected local gates, so it belongs in a separate operator-verified configuration landing rather than #722's product implementation.

## Spec

Declare the optional local check drain-acceptance in .sdlc/local-checks.json, command ["make", "test-drain-acceptance"], timeout_minutes 15 and weight 1. Do not make it a globally required check or add it to ordinary make test. #722 owns the target, disposable real-Couch harness and its per-issue close/pre-stage requirement. Coordinate availability of that target: an absent harness must refuse, never record a vacuous pass. Until this declaration lands, TL authorized retaining #722's exact-head matrix argv/result/log/hash evidence manually in its Log.

This is a local-gates change. The operator verifies the configuration and applies local-gates-verified through TL; an agent must never self-label. Preserve all existing declared checks and their policies.

## Done when

- Main declares drain-acceptance with the agreed command, timeout and weight as an opt-in check.
- Ordinary make test and required checks do not start the seven installed-fleet cases.
- Recorder discovery accepts the declaration; when #722's target is available, a real matrix run records sdlc-local/drain-acceptance with exact head, exit, duration and log hash. An absent target cannot produce passing evidence.
- The operator verifies and labels the configuration PR before ordinary gated landing.

## Plan

- Coordinate the target/recording seam with #722, add only the optional declaration, verify discovery and existing required-check behavior, then close and land after the operator's label.

## Log


- 2026-10-20: abandoned (wontfix): Superseded by the operator-reviewed in-memory drain contract for pair#722 (TL4160a300): no queue/ack conformance declaration is needed; retain bounded installed-Couch qualification within #722.
Filed unclaimed from pair:5 under authenticated TL dispatch5c8b8b29. This is #722's dependency for named-check recording; it does not own the drain implementation. No configuration changed while filing.
