# Boundary Review — pair#869 (whole-issue close)

| field | value |
|-------|-------|
| issue | 869 — Publish cancellation fixture PID readiness atomically |
| repo | pair |
| issue file | workshop/issues/000869-cli-fixture-pid-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | f9c06a92fe8d4b3fe1278f087ed85bfa60beffb6..ccc3b57741a240945922e20a8d5ffdfddc054f14 |
| command | sdlc close --issue 869 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T08:42:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Atomic rename fixes the empty-PID publication race, and the held-writer regression detects the original defect. However, the external `mv` introduces another cancellation race: readiness becomes visible before the shell has reaped the publication helper. Repeated tests on the pinned head reproduced cleanup-uncertainty failures.

Done-when: current — the clauses cover atomic publication, both consumers, preserved cleanup assertions, and repeated checks; the implementation does not yet satisfy them reliably.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: readiness-requires-helper-quiescence
    title: |
      PID readiness can trigger cancellation before the publication helper is reaped
    detail: |
      cmd/internal/testcouch/agentfixture/cli_group_test.go:17 publishes the final marker inside external mv, before mv exits and its parent shell reaps it. Cancellation can therefore kill both processes and leave a group member observed by runCLI's strict cleanup check, producing CleanupUncertain. The complete affected family is TestCLILiveLeaderCancellationJoinsBeforeCompletion (cli_group_test.go:77–79), TestRequestCancellationAcknowledgesJoinedChild (request_lifetime_test.go:70–81), and TestCLIPIDReadinessWaitsForCompletePublication (cli_group_test.go:133–159). The unchanged pinned head failed focused count=30 in both existing cancellation tests and count=100 in the CLI test; restoring the original producer through a temporary Go overlay passed the two existing tests at count=100. A temporary control holding a publication subprocess after rename reproduced cleanup uncertainty across all three consumers. Require readiness only after the publication helper is reaped, or publish within the leader without spawning that helper, and add a regression covering the post-rename/pre-reap interval. Preserve production cleanup strictness. This also flags ARCH-PURPOSE: replacing the original fixture race with another does not deliver reliable cancellation readiness.
```

1. **Strengths**

   - Both existing cancellation fixtures reuse one producer.
   - The held-writer control exercises real shell redirection. Restoring the original producer made it fail at `cli_group_test.go:144` with early readiness visibility.
   - Production lifetime and group cleanup code remains unchanged.

2. **Critical findings**

   - The publication-helper race above blocks shipping. Extend the readiness protocol and regression to cover helper completion.

3. **Important findings**

   - None.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Focused three-test `-count=30`: **failed**, as detailed above.
   - Existing cancellation tests `-count=100`: **failed** on pinned head; **passed** with the original producer.
   - Focused `-race -count=10`: **passed**.
   - `make test-affected-inventory` and range `git diff --check`: **passed**.
   - Temporary overlays were removed; repository files were unchanged. The full pre-close check was not independently verified here.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared producer covers both intended fixtures.
   - **ARCH-PURE: pass** — changes stay within the process/filesystem integration fixture.
   - **ARCH-PURPOSE: flag** — readiness must preserve reliable joined cancellation.
   - **ARCH-EVOLVE: pass** — no persisted format or reader/writer behavior changes.
   - README and atlas updates are unnecessary for this private fixture correction.

7. **Plan revision recommendations**

   - Add a `## Revisions` entry requiring readiness to follow publication-helper reaping, enumerate all three consumers above, and require a held post-rename regression alongside the existing held-write control.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0b940833-2243-4afc-92e9-38ea1170aefa","repository":"github.com/xianxu/pair","issue_id":"000869","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c9117ec18166e710","reviewed_head":"ccc3b57741a240945922e20a8d5ffdfddc054f14","slot":"pair:9","requested_at":"2026-10-24T15:39:38.694307Z"},"phase":"finished","pid":24419,"started_at":"2026-10-24T15:39:38.842149Z","finished_at":"2026-10-24T15:42:22.483079Z","session_id":"01a16e91-4fe6-73b1-905e-2f2c93ae217f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.616455125s
head: "ccc3b57741a240945922e20a8d5ffdfddc054f14"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T08:52:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 869 — Publish cancellation fixture PID readiness atomically |
| repo | pair |
| issue file | workshop/issues/000869-cli-fixture-pid-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | f9c06a92fe8d4b3fe1278f087ed85bfa60beffb6..be270eeb7be0d806d7aaa76fc18bf331e9ac69e7 |
| command | sdlc close --issue 869 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T08:52:50-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The readiness protocol now waits until the publication helper is reaped before allowing cancellation. Both cancellation fixtures and the conformance test share this protocol. BR-1 is verified addressed; no new findings.

Done-when: current — its publication, live-leader cancellation, and joined-cleanup clauses cover the revised protocol.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      cli_group_test.go:18 creates .ready with a shell builtin after mv returns; all three enumerated consumers use fixtureLeaderPIDReady. The publication-helper regression holds the post-rename interval. Reverting readiness to PID-file existence through a temporary overlay makes that regression fail at line 155. Production cleanup remains unchanged.
```

1. **Strengths**

   - Shared producer and readiness predicate cover all three consumers.
   - Held-write and held-helper controls exercise real shell behavior.
   - Existing process-death assertions and production cleanup strictness remain intact.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused three-test run, `-count=100`: passed.
   - Focused race run, `-count=20`: passed.
   - Full `agentfixture` package: passed.
   - `make test-affected-inventory` and range `git diff --check`: passed.
   - Reverted-readiness control: failed as expected.
   - Repository remained unchanged. Repository-wide pre-close checks were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY: pass** — producer and readiness observation are shared.
   - **ARCH-PURE: pass** — changes remain within the process/filesystem integration fixture.
   - **ARCH-PURPOSE: pass** — both publication intervals are covered, preserving reliable joined cancellation.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.
   - No README or atlas update is needed for this private fixture correction.

7. **Plan revision recommendations:** None; the existing revision records helper quiescence and enumerates all affected consumers.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"04369f17-1b51-4326-87d7-ed60cf67f3e0","repository":"github.com/xianxu/pair","issue_id":"000869","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9adaa9668207d032","reviewed_head":"be270eeb7be0d806d7aaa76fc18bf331e9ac69e7","slot":"pair:9","requested_at":"2026-10-24T15:51:36.06761Z"},"phase":"finished","pid":58904,"started_at":"2026-10-24T15:51:36.245143Z","finished_at":"2026-10-24T15:52:48.610398Z","session_id":"01a16e9c-4331-78e3-9e34-e93547d13cce","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.468090584s
head: "be270eeb7be0d806d7aaa76fc18bf331e9ac69e7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
