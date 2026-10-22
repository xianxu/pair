# Boundary Review — pair#727 (whole-issue close)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8620374573f7a42310ffe6348d7946d5326dd15f..4617a1608cf2dee3fdd037436587f2806474d031 |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T09:23:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Cell repair, bounded diagnostics, geometry waiting, and teardown tests are substantial and pass the affected checks. Two behavior gaps block shipping: input failures can still close the pane without shell recovery, and missing-pane recovery steals focus despite the documented shortcut contract. Native conformance coverage also needs to exercise the new creation path.

```findings
findings:
  - id: new
    severity: Critical
    family: recovery-outcome-classification
    title: |
      Input failures are classified as normal completion and bypass shell recovery
    detail: |
      cmd/internal/termcmd/run.go:430-432 discards the input pump's outcome and calls stopLocked(nil); line 457 then classifies mux.done as RecoveryChildExit. A decoder failure, such as invalid UTF-8, only displays a notice before returning, while read errors are discarded outright. With successful cleanup, recovery receives nil and exits without a fallback shell. Return a typed pump outcome, preserve unexpected failures, and test decoder failure and non-EOF read failure through runShellRecovery. This violates the expanded recovery contract and ARCH-PURPOSE/ARCH-ORDER.
  - id: new
    severity: Critical
    family: shortcut-focus-preservation
    title: |
      Recreating the missing terminal moves focus away from the invoking pane
    detail: |
      cmd/internal/layoutcmd/layoutcmd.go:252 creates the pane without --no-focus, contrary to README.md:174's explicit focus-preserving contract. Installed Zellij 0.45.1 documents --no-focus as the option that prevents client focus changes; the new fake likewise marks this pane focused. Add --no-focus and assert that the invoking pane/client retains focus after recovery, so subsequent typing stays in the draft or agent. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: external-fake-conformance
    title: |
      The new native pane-creation behavior has no live conformance coverage
    detail: |
      cmd/internal/layoutcmd/terminal_recovery_test.go:46 exercises only the Python fake and exact argv equality. The range adds no native conformance test for the new --tab-id/--near-current-pane creation path, and existing live creation coverage uses a different downward-split command. Add a disposable native test through the production routing seam that verifies visible creation in the invoking tab and preserved client focus, and include it in scheduled conformance. ARCH-MOCK.
```

1. **Strengths**

   - Backend repair covers screen and history snapshots while preserving strict direct-frame validation and backend contents.
   - Generated row tests check column preservation and safe wide-cell handling; the presenter oracle verifies continued rendering and input.
   - Recovery tests cover held teardown, termination precedence, closed signal channels, and an actual interactive shell handoff.
   - README and atlas updates cover the introduced recovery surface.

2. **Critical findings**

   - [run.go:430](/Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa/cmd/internal/termcmd/run.go:430): propagate input completion reasons into recovery classification.
   - [layoutcmd.go:252](/Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa/cmd/internal/layoutcmd/layoutcmd.go:252): preserve focus during pane recreation.

