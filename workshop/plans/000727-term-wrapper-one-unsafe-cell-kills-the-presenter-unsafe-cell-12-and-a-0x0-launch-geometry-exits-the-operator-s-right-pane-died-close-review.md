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
