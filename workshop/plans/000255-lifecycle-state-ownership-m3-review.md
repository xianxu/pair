# Boundary Review — 000255-lifecycle-state-ownership#255 (milestone M3)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | c5ec1728810e89cbf0c9c8151bc3696c08a321ab..f32bb4cfaf06df17ba34b62968ba93dc479f6f24 |
| command | sdlc milestone-close --issue 255 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-15T14:54:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The migration establishes shared terminal ownership and passes the focused normal/race suites, but independent review reproduced a rendering regression and missing child disposal. Both need correction before M3 closes. The new build-time compiler dependency also lacks the required test seam.

## 1. Strengths

- Both consumers use the shared endpoint/presenter; raw replay no longer controls display state.
- Publication delivery has explicit packet/byte bounds, cancellation, and final-output acknowledgment.
- Independent terminal-oracle tests cover history, wrapping, and reflow. README and atlas updates accompany the migration.

## 2. Critical findings

**Erased-cell backgrounds disappear during presentation.**  
`cmd/internal/terminal/history_render.go:213` — `cells` emits cursor movement for empty-content cells without painting their background. The later background-restoration loop only handles cells beyond `UsedColumns`.

Reproduction through the production presenter: feed `ESC[41m ESC[2K ESC[0m ESC[1;4H X`. Independent xterm interpretation shows the first cell has red background directly, but default background after presentation. This affects ordinary colored erasures before later text.

**Fix:** preserve erased-cell attributes throughout viewport and history serialization while retaining blank provenance. Add independent comparisons covering interior/trailing blanks, append/rebuild, and alternate screens. **ARCH-PURPOSE.**

## 3. Important findings

**Couch retires exited origins without disposing their children.**  
`cmd/internal/couchtty/console.go:934` — removal calls `Presenter.Retire`, but never `Child.Close`. `publication.run` remains waiting at `cmd/internal/ptychild/publication.go:93` until explicitly canceled. `Couch.Forget` only updates registry storage.

A scratch regression exits a background child, waits for removal and a subsequent console command, then verifies disposal: its endpoint remains usable.

**This is the 2nd finding in family `artifact-lifetime-ownership`.** State and enforce the lifecycle rule across natural exit, park/relaunch, failed attachment, and teardown: drain final output, deselect/retire, then dispose terminal resources. Do not patch only this callback. **ARCH-FUNERAL / ARCH-ORDER.**

**The new `tic` dependency bypasses an injectable seam.**  
`cmd/internal/runtimebundlegen/generate.go:84` — `Generate` directly executes the installed compiler; generator tests consequently require that real binary. There is no compiler fake or dedicated conformance comparison.

**Fix:** inject compilation, model generated files and failures in a temporary-directory fake, and retain a separate real-compiler conformance test. **ARCH-MOCK.**

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- Focused terminal, PTY, host, Pair term, Couch TTY, and wrapper suites.
- Race suites for terminal, PTY, host, Pair term, and Couch TTY.
- Vendored VT suite and shared terminal suite with the independent oracle required.
- Pinned-range whitespace check.

Two scratch-overlay regressions failed: erased-background fidelity and exited-child disposal. Repository files were unchanged. Native operator workflows and extended M4 measurements were not rerun.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared publication and presentation replace competing display paths. |
| ARCH-PURE | Pass — frame/history transformations remain separate from IO. |
| ARCH-PURPOSE | Flag — erased-cell rendering violates faithful presentation. |
| ARCH-MOCK | Flag — direct compiler dependency lacks the required seam. |
| ARCH-CONSTRAINTS | Pass for M3 bounds; whole-application measurements remain M4 obligations. |
| ARCH-SECURE | Pass — inspected typed frame/history validation rejects unsafe payloads. |
| ARCH-ORDER | Flag — retired child resources outlive their consumer ownership. |
| ARCH-FUNERAL | Flag — publication workers lack disposal on Couch removal. |

## 7. Plan revision recommendations

Add `## Revisions` entries specifying:

