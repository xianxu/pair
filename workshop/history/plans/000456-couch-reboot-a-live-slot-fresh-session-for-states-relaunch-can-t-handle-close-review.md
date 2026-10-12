# Boundary Review — pair#456 (whole-issue close)

| field | value |
|-------|-------|
| issue | 456 — couch: reboot a live slot (fresh session), for states relaunch can't handle |
| repo | pair |
| issue file | workshop/issues/000456-couch-reboot-a-live-slot-fresh-session-for-states-relaunch-can-t-handle.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23cbf6b444e8830cbd1590397b4078d683593b41..42774e494a4a982c15bce55d0a43ac37614ef140 |
| command | sdlc close --issue 456 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T23:41:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing — checking how the implicit `require-settled` argument is guarded and how recover dispatches reboot.

I've finished the review: the live-reboot design is sound, but one ordering bug should be fixed first.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** The diff does what the issue asks. A live row is admitted only when the call carries an idle verdict (`RebootAdmitted`). Under the reboot hold it re-checks busy (`ConfirmNotBusy`), parks the slot (`ParkExpected`), classifies it again, and from there runs the existing parked-reboot path. Remote admission reuses `DecideLiveRestart`, and a dirty tree only adds a note. Relaunch's `unbound`/`provisional` refusals now name reboot. Help, README, atlas and the skill file are all updated.

The problem is that this breaks reboot's own stated order. `Reboot`'s ORDER contract (`reboot.go:52-56`) promises: "a profile that cannot resolve refuses with the old agent still running -- reboot never stops an agent and then starts nothing". The new `rebootAdmit` parks the live agent before the fresh-profile preflight runs. If that preflight then fails, the operator's agent is stopped, nothing starts, and the error still says "so nothing was stopped". The fix is a cheap reorder. I'm not calling it Critical because the parked row can be rebooted again.

**Inspection.** I ran the stat, name-status and full diff commands. I read `reboot.go`, `recover_action.go`, `actor_actions.go` and the dispatch for `require-settled`. Targeted `couchcore` tests pass. I could not build `couchtty` in this checkout: `runtimebundle` is missing its generated assets, which is an environment problem, not the diff.

1. **Strengths**
   - Reboot's whole admission is one named predicate, `RebootAdmitted` (`actionableinventory.go:464-470`), and the switcher's offer table is checked against it (`action_agreement_test.go:40`). Archive's rule was not widened, so archive still refuses live rows (ARCH-DRY).
   - `require-settled` is declared `Implicit` (`ops.go:431`), so a CLI caller can't send `forced` themselves. Only `prepareLiveRestart` or the switcher sets it.
   - In `rebootSlot`, admission now runs inside the thread hold (`reboot.go:289-292`). Before, it ran ahead of the hold. That move is required now that admission can park.
   - Recover's reboot step still can't reach a live row: `awaitRecoverStep` gates on `ActorActions`, which offers nothing for `ThreadLive` (`actor_actions.go:65-93`).
   - The exhaustive `DecideLiveRestart` test now includes `OpReboot` and checks that an unclean tree is reported, not refused (`live_restart_test.go:221`).

2. **Critical findings:** none.

3. **Important findings**
   - **Live reboot parks before the preflight, breaking the ORDER contract** (`reboot.go:94-99`, which runs before `reboot.go:176-190` on `:0` and `reboot.go:301-313` on slots).
     - Today `rebootAdmit` parks; only afterwards do `resolveStartResolution` / `slotLaunchProfile` / `ValidateFreshAgentArgs` run, plus `enrollPrimaryResolution` on `:0`.
     - If any of those fail, the live agent is already stopped, nothing starts, and the message "so nothing was stopped" is false.
     - Fix: split admission into a classify step with no side effects, run while the hold is taken, then the preflight, then park and classify again just before `prepareRetirement`. Add a test where a live row's profile fails to resolve and the agent must still be running (ARCH-ORDER).
   - **No unit test for a live reboot through the slot path** (`reboot.go:282-293`).
     - `TestRebootLiveUnboundThread` only addresses by tag, which goes through `rebootPrimary`.
     - `TestPrepareSlotOperationRebootLive` only checks admission; it never runs the slot reboot.
     - So moving `rebootAdmit` inside the hold, and parking a slot then re-observing it, is covered only by the operator's manual check on pair:6. A revert would not turn any test red.
     - Fix: dispatch a slot reboot with `path` and `require-settled` on `liveSlotFixture`, and assert the old record is archived, a fresh agent started, and exactly one busy re-check ran.

