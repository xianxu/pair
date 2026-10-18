# Boundary Review — pair#621 (whole-issue close)

| field | value |
|-------|-------|
| issue | 621 — Rename shell test fails legacy prefix case |
| repo | pair |
| issue file | workshop/issues/000621-rename-prefix-test.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ef12f619bc75a533b69c461f955f683b843133d..0df8e47804a65518ba75f0d4fa4ed06bf611998b |
| command | sdlc close --issue 621 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T14:35:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes reconcile the stale shell expectation with the existing exact-tag implementation. The updated matrix passes all 67 assertions, documentation matches the code, and the test now participates in `make test`. No blocking findings.

Done-when: current — the clauses still cover the clarified prefix contract, completion of every shell case, and exact-tag/substring safety.

```findings
{}
```

1. **Strengths**
   - `tests/pair-rename.sh:177` checks literal prefixed identities, preserves bare source/destination content, and refuses fallback when the exact source is absent.
   - `tests/pair-rename.sh:18` reuses the shared candidate helper and owns generated artifacts through one cleanup root.
   - `Makefile.local:228` includes the matrix in the normal suite.
   - Help, README, and atlas consistently describe the existing implementation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently passed `make test-pair-rename` (67 assertions), launcher `Rename|ParseRestart` tests, shell syntax, and diff whitespace checks. The full suite was not rerun during review.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared environment and binary helpers are reused.
   - **ARCH-PURE: pass** — existing pure validation/planning and injected runtime boundaries remain intact.
   - **ARCH-PURPOSE: pass** — CLI help, regression expectations, documentation, and suite inclusion deliver the chosen exact-tag contract.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f6171611-87da-4e6a-83e2-c471d524316a","repository":"github.com/xianxu/pair","issue_id":"000621","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0caf7326eed99e81","reviewed_head":"0df8e47804a65518ba75f0d4fa4ed06bf611998b","slot":"pair:7","requested_at":"2026-10-17T21:34:25.155652Z"},"phase":"finished","pid":64011,"started_at":"2026-10-17T21:34:25.289856Z","finished_at":"2026-10-17T21:35:48.422012Z","session_id":"01a14bc9-9906-7581-91b1-c357f5f3091b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.591413792s
head: "0df8e47804a65518ba75f0d4fa4ed06bf611998b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