3. **Important findings**

   - [terminal_recovery_test.go:46](/Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa/cmd/internal/layoutcmd/terminal_recovery_test.go:46): qualify native creation semantics beyond the fake’s argv assertions.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test` for termcmd, layoutcmd, terminal, and workbenchshortcut.
   - `go test -race` for termcmd, layoutcmd, hostty, ptychild, terminalcapture, and zellijtest.
   - Pinned-range `git diff --check`.

   The findings follow reachable code paths; the green tests omit their failure conditions. No full-suite or operator acceptance result was independently established. The checkout remains unchanged.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared terminal command and existing selectors are reused.
   - **ARCH-PURE — pass:** repair and recovery transitions have direct pure tests; IO uses boundary seams.
   - **ARCH-PURPOSE — flag:** input-failure recovery and shortcut focus preservation remain incomplete.
   - **ARCH-MOCK — flag:** native creation semantics lack conformance coverage.
   - **ARCH-CONSTRAINTS — pass:** launch waiting and evidence scanning are bounded.
   - **ARCH-SECURE — pass:** repaired publications retain strict safety admission; diagnostics omit raw hyperlinks.
   - **ARCH-ORDER — flag:** the transition model cannot enforce recovery when the input shell discards the originating failure.
   - **ARCH-FUNERAL — pass:** diagnostics reuse managed retention; children and producers are joined.
   - **ARCH-EVOLVE — pass:** no existing persisted format is rewritten by this change.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - Input outcomes—child exit, EOF/disconnection, decoder failure, read failure, and cancellation—and their recovery authority.
   - Focus preservation as a recovery invariant, with stateful-fake assertions and scheduled native conformance through the production seam.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"501abd18-d14e-459a-bb1b-6bb3c09076dd","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bcb8e3e2dacdfffa","reviewed_head":"4617a1608cf2dee3fdd037436587f2806474d031","slot":"pair:7","requested_at":"2026-10-20T16:20:23.152605Z"},"phase":"finished","pid":32221,"started_at":"2026-10-20T16:20:23.247057Z","finished_at":"2026-10-20T16:23:15.227509Z","session_id":"01a15a1d-2b41-70e2-b7e6-f3168ac30bab","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.408227584s
head: "4617a1608cf2dee3fdd037436587f2806474d031"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T09:32:32-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8620374573f7a42310ffe6348d7946d5326dd15f..97c32eaad2245af63532d100ac67e53283b268eb |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T09:32:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised Spec and Plan. All three prior findings are addressed with executable regression evidence. No new blocking findings emerged. README and atlas updates cover the changed behavior. Operator acceptance remains deferred to the next natural relaunch, as explicitly approved in the tracker.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      run.go:430-432 propagates typed pump failures into recovery. TestRecoveryOwnsInputPumpEnding passes; disabling propagation in a disposable checkout reproduces missing errors and zero handoffs for decoder and non-EOF read failures.
  - id: BR-2
    disposition: addressed
    note: |
      layoutcmd.go:252 includes --no-focus. Fake and native tests preserve invoking-pane focus; removing the flag makes both fake regression cases fail. The native negative control also demonstrates redirected keyboard input.
  - id: BR-3
    disposition: addressed
    note: |
      TestTerminalRecoveryZellijConformance passed against Zellij 0.45.1 through production routing, verifying same-tab visible creation, actual client focus and foreign-tab isolation. The scheduled native runner requires its execution; all eight runner proof tests passed.
```

1. **Strengths**

   - Backend repair covers screen and history snapshots while keeping direct Frame validation strict (`terminal/cell_degrade.go:25`, `terminal/endpoint.go:260`).
   - Recovery preserves failure causes, requires completed cleanup, and suppresses handoff after termination or transport loss (`termcmd/recovery.go:192`).
   - Missing-pane creation requires confirmed same-tab absence and preserves focus (`layoutcmd/layoutcmd.go:235`).
   - Regression tests distinguish real behavior: malformed cells reach the presenter, fallback accepts commands over a PTY, and native focus is checked using keyboard input.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed nine affected packages, including terminal, termcmd, layoutcmd, terminalcapture, ptychild, hostty and artifactpath.
   - Focused recovery and routing tests passed under the race detector.
   - Both behavior fixes produced semantic failures when disabled in disposable checkouts; those checkouts were removed.
   - Native conformance passed with `TMPDIR=/tmp`; the default long temporary path exceeded Zellij’s socket-path limit.
   - The repository-wide recorded suite was not rerun during this review. The reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared launch command serves split and recovery; existing conformance checks pin layout agreement.
   - **ARCH-PURE — pass:** Repair and recovery decisions are directly testable pure logic; IO remains in boundary callers.
   - **ARCH-PURPOSE — pass:** Cell resilience, geometry waiting, shell fallback and missing-pane recovery are delivered.
   - **ARCH-MOCK — pass:** Stateful fake coverage is paired with scheduled native conformance.
   - **ARCH-CONSTRAINTS — pass:** Geometry waiting, evidence sampling and repair work have explicit bounds and tests.
   - **ARCH-SECURE — pass:** Untrusted cells are repaired before publication; diagnostics exclude raw hyperlinks and control sequences.
   - **ARCH-ORDER — pass:** Production handoff uses the transition function; controlled tests exercise termination during teardown.
   - **ARCH-FUNERAL — pass:** Diagnostics reuse managed retention; producers and children are joined before handoff.
   - **ARCH-EVOLVE — pass:** Existing sidecar formats remain unchanged; diagnostic records introduce no lossy rewrite path.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"101cd5b0-82e0-43eb-8b2a-438e0085f4a9","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d84b248216367ff5","reviewed_head":"97c32eaad2245af63532d100ac67e53283b268eb","slot":"pair:7","requested_at":"2026-10-20T16:29:54.970465Z"},"phase":"finished","pid":258,"started_at":"2026-10-20T16:29:55.162863Z","finished_at":"2026-10-20T16:32:32.169783Z","session_id":"01a15a25-e5d3-70a3-9842-f8ebd5f87dbf","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.89223075s
