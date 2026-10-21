---
id: 000757
status: done
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 6.657
collaboration-mode: collaboration
card_mirror: '6b01bc5f8ee3591583c68cc1b3c24b93a86ac3cd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T02:15:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 5.43
---

# mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping)

## Problem

product-shaping (ops) grows the real app out of its mock screen by screen. In development, unbuilt screens should
still be clickable as mocks; nothing half-mocked may ship.

## Spec

Framework-neutral mock-to-real development uses the existing Couch mock server as an optional front:
- `couch mock serve DIR --real http://localhost:3000` reads one `screens.json`, dispatches declared real screens to the already-running app, and serves declared mock screens from the selected version with a visible MOCK badge and catalogue-derived progress count. It supports ordinary app assets/APIs and development WebSocket traffic without requiring Go in the app.
- Screen state is explicit in the catalogue, not inferred from handler failures. Real application errors never substitute mocks. Full document navigation crosses real/mock boundaries; arbitrary client-only SPA transitions require app cooperation and are not transparently intercepted.
- Production is the app's own build. Missing screens render a plain `Not implemented yet` page with a way back; links to missing screens are visibly marked. Build succeeds and reports missing screens. No mock code, fixtures or MOCK badge ships.
- `couch mock check DIR` reports catalogue progress and unimplemented screen routes, returning success for incomplete implementation and errors only for invalid/unreadable input. It is a read-only command, not a build wrapper or universal bundle verifier.
- A Go worked sample demonstrates the convention with two real screens and one mock screen, production placeholders and fixture-shape contract tests. Other frameworks follow the same file and production convention without importing the sample.

The operator approved this framework-neutral revision and its four limits on 2026-10-21 at11:00 PDT (TL receipt8d8b6147-e2fb-4e7f-ae90-2d150697de6c). TL/operator priority changed at12:54 PDT: park #769 and commit/enter change-code for #757 now (receipt2fe48cdc-1a0b-4a7d-a692-e2c3852ba839). The approved design and accepted limits are in the revised plan.

## Done when

- Framework-neutral front serves two real screens and one declared mock screen; full-page clicks across both directions work, mock badge/count derive from one catalogue, real errors never fall back, and app assets/APIs/streaming/upgrades preserve their behavior.
- Existing pure mock mode, selection, confinement, caps and reload remain qualified. Catalogue edits publish routing/count/revision together; invalid state is visible, and owned request/upgrade work ends on front shutdown without stopping the external app.
- `couch mock check` deterministically reports incomplete screens without refusing an otherwise valid build, touching fixtures, writing files or launching processes/listeners.
- Go sample's production build succeeds with one missing screen, prints its report, serves a plain missing-screen page with a working way back and visibly marked links, and contains no mock/fixture/badge surface. After detail is implemented and its same catalogue row is real, report/placeholder disappear on rebuild.
- Real backend response contracts catch required-field/type/array shape drift against fixtures independently of sample backend data. Root verification includes the nested sample, actual host integration and bounded browser navigation qualification.
- Help/README/atlas explain the HTTP/SPA boundary, any-framework production responsibility and sample workflow; full-flow review and required suite/CI pass before landing.

## Plan

- [x] Deliver the operator-confirmed framework-neutral revision of workshop/plans/000757-mock-to-real-plan.md, including optional front, read-only report and Go worked sample.
- [x] Complete targeted implementation verification and prepare the single atomic review boundary.

Recorded full-suite pass, mandatory close review, final-head CI and landing remain workflow gates after this implementation checklist.

## Log

### 2026-10-20

### 2026-10-21 — design proposal
- 2026-10-21: closed — Review fixes: redirect and complete-endpoint behavioral RED then race GREEN; stateful compiler failure/validation/cancellation pins and real compiler conformance GREEN; mockserve/root mocktemplate race PASS2.832s/3.895s; buildsample race PASS2.882s; artifactpath/repolint PASS18.815s/13.998s; affected runner13 tests and inventory PASS. Pinned code-head cc545350b41e2a531dee9de2faef8e04cf1c195c host browser full navigation PASS2.479s; delivered-progress suppression negative control fails intended assertion, raw evidence in plan side folder. Replacement recorded suite runs on pushed final code; pass required before landing.; review verdict: SHIP

