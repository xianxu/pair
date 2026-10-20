# Boundary Review — pair#622 (whole-issue close)

| field | value |
|-------|-------|
| issue | 622 — Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not |
| repo | pair |
| issue file | workshop/issues/000622-couch-server-spends-1-core-emulating-every-hosted-client-s-output-in-a-vt-parser-visible-or-not.md |
| boundary | whole-issue close |
| milestone | — |
| window | cecb0d73f8d9aed399c37daf98f11f4088167dba..90384d10782ab96507d6f735d4527f69b28becb1 |
| command | sdlc close --issue 622 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T09:01:50-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range changes acceptance documentation only. No blocking defects found. The revised contract clearly separates production conformance from causal CPU acceptance assigned to #723. Retained measurement files support the reported census, throughput and CPU figures; operator console acceptance remains testimony recorded in the issue.

```findings
{}
```

1. **Strengths**

   - Independently verified all retained manifest hashes, including both native profiles.
   - Recomputed 775,088.63 bytes/s across 81.799429 seconds, with twenty unchanged client identities and zero omissions.
   - Documentation preserves workload mismatch and timestamp skew instead of attributing lower live CPU to the optimization.
   - Existing ASCII differential and counter tests passed during this review.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed focused terminal validation and Child ingress-counter tests, plus VT ASCII boundary, differential partition, generated sequence and oracle-mutation tests. `git diff --check` passed. No runtime changes require new regression tests in this range; the historical full suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Retains the existing ingest/Endpoint measurement seam.
   - **ARCH-PURE — pass:** No new mixing of computation and IO.
   - **ARCH-PURPOSE — pass:** Implemented optimization remains present; the revised acceptance contract explicitly preserves causal measurement as separate work.
   - **ARCH-MOCK — pass:** Existing fake Child exercises the production parser/publication path.
   - **ARCH-CONSTRAINTS — pass:** Reports workload and attribution limits; introduces no runtime resource demand.
   - **ARCH-SECURE — pass:** Raw terminal evidence remains private; no new credential or input-processing surface.
   - **ARCH-ORDER — pass:** Measurements distinguish client incarnations and disclose unequal observation windows.
   - **ARCH-FUNERAL — pass:** Temporary experiment ownership and cleanup are specified.
   - **ARCH-EVOLVE — pass:** No persisted runtime format changes.

   README and atlas already describe the landed diagnostics and ASCII optimization. This range introduces no additional user-facing or architectural surface requiring updates.

7. **Plan revision recommendations:** None. The appended transfer revision identifies the preceding experiment checklist as historical scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"409271bc-e376-488c-9ee9-0beaf83bb3ee","repository":"github.com/xianxu/pair","issue_id":"000622","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ad3bbfa6cf298e25","reviewed_head":"90384d10782ab96507d6f735d4527f69b28becb1","slot":"pair:5","requested_at":"2026-10-20T16:00:00.752748Z"},"phase":"finished","pid":93910,"started_at":"2026-10-20T16:00:00.843873Z","finished_at":"2026-10-20T16:01:22.009186Z","session_id":"01a15a0a-8451-7f10-9ee2-d0185e7721ff","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.444193208s
head: "90384d10782ab96507d6f735d4527f69b28becb1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
