---
id: 000758
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '78174d1dae810886cdd450ccb341e69fa15ac486' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T04:59:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Couch-owned editor endpoint registry and lifecycle

## Problem
The remote document adapter cannot discover the live nvim buffer: existing registration identifies the document and owner, but advertises no editor socket or incarnation. A per-file socket alone cannot distinguish retry, explicit desktop takeover, stale cleanup or PID reuse. parley.nvim#405 needs the producer contract first.

remote-session (ops project, operator-approved 10-20): the phone talks only to Couch; Couch serves the session views by
proxying to the nvim that has each file open (registered by parley, parley.nvim issue filed alongside).

## Spec
Couch-owned editor endpoint register/lookup/compare-unregister keyed by canonical namespace plus immutable document ID, with editor incarnation, monotone attachment attempt, socket/buffer, native process identity and binding generation. Last explicit desktop attach wins; headless is fallback only. Exact current retry is idempotent; superseded retry never reacquires, including after restart. Couch owns replacement notices and exact stale-binding cleanup; Parley owns private listener/runtime files. CLI/ack/callback contract confirmed with parley.nvim:1 before code.

This issue is the producer only, independently landable before parley.nvim#405. Original document adapter/security integration moved to #798 (deps #758, parley.nvim#405, #759). No public remote editor, generic RPC proxy, headless provisioning or phone acceptance inside #758. Detailed wire contract, states, bounds and tests: [plan](../plans/000758-editor-endpoints-plan.md).

## Done when
- Confirmed contract is implemented through real CLI/private IPC and a Couch-owned persistent registry.
- Stateful tests prove exact retries, supersession/restart, desktop/headless priority, rename, delayed notices/acks, batch compare-release and positive stale cleanup without erasing successors.
- Persisted unknown fields survive rewrites; all requests/storage/operations are bounded, uncertain evidence refuses, no Parley poller or automatic takeover.
- Private installed-nvim conformance verifies fixed callbacks and endpoint identity; ordinary outside-pane submit is unchanged and editor-owned paths are never unlinked by Couch.
- Atlas/README/reader inventories, recorded suite and normal whole-issue review pass; producer lands for #405/#798.

## Plan
- [ ] Pin final consumer contract and record approved producer-only split.
- [ ] Implement pure model, portable store and stateful RPC/process seams with failing tests first.
- [ ] Wire CLI/private IPC, Namespace projection, bounded lifetime and stale cleanup; qualify exact failure/order sequences and N8/16 cost bounds.
- [ ] Run private installed-nvim conformance, update atlas/README/inventories, record suite, close/review and land the producer.

## Log
### 2026-10-22 — claimed and planned producer prerequisite
Card checked unowned, claimed by pair:6. #759 pushed branch preserved; switched to resting main-slot6 before start-plan. Catch-up twice generated the same unrelated merge-check.yml toolchain grep fix; each exact workflow-only diff was retained in a named stash, not mixed into #758. Third start-plan succeeded from current main; no stacked branch. Current code remains untouched.

Read remote-session project/design and existing sessionsync namespace/registry/process seams. Consumer receipts fc5fd55f, 8b7dcc76 and 66ccd77d pin constant endpoint/superseded RPC callbacks, namespace projection, monotone AttachID, bounded batch exit, no polling/auto-takeover and established-channel fencing. Draft durable plan awaits final consolidated contract confirmation and hub approval. No change-code, estimate, native run or implementation yet.

### 2026-10-20

### 2026-10-22 — split approved, contract confirmed

2026-10-22: filed #798 successfully on origin, unclaimed; set #758 card title through sdlc to Couch-owned editor endpoint registry and lifecycle. Final consumer confirmation7d43957e recorded above, plan approved by TL79aac81f. Proceed to change-code after this durable checkpoint; no implementation or estimate yet.

## Revisions

### 2026-10-22 — producer-only whole issue

TL79aac81f splits the initial producer milestone into the entire #758 scope, so it can land before #405. Original allowlisted document/nvim adapter and integrated security/phone qualification are preserved in unclaimed #798, deps #758, parley.nvim#405 and #759. No milestone-only landing or stacked development. #759's operator iPhone check retains dispatch priority.

Final consolidated producer contract confirmed by parley.nvim:1 receipt7d43957e (fields, monotone decimal AttachID, codes/bounds/deadlines, namespace/private root ownership, nil exact retirement, notice-before-ack). Parley fences advertised attachment and listener only; established native RPC channels cannot be revoked by the plugin. Future adapter (#798) gates every operation by generation and retires its old channels; typed editor methods also reject retired tuples. Native crash-orphan directory policy is separately with TL; producer never deletes editor runtime paths.