Claimed and started after #755 landed as PR507. Read the product-shaping design and surveyed integration constraints. The host is Go-internal and owns its watcher only in the standalone serve lifecycle; direct template import cannot reuse it. Proposed a self-contained nested Go module with a thin development HTTP fallback to the existing host, one screen list and guarded production build; root integration test explicitly runs the nested module. Proposed strict build refusal without partial release, prototype-based fixture contracts, and bounded/owned effects. Detailed alternatives, state table and costs are in the durable plan. Awaiting hub/operator design approval; no implementation started and no estimate derived.

- TL confirmed the plan is with the operator in Needs you (authenticated receipt9fd58461-e8aa-43b1-bf57-4780481fc7d9). TL recommends two-command development reuse and strict production refusal; suggested considering a make dev convenience command that starts both. Holding code until the relayed answer. Any two-process convenience runner must own cancellation and join both processes; it is not yet an approved implementation addition.

### 2026-10-21 — implementation checkpoint

Full-flow change-code PASSED: plan-quality CLEAN after PQ-1/PQ-2; estimate-quality INFO accepted derivation and noted broad uniform concern budgets. Proxy ownership budget covers pure ownership/admission, API budget covers real transport/stream/upgrade wiring, qualification covers active suite/CI/landing work (queue idle excluded), and real-API discovery covers ReverseProxy hijack/cancel behavior. Eight concerns fit standard Go/host seams but qualification remains the optimistic component; no estimate back-fit.

Implemented explicit catalogue/report, optional framework-neutral front, paired scan/route/count revisions, exact declared mock delivery, mock-only chrome, external-app ownership and finite proxy/upgrade capacity. Catalogue/route/shape tests observed RED then GREEN and bounded fuzz; stateful upstream tests cover real errors/methods/assets/redirects, private namespace denial and no request scans. Production Go example builds partial and complete private copies, checks named missing handlers, preserves old outputs and excludes unique mock/fixture sentinels. The app's fixture data remains independent. Root integration runs nested tests and actual host+sample and proves front stop leaves app alive.

Verification so far: mockserve race checks pass; root template+host checks pass; private browser passed3 consecutive runs plus a subsequent integrated run2.941s, and suppressed delivered-progress negative control fails the intended assertion. First browser run overlapped production edits and failed before diagnostics; no stable-head flake is claimed. Inventory passed after auditing six new readers; targeted artifactpath12.351s, repolint12.467s and root template5.040s pass. Python affected-runner13 tests pass with finite nested-module source-preflight grouping. Root sample executable uses suite-owned cache, not per-test builds. Final incomplete-upload and padded-port probes were RED and fixes are being verified. No full recorded suite or final close is claimed yet. Private GOCACHE remains on main SSD, for removal after completion; no local-check/merge runner is active in this slot.


### 2026-10-21 — final targeted verification

Incomplete-upload cancellation and zero-padded origin probes both failed before fixes and pass afterward (mockserve targeted race1.966s; public Couch mock CLI0.373s). Combined mockserve/root-template race passed5.023s/5.558s and vet passed. Final private browser plus lifecycle/port/page-work verification passed; previous integrated browser run2.941s. Counted page traversal is bounded at N/2N. Contract traversal counts128→256 rows: inference2043→4091 visits for1026→2050 nodes (<=2 per node), matching1026→2050 (exactly1 per node); ambiguity comparisons are counted, not hidden reflect scans. Shape fuzz also passed76087 executions after this guard. Private selection lock/staging files are now git-ignored in the copyable sample. No full recorded suite/close yet.

- Close preflight correctly refused the issue Plan's circular future close/land checkbox before any boundary review ran. Replaced it with the completed targeted-verification/preparation step and explicit remaining workflow gates; no passing suite/review/landing is claimed. Retained suite remains on unchanged code c34fc2c3; only issue documentation changed. TL granted3-hour priority (receipt044e3ddf-9cb2-443b-8ed9-f28a7f08b125); report landing immediately for revocation.

## Revisions

- 2026-10-21 09:06 PDT — Operator approved partial production instead of refusal (TL receipt9e25f576-8ab1-4752-a2c4-21084fb68b10): plain Not implemented yet page + way back, marked links, report from successful build, no mock/fixtures/badge in production. The prior strict refusal contract is superseded.
- 2026-10-21 09:07 PDT — Operator requires any web framework. TL proposed optional couch mock serve --real front, one screens.json, framework-owned production build and couch mock check, retaining Go as a sample (receiptd475db2c-2dd9-4cee-8bd4-a20901b900ce). Revised Spec/Done when above describe this proposed replacement; operator confirmation remains pending, so hold change-code. Preserve original Log as historical design evidence.
- 2026-10-21 09:07 PDT — TL requires scratch-only drafting while #769's retained merge runner occupies canonical pair-slot4; commit this revision on #757 after #769 lands. Reconsider with hub at10:30 if GitHub remains degraded (receipt658d29be-e8b8-4a98-bf42-1212130e50fb).

