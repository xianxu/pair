# Boundary Review — pair#642 (whole-issue close)

| field | value |
|-------|-------|
| issue | 642 — Changelog Lua test shares reload scratch path across concurrent suites |
| repo | pair |
| issue file | workshop/issues/000642-changelog-shared-scratch.md |
| boundary | whole-issue close |
| milestone | — |
| window | f823355f9b9988dfb9888eed1bac54de44edc10f..c03fe7d4d8121e1b1b1850fb250da0d0c2b121f6 |
| command | sdlc close --issue 642 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T06:52:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fulfills the issue’s scratch-ownership contract and preserves the original assertions. The deterministic regression passes at head and fails against the pinned base with both original reload failures. No blocking findings.

Done-when: current — it covers concurrent isolation, preserved assertions, and cleanup on success, assertion failure, and Lua error.

```findings
{}
```

1. **Strengths**

   - All three fixture paths derive from one atomic scratch-root allocation (`nvim/changelog_test.lua:10`).
   - Cleanup follows the protected body and runs before reporting either assertion failures or Lua errors (`nvim/changelog_test.lua:137`).
   - The regression enforces write-A → complete-B → release-A ordering using real Neovim processes, and checks sibling-file survival and caller-sentinel preservation (`tests/changelog-isolation-test.py:75`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently verified:

   - `make test-lua` passed, including the new regression.
   - Standalone changelog assertions passed.
   - Unset and empty TMPDIR both allocated beneath `/tmp` and cleaned up.
   - The committed collision regression against archived base source failed with both expected reload diagnostics.
   - `git diff --check` passed.

   The full repository suite was not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Reload, annotation, and named-buffer paths share one ownership source; no fixed-path sibling remains in this fixture.
   - **ARCH-PURE: pass.** Changes remain within filesystem/process integration tests; no production business logic or purported pure entity gained IO.
   - **ARCH-PURPOSE: pass.** The implementation addresses the complete three-path family and cleanup contract.
   - Docs gate: no new user-facing or architectural surface requiring README or atlas updates.

7. **Plan revision recommendations:** None. The checked implementation items match the diff; close review and landing remain explicitly identified as workflow gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2fdcf2cb-567c-48a3-930e-b715c961f204","repository":"github.com/xianxu/pair","issue_id":"000642","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9ee0434082eb51d0","reviewed_head":"c03fe7d4d8121e1b1b1850fb250da0d0c2b121f6","slot":"pair:1","requested_at":"2026-10-17T13:51:03.251325Z"},"phase":"finished","pid":72852,"started_at":"2026-10-17T13:51:03.351742Z","finished_at":"2026-10-17T13:52:21.176385Z","session_id":"01a14a21-5fa2-7f80-b9c8-c3d95acd4e3d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.374389542s
head: "c03fe7d4d8121e1b1b1850fb250da0d0c2b121f6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
