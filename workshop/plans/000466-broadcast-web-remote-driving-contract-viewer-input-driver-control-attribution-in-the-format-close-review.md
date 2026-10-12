# Boundary Review — pair#466 (whole-issue close)

| field | value |
|-------|-------|
| issue | 466 — broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format |
| repo | pair |
| issue file | workshop/issues/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format.md |
| boundary | whole-issue close |
| milestone | — |
| window | 26292514a113e10cb14354487c2ceb03fe688649..c4e3747c352d58d17cbd18bc761d618e4b3cc5e2 |
| command | sdlc close --issue 466 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:37:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the #466 diff: read the recording changes and the drive package's core; now going through the fakes and contract suites.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The format side of #466 is solid. `NextControl` is the single transition function that both the reducer and the fake controller use. `by` attribution is one embedded `Source` with one generic check. The catch-up keeps the latest `ctl`, and the late-joiner baseline rule is tested at every cut of `DrivenRecording`. The `driven` fixture is pinned, `ParseBatch` is strict with fixed errors, and `drive.Answer` turns the atlas's answers into code. Tests for `webcast/...` and `broadcast/...` pass, and vet and gofmt are clean. `artifactpath` fails in this checkout, but only because the generated `runtimebundle/assets/runtime/files` is missing; nothing in that failure mentions `drive`.

Three things should be fixed before close:
1. **The two fakes don't work together.** I wired `FakeController` to `FakeInjector` in a throwaway test. A properly granted driver's input was refused with an untyped reducer error ("by "v1", but the driver is """), which `Answer` turns into a 500. The atlas promises that "a batch overtaken by a revoke is refused" where it is dispatched. Nothing in the `Injector` seam supports that and no test checks it.
2. **`Confine` lets through keys that reach Chrome's UI:** Meta+Y (History on macOS), Shift+Escape (Task Manager), and a `code` field it never checks.
3. **Plan drift** (Minor): the plan names files and helpers that differ from what shipped.

**Strengths**
- `recording/control.go:336` — `NextControl` is pure, never changes its input (tested), and is reused by the reducer, `DrivenRecording` and `FakeController.changeLocked`, so there is one source of truth for control (ARCH-DRY, ARCH-ORDER).
- `recording/control.go:407`, `applyCtl`: the baseline rule (`ctlSeen`, `whole`) is explicit. It is tested both ways (`TestFirstCtlIsABaselineExceptForAWholeRecording`), and `CheckCatchUp` compares `Control` at every cut (`catchup_test.go:123`).
- `drive/act.go:107`, `ParseBatch`: strict decode through the now-shared `broadcast.DecodeStrict` (ARCH-DRY), with named limits, fixed error values and a no-echo test.
- `drive/controller_contract.go`: there is a real test that Revoke doesn't block while a batch is mid-dispatch, plus grace-leave, reattach-within-grace and seats that never attach. The suite ends by checking that the controller's records pass `Validate`.
- `drive/answer.go`: the error-to-status/sentence mapping is code, not prose.

**Critical findings**
None.

**Important findings**
1. **The fakes don't share control state, and the injector-side recheck of the driver is untested** (`drive/fake_injector.go:46`, `drive/injector.go:13`, `atlas/webcast-format.md` "One writer"; ARCH-MOCK, ARCH-ORDER).
   - `FakeInjector`'s page state is fixed at construction and never sees the controller's `ctl` records.
   - So `FakeController` + `FakeInjector` refuses a granted driver (confirmed with a probe test) and would accept a revoked one whose stale grant is in its page.
   - `RunControllerContract` only uses a spy injector, and `RunInjectorContract` never revokes. The atlas sentence "It rechecks that `by` drives when it dispatches, so a batch overtaken by a revoke is refused there" is not enforced anywhere.
   - Fix sketch:
     - Give the injector a way to learn control: a `Control()` or `Apply(*Ctl)` input, or route `ctl` through a shared sink the injector reads.
     - Have the recheck return a typed error (`ErrNotDriver`) instead of the reducer's string.
     - Add a contract step: grant, then revoke, then inject as the old driver; expect `ErrNotDriver` and no records.
     - Add one composed test of `FakeController` over `FakeInjector`.
2. **`Confine` allows accelerators that reach browser UI** (`drive/policy.go:55-61`; ARCH-SECURE).
   - Meta+Y passes. On macOS Chrome it opens History, and redo on a Mac is Shift+Meta+Z, which is already allowed.
   - Shift+Escape passes because Shift is allowed with every named key. It opens Chrome's Task Manager on Windows, Linux and ChromeOS.
   - Fix sketch: allow only Ctrl+Y (not Meta+Y), refuse Shift with Escape, and add both cases to `TestConfine`.
3. **`Confine` checks `Key` but not `Code`** (`drive/policy.go:47`, `drive/act.go:136`; ARCH-SECURE).
   - `{key:"a", code:"KeyW", mods:["meta"]}` passes.
   - If #467's injector builds `windowsVirtualKeyCode` from `code` (common practice), Chrome sees Cmd+W.
   - Fix sketch: either require `Code` to be consistent with `Key`, or state in the `Injector` contract that dispatch derives everything from `Key` and ignores `Code`. Test whichever you choose.

