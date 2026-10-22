# Boundary Review — pair#778 (whole-issue close)

| field | value |
|-------|-------|
| issue | 778 — flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub |
| repo | pair |
| issue file | workshop/issues/000778-flake-testbarecouchinstalledcommand-pty-launches-and-attaches-current-hub.md |
| boundary | whole-issue close |
| milestone | — |
| window | d2a4bb5bbfc1a14c205c996af8b93cb5afacdbd7..d729188af8159560db1bfbf0d610fe0886eb5346 |
| command | sdlc close --issue 778 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:34:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change fixes the fixture-name collision through both constructors while preserving production allocation, launch admission, and timeout behavior. Independent targeted checks pass, and the recorded full-suite receipt and matching raw-log hash confirm a pass on identical executable source.

Done-when: current — it covers both constructors, allocator continuity, PTY expectations, seed safety, lint enforcement, and recorded verification.

```findings
{}
```

1. **Strengths**
   - Both constructors reuse `testpair.SeedCouchIdentity`; no parallel seeding implementation.
   - Exclusive creation preserves existing authorities, with explicit seed bounds and permanent sequence exhaustion.
   - Constructor regressions exercise the real allocator; the installed PTY assertion derives its expected tag from allocated C.
   - Atlas documents fixture identity ownership and the syntax guard’s limitations.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None. Whitespace diagnostics are confined to retained evidence logs.

5. **Test coverage notes**
   - Passed affected packages: `testpair`, `testcouch`, `repolint`, and `artifactpath`.
   - Passed installed PTY, pipe refusal, constructor isolation, focused race checks, and `make test-affected-inventory`.
   - Verified recorded suite attempt `54cda992757ffb884f353a200db8f185`: exit 0, matching log SHA256. Only issue notes changed afterward.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared initializer serves both enumerated consumers.
   - **ARCH-PURE: pass** — seed calculation and AST matching are pure; filesystem work remains in fixture setup.
   - **ARCH-PURPOSE: pass** — both authority constructors are covered without weakening collision refusal.
   - **ARCH-EVOLVE: pass** — no production format change or existing-record rewrite is introduced. Atlas is updated; no new user-facing surface requires README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b39404fc-3a1a-43fd-b3ab-4a900c1fd7b8","repository":"github.com/xianxu/pair","issue_id":"000778","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e35a90fffc6705a3","reviewed_head":"d729188af8159560db1bfbf0d610fe0886eb5346","slot":"pair:5","requested_at":"2026-10-22T03:32:29.228546Z"},"phase":"finished","pid":4418,"started_at":"2026-10-22T03:32:29.500704Z","finished_at":"2026-10-22T03:34:16.220547Z","session_id":"01a161aa-dcca-7703-aa3f-f9067a89a17b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.731225333s
head: "d729188af8159560db1bfbf0d610fe0886eb5346"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T23:55:55-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 778 — flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub |
| repo | pair |
| issue file | workshop/issues/000778-flake-testbarecouchinstalledcommand-pty-launches-and-attaches-current-hub.md |
| boundary | whole-issue close |
| milestone | — |
| window | 654e4cf0ca31c8f7aed298d3e0b5a6737b821206..b9ea92045914dc1017e4e59ab473c7c73c015fe8 |
| command | sdlc close --issue 778 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T23:55:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation passes the targeted review: both constructors share exclusive identity seeding, real allocation remains intact, and production admission and timeouts are unchanged. Whole-issue completion remains blocked by the required full-suite result on the integrated source: its receipt is still inflight.

Done-when: current — it covers both constructors, allocator continuity, seed safety, PTY behavior, lint enforcement, and full-suite verification.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: completion-requires-verification
    title: |
      Required full-suite pass is unavailable for the integrated source
    detail: |
      workshop/issues/000778-flake-testbarecouchinstalledcommand-pty-launches-and-attaches-current-hub.md:51 marks verification complete, but line 134 records the replacement suite as queued. Checking the host test-stats/attempts/fa50b4670ad83418945dc02f84c68676.json found no completed receipt; its inflight directory exists. The verified older pass, 54cda992757ffb884f353a200db8f185, covers f025dba4 before the operator-test integration. Family instances in this window are the checked Plan verification row and the outstanding final-source acceptance clause at line 45. Complete the existing run and provide its successful receipt plus matching raw-log hash for source d7198a27, whose executable source matches this head; otherwise fix failures and reverify.
