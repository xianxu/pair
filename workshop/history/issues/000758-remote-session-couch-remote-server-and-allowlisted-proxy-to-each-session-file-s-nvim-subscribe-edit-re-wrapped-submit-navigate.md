---
id: 000758
status: done
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-22
estimate_hours: 4.136
collaboration-mode: collaboration
card_mirror: '0bc07950c4d5fd495b281fc8e902dc2d7166a756' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T04:59:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.65
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
- 2026-10-22: closed — BR1/2/3 independently disposed by round2 with failing pre-fix overlays. BR4 now measures actual writer payloads atN8/16 over16 reopen rounds:648152B/144pubs and2415268B/272pubs, zero retry/no-op/reload bytes, bounded decimal-width plateau; race passed3.218s and catalog/reader inventory pass. Pushedbe457f2be2f3 recorded suite queued, required before landing. Review overlap per tl-worker; no-plan-check only final suite/review/landing workflow pending; project TL-owned.; review verdict: SHIP
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

### 2026-10-22 — private native death and buffer qualification

Three installed-nvim conformance tests pass with race detection (package2.689s). Two physical private buffers return their exact buffer/File tuples; invalid buffer is positive nil. The selected second editor stops through the fixture's own idempotent cancellation/join, native PID/birth evidence confirms death/replacement, and Sweep removes only its incarnation. The first live editor's retained record is unchanged, and its displaced headless candidate never resurrects. Fixture children retain explicit empty-base environment and poisoned parent identity assertions. No operator/native live session or editor-owned runtime path is touched. Reader certificate re-audited individually; inputs remain generated private fixtures and literal callback source.

### 2026-10-22 — final composition qualification and review overlap

Actual EndpointOwner/EndpointStore -> service.handle -> private framed Unix server/Call -> CLI now exercises register, exact no-write retry, lookup, second document, batch unregister, successful-null lookup, and publication failures before/after commit. Stateful fake tracks native reads/callbacks/publications/entropy; persisted revisions/current authority are read back. Normal0.420s and race1.564s pass. Added the shared-helper map to atlas/couch.md. The initial suite lease2607563130 on d7b4258c was canceled only through its exact owned local-check run before admission because this executable regression test changed the head; terminal joined, no suite result claimed.

Full review may overlap the replacement queued suite per tl-worker. The only unchecked issue Plan item is the final suite/review/landing workflow itself; close may use precise --no-plan-check to review completed implementation while that workflow is pending, and --no-project because TL owns the remote-session project. Neither waives recorded suite or merge validation: a final-head recorded pass and normal merge gates still precede landing. No full-suite success or producer landing is claimed here.

### 2026-10-22 — review round1 and bridge split

Whole-issue close round1 returned REWORK on pinned1ad3d3b8: BR-1 late negative lookup falsely returns successful absence despite a successor; BR-2 deterministic encoded-size refusal incorrectly poisons healthy authority; BR-3 notification fake cannot hold/fail/join delivery despite checked qualification. Verified against actual owner/store/tests; fixing each class with controlled RED regressions. Earlier native qualification did not cover held notification; that checklist claim is corrected to pending. Canceled only owned queued suite lease2373422974 through local-check before admission because executable fixes are required; terminal joined, no passing result claimed.

TL initially requested a small external bridge, then accepted the evidence that #408 never held a daemon and testcouch is Go/testing.TB-owned. Per TL5e43c052 filed unclaimed #801 (deps #758) for the Go shared owner/driver plus thin external wrapper, confirmed readiness/environment, EOF/TERM and exact Close/AssertCleaned. parley.nvim#405 may land M1 on fakes/native unit evidence; only real-producer acceptance waits for #801. No bridge implementation enters #758. #759 operator acceptance still takes dispatch priority.

### 2026-10-22 — BR-1/BR-2/BR-3 corrections verified

BR-1 RED reproduced successful-null from a late nil probe despite a successor. Nil/mismatch/dead/document-absent race cases now return stale while preserving the successor and avoiding a stale write; negative response and retirement share the authority lock.

