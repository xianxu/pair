---
id: 000531
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '27453f877a15aa5dc424b3c9370de632eb3eb6e6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T13:00:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# couch: switching an ariadne slot from Claude to Codex fails before Codex's launch ledger and reverts to Claude; keep the failed launch's stderr

## Problem

An Ariadne Claude-to-Codex switch failed before any Codex launch ledger entry. The operator saw only `target launch failed; the source is still parked ... await Pair registration ... context canceled`; the target Pair output and exit evidence were not retained. Later inspection could not distinguish a failed helper from an interrupted registration wait. The current scope is diagnostic preservation, not an unproven Ariadne-specific fix.

## Spec

Retain bounded, attempt-scoped evidence for failed Couch launches, including the final switch-agent error, so failure remains inspectable after rollback, a Couch restart, or a later successful attempt. Capture the target helper's stderr in the ordinary runner, or combined stdout/stderr in a PTY (label it accurately), plus whether and when its exit was actually observed. A canceled wait alone must never claim that the helper exited or that startup failed at a particular stage.

Use the shared launch path and existing runner/PTY capture seams (ARCH-DRY). Keep diagnostics separate from authoritative thread lifecycle records so rollback cannot discard the evidence and older thread decoders are unaffected. Proposed bounded namespace store: newest 16 failed attempts, output tail at most 16 KiB and error at most 4 KiB per attempt, with an encoded-store bound. Exact thread address, attempt identity, target agent, stage/time and observed exit accompany the output. No environment dump or unbounded transcript. Record write failures must be reported without replacing the original launch error.

Immediate failure text and `couch --show` should identify the attempt and show readable error/exit/output evidence. Escape sequences from child output must not execute in the operator's terminal. Successful starts do not erase a previous failure; the bounded history evicts oldest failures. The diagnostic recorder must not take focus, change launch admission or retry automatically.

Only targeted tests while machine load is at least 40. No full suite until load is below 40. No further live reproduction; no changes to ariadne:1-4, ariadne:5, or pair:0 for this investigation. The independent start-slot agent-choice bug is #529 and is out of scope.

## Done when

- A target that prints an error and exits before registration leaves durable evidence containing its exact attempt identity, bounded output and observed exit code, including the switch's final error.
- Timeout and cancellation while the helper has not observably exited retain that uncertainty; cleanup-caused exit is distinguished from an exit observed before cleanup.
- Evidence survives launch rollback and re-opening the diagnostic store, and `couch --show` can inspect it without launching or focusing a slot.
- Duplicate updates enrich one attempt; concurrent or late updates cannot replace another attempt's evidence; storage and rendering remain bounded and sanitize terminal controls.
- Tests exercise real short-lived PTY and ordinary subprocesses plus the existing stateful launch fake, including pre-spawn refusal, output-before-exit, canceled wait, success, persistence failure and truncation.

## Plan

- [ ] Approve the diagnostic design and durable implementation plan.
- [ ] Implement bounded capture, persistence and failure reporting through the shared launch seams.
- [ ] Verify targeted regressions and documentation; run the required full suite only below the operator's load threshold, then close and land through SDLC.

## Log

### 2026-10-13 — investigation

The switch form parses explicit argv, fingerprints the accepted preview, and passes an explicit FreshRequired launch profile through PAIR_COUCH_LAUNCH_PROFILE to Pair. Fresh launches bypass saved-config, repository-default and config-picker reads. Last successful preferences remain Claude when the replacement fails; that alone does not mean the supplied arguments were ignored.

Live Ariadne and Pair processes had identical PATH and PAIR_HOME. The supervisor resolved Pair from /Users/xianxu/workspace/pair/bin/pair and Codex from /opt/homebrew/bin/codex. Launch uses Go exec directly, so shell dev-alias functions and weave's build-only generated tool environment do not participate. Neither observed Couch process had PAIR_CODEX_ALT_SCREEN set.

Read-only inspection found seven agent-bearing numbered-Ariadne ledger rows, all Claude. This proves no Codex attempt reached AppendLedger, not that the Pair helper never ran. Archived source 1-ariadne-45 had successful Claude park at 2026-10-12T23:11:29.994936-07:00 and no retained target stderr. pair:2 reported a supervisor restart at 23:12:34 during that attempt. Cancellation is consistent with shutdown; the earlier wait remains unexplained. Today's attempt time was not captured; replacement Claude source48 started at 10:54.

Scratch tests used a copied real Pair binary (build revision 69a1e510f54ba4551df0d04015be4e408e54c0bb, modified=true), isolated HOME/data/assets and a git repo named ariadne. External zellij/agents were fakes. Both empty-history and existing-Claude/no-Codex-config cases reached the ledger. With exact operator argv `--sandbox danger-full-access`, existing Claude tag and exact Couch session-intent protocol, Pair exited 0 with empty stderr; final PAIR_AGENT_COMMAND was `codex --disable shell_snapshot --sandbox danger-full-access --no-alt-screen`. Both runtime flags survive explicit parameters. These tests establish the controlled pre-ledger path, not real-agent readiness. Raw investigation: /tmp/ariadne-switch-investigation.txt; scratch command capture: /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-switch-probe-mm8dnww1/exact-command.json.

The start/add-slot form separately has an uninitialized Agents inventory. Constructor-based diagnostic confirmed it; #529 was filed unclaimed with its own proposed fix. It is not the cause of the switch failure.

### 2026-10-13 — operator live non-reproduction

TL receipt c975c7b5-a1c6-4853-a408-f234fc4ac58a reports the operator created ariadne:5 and successfully switched Claude to Codex with exactly `--sandbox danger-full-access`. No agent-driven live test was performed. Earlier failures do not reproduce on demand; today's historical cause remains unknown.

## Revisions

### 2026-10-13 13:02 PDT — operator narrows scope after live success

Supersedes the initial two-part scope (diagnostics, then agent-driven live investigation in ariadne:5): deliver durable failed-launch output/exit and the switch's own error only. Record the operator's successful live attempt; no further live reproduction or speculative startup fix. Machine-load threshold for a full suite remains below 40.