```

1. **Strengths**

   - Both authority constructors reuse `SeedCouchIdentity`; the collision fix covers the enumerated family.
   - Exclusive creation preserves existing files, while bounds reject invalid seeds and allocator exhaustion.
   - Constructor tests exercise the real allocator; the installed PTY assertion derives its expected tag from C.
   - Atlas documents ownership and the AST guard’s limitations. No new user-facing command requires README changes.

2. **Critical findings:** None.

3. **Important findings:** The final-source full-suite evidence above. Finish the retained run and record its result before claiming completion.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed affected packages (`testpair`, `testcouch`, `repolint`, `artifactpath`), installed PTY and pipe-refusal tests, constructor isolation, operator executable tests, focused seed race tests, and affected-reader inventory. Verified the older successful suite’s raw-log hash and retained native/compile-failure hashes. These establish useful evidence but do not replace the outstanding integrated suite.

6. **Architectural notes**

   - **ARCH-DRY: pass** — one helper serves both consumers.
   - **ARCH-PURE: pass** — seed calculation and AST matching remain separate from fixture IO.
   - **ARCH-PURPOSE: flag** — implementation scope is delivered; required completion evidence remains outstanding.
   - **ARCH-EVOLVE: pass** — no production format change or existing-record rewrite is introduced.

7. **Plan revision recommendations**

   Add a `## Revisions` entry marking final integrated verification pending, then replace that status with the completed receipt and log hash when available.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"78c86080-faf8-479a-9818-ca2270bf3a63","repository":"github.com/xianxu/pair","issue_id":"000778","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f8f9caa8172f3cde","reviewed_head":"b9ea92045914dc1017e4e59ab473c7c73c015fe8","slot":"pair:5","requested_at":"2026-10-22T06:53:14.861163Z"},"phase":"finished","pid":37249,"started_at":"2026-10-22T06:53:15.026792Z","finished_at":"2026-10-22T06:55:53.49976Z","session_id":"01a16262-a94e-7e81-a352-341b12238112","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.054906875s
head: "b9ea92045914dc1017e4e59ab473c7c73c015fe8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T00:15:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 778 — flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub |
| repo | pair |
| issue file | workshop/issues/000778-flake-testbarecouchinstalledcommand-pty-launches-and-attaches-current-hub.md |
| boundary | whole-issue close |
| milestone | — |
| window | 654e4cf0ca31c8f7aed298d3e0b5a6737b821206..5dc34bb27415a64474ef501fc08bf27831507d86 |
| command | sdlc close --issue 778 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T00:15:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation fixes fixture identity collisions through both constructors while preserving production launch admission. BR-1 is addressed: the completed host receipt and matching raw log verify the integrated source, and subsequent changes contain no executable code. No blocking findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Host attempt fa50b4670ad83418945dc02f84c68676 matches the committed integrated-suite-pass.json: source d7198a27791bfac191552e79b19ea45ed6e93f40 passed with exit 0 in 768.631 seconds. Independently hashing its host suite.log produced 38c6ab4eddde369a20be795981ec0cd8e40f1f4b74ebf1ef89e02eaa4d7e38ba, matching the receipt. The tested source differs from this head only in process/docs artifacts; the checked verification row and final-source acceptance clause are now supported.
```

Done-when: current — it covers both constructors, real allocation, derived PTY expectations, validation, concurrency, lint enforcement, and recorded verification.

1. **Strengths**

   - Both constructors reuse the exclusive initializer, covering the identified collision family (`cmd/couch/main_test.go:244`, `cmd/internal/testcouch/environment.go:72`).
   - Tests exercise real allocation, stable C with advancing N/M, existing-authority refusal, exhaustion, and concurrent initialization.
   - Atlas documents the helper’s scope, PID-reuse limitation, and syntax guard accurately.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently passed focused seed/constructor/lint tests, seed race tests, installed PTY and pipe-refusal tests, operator executable tests, full repolint and artifactpath packages, and affected-reader inventory. Verified the recorded full-suite receipt against its host raw log. Retained native evidence covers unchanged helper and constructor code.

6. **Architectural notes**

   - **ARCH-DRY: pass** — both consumers share one initializer.
   - **ARCH-PURE: pass** — seed calculation and AST matching remain separate from filesystem setup.
   - **ARCH-PURPOSE: pass** — both identified authority constructors are covered; admission and readiness budgets remain unchanged.
   - **ARCH-EVOLVE: pass** — no production format or rewrite behavior changes; initialization exclusively creates fresh private state.

   Atlas updates cover the internal surface. No new user-facing command, flag, or configuration requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"df78efbf-71a4-44f9-9c20-0853ac4e7a37","repository":"github.com/xianxu/pair","issue_id":"000778","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1a43b9f461623702","reviewed_head":"5dc34bb27415a64474ef501fc08bf27831507d86","slot":"pair:5","requested_at":"2026-10-22T07:13:21.395777Z"},"phase":"finished","pid":9998,"started_at":"2026-10-22T07:13:21.703339Z","finished_at":"2026-10-22T07:15:26.879427Z","session_id":"01a16275-1420-75c3-be40-09ec2b56c9be","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.151386416s
head: "5dc34bb27415a64474ef501fc08bf27831507d86"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
