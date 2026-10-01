# Boundary Review — pair#373 (whole-issue close)

| field | value |
|-------|-------|
| issue | 373 — Isolate Couch pane latency under output pressure |
| repo | pair |
| issue file | workshop/issues/000373-couch-output-pressure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 31bc6185faeb880f24040063da1f2114ff2a5dde..2e1316a790335102b2250fe6803cedf4921a1f8a |
| command | sdlc close --issue 373 |
| reviewer | codex |
| timestamp | 2026-10-01T15:57:22-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The experiment exercises production delivery and independently measures receipt, display and switcher latency. All 24 trials passed without reproducing a selective stall. However, the focused race test failed during teardown, and the harness needs stronger deadline and recovery checks before its bounded-experiment contract is satisfied.

1. **Strengths**

   - Real PTY receipt timestamps precede ACK output, separating input delivery from rendering.
   - Switcher opening and dismissal use production input dispatch with populated inventory.
   - Fake and PTY workloads share one producer.
   - The recorded conclusion appropriately limits what this negative result establishes; atlas documentation covers the experiment.

2. **Critical findings**

   - **Teardown races with emulator reads** — `cmd/internal/couchtty/terminal_pressure_test.go:158,203`. The drain reads `Emulator.closed` concurrently with `Close` writing it (`third_party/vt/emulator.go:288,301`). This failed `go test -race ./cmd/internal/couchtty -run '^TestCouchPressureControl$' -count=1 -timeout=30s`. Use a concurrency-safe reply-drain/close mechanism and verify teardown under the race detector. This single shared drain/close pair affects every trial.

3. **Important findings**

   - **Trial deadlines do not bound blocking operations** — `terminal_pressure_test.go:152,333,371,411,415`. The trial context has no deadline, synchronous pipe writes can block before another timeout check, and polling deadlines cannot interrupt blocked predicates. Separately renewed recovery waits also exceed the documented single five-second recovery budget. Use one absolute trial/recovery deadline, cancellation that releases blocked operations, and joined helpers. Cover stalled input, observation and recovery paths. **ARCH-PURPOSE**.
   - **Recovery can finish before PTY output is ingested** — `terminal_pressure_test.go:419–447`. The side-channel completion notification proves the child finished writing; `FlushOutput` only drains already queued batches. Unread PTY bytes may remain. The final assertion accepts any `PROGRESS` value from child zero, while aggregate emitted/ingested counts are merely logged. Require completion evidence for every child before measuring recovery, then verify final presentation. Test delayed trailing output.
   - **README discovery is missing** — `README.md:1076`. The range introduces the runnable `PAIR_COUCH_PRESSURE=1` experiment but documents it only in atlas. Add its invocation and limitations alongside existing terminal-soak instructions. This is the only new operator-facing invocation family in the window; the child-helper variable is internal.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Full pressure matrix plus ordinary control: passed in 51.905s.
   - All 24 matrix trials reported no selective stall and no censored ACKs.
   - Focused race control: failed with the teardown race above.
   - Pinned-range `git diff --check`: passed.
   - Missing adversarial coverage: blocked trial operations and completion arriving before trailing PTY ingestion.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared producer and existing host/child seams avoid parallel implementations.
   - **ARCH-PURE — pass:** this is an integration experiment; it introduces no production business logic or misleading PURE classification.
   - **ARCH-PURPOSE — flag:** bounded execution and verified recovery are part of the investigation’s stated purpose, and need enforcement under stalled conditions.

7. **Plan revision recommendations**

   Append a timestamped `## Revisions` entry specifying the absolute recovery deadline, cancellation/join strategy and per-child completion evidence. Update verification evidence after fixing the race and rerunning the controls.

