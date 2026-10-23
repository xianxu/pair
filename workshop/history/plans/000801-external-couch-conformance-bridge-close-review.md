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

---

## Re-review — 2026-10-22T15:05:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 801 — External hold-open Couch conformance bridge |
| repo | pair |
| issue file | workshop/issues/000801-external-couch-conformance-bridge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..a7083ce3621a294233a8a513e4d93462d00ece7a |
| command | sdlc close --issue 801 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:05:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, with regression evidence for both behavioral fixes. The bridge has clear ownership, bounded IO, and meaningful cancellation tests. One additional compatibility regression blocks shipping: the existing `Build(t)` wrapper ignores the suite-selected Pair executable and silently builds another.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Exact-member decoding covers registration, nested document identity, and endpoint replies. Existing regression tests pass at head and fail when the pre-fix native.go is restored, including case-shadow and null-error controls.
  - id: BR-2
    disposition: addressed
    note: |
      Build's default selection now uses the shared canonicalization helper. TestDefaultBuildConfigCanonicalizesTempAliasWithoutScratchVolume passes at head and fails with canonicalization removed; explicit aliases remain rejected before effects.
  - id: BR-3
    disposition: addressed
    note: |
      Plan lines 27-46 now identify Options, State/Event/Reduce, RunBridge, BridgeIO in run.go, and the concrete native/pipe/diagnostic adapters. The close BR-3 Revision also maps earlier strategy labels to implemented functions.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: legacy-wrapper-compatibility
    title: |
      Legacy Build ignores the suite-selected Pair candidate
    detail: |
      cmd/internal/testcouch/binaries.go:38 routes Build(t) through BuildOwned, whose owner_build.go:227 calls PlaceCandidate without Candidate; candidate.go:49 therefore supplies an empty PAIR_TEST_BINARY and rebuilds Pair. Previously Build used testpair.AtContext, honoring the suite candidate documented at README.md:2060-2069. Targeted qualification can consequently exercise a different executable from the one requested. An injected-tool regression fails on the pinned head and passes with the original Build implementation. ARCH-PURPOSE: preserve candidate-selection behavior at the testing wrapper while keeping standalone BuildOwned isolated from ambient candidate selection.
      This is the 2nd finding in family `legacy-wrapper-compatibility`. Earlier rounds fixed instances. Do NOT fix this instance alone: state and enforce the rule that existing testing wrappers preserve legacy selection and ownership policies while standalone entrypoints use explicit isolated inputs. Sweep Build/New and candidate placement together. The affected Build consumer enumeration contains three direct callers across two packages: TestDisposableCouchFleet, TestDisposableCouchStartupBudget, and TestLiveProvenOrphanReap. Commit regression coverage for supplied-candidate reuse, unset-candidate fallback, candidate lifetime, and standalone isolation.
