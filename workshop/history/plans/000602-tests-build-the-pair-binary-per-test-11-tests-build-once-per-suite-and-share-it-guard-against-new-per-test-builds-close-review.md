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

---

## Re-review — 2026-10-16T09:12:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 602 — Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds |
| repo | pair |
| issue file | workshop/issues/000602-tests-build-the-pair-binary-per-test-11-tests-build-once-per-suite-and-share-it-guard-against-new-per-test-builds.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53c96ded8854c4da834c594a9901b08c6f8078f2..0f07e6243f86ec8807daf6c1a7991c4189bdce62 |
| command | sdlc close --issue 602 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T09:12:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, and focused shared-binary and standalone checks pass. One guard defect blocks closure: it misses six of the eleven original build sites because it does not recognize their relative package paths.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      tests/lib/pair-test-binary.sh:25-30 preserves caller-relative outputs before selecting its checkout. The new regressions at tests/lib/pair-test-binary-test.sh:31-33 pass with the fix and fail against the pre-fix builder. The real embedded-runtime script also passes from tests/ with PAIR_TEST_BINARY unset.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1703-1713 now documents the plain suite binary, supplied absolute executable, lazy package fallback, cleanup ownership, and development linker distinction. These statements match Makefile.local:224-231, testpair.BinaryContext/Run, and the shell wrapper.
findings:
  - id: new
    severity: Important
    kind: defect
    family: full-pair-target-normalization
    title: |
      Source guard accepts the relative package paths used by six original offenders
    detail: |
      cmd/internal/testpair/build_guard_test.go:194-196 recognizes ./cmd/pair-go and paths ending /cmd/pair-go, but misses ../../pair-go. Running the guard against all eleven original Go sources detects only five; the missed family comprises couchcmd/continuation_acceptance_test.go, couchcmd/switchagent_acceptance_test.go, hoprttcmd/hoprtt_test.go, launcher/agent_command_acceptance_test.go, launcher/couch_launch_prompt_acceptance_test.go, and launcher/selected_root_test.go, all under cmd/internal/. Additional direct, CommandContext, and table mutation controls using ../../pair-go, plus a sibling ../pair-go control from cmd/couch, all pass undetected. ARCH-PURPOSE: these per-test builds can be reintroduced without failing the guard. Normalize package targets against the relevant working directory and add regression controls covering every original recipe while retaining unrelated-build negative controls.
```

1. **Strengths**

   - All eleven Go consumers and four shell consumers use the centralized binary infrastructure.
   - Lifecycle tests verify once-only construction, supplied-binary preservation, cancellation, and cleanup.
   - README and atlas describe the new surface and ownership.

2. **Critical findings:** None.

3. **Important findings:** The target-normalization defect above leaves the guard incomplete.

4. **Minor findings:** None.

5. **Test coverage notes:** `testpair -race` passed. Focused artifactpath, launcher, and pair-go checks passed; standalone hoprttcmd and embedded-runtime checks passed. Review-only overlay tests demonstrated the guard gap without changing tracked files. The full suite was not rerun.

6. **Architectural notes:** **ARCH-DRY: pass**—construction is centralized. **ARCH-PURE: pass**—guard analysis is separated from filesystem traversal; executable handling remains test-boundary glue. **ARCH-PURPOSE: flag**—enforcement misses the original relative-path recipes.

7. **Plan revision recommendation:** Add a `## Revisions` entry requiring the guard to reject all eleven original recipes, including package-relative paths, and add those recipes to its regression controls before re-closing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5d552aed-d39f-41f3-8656-2faa556e8b19","repository":"github.com/xianxu/pair","issue_id":"000602","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8fe64bf90af9ddec","reviewed_head":"0f07e6243f86ec8807daf6c1a7991c4189bdce62","slot":"pair:1","requested_at":"2026-10-16T16:10:03.748738Z"},"phase":"finished","pid":535,"started_at":"2026-10-16T16:10:03.891458Z","finished_at":"2026-10-16T16:12:34.505336Z","session_id":"01a1457a-485e-7072-8bf6-0ba91087b947","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.895003167s
head: "0f07e6243f86ec8807daf6c1a7991c4189bdce62"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T09:33:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 602 — Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds |
| repo | pair |
| issue file | workshop/issues/000602-tests-build-the-pair-binary-per-test-11-tests-build-once-per-suite-and-share-it-guard-against-new-per-test-builds.md |
| boundary | whole-issue close |
| milestone | — |
| window | 53c96ded8854c4da834c594a9901b08c6f8078f2..78826bb5fac275cc5cefa49d3b8882d66fc9821c |
| command | sdlc close --issue 602 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T09:33:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The shared-build migration matches the Spec and Plan. BR-3 is addressed with meaningful regression evidence: all eleven original Go sources are detected, while restoring the previous guard reproduces the six misses. No new blocking findings.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      build_guard_test.go:212-260 normalizes targets against package/command working directories. Committed regression fixtures pass at head and fail with the pre-fix implementation restored through a temporary overlay. Independently testing all eleven complete base-version source files confirms detection improves from 5/11 to 11/11; unrelated-build negative controls remain green.
```

1. **Strengths**

   - Construction is centralized in `tests/lib/pair-test-binary.sh`; suite and package lifetimes own cleanup.
   - All eleven Go consumers, four shell consumers, and the native CI candidate use the shared infrastructure while preserving explicit overrides.
   - Guard fixtures distinguish forbidden Pair builds from unrelated helper, probe, and test-executable builds.
   - README and atlas explain the new control, ownership, fallback, and plain-binary distinction.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes:** `testpair -race`, Pair CLI tests, and artifact-path tests passed. Guard regressions were verified red without the fix and green with it. Hoprtt initially exceeded its unchanged timing ceiling twice; its isolated test and final full-package rerun passed. The original helper also passed comparison runs, so the timing failures remain an observed limitation without an established connection to this diff. The full repository suite was not independently rerun.

6. **Architectural notes:**

   - **ARCH-DRY: pass** — one builder serves every migrated consumer.
   - **ARCH-PURE: pass** — source analysis is separate from filesystem traversal; executable handling stays in test infrastructure.
   - **ARCH-PURPOSE: pass** — the consumer sweep is complete, and enforcement covers every original recipe.

7. **Plan revision recommendations:** None; the existing R2 revision matches the delivered correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"84bfe20c-a9c8-414f-aad8-209b2f828b5a","repository":"github.com/xianxu/pair","issue_id":"000602","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9574c3b2f70066c0","reviewed_head":"78826bb5fac275cc5cefa49d3b8882d66fc9821c","slot":"pair:1","requested_at":"2026-10-16T16:29:19.249606Z"},"phase":"finished","pid":38327,"started_at":"2026-10-16T16:29:19.503089Z","finished_at":"2026-10-16T16:33:04.385506Z","session_id":"01a1458b-eae5-7fd2-8c11-23b794e60a6d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.768799667s
head: "78826bb5fac275cc5cefa49d3b8882d66fc9821c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
