# Boundary Review — pair#638 (whole-issue close)

| field | value |
|-------|-------|
| issue | 638 — Couch Ctrl+Y copy still fails live after #604; the error vanishes and isn't logged |
| repo | pair |
| issue file | workshop/issues/000638-couch-ctrl-y-copy-still-fails-live-after-604-the-error-vanishes-and-isn-t-logged.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7d98e3a9b1a258e833008d0aa60c7cd5be8f939..ee8083d5ce6a5bc13d903040fbacd99c47b84095 |
| command | sdlc close --issue 638 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T03:23:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised instrumentation-first acceptance contract. Native diagnostics survive the Console path, failed or unconfirmed attempts retain their original retry payload, and supervision receives metadata without copied content or stderr. No blocking defects found. The historical live clipboard failure remains unresolved, as explicitly documented in the acceptance revision.

```findings
{}
```

1. **Strengths**

   - One clipboard executor preserves typed errors, exit status, cancellation, and bounded stderr while maintaining legacy behavior.
   - `ReduceCopyFeedback` separates retained copy failures from ordinary notices and preserves the original payload across retries.
   - Banner space is reserved before frame windowing; tests cover selected actions, cursor placement, and minimum geometry.
   - Supervision reuses the existing bounded writer. README and atlas document the new behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed the complete `clipcmd`, `couchtty`, `couchcmd`, and `artifactpath` package suites. Coverage exercises native failures, stderr bounds and sanitization, payload retention, independent fallback errors, metadata privacy, and deterministically ordered shutdown cancellation. `git diff --check` passed.

   The real clipboard harness remained disabled. I did not independently rerun the repository-wide recorded suite; its passing result is documented in the tracker.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared executor, sanitizer, reducer, and diagnostic writer are reused.
   - **ARCH-PURE — pass:** Feedback transitions and projections remain pure; execution and logging stay at integration boundaries. Core-concepts entries match the implementation.
   - **ARCH-PURPOSE — pass:** Delivers the revised landing contract without claiming the historical failure is fixed.
   - **ARCH-MOCK — pass:** Private executable doubles exercise the production subprocess boundary; opt-in native conformance is available.
   - **ARCH-CONSTRAINTS — pass:** Existing deadline remains; stderr retention is bounded and logging admission is nonblocking.
   - **ARCH-SECURE — pass:** Payload stays off argv and supervision; displayed stderr is sanitized.
   - **ARCH-ORDER — pass:** Explicit feedback outcomes, serialized attempts, retained retries, and shutdown cancellation have sequence coverage.
   - **ARCH-FUNERAL — pass:** Retry payload is released on dismissal, success, or teardown; new events inherit existing log retention.

7. **Plan revision recommendations:** None. Existing revisions clearly supersede pre-close live staging and paste verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c4ea9f34-0b04-4f3f-9c41-3693987b8072","repository":"github.com/xianxu/pair","issue_id":"000638","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b760feda66736c8e","reviewed_head":"ee8083d5ce6a5bc13d903040fbacd99c47b84095","slot":"pair:7","requested_at":"2026-10-17T10:21:34.505624Z"},"phase":"finished","pid":84352,"started_at":"2026-10-17T10:21:34.746355Z","finished_at":"2026-10-17T10:23:18.499141Z","session_id":"01a14961-9866-7693-b0c7-0621449cd880","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.799526875s
head: "ee8083d5ce6a5bc13d903040fbacd99c47b84095"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T05:44:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 638 — Couch Ctrl+Y copy still fails live after #604; the error vanishes and isn't logged |
| repo | pair |
| issue file | workshop/issues/000638-couch-ctrl-y-copy-still-fails-live-after-604-the-error-vanishes-and-isn-t-logged.md |
| boundary | whole-issue close |
| milestone | — |
| window | f948635fdf6f4fa391f4f4f00e286a4001d73282..f32f89001161ce26932530f69ee6c39c96bff66c |
| command | sdlc close --issue 638 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T05:44:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves successful clipboard completion when only stderr draining times out, while rejecting incomplete input. Both regression tests fail when their corresponding correction is removed. No blocking defects found. The revisions record this executor correction without expanding acceptance scope.

```findings
findings:
  - id: new
    severity: Minor
    family: shared-test-fixture-ownership
    title: |
      ARCH-DRY: descriptor-holder setup and cleanup are duplicated
    detail: |
      cmd/internal/clipcmd/runtime_clipboard_test.go:109-132 and :156-179 repeat environment setup and release/done polling. Extract a shared test helper owning the holder lifecycle so both regressions retain consistent cleanup.
```

1. **Strengths**
   - The shared executor correction covers legacy and local callers (`runtime.go:137`).
   - Successful completion retains diagnostic uncertainty through `StderrTruncated`.
   - Positive and negative fixtures exercise real subprocess descriptors without touching the host clipboard.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** Duplicated fixture lifecycle code, as recorded above.

5. **Test coverage:** Full `clipcmd` package and clipboard race tests passed. Removing the success exception fails both positive-test modes; removing the input guard fails both negative-test modes. Temporary overlays were removed; checkout remains clean.

6. **Architecture:** ARCH-DRY flagged only for test duplication. ARCH-PURE passes: execution remains at the existing IO boundary. ARCH-PURPOSE passes: both callers use the corrected completion rule. README documents the correction; no new architectural or user-facing surface requires an atlas addition.

7. **Plan revisions:** None required. There were no prior findings to dispose.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a2f86ae8-b962-46ce-9bee-71e21f315cfd","repository":"github.com/xianxu/pair","issue_id":"000638","recipe":"fix-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/15a8f84cde0c0be2","reviewed_head":"f32f89001161ce26932530f69ee6c39c96bff66c","slot":"pair:7","requested_at":"2026-10-17T12:42:46.274443Z"},"phase":"finished","pid":87160,"started_at":"2026-10-17T12:42:46.467441Z","finished_at":"2026-10-17T12:44:02.690333Z","session_id":"01a149e2-dd55-75f1-9b7b-d0599379e2ea","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.694949s
head: "f32f89001161ce26932530f69ee6c39c96bff66c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
