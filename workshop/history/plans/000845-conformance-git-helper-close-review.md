# Boundary Review — pair#845 (whole-issue close)

| field | value |
|-------|-------|
| issue | 845 — Native Couch conformance isolation omits Git upload-pack helper |
| repo | pair |
| issue file | workshop/issues/000845-conformance-git-helper.md |
| boundary | whole-issue close |
| milestone | — |
| window | e6fc81eac82c5106f942c09dca4fac7139b17a0b..10a3fad911e2d37de16ace8002407fc1e745c385 |
| command | sdlc close --issue 845 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T14:31:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The helper implementation matches the plan, and all three focused packages passed. Native qualification remains unverified: my four-minute test watchdog expired while waiting for agent readiness. That watchdog was shorter than the bridge’s five-minute startup budget, so this result does not establish a product defect or confirm cleanup.

Done-when: current — the criteria still cover isolated fetching, native readiness, replacement-environment requests, and joined cleanup.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: native-qualification-evidence
    title: |
      Native readiness and cleanup qualification remains unverified
    detail: |
      PAIR_NATIVE_COUCH_BRIDGE=1 go test ./cmd/internal/testcouchbridge -run '^TestNativeCouchBridge$' -count=1 -timeout 4m -v timed out in the Claude cohort at native_live_test.go:148; its stack showed Fleet.Start waiting in WaitAgent, without a readiness or cleanup result. The reviewer-selected watchdog was shorter than StartupBudget (300 seconds), so this cannot establish a startup-contract failure. This family's instances are the uncompleted Claude cohort and unreached Codex cohort. Supply a complete pinned-head native qualification run with a watchdog covering the test's 630-second context and cleanup, after reconciling any residue from this interrupted attempt; include readiness, read-only request, EOF/TERM join, and owned-root removal results.
```

1. **Strengths**
   - Selected Git supplies its helper under the replacement build environment; selection failures retain the existing partial owner (`owner_build.go:220`).
   - Runtime installation preserves the `git-upload-pack` basename and validates sources before effects (`runtime_files.go:82`).
   - The real-fetch regression checks the fetched commit, private PATH, and poisoned ambient Git settings (`git_helper_test.go:13`).
   - Atlas documentation accurately describes the narrow capability.

2. **Critical findings:** None.

3. **Important findings:** Native qualification evidence, detailed above.

4. **Minor findings:** None.

5. **Test coverage:** Passed `testcouch` (38.997s), `testcouchbridge` (0.398s), and `couch-conformance-bridge` (0.192s). Invalid helper selection and installation refusals are covered. Native qualification timed out as described.

6. **Architecture**
   - **ARCH-DRY: pass** — reuses resolution, validation, installation, and cleanup.
   - **ARCH-PURE: pass** — added IO stays in the injected build seam.
   - **ARCH-PURPOSE: pass structurally** — helper selection reaches the runtime consumer; native acceptance remains unverified.
   - **ARCH-EVOLVE: pass** — no persisted-format change.

7. **Plan revisions:** No design revision needed. Record complete native qualification evidence before closing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fb86e8c2-7978-443d-b926-9a379e9432b2","repository":"github.com/xianxu/pair","issue_id":"000845","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/544eb5a779957616","reviewed_head":"10a3fad911e2d37de16ace8002407fc1e745c385","slot":"pair:4","requested_at":"2026-10-23T21:26:29.147155Z"},"phase":"finished","pid":34993,"started_at":"2026-10-23T21:26:29.282549Z","finished_at":"2026-10-23T21:31:39.897731Z","session_id":"01a16aa8-7e30-7981-8076-61aa2f1071da","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.01132625s
head: "10a3fad911e2d37de16ace8002407fc1e745c385"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-23T14:38:12-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 845 — Native Couch conformance isolation omits Git upload-pack helper |
| repo | pair |
| issue file | workshop/issues/000845-conformance-git-helper.md |
| boundary | whole-issue close |
| milestone | — |
| window | e6fc81eac82c5106f942c09dca4fac7139b17a0b..4651b220e5ddb2f2a88b166dc0734c0200ec0fa6 |
| command | sdlc close --issue 845 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T14:38:12-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation fulfills the selected-Git isolation contract without widening the runtime environment. Focused tests and reader inventory pass. BR-1 is resolved by complete native qualification evidence, verified against the test assertions and identical executable source at the pinned head.

Done-when: current — the criteria cover isolated fetching, readiness, replacement-environment requests, and joined cleanup; the revisions do not introduce uncovered behavior.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      workshop/evidence/000845/native-qualification.log records both Claude/EOF and Codex/TERM passing with readiness, read-only requests, joined shutdown, and owned-root removal. Inspected native_live_test.go assertions establish those checks. Recorded revision 5144008b4 and pinned head differ only in issue/evidence files; all four interrupted-run paths and four qualification fleet/candidate paths are currently absent. This correction supplies missing measurements without changing executable behavior.
```

1. **Strengths**

   - Helper selection queries the selected Git under the replacement environment and preserves actionable errors (`cmd/internal/testcouch/owner_build.go:268`).
   - Runtime installation retains the `git-upload-pack` invocation name and validates all sources before effects (`cmd/internal/testcouch/runtime_files.go:82`).
   - The real-fetch regression checks commit identity, private PATH, and rejection of ambient Git settings (`cmd/internal/testcouch/git_helper_test.go:13`).
   - Atlas accurately documents the added capability. No new user-facing command, flag, or configuration requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - `testcouch`: 46.720s
   - `testcouchbridge`: 0.484s
   - `couch-conformance-bridge`: 0.198s
   - `make test-affected-inventory`
   - Pinned-range whitespace check

   Coverage includes invalid helper selection, pre-effect installation refusal, builder wiring, and actual isolated fetching. Native qualification was assessed from retained complete output and its assertions; it was not rerun this round.

6. **Architectural notes**

   - **ARCH-DRY: pass** — reuses tool resolution, runtime inventory, and cleanup ownership.
   - **ARCH-PURE: pass** — added IO remains in the injected candidate-build seam.
   - **ARCH-PURPOSE: pass** — selection flows through `Binaries.GitUploadPack` into runtime installation and the actual fetch consumer.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

7. **Plan revision recommendations:** None. The recorded full-suite and downstream qualification gates remain separate from this review verdict.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4c4ee095-221e-4481-9747-40a048e4648a","repository":"github.com/xianxu/pair","issue_id":"000845","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3e0ad576f9ea0cc7","reviewed_head":"4651b220e5ddb2f2a88b166dc0734c0200ec0fa6","slot":"pair:4","requested_at":"2026-10-23T21:36:30.210492Z"},"phase":"finished","pid":16877,"started_at":"2026-10-23T21:36:30.372356Z","finished_at":"2026-10-23T21:38:10.981759Z","session_id":"01a16ab1-aa37-7402-8e28-e119c57eb2ae","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.524179166s
head: "4651b220e5ddb2f2a88b166dc0734c0200ec0fa6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
