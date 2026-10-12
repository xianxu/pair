---
gate: boundary-review
issue: 456
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T23:41:03-07:00"
      dispatched: "2026-10-11T23:38:52-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Live reboot parks the agent before the fresh-profile preflight, breaking Reboot's ORDER contract
          detail: 'rebootAdmit (reboot.go:94-99) calls parkForReboot before resolveStartResolution/slotLaunchProfile/ValidateFreshAgentArgs run, so a profile failure stops the live agent, starts nothing, and reports "so nothing was stopped". Split admission: classify (no effects) under the hold, preflight, then park+reclassify just before prepareRetirement; add a regression test with an unresolvable profile on a live row.'
          family: irreversible-effect-before-preflight
          round: 1
        - id: BR-2
          severity: Important
          title: Slot-path live reboot (rebootSlot with require-settled) has no unit test
          detail: TestRebootLiveUnboundThread only exercises tag addressing (rebootPrimary). The rebootAdmit move under the hold in rebootSlot and the park-then-reobserve flow through startFreshSlot are covered only by the operator's live check; add a dispatch with path + require-settled on liveSlotFixture.
          family: effect-path-untested
          round: 1
        - id: BR-3
          severity: Minor
          title: ActorOperationArgs injects forced for any live reboot; recover safety now rests solely on the ActorActions gate
          detail: liveRebootArgs (slot_operation.go:503) is in a shared helper recover also uses (recover_action.go:176). Make the dependency explicit with a comment or a switcher-only helper.
          family: implicit-admission-coupling
          round: 1
        - id: BR-4
          severity: Minor
          title: Done-when names Claude and Codex; unit test binds only claude and the live check's agent is unstated
          family: done-when-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T23:56:53-07:00"
      dispatched: "2026-10-11T23:54:49-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: rebootStopLive now runs after the profile preflight on both kinds (reboot.go:193, :349); TestLiveRebootProfileFailureStopsNothing goes red when the stop is moved back into admission (verified by mutation).
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestRebootLiveSlotByPath dispatches path + require-settled through rebootSlot/startFreshSlot and checks the no-verdict refusal; passes.
          round: 2
        - id: BR-3
          disposition: addressed
          note: liveRebootArgs carries a comment naming ActorActions as recover's guard (slot_operation.go:601-607); the core still refuses verdict-less live calls.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Log concedes Codex was not exercised; no unit test binds codex or BindingProvisional (withRebootHint's provisional branch untested). Minor, non-blocking.
          round: 2
      findings:
        - id: BR-5
          severity: Important
          title: atlas/couch.md live-reboot paragraph still says rebootAdmit re-checks busy and parks, contradicting the BR-1 ordering fix
          detail: Since BR-1, busy re-check/park/reclassify live in rebootStopLive after the fresh-profile preflight; rewrite the atlas paragraph, and the Plan Design line that says the same, to state admit (no effect), then preflight, then stop, then prepareRetirement.
          family: fix-not-swept-to-docs
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T23:59:50-07:00"
      dispatched: "2026-10-11T23:58:44-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: atlas/couch.md:1122-1128 and table row :2950, reboot.go:57-59, actionableinventory.go:467 and the Plan Design line all state admit, then preflight, then rebootStopLive, then prepareRetirement, matching reboot.go:167/198/202 and :305/333/338; a grep finds no stale wording.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: TestWithRebootHint covers the provisional hint branch, but no live reboot test binds a Codex profile (all use Agent "claude"); parametrize TestRebootLive* over claude/codex, or narrow Done-when / add a Revision.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 282fba74043d8848c6878dd798b0845daabb829b
      blocked: false
---

# Gate ledger — pair#456 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T23:41:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `irreversible-effect-before-preflight` Live reboot parks the agent before the fresh-profile preflight, breaking Reboot's ORDER contract
  rebootAdmit (reboot.go:94-99) calls parkForReboot before resolveStartResolution/slotLaunchProfile/ValidateFreshAgentArgs run, so a profile failure stops the live agent, starts nothing, and reports "so nothing was stopped". Split admission: classify (no effects) under the hold, preflight, then park+reclassify just before prepareRetirement; add a regression test with an unresolvable profile on a live row.
- **BR-2** [Important] `effect-path-untested` Slot-path live reboot (rebootSlot with require-settled) has no unit test
  TestRebootLiveUnboundThread only exercises tag addressing (rebootPrimary). The rebootAdmit move under the hold in rebootSlot and the park-then-reobserve flow through startFreshSlot are covered only by the operator's live check; add a dispatch with path + require-settled on liveSlotFixture.
- **BR-3** [Minor] `implicit-admission-coupling` ActorOperationArgs injects forced for any live reboot; recover safety now rests solely on the ActorActions gate
  liveRebootArgs (slot_operation.go:503) is in a shared helper recover also uses (recover_action.go:176). Make the dependency explicit with a comment or a switcher-only helper.
- **BR-4** [Minor] `done-when-coverage` Done-when names Claude and Codex; unit test binds only claude and the live check's agent is unstated

## Round 2 — 2026-10-11T23:56:53-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — rebootStopLive now runs after the profile preflight on both kinds (reboot.go:193, :349); TestLiveRebootProfileFailureStopsNothing goes red when the stop is moved back into admission (verified by mutation).
- BR-2 — addressed — TestRebootLiveSlotByPath dispatches path + require-settled through rebootSlot/startFreshSlot and checks the no-verdict refusal; passes.
- BR-3 — addressed — liveRebootArgs carries a comment naming ActorActions as recover's guard (slot_operation.go:601-607); the core still refuses verdict-less live calls.
- BR-4 — not-addressed — Log concedes Codex was not exercised; no unit test binds codex or BindingProvisional (withRebootHint's provisional branch untested). Minor, non-blocking.

### Raised

- **BR-5** [Important] `fix-not-swept-to-docs` atlas/couch.md live-reboot paragraph still says rebootAdmit re-checks busy and parks, contradicting the BR-1 ordering fix
  Since BR-1, busy re-check/park/reclassify live in rebootStopLive after the fresh-profile preflight; rewrite the atlas paragraph, and the Plan Design line that says the same, to state admit (no effect), then preflight, then stop, then prepareRetirement.

## Round 3 — 2026-10-11T23:59:50-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — atlas/couch.md:1122-1128 and table row :2950, reboot.go:57-59, actionableinventory.go:467 and the Plan Design line all state admit, then preflight, then rebootStopLive, then prepareRetirement, matching reboot.go:167/198/202 and :305/333/338; a grep finds no stale wording.
- BR-4 — not-addressed — TestWithRebootHint covers the provisional hint branch, but no live reboot test binds a Codex profile (all use Agent "claude"); parametrize TestRebootLive* over claude/codex, or narrow Done-when / add a Revision.

## Open findings

- **BR-4** [Minor] `done-when-coverage` Done-when names Claude and Codex; unit test binds only claude and the live check's agent is unstated
