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

---

## Re-review — 2026-10-12T01:44:58-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 466 — broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format |
| repo | pair |
| issue file | workshop/issues/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format.md |
| boundary | whole-issue close |
| milestone | — |
| window | 26292514a113e10cb14354487c2ceb03fe688649..44e602e7c483f8f38d82c9f521ff171217ad2cc4 |
| command | sdlc close --issue 466 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:44:58-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on round-1 fix commit first.

All round-1 fixes go red when reverted. Now checking the rest of the Confine allowlist and the Ended/CatchUp interaction.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

I verified all eight round-1 findings. Seven are fixed, and in each case reverting the fix makes its tests fail. BR-1, the injector refusing anyone who isn't the driver, now has a working stream input and a test that wires the two fakes together. One new Important finding is the third in the `confinement-allowlist` family: Shift+Delete still gets through Confine, and on non-mac Chrome it cuts to the clipboard. One new Minor finding: a `ctl` record is still accepted after `stop end`. BR-7 was deliberately left open and is non-blocking.

**How I inspected:** I ran the stat and name-status recipes, read the full round-1 diff (c3e74364) and the atlas/plan update (44e602e7), and ran `go vet` and `go test` on `webcast/...`, `broadcast` and `artifactpath`. Webcast, drive (also under `-race`) and broadcast pass. `artifactpath` fails two tests because `cmd/internal/runtimebundle/assets/` is gitignored, generated content that this checkout doesn't have; that failure has nothing to do with this diff. I reverted each fix in the disposable checkout, saw the tests fail, and restored it. The working tree is clean.

1. **Strengths**
   - `fake_injector.go:59-61` and `contract.go:82-86,133-149`: the driver check runs where the batch is dispatched, and the contract covers both "a viewer who isn't driving" and "revoke, then the old driver injects". Removing the check fails 3 tests.
   - `TestFakesCompose` connects FakeController to FakeInjector through `recording.Tee`. That is the same boundary the real pieces use, which settles the `fakes-share-the-production-boundary` family.
   - `policy.go:93-109` `codeFits`: when Ctrl or Meta is held, `code` must be exactly the US key, but plain typing still accepts any layout. It also has tests for AZERTY, NumpadEnter and Space.
   - `reduce.go:174,192`: `Ended` makes `stop end` final for segments, and `consistent()` now refuses a driver with an empty id.
   - The plan's `## Revisions` entry ("as built") matches the code honestly, including the advisory that was left open.

2. **Critical findings:** none.

3. **Important findings**
   - `drive/policy.go:52-55`: Shift goes with every named key, so Shift+Delete passes Confine. In Blink's editing key table on Windows and Linux this is **Cut**, which writes the operator's clipboard. Clipboard writes are exactly what Confine says it refuses. The README says Linux "probably" works. **This is the 3rd finding in family `confinement-allowlist`** (after Meta+Y/Shift+Escape and key/code mismatch), so the fix should be a rule rather than another single exception. The rule: Confine allows a closed table of chords (key × modifier set), and a chord goes in the table only if no Chrome or Blink binding handles it on any supported platform. A test should walk the full product of `namedKeys` and the modifier sets and check each result against that table, so every allowed chord is a deliberate entry. Prevalence so far: 4 holes in 2 rounds, all in the predicate-style checks at `policy.go:52-66`.

4. **Minor findings**
   - `recording/control.go:152` `applyCtl` ignores `s.Ended`. I probed this: Page → `stop end` → `ctl` request → `ctl` grant reduces with no error to `Ended=true, Driver="v1"`. The atlas says control "ends at `stop end`, which is final". **This is the 2nd finding in family `control-lifetime`.** The rule to state: after `stop end`, no record may change cast state (`seg`, `ctl`). Enforce it in one guard in `State.apply` and test both record kinds.

