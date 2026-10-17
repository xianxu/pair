# Boundary Review — pair#639 (whole-issue close)

| field | value |
|-------|-------|
| issue | 639 — Right terminal: Alt+Shift+Return toggles Normal and Focus (75% centered) only; drop Maximize |
| repo | pair |
| issue file | workshop/issues/000639-right-terminal-alt-shift-return-toggles-normal-and-focus-75-centered-only-drop-maximize.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7607d02eaf742a5a9fb527e71c0bce15a0c67947..d475bb45cc41e40fb9ecaccef838dd0eae58424a |
| command | sdlc close --issue 639 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T01:13:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two-state behavior is implemented correctly in the reviewed paths. Affected package tests, actual-chord coverage, and direct native conformance pass. One existing live test still assumes three presses, fails against this change, and prevents verification across the shipped layout rungs.

Done-when: current — it covers the two-state cycle, native-fullscreen compatibility exit, documentation, and explicitly deferred operator smoke; the return-focus revision supports those requirements.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: transition-contract-test-consumers
    title: |
      Rung conformance still assumes a three-press cycle
    detail: |
      cmd/internal/layoutcmd/rightpane_rungs_live_test.go:171–176 invokes press(draft), press(target), press(target). With the revised PlanRightPane, the second press restores Normal and the third re-enters Focus. Running TestRightPaneRungsZellijConformance with PAIR_LIVE_ZELLIJ=1 and a short private ZELLIJ_SOCKET_DIR fails after 11.29s with "timed out waiting for base restored". Remove the extra press and rerun all six shared-helper cases: base, minimized, third, dirty-split-bottom, split-top, and split-next-rung-top. This is the remaining three-press test consumer found in the cycle sweep; the other cycle tests were updated.
```

1. **Strengths**

   - `rightpane.go:162` restores tiling, explicitly restores return focus, clears the record, and nudges dimming without targeting fullscreen.
   - `fullscreen.go:93` shares return-pane lookup between Focus exit and native-fullscreen exit.
   - The already-focused exception requires both exit status 2 and the exact target-specific diagnostic; negative tests reject unrelated failures.
   - README, atlas, and source-derived key help consistently describe Normal ↔ Focus.

2. **Critical findings:** None.

3. **Important findings:** The stale rung-conformance sequence above. Remove its third press so the existing geometry assertions validate the revised round trip.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: `layoutcmd`, `termcmd`, `keyhelp`, and `workbenchshortcut` package tests.
   - Passed: actual-chord live test across all five paths.
   - Passed: direct native conformance, including return focus and one-press fullscreen exit.
   - Failed: rung conformance as detailed above.
   - Direct conformance initially encountered an inherited socket-path limit; it passed with a short private socket directory. Operator smoke remains deferred as documented.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared return lookup and existing restore operations are reused.
   - **ARCH-PURE: pass** — transition planning remains deterministic; native actions stay behind the injected runtime.
   - **ARCH-PURPOSE: pass** — observed behavior fulfills the two-state purpose while retaining the fullscreen escape path. The remaining finding concerns test migration.

7. **Plan revision recommendations:** No contract revision needed. Record the corrected six-case rung verification after fixing the test.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5534ac71-075b-4f32-b3df-a6759fb82051","repository":"github.com/xianxu/pair","issue_id":"000639","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/da51b32e71fa6b4d","reviewed_head":"d475bb45cc41e40fb9ecaccef838dd0eae58424a","slot":"pair:1","requested_at":"2026-10-17T08:12:19.238643Z"},"phase":"finished","pid":17178,"started_at":"2026-10-17T08:12:19.363024Z","finished_at":"2026-10-17T08:13:55.589651Z","session_id":"01a148eb-41a6-7400-96a4-e94b2cc48b14","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.984093334s
head: "d475bb45cc41e40fb9ecaccef838dd0eae58424a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