**Minor findings**
- Plan drift:
  - `RunControllerContract` lives in `controller_contract.go`, not `contract.go`.
  - There is no `drivenFixture` in `fixturegen/notes.go`; `main.go` calls `rec.DrivenRecording()` directly.
  - `RunInjectorContract` returns `func() []Record`, not `*MemSink`.
  - The plan's task checkboxes are still unchecked.
- `Ctl.consistent` accepts a baseline whose `driver` is `{"id":""}`. `Driving()` then reports the operator while `Driver != nil`. Refuse a driver with an empty id.
- The reducer accepts a `seg` after `stop end`. In that case `CatchUp` would carry a pre-end `ctl` as the baseline, and the late joiner's control would differ from the whole recording's. Either make `end` terminal or drop a `ctl` that precedes a later `stop end`.
- `ParseBatch` doesn't range-check `ime` `Sel` (negative values, or past `len(V)`).
- An `ActText` with nothing focused is dropped silently in `FakeInjector`. That is documented, but the caller can't see it happened.
- The grace timers use real `time.AfterFunc` with no injectable timer, so the tests rely on sleeps and see only one ordering of timer firing versus Attach (ARCH-ORDER).
- `label()` strips C0 control characters but not bidi or line-separator code points.
- The coverage test doesn't assert the plan's "a note by the driver", although the fixture has one.

**Test coverage notes**
- Good coverage: the transition table, attribution after grant, pending and revoke, survival across pause and navigation, catch-up at every cut, wire refusals, and the Answer mapping.
- Missing:
  - the revoke-then-stale-batch effect (Important 1);
  - fake composition;
  - Meta+Y and Shift+Esc;
  - `Code` versus `Key`;
  - the type of the error the injector returns when it refuses a non-driver (`TestFakeInjectorRefusesAnUnattributableDriver` only checks that the error isn't nil).

**Architecture**
- ARCH-DRY: pass (`DecodeStrict` is shared; one `Source` check covers every kind).
- ARCH-PURE: pass (`NextControl`, `Confine`, `ParseBatch` and `Answer` are pure).
- ARCH-PURPOSE: pass, apart from Important 1.
- ARCH-MOCK: flag (Important 1).
- ARCH-CONSTRAINTS: pass (limits enforced; rate limits explicitly left to #467 in the atlas).
- ARCH-SECURE: flag (Important 2 and 3).
- ARCH-ORDER: flag (Important 1; timer seam is Minor).
- ARCH-FUNERAL: pass (seats are memory only, timers stop at Close, `ctl` growth is bounded per human action).

**Architectural notes for upcoming work**
- #467 should put Request, Grant, Revoke and Input through the recorder's queue as planned. The `Injector` seam then needs that queue's view of control, which is what Important 1 asks for.
- Decide whether `stop end` is terminal before #460 serves catch-ups.

**Plan revision recommendations**
- A "## Revisions" entry that corrects the file locations (`controller_contract.go`), drops `drivenFixture`, and fixes the `RunInjectorContract` signature.
- Record that Ctrl/Meta+Enter was added to the allow-list.
- Record how the injector learns current control (once Important 1 is fixed).

```findings
findings:
  - id: new
    severity: Important
    family: fakes-share-the-production-boundary
    title: |
      FakeInjector never sees the controller's ctl records, so the fakes don't compose and the atlas's "injector rechecks by; an overtaken batch is refused" is untested
    detail: |
      A probe wiring FakeController to FakeInjector refused a granted driver with an untyped reducer error (500 via Answer), and a revoked driver would pass on a stale page. Add a control input to the Injector seam, a typed ErrNotDriver, a revoke-then-inject step in the contract, and one composed-fakes test.
  - id: new
    severity: Important
    family: confinement-allowlist
    title: |
      Confine allows Meta+Y (Chrome History on macOS) and Shift+Escape (Chrome Task Manager)
    detail: |
      policy.go:55-61 allows Meta with y and Shift with every named key. Allow only Ctrl+Y, refuse Shift+Escape, and add both cases to TestConfine.
  - id: new
    severity: Important
    family: confinement-allowlist
    title: |
      Confine checks Key but not Code; {key:a, code:KeyW, mods:[meta]} passes
    detail: |
      If #467 builds windowsVirtualKeyCode from code, the act becomes Cmd+W. Either require Code to agree with Key, or state in the Injector contract that dispatch derives from Key only, and test it.
  - id: new
    severity: Minor
    family: plan-matches-code
    title: |
      Plan names contract.go for RunControllerContract, a drivenFixture in fixturegen/notes.go, and a *MemSink return; the code differs
  - id: new
    severity: Minor
    family: untrusted-record-parse
    title: |
      A baseline ctl whose driver is {"id":""} is accepted: Driver is non-nil while Driving() reports the operator
  - id: new
    severity: Minor
    family: control-lifetime
    title: |
      The reducer allows a seg after stop end, so CatchUp can carry a pre-end ctl as the baseline and diverge from the whole recording
  - id: new
    severity: Minor
    family: ordering-seam
    title: |
      Grace timers use real time.AfterFunc with no injectable timer; tests sleep and see one ordering of timer versus Attach
  - id: new
    severity: Minor
    family: wire-validation
    title: |
      ParseBatch doesn't range-check ime sel (negative values, or past the length of v)
```
