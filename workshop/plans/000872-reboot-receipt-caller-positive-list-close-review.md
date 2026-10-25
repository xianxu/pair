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

---

## Re-review — 2026-10-24T18:07:23-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 872 — Admitted worker reboot loses steward positive-list authentication during receipt polling |
| repo | pair |
| issue file | workshop/issues/000872-reboot-receipt-caller-positive-list.md |
| boundary | whole-issue close |
| milestone | — |
| window | c1b1ed3e94f951183f10bc8f06feb21d5c3977cb..0294ca6e9a790b954b0365792fd3f978d212b0af |
| command | sdlc close --issue 872 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T18:07:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff repairs source inventory, fixture conformance, and lint ownership qualifications without changing diagnostic production behavior or acceptance scope. No prior findings require disposition, and no new blocking defects were found. The main integration has no recorded conflict resolutions.

```findings
{}
```

1. **Strengths**
   - The three source classifications match their implementations and pass the exhaustive artifact inventory check.
   - The caller-listing fixture now uses the shared reply generator and passes production-decoder conformance.
   - The lint exemption identifies the actual `Fleet.runStartupQuery` owner; regression cases reject the former owner, wrong receiver, and wrong file.
   - The caller test’s reader classification correctly accounts for its new repository-backed fixture input.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Passed focused artifact inventory, shared fixture generator/decoder contracts, lint rule and tracked-file catalog, caller-evidence tests, `make test-affected-inventory`, and diff whitespace checks. No native sample was repeated.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses the canonical fixture serializer and existing inventories.
   - **ARCH-PURE — pass:** No new IO enters the pure diagnostic entities.
   - **ARCH-PURPOSE — pass:** Repairs cover all three omitted source classifications and narrowly qualify the real raw-query owner.
   - **ARCH-EVOLVE — pass:** No persisted schema or field semantics change. No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None; the plan and acceptance criteria remain unchanged in this window.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"547355fe-1a4e-43de-bae1-d9bf2f3ad202","repository":"github.com/xianxu/pair","issue_id":"000872","recipe":"fix-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bc7d6f909f22c87d","reviewed_head":"0294ca6e9a790b954b0365792fd3f978d212b0af","slot":"pair:2","requested_at":"2026-10-25T01:06:19.803692Z"},"phase":"finished","pid":68181,"started_at":"2026-10-25T01:06:19.927883Z","finished_at":"2026-10-25T01:07:21.52596Z","session_id":"01a17098-22ef-7a92-a1cc-2412ee71380c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.688996375s
head: "0294ca6e9a790b954b0365792fd3f978d212b0af"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