- Attribute preservation for erased cells across every serializer path, with independent regression evidence.
- The complete child-disposal ownership rule and enumerated lifecycle paths.
- The compiler seam, fake output model, and real-compiler conformance check.

```findings
findings:
  - id: new
    severity: Critical
    family: terminal-cell-attribute-fidelity
    title: |
      History serialization drops backgrounds on erased interior cells
    detail: |
      cmd/internal/terminal/history_render.go:213 emits only CUF for empty-content cells; background restoration covers only the trailing extent. A production-presenter scratch regression with red EL2 followed by default-style text fails independent xterm comparison: BG=-1 instead of BG=1. Preserve erased-cell attributes across viewport/history append and rebuild paths, with independent regressions. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: artifact-lifetime-ownership
    title: |
      Couch removes exited children without disposing their publication workers
    detail: |
      cmd/internal/couchtty/console.go:934 retires the origin without Child.Close; cmd/internal/ptychild/publication.go:93 waits indefinitely without cancellation. A scratch regression confirms the endpoint remains usable after removal and a subsequent console command. This is the 2nd finding in family artifact-lifetime-ownership. State the drain/deselect/retire/dispose rule and sweep natural exit, park/relaunch, failed attachment, and teardown rather than fixing only this instance. ARCH-FUNERAL / ARCH-ORDER.
  - id: new
    severity: Important
    family: external-dependency-seam
    title: |
      Runtime generation directly invokes tic without an injectable compiler seam
    detail: |
      cmd/internal/runtimebundlegen/generate.go:84 introduces a direct external compiler dependency; generator tests use the installed binary rather than a fake behind the production seam. Inject compilation, model output files and failures in portable test storage, and add a dedicated real-compiler conformance comparison. ARCH-MOCK.
```

---

## Re-review — 2026-09-15T15:11:38-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | c5ec1728810e89cbf0c9c8151bc3696c08a321ab..f45fbc770ea9ab985bf5333fbfdbc4630444a070 |
| command | sdlc milestone-close --issue 255 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-15T15:11:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-13, BR-14, and BR-15 are addressed, with regression tests that fail when their fixes are removed. Two new issues block M3: resize allocates before validating geometry, and a teardown test uses an invalid completion signal. The shared endpoint/presenter migration is otherwise supported by focused normal, race, and independent-oracle evidence. Repository files remained unchanged.

## 1. Strengths

- Both consumers derive display state from shared endpoint publications and presentation.
- Erased-background regressions compare production output against independent xterm interpretation across viewport, history, and alternate-screen paths.
- Child disposal now covers accepted ownership, natural exit, rejected attachment, startup rollback, and teardown.
- Compiler tests use the production injection seam; separate real `tic`/`infocmp` conformance passed.

## 2. Critical findings

**Validate resize geometry before allocation.**  
`cmd/internal/terminal/presenter.go:582` allocates `make([]Cell, host.Cols)` before `resizeLayout` validates geometry. After selecting a view with chrome, a scratch regression calling `Resize(..., Geometry{Cols: -1, Rows: 5}, nil)` crashes the presenter goroutine with `makeslice: len out of range`. This is an API-level reproduction, not an observed normal host-size event.

**Fix:** validate geometry before allocating; test negative, zero, and over-limit dimensions through both resize entrypoints. **ARCH-SECURE / ARCH-CONSTRAINTS.**

## 3. Important findings

**Teardown test does not wait for teardown.**  
`cmd/internal/couchtty/vtscreen_test.go:165` waits for `ResetRegion` anywhere in accumulated output. Ordinary presentation already emits it at `history_render.go:301`, so simulated shell writes can race shutdown. The test failed in the normal suite and **2 of 20 isolated repetitions**.

**This is the 3rd finding in family `qualification-observation-equivalence`.** State the rule: lifecycle completion requires the corresponding completion acknowledgment. Enumerate teardown assertions across both consumers and replace historical-output predicates with joined completion before probing the restored terminal. **ARCH-ORDER / ARCH-PURPOSE.**

## 4. Minor findings

