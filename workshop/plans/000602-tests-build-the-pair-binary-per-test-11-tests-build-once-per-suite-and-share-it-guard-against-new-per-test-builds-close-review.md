# Boundary Review — pair#602 (whole-issue close)

| field | value |
|-------|-------|
| issue | 602 — Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds |
| repo | pair |
| issue file | workshop/issues/000602-tests-build-the-pair-binary-per-test-11-tests-build-once-per-suite-and-share-it-guard-against-new-per-test-builds.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53c96ded8854c4da834c594a9901b08c6f8078f2..ae4ddaf5c2a262304293d0edc6b5700dd848fdbd |
| command | sdlc close --issue 602 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T08:54:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared builder, package lifetime management, and consumer migrations largely match the Spec. Focused tests pass, but the centralized fallback introduces a reproduced working-directory regression, and the new test configuration lacks the required README documentation.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: standalone-build-location-independence
    title: |
      Centralized fallback breaks the embedded-runtime script from a checkout subdirectory
    detail: |
      tests/lib/pair-test-binary.sh:26 resolves ./cmd/pair-go against the caller's directory. tests/pair-embedded-runtime-test.sh:18 previously built an absolute package path and does not change directory. From tests/, the new helper exits 1 with "tests/cmd/pair-go: directory not found"; the previous absolute-path build exits 0. Family sweep: this is the affected migrated caller; adapt-schema explicitly changes to the root, retention/submission already required root-relative builds, and Go/native/wrapper callers establish the root. Resolve the builder's checkout independently of caller cwd, preserving relative output-path semantics, and add a non-root fallback regression test. ARCH-PURPOSE: standalone routes must remain functional.
  - id: new
    severity: Important
    kind: defect
    family: readme-new-test-controls
    title: |
      README update appears missing for PAIR_TEST_BINARY and shared test builds
    detail: |
      Makefile.local:224-231 and cmd/internal/testpair/binary.go:43 introduce PAIR_TEST_BINARY as a caller-controlled test setting, but README.md is unchanged. The complete new documentation surface is the supplied absolute executable, suite-owned plain binary, standalone package fallback, and their cleanup ownership. Document these together near README.md:1695 and clarify how plain test binaries differ from development binaries carrying defaultPairHome. atlas/index.md:40-48 already covers the architectural surface.
```

1. **Strengths**

   - `testpair.Run` cleans only its owned fallback after tests and cleanups finish.
   - Private placements preserve executable-location behavior through hard links, with exclusive cross-filesystem copy fallback.
   - Guard fixtures cover direct, context, table, shell, and Python builds while allowing unrelated fixtures.
   - Native CI explicitly clears the supplied-binary override to retain candidate freshness.

2. **Critical findings**

   - Builder working-directory regression, detailed above.

3. **Important findings**

   - Missing README documentation, detailed above.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   Passed `go test ./cmd/internal/testpair`, the complete `hoprttcmd` package, and both migrated Pair route tests. Independently reproduced the old/new build difference from `tests/`. The 13-minute full suite was not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Construction and placement share one builder; checkout discovery reuses `reporoot`.
   - **ARCH-PURE: pass.** Guard analysis is separated from filesystem traversal; executable support remains a small IO boundary.
   - **ARCH-PURPOSE: flag.** Consumers are migrated, but the standalone compatibility promise fails for the enumerated shell route.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording caller-independent fallback construction and a regression check from a checkout subdirectory. Include README documentation in the completion checklist.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"965278e6-3aab-4d71-822d-93ae9d3ea6df","repository":"github.com/xianxu/pair","issue_id":"000602","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f78f5df7c7076496","reviewed_head":"ae4ddaf5c2a262304293d0edc6b5700dd848fdbd","slot":"pair:1","requested_at":"2026-10-16T15:52:28.281709Z"},"phase":"finished","pid":3817,"started_at":"2026-10-16T15:52:28.405121Z","finished_at":"2026-10-16T15:54:18.124053Z","session_id":"01a1456a-2cf8-7392-a0a4-e4e69202f9ed","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.826847917s
head: "ae4ddaf5c2a262304293d0edc6b5700dd848fdbd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