- 2026-10-21 11:00 PDT — Operator approved the framework-neutral draft and four limits (receipt8d8b6147-e2fb-4e7f-ae90-2d150697de6c). Strict production means no mock content ships; it does not mean build refusal. Real stack owns missing-screen page/popup behavior and its own production build; mock check reports. Commit and start only after #769 lands. This supersedes the pending approval noted above.

- 2026-10-21 12:54 PDT — TL/operator made #757 high priority and authorized parking #769 and starting this issue now, superseding the prior wait-for-landing sequence. PR524, reviews and passed suite preserved; its runner had already exited1 after train238 conflict hold, so no cancellation was needed. Private GOCACHE=$HOME/.cache/go-build-757 on main SSD for builds/tests, to be removed at completion. No Go cache on TestScratch.

- 2026-10-21: Full-flow plan review round1 requested PQ-1 exact missing-declared-mock delivery policy and PQ-2 named per-function test strategies. Added startup/watch/selection repair and request-time503 semantics without index/upstream fallback in hybrid mode, preserving standalone behavior; compressed active tasks into function/strategy table. Re-running plan-quality before code.


- 2026-10-21 close round1: REWORK, no SHIP. BR-1 redirect reference preservation and BR-3 endpoint identity now use a shared pure pinned endpoint policy; behavioral RED reproduced root/query/fragment/double-slash redirect loss and distinct-address same-port rejection, then targeted race GREEN. BR-2 active plan now classifies catalogue.Read as INTEGRATION and names app/buildsample. BR-4 private compiler executor plus stateful fake pins partial compile/validation failures, validate-before-publish, and cancellation join before cleanup; real-compiler conformance retained. New fake test reads repository sources only via existing audited copySample helper; explicit nested SOURCE_GUARDS includes all three tests, inventory passes (the focused file itself is not a detected direct reader, so recording a redundant certificate was refused).
- The outdated recorded suite run 1b3a56a50da17cd4b4c44cf15ab48f28 on c34fc2c3 was canceled through local-check after review required code fixes; it is neither pass nor failure evidence. Fix-head targeted race tests: mockserve PASS2.832s, root mocktemplate PASS3.895s, nested buildsample PASS2.882s; affected-runner13 tests PASS, reader inventory PASS. BR-5 final-head host browser pass and negative-control evidence still pending, followed by replacement recorded suite and re-close.

- BR-5 pinned host qualification on code head cc545350b41e2a531dee9de2faef8e04cf1c195c: WEBCAST_LIVE_CHROME=1 full TestDevelopmentFrontBrowser PASS2.479s (test2.27s). Delivered-progress suppression negative control EXIT1 at the intended real-to-mock authoritative visible progress assertion (test8.96s), after successful initial real-page navigation. Raw command/head/exit/output retained in workshop/plans/000757-mock-to-real-plan.md.side/browser-pass.log and browser-progress-negative-control.log. This distinguishes the review environment's early CDP EOF from product behavior. The negative control is expected failure evidence, not a recorded suite attempt.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is marked stale by estimate-source, so provisional. Full scope includes already-incurred spec/operator work and the framework-neutral pivot, implementation, native browser/HTTP qualification and one close/landing boundary; queue idle is excluded.

Eight bounded new concerns: catalogue, pure dispatch, proxy ownership, scan publication, mock chrome, fixture shape, sample production projection/build, and root/browser qualification. Standard library JSON/net/http/httputil and existing x/net/html/host ownership are available, so each greenfield primitive uses1.0 raw design ×0.5 library availability ×0.2 thorough-spec =0.10; impl0.8 ×0.4 =0.32. Upstream HTTP/WebSocket API wiring adds1.0 ×0.2 design and1.5 ×0.4 implementation, plus separate runtime API discovery0.6 ×0.4. Familiarity1.0: Go/host/test ownership known; protocol edge qualification remains explicitly budgeted. Issue authoring uses1.0 design without discount because it accounts for shaping already done. Scope pivot0.3/0.3raw, docs0.2 ×0.2/0.2raw, review0.2 ×0.2/0.5raw. Thorough plan design buffer15%; no cross-repo implementation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.0 impl=0.12
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.20 impl=0.60
item: real-api-discovery design=0 impl=0.24
item: scope-pivot design=0.30 impl=0.12
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 6.657
```
