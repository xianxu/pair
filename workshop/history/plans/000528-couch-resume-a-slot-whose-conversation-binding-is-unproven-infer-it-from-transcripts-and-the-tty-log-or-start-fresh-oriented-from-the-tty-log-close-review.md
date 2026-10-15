# Boundary Review — pair#528 (whole-issue close)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4780a55286f5a171d64907cf03d9c12c7a2ce656..faa2056195acaec505029ab5fbf7e31431c2857f |
| command | sdlc close --issue 528 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T20:18:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches #528’s final approved M1 scope: listed-live attachment, bounded retries, background client attachment, independent messaging, and guarded dead-client repair. README and atlas document the changed behavior. No new findings; no previously disposed finding is reopened. The remaining recovery work is explicitly assigned to #556/#557.

```findings
{}
```

1. **Strengths**
   - Continue bypasses cold-launch and lifecycle-adoption prerequisites while rechecking the exact listed terminal.
   - Automatic broker reads use preview access without replaying pending journals; regression fixtures verify unchanged durable bytes.
   - Dead-client repair rechecks revision and exact process identity before publication.
   - Final fixture corrections use atomic publication and preserve existing assertion deadlines.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** All 15 exercised package suites passed, including couchcore (383.787s), couchcmd, couchtty, launcher, command entry points, and storage/native-evidence packages. Startup and pane-independent reconnect regressions each passed ten runs; shutdown passed twenty. Focused race checks passed. Production/docs whitespace checks passed. Live external conformance and the complete `make test` were not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared projection, attachment, and metadata-reader paths.
   - **ARCH-PURE — pass:** pure parsing/ranking/reducers remain separate from IO adapters.
   - **ARCH-PURPOSE — pass:** implementation follows the explicitly revised M1 scope.
   - **ARCH-MOCK — pass:** stateful terminal/process doubles exercise shared production seams; live conformance hooks remain.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, retries, scans, and attachment concurrency.
   - **ARCH-SECURE — pass:** persisted inputs receive structural, identity, and regular-file checks.
   - **ARCH-ORDER — pass:** tests exercise cancellation, changed observations, late completions, and effect-time revalidation.
   - **ARCH-FUNERAL — pass:** lock files have a fixed 256-stripe bound; retained recovery evidence uses existing ownership and retention.

7. **Plan revision recommendations:** None. Existing revisions record the scope split and superseded designs.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.893210958s
head: "faa2056195acaec505029ab5fbf7e31431c2857f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T21:09:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 192b688aa1884d9752c38a86a24b6e208c735086..d0862f78585dcca647c6494d6a8fd6942d7bf5e1 |
| command | sdlc close --issue 528 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T21:09:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

No defects found in the interdiff. Both conflict resolutions preserve the intended behavior, and the three new regressions fail when their corresponding corrections are removed. Prior dispositions remain unchanged. Live conformance remains partly unsuccessful, with failures also reproduced using the pinned base fixtures.

```findings
{}
```