- `atlas/how-to-bring-up-a-new-harness-cli.md:108` still recommends the deleted `stdoutChunk()` filtering path. Update it to the implemented wrapper contract.

## 5. Test coverage notes

- Passed: terminal tests with independent oracle required; PTY, Couch core, term, host, and generator normal suites; vendored VT suite.
- Passed: terminal, PTY, Couch TTY, term, host, and generator race suites.
- All three prior-finding mutation checks failed as expected.
- Couch TTY normal testing exposed the teardown flake above.
- Wrapper argument tests encountered sandbox-denied storage access; no passing claim for that suite.
- Native Zellij workflows, sustained M4 runs, and operator acceptance were not rerun.

## 6. Architectural notes

| Marker | Result |
|---|---|
| ARCH-DRY | Pass — shared publication/presentation replaces competing display paths. |
| ARCH-PURE | Pass — inspected frame, view, and history transformations have direct deterministic tests. |
| ARCH-PURPOSE | Flag — teardown evidence does not establish its claimed lifecycle boundary. |
| ARCH-MOCK | Pass — compiler injection, filesystem fake, and native conformance are exercised. |
| ARCH-CONSTRAINTS | Flag — resize allocates before enforcing geometry limits. |
| ARCH-SECURE | Flag — invalid geometry crashes instead of returning validation failure. |
| ARCH-ORDER | Flag — teardown test observes historical bytes instead of joined completion. |
| ARCH-FUNERAL | Pass — inspected production child-retirement paths now retain disposal ownership. |

README and atlas updates accompany the new runtime/profile and terminal surfaces. Sustained performance and live acceptance remain M4 obligations.

## 7. Plan revision recommendations

Add `## Revisions` entries requiring:

- Geometry validation before allocation across presenter resize entrypoints.
- Completion-based teardown evidence, with an enumerated consumer-test sweep.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Independent erased-background oracle tests pass at head; restoring the previous serializer makes indexed, RGB, and alternate viewport regressions fail.
  - id: BR-14
    disposition: addressed
    note: |
      Production ownership and disposal paths plus exit, teardown, refusal, and rollback tests were inspected; restoring the previous Console makes the exited-origin disposal regression fail.
  - id: BR-15
    disposition: addressed
    note: |
      Injected compiler and real tic/infocmp conformance tests pass; bypassing the injected compiler makes the PATH-isolated regression fail.
findings:
  - id: new
    severity: Critical
    family: validate-before-allocation
    title: |
      Presenter resize allocates before validating geometry
    detail: |
      cmd/internal/terminal/presenter.go:582 allocates using host.Cols before validation. A scratch production-API regression with selected chrome and Cols=-1 crashes the presenter goroutine. Validate before allocating and cover invalid and over-limit geometry through both resize entrypoints. ARCH-SECURE / ARCH-CONSTRAINTS.
  - id: new
    severity: Important
    family: qualification-observation-equivalence
    title: |
      Teardown regression mistakes ordinary paint output for shutdown completion
    detail: |
      cmd/internal/couchtty/vtscreen_test.go:165 accepts ResetRegion already emitted by history_render.go:301; simulated shell writes race teardown. Reproduced in the normal suite and 2 of 20 isolated repetitions. This is the 3rd finding in this family: require lifecycle completion acknowledgments and sweep teardown assertions across both consumers. ARCH-ORDER / ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: documentation-surface-accuracy
    title: |
      Harness guide still recommends deleted wrapper filtering
    detail: |
      atlas/how-to-bring-up-a-new-harness-cli.md:108 describes stdoutChunk and Codex synchronized-output stripping, both removed by this range. Update the guide to the implemented wrapper contract.