5. **Test coverage notes**
   - Every claimed behavior fix has a test that fails without it:
     - BR-1: 3 tests fail.
     - BR-2 and BR-3: 5 `TestConfine` cases fail.
     - BR-5 and BR-6: 2 `recording` tests fail.
     - BR-8: 3 `ParseBatch` cases fail.
   - The injector's "dispatch from Key only" promise is documented, not tested; Confine is what enforces it. That's acceptable until #467 builds the real dispatch.

6. **Architectural notes for upcoming work**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. Confine, codeFits and NextControl are pure.
   - **ARCH-PURPOSE:** pass. The atlas now describes code that exists.
   - **ARCH-MOCK:** pass. The fakes compose over the production seam.
   - **ARCH-CONSTRAINTS:** pass. Body, act and text limits exist.
   - **ARCH-SECURE:** flagged (the Shift+Delete finding).
   - **ARCH-ORDER:** flagged twice (the `ctl` after `stop end` finding, and BR-7). BR-7: #467's real controller must take its grace timers as queue inputs, so timer-versus-attach orderings can be replayed.
   - **ARCH-FUNERAL:** pass. Nothing new is durable beyond the fixtures.

7. **Plan revision recommendations**
   - Add the closed chord table rule to the Confine bullet.
   - Add "`stop end` is final for `ctl` as well as `seg`" next to the round-1 entry.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      ErrNotDriver in FakeInjector.Inject plus Sink follow; contract revoke step and TestFakesCompose go red when the check is removed (3 failures).
  - id: BR-2
    disposition: addressed
    note: |
      policy.go refuses Meta+Y and Shift+Escape; reverting policy.go fails both TestConfine cases.
  - id: BR-3
    disposition: addressed
    note: |
      codeFits holds Code to Key (exact US key under Ctrl/Meta); Injector doc states dispatch derives from Key; 3 TestConfine cases go red on revert.
  - id: BR-4
    disposition: addressed
    note: |
      Plan Revisions 2026-10-12 entry names controller_contract.go, fixturegen -driven writing DrivenRecording directly, and the new factory signature, matching the code.
  - id: BR-5
    disposition: addressed
    note: |
      control.go consistent() refuses a driver with an empty id; TestCtlRefusesADriverWithoutAnID goes red on revert.
  - id: BR-6
    disposition: addressed
    note: |
      State.Ended refuses a seg after stop end; the test goes red on revert (sibling ctl-after-end raised separately).
  - id: BR-7
    disposition: not-addressed
    note: |
      Left open on purpose (plan Revisions); FakeController still uses time.AfterFunc. Minor, deferred to #467's queue-driven timers.
  - id: BR-8
    disposition: addressed
    note: |
      act.go range-checks sel against the rune count, and only for ime; 3 ParseBatch cases go red on revert.