4. **Minor findings**
   - `ActorOperationArgs` now quietly adds `forced` for any live reboot (`slot_operation.go:503-507`). The "recover can't stop a live agent" promise in `reboot.go:19-22` now rests only on the `ActorActions` gate. A short comment at `recover_action.go:176`, or a dedicated switcher-only helper, would make that dependency visible.
   - Done-when says "Claude and Codex", but the unit test only binds `claude`, and the log doesn't say which agent pair:6 ran.
   - `RequireSettled` is a string with three values (`""` / `known` / `forced`). That follows the existing pattern, but a small enum would make an invalid value impossible to represent.

5. **Test coverage notes.** The guard decisions are covered: busy, busy-unknown, forced, and the dirty-tree note, through both `DecideLiveRestart` and `PrepareSlotOperation`. Also covered: the busy re-check under the hold refusing with no side effects, the switcher sending `forced` only for live rows, and relaunch's hint text. Missing: a failure after the park (the first Important finding), and the slot path end to end (the second).

6. **Architecture**
   - ARCH-DRY passes: live admission reuses `DecideLiveRestart` and `prepareLiveRestart`, and the retirement reuses the parked path.
   - ARCH-PURE passes: the decisions are pure (`RebootAdmitted`, `DecideLiveRestart`), and IO happens in `parkForReboot`.
   - ARCH-PURPOSE passes: both remote and console routes ship, and both refusal codes carry the hint.
   - ARCH-MOCK passes: tests use the existing `fakeLiveRestartProbe` and the PairLifecycle fakes, and the operator did a live check.
   - ARCH-CONSTRAINTS passes: nothing new on a hot path.
   - ARCH-SECURE passes: the verdict is an `Implicit` argument, so the CLI can't forge it.
   - ARCH-ORDER is flagged: the stop now happens before checks that may still fail (first Important finding).
   - ARCH-FUNERAL passes: nothing new is persisted; the archive record follows its existing lifecycle, and a failed park leaves park's own transaction, which park's existing modes handle.

7. **Plan revision recommendations.** After the fix, add a Revisions entry along these lines: "Live reboot runs the fresh-profile preflight before parking. The park and second classification sit between the preflight and `prepareRetirement`, so the ORDER contract (never stop then start nothing) still holds."

```findings
findings:
  - id: new
    severity: Important
    family: irreversible-effect-before-preflight
    title: |
      Live reboot parks the agent before the fresh-profile preflight, breaking Reboot's ORDER contract
    detail: |
      rebootAdmit (reboot.go:94-99) calls parkForReboot before resolveStartResolution/slotLaunchProfile/ValidateFreshAgentArgs run, so a profile failure stops the live agent, starts nothing, and reports "so nothing was stopped". Split admission: classify (no effects) under the hold, preflight, then park+reclassify just before prepareRetirement; add a regression test with an unresolvable profile on a live row.
  - id: new
    severity: Important
    family: effect-path-untested
    title: |
      Slot-path live reboot (rebootSlot with require-settled) has no unit test
    detail: |
      TestRebootLiveUnboundThread only exercises tag addressing (rebootPrimary). The rebootAdmit move under the hold in rebootSlot and the park-then-reobserve flow through startFreshSlot are covered only by the operator's live check; add a dispatch with path + require-settled on liveSlotFixture.
  - id: new
    severity: Minor
    family: implicit-admission-coupling
    title: |
      ActorOperationArgs injects forced for any live reboot; recover safety now rests solely on the ActorActions gate
    detail: |
      liveRebootArgs (slot_operation.go:503) is in a shared helper recover also uses (recover_action.go:176). Make the dependency explicit with a comment or a switcher-only helper.
  - id: new
    severity: Minor
    family: done-when-coverage
    title: |
      Done-when names Claude and Codex; unit test binds only claude and the live check's agent is unstated
```

---

