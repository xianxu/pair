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
    - "n": 2
      timestamp: "2026-10-12T01:44:58-07:00"
      dispatched: "2026-10-12T01:41:33-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: ErrNotDriver in FakeInjector.Inject plus Sink follow; contract revoke step and TestFakesCompose go red when the check is removed (3 failures).
          round: 2
        - id: BR-2
          disposition: addressed
          note: policy.go refuses Meta+Y and Shift+Escape; reverting policy.go fails both TestConfine cases.
          round: 2
        - id: BR-3
          disposition: addressed
          note: codeFits holds Code to Key (exact US key under Ctrl/Meta); Injector doc states dispatch derives from Key; 3 TestConfine cases go red on revert.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Plan Revisions 2026-10-12 entry names controller_contract.go, fixturegen -driven writing DrivenRecording directly, and the new factory signature, matching the code.
          round: 2
        - id: BR-5
          disposition: addressed
          note: control.go consistent() refuses a driver with an empty id; TestCtlRefusesADriverWithoutAnID goes red on revert.
          round: 2
        - id: BR-6
          disposition: addressed
          note: State.Ended refuses a seg after stop end; the test goes red on revert (sibling ctl-after-end raised separately).
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: 'Left open on purpose (plan Revisions); FakeController still uses time.AfterFunc. Minor, deferred to #467''s queue-driven timers.'
          round: 2
        - id: BR-8
          disposition: addressed
          note: act.go range-checks sel against the rune count, and only for ime; 3 ParseBatch cases go red on revert.
          round: 2
      findings:
        - id: BR-9
          severity: Important
          title: Confine allows Shift+Delete, which Blink maps to Cut (writes the operator's clipboard) on Windows/Linux
          detail: '3rd finding in confinement-allowlist (4 holes in 2 rounds: Meta+Y, Shift+Escape, key/code mismatch, Shift+Delete). Fix the rule, not the instance: make Confine a closed table of allowed key x modifier chords, each allowed only if no Chrome/Blink binding handles it on any supported platform, and have a test walk namedKeys x modifier sets against that table.'
          family: confinement-allowlist
          round: 2
        - id: BR-10
          severity: Minor
          title: A ctl after stop end still reduces, leaving Ended=true with a driver granted
          detail: '2nd finding in control-lifetime. Probe: Page, stop end, ctl request, ctl grant reduces with no error to Driver v1. Rule: after stop end no record may change cast state (seg, ctl); enforce it in one guard in State.apply and test both kinds.'
          family: control-lifetime
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T01:47:58-07:00"
      dispatched: "2026-10-12T01:46:10-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: not-addressed
          note: 'Still real time.AfterFunc; explicitly deferred to #467 in plan Revisions, acceptable as Minor advisory.'
          round: 3
        - id: BR-9
          disposition: addressed
          note: Shift+Delete refused via closed chord table (policy.go:17-83), pinned by TestConfine row; see new finding for remaining table holes.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Single Ended guard for seg and ctl in State.apply (reduce.go:166); control_test.go:147 goes red when reverted.
          round: 3
      findings:
        - id: BR-11
          severity: Important
          title: Chord table allows Ctrl/Meta/Shift+Enter and Shift+Space, keyboard twins of the modified clicks Confine refuses; the exhaustive test is tautological
          detail: '4th in family. Blink takes the navigation policy from the activating key event''s modifiers, so Ctrl/Meta+Enter on a focused link opens a new tab, Shift+Enter opens a new window, and Shift+Space on a submit button sends the form result to a new window. The table''s own rule ("no Chrome/Blink binding") is broken, and policy_test.go:55 pins Meta+Enter as allowed. TestConfineIsExactlyTheChordTable compares Confine to chordAllowed, which Confine calls, so it cannot catch a bad entry. Rule to fix: activation keys (Enter, Space) take no ctrl/meta/shift, and every refused pointer act is refused in its keyboard form too. Check each chord against a separate, cited binding list that includes Blink''s activation rules, then correct the plan Revisions and atlas line 377.'
          family: confinement-allowlist
          round: 3
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

## Round 2 — 2026-10-12T01:44:58-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — ErrNotDriver in FakeInjector.Inject plus Sink follow; contract revoke step and TestFakesCompose go red when the check is removed (3 failures).
- BR-2 — addressed — policy.go refuses Meta+Y and Shift+Escape; reverting policy.go fails both TestConfine cases.
- BR-3 — addressed — codeFits holds Code to Key (exact US key under Ctrl/Meta); Injector doc states dispatch derives from Key; 3 TestConfine cases go red on revert.
- BR-4 — addressed — Plan Revisions 2026-10-12 entry names controller_contract.go, fixturegen -driven writing DrivenRecording directly, and the new factory signature, matching the code.
- BR-5 — addressed — control.go consistent() refuses a driver with an empty id; TestCtlRefusesADriverWithoutAnID goes red on revert.
- BR-6 — addressed — State.Ended refuses a seg after stop end; the test goes red on revert (sibling ctl-after-end raised separately).
- BR-7 — not-addressed — Left open on purpose (plan Revisions); FakeController still uses time.AfterFunc. Minor, deferred to #467's queue-driven timers.
- BR-8 — addressed — act.go range-checks sel against the rune count, and only for ime; 3 ParseBatch cases go red on revert.

### Raised

- **BR-9** [Important] `confinement-allowlist` Confine allows Shift+Delete, which Blink maps to Cut (writes the operator's clipboard) on Windows/Linux
  3rd finding in confinement-allowlist (4 holes in 2 rounds: Meta+Y, Shift+Escape, key/code mismatch, Shift+Delete). Fix the rule, not the instance: make Confine a closed table of allowed key x modifier chords, each allowed only if no Chrome/Blink binding handles it on any supported platform, and have a test walk namedKeys x modifier sets against that table.
- **BR-10** [Minor] `control-lifetime` A ctl after stop end still reduces, leaving Ended=true with a driver granted
  2nd finding in control-lifetime. Probe: Page, stop end, ctl request, ctl grant reduces with no error to Driver v1. Rule: after stop end no record may change cast state (seg, ctl); enforce it in one guard in State.apply and test both kinds.

## Round 3 — 2026-10-12T01:47:58-07:00 (claude) — BLOCKED

### Disposed

- BR-7 — not-addressed — Still real time.AfterFunc; explicitly deferred to #467 in plan Revisions, acceptable as Minor advisory.
- BR-9 — addressed — Shift+Delete refused via closed chord table (policy.go:17-83), pinned by TestConfine row; see new finding for remaining table holes.
- BR-10 — addressed — Single Ended guard for seg and ctl in State.apply (reduce.go:166); control_test.go:147 goes red when reverted.

### Raised

- **BR-11** [Important] `confinement-allowlist` Chord table allows Ctrl/Meta/Shift+Enter and Shift+Space, keyboard twins of the modified clicks Confine refuses; the exhaustive test is tautological
  4th in family. Blink takes the navigation policy from the activating key event's modifiers, so Ctrl/Meta+Enter on a focused link opens a new tab, Shift+Enter opens a new window, and Shift+Space on a submit button sends the form result to a new window. The table's own rule ("no Chrome/Blink binding") is broken, and policy_test.go:55 pins Meta+Enter as allowed. TestConfineIsExactlyTheChordTable compares Confine to chordAllowed, which Confine calls, so it cannot catch a bad entry. Rule to fix: activation keys (Enter, Space) take no ctrl/meta/shift, and every refused pointer act is refused in its keyboard form too. Check each chord against a separate, cited binding list that includes Blink's activation rules, then correct the plan Revisions and atlas line 377.

## Open findings

- **BR-7** [Minor] `ordering-seam` Grace timers use real time.AfterFunc with no injectable timer; tests sleep and see one ordering of timer versus Attach
- **BR-11** [Important] `confinement-allowlist` Chord table allows Ctrl/Meta/Shift+Enter and Shift+Space, keyboard twins of the modified clicks Confine refuses; the exhaustive test is tautological
