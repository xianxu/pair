# Boundary Review — pair#572 (whole-issue close)

| field | value |
|-------|-------|
| issue | 572 — Build and test with -trimpath so worktrees share the Go build cache |
| repo | pair |
| issue file | workshop/issues/000572-build-and-test-with-trimpath-so-worktrees-share-the-go-build-cache.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6da7d4b809729db8a8766bd71258ca8f3326d9a..4b989cc9206b4dd344de55150d9c3c3b410666c5 |
| command | sdlc close --issue 572 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T12:01:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation delivers the trimpath defaults, shared checkout discovery, documentation, and measured package-cache reuse. Targeted checks passed except two new tests that assume temporary directories lie outside the checkout. Fix that test isolation issue before crossing the boundary.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: test-filesystem-isolation
    title: |
      Negative checkout-discovery tests depend on TMPDIR being outside Pair
    detail: |
      internal/reporoot/root_test.go:41 and probes/zellijprobe/reporoot_test.go:62 treat t.TempDir() as outside any Pair checkout. With this review's TMPDIR under .sdlc-review/tmp, ancestor discovery correctly finds Pair, causing TestFindRejectsMissingOrUnreadableModule/missing and TestRepoRootOutsideCheckoutExplainsOverride to fail. Make the negative fixtures establish their own module boundary or inject filesystem ancestry, rather than relying on host temporary-directory placement; rerun both packages with TMPDIR inside the checkout. ARCH-PURE: keep this environment dependency explicit in integration fixtures.
```

1. **Strengths**

   - `Makefile.local:6–9` preserves caller flags, avoids duplicate trimpath flags, and exports them to nested commands; the Make regression passed.
   - `internal/reporoot/root.go` centralizes discovery and rejects foreign modules without falling through to enclosing assets.
   - The source sweep found no remaining `runtime.Caller` asset lookups. README and atlas cover the new defaults and probe override.
   - Retained compiler traces and measurement data support cache reuse: trimmed worktree B performed no compilation and added zero cache bytes.

2. **Critical findings**

   None.

3. **Important findings**

   The test-isolation defect above. Reproduced with:

   `GOFLAGS='-p=3 -trimpath' go test ./internal/reporoot ./probes/zellijprobe -count=1`

4. **Minor findings**

   None.

5. **Test coverage notes**

   Make environment checks, recording fixtures, artifact inventory, birthrace regression, and targeted tests across all eight affected command packages passed. Both required-oracle failure regressions passed. The retained full-suite log matches its documented SHA-256 prefix; only tracker/plan files changed after that tested commit. I did not rerun the full suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared resolver and probe adapter replace repeated discovery.
   - **ARCH-PURE — flag:** negative integration fixtures depend on uncontrolled filesystem ancestry.
   - **ARCH-PURPOSE — pass:** consumers were swept; defaults and compile-cache evidence satisfy the intended purpose.
   - **ARCH-MOCK — pass:** no new external-service dependency; filesystem discovery uses temporary fixture trees.
   - **ARCH-CONSTRAINTS — pass:** caller concurrency flags remain intact; measurement was bounded.
   - **ARCH-SECURE — pass:** module identity and explicit overrides are validated; invalid overrides fail visibly.
   - **ARCH-ORDER — pass:** discovery adds no cached state across events; probe roots resolve before trials.
   - **ARCH-FUNERAL — pass:** test artifacts use managed temporary lifetimes; measurement cleanup is documented.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording that negative discovery fixtures must remain valid when `TMPDIR` is inside the checkout, and include that environment in qualification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bb368dd8-672b-4449-825d-43b9c4ccbada","repository":"github.com/xianxu/pair","issue_id":"000572","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8d7b306e96ed3b64","reviewed_head":"4b989cc9206b4dd344de55150d9c3c3b410666c5","slot":"pair:1","requested_at":"2026-10-15T18:59:44.025903Z"},"phase":"finished","pid":97863,"started_at":"2026-10-15T18:59:44.257638Z","finished_at":"2026-10-15T19:01:12.501355Z","session_id":"01a140ef-4417-7b23-8312-216cf76e4f0d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.639174542s
head: "4b989cc9206b4dd344de55150d9c3c3b410666c5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T12:06:15-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 572 — Build and test with -trimpath so worktrees share the Go build cache |
| repo | pair |
| issue file | workshop/issues/000572-build-and-test-with-trimpath-so-worktrees-share-the-go-build-cache.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6da7d4b809729db8a8766bd71258ca8f3326d9a..96e5c25094d656db03eb72c64354cb10d7b6fee5 |
| command | sdlc close --issue 572 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T12:06:15-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers repository-local trimpath defaults, path-independent asset discovery, and documented compile-cache reuse. BR-1 is addressed with verified regression evidence. No new blocking findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Negative fixtures now establish foreign-module boundaries. Both packages and birthrace pass with TMPDIR inside/outside the checkout, with/without trimpath. Overlaying the previous fixtures reproduces both original failures under in-checkout TMPDIR.
```

Done-when: current — the full-suite and cross-worktree cache criteria still cover the revised fixture-isolation contract and delivered behavior.

1. **Strengths**

   - `Makefile.local:6` preserves caller GOFLAGS, exports trimpath, and supports the documented opt-out; nested Make regression checks pass.
   - `internal/reporoot/root.go:22` centralizes discovery and rejects foreign module boundaries.
   - The source sweep found no remaining `runtime.Caller` asset lookups. README and atlas describe the new defaults and probe override.
   - Retained cache traces support identical trimmed archives with zero second-worktree compilation or cache growth.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the four TMPDIR/trimpath combinations, Make environment checks, artifact inventory suite, targeted fixture/oracle and affected-package tests, and compilation of all probes. Previous fixture versions reproduce BR-1’s failures. The retained full-suite log matches SHA-256 prefix `7943372536ad`; subsequent code changes are the two tested fixture corrections. The full suite was not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one shared resolver; consumers use it.
   - **ARCH-PURE — pass:** filesystem discovery remains a small integration boundary; fixture ancestry is explicit.
   - **ARCH-PURPOSE — pass:** affected consumers were swept; defaults and cache evidence fulfill the issue.
   - **ARCH-MOCK — pass:** no new external-service seam; discovery tests use isolated filesystem trees.
   - **ARCH-CONSTRAINTS — pass:** caller concurrency flags survive; measurement is bounded.
   - **ARCH-SECURE — pass:** module identity and overrides are validated; invalid overrides fail visibly.
   - **ARCH-ORDER — pass:** discovery holds no state between calls; probe roots resolve before trials.
   - **ARCH-FUNERAL — pass:** temporary fixtures have managed cleanup; measurement artifacts have documented removal.

7. **Plan revision recommendations:** None. Existing revisions accurately record the final command, override semantics, and BR-1 correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d3bf6ae6-f3c5-4477-bddb-647d0a8d2245","repository":"github.com/xianxu/pair","issue_id":"000572","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/82a67b694470feab","reviewed_head":"96e5c25094d656db03eb72c64354cb10d7b6fee5","slot":"pair:1","requested_at":"2026-10-15T19:04:52.235721Z"},"phase":"finished","pid":35745,"started_at":"2026-10-15T19:04:52.430688Z","finished_at":"2026-10-15T19:06:14.883149Z","session_id":"01a140f3-f77a-7050-8bf7-1f08782a7cb9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.417232625s
head: "96e5c25094d656db03eb72c64354cb10d7b6fee5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
