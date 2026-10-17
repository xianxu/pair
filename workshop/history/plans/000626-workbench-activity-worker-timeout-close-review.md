# Boundary Review — pair#626 (whole-issue close)

| field | value |
|-------|-------|
| issue | 626 — Workbench activity invariant intermittently times out waiting for cold worker |
| repo | pair |
| issue file | workshop/issues/000626-workbench-activity-worker-timeout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1928183114579d89eabd58cc4e35f01c254a35ca..64bae95b018325198f47183bfecfd45ffd4d917d |
| command | sdlc close --issue 626 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T09:12:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change removes demonstrated quadratic delimiter-search work without changing framing semantics, deadlines, or cache authority. Regression evidence is reproducible, and retained progress and suite logs match their recorded hashes. The original historical timeout’s exact attribution remains explicitly unproven.

Done-when: current — the clauses cover the demonstrated cold-work defect, its regression, preserved accounting, and package/full-suite verification; the revision only clarifies workflow sequencing.

```findings
{}
```

1. **Strengths**
   - The shared framer fixes all existing callers without duplicating reader logic (`cmd/internal/sessioninventory/scan_helpers.go:60`).
   - Deterministic searched-byte assertions catch the performance defect independently of host timing (`scan_helpers_work_test.go:23`).
   - Boundary tests preserve offsets, CR handling, tails, limits, and early termination.
   - The UI invariant retains twenty 30MB ledgers, its 30-second guard, and unchanged-tick zero-read/decode assertions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Sessioninventory, threadactivity, and workbenchperftest packages passed.
   - The UI invariant passed in 5.80s.
   - Restoring the old search behavior through a temporary overlay made all three search-work regression cases fail.
   - Inspected the recorded full-suite log; subsequent changes affect only tracker/lesson text. Both retained diagnostic runs contain 120 successful probes across six completed generations.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Existing chunked-reader consumers share the correction.
   - **ARCH-PURE: pass.** IO remains injected through `Runtime`; regression tests use in-memory data.
   - **ARCH-PURPOSE: pass.** Removes measured excess work while preserving the original workload and evidence.
   - Atlas documentation is updated. No new user-facing command, flag, or configuration requires a README change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2db3f895-7a8c-44af-b81d-c7c5c17d572f","repository":"github.com/xianxu/pair","issue_id":"000626","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f3ed271f4557dd3a","reviewed_head":"64bae95b018325198f47183bfecfd45ffd4d917d","slot":"pair:6","requested_at":"2026-10-17T16:11:40.160755Z"},"phase":"finished","pid":66323,"started_at":"2026-10-17T16:11:40.386777Z","finished_at":"2026-10-17T16:12:46.309184Z","session_id":"01a14aa2-1dad-7e10-8e0b-f192c5557082","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.01287925s
head: "64bae95b018325198f47183bfecfd45ffd4d917d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
