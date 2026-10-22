---
id: 000803
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 1.7935
collaboration-mode: collaboration
card_mirror: 'c221394b6d2d817dc0cf4a56bdb1f3a44d9b8940' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T09:04:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

Approved boundary (2026-10-22): this issue covers every **Go** production query
consumer through an independent lower-level seam, with layout first, then clip,
focus, opener, title liveness and launcher probes. #761 false-miss handling and
per-attempt observation survive the extraction. Mutations never retry; raw
diagnostic probes remain excluded. Lua consumers are explicitly deferred to
#804, which will use a thin CLI adapter over this same seam. Full flow applies.

## Done when

- The layout consumer retries an empty successful reply within the existing bound, through the shared seam (test with
  a fake that returns empty once).
- The live conformance test passes without any test-only retry; pair#743's hosted run goes green.
- Every in-scope Go query consumer derives bounded read recovery from one
  lower-level owner without a launcher import cycle; the atlas names #804's
  remaining Lua exclusion and the raw-diagnostic exclusion explicitly.
- Counted 8/16-read tests, cancellation/collector-error cases, strict malformed
  refusal and no mutation replay pin the shared bound and authority contract.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`, Method A only. Calibration is tagged stale by SDLC;
hours are provisional. Familiar stack and existing collector/parser/fake libraries
cover the work; no greenfield library or external API is required. Design for
the extraction/module/docs uses the thorough-spec 0.2 discount; implementation
values already use the v3.1 40% scale. Native qualification uses the existing
installed Zellij and controlled fixture, with discovery at the upper bound.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.03 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 1.7935
```

## Plan

- [x] Claim, enter planning, and enumerate query consumers and existing retry authority.
- [x] Settle the Go/Lua boundary with the hub, then checkpoint the executable plan before change-code.
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

### 2026-10-22 — approved scope and implementation plan

Authenticated TL receipt a27b640f-2e13-4c62-b6d8-b2faf5b422e0 approves Go-only
scope, full flow, preserving false-miss and per-attempt evidence. Filed #804
unclaimed with dependency pair#803 for the Lua consumers and future thin CLI
adapter. The executable plan is
`workshop/plans/000803-shared-zellij-read-plan.md`. It extracts the existing
read/classification into independent zellijquery and keeps launcher compatibility
aliases. Original process/collector errors remain distinct from synthetic read
classification so the name probe keeps its exact diagnostic authority.

## Revisions

2026-10-22 — TL-approved scope refinement: "every query consumer" means every
Go production consumer in #803; #804 owns the Lua integration. Expanded to full
flow because moving the existing strict classifiers and all Go adapters exceeds
the quick-flow production-line shell. No mutation or raw diagnostic retry is
added, and no caller's strict confirmation is relaxed.