1. **Strengths**
   - Cross-scope binding checks retain shared caching and detect changed claims (`artifactcollision.go:438`).
   - Queued attachments check cancellation before dispatch (`console_reattach.go:68`).
   - Socket transport preserves its original spelling while aliases retain the same canonical lease identity (`session_handoff_os.go:59`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Complete artifactpath suite and focused launcher, couchtty, and couchcore checks passed.
   - All three new regressions failed under temporary overlays removing their fixes.
   - Live warm recovery and intent-only mutation passed. Park ownership and checkpoint-registration failures also occurred with base fixtures; they are not attributed to this interdiff.
   - Full repository suite was not rerun. Checkout remains unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** conflict resolution reuses the shared index reader.
   - **ARCH-PURE — pass:** changes remain in integration boundaries without adding business logic to IO adapters.
   - **ARCH-PURPOSE — pass:** fixes preserve attachment and recovery contracts without expanding scope. No new user-facing surface requires README or atlas changes.

7. **Plan revision recommendations:** None. Appended revisions accurately describe the corrections.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.650990125s
head: "d0862f78585dcca647c6494d6a8fd6942d7bf5e1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T01:53:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0da5baebf64904d7489a28d12da1c76255184bbf..65517e605e94572a4a70be21eb3cd88313368cc1 |
| command | sdlc close --issue 528 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T01:53:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides’ intent across all four conflict resolutions. Warm timing reports actual attachment work, managed detach avoids unnecessary previews, and borrowed detach retains its read-only behavior. No new blocking defects or scope expansion found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Prior disposition retained; uncertainty policy is unchanged in this window.
  - id: BR-2
    disposition: addressed
    note: |
      Prior disposition retained; agent-generation authority is unchanged.
  - id: BR-3
    disposition: addressed
    note: |
      Prior disposition retained; admission read bounds are unchanged.
  - id: BR-4
    disposition: addressed
    note: |
      Prior disposition retained; coordination-file lifecycle is unchanged.
  - id: BR-5
    disposition: addressed
    note: |
      Prior disposition retained; capture and configuration readers are unchanged.
  - id: BR-6
    disposition: addressed
    note: |
      Prior disposition retained; native launch ownership resolution is unchanged.
  - id: BR-7
    disposition: addressed
    note: |
      Prior disposition retained; recovery primary-action routing is unchanged.
  - id: BR-8
    disposition: addressed
    note: |
      Prior disposition retained; native conformance parsing is unchanged.
  - id: BR-9
    disposition: addressed
    note: |
      Positive-listing admission and final-effect rechecks survive launcher conflict resolution; selected-attach tests pass.
  - id: BR-10
    disposition: addressed
    note: |
      Prior documentation disposition retained; this window adds no new command, flag, or keybinding.
  - id: BR-11
    disposition: addressed
    note: |
      Prior disposition retained; the changed startup fixture compiles with the current beginConsole signature.
  - id: BR-12
    disposition: addressed
    note: |
      Read-only detach snapshot and borrowed preview remain; journal-preservation regression passes.
  - id: BR-13
    disposition: addressed
    note: |
      Prior disposition retained; contextual broker artifact readers are unchanged.
  - id: BR-14
    disposition: addressed
    note: |
      Selected attachment retains migration bypass and positive-listing checks; affected attachment tests pass.
  - id: BR-15
    disposition: addressed
    note: |
      Production-symbol and artifact-classification guards pass at the pinned head.
```

1. **Strengths**
   - [park.go:189](cmd/internal/couchcore/park.go:189) retains the shared owner-observation scope alongside the read-only detach snapshot.
   - [console_reattach.go:41](cmd/internal/couchtty/console_reattach.go:41) preserves shared-pass ownership without painting before Run; the new regression verifies arm-once behavior.
   - [lifecycle.go:133](cmd/internal/launcher/lifecycle.go:133) preserves positive-listing verification immediately before attachment.
   - The borrowed-detach regression checks unchanged journal and record bytes plus exact helper-only signaling.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Targeted tests passed across couchcore, couchtty, launcher, and couchcmd. Broader Leave/Detach/warm-attachment regressions passed, as did production-symbol and artifact-classification guards. Full suite, live conformance, and mutation checks were not rerun during this review.

6. **Architecture:** ARCH-DRY passes: timing reuses `launchSteps`, and both detach entry points share the optional-preview predicate. ARCH-PURE passes: changes remain in existing integration boundaries with injected dependencies. ARCH-PURPOSE passes: conflict resolutions preserve attachment authority and read-only behavior. Atlas documents the integration; no new README-facing surface appears.

7. **Plan revisions:** None required. The appended entries describe integration corrections without changing the accepted recovery scope.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.445840375s
head: "65517e605e94572a4a70be21eb3cd88313368cc1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T02:48:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0bf9315528f7b9ffbc2ecf247a863bef298ed399..15bf3855722960fd8a97ddd2f4bf52395e7be08b |
| command | sdlc close --issue 528 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T02:48:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The conflict resolution preserves both intents: inventory stays blocked until the header-only readiness check rejects the empty menu, then Enter waits for the current “Resume from disk” label. No new defects or scope expansion found in this interdiff.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Prior disposition retained; observation policy is unchanged.
  - id: BR-2
    disposition: addressed
    note: |
      Prior disposition retained; process-generation checks are unchanged.
  - id: BR-3
    disposition: addressed
    note: |
      Prior disposition retained; admission readers are unchanged.
  - id: BR-4
    disposition: addressed
    note: |
      Prior disposition retained; coordination-file lifecycle is unchanged.
  - id: BR-5
    disposition: addressed
    note: |
      Prior disposition retained; capture and configuration readers are unchanged.
  - id: BR-6
    disposition: addressed
    note: |
      Prior disposition retained; native ownership resolution is unchanged.
  - id: BR-7
    disposition: addressed
    note: |
      Prior disposition retained; recovery routing is unchanged.
  - id: BR-8
    disposition: addressed
    note: |
      Prior disposition retained; native conformance parsing is unchanged.
  - id: BR-9
    disposition: addressed
    note: |
      Prior disposition retained; attachment admission and execution are unchanged.
  - id: BR-10
    disposition: addressed
    note: |
      Prior documentation disposition retained; no user-facing surface changes.
  - id: BR-11
    disposition: addressed
    note: |
      Prior disposition retained; startup integration is unchanged.
  - id: BR-12
    disposition: addressed
    note: |
      Prior disposition retained; observation-only store access is unchanged.
  - id: BR-13
    disposition: addressed
    note: |
      Prior disposition retained; contextual broker readers are unchanged.
  - id: BR-14
    disposition: addressed
    note: |
      Prior disposition retained; automatic attachment and dead-client repair are unchanged.
  - id: BR-15
    disposition: addressed
    note: |
      Production-symbol and artifact-classification guards pass at the pinned head.
```

1. **Strengths**
   - [cold_resume_origins_test.go:174](cmd/internal/couchcmd/cold_resume_origins_test.go:174) retains deterministic inventory blocking and the header-only negative assertion.
   - [cold_resume_origins_test.go:195](cmd/internal/couchcmd/cold_resume_origins_test.go:195) waits for the label emitted by the current renderer.
   - Both switcher and remote paths retain cold-launch registration assertions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Cold-resume test passed ten repetitions, exercising both origins. Targeted continuity tests and production-symbol/artifact-classification guards passed. Diff whitespace check passed; checkout remains clean. Full suite, live conformance, and mutation checks were not rerun.

6. **Architecture:** **ARCH-DRY passes:** no duplicated production behavior introduced. **ARCH-PURE passes:** injected inventory seam preserves deterministic synchronization. **ARCH-PURPOSE passes:** resolution retains both the scheduling regression and current recovery label. No new architectural or user-facing surface requires atlas/README updates.

7. **Plan revisions:** None required; the appended revision accurately describes the test-only resolution.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.550438083s
head: "15bf3855722960fd8a97ddd2f4bf52395e7be08b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
