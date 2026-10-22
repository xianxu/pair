---
id: 000758
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-22
estimate_hours: 4.136
collaboration-mode: collaboration
card_mirror: '4d651715eb3071d5971c7e99348a29f8b91adaf8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T04:59:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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
- [x] Pin final consumer contract and record approved producer-only split.
- [x] Implement pure model, portable store and stateful RPC/process seams with failing tests first.
- [x] Wire CLI/private IPC, Namespace projection, bounded lifetime and stale cleanup; qualify exact failure/order sequences and N8/16 cost bounds.
- [ ] Run private installed-nvim conformance, update atlas/README/inventories, record suite, close/review and land the producer.

## Log
### 2026-10-22 — claimed and planned producer prerequisite
Card checked unowned, claimed by pair:6. #759 pushed branch preserved; switched to resting main-slot6 before start-plan. Catch-up twice generated the same unrelated merge-check.yml toolchain grep fix; each exact workflow-only diff was retained in a named stash, not mixed into #758. Third start-plan succeeded from current main; no stacked branch. Current code remains untouched.

Read remote-session project/design and existing sessionsync namespace/registry/process seams. Consumer receipts fc5fd55f, 8b7dcc76 and 66ccd77d pin constant endpoint/superseded RPC callbacks, namespace projection, monotone AttachID, bounded batch exit, no polling/auto-takeover and established-channel fencing. Draft durable plan awaits final consolidated contract confirmation and hub approval. No change-code, estimate, native run or implementation yet.

### 2026-10-20

### 2026-10-22 — split approved, contract confirmed

2026-10-22: filed #798 successfully on origin, unclaimed; set #758 card title through sdlc to Couch-owned editor endpoint registry and lifecycle. Final consumer confirmation7d43957e recorded above, plan approved by TL79aac81f. Proceed to change-code after this durable checkpoint; no implementation or estimate yet.

### 2026-10-22 — plan-quality round1 requires evidence-preserving process seam

Initial unpinned change-code inferred quick from the short design despite this producer exceeding the implementation shell; reran with full flow for the approved durable plan, no code. Plan-quality round1 blocked PQ-1/PQ-2: Source.Identity drops native errors, and tests need named risky functions/strategies. Plan now specifies additive Source.ReadIdentity with original errors plus a narrow pure alive/replaced/dead/unknown classifier; existing consumers unchanged, no host-table scans. Named strategies use generated authority invariants, fuzzed strict/unknown-preserving codecs and controlled IO ordering/counts. No estimate or implementation before these clear.

### 2026-10-22 — producer implementation checkpoint and approved codec prerequisite

Implemented pure authority/high-water transitions, retained-raw endpoint store, error-preserving process evidence, installed-nvim fixed callback client, bounded owner admission/publication, CLI/private service and additive document Namespace projection. Stateful ordering/count tests and two private installed nvim editors exercise replacement, nil retirement, stale unregister and headless priority. No real Parley/remote adapter acceptance is claimed. Targeted packages passed (editorendpoint 0.692s, couchcmd 54.057s, procutil 0.414s, sessionsync 2.172s); focused race passed; artifact inventory, reader inventory and lint passed after individual source classifications. Full recorded suite and boundary review remain pending.

The installed nvim client replaces the initially proposed Go RPC library, per TLe0091eff: fixed Lua and JSON data are separated through luaeval/json_decode, literal process argv, bounded output and owned timeout/group cleanup. A 1s operation budget can additionally incur the existing 2s cleanup grace. No custom MessagePack codec or added dependency. Same-process rejected headless attempts retain an older selected desktop through validated Current metadata, with CurrentBinding as the sole authority projection.

TL6bf10860 approves the additive preserve-number duplicate checker plus SplitObject and EncodeObject entry points, only for new retained-raw endpoint state; existing strict readers and equivalence tests stay unchanged. This prerequisite is pending at this checkpoint. #759 has the sibling checker: whichever issue lands second must consolidate to the single shared helper and delete its duplicate (ARCH-DRY). Both issue Logs will record the coordination.

Recovery screen unavailable on this fresh session; branch, plan and working diff are intact at e4d048845. Earlier test outcomes above are retained session evidence; new executable changes will receive fresh verification. No suite, review or merge runner is being claimed as surviving.

### 2026-10-22 — numeric preservation and real-service wire regressions