BR-2 RED reproduced near8MiB candidate refusal reported uncertain and poisoning despite no writer call. Store now returns an internal attempted-writer outcome without duplicate encoding; only invoked-writer errors poison. Typed encoded-capacity refusal maps to capacity through a private prewriter tag. Regression also caught a writer returning the same public capacity sentinel: it remains uncertain first and unavailable after poison, never misclassified as candidate capacity. Continued lookup, unregister, live/dead Sweep and retained unknown bytes pass after no-write refusal. Prewriter validation refusal is unavailable without poison.

BR-3 stateful pending/current attachment fixture controls held, failed and canceled Superseded delivery. Durable publication and lookup select the successor before held notice completion; failed delivery preserves exact retry without another publication/notification; Close cancels and waits for held notification teardown. This required qualification tests, not a new production lifecycle. Native tuple/death qualification and held-owner-seam qualification are stated separately.

Combined endpoint/couchcmd focused race passed10.776s/1.945s; repolint13.979s, artifactpath14.322s and reader inventory pass. No recorded full-suite pass or close approval is claimed; replacement suite and full re-close follow this checkpoint.

### 2026-10-22 — review round2 byte qualification gap

Round2 independently verified BR-1/BR-2 regressions fail with the pre-fix owner overlay and pass at head, and disposed BR-1/BR-2/BR-3. No Critical remains. Verdict FIX-THEN-SHIP did not finalize because Important BR-4 is still open: the N8/16 tests counted writer calls and retained rows but did not measure actual publication bytes. The earlier byte-qualification claim was premature. Adding raw-payload byte accounting to the injected writer and explicit snapshot/cumulative/zero-write/plateau assertions; production mechanism unchanged. The bound remains O(M*S) total bytes for M snapshot publications over S retained pairs, not a false linear-byte claim. Canceled own queued lease1152919602 through exact local-check before admission; no suite pass claimed.

### 2026-10-22 — BR-4 measured writer-byte qualification

Added a mutex-protected ledger recording actual raw payload lengths at the injected writer boundary (attempted encoded payload bytes, not filesystem block/metadata writes). The N8/16 test now runs16 registration/release/reopen rounds, checks every snapshot and cumulative budget, exact publication counts, zero retry/no-op/reload bytes, one publication per batch and actual on-disk sizes after reopen. Upper growth bounds permit the declared O(M*S) snapshot rewrite cost rather than falsely requiring linear cumulative bytes; they do not reject cheaper encodings. Independent negative controls reject duplicate ledger publications and added per-attempt retained bytes.

Root race run PASS3.218s: N8 initialSnapshot4605B -> final4615B, initial registration sum21216B, total648152B/144publications; N16 initial9120B -> final9137B, initialsum78591B, total2415268B/272publications. Path lengths cause small measurement variation across fixtures. Bounded +10B/+17B growth is canonical AttachID/Revision decimal width, not retained history. Retry/no-op/reload effects are exactly zero. Agent owner race passed7.031s and catalog2.280s; final root reader inventory passes. Production code is unchanged from3a98841ea. Full review ledger and recorded suite remain pending.

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

### 2026-10-22 — review corrections preserve approved authority semantics

BR-1: negative lookup completion must atomically decide exact retirement and its response against current selected authority; changed successor returns stale, not successful-null. BR-2: deterministic candidate encoding/size refusal occurs before writer invocation, returns capacity for the size bound without poison; only attempted writes create publication uncertainty. BR-3: held/failed/shutdown-interrupted notifications and pending/current editor state qualify publish-before-notice, successor preservation and joined shutdown. These implement the existing generation, bounded-admission and notification lifetime contract; no new takeover/reset authority. External built-binary daemon conformance bridge is separately #801 per TL5e43c052.

### 2026-10-22 — publication-byte qualification correction

BR-4 keeps byte qualification pending until actual writer payload lengths, cumulative bytes at N8/16, retry/no-op zero bytes and same-pair reopen snapshot plateau are measured and asserted. Call counts alone do not satisfy the approved operating envelope. No new journal, durable format or production IO path is added.
