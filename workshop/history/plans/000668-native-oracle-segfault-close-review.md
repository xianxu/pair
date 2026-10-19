# Boundary Review — pair#668 (whole-issue close)

| field | value |
|-------|-------|
| issue | 668 — Native terminal oracle subprocess crashes with SIGSEGV in macOS conformance |
| repo | pair |
| issue file | workshop/issues/000668-native-oracle-segfault.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4d0037433b6b1882319432779974522a3934265b..5de5664237a15e4266dee7a9c0fad1db6adaed0f |
| command | sdlc close --issue 668 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T14:41:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diagnostic implementation preserves existing oracle assertions and captures failure evidence across all five snapshot callers. No correctness defect was found. The root README needs the new user-facing diagnostic instructions before this boundary passes.

Done-when: current — it explicitly covers the diagnostic stage, evidence retention, regression controls, CI publication, and subsequent reopening of the unresolved causal issue.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: root-readme-surface-coverage
    title: |
      README update appears missing for oracle diagnostic commands
    detail: |
      tests/terminal-oracle/README.md:30-37 introduces manual retention flags (-artifacts -outputdir <owned-dir>), exact-request replay (node tests/terminal-oracle/driver.cjs < request.json), and headless controls (node --test tests/terminal-oracle/driver_test.cjs). These are all new user-facing invocation surfaces in this window, but root README.md is unchanged. Update its existing oracle/native-testing guidance around lines 2166 and 2218 with retention usage and a link covering replay, controls, and diagnostic limits.
```

1. **Strengths**

   - `internal/testoracle/run.go:19` clones the request and separates streams, preserving exact evidence despite later caller mutation or assertion failure.
   - All five snapshot callers share the helper without changing frame assertions or existing contexts.
   - CI publishes complete owned evidence before scratch cleanup; regressions check final request/report markers.
   - Atlas and oracle documentation clearly distinguish diagnostic evidence from a demonstrated SIGSEGV fix.

2. **Critical findings:** None.

3. **Important findings:** Root README coverage, detailed above. A short documentation update resolves it.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the Go helper regressions, seven Python runner tests, four real Node controls, and targeted terminal, couchtty, and artifact-manifest checks. Node controls passed after installing locked dependencies. `git diff --check` passed. Native conformance and the recorded full suite were not independently rerun; their ordinary gates remain required.

6. **Architectural notes**

   - **ARCH-DRY: pass** — all five snapshot consumers derive subprocess handling from one helper.
   - **ARCH-PURE: pass** — the helper is a small IO boundary; terminal assertions remain in callers, and tests exercise owned children.
   - **ARCH-PURPOSE: pass** — the implementation delivers the explicitly revised diagnostic scope without retries or weakened assertions. The causal bug remains open.

7. **Plan revision recommendations:** None; the active plan matches the implementation. Complete the existing gate and post-landing reopening actions.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c20971ce-3245-4a2c-b228-a72e25146cbf","repository":"github.com/xianxu/pair","issue_id":"000668","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/46b0b590e8538e7d","reviewed_head":"5de5664237a15e4266dee7a9c0fad1db6adaed0f","slot":"pair:3","requested_at":"2026-10-18T21:39:46.559069Z"},"phase":"finished","pid":56040,"started_at":"2026-10-18T21:39:46.684305Z","finished_at":"2026-10-18T21:41:15.077363Z","session_id":"01a150f4-dca2-79c2-b3e1-567a31bfee2f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.701468167s
head: "5de5664237a15e4266dee7a9c0fad1db6adaed0f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T14:43:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 668 — Native terminal oracle subprocess crashes with SIGSEGV in macOS conformance |
| repo | pair |
| issue file | workshop/issues/000668-native-oracle-segfault.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4d0037433b6b1882319432779974522a3934265b..3853781e7f2f3c95c6b6e6b31448dbb6df3f46fe |
| command | sdlc close --issue 668 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T14:43:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The diagnostic implementation matches the revised scope, preserves existing frame assertions and deadlines, and covers all five snapshot callers. BR-1 is addressed. No new blocking findings were found; the recorded suite, ordinary gates, and post-landing reopening remain required.

Done-when: current — it covers diagnostic evidence, regression controls, CI publication, and reopening the unresolved causal issue.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:2169-2170 links headless controls, exact-request replay and report limits; README.md:2223-2228 documents retention flags and CI publication. These match tests/terminal-oracle/README.md:17-37 and the helper/runner implementation. This is a prose-only correction.
```

1. **Strengths**

   - `internal/testoracle/run.go:19` clones input and separates streams, preserving exact evidence through later assertion failures.
   - All five snapshot callers use the shared helper without weakening assertions.
   - CI publication tests verify complete request/report tails survive before scratch cleanup.
   - README and atlas explain report limitations without claiming the historical SIGSEGV is fixed.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed Go helper regressions, seven Python tests, four real Node controls after installing locked dependencies, targeted terminal/artifact checks, and `git diff --check`. Couchtty compiled but the selector matched no tests. Native conformance and the recorded full suite were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY: pass** — all five snapshot consumers share subprocess handling.
   - **ARCH-PURE: pass** — IO stays in a small boundary helper; tests exercise owned children.
   - **ARCH-PURPOSE: pass** — the revised diagnostic scope is delivered without retries or relaxed checks; causal resolution remains explicitly deferred.

7. **Plan revision recommendations:** None. Complete the existing gate and post-landing reopening actions.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"981ad3fa-5f8d-4433-81aa-c549a3a31782","repository":"github.com/xianxu/pair","issue_id":"000668","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1c252b73ed6c4e9b","reviewed_head":"3853781e7f2f3c95c6b6e6b31448dbb6df3f46fe","slot":"pair:3","requested_at":"2026-10-18T21:42:50.129962Z"},"phase":"finished","pid":26966,"started_at":"2026-10-18T21:42:50.229406Z","finished_at":"2026-10-18T21:43:29.419888Z","session_id":"01a150f7-a961-7f72-a263-b2512d6535a0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.455999834s
head: "3853781e7f2f3c95c6b6e6b31448dbb6df3f46fe"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
