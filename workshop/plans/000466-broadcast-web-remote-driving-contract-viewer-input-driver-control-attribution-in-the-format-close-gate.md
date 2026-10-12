---
gate: boundary-review
issue: 466
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T01:37:55-07:00"
      dispatched: "2026-10-12T01:33:56-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: FakeInjector never sees the controller's ctl records, so the fakes don't compose and the atlas's "injector rechecks by; an overtaken batch is refused" is untested
          detail: A probe wiring FakeController to FakeInjector refused a granted driver with an untyped reducer error (500 via Answer), and a revoked driver would pass on a stale page. Add a control input to the Injector seam, a typed ErrNotDriver, a revoke-then-inject step in the contract, and one composed-fakes test.
          family: fakes-share-the-production-boundary
          round: 1
        - id: BR-2
          severity: Important
          title: Confine allows Meta+Y (Chrome History on macOS) and Shift+Escape (Chrome Task Manager)
          detail: policy.go:55-61 allows Meta with y and Shift with every named key. Allow only Ctrl+Y, refuse Shift+Escape, and add both cases to TestConfine.
          family: confinement-allowlist
          round: 1
        - id: BR-3
          severity: Important
          title: Confine checks Key but not Code; {key:a, code:KeyW, mods:[meta]} passes
          detail: 'If #467 builds windowsVirtualKeyCode from code, the act becomes Cmd+W. Either require Code to agree with Key, or state in the Injector contract that dispatch derives from Key only, and test it.'
          family: confinement-allowlist
          round: 1
        - id: BR-4
          severity: Minor
          title: Plan names contract.go for RunControllerContract, a drivenFixture in fixturegen/notes.go, and a *MemSink return; the code differs
          family: plan-matches-code
          round: 1
        - id: BR-5
          severity: Minor
          title: 'A baseline ctl whose driver is {"id":""} is accepted: Driver is non-nil while Driving() reports the operator'
          family: untrusted-record-parse
          round: 1
        - id: BR-6
          severity: Minor
          title: The reducer allows a seg after stop end, so CatchUp can carry a pre-end ctl as the baseline and diverge from the whole recording
          family: control-lifetime
          round: 1
        - id: BR-7
          severity: Minor
          title: Grace timers use real time.AfterFunc with no injectable timer; tests sleep and see one ordering of timer versus Attach
          family: ordering-seam
          round: 1
        - id: BR-8
          severity: Minor
          title: ParseBatch doesn't range-check ime sel (negative values, or past the length of v)
          family: wire-validation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#466 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T01:37:55-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fakes-share-the-production-boundary` FakeInjector never sees the controller's ctl records, so the fakes don't compose and the atlas's "injector rechecks by; an overtaken batch is refused" is untested
  A probe wiring FakeController to FakeInjector refused a granted driver with an untyped reducer error (500 via Answer), and a revoked driver would pass on a stale page. Add a control input to the Injector seam, a typed ErrNotDriver, a revoke-then-inject step in the contract, and one composed-fakes test.
- **BR-2** [Important] `confinement-allowlist` Confine allows Meta+Y (Chrome History on macOS) and Shift+Escape (Chrome Task Manager)
  policy.go:55-61 allows Meta with y and Shift with every named key. Allow only Ctrl+Y, refuse Shift+Escape, and add both cases to TestConfine.
- **BR-3** [Important] `confinement-allowlist` Confine checks Key but not Code; {key:a, code:KeyW, mods:[meta]} passes
  If #467 builds windowsVirtualKeyCode from code, the act becomes Cmd+W. Either require Code to agree with Key, or state in the Injector contract that dispatch derives from Key only, and test it.
- **BR-4** [Minor] `plan-matches-code` Plan names contract.go for RunControllerContract, a drivenFixture in fixturegen/notes.go, and a *MemSink return; the code differs
- **BR-5** [Minor] `untrusted-record-parse` A baseline ctl whose driver is {"id":""} is accepted: Driver is non-nil while Driving() reports the operator
- **BR-6** [Minor] `control-lifetime` The reducer allows a seg after stop end, so CatchUp can carry a pre-end ctl as the baseline and diverge from the whole recording
- **BR-7** [Minor] `ordering-seam` Grace timers use real time.AfterFunc with no injectable timer; tests sleep and see one ordering of timer versus Attach
- **BR-8** [Minor] `wire-validation` ParseBatch doesn't range-check ime sel (negative values, or past the length of v)

## Open findings

- **BR-1** [Important] `fakes-share-the-production-boundary` FakeInjector never sees the controller's ctl records, so the fakes don't compose and the atlas's "injector rechecks by; an overtaken batch is refused" is untested
- **BR-2** [Important] `confinement-allowlist` Confine allows Meta+Y (Chrome History on macOS) and Shift+Escape (Chrome Task Manager)
- **BR-3** [Important] `confinement-allowlist` Confine checks Key but not Code; {key:a, code:KeyW, mods:[meta]} passes
- **BR-4** [Minor] `plan-matches-code` Plan names contract.go for RunControllerContract, a drivenFixture in fixturegen/notes.go, and a *MemSink return; the code differs
- **BR-5** [Minor] `untrusted-record-parse` A baseline ctl whose driver is {"id":""} is accepted: Driver is non-nil while Driving() reports the operator
- **BR-6** [Minor] `control-lifetime` The reducer allows a seg after stop end, so CatchUp can carry a pre-end ctl as the baseline and diverge from the whole recording
- **BR-7** [Minor] `ordering-seam` Grace timers use real time.AfterFunc with no injectable timer; tests sleep and see one ordering of timer versus Attach
- **BR-8** [Minor] `wire-validation` ParseBatch doesn't range-check ime sel (negative values, or past the length of v)
