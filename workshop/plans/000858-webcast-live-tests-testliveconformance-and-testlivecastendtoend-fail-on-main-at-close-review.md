# Boundary Review — pair#858 (whole-issue close)

| field | value |
|-------|-------|
| issue | 858 — Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click |
| repo | pair |
| issue file | workshop/issues/000858-webcast-live-tests-testliveconformance-and-testlivecastendtoend-fail-on-main-at.md |
| boundary | whole-issue close |
| milestone | — |
| window | 63d35ac8295d6b4ff04001c683b038eca1d0e3d7..5213a67eaa9e2dffc3d3d561a135c6028af138ef |
| command | sdlc close --issue 858 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T00:27:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The fixture now targets the actual closed-shadow button and preserves real mouse input, production code, and timeout constants. However, its geometry check does not fully enforce the specified hidden-control refusal. Native qualification also remains unverified because both attempted runs failed during Chrome startup.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: control-readiness-before-input
    title: |
      Positive box dimensions do not establish that the control is visible and usable
    detail: |
      cmd/internal/webcast/live_test.go:162-180 accepts any eight-coordinate border with positive width and height, then dispatches mouse input. A visibility:hidden button retains those dimensions, so the helper clicks despite the Done-when requirement that hidden controls refuse. Clipping or off-viewport placement likewise does not invalidate this check. Family enumeration: the single readiness gate in liveHarness.clickControl serves both conformance clicks (pause and resume) and the end-to-end pause click. Check visibility and usable hit geometry before dispatch, and add regression coverage for visible, missing, and hidden controls that proves refusal sends no mouse input. This leaves part of the stated purpose undelivered (ARCH-PURPOSE).
  - id: new
    severity: Important
    kind: unverifiable
    family: native-regression-qualification
    title: |
      Native pause/resume and viewer-state regressions could not be verified
    detail: |
      Attempted separately at the pinned head: WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestLiveConformance$' -count=1 -v -timeout=90s and the equivalent command for '^TestLiveCastEndToEnd$'. Both failed before opening a page with “GPU process isn't usable”; the latter also reported “sandbox initialization failed: Operation not permitted”. Neither reached clickControl. Supply measured host evidence of both commands passing against the reviewed source, plus a clean-base failure or fix-reversion run showing the native regression goes red without the correction, or provide an environment where those checks can execute.
```

1. **Strengths**

   - Closed-shadow traversal selects `.cast-button`, matching the actual production listener owner in `inject.js`.
   - Mouse press/release still exercises production hit testing and toggle binding.
   - Missing nodes, missing roots, and unavailable box models retry within the existing readiness loop.
   - The complete code diff is confined to the test fixture; production behavior and timeout constants are unchanged.

2. **Critical findings**

   - `live_test.go:162`: strengthen readiness beyond positive dimensions and cover refusal behavior, as detailed above.

3. **Important findings**

   - Native qualification requires executable host evidence; the issue Log’s reported results alone do not resolve the observed verification gap.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - `TestConformanceFake` passed.
   - Both native tests failed during Chrome startup, before exercising the change.
   - `git diff --check` passed.
   - Existing native assertions cover pause, resume, and paused viewer state. No tests exercise the missing/hidden refusal contract.

6. **Architectural notes**

   - **ARCH-DRY: pass.** One shared helper serves every native consumer.
   - **ARCH-PURE: pass.** The change stays within browser integration glue.
   - **ARCH-PURPOSE: flag.** Hidden-control refusal is incomplete.
   - **ARCH-EVOLVE: pass.** No persisted formats change.
   - No new public or architectural surface requires README or atlas updates. The quick plan has no Core concepts table.

7. **Plan revision recommendations**

   - Add a `## Revisions` entry explicitly including visible, missing, and hidden control qualification, with refusal verified before mouse dispatch.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ebee96f1-c871-48fa-a04f-fe12ea4376f1","repository":"github.com/xianxu/pair","issue_id":"000858","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/55e60b9398a8ca90","reviewed_head":"5213a67eaa9e2dffc3d3d561a135c6028af138ef","slot":"pair:4","requested_at":"2026-10-25T07:26:07.812382Z"},"phase":"finished","pid":72719,"started_at":"2026-10-25T07:26:07.968431Z","finished_at":"2026-10-25T07:27:34.818745Z","session_id":"01a171f3-d8d8-71d3-b3b6-83194ec38ded","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.713616666s
head: "5213a67eaa9e2dffc3d3d561a135c6028af138ef"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
