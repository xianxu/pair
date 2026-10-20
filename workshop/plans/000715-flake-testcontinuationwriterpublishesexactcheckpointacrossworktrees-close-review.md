# Boundary Review — pair#715 (whole-issue close)

| field | value |
|-------|-------|
| issue | 715 — flake: TestContinuationWriterPublishesExactCheckpointAcrossWorktrees |
| repo | pair |
| issue file | workshop/issues/000715-flake-testcontinuationwriterpublishesexactcheckpointacrossworktrees.md |
| boundary | whole-issue close |
| milestone | — |
| window | efffa07be8d629ae4ef4894454ef312a508fc430..4d6c82c9c3a37b94f933370074388e16888b9759 |
| command | sdlc close --issue 715 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T18:22:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The ownership fix matches the Spec and passes focused behavioral, race, lint, and continuation tests. Independently removing the watcher join makes the new regression fail after observing actual retention acquisition following wrapper return. One inexpensive atlas correction remains before admission.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: atlas-surface-traceability
    title: |
      Document the new Pair-local catalog extension.
    detail: |
      internal/repolint/local_catalog_test.go:11 introduces pairCatalog and wrapper-child-release, and catalog_test.go:304 activates them, but atlas/go-lint-catalog.md does not describe this architectural surface. Both relevant passages remain upstream-only: the opening rule-maintenance/adoption instructions and the closing instructions for new mechanical mistake classes. Update both passages and document the local rule's scope, owning file, shared checker/suppression behavior, and preservation of the upstream catalog for sleep-debt matching. These are all instances of this documentation family in the reviewed window.
```

1. **Strengths**

   - `wrap.go:2574` preserves `Setsid` while retaining direct child ownership through the existing `ownedGroup`.
   - `wrap.go:3028` registers shutdown immediately after successful creation, covering subsequent wrapper returns.
   - `watcher_lifetime_test.go` exercises five meaningful startup/shutdown cases, including a TERM-refusing descendant.
   - The lint extension copies the upstream catalog and tests its integration through the repository scanner.

2. **Critical findings:** None.

3. **Important findings:** Document the local catalog extension as detailed above. The lifecycle atlas updates accurately describe the inspected implementation. No new user-facing commands, flags, or configuration require README changes.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Five watcher-lifetime cases.
   - Codex/Claude detach-before-first-turn acceptance.
   - Single-watcher, process-group isolation, and wrapper restart checks.
   - Focused race checks and full `internal/repolint` tests.
   - All five original continuation acceptance modes.

   Mutation verification omitted only `watcherGroup.Wait()` in a disposable copy: the normal regression failed with the expected late retention acquisition. The copy was removed. Full-suite and loaded-run claims in the issue log were not independently reproduced.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses `ownedGroup` and the shared lint checker.
   - **ARCH-PURE — pass:** production changes remain within process IO; the matcher operates on AST input.
   - **ARCH-PURPOSE — pass:** the production caller and all spawn mocks were updated; detach survival and shutdown ownership are exercised.
   - **ARCH-EVOLVE — pass:** no persisted formats or readers/writers change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry covering documentation of the Pair-local catalog composition and `wrapper-child-release` rule. No implementation-scope revision is needed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"da22683d-4059-44ff-ae5d-ed0947ab57a2","repository":"github.com/xianxu/pair","issue_id":"000715","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8c6548422ae3e3f2","reviewed_head":"4d6c82c9c3a37b94f933370074388e16888b9759","slot":"pair:3","requested_at":"2026-10-20T01:21:41.252129Z"},"phase":"finished","pid":7214,"started_at":"2026-10-20T01:21:41.340487Z","finished_at":"2026-10-20T01:22:58.801647Z","session_id":"01a156e6-62c5-7430-8b3a-b6867474bef4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.503104708s
head: "4d6c82c9c3a37b94f933370074388e16888b9759"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