```findings
findings:
  - id: new
    severity: Critical
    family: concurrent-resource-shutdown
    title: |
      Emulator teardown races with the reply drain
    detail: |
      terminal_pressure_test.go:158,203 starts io.Copy on the emulator and closes it before joining the reader; Emulator.Read and Close access closed without synchronization. The focused race control failed at emulator.go:288,301. This shared pair is the only instance in the window and affects every trial. Use concurrency-safe draining/shutdown and verify with the race detector.
  - id: new
    severity: Important
    family: experiment-deadline-enforcement
    title: |
      Blocking trial operations escape the promised deadlines
    detail: |
      terminal_pressure_test.go:152 uses cancellation without a deadline; input writes at 371,411,415 and synchronous observations at 333–396 can prevent timeout checks and cleanup from running. Recovery waits at 414,418,419,427 receive separate budgets rather than one five-second deadline. ARCH-PURPOSE: enforce an absolute deadline across these paths, release blocked operations on cancellation, join helpers and test stalled execution.
  - id: new
    severity: Important
    family: completion-evidence-before-success
    title: |
      Recovery is reported without proving all PTY output arrived
    detail: |
      terminal_pressure_test.go:419–447 treats side-channel producer completion plus queued-publication flush as output completion, although unread PTY bytes can remain. Final progress checks only child zero and accepts any sequence; emitted and ingested totals are only logged at 453. Require per-child terminal completion evidence and final presentation before reporting recovery, with a regression case delaying trailing PTY output.
  - id: new
    severity: Important
    family: readme-surface-discovery
    title: |
      README omits the new pressure experiment invocation
    detail: |
      atlas/couch.md documents PAIR_COUCH_PRESSURE=1 and TestCouchOutputPressure, but README.md is unchanged despite its existing terminal-testing section at 1076. Add the command and scope limitations there. This is the sole new operator-facing invocation family; PAIR_COUCH_PRESSURE_CHILD is an internal helper.
```

---

## Re-review — 2026-10-01T16:06:11-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 373 — Isolate Couch pane latency under output pressure |
| repo | pair |
| issue file | workshop/issues/000373-couch-output-pressure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 31bc6185faeb880f24040063da1f2114ff2a5dde..c5feb527c8734f06491ea2c30ab74d8dd86efc99 |
| command | sdlc close --issue 373 |
| reviewer | codex |
| timestamp | 2026-10-01T16:06:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The experiment and all 24 trials pass, and the shutdown, deadline, and README corrections are supported by evidence. BR-3 remains open: its regression still passes when the completion checks are removed, so it does not protect the promised rejection of premature recovery.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      terminal_pressure_test.go:230–251 closes the reply pipe and joins readers/writers before Emulator.Close. Focused race tests passed three repetitions; a scratch mutation restoring early Emulator.Close triggered race failures.
  - id: BR-2
    disposition: addressed
    note: |
      Operations use pressureAwait, recovery shares window-end-plus-five-second cancellation, and cleanup closes resources before joining workers. The stalled-operation regression passed; removing pressureAwait's cancellation branch made it time out.
  - id: BR-3
    disposition: not-addressed
    note: |
      terminal_pressure_test.go:489–498 checks marker absence and releases trailing output before invoking recovery checks. Removing lines 500–524 in a scratch Go overlay left TestCouchPressureTrailingPTYOutput passing three repetitions, including one reporting ingested_raw=2061 instead of the required 2087 bytes. The implementation adds completion checks, but the required fail-without-fix regression is missing. Exercise the actual recovery operation while trailing output remains withheld and assert it cannot report success.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:1090–1101 now documents the opt-in command and limitations. The invocation matches TestCouchOutputPressure's environment guard and 2×4×3 matrix; atlas/couch.md:2207–2218 documents the same surface.
