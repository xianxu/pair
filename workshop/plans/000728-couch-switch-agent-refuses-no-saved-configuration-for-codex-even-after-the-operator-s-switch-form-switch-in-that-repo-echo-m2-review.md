# Boundary Review — pair#728 (milestone M2)

| field | value |
|-------|-------|
| issue | 728 — couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo) |
| repo | pair |
| issue file | workshop/issues/000728-couch-switch-agent-refuses-no-saved-configuration-for-codex-even-after-the-operator-s-switch-form-switch-in-that-repo-echo.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 98bfa611e4b61d6aef88bd386805a13cb453e89b..165ab40890ba9c506ecc26b3f93d41018a5db8f4 |
| command | sdlc milestone-close --issue 728 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T16:22:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 delivers literal argument transport, editable forms, retained-target selection and guarded force switching, with passing package tests. Two gaps remain: local cooperative switching loses its pinned wrapper token before the final park check, and force cancellation mutates shared reducer state.

```findings
findings:
  - id: new
    severity: Critical
    family: switch-source-generation-fencing
    title: |
      Preserve the local interrupt phase's source token through the final park guard
    detail: |
      cmd/internal/couchcmd/switch_interrupt.go:49 pins the current wrapper token only in a local variable, while InterruptBeforeAgentSwitch returns only error. Its caller at cmd/internal/couchcore/switchagent.go:443 therefore retains the form's empty request.Source. If the wrapper reconnects after the interrupt phase returns but before the final guard, an unchanged thread record produces the same fingerprint, and inspectSwitchLiveSourceWithOptions accepts the new connection because source is empty. This violates the plan's exact-generation/reconnect refusal contract (ARCH-ORDER, ARCH-SECURE). Carry pinned source evidence across the whole operation and require it at the final guard; sweep ordinary, cooperative and force local/remote paths. Add a controlled reconnect between phase completion and final confirmation and assert zero park/start effects.
  - id: new
    severity: Important
    family: immutable-reducer-state
    title: |
      Copy the prepared preview before clearing its force offer
    detail: |
      cmd/internal/couchtty/menu_switchagent.go:23 and :108 assign through frame.SwitchPrepared.Force, but cloneMenuState at cmd/internal/couchtty/menu.go:1968 copies frames without copying SwitchPrepared. Reducing Escape or Cancel therefore changes the input MenuState and existing snapshots. showMenuPublishing renders such a snapshot after releasing c.mu, so cancellation can also race its Force read at menu_switchagent.go:255 (ARCH-PURE, ARCH-ORDER). Copy the PreparedAgentSwitch value before replacing its Force field, retaining the same one-shot offer object where needed. Test both cancellation routes for input immutability and concurrent snapshot rendering.
```

1. **Strengths**

   - `SwitchArguments` preserves absent versus explicit-empty arguments; receipt tests cover changed argument and interrupt intent.
   - Force consent is local, nonserializable and consumed once. Tests retain the existing completion-plus-helper-death requirement and verify dirty-file preservation.
   - Counted retained-target tests enforce bounded workspace probes while preserving queued/effect revalidation.
   - README, atlas and embedded Couch instructions cover the new controls.

2. **Critical findings**

   Source-token continuity, detailed above. The new interrupt phase pins authority internally but does not return that authority to its caller.

3. **Important findings**

   Shared preview mutation, detailed above. Cancellation must change only the returned reducer state.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Complete `couchcore`, `couchcmd`, `couchmessage`, `couchtty` and `testcouch/...` packages.
   - Focused race tests for interrupt, force, admission, source evidence and target resolution.
   - Affected-reader inventory and diff whitespace checks.
   - Startup idle predicates and synthetic usage-limit conformance test.

   Native fleet tests were gated off; this review does not claim native or operator acceptance. Existing tests omit the two sequences identified above.

6. **Architectural notes**

   - **ARCH-DRY — pass:** argument decoding and existing park/drive owners are reused.
   - **ARCH-PURE — flag:** cancellation mutates shared reducer input.
   - **ARCH-PURPOSE — pass:** the governing M2/M3 split is respected; same-agent execution refuses before silently starting fresh.
   - **ARCH-MOCK — pass:** stateful store, queue, process and drive fixtures exercise the production seams.
   - **ARCH-CONSTRAINTS — pass:** argument bounds, counted target probes and bounded interrupt observation are implemented.
   - **ARCH-SECURE — flag:** local cooperative authority loses its connection-token fence.
   - **ARCH-ORDER — flag:** token continuity and immutable snapshots need the identified corrections.
   - **ARCH-FUNERAL — pass:** force capabilities remain scoped to existing operation owners; no new durable production artifact family appears.
   - **ARCH-EVOLVE — pass:** this range retains raw-preserving preference encoding and consolidates strict argument-array decoding.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying operation-wide source-token ownership and immutable preview updates. Update the source-evidence mapping to identify where the live connection token is carried, and name the reconnect and cancellation regression sequences required before M2 closes.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0d7930a8-d074-46f8-a5fb-44c448735f85","repository":"github.com/xianxu/pair","issue_id":"000728","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/49cb243041941ffe","reviewed_head":"165ab40890ba9c506ecc26b3f93d41018a5db8f4","slot":"pair:3","requested_at":"2026-10-22T23:17:27.172147Z"},"phase":"finished","pid":64008,"started_at":"2026-10-22T23:17:27.273012Z","finished_at":"2026-10-22T23:22:26.688002Z","session_id":"01a165e7-b97b-7580-b74a-b55cd19b1a94","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.88521675s
head: "165ab40890ba9c506ecc26b3f93d41018a5db8f4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
