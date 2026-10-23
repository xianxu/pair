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
