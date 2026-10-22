# Boundary Review — pair#801 (whole-issue close)

| field | value |
|-------|-------|
| issue | 801 — External hold-open Couch conformance bridge |
| repo | pair |
| issue file | workshop/issues/000801-external-couch-conformance-bridge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..70954412f60ce3ca578ea72314a8a20f1e6b8a94 |
| command | sdlc close --issue 801 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:52:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The bridge reuses the existing ownership machinery, separates lifecycle decisions from IO, and includes meaningful cancellation tests. All six reviewed packages pass, including bridge/CLI race checks. Three contract gaps block shipping: ambiguous responses can establish readiness, the legacy build wrapper rejects previously accepted temporary-directory aliases, and the Core concepts inventory contradicts the implementation.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: strict-boundary-decoding
    title: |
      Case-variant response keys can overwrite a failed admission result
    detail: |
      cmd/internal/testcouchbridge/native.go:239-273 checks exact duplicate keys, then uses case-insensitive json.Unmarshal. An endpoint reply {"Version":1,"Code":"unavailable","code":"ok","Error":"","EditorEndpoint":null} passes both checks and establishes readiness. Error:null also becomes an accepted empty string. ARCH-SECURE: decode known members with exact names, required-field/type validation and rejection of null strings, while tolerating additive metadata. Sweep registration and endpoint responses and add regression cases.
  - id: new
    severity: Critical
    kind: defect
    family: legacy-wrapper-compatibility
    title: |
      Build now rejects symlinked platform temporary directories
    detail: |
      cmd/internal/testcouch/binaries.go:35-39 passes os.TempDir() unchanged into BuildOwned, whose owner_build.go:119-120 rejects physical-path aliases. Without /Volumes/TestScratch, supported configurations such as macOS TMPDIR=/tmp now fail before building; the previous implementation accepted these paths. Reuse legacyScratchParent(), which already canonicalizes this selection for New, and add a regression exercising alias selection without the TestScratch override.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      The Core concepts table does not identify the implemented entities
    detail: |
      workshop/plans/000801-external-couch-bridge-plan.md:27-42 names BridgeOptions, BridgeState, BridgeEvent, ReduceBridge and BridgeRunner, but the code declares Options, State, Event, Reduce and RunBridge. It locates BridgeIO in nonexistent testcouchbridge/io.go; the interface is in run.go, with implementations split across native.go and pipes_unix.go. Under the required Core concepts cross-check, this contradiction is blocking. ARCH-PURPOSE: correct the inventory and record the implementation mapping in Revisions.
```

1. **Strengths**

   - `model.go` centralizes phase transitions; generated sequence tests check duplicate effects and stop dominance.
   - `run.go` cancels and joins active work before invoking shared cleanup.
   - Native readiness requires actual registration and endpoint replies, with identity checks.
   - README and atlas document invocation, replacement environments, limits, and cleanup uncertainty.

2. **Critical findings**

   The three findings above require strict response decoding, restoration of legacy scratch selection, and an accurate Core concepts inventory.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed `go test -count=1` for `testcouch`, `testcouchbridge`, `couch-conformance-bridge`, `testpair`, and `couchmessage`; bridge and CLI also passed with `-race`. Existing malformed-response tests cover exact duplicate keys, but omit case aliases and null string fields. Scratch tests omit the legacy wrapper’s aliased default.

   Native qualification was not rerun. Retained log hashes match their manifests, and `cmd/` is unchanged from the recorded native qualification commit. This verifies evidence consistency, not an independent live qualification.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared build and Fleet ownership; reuse the existing scratch helper for the regression.
   - **ARCH-PURE — pass:** options, readiness validation, and reducer tests run without IO.
   - **ARCH-PURPOSE — flag:** implementation inventory needs reconciliation; real-producer admission is implemented.
   - **ARCH-MOCK — pass:** injected stateful seams and native conformance exercise the production boundaries.
   - **ARCH-CONSTRAINTS — pass:** explicit deadlines, bounded output, and cancellable pipe tests.
   - **ARCH-SECURE — flag:** ambiguous external responses can become readiness evidence.
   - **ARCH-ORDER — pass:** reducer-controlled phases and causally gated cancellation tests.
   - **ARCH-FUNERAL — pass:** shared owners remove confirmed-clean resources and retain uncertain evidence.
   - **ARCH-EVOLVE — pass:** no persisted format change; additive response metadata is tested.

7. **Plan revision recommendations**

   Add a `## Revisions` entry mapping actual symbols and files to PURE/INTEGRATION classifications, including `NativeOwner`, `PipeIO`, and diagnostic adapters. Correct the Core concepts tables accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f43077d7-8315-4d80-8a27-f29b1c8cf320","repository":"github.com/xianxu/pair","issue_id":"000801","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6551e851dafaa65f","reviewed_head":"70954412f60ce3ca578ea72314a8a20f1e6b8a94","slot":"pair:7","requested_at":"2026-10-22T21:50:39.865009Z"},"phase":"finished","pid":73933,"started_at":"2026-10-22T21:50:40.115273Z","finished_at":"2026-10-22T21:52:49.407591Z","session_id":"01a16598-45d3-7ab0-a038-9627f6dc4eb6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.363845208s
head: "70954412f60ce3ca578ea72314a8a20f1e6b8a94"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