```

1. **Strengths**

   - Readiness requires actual registration and successful public endpoint replies, with strict consumed fields and additive metadata tolerance.
   - The pure reducer handles duplicate and late events; generated sequence tests assert independent invariants.
   - Pipe cancellation and nonblocking diagnostics prevent unread output from indefinitely blocking cleanup.
   - README and atlas updates cover the executable, environment replacement, bounds, and cleanup contract.

2. **Critical findings**

   The candidate-selection regression above violates the promised wrapper compatibility. Carry the legacy resolver’s selection explicitly through the shared placement seam, retaining strict standalone isolation.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed complete portable tests for `testcouch`, `testcouchbridge`, the bridge CLI, `testpair`, and `couchmessage`; bridge and CLI race tests also passed.

   Both prior behavioral fixes produced failing regression tests when removed. The new compatibility test failed at head and passed with original Build behavior. Disposable test copies were removed; the reviewed checkout remains unchanged.

   Retained native/consumer log hashes match their evidence records. Native qualification was not rerun during this review.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared fleet ownership and candidate-placement seams. |
   | ARCH-PURE | Pass: options, readiness validation, and reducer tests require no IO. |
   | ARCH-PURPOSE | **Flag:** legacy candidate-selection contract is lost. |
   | ARCH-MOCK | Pass: injected owner, command, and pipe seams support controlled failures. |
   | ARCH-CONSTRAINTS | Pass: absolute deadlines, output caps, and blocked-pipe controls. |
   | ARCH-SECURE | Pass: strict response parsing and private runtime environment. |
   | ARCH-ORDER | Pass: reducer-driven phases and causal cancellation tests. |
   | ARCH-FUNERAL | Pass: joined cleanup removes owned roots; uncertainty retains evidence. |
   | ARCH-EVOLVE | Pass: no persisted-format rewrite introduced; opaque response metadata remains tolerated. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the compatibility rule for testing wrappers versus standalone owners, enumerating the affected consumers, and naming the regression tests that enforce candidate selection and lifetime.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7425ce25-1693-49f2-a410-f51e0bf5172d","repository":"github.com/xianxu/pair","issue_id":"000801","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ea5be700998fe23d","reviewed_head":"a7083ce3621a294233a8a513e4d93462d00ece7a","slot":"pair:7","requested_at":"2026-10-22T22:01:51.515836Z"},"phase":"finished","pid":42558,"started_at":"2026-10-22T22:01:51.632722Z","finished_at":"2026-10-22T22:05:41.323822Z","session_id":"01a165a2-8479-79c2-aa34-86463ebf18b1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.204567583s
head: "a7083ce3621a294233a8a513e4d93462d00ece7a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T15:22:30-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 801 — External hold-open Couch conformance bridge |
| repo | pair |
| issue file | workshop/issues/000801-external-couch-conformance-bridge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..510d18f8b3893c19c177016774019b57740b2222 |
| command | sdlc close --issue 801 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:22:30-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the issue’s producer, isolation, readiness, and joined-cleanup contracts. BR-4 is addressed across the documented compatibility boundary, with regression tests that fail when the corrections are disabled. No new blocking findings emerged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Exact-member decoding remains enforced in native.go; malformed-member and public-envelope regression tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Legacy default selection canonicalizes temporary-directory aliases; explicit standalone aliases remain rejected before effects. Both controls pass.
  - id: BR-3
    disposition: addressed
    note: |
      The plan's Core concepts tables and BR-3 Revision identify the implemented symbols and adapter files; the declared pure entities contain no IO.
  - id: BR-4
    disposition: addressed
    note: |
      Build forwards BinaryContext's selected executable into shared placement while BuildOwned remains isolated. Real-wrapper subprocess tests cover supplied and fallback selection, repeated callers, placement cleanup, and source lifetime. Disabling forwarding with a temporary compiler overlay makes both selection modes fail. Removing legacy destination normalization separately makes all four placement controls fail.
```

1. **Strengths**

   - Candidate policy stays at the compatibility wrapper; shared placement consumes the explicit selection (`cmd/internal/testcouch/binaries.go:43`, `owner_build.go:234`).
   - Cleanup uses one budget for cancellation, joins, Close, and AssertCleaned, retaining uncertainty (`cmd/internal/testcouchbridge/run.go:137`).
   - Tests execute the actual candidate resolver and placement script while substituting compiler/tool effects (`cmd/internal/testcouch/legacy_build_test.go:65`).
   - README and atlas document the executable, environment replacement, bounds, pipe requirements, and recovery limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed all tests in `testcouchbridge`, `couch-conformance-bridge`, `testpair`, and `testcouch`; bridge/CLI race tests; socket-path tests; and `make test-affected-inventory`.

   Both temporary-overlay negative controls failed for the expected behavioral reasons. The checkout remains clean. Native qualification was not rerun; retained native and consumer evidence explicitly identifies its earlier source revision.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared construction and candidate-placement pipelines.
   - **ARCH-PURE — pass:** Validation and reducer logic are independent of IO.
   - **ARCH-PURPOSE — pass:** Genuine registration and endpoint admission precede readiness; legacy candidate selection is preserved.
   - **ARCH-MOCK — pass:** Stateful owner and injected native boundaries support causal failure tests; native conformance is retained.
   - **ARCH-CONSTRAINTS — pass:** Explicit deadlines and output limits, including blocked-pipe coverage.
   - **ARCH-SECURE — pass:** Private environments, physical scratch admission, and strict response decoding.
   - **ARCH-ORDER — pass:** Production orchestration uses the reducer; stop dominance and late completions have sequence coverage.
   - **ARCH-FUNERAL — pass:** Owners remove confirmed-clean resources and retain uncertain evidence.
   - **ARCH-EVOLVE — pass:** No persisted-format mutation; additive response metadata remains tolerated with strict consumed fields.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"378b597a-f479-49f1-84ad-71fc9e29673e","repository":"github.com/xianxu/pair","issue_id":"000801","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b204408c40b7cdf5","reviewed_head":"510d18f8b3893c19c177016774019b57740b2222","slot":"pair:7","requested_at":"2026-10-22T22:20:10.576193Z"},"phase":"finished","pid":82735,"started_at":"2026-10-22T22:20:10.771383Z","finished_at":"2026-10-22T22:22:29.534955Z","session_id":"01a165b3-4a6b-7f31-bfe0-d32d93209652","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.571851833s
head: "510d18f8b3893c19c177016774019b57740b2222"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
