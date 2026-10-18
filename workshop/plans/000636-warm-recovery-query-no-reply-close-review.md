# Boundary Review — pair#636 (whole-issue close)

| field | value |
|-------|-------|
| issue | 636 — Warm recovery live conformance intermittently receives no Zellij query reply |
| repo | pair |
| issue file | workshop/issues/000636-warm-recovery-query-no-reply.md |
| boundary | whole-issue close |
| milestone | — |
| window | 861e4f1de857889c0d55d716b026828f88338f58..9ca9253b21b9a5efc7a6f6402b82dcf348ea1ea4 |
| command | sdlc close --issue 636 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T08:47:05-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The isolation and recovery-refusal coverage are solid, and targeted normal/race checks pass. One diagnostic correctness gap blocks acceptance: captured deadlines describe the fixture context rather than the context governing each query.

Done-when: current — it explicitly covers diagnostic/isolation delivery and defers native cause qualification.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: observation-context-fidelity
    title: |
      Query capture records the installation deadline instead of the query deadline
    detail: |
      cmd/internal/couchcore/recovery_query_capture_test.go:234 closes over the context supplied to Context; lines 250–251 then report that context's deadline for every query. Recovery creates shorter observation contexts (recovery_execute.go:105,133), and the native adapter also bounds queries to five seconds. The affected consumers are warm/checkpoint recovery (recovery_conformance_live_test.go:70), normal park (park_lifecycle_live_test.go:227), and intent-only park (:249), whose later 750ms context is likewise invisible to this closure. This violates plan task 3's call-context measurement contract and ARCH-PURPOSE. Carry the actual query caller/effective deadline in the observer event and consume that value. Add a regression installing capture on a longer-lived parent, querying through a shorter-lived child, and asserting the recorded deadline; it must fail with the current closure.
```

1. **Strengths**
   - Twelve native-reply fault cases exercise production recovery and assert unchanged durable state, no effects, and no retry.
   - Private roots survive both direct and nested child environments; poisoned ambient values, cleanup, restoration, and socket-path capacity are checked.
   - Capture owns copied bytes, bounds storage, and flushes after fixture cleanup. The later diagnostic query was removed.

2. **Critical findings:** None.

3. **Important findings:** Deadline fidelity, detailed above.

4. **Minor findings:** None.

5. **Test coverage:** Targeted recovery/isolation/capture tests passed normally and with `-race`. Complete `zellijreply`, `zellijtest`, `testpair`, and `pairlifecycletest` packages passed. Diff whitespace checks passed. No native qualification was run.

6. **Architecture:** ARCH-DRY passes: shared fake, environment boundary, and observer are reused. ARCH-PURE passes: integration helpers remain test-only; existing pure decisions/decoders remain unchanged. ARCH-PURPOSE flags the inaccurate deadline measurement above. Atlas documents the new diagnostic surface; no new README-facing commands or configuration were introduced.

7. **Plan revisions:** No scope revision needed. Correct task 4’s selector from `TestRecoveryNativeReply` to `TestRecoverThreadNative` so the documented command includes the new refusal and healthy-control tests.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8f8d6bdd-c48a-4a55-a32f-f98c30469118","repository":"github.com/xianxu/pair","issue_id":"000636","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1939dabf624321ab","reviewed_head":"9ca9253b21b9a5efc7a6f6402b82dcf348ea1ea4","slot":"pair:6","requested_at":"2026-10-18T15:45:27.178775Z"},"phase":"finished","pid":39810,"started_at":"2026-10-18T15:45:27.380778Z","finished_at":"2026-10-18T15:47:05.17191Z","session_id":"01a14fb0-790b-73a1-8508-ed46bd2695f5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.495410334s
head: "9ca9253b21b9a5efc7a6f6402b82dcf348ea1ea4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
