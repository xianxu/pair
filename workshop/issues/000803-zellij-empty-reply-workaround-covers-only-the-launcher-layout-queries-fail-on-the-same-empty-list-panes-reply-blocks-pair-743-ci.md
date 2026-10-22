---
id: 000803
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '86950eb171dfc54a50d405413b9aad047be1127f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T09:04:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Zellij empty-reply workaround covers only the launcher; layout queries fail on the same empty list-panes reply (blocks pair#743 CI)

## Problem

## Spec

pair#743's hosted conformance run 39759978845 failed `TestTerminalRecoveryZellijConformance`
(terminal_recovery_live_test.go:146): native `zellij action list-panes --json --command --state --geometry` returned
exit 0 with empty stdout and stderr after an earlier valid 4996-byte answer, and the strict parser refused it. pair:3
traced production: `layoutcmd.OSRuntime.execZellijContext` runs once and `confirmWorkbenchPanesContext` uses
`zellijreply.Confirm` (exactly two valid observations, no retry); `ParsePanes` rejects empty JSON. Only
`launcher.ZellijSource.runContext` has the bounded 3-attempt empty or crossed-reply workaround (pair#748), and the
atlas explicitly excludes other consumers. So the layout path fails in production on the same Zellij behaviour
(upstream zellij-org/zellij#5751 family; pair#775 evidence).

Fix: one shared read-compatibility seam (#748's bounded retry on an empty or crossed reply, unchanged bounds) used by
every Zellij query consumer, layout first, not a per-consumer copy and not a test-only retry. Update the atlas
exclusion.

## Done when

- The layout consumer retries an empty successful reply within the existing bound, through the shared seam (test with
  a fake that returns empty once).
- The live conformance test passes without any test-only retry; pair#743's hosted run goes green.

## Plan

- [x] Claim, enter planning, and enumerate query consumers and existing retry authority.
- [ ] Settle the Go/Lua boundary with the hub, then checkpoint the executable plan before change-code.
- [ ] Implement and qualify the approved shared seam, keeping strict confirmation and mutation behavior intact.
- [ ] Record verification, close review, and land through SDLC; #759 operator acceptance takes precedence.

## Log

### 2026-10-22

Claimed by pair:6 and entered start-plan. The catch-up rebuilt the sibling
Ariadne tools and generated an unrelated merge-check workflow refresh (draining
grep instead of quiet grep); inspected and stashed only that workflow as
`803 planning: preserve weave-generated workflow refresh`. No CI gate changes
belong to this issue.

The read-consumer sweep found launcher inventory, legacy listing and name-probe
queries; layout panes/current-tab; clipboard panes/clients; Couch shortcut focus;
title liveness; opener panes/dump-screen; wrapper peer-viewport dump-screen; and
failure-only version diagnostics. Launcher retries currently preserve per-attempt
observers, exact absence diagnostics, and #761 false-session-miss classification.
Existing `zellijexec.Run` owns joined private process groups but has no read retry.
`zellijreply.Confirm` must still require two independently valid agreeing reads;
malformed same-kind replies must never become retryable parse failures.

ARCH-PURPOSE shadow sweep also found production Lua consumers:
`nvim/workbench_route.lua`, `nvim/dom_transport.lua`, and `nvim/pair_poke.lua`
query panes through `zellij_observation.query` under an existing admission budget;
`nvim/review/restore_client.lua` queries panes and floating-pane visibility through
its asynchronous runner. A Go-only seam cannot cover these consumers. Raw native
reproduction/latency probes intentionally observe the unmodified binary and must
remain outside retry. The shared seam must not replay native mutations.

Sent the hub the boundary decision before code (delivery
bd248d7c-4d1e-4931-bd42-1252539377aa): recommend one lower-level Go read seam with a
thin CLI adapter for Lua, requiring a short full-flow plan; alternative is an
explicit Go-only scope plus a separate Lua follow-up. Awaiting that decision;
no implementation or new refusal mechanism has been added. #759's iPhone
acceptance retains priority.