## Re-review — 2026-10-11T23:56:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 456 — couch: reboot a live slot (fresh session), for states relaunch can't handle |
| repo | pair |
| issue file | workshop/issues/000456-couch-reboot-a-live-slot-fresh-session-for-states-relaunch-can-t-handle.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23cbf6b444e8830cbd1590397b4078d683593b41..8a14b844c34911f0299410fa5dafe237021da4af |
| command | sdlc close --issue 456 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T23:56:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

couchcore tests pass. The couchtty package won't build in this checkout because its generated runtime assets are missing. Next I'm running a mutation check of the BR-1 fix.

I ran the stat, name-status and full code-diff recipes, read the issue's Spec, Plan and Revisions, ran the couchcore tests (all pass), and did a revert-and-test mutation check on the BR-1 fix.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four findings from the first review round are handled in the code. For BR-1, the stop of a live agent now runs only after the fresh profile resolves, on both the primary (`:0`) and slot paths. The new regression test fails when I move the stop back into admission (a "mutation" check): the `primary` subtest went red with `VerifiedPark` set and pid 42 killed. For BR-2, the slot path now has its own test, covering calls with and without the idle verdict. One thing should be fixed before close. The atlas paragraph added in this range still describes the order the BR-1 fix removed: it says `rebootAdmit` re-checks busy and parks. The fix was made in the code but not in the docs that describe it. Codex coverage (BR-4) is still unexercised; that is Minor.

Done-when: current. The only Revision moved the stop after the preflight, inside the same contract. Done-when's items (reboot from the CLI and from Couch, busy and busy-unknown guards, tests, a live check, the operator's smoke test) still cover what the diff builds.

### 1. Strengths
- **Admission vs stop:** `rebootAdmit` (`reboot.go:82-92`) no longer has side effects, and `rebootStopLive` sits right before `prepareRetirement` in both `rebootPrimary` (`reboot.go:193-199`) and `rebootSlot` (`reboot.go:349-353`). Admission in `rebootSlot` now also runs under the reboot hold.
- **Allow/refuse decisions:** the exhaustive `DecideLiveRestart` enumeration now includes `OpReboot` (`live_restart_test.go:142`). A dirty tree is only noted, and the test asserts the note is present (`live_restart_test.go:160`).
- **No live stop without a verdict:** `RebootAdmitted` is a single named rule. A call with no verdict, such as recover's reboot step, still refuses a live row; `TestRebootLiveSlotByPath` checks this on the slot path. The offer-vs-permission agreement test uses the same rule.
- **Remote overrides the switcher:** a remote call's admission result replaces the switcher's default `forced` (`live_restart.go:128-134`), and `TestPrepareSlotOperationRebootLive` pins all four cases: known, busy, unknown, forced.
- **Relaunch hint:** `withRebootHint` copies the refusal rather than mutating it, and only adds the hint for the unbound and provisional codes.

### 2. Critical findings
None.

### 3. Important findings
- **The atlas describes the order the fix removed** (`atlas/couch.md`, the "Reboot of a live row (pair#456)" paragraph). It says: "Under the reboot hold, `rebootAdmit` re-checks busy (`LiveRestart.ConfirmNotBusy`), parks the thread … reclassifies, and continues as a parked reboot." Since BR-1, the busy re-check, park and reclassify happen in `rebootStopLive`, after the fresh-profile check. This is the ordering promise the close review exists to protect. Rewrite it along these lines: "`rebootAdmit` classifies without side effects under the hold; after the fresh-profile check, `rebootStopLive` re-checks busy, parks via `ParkExpected`, and reclassifies, so a profile that can't resolve stops nothing." The fix should also check the other places that restate this order: the Plan's Design paragraph says the same thing ("re-checks busy… parks… reclassifies" inside `rebootAdmit`). The `lessons.md` entry and the Revisions entry are already correct.
  - Family: this is a new rule, not a repeat of `irreversible-effect-before-preflight`. When a fix changes behavior, every document that restates that behavior must move with it.

### 4. Minor findings
- **BR-4 is still open:** no test or live check binds a Codex agent or the `BindingProvisional` state. `withRebootHint`'s provisional branch is untested (only unbound is covered), and that branch is the old-wrapper Codex case from the Problem statement. A cheap fix: one table row in `TestRebootLiveUnboundThread` that uses `SetNativeBinding(…, "codex", BindingProvisional, …)`.