New regressions observed RED at retained unknown 1e9999 and at a real empty producer lookup through private Unix IPC/CLI (unexpected exit5). Added shared preserve-number structural/object entry points used only by endpoint retained state; known fields remain typed and existing strict/equivalence tests unchanged. Raw over/underflow exponents, exact fractions, negative zero and 400-digit integers survive all four object levels plus actual store reopen. Focused tests/race pass; codec fuzz passed123,288 executions. Shared checker decode.go is identical to #759's current implementation; its Log is updated and PR538 pushed at ec201f3b5.

Corrected owner-to-wire outcomes: positive absence is ok/null, positive missing registration is not-registered, attempted publication failure is uncertain, and later poisoned-owner calls are unavailable. Controlled before/after write variants and actual private service empty/damaged-store checks defend this boundary. Combined focused race passed editorendpoint4.447s, strictjson1.079s, procutil1.167s, couchcmd1.921s. Strictjson package passed0.204s; artifactpath20.002s; repolint17.488s. An initial guard command named a nonexistent cmd/internal/repolint package; corrected to internal/repolint, no code workaround. Native reader re-audited after outcome assertion correction; no new repository inputs.

Private native process-death collection qualification is being completed before the full recorded suite. No suite/review/landing pass is claimed yet.

## Revisions

### 2026-10-22 — producer-only whole issue

TL79aac81f splits the initial producer milestone into the entire #758 scope, so it can land before #405. Original allowlisted document/nvim adapter and integrated security/phone qualification are preserved in unclaimed #798, deps #758, parley.nvim#405 and #759. No milestone-only landing or stacked development. #759's operator iPhone check retains dispatch priority.

Final consolidated producer contract confirmed by parley.nvim:1 receipt7d43957e (fields, monotone decimal AttachID, codes/bounds/deadlines, namespace/private root ownership, nil exact retirement, notice-before-ack). Parley fences advertised attachment and listener only; established native RPC channels cannot be revoked by the plugin. Future adapter (#798) gates every operation by generation and retires its old channels; typed editor methods also reject retired tuples. Native crash-orphan directory policy is separately with TL; producer never deletes editor runtime paths.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration currently marked stale by estimate-source; use provisionally rather than invent another model.

Decomposition in order: initial contract/spec coordination; pure endpoint transition model; retained-raw codec/store extension; owner admission/publication lifecycle; fixed Neovim API integration and stateful fake; existing CLI/IPC/namespace extension; cleanup/lifetime qualification; one native API conformance discovery; one whole-issue boundary review; atlas/README/inventory. No adapter/phone feature budget (#798).

Initial issue/spec design0.6h was not pre-resolved at claim. Other component designs use the thorough-spec x0.2 discount: model and owner each1.0->0.2; codec/CLI0.3->0.06; cleanup0.5->0.1; docs0.1->0.02. API integration design2.0 uses the library-availability half and x0.2 spec discount ->0.2: official github.com/neovim/go-client/nvim provides typed client/ExecLua, avoiding a custom MessagePack implementation; durability/process/IPC reuse existing seams. Implementation picks within the v2 table, scaled to40% per v3.1 (for example API1.5->0.6, model0.8->0.32). Familiarity1.0: existing Go/native-lifecycle patterns, one bounded API already separated and budgeted for discovery. One review is used although the closed primitive is named milestone-review. No vendor-mode cross-repo code propagation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.6 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.2 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.1 impl=0.24
item: real-api-discovery design=0 impl=0.24
item: milestone-review design=0 impl=0.2
item: atlas-docs design=0.02 impl=0.08
design-buffer: 0.15
total: 4.136
```

Design subtotal 1.44h x1.15 plus implementation subtotal 2.48h = 4.136h. This is focused ship-time, not a deadline; suite queue time is excluded by the actuals system.

### 2026-10-22 — additive codec prerequisite and transport refinement

Per TL6bf10860, #758 owns additive preserve-number structural validation/object splitting/encoding for endpoint retained raw fields only. Strict APIs retain historical overflow refusal. #759 and #758 reconcile to one shared helper when the second lands. The initial Go RPC library option is superseded by the bounded installed-nvim client approved in TLe0091eff; semantic callbacks stay fixed and typed. No producer filesystem cleanup of editor-owned paths and no new adapter scope.
