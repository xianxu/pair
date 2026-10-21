---
id: 000757
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '759cd5f5efe753f8ced70aa4eebf89058e9fe0c2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T02:15:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "90a93460", done: "d5f9337d"}
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

- [ ] Deliver the operator-confirmed framework-neutral revision of workshop/plans/000757-mock-to-real-plan.md, including optional front, read-only report and Go worked sample.
- [ ] Verify, close and land the single atomic issue.

## Log

### 2026-10-20

### 2026-10-21 — design proposal

Claimed and started after #755 landed as PR507. Read the product-shaping design and surveyed integration constraints. The host is Go-internal and owns its watcher only in the standalone serve lifecycle; direct template import cannot reuse it. Proposed a self-contained nested Go module with a thin development HTTP fallback to the existing host, one screen list and guarded production build; root integration test explicitly runs the nested module. Proposed strict build refusal without partial release, prototype-based fixture contracts, and bounded/owned effects. Detailed alternatives, state table and costs are in the durable plan. Awaiting hub/operator design approval; no implementation started and no estimate derived.

- TL confirmed the plan is with the operator in Needs you (authenticated receipt9fd58461-e8aa-43b1-bf57-4780481fc7d9). TL recommends two-command development reuse and strict production refusal; suggested considering a make dev convenience command that starts both. Holding code until the relayed answer. Any two-process convenience runner must own cancellation and join both processes; it is not yet an approved implementation addition.

## Revisions

- 2026-10-21 09:06 PDT — Operator approved partial production instead of refusal (TL receipt9e25f576-8ab1-4752-a2c4-21084fb68b10): plain Not implemented yet page + way back, marked links, report from successful build, no mock/fixtures/badge in production. The prior strict refusal contract is superseded.
- 2026-10-21 09:07 PDT — Operator requires any web framework. TL proposed optional couch mock serve --real front, one screens.json, framework-owned production build and couch mock check, retaining Go as a sample (receiptd475db2c-2dd9-4cee-8bd4-a20901b900ce). Revised Spec/Done when above describe this proposed replacement; operator confirmation remains pending, so hold change-code. Preserve original Log as historical design evidence.
- 2026-10-21 09:07 PDT — TL requires scratch-only drafting while #769's retained merge runner occupies canonical pair-slot4; commit this revision on #757 after #769 lands. Reconsider with hub at10:30 if GitHub remains degraded (receipt658d29be-e8b8-4a98-bf42-1212130e50fb).

- 2026-10-21 11:00 PDT — Operator approved the framework-neutral draft and four limits (receipt8d8b6147-e2fb-4e7f-ae90-2d150697de6c). Strict production means no mock content ships; it does not mean build refusal. Real stack owns missing-screen page/popup behavior and its own production build; mock check reports. Commit and start only after #769 lands. This supersedes the pending approval noted above.

- 2026-10-21 12:54 PDT — TL/operator made #757 high priority and authorized parking #769 and starting this issue now, superseding the prior wait-for-landing sequence. PR524, reviews and passed suite preserved; its runner had already exited1 after train238 conflict hold, so no cancellation was needed. Private GOCACHE=$HOME/.cache/go-build-757 on main SSD for builds/tests, to be removed at completion. No Go cache on TestScratch.

- 2026-10-21: Full-flow plan review round1 requested PQ-1 exact missing-declared-mock delivery policy and PQ-2 named per-function test strategies. Added startup/watch/selection repair and request-time503 semantics without index/upstream fallback in hybrid mode, preserving standalone behavior; compressed active tasks into function/strategy table. Re-running plan-quality before code.