```

---

## Re-review — 2026-09-15T15:21:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | c5ec1728810e89cbf0c9c8151bc3696c08a321ab..e1a18517e63eeb174576c7c7f841a069cb794ab9 |
| command | sdlc milestone-close --issue 255 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-15T15:21:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned M3 range addresses BR-16–BR-18. Both consumers use shared endpoint/presenter ownership, and the focused normal, race, and independent-oracle checks passed. No new blocking findings emerged. This verdict covers M3 migration; sustained native conformance and operator acceptance remain M4 obligations. Repository files were unchanged.

## 1. Strengths

- Geometry validation now precedes allocation and cloning across presenter entrypoints (`cmd/internal/terminal/presenter.go:583`).
- Teardown probes wait for lifecycle completion before checking restoration or simulating successor-shell output (`cmd/internal/couchtty/vtscreen_test.go:168`).
- Both consumers derive presentation and negotiated modes from shared terminal state; the inspected production paths no longer use raw replay as display authority.
- Bounded publication queues, final-output acknowledgment, and explicit child disposal have production-seam tests.
- README and atlas updates cover compiled terminfo, typed history, ownership, and wrapper behavior.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

None newly raised.

## 5. Test coverage notes

Passed:

- Normal suites: terminal, PTY child, Couch TTY/core, Pair term, host, runtime generator, and wrapper.
- Race suites: terminal, PTY child, Couch TTY, Pair term, host, and runtime generator—with the independent terminal oracle required.
- Vendored VT suite.
- Revised teardown tests: 30 repetitions.

Correction evidence:

- **BR-16:** Restoring the previous presenter through a scratch overlay makes the committed regression panic at the original allocation.
- **BR-17:** Restoring the previous teardown test reproduces failure **7/100 times**. Revised tests use completion acknowledgments; inspected Pair teardown assertions already follow joined completion.
- **BR-18:** The corrected guide matches wrapper passthrough and normalized-stream observation.

Full root testing, native Zellij workflows, and sustained M4 measurements were not rerun. Range whitespace checking reported only Markdown hard-break spaces in the recorded review document.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared endpoint/presenter replaces competing consumer display authority. |
| ARCH-PURE | Pass — inspected frame, view, and serialization logic remains separate from transport; integration components are identified accordingly. |
| ARCH-PURPOSE | Pass for M3 — both consumers and wrapper migration are implemented; sustained acceptance remains explicitly pending. |
| ARCH-MOCK | Pass — stateful transport/child doubles share production seams; compiler injection and native conformance are present. |
| ARCH-CONSTRAINTS | Pass for M3 — geometry, history, publication, and write bounds are explicit and exercised. Aggregate performance remains M4 work. |
| ARCH-SECURE | Pass — inspected geometry and typed presentation inputs validate before allocation or emission. |
| ARCH-ORDER | Pass — selection admission follows successful presentation; teardown evidence now observes completion. |
| ARCH-FUNERAL | Pass — child disposal, worker joins, and temporary compiler-storage cleanup have defined owners and tests. |

## 7. Plan revision recommendations

No additional revision required. Existing geometry/completion revisions describe the corrections. Preserve the M4 sustained-testing and operator-acceptance gates.

```findings
dispose:
  - id: BR-16
    disposition: addressed
    note: |
      presenter.go:583 validates geometry before allocation; both resize APIs have invalid-geometry regressions. Restoring the prior presenter in a scratch overlay makes the committed test panic at the original allocation.
  - id: BR-17
    disposition: addressed
    note: |
      vtscreen_test.go:168 waits for Run completion before restoration checks and successor-shell simulation. Mouse and keyboard assertions were swept; Pair assertions use joined completion. Revised tests pass 30 repetitions; restoring the old test fails 7/100 runs.
  - id: BR-18
    disposition: addressed
    note: |
      atlas/how-to-bring-up-a-new-harness-cli.md:108-121 replaces deleted filtering advice and retires its telemetry expectation. wrap.go:2905 preserves normalized passthrough and wrap.go:2963 observes that queued stream.
  - id: BR-13
    disposition: addressed
    note: |
      Prior disposition retained; erased-background oracle coverage passes with the independent oracle required.
  - id: BR-14
    disposition: addressed
    note: |
      Prior disposition retained; inspected retirement/disposal ownership and passing consumer disposal tests support the correction.
  - id: BR-15
    disposition: addressed
    note: |
      Prior disposition retained; injected compiler failure/preservation tests and native compiler conformance pass.
```