### 5. Test coverage notes
- `go test ./cmd/internal/couchcore/ -run 'Reboot|LiveRestart|PrepareSlotOperation|Relaunch|RequireSettled'` passes.
- I couldn't build `couchtty` in this checkout (`runtimebundle/embed.go: pattern assets/runtime/files: no matching files found`). That's a missing generated asset in the review checkout, not a problem in the diff. So `TestLiveRebootDispatchCarriesTheIdleVerdict` and the menu tests weren't run here; I only read them.
- Mutation check: with the BR-1 fix reverted, `TestLiveRebootProfileFailureStopsNothing/primary` fails. I didn't mutate the slot path separately; its subtest has the same shape.

### 6. Architecture
- **ARCH-DRY: pass.** `withParkLifecycle` was pulled out of `envWithLiveThread` and is now shared, and remote admission reuses `DecideLiveRestart`.
- **ARCH-PURE: pass.** The allow/refuse rules (`RebootAdmitted`, `DecideLiveRestart`'s `OpReboot` branch) are pure functions tested without IO. The park is an IO step behind `PairLifecycle`.
- **ARCH-PURPOSE: flag.** The atlas wasn't swept after the BR-1 fix (the Important finding above). The code itself delivers the CLI, Couch and hint paths the issue asks for.
- **ARCH-MOCK: pass.** Tests run against the stateful `pairlifecycletest` model and `fakeLiveRestartProbe`, which sit behind the production seams.
- **ARCH-CONSTRAINTS: pass.** This is an operator-triggered, one-off action with no hot path, and the park keeps its existing timeouts.
- **ARCH-SECURE: pass.** `require-settled` is an implicit argument. Its two values are checked against constants, and an empty value refuses.
- **ARCH-ORDER: pass after the fix.** All checks that can fail come before the first irreversible effect. If the park itself fails, a clear error tells the operator how to recover ("recover the park (retry, recover or abandon)") instead of failing silently.
- **ARCH-FUNERAL: pass.** Nothing new is stored durably. The archived record goes through the existing archive family, and the park history uses the existing park records.

### 7. Plan revision recommendations
- Rewrite the Plan's "Design:" paragraph so it no longer claims `rebootAdmit` parks: "admits (no effect) → preflight → `rebootStopLive` (`ConfirmNotBusy`, `ParkExpected`, reclassify) → `prepareRetirement`." It should match the existing Revisions entry and the corrected atlas.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      rebootStopLive now runs after the profile preflight on both kinds (reboot.go:193, :349); TestLiveRebootProfileFailureStopsNothing goes red when the stop is moved back into admission (verified by mutation).
  - id: BR-2
    disposition: addressed
    note: |
      TestRebootLiveSlotByPath dispatches path + require-settled through rebootSlot/startFreshSlot and checks the no-verdict refusal; passes.
  - id: BR-3
    disposition: addressed
    note: |
      liveRebootArgs carries a comment naming ActorActions as recover's guard (slot_operation.go:601-607); the core still refuses verdict-less live calls.
  - id: BR-4
    disposition: not-addressed
    note: |
      Log concedes Codex was not exercised; no unit test binds codex or BindingProvisional (withRebootHint's provisional branch untested). Minor, non-blocking.
findings:
  - id: new
    severity: Important
    family: fix-not-swept-to-docs
    title: |
      atlas/couch.md live-reboot paragraph still says rebootAdmit re-checks busy and parks, contradicting the BR-1 ordering fix
    detail: |
      Since BR-1, busy re-check/park/reclassify live in rebootStopLive after the fresh-profile preflight; rewrite the atlas paragraph, and the Plan Design line that says the same, to state admit (no effect), then preflight, then stop, then prepareRetirement.
```

---

## Re-review — 2026-10-11T23:59:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 456 — couch: reboot a live slot (fresh session), for states relaunch can't handle |
| repo | pair |
| issue file | workshop/issues/000456-couch-reboot-a-live-slot-fresh-session-for-states-relaunch-can-t-handle.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23cbf6b444e8830cbd1590397b4078d683593b41..282fba74043d8848c6878dd798b0845daabb829b |
| command | sdlc close --issue 456 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T23:59:50-07:00 |
| verdict | SHIP |

## Review

Focus on the open findings: BR-5 (atlas) and BR-4. Look at the last commit and atlas.

```verdict
verdict: SHIP
confidence: high
```

This round covered the last docs commit (`282fba74`) and checked both open findings against the code. BR-5 is fixed. The atlas paragraph (`atlas/couch.md:1122-1128`), the predicate table row (`atlas/couch.md:2950`), the `Reboot` ORDER comment (`reboot.go:57-59`), the `RebootAdmitted` comment (`actionableinventory.go:467`) and the Plan Design line now all give the same order. That order matches the code: `rebootAdmit` runs with no effect at `reboot.go:167` / `:305`, then the fresh-profile preflight, then `rebootStopLive` at `:198` / `:333`, then `prepareRetirement` at `:202` / `:338`. A grep for the old "re-checks busy … parks" wording finds no stale copies. BR-4 is only half-answered: the new test covers the provisional hint branch, but it never runs a live reboot with a Codex profile. It is Minor and does not block. No new findings.

Done-when: current — the Spec/Revisions change only reorders the internal steps (preflight before stop). The outcomes Done-when lists are unchanged and still cover this diff: live reboot by CLI and from Couch, busy refused, `--force-unknown`, guard tests, a live check, and an operator smoke test.

1. **Strengths**
   - `rebootAdmit` (no effect) is now separate from `rebootStopLive` (has effects). Both the thread path and the slot path share this split, so a profile that can't resolve leaves the agent running (`reboot.go:81-135`).
   - The docs sweep this round covered every place that states the order, not just the one BR-5 named.
   - `TestWithRebootHint` (`reboot_test.go:676`) checks both hinted codes and makes sure the others pass through unchanged, including that the diagnostic code survives.
   - `withRebootHint` copies the refusal value instead of mutating the shared error (`relaunch.go:169-177`).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** BR-4 is still open (see dispose).

5. **Test coverage notes**
   - Ran `go test ./cmd/internal/couchcore -run 'Reboot|Relaunch|LiveRestart|SlotOperation|WithRebootHint'`: passes.
   - `couchtty` and `couchcmd` failed at setup in this checkout: `runtimebundle/embed.go:8: pattern assets/runtime/files: no matching files found`. The generated embed assets aren't in the review checkout, so this is an environment gap, not a failure caused by the diff. I could not run those packages here.

6. **Architecture**
   - ARCH-DRY: pass. The `rebootAdmit` / `rebootStopLive` helpers are shared by the thread and slot paths.
   - ARCH-PURE: pass. `RebootAdmitted` is a pure predicate; the effects are in the stop step.
   - ARCH-PURPOSE: pass. The docs sweep covered every site that states the order.
   - ARCH-MOCK: pass. Tests use the existing park-lifecycle and live-restart fakes.
   - ARCH-CONSTRAINTS: N/A. These are operator-triggered, one-off operations.
   - ARCH-SECURE: pass. No new untrusted input or secrets.
   - ARCH-ORDER: pass. The preflight now comes before the irreversible stop, and the order is pinned by `TestLiveRebootProfileFailureStopsNothing`.
   - ARCH-FUNERAL: pass. Nothing new is created that persists; reboot keeps using the existing retirement journal.

7. **Plan revision recommendations:** none beyond what BR-4 says. Either add a Codex-profile live reboot case, or narrow Done-when / record in Revisions that the Codex live case relies on the path being agent-agnostic.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      atlas/couch.md:1122-1128 and table row :2950, reboot.go:57-59, actionableinventory.go:467 and the Plan Design line all state admit, then preflight, then rebootStopLive, then prepareRetirement, matching reboot.go:167/198/202 and :305/333/338; a grep finds no stale wording.
  - id: BR-4
    disposition: not-addressed
    note: |
      TestWithRebootHint covers the provisional hint branch, but no live reboot test binds a Codex profile (all use Agent "claude"); parametrize TestRebootLive* over claude/codex, or narrow Done-when / add a Revision.
```