head: "97c32eaad2245af63532d100ac67e53283b268eb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-20T09:48:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4573ad6cba12d99fc27285255bde6cecd4062d6e..4db6b859d8c617b65dbe087214fde3d242c741a2 |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T09:48:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly repairs a stale shell-test assertion after the terminal command moved to its shared owner. Production behavior, Spec, and Plan remain unchanged. The corrected test passes; restoring the previous assertion reproduces the documented failure. No new defects found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Prior correction remains unchanged; TestRecoveryOwnsInputPumpEnding passes in this review.
  - id: BR-2
    disposition: addressed
    note: |
      The --no-focus correction remains unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes both cases.
  - id: BR-3
    disposition: addressed
    note: |
      Previously accepted native conformance coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.
```

1. **Strengths**
   - `tests/term-pane-shortcuts-test.sh:344` checks the shared command owner and both consumers.
   - The behavioral split-command equality assertion remains intact at line 140.
   - Tracker updates accurately record the failed suite without claiming a full pass.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Complete shortcut shell test, focused Go regressions, eight runner proof tests, and diff whitespace checks passed. Restoring the old assertion produced the expected failure. Temporary snapshot removed; checkout clean. Full suite not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Assertion follows the existing shared authority.
   - **ARCH-PURE — pass:** No business logic or IO boundaries changed.
   - **ARCH-PURPOSE — pass:** Repairs the stale check while retaining behavioral validation.
   - **ARCH-EVOLVE — pass:** No persisted-format contract changed.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"570f3ede-e1f2-40bd-af09-c6bb20481737","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a8a85d36a450fe8d","reviewed_head":"4db6b859d8c617b65dbe087214fde3d242c741a2","slot":"pair:7","requested_at":"2026-10-20T16:47:05.529827Z"},"phase":"finished","pid":65843,"started_at":"2026-10-20T16:47:05.679141Z","finished_at":"2026-10-20T16:48:26.617976Z","session_id":"01a15a35-9f49-7f82-aef8-c7bc1a807cdd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.191946708s
head: "4db6b859d8c617b65dbe087214fde3d242c741a2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T10:15:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | cf7a473df1a2a5441eb065d5f3940f90c79cdd82..7bda7379dddfada81f0dd81cd502fecec5c74201 |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T10:15:50-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly updates the exhaustive effect inventory without weakening admission checks or startup exemptions. Production behavior, Spec, Done-when, and implementation Plan remain unchanged. No new defects found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Previously accepted correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
  - id: BR-2
    disposition: addressed
    note: |
      The --no-focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
  - id: BR-3
    disposition: addressed
    note: |
      Previously accepted native coverage remains unchanged and required by the scheduled runner; all eight runner proof tests pass. Native conformance was not rerun in this narrow review.
```

1. **Strengths**
   - `effect_coverage_test.go:23` requires the existing `ctx` admission token for terminal creation.
   - Renamed startup entries retain their exact exemptions.
   - The tracker accurately records the outstanding full-suite requirement.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** The zellijreply package, focused recovery/focus regressions, eight native-runner proof tests, and whitespace checks passed. A temporary Go overlay restoring the base manifest reproduced all five expected classification failures. Checkout remains clean. Full suite not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Updates the existing authoritative inventory.
   - **ARCH-PURE — pass:** No business logic or IO boundary changes.
   - **ARCH-PURPOSE — pass:** Repairs every stale or missing entry reported by the exhaustive check.
   - **ARCH-EVOLVE — pass:** No persisted-format contract changes.

   No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b5601226-9169-4860-8db0-285e81ee4121","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/2b96cac31e037de8","reviewed_head":"7bda7379dddfada81f0dd81cd502fecec5c74201","slot":"pair:7","requested_at":"2026-10-20T17:14:49.873072Z"},"phase":"finished","pid":35561,"started_at":"2026-10-20T17:14:50.051299Z","finished_at":"2026-10-20T17:15:49.49033Z","session_id":"01a15a4f-04ba-7080-830e-e11d056f8ef5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.153221458s
