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

---

## Re-review — 2026-10-25T01:03:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 858 — Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click |
| repo | pair |
| issue file | workshop/issues/000858-webcast-live-tests-testliveconformance-and-testlivecastendtoend-fail-on-main-at.md |
| boundary | whole-issue close |
| milestone | — |
| window | 63d35ac8295d6b4ff04001c683b038eca1d0e3d7..d8b65cc621abe2fb9ba5356158cc23d7b5c026e2 |
| command | sdlc close --issue 858 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T01:03:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The fixture now targets the actual closed-shadow button and checks visibility and hit geometry before dispatching real mouse input. Both prior findings are resolved. Local native runs remain blocked during Chrome startup, but the retained host measurements match the reviewed source and substantiate the required behavior.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      live_test.go:183-201 checks ancestor visibility, opacity, viewport bounds and shadow/document hit targets before input. TestLiveControlReadiness exercises visible/restored, hidden, transparent, offscreen, clipped and missing controls, counting forwarded mouse events. Its refusal assertions catch the positive-box behavior identified previously; retained readiness evidence passes on source byte-identical to this head.
  - id: BR-2
    disposition: withdrawn
    note: |
      All four retained logs match their SHA-256 metadata. Both fixture files at evidence head 7b104dd5 are byte-identical to the reviewed head. Separate native conformance, end-to-end and readiness runs pass without skips; the controlled clean-main fixture reversion fails at the pause assertion, and its source hash matches the base. Local retries reproduce only the pre-page GPU/sandbox startup failure, which does not contradict this measured host evidence.
```

Done-when: current — it covers actual-button targeting, unusable-control refusal, both native consumers, headless execution and unchanged production behavior and deadlines.

1. **Strengths**

   - One shared readiness gate serves conformance pause/resume and end-to-end pause.
   - Real mouse press/release preserves production hit testing and toggle handling ([live_test.go:214](/Volumes/TestScratch/review-checkouts/670b0764da03a015/cmd/internal/webcast/live_test.go:214)).
   - The regression uses real closed-shadow DOM and verifies zero input on refusal ([live_control_test.go:51](/Volumes/TestScratch/review-checkouts/670b0764da03a015/cmd/internal/webcast/live_control_test.go:51)).
   - Production code and existing timeout constants remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - `TestConformanceFake` and range whitespace checks pass locally.
   - All three targeted native attempts fail before page creation because Chrome cannot initialize its GPU/sandbox.
   - Retained host evidence supplies native passes and a baseline-reversion failure. The readiness regression’s sensitivity to the previous positive-box check is evident from its assertions; I could not independently rerun that reversal here.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Shared gate; no duplicated readiness implementation.
   - **ARCH-PURE: pass.** Browser inspection remains integration glue with an injected CDP caller.
   - **ARCH-PURPOSE: pass.** Both consumers and the previously missing refusal contract are covered.
   - **ARCH-EVOLVE: pass.** No persisted format changes.
   - No new user-facing or architectural surface requires README or atlas updates. The quick plan has no Core concepts table.

7. **Plan revision recommendations:** None; the existing `## Revisions` describes the implemented readiness rule and qualification scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e774c23f-b31c-42dc-ac11-08e7b8d3df22","repository":"github.com/xianxu/pair","issue_id":"000858","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/670b0764da03a015","reviewed_head":"d8b65cc621abe2fb9ba5356158cc23d7b5c026e2","slot":"pair:4","requested_at":"2026-10-25T07:57:39.748941Z"},"phase":"finished","pid":77789,"started_at":"2026-10-25T07:57:39.890556Z","finished_at":"2026-10-25T07:59:57.233602Z","session_id":"01a17210-b62b-7123-a1e0-13dc3acf726c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.759443291s
head: "d8b65cc621abe2fb9ba5356158cc23d7b5c026e2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-25T04:15:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 858 — Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click |
| repo | pair |
| issue file | workshop/issues/000858-webcast-live-tests-testliveconformance-and-testlivecastendtoend-fail-on-main-at.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5e6e08ec7bd2238f0345f3d9e318eea957cdf8a7..c927d00d4d3b4840a771ad06dd80a5bd8c2f0754 |
| command | sdlc close --issue 858 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T04:15:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The interdiff restores the readiness wait to its admitted owner without changing deadlines or production behavior. Input errors remain outside readiness retries. No new defects or scope changes were found. Local native verification failed during Chrome startup; retained native evidence matches the reviewed fixture source.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The visibility and hit-test gate remains intact. TestLiveControlReadiness retains hidden, transparent, offscreen, clipped and missing-control refusals with zero input assertions. The retained passing readiness log matches its hash and byte-identical fixture source.
  - id: BR-2
    disposition: withdrawn
    note: |
      Retained conformance, end-to-end and readiness logs match their metadata hashes and current fixture bytes. The baseline-reversion log records failure at the pause assertion. Local Chrome again fails before page creation, which does not contradict that evidence.
```

Done-when: current — it covers usable-control targeting, native pause/resume, viewer state, unchanged deadlines and production behavior. This window changes neither Spec nor Revisions.

1. **Strengths**

   - The existing 10-second deadline and 100-millisecond wait remain in `liveHarness.clickControl` (`live_test.go:113–125`).
   - The caller treats input errors as terminal (`live_test.go:115–120`).
   - The added regression asserts input-error identity, readiness classification and exactly one dispatch attempt (`live_control_test.go:82–92`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: tracked-source sleep catalog, fake conformance, affected-reader inventory and diff whitespace checks.
   - Local native readiness failed before page creation with Chrome’s “GPU process isn't usable” error.
   - All seven retained native logs match their metadata hashes. The three post-fix passing measurements reference fixture files byte-identical to this head. No reversal was performed during this read-only review.

6. **Architectural notes**

   - **ARCH-DRY: pass.** One shared readiness/input implementation.
   - **ARCH-PURE: pass.** Browser integration uses an injected CDP caller.
   - **ARCH-PURPOSE: pass.** Restores catalog compliance while preserving control qualification and both native consumers.
   - **ARCH-EVOLVE: pass.** No persisted-format changes.
   - No new user-facing or architectural surface requires README or atlas updates. No Core concepts table applies.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6bf3ed43-3b2c-45d9-bc51-94980a0ebc95","repository":"github.com/xianxu/pair","issue_id":"000858","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6c309c6d45c3c352","reviewed_head":"c927d00d4d3b4840a771ad06dd80a5bd8c2f0754","slot":"pair:4","requested_at":"2026-10-25T11:13:47.90201Z"},"phase":"finished","pid":51676,"started_at":"2026-10-25T11:13:48.003454Z","finished_at":"2026-10-25T11:15:06.87982Z","session_id":"01a172c4-4702-7933-a481-43d1c001950a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.214712083s
head: "c927d00d4d3b4840a771ad06dd80a5bd8c2f0754"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
