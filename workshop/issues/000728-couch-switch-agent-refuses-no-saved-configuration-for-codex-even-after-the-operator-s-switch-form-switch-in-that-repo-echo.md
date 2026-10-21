---
id: 000728
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e31e46ec79b082813dff4e1f855b47e6d8578f38' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T23:31:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo)

## Problem

ops-echo:0 (10-20 ~09:20): `couch --switch-agent echo:2 --agent codex --confirm` refuses with "no saved configuration
for codex; the operator must configure it in the switch form first" (requests 8e69e8d8 and 0ad9d2c3), and still
refused after the operator switched echo:0 to Codex by hand in the switch form. So the form's switch doesn't save the
per-repo configuration the CLI reads (or saves it under a key the CLI doesn't look up). In the devinfra repos the
same CLI switch worked (ariadne:0, ariadne:5, parley.nvim:3, :4 on 10-20), so a saved codex configuration exists there.

## Spec

Find where the switch form records its agent configuration and where `--switch-agent` reads it; make a successful
form switch leave the configuration the CLI needs (same key, same repo scope), or have the CLI fall back to the
agent's last successful launch configuration in that repo. The refusal names exactly what's missing and where.

## Done when

- A form-driven switch in a repo without prior CLI config, followed by a CLI switch on another slot of that repo,
  succeeds (test against the stateful fake/store).
- echo:2 and echo:3 switch to Codex by CLI live.

## Plan

- [ ]

## Log

### 2026-10-20

## Log

- 10-20 10:05 (TL): same refusal in a devinfra repo: `couch --switch-agent ariadne:4 --agent codex --confirm` →
  "no saved configuration for codex; the operator must configure it in the switch form first", although the same
  command succeeded on ariadne:0 and ariadne:5 an hour earlier. So the saved configuration looks per slot (or per
  launch history), not per repo: a slot that never ran Codex has none.
- 10-20 14:15 (TL): third case: the operator's switch-form switch of pair:8 (Qoder → Codex) failed with
  "switch-agent: source changed or disconnected; retry switch" (launch-diagnostics failures.json, operation
  2e048ac2508c98354822c8fa342aa0e3). pair:8's Qoder session was a recording only (no live wrapper). The switch should
  handle a parked or disconnected source (start fresh on the new agent) instead of refusing.
- 10-20 21:15 (TL, operator report): there is no way to change a running Codex slot's startup parameters and restart
  it. Choosing the switch form with the **same** agent shows a parameters box, but the box **accepts no keyboard
  input**. Scope for this issue now: (1) the parameters box in the switch form takes input; (2) "restart with new
  parameters" for the same agent is a supported path (form and CLI: `couch --switch-agent repo:N --agent codex
  --agent-args '…' --confirm` on the current agent), saved as that agent's remembered arguments; (3) the earlier cases
  (no saved configuration; a parked or disconnected source). Operator-facing; the operator hits it now.
- 10-20 22:35 (TL): root cause of the operator's latest report ("switching to codex in ops didn't use the recorded
  parameters"): remembered arguments are keyed by **exact physical path**. The ops primary checkout's preference file
  (`threadstore/path-preferences/b96c0dde….json`, physical_path /Users/xianxu/workspace/ops) records codex
  `["--sandbox","danger-full-access"]`, but slot worktrees (e.g. /Users/xianxu/workspace/worktree/ops-slot1/ops) have
  **no preference file at all**, so a switch or fresh start there falls back to the agent's defaults. Fix to include
  in this issue: when a slot path has no remembered arguments for an agent, fall back to the same repository's
  (repo_identity) remembered arguments, then save them for the slot on a successful launch; show which source was used
  in the switch form.
- 10-20 22:45 (TL): the operator ran `pair codex -- --sandbox danger-full-access` in the ops-slot1 worktree and the
  switch still ignored it: Pair's remembered-arguments store and Couch's `threadstore/path-preferences/` are separate,
  and Couch never got an entry for that folder. The fix must give remembered arguments one source of truth shared by
  Pair and Couch (plus the repo-level fallback for slot folders above).

## Revisions

### 2026-10-20 23:36 — Authenticated ops:0 dispatch; four-report scope

Delivery f55d328f-fc86-4a9a-8310-da4caa858833 is authenticated submitted from the owning hub. #743 is parked pending ariadne#509; this workspace now owns #728 on its own main-based branch. In addition to the original reports, unify Pair's repository defaults and Couch's path preferences under one current-preference authority with repository fallback for slot folders. Fix the form field, support same-agent fresh restart and explicit `--agent-args` through the CLI, and admit parked/disconnected sources only when existing core classification proves vacancy. Unknown or surviving native ownership must still refuse. Operator-facing plan goes to hub before code.

Proposed design: [shared startup preferences plan](../plans/000728-shared-agent-startup-preferences-plan.md). Two real review boundaries: M1 shared preference authority/migration; M2 form, CLI and guarded switch/restart with live acceptance. Governing completion additionally requires explicit empty/nonempty argv parity, shared settings from one slot visible to another/Pair, no learning on failed launch or recovery/warm attach, safe refusal races and operator-approved live checks. The plan explicitly requests approval for same-agent fresh-conversation semantics and started-with-warning preference publication after local registration; neither is assumed approved yet. Original reports/Spec remain as historical shaping context.

- [ ] M1 — Shared preference authority and bounded compatibility migration, with all current-default consumers moved and historical invocation evidence preserved.
- [ ] M2 — Editable form, explicit CLI args and safe live/vacant same/different-agent flow; required checks and hub-selected live acceptance before landing.

### Investigation checkpoint

Read-only traces identify launcher/createflow.go skipping default persistence for every Couch-owned/fresh launch; threadstore.go advances only exact-path preferences; switchagent.go reads that path plus a separate primary-folder default. Thus form success cannot seed another slot. switchServiceGuard requires a connected wrapper before core SwitchableState, excluding already-supported proven vacancy. Same-agent preparation explicitly errors; form typing requires SwitchPrepared, coupling preview refusal to a dead field. CLI/protocol omit explicit argv intent. No implementation edits or live switches.

The proposed leaf launchprefs component avoids launcher/couchcore import cycles. Local registration already persists the exact executed profile, and both successful-registration sites can publish preferences afterward via the existing warning channels. A publication error must never enter launch failure handling, which would otherwise stop a successfully registered helper. Plan makes the post-registration crash/write-failure limit visible rather than inventing a cross-root journal. One advisory proposal audit is running before submission; mandatory plan-quality review remains change-code's gate after hub approval.
