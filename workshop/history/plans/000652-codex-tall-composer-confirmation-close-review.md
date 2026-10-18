# Boundary Review — pair#652 (whole-issue close)

| field | value |
|-------|-------|
| issue | 652 — Codex tall composer misses submission confirmation beyond 20 rows |
| repo | pair |
| issue file | workshop/issues/000652-codex-tall-composer-confirmation.md |
| boundary | whole-issue close |
| milestone | — |
| window | a10bc8d1d822afb0b75bfee696d8202b3629a76b..0ffeae407081ef795cfbc77a5023aecd0039fd82 |
| command | sdlc close --issue 652 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T16:28:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the issue’s approved design. Tall Codex recognition now shares positive geometry between Return handling and peer confirmation, while preserving short-input behavior, delivery ordering, deadlines, and the disabled Codex batch rollout. No blocking findings.

```findings
{}
```

1. **Strengths**

   - `composer_recognizers.go:46` centralizes prompt authority and requires complete, separated, trailing footer evidence beyond the old recognition limit.
   - `peer_composer.go:141` reuses those bounds and excludes native footer text from the recovered draft.
   - `codex_tall_composer_test.go:157` exercises the actual writer and delivery reducer, checking exact paste bytes, one submit, positive confirmation, and no replay.
   - README and atlas both document the changed behavior and conservative fallback.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed during review:

   - Focused Codex/composer/tall-peer/integration/PTY-geometry tests: **1.289s**.
   - The same selection with `-race`: **6.471s**.
   - Complete `wrapcmd` package: **126.160s**.
   - `git diff --check`.

   Tests cover boundary heights, Unicode and blank lines, broken geometry, changed footer wording, partial frames, stale clears, incomplete writes, operator interruption, overlays, and child exit. The authenticated native Codex run was not repeated; its opt-in harness and evidence assertions were inspected.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Both consumers derive authority from one helper; the footer predicate is moved and reused.
   - **ARCH-PURE — Pass:** Recognition and extraction remain deterministic snapshot operations.
   - **ARCH-PURPOSE — Pass:** Both Return safety and active-turn submission confirmation are delivered.
   - **ARCH-MOCK — Pass:** Stateful fake tests exercise production delivery paths; native qualification uses the existing owned PTY seam.
   - **ARCH-CONSTRAINTS — Pass:** Storage validation precedes bounded viewport scans; no new per-key IO or repeated footer-candidate suffix scans.
   - **ARCH-SECURE — Pass:** Malformed or incomplete tall evidence declines authority; existing input ownership guards remain.
   - **ARCH-ORDER — Pass:** Delivery transitions remain unchanged and controlled event sequences test uncertainty and replay prevention.
   - **ARCH-FUNERAL — Pass:** No new production artifact family or growing state; the native fixture retains existing teardown ownership.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"aa48ab05-072f-408a-9601-4461d068e6d8","repository":"github.com/xianxu/pair","issue_id":"000652","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/321b148f2e44a545","reviewed_head":"0ffeae407081ef795cfbc77a5023aecd0039fd82","slot":"pair:3","requested_at":"2026-10-17T23:25:55.945376Z"},"phase":"finished","pid":78155,"started_at":"2026-10-17T23:25:56.290741Z","finished_at":"2026-10-17T23:28:43.577542Z","session_id":"01a14c2f-b266-7ff2-81e1-d59cff77ae25","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.848308s
head: "0ffeae407081ef795cfbc77a5023aecd0039fd82"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