findings:
  - id: new
    severity: Important
    family: confinement-allowlist
    title: |
      Confine allows Shift+Delete, which Blink maps to Cut (writes the operator's clipboard) on Windows/Linux
    detail: |
      3rd finding in confinement-allowlist (4 holes in 2 rounds: Meta+Y, Shift+Escape, key/code mismatch, Shift+Delete). Fix the rule, not the instance: make Confine a closed table of allowed key x modifier chords, each allowed only if no Chrome/Blink binding handles it on any supported platform, and have a test walk namedKeys x modifier sets against that table.
  - id: new
    severity: Minor
    family: control-lifetime
    title: |
      A ctl after stop end still reduces, leaving Ended=true with a driver granted
    detail: |
      2nd finding in control-lifetime. Probe: Page, stop end, ctl request, ctl grant reduces with no error to Driver v1. Rule: after stop end no record may change cast state (seg, ctl); enforce it in one guard in State.apply and test both kinds.
```

---

## Re-review — 2026-10-12T01:47:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 466 — broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format |
| repo | pair |
| issue file | workshop/issues/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format.md |
| boundary | whole-issue close |
| milestone | — |
| window | 26292514a113e10cb14354487c2ceb03fe688649..eb5f8af1c0237d4e6995096a37010119c526ce81 |
| command | sdlc close --issue 466 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:47:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on the last round's changes first.

Reviewing is nearly done: I checked the round‑2 fixes (reverting them turns the tests red) and found one remaining gap in the key allowlist. Last check is the plan's Revisions section.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

Round 2 fixed both instances it was asked to fix, and the regression tests prove it. When I reverted `reduce.go`, `TestCtlAppliesWhilePausedAndStopEndEndsControl` failed with "a ctl after stop end reduced: <nil>". The Shift+Delete row in `TestConfine` covers the old policy, which allowed Shift on any named key. `Confine` is now a closed chord table, as the BR-9 rule asked. The webcast test suite passes at head.

What stops a straight SHIP is the table's own contents. It allows Shift, Ctrl and Meta with Enter, and Shift with Space. In Blink these are the keyboard versions of modified clicks: Blink sets the navigation policy (new tab, new window) from the modifiers on the key event that activates a link or a submit button. `Confine` already refuses modified clicks for exactly that reason. On top of that, the new "exhaustive" test checks `Confine` against `chordAllowed`, which `Confine` itself calls, so the walk can never catch a bad table entry. Only the hand-written binding list tests anything real. This is the 4th finding in `confinement-allowlist`.

**1. Strengths**
- `policy.go:17-83`: `Confine` is now a closed table of key × exact modifier set. The old maze of flag checks is gone, and anything not in the table is refused by construction. This is the right structure.
- `reduce.go:166-173`: one guard at the top of `State.apply` blocks both `seg` and `ctl` after `stop end`, which is the rule BR-10 asked for. A test covers it, and that test fails when the fix is reverted.
- The atlas changes (`atlas/webcast-format.md` lines 68, 217, 374-379) match the code.
- `codeFits` (`policy.go:131-153`) closes the key/code mismatch by deriving the allowed code from `key`.

**2. Critical findings**
None.

**3. Important findings**
- **This is the 4th finding in family `confinement-allowlist`** (after Meta+Y / Shift+Escape, the key/code mismatch, and Shift+Delete). Earlier rounds fixed instances, so I'm not asking for this instance to be fixed on its own.
  - **The hole:** `policy.go:26` allows Enter with `shift`, `ctrl` and `meta`, and `printableChords` allows Shift+Space. `policy_test.go:55` even pins Meta+Enter as "submit". On a focused `<a>`, Blink turns Enter into a simulated click that keeps the key event's modifiers: Ctrl/Meta+Enter opens a background tab and Shift+Enter opens a new window. The same applies to Space on a focused submit button, and Shift+Enter in a form field can send the form's result to a new window.
  - **Why it matters:** the table comment says a chord is listed only if no Chrome or Blink binding handles it. These chords break that rule, and they reproduce the exact effect `Confine` refuses for clicks.
  - **The rule to fix:** an activation key (Enter, Space) takes no Ctrl, Meta or Shift, because a modified activation is a modified click. More generally, every pointer act that `Confine` refuses must also be refused in its keyboard form.
  - **The test fix:** `TestConfineIsExactlyTheChordTable` compares `Confine` against `chordAllowed`, which `Confine` calls (`policy.go:107`), so the walk can't fail on a bad entry. It should instead check every chord against a separate, cited list of browser bindings, and that list should include Blink's activation rules.
  - **Plan and atlas:** the "Ctrl/Meta+Enter is allowed (page-handled)" line in the plan's Revisions and line 377 of the atlas need correcting. If Ctrl+Enter submit is a product requirement, record it as a known, accepted escape. Don't describe it as page-handled.

**4. Minor findings**
- BR-7 is still open. It was deferred on purpose to #467 in the plan's Revisions, and that's reasonable for a fake.
- `chords` uses string keys like `"ctrl+shift"`, so its entries must follow the modifier order in `chordOf`. A typo such as `"shift+ctrl"` would silently never match. A typed modifier bitset would make that impossible.

**5. Test coverage notes**
- BR-10: confirmed by reverting the fix, which turns the test red.
- BR-9: the Shift+Delete row in `TestConfine` covers it. A full revert doesn't compile against the new test file, so I relied on reading the old policy code, which allowed Shift on all named keys.
- The exhaustive walk only checks `Confine` against itself (see the Important finding). The real oracle is the 27-entry binding list at `policy_test.go:101-108`.
- `go test ./cmd/internal/webcast/...` passes at head.

**6. Architectural notes**
- **ARCH-DRY:** pass. `namedKeys` is now built from `chords`, so the table is the only source.
- **ARCH-PURE:** pass. `Confine`, `chordAllowed` and `codeFits` are pure.
- **ARCH-PURPOSE:** flag. This is the class-versus-instance problem from the Important finding: the table exists, but nothing independent of the table checks its entries.
- **ARCH-MOCK:** pass. The fakes compose through `recording.Tee` (BR-1 was settled in an earlier round).
- **ARCH-CONSTRAINTS:** pass. Not relevant here.
- **ARCH-SECURE:** flag. The confinement boundary still lets activation keys with modifiers through (the Important finding).
- **ARCH-ORDER:** pass with a note. The "ended" state now has one guard. The grace-timer ordering seam (BR-7) is deferred to #467.
- **ARCH-FUNERAL:** pass. Seats live in memory and are reclaimed after `grace` or at `Close`. The round adds nothing durable.

**7. Plan revision recommendations**
- Add a Revisions entry for round 2:
  - `Confine` became a closed chord table, and Shift+Delete is refused.
  - `stop end` is final for `ctl` as well as `seg`.
  - Correct the "Ctrl/Meta+Enter is allowed (page-handled)" line once the Important finding is resolved.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Still real time.AfterFunc; explicitly deferred to #467 in plan Revisions, acceptable as Minor advisory.
  - id: BR-9
    disposition: addressed
    note: |
      Shift+Delete refused via closed chord table (policy.go:17-83), pinned by TestConfine row; see new finding for remaining table holes.
  - id: BR-10
    disposition: addressed
    note: |
      Single Ended guard for seg and ctl in State.apply (reduce.go:166); control_test.go:147 goes red when reverted.
findings:
  - id: new
    severity: Important
    family: confinement-allowlist
    title: |
      Chord table allows Ctrl/Meta/Shift+Enter and Shift+Space, keyboard twins of the modified clicks Confine refuses; the exhaustive test is tautological
    detail: |
      4th in family. Blink takes the navigation policy from the activating key event's modifiers, so Ctrl/Meta+Enter on a focused link opens a new tab, Shift+Enter opens a new window, and Shift+Space on a submit button sends the form result to a new window. The table's own rule ("no Chrome/Blink binding") is broken, and policy_test.go:55 pins Meta+Enter as allowed. TestConfineIsExactlyTheChordTable compares Confine to chordAllowed, which Confine calls, so it cannot catch a bad entry. Rule to fix: activation keys (Enter, Space) take no ctrl/meta/shift, and every refused pointer act is refused in its keyboard form too. Check each chord against a separate, cited binding list that includes Blink's activation rules, then correct the plan Revisions and atlas line 377.
```

---

## Re-review — 2026-10-12T01:51:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 466 — broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format |
| repo | pair |
| issue file | workshop/issues/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format.md |
| boundary | whole-issue close |
| milestone | — |
| window | 26292514a113e10cb14354487c2ceb03fe688649..8e761007bb68939ffc431b150b68fcdf91fa6f55 |
| command | sdlc close --issue 466 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:51:35-07:00 |
| verdict | SHIP |

## Review

The open findings are BR-7 and BR-11. Let me look at the policy first, then the last commit.

```verdict
verdict: SHIP
confidence: high
```

Round 3 fixes BR-11 properly. The chord table now gives the activation keys (Enter, Space) no modifiers (`policy.go:30-31`). The exhaustive test now compares `Confine` with a hand-written expectation instead of the table it calls (`policy_test.go:89-131`). I checked that the test catches the bug: I put back the round-2 `policy.go` from `eb5f8af1`, and `TestConfine` plus `TestConfineAllowsExactlyTheseChords` both failed on Ctrl/Meta/Shift+Enter and Shift+Space. With the fix restored, everything under `./cmd/internal/webcast/...` passes. The atlas (`atlas/webcast-format.md:374-383`) and the plan's Revisions both describe the new rule. `./cmd/internal/artifactpath` fails in this checkout because the generated `runtimebundle/assets` directory is missing. That is a checkout problem, not this diff: none of the failing paths are in the drive package. BR-7 stays open as a Minor that was deliberately handed to #467. I'm raising one new Minor, in family `confinement-allowlist`, about the table's stated rule being too absolute. Neither blocks shipping.

**1. Strengths**
- `policy.go:17-46`: the closed chord table (each key with its exact modifier set) is the right rule for this family. It replaces round-by-round exceptions with one rule that refuses anything not listed.
- `policy_test.go:89-131`: the expectation is written by hand, and the list of Chrome and Blink bindings cites its sources. This removes the self-confirming check the earlier test had.
- `policy.go:26-29` and the atlas: the click-to-keyboard link (activation keys take the event's navigation policy) is written down as a rule, with its cost stated (no Shift+Enter soft newline).
- `codeFits`: a key's `code` must name that key's own physical key when Ctrl/Meta is held, but other keyboard layouts are still allowed.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **This is the 5th finding in family `confinement-allowlist`.** The table's comment (`policy.go:17-20`) and the atlas both say a chord is allowed "only if no Chrome or Blink binding handles it on any platform". Plain Escape breaks that as written: Chrome's accelerator table maps it to close-find/stop-loading, and it also exits fullscreen. The effects are harmless, so don't remove Escape. Instead, restate the rule as what the code actually enforces: *no binding whose effect lands outside the page's document* (navigation history, new tab or window, clipboard, browser UI, devtools, zoom). Then list Escape as a reviewed, harmless binding. With the rule stated that way, later reviews can check against a criterion that can actually be met, rather than an absolute that keeps producing more instances.
- BR-7 is still open: `fake_controller.go:117` uses real `time.AfterFunc`. It is deferred to #467 in the plan's Revisions.

**5. Test coverage**
The BR-11 regression is confirmed: the tests fail without the fix. The expectation map only samples printable characters (`q`, `1`, `/` and a few letters). That's enough, because `chordAllowed` treats every printable character the same way.

**6. Architecture**
- ARCH-DRY: pass.
- ARCH-PURE: pass. `Confine` and the reducer are pure.
- ARCH-PURPOSE: pass. This round fixed the rule, not just the instances.
- ARCH-MOCK: pass. The fakes go through the production boundary (BR-1).
- ARCH-CONSTRAINTS: pass.
- ARCH-SECURE: pass, apart from the Minor rule wording above.
- ARCH-ORDER: flag (Minor, BR-7). The grace-timer ordering against Attach can't be injected into the fake. That belongs with #467.
- ARCH-FUNERAL: pass. Seats are collected after the grace period, and nothing durable is added beyond recording records.

**7. Plan revisions**
Optional: add a line to the round 2–3 Revisions saying the rule is "no effect outside the page", with Escape listed as a reviewed harmless binding.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Still time.AfterFunc at fake_controller.go:117; deferred to issue 467 in plan Revisions; Minor, non-blocking.
  - id: BR-11
    disposition: addressed
    note: |
      policy.go:30-31 Enter/Space take only ""; reverting policy.go to eb5f8af1 turns TestConfine and TestConfineAllowsExactlyTheseChords red on Ctrl/Meta/Shift+Enter and Shift+Space; atlas 374-383 and plan Revisions updated.
findings:
  - id: new
    severity: Minor
    family: confinement-allowlist
    title: |
      Chord table states an absolute rule (no Chrome/Blink binding at all) that plain Escape already breaks
    detail: |
      5th in family. Chrome's accelerator table binds Escape to close-find/stop-loading and it exits fullscreen; the effects are harmless but the stated rule (policy.go:17-20, atlas) is false. Restate the rule as "no binding whose effect lands outside the page's document (history navigation, new tab/window, clipboard, browser UI, devtools, zoom)" and list Escape as a reviewed harmless binding, so future rounds check against a rule that can be met.
```

---

## Re-review — 2026-10-12T01:53:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 466 — broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format |
| repo | pair |
| issue file | workshop/issues/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format.md |
| boundary | whole-issue close |
| milestone | — |
| window | 84a1b2ad7189f9c505089e950cc42c11ec233e92..38a1369bc124dc3c505d338861971a3574531ec5 |
| command | sdlc close --issue 466 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:53:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This window only fixes prose: the chord-table comment in `cmd/internal/webcast/drive/policy.go:17-23` and its atlas twin at `atlas/webcast-format.md:374-377`, plus updates to the gate and review ledgers. No behavior changed, so the fix needs no regression test. The issue file only records the close, `actual_hours` and the switch from quick to full flow. `## Spec`, `## Revisions` and `## Done when` are unchanged, so scope did not change. BR-12 is fixed at the two places it named: the rule now covers only effects outside the page's document, and Escape is listed as a reviewed harmless binding. Three other places still state the old absolute rule, so I dispose BR-12 `not-addressed` (Minor, non-blocking). BR-7 is untouched and is still deferred. The window introduces no Critical or Important defect.

1. **Strengths**
   - The restated rule can actually be met and is testable. It lists the outside-the-document effect classes (history, tab or window, clipboard, browser UI, DevTools, zoom). It also names Escape as a reviewed exception instead of leaving the rule silently false.
   - The atlas and the code comment were changed together and say the same thing.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-12 is fixed only where it was cited (ARCH-PURPOSE). The old absolute wording ("no Chrome or Blink binding handles it") is still in three places:
     - the `Confine` doc comment, `policy.go:102`
     - the plan Revisions, plan.md:191-192
     - `workshop/lessons.md:1144`

     Reword all three to point at the rule in the chord-table comment instead of restating it.
   - `policy.go:23` was not reflowed and is well over the comment width of the lines around it. `gofmt` is clean, so this is cosmetic only.

5. **Test coverage:** this is a prose change, so no new test is needed. The existing tests `TestConfine` and `TestConfineAllowsExactlyTheseChords` already pin Escape as `{""}`, and Shift+Escape as refused.

6. **Architecture**
   - **ARCH-DRY:** flag (Minor). The rule is written out in five places (chord-table comment, `Confine` doc, atlas, plan, lessons), and three of them have drifted. One authoritative statement plus references would stop this from recurring.
   - **ARCH-PURE:** pass. Only comments changed.
   - **ARCH-PURPOSE:** flag (Minor). The fix covers the named instance, not the whole class (see BR-12 below).

7. **Plan revisions:** amend the Revisions bullet at plan.md:191 to the new rule: no binding whose effect lands outside the page's document, with Escape as a reviewed harmless binding.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Unchanged in this window; grace timers still use real time.AfterFunc, deferred to issue 467 per plan Revisions; Minor, non-blocking.
  - id: BR-12
    disposition: not-addressed
    note: |
      Fixed at the cited sites (policy.go:17-23, atlas:374-377), but the old absolute rule remains at policy.go:102 (Confine doc), plan.md:191-192 (Revisions) and lessons.md:1144; reword those to reference the chord-table rule. Minor, non-blocking.
```
