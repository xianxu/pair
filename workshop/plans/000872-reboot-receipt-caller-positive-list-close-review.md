# Boundary Review — pair#872 (whole-issue close)

| field | value |
|-------|-------|
| issue | 872 — Admitted worker reboot loses steward positive-list authentication during receipt polling |
| repo | pair |
| issue file | workshop/issues/000872-reboot-receipt-caller-positive-list.md |
| boundary | whole-issue close |
| milestone | — |
| window | f01ccc06b55fd3c5ac383fe172f4a4bb3ab74bcf..533c58321b81755bd1e317ab129af2d95c97f081 |
| command | sdlc close --issue 872 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T16:38:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes satisfy the revised diagnostic-only acceptance criteria. Exact judged-listing evidence survives refusal, authorization remains conservative, and accepted mutations are not blindly resent. The original cause remains explicitly unclassified. No blocking findings.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/launcher/listing_evidence.go:19` preserves exact bounded bytes with full-input hash/count and injected observation time.
   - `cmd/internal/couchcmd/caller_listing_diagnostic_test.go:61` verifies missing/EXITED caller refusal, retained request ID, uncertainty, and no resend.
   - `cmd/internal/testcouch/caller_evidence.go:27` bounds later observations and clearly distinguishes them from authorization evidence.
   - Atlas updates accurately document the new diagnostics and unresolved cause.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused launcher, couchcore, couchcmd, and testcouch tests passed.
   - `make test-affected-inventory` passed.
   - Independently verified the retained native log’s recorded SHA256, worker5 completion, and exact cleanup confirmation. No native sample was repeated.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared parser, poller, and projection reused.
   - **ARCH-PURE — pass:** evidence construction is deterministic; observation remains in integration boundaries.
   - **ARCH-PURPOSE — pass:** revised diagnostic scope delivered through resolver, CLI, and harness.
   - **ARCH-MOCK — pass:** deterministic listing failures use the existing stateful fixture.
   - **ARCH-CONSTRAINTS — pass:** byte caps, query counts, and shared deadlines are exercised.
   - **ARCH-SECURE — pass:** private socket/path checks and allowlisted capture remain enforced.
   - **ARCH-ORDER — pass:** later recovery cannot replace judged evidence; uncertain outcomes retain no-resend behavior.
   - **ARCH-FUNERAL — pass:** no new durable artifact family or background worker.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.

7. **Plan revision recommendations:** None; existing revisions explain the scope and budget corrections.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e962227e-e6b3-45bf-91f3-c64fe8543ab0","repository":"github.com/xianxu/pair","issue_id":"000872","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4a4f1dcd854883a3","reviewed_head":"533c58321b81755bd1e317ab129af2d95c97f081","slot":"pair:2","requested_at":"2026-10-24T23:35:21.332266Z"},"phase":"finished","pid":1830,"started_at":"2026-10-24T23:35:21.430648Z","finished_at":"2026-10-24T23:36:32.270234Z","session_id":"01a17044-d70c-7dd2-8e9b-344f078880d9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.53752675s
head: "533c58321b81755bd1e317ab129af2d95c97f081"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
