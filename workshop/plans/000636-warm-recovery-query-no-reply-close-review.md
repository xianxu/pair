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

---

## Re-review — 2026-10-18T09:22:11-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 636 — Warm recovery live conformance intermittently receives no Zellij query reply |
| repo | pair |
| issue file | workshop/issues/000636-warm-recovery-query-no-reply.md |
| boundary | whole-issue close |
| milestone | — |
| window | 861e4f1de857889c0d55d716b026828f88338f58..823bafe3b328613b8016edd9baebe32be5b23a38 |
| command | sdlc close --issue 636 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T09:22:11-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the approved diagnostic and isolation scope. BR-1 is fixed with a regression that detects the original error. No new findings block shipping.

Done-when: current — it covers diagnostic capture, fixture isolation and refusal behavior, while explicitly deferring native cause qualification.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      zellij.go:199–217 supplies actual caller and effective deadlines; recovery_query_capture_test.go:266–277 consumes them for warm/checkpoint and both park modes. TestRecoveryQueryCaptureUsesQueryChildDeadline passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child. Adapter tests also cover absent, shorter and longer caller deadlines.
```

1. **Strengths**
   - Twelve reply-fault cases exercise production recovery and verify unchanged durable state, no effects and no retry.
   - Private roots survive direct and nested child environments; tests cover poisoned ambient values, restoration, cleanup and socket capacity.
   - Capture owns bounded response copies, flushes after cleanup and retains the original failed query without adding another query.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Focused recovery, isolation, capture and observer tests passed normally and with race detection. Complete launcher, zellijreply, zellijtest, testpair and pairlifecycletest packages passed. Additional recovery decision/refusal/deadline checks and diff whitespace checks passed. The mutation overlay was removed; tracked files remain unchanged. Native qualification and the full recorded suite were not rerun during this review.

6. **Architecture:** ARCH-DRY passes through reuse of the shared fake, environment boundary and observer. ARCH-PURE passes: integration measurement stays at the boundary, with existing pure recovery decisions unchanged. ARCH-PURPOSE passes against the explicitly revised delivery; deadline fidelity now applies across all capture consumers. Atlas covers the new surface; no README-facing interface was introduced.

7. **Plan revision recommendations:** None required; the existing revisions document the scope and BR-1 correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c4d2f279-28d6-4f52-ae92-8d6fb1f9de1a","repository":"github.com/xianxu/pair","issue_id":"000636","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8b67ddf6bac0d95a","reviewed_head":"823bafe3b328613b8016edd9baebe32be5b23a38","slot":"pair:6","requested_at":"2026-10-18T16:18:59.092567Z"},"phase":"finished","pid":12027,"started_at":"2026-10-18T16:18:59.275562Z","finished_at":"2026-10-18T16:22:11.024289Z","session_id":"01a14fcf-2c46-7623-9d57-a65ce8e2dbde","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.501091667s
head: "823bafe3b328613b8016edd9baebe32be5b23a38"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T09:52:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 636 — Warm recovery live conformance intermittently receives no Zellij query reply |
| repo | pair |
| issue file | workshop/issues/000636-warm-recovery-query-no-reply.md |
| boundary | whole-issue close |
| milestone | — |
| window | 19457fcdc3a298b967cac0bbc775b6540ceacd63..bf4af0d3b6ae83ca87b249e00148f7a903d28227 |
| command | sdlc close --issue 636 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T09:52:16-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff adds two justified lint exceptions and review records. Both exceptions apply to existing short Unix socket roots and preserve private allocation and cleanup. No new blocking defects or scope expansion found.

Done-when: current — the revision records lint integration for the approved diagnostic/isolation delivery; acceptance remains unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      launcher/zellij.go:199–217 supplies query caller/effective deadlines; recovery_query_capture_test.go:266–277 consumes them. The shorter-child regression passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child.
```

1. **Strengths**

   - Exceptions at `recovery_live_environment_test.go:23` and `recovery_native_reply_test.go:70` identify concrete socket-length constraints.
   - Removing each directive in memory produces exactly one `tmp-literal` finding; both original files pass that check.
   - Existing tests verify real socket binding, private-root propagation and cleanup.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Focused recovery, isolation, child-deadline and adapter-deadline tests passed. BR-1 mutation failed as expected. Diff whitespace checks passed. The repository-wide catalog test could not complete because this checkout has no remote main ref; direct catalog checks verified both changed files. Temporary artifacts were removed; tracked files remain unchanged.

6. **Architecture:** ARCH-DRY passes: both sites use the shared catalog mechanism. ARCH-PURE passes: no business logic or IO boundaries change. ARCH-PURPOSE passes: the exceptions preserve tested socket constraints without weakening runtime assertions. No new surface requires atlas or README updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"70983eb1-f791-42b1-a2e1-b9202c4ff552","repository":"github.com/xianxu/pair","issue_id":"000636","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fbaf4c690bbbb1c2","reviewed_head":"bf4af0d3b6ae83ca87b249e00148f7a903d28227","slot":"pair:6","requested_at":"2026-10-18T16:50:51.056979Z"},"phase":"finished","pid":61396,"started_at":"2026-10-18T16:50:51.152981Z","finished_at":"2026-10-18T16:52:15.722521Z","session_id":"01a14fec-5779-7fd2-9cf6-cee75bba9f64","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.481425291s
head: "bf4af0d3b6ae83ca87b249e00148f7a903d28227"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
