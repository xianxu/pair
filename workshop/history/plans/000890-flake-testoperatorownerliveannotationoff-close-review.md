# Boundary Review — pair#890 (whole-issue close)

| field | value |
|-------|-------|
| issue | 890 — flake: TestOperatorOwnerLiveAnnotationOff |
| repo | pair |
| issue file | workshop/issues/000890-flake-testoperatorownerliveannotationoff.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11fece5cfb3b7da9ab93e6e592ec4f940919887a..4b0b9f8e9d4d9b0c9d5c137bff879c5432319669 |
| command | sdlc close --issue 890 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T10:54:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The change appears consistent with the issue’s purpose: it waits for resize handling before sampling click geometry and preserves native input, permissions, and timeout. Static review found no concrete correctness defect. Browser verification is blocked because Chrome exits before either focused test reaches its assertions; the recorded implementation evidence cannot independently establish those results.

Done-when: current — the existing clauses cover the resize barrier, lifecycle controls, and native owner scenario, including the revised document-readiness ordering.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: executable-regression-evidence
    title: |
      Native resize and owner regressions could not execute in the review environment
    detail: |
      At cmd/internal/webcast/live_viewport_test.go:50 and operator_owner_live_test.go:48, WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^(TestLiveViewportResizeJoinsPageHandlers|TestOperatorOwnerLiveAnnotationOff)$' -count=3 -timeout=180s -v failed before assertions in all three iterations: Chrome reported sandbox initialization failure and “GPU process isn't usable.” These are both affected tests in this window. Provide measured host evidence at pinned head 4b0b9f8e9d4d9b0c9d5c137bff879c5432319669 showing both pass, plus the held-event regression failing with the metrics-only implementation, or provide an environment where the reviewer can execute those checks.
```

1. **Strengths**

   - The injected CDP helper centralizes receipt creation, waiting, and cleanup; cleanup errors are joined with the original error (`live_viewport_test.go:16–46`).
   - The owner fixture establishes document readiness before creating a document-owned receipt (`operator_owner_live_test.go:101`).
   - The regression exercises held delivery, unchanged geometry, and metrics/wait failures using Chrome rather than merely asserting mocked call order.
   - The atlas documents the new helper and its readiness precondition.

2. **Critical findings**

   None.

3. **Important findings**

   The browser verification blocker above. No additional code defect identified.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused tests compiled, but browser startup prevented behavioral verification and a meaningful RED check.
   - `TestTrackedGoFilesPassCatalog` passed in 7.162s.
   - `git diff --check` passed; checkout remains unchanged.
   - Normal landing-suite completion remains unverified.

6. **Architectural notes**

   - **ARCH-DRY: pass.** The owner scenario’s three resize requests share one helper.
   - **ARCH-PURE: pass.** This is test integration glue with an injected protocol caller; no business logic was moved into IO.
   - **ARCH-PURPOSE: pass statically.** The barrier addresses consumed resize rather than viewport dimensions alone. Runtime confirmation remains blocked.
   - **ARCH-EVOLVE: pass.** No persisted format changes.
   - Atlas coverage is present. No new user-facing command, flag, or configuration requires a README update. No Core concepts table is claimed.

7. **Plan revision recommendations**

   None. Supply the executable evidence and complete the remaining landing checks before marking final verification complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"36daa87b-ae6c-4809-bfce-1c37009e60d2","repository":"github.com/xianxu/pair","issue_id":"000890","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c3e0081e8f6a879a","reviewed_head":"4b0b9f8e9d4d9b0c9d5c137bff879c5432319669","slot":"pair:9","requested_at":"2026-10-25T17:52:44.764516Z"},"phase":"finished","pid":73688,"started_at":"2026-10-25T17:52:44.902672Z","finished_at":"2026-10-25T17:54:15.325967Z","session_id":"01a17431-86dd-7b62-a2e5-84d668a3830b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.788744875s
head: "4b0b9f8e9d4d9b0c9d5c137bff879c5432319669"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-25T11:05:05-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 890 — flake: TestOperatorOwnerLiveAnnotationOff |
| repo | pair |
| issue file | workshop/issues/000890-flake-testoperatorownerliveannotationoff.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11fece5cfb3b7da9ab93e6e592ec4f940919887a..99919a7e9fd4cceb1746a2ef24758ceed9fd30a2 |
| command | sdlc close --issue 890 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T11:05:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The resize barrier matches the documented contract, preserves native input and the existing timeout, and has meaningful RED/GREEN host evidence. No code defect was identified. Local Chrome still fails before assertions, but the submitted evidence answers BR-1 on code unchanged at this pinned head.

Done-when: current — its clauses cover consumed-resize ordering, unchanged geometry, error cleanup, native owner behavior, and the revised document-readiness requirement.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Inspected host-proof.md and both complete logs; SHA256 hashes match. At requested head 4b0b9f8e9d4d9b0c9d5c137bff879c5432319669, both tests passed three repetitions in 18.909s, and the metrics-only overlay failed the held-event assertion in 1.347s. Git comparison confirms no cmd changes between that head and pinned head 99919a7e9fd4cceb1746a2ef24758ceed9fd30a2. This answers the requested positive and counterfactual measurements despite local Chrome startup failure.
```

1. **Strengths**

   - `live_viewport_test.go:16` centralizes receipt creation, waiting, cancellation, and object release; cleanup errors preserve the original error.
   - `operator_owner_live_test.go:101` establishes the fixture document before creating its document-owned receipt.
   - The regression holds actual Chrome resize delivery, checks panel clamping after release, and exercises unchanged geometry and metrics/wait failures.
   - `atlas/webcast.md` documents the helper and its readiness precondition.

2. **Critical findings**

   None.

3. **Important findings**

   None remaining.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Host evidence establishes three passing repetitions of both affected tests and a failing metrics-only control.
   - The local focused command failed all repetitions during Chrome startup with “GPU process isn't usable”; no assertions executed locally.
   - `TestTrackedGoFilesPassCatalog` passed in 6.549s.
   - Pinned-range `git diff --check` passed; checkout remains unchanged.
   - Completion of the recorded affected check and normal landing suite remains a landing prerequisite; this window does not establish their completion.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY: pass.** Owner viewport changes share one helper.
   - **ARCH-PURE: pass.** The change is thin integration glue with an injected CDP caller; it introduces no business logic mixed with IO.
   - **ARCH-PURPOSE: pass.** Completion now depends on consumed resize, matching the synchronous handlers in `web/floating-panel.js:173` and `web/cast-tools.js:44`.
   - **ARCH-EVOLVE: pass.** No persisted format changes.
   - Atlas coverage is present. No new user-facing command, flag, or configuration requires a README update. No Core concepts table is claimed.

7. **Plan revision recommendations**

   None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"32d76918-1d5e-457e-b080-5fed06b048fa","repository":"github.com/xianxu/pair","issue_id":"000890","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e381b82178cc6b83","reviewed_head":"99919a7e9fd4cceb1746a2ef24758ceed9fd30a2","slot":"pair:9","requested_at":"2026-10-25T18:03:31.138013Z"},"phase":"finished","pid":26688,"started_at":"2026-10-25T18:03:31.307558Z","finished_at":"2026-10-25T18:05:03.560874Z","session_id":"01a1743b-641e-7601-bd68-65a4970d97bd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.324094s
head: "99919a7e9fd4cceb1746a2ef24758ceed9fd30a2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
