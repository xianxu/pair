# Boundary Review — pair#612 (whole-issue close)

| field | value |
|-------|-------|
| issue | 612 — Warm reattach acceptance intermittently resolves native binding |
| repo | pair |
| issue file | workshop/issues/000612-warm-reattach-native-query.md |
| boundary | whole-issue close |
| milestone | — |
| window | f58920969d8453b286316249746bd2221a510ff6..7d7e566b2ec852d97ee23759459f544715337be1 |
| command | sdlc close --issue 612 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T06:12:58-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fulfills #612’s Spec and Plan. The fake now models attachment without erasing surviving session evidence, and the acceptance test deterministically exercises inventory before terminal adoption. No blocking defects or documentation gaps found.

```findings
{}
```

1. **Strengths**

   - `artifactcollision_fake.go:178` provides one atomic attachment transition preserving binding, presence, and panes.
   - All four planned fixture sites use the transition; remaining clear operations model disappearance or replacement.
   - `artifactcollision_fake_test.go:92` covers competing index owners both present and absent, outside the requested candidates.
   - `warm_reattach_test.go:39` exercises real inventory at the problematic ordering while retaining the zero-resolution assertion.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed focused fake/production-contract tests, attached-client public attach, socket acceptance, continuation acceptance, 20 warm-reattach repetitions, and five race-enabled repetitions.

   Temporary overlays confirmed meaningful regression failures for all three controls: restoring the old warm transition, restoring detached-only claim counting, and making attachment a no-op. Temporary files were removed; the checkout remains unchanged.

   Inspected the recorded full-suite log and verified its SHA-256 prefix `a0871056f9cf`. Its tested commit differs from the pinned head only in tracker and lesson text. Also inspected the recorded complete-package repeat and race results.

6. **Architectural notes**

   - **ARCH-DRY — pass:** both query paths share `sessionNamesLocked` and production claim-counting logic.
   - **ARCH-PURE — pass:** the change stays in the injected fake boundary; existing pure projectors retain policy ownership.
   - **ARCH-PURPOSE — pass:** the ordering is reproducible, the zero-query contract remains enforced, and the planned fixture family is swept.

   No new production architecture or user-facing commands, flags, or configuration require atlas or README updates. No Core concepts table is present.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a4d94d68-b425-41f1-ae58-658153d861ec","repository":"github.com/xianxu/pair","issue_id":"000612","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4da6eaf4132c40fa","reviewed_head":"7d7e566b2ec852d97ee23759459f544715337be1","slot":"pair:1","requested_at":"2026-10-16T13:10:48.932574Z"},"phase":"finished","pid":36238,"started_at":"2026-10-16T13:10:49.063218Z","finished_at":"2026-10-16T13:12:55.350281Z","session_id":"01a144d6-2d0c-7460-87ad-786865f8260d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.044877208s
head: "7d7e566b2ec852d97ee23759459f544715337be1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