```

1. **Strengths**
   - Independent receipt timestamps distinguish child input delivery from endpoint and displayed ACK latency.
   - Menu input occurs during pressure through production dispatch.
   - Shutdown ordering now joins emulator users before closing its unsynchronized state.
   - README and atlas accurately limit what the negative experiment establishes.

2. **Critical findings:** None remaining.

3. **Important findings:** **BR-3 remains open**, at [terminal_pressure_test.go:489](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/couchtty/terminal_pressure_test.go:489). The assertion proves that the marker is absent, not that recovery refuses success while it is absent. This family has one shared recovery path, covering all trials: per-child markers, exact byte accounting, publication completion, and selected-screen presentation at lines 500–524. Test that shared operation independently of the trailing-output release; verify that removing its safeguards makes the regression fail.

4. **Minor findings:** None.

5. **Test coverage notes:** The 24-trial matrix passed in 49.936s. Focused control, stalled-operation, and trailing-output tests passed three repetitions under `-race`. Shutdown and deadline mutations failed as expected; the completion-check mutation incorrectly passed three times. Pinned-range `git diff --check` passed. Repository files were unchanged; mutations used temporary overlays.

6. **Architectural notes:** **ARCH-DRY: pass**—existing host/child seams and shared trial logic are reused. **ARCH-PURE: pass**—integration work stays in the experimental fixture with injected transports. **ARCH-PURPOSE: flag, BR-3**—the explicit premature-completion regression contract is not yet demonstrated.

7. **Plan revision recommendation:** Append a timestamped `## Revisions` entry correcting the claim that the trailing-output regression rejects premature recovery. Record the mutation evidence and the replacement test’s fail-without-fix result before closing.

---

## Re-review — 2026-10-01T16:10:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 373 — Isolate Couch pane latency under output pressure |
| repo | pair |
| issue file | workshop/issues/000373-couch-output-pressure.md |
| boundary | whole-issue close |
| milestone | — |
| window | 31bc6185faeb880f24040063da1f2114ff2a5dde..5dccaa0004fc177dd6f4636a01be5d2855186d41 |
| command | sdlc close --issue 373 |
| reviewer | codex |
| timestamp | 2026-10-01T16:10:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the issue’s Spec and Done-when criteria. BR-3 now has a regression that fails when recovery checks are removed. The full package race suite and all 24 pressure trials passed; no selective stall reproduced. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      terminal_pressure_test.go:230–250 joins emulator readers and writers before closing emulator state. The full package race suite passed.
  - id: BR-2
    disposition: addressed
    note: |
      Blocking trial operations use tracked cancellation; recovery shares the window deadline. The stalled-operation regression passed in the full race suite.
  - id: BR-3
    disposition: addressed
    note: |
      terminal_pressure_test.go:491–554 shares recovery checks between trials and the withheld-output regression. It checks every child's completion marker, exact byte accounting, publication flush and selected final presentation. A scratch overlay replacing recovery with immediate success failed with “got <nil>, want deadline exceeded”; the unchanged regression passed.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:1090–1101 documents the opt-in invocation and limitations, matching the environment guard and 24-trial matrix. atlas/couch.md documents the experiment.
```

1. **Strengths**
   - Independent receipt, endpoint, display and menu observations distinguish the relevant latency paths.
   - Recovery proves terminal-stream completion rather than trusting producer completion.
   - Shared fake/PTY trials exercise production delivery without production changes.
   - Documentation accurately limits the negative result’s implications.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Full Couch package race suite passed: **26.009s**.
   - All **24 pressure trials** passed: **49.993s**, with no censored ACKs or selective stalls.
   - Recovery-removal mutation failed as expected: **2.127s**.
   - Pinned-range `git diff --check` passed. The matrix required a temporary Go cache after sandbox denial of the default cache.

6. **Architecture**
   - **ARCH-DRY — pass:** Reuses existing host/PTY seams and shares workload and recovery logic.
   - **ARCH-PURE — pass:** IO remains in the test integration harness with injected transport and host behavior; no production business logic added.
   - **ARCH-PURPOSE — pass:** Delivers the bounded investigation, real-PTY corroboration, completion evidence and documented limitations.

7. **Plan revisions:** None required. Existing revisions match the implemented corrections.