head: "7bda7379dddfada81f0dd81cd502fecec5c74201"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T11:47:23-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | 00890cc01bddc334cff613db81abc4eadd1d04b6..b5291a95ec1267728faa50794d13bb8396565921 |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:47:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned interdiff changes only review records and tracker documentation. Production code, tests, Spec, Done-when, and the implementation plan remain unchanged. The new revision records verification and preserves the previously approved natural-relaunch acceptance; it introduces no scope change or new defect.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Previously accepted recovery correction is unchanged; TestRecoveryOwnsInputPumpEnding passes.
  - id: BR-2
    disposition: addressed
    note: |
      Previously accepted focus correction is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
  - id: BR-3
    disposition: addressed
    note: |
      Native conformance coverage remains unchanged and required by the scheduled runner; all eight runner tests pass. Native conformance was not rerun.
```

1. **Strengths**
   - Git comparison confirms production and test files match the previously reviewed `7bda7379` head.
   - The tracker revision explicitly preserves deferred live acceptance and does not claim landing.
   - Recovery, focus, and native-runner enforcement regressions pass.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Focused Go regressions, eight Python runner tests, and diff whitespace checks passed. Full suite and native conformance were not rerun; the recorded full-suite result was not independently reproduced.

6. **Architectural notes**
   - **ARCH-DRY — pass:** No duplicated implementation introduced.
   - **ARCH-PURE — pass:** No logic or IO boundary changes.
   - **ARCH-PURPOSE — pass:** Documentation preserves existing acceptance obligations.
   - **ARCH-EVOLVE — pass:** No persisted-format contract changes.

   No new surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3f58a2b8-57c5-4b73-b0fa-872f564db048","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3b93d00131395836","reviewed_head":"b5291a95ec1267728faa50794d13bb8396565921","slot":"pair:7","requested_at":"2026-10-20T18:46:48.06569Z"},"phase":"finished","pid":79992,"started_at":"2026-10-20T18:46:48.193154Z","finished_at":"2026-10-20T18:47:23.298268Z","session_id":"01a15aa3-37b9-7ad1-a889-e803f9fafcec","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.805087541s
head: "b5291a95ec1267728faa50794d13bb8396565921"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T12:54:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 727 — term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died |
| repo | pair |
| issue file | workshop/issues/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died.md |
| boundary | whole-issue close |
| milestone | — |
| window | 84eb4a1b3b26ea73a98c78cdc996679b9e1927a7..0bb089faf0a0ecff33e26ab5868dd1d0817f7406 |
| command | sdlc close --issue 727 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T12:54:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly reconciles the source-reader inventory with the integrated tests while retaining scanner enforcement. Production code, Spec, Done-when, and implementation plan are unchanged. No new defects or scope changes were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Recovery implementation and regression are unchanged; TestRecoveryOwnsInputPumpEnding passes.
  - id: BR-2
    disposition: addressed
    note: |
      Focus-preserving implementation is unchanged; TestNewTerminalRecreatesConfirmedAbsentPane passes.
  - id: BR-3
    disposition: addressed
    note: |
      Native conformance test and required runner registration are unchanged; all eight native-runner tests pass. Native conformance was not rerun.
```

1. **Strengths**
   - The three added inventory entries correctly classify fixture literals and private fixture reads.
   - Both repository scanners retain `fixed_scanner` classification and complete preflight registration.
   - Independent audit reproduces five failures with the base inventory and zero with the corrected inventory.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** All eight inventory tests pass, including the actual affected-runner negative control. Focused recovery, focus, scanner, and native-runner regressions pass. Diff whitespace checks pass. Full suite and native conformance were not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Existing inventory authority is reused.
   - **ARCH-PURE — pass:** No logic or IO boundary changes.
   - **ARCH-PURPOSE — pass:** All five inventory mismatches are resolved without weakening enforcement.
   - **ARCH-EVOLVE — pass:** Existing schema and field meanings are preserved.

   No new surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8006eefd-cc00-4b67-9c54-16c6a8daabf5","repository":"github.com/xianxu/pair","issue_id":"000727","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/085275e107509d98","reviewed_head":"0bb089faf0a0ecff33e26ab5868dd1d0817f7406","slot":"pair:7","requested_at":"2026-10-20T19:53:02.646583Z"},"phase":"finished","pid":64431,"started_at":"2026-10-20T19:53:02.821583Z","finished_at":"2026-10-20T19:54:24.49226Z","session_id":"01a15adf-dde5-70d0-ab65-832b0bd64e3b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.420422833s
head: "0bb089faf0a0ecff33e26ab5868dd1d0817f7406"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
