# Boundary Review — pair#691 (whole-issue close)

| field | value |
|-------|-------|
| issue | 691 — couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage) |
| repo | pair |
| issue file | workshop/issues/000691-couch-keys-allow-esc-to-dismiss-a-slash-command-panel-the-hub-opened-e-g-usage.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3dd13c0ea80aa05d1aff1ec3eb25437a5f082ddb..28bd9ba860eb03dc5f5c0aeb7f6cf97a66b1d0f3 |
| command | sdlc close --issue 691 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T18:13:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Running the package tests now, then a quick mutation check on the reducer's dismissal guard.

Next, a mutation check: I'll remove the reducer's dismissal guard and confirm the tests go red.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code fully covers the recognizer fixture part of Done-when. The one rule, `agentdrive.Request.Admits` (`cmd/internal/agentdrive/recipe.go:353`), is now used by the Couch service, its pre-input guard, the wrapper's start and pre-write checks, and the reducer. The panel recognizer (`cmd/internal/agentdrive/panel.go:131`) reads only the viewport and is built from the real Claude 2.1.304 capture. Only an empty composer confirms the dismissal; a panel still showing reports unconfirmed, and nothing retries. All of this is tested. In this checkout `agentdrive`, `wrapcmd`, `couchmessage`, `artifactpath` and `couchcmd -run Drive` pass. The full `couchcmd` run fails only in tests that call `git init` outside the sandbox (sdlc #339), which this diff doesn't touch. I reverted two guards and both went red as expected: dropping the reducer's dismissal guard failed `TestDriveDismissalEndsDoneOrUnconfirmed/unrecognized frame` and `TestAgentDrivePanelAdmitsOnlyEsc`, and dropping the numbered-choice check failed the "numbered row in panel" negative. The one thing between this and SHIP is the second half of the first Done-when bullet, "plus a live check". The Plan still lists it unchecked (`[ ] Live check after landing`), and this is a whole-issue close.

Done-when: current — both bullets (Esc closes the panel and the slot returns to an idle composer; other keys refuse) still describe what the diff builds. The outcome-timing revision only makes "idle composer returns" the sole confirmation, which matches the first bullet.

1. **Strengths**
   - `Request.Admits` is one shared admission rule, and the old hand-copied `Idle || recoveryOnly && Recoverable` checks in `couchcmd/agent_drive_service.go:401`, `wrapcmd/agent_drive.go:548` and `agentdrive/model.go:337` were replaced, not left alongside it (passes ARCH-DRY / ARCH-PURPOSE).
   - The recognizer is pure and fails closed. It refuses on invalid UTF‑8, unbalanced markup, more than 200 rows, other agents, no rule, a `❯` row or a numbered row below the rule. The panel is taken as the rows below the last rule, which correctly handles the `❯ /usage` row still visible above it in the capture (`panel.go:145-161`).
   - `agentDriveScreenLocked` now pulls the input-ownership conditions into one `owned` value used for both `Recoverable` and `Dismissable` (`wrapcmd/agent_drive.go:520-530`). `TestAgentDrivePanelProofNeedsInputOwnership` covers turn, picker, buffered input, in-flight reply, peer and partial frame.
   - The reducer handles the uncertainty correctly. A panel repaint after the write proves nothing (`model.go:405-410`), a panel at the deadline is unconfirmed with its tail, and a frame it doesn't recognize also ends unconfirmed (tested at `panel_test.go:305`).
   - Mixed-version setups refuse the Esc rather than break: `DriveObservation.Dismissable` is an added field, the endpoints decode without `DisallowUnknownFields`, and if either side is older the field reads false, so it refuses as before.

2. **Critical findings**
   - None.

3. **Important findings**
   - **Done-when live check not delivered.** The first Done-when bullet requires "recognizer fixture plus a live check", and Plan line `[ ] Live check after landing` is still open. The Log has a replayed recording, but that was evidence the old build refused, not a live run of this build. Fix: have the hub run `--keys SLOT clear-line text:/usage submit` then `--keys SLOT esc` on this build, and record the outcome in the Log. Or, if the live check is meant to run after close, add a Revision that splits it out of Done-when explicitly.

4. **Minor findings**
   - `s.dismissal = !Idle && !Recoverable` (`model.go:340`) is correct only because `Admits` has already passed. `= event.Screen.Dismissable && !Idle && !Recoverable` would say what it means (ARCH-ORDER readability).
   - The pre-write check (`wrapcmd/agent_drive.go:566`) now lets an `[esc]` admitted as Recoverable go ahead if the screen turned into a panel before the write. Esc there is harmless, but that drive then confirms by the raw any-fresh-frame rule, not the composer-return rule. This is an unlikely ordering.
   - Viewports taller than 200 rows silently fall back to refusing (`panel.go:132`). That fails closed but isn't documented anywhere.
   - The "owned" conditions still appear three times: `driveAvailableLocked`, `owned` in `agentDriveScreenLocked`, and `common` in dispatch. This predates the diff, and the diff slightly reduces it.

5. **Test coverage notes**
   - The recognizer has a positive test on the live capture, plus 11 negatives that include the captured closed panel and `/model` picker.
   - The admission matrix runs at three layers: pure `Admits`, the Couch service with its guard through the drive fake (`TestDrivePanelAdmitsOnlyEsc`), and the wrapper through the harness fake.
   - The reducer covers four outcomes: done, panel persists, unrecognized frame, and refusal of other keys.
   - The permission-prompt negative is synthetic, not captured. That's acceptable, since the numbered-row rule is the guard and a mutation test pins it.

6. **Architectural notes**
   - ARCH-DRY: pass (one rule shared across layers). ARCH-PURE: pass (recognizer, `Admits` and the reducer are pure; the wrapper only does glue). ARCH-PURPOSE: pass on the code; the Spec's `couch --usage` alternative and other agents' panels are fairly declared as separable follow-ups. Flagged only for the live check above.
   - ARCH-MOCK: pass. Tests use the drive service fake and the harness session fake, and the live check is the stated conformance step, which is still pending.
   - ARCH-CONSTRAINTS: pass. The keystroke path does a bounded viewport scan.
   - ARCH-SECURE: pass. It reads only the viewport, so scrollback can't authorize a key, and unparsable markup refuses.
   - ARCH-ORDER: pass. The reducer stays the only transition owner; one new mode bit is set once at start (see the Minor note).
   - ARCH-FUNERAL: pass. Creates nothing durable beyond test fixtures.
   - ARCH-EVOLVE: pass. The added wire field fails closed.
   - Later work: the footer table `claudePanelFooters` is the place to extend when Claude's wording changes or other agents gain panels. Each new entry should come with a captured fixture, as this one does.

7. **Plan revision recommendations**
   - If the live check stays after landing: add a `## Revisions` entry saying the Done-when live check runs after merge, and close the issue only once the hub's result is in the Log.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: done-when-evidence-missing
    title: |
      Done-when live check (clear-line text:/usage submit, then esc, on a live Claude slot) is not delivered at whole-issue close
    detail: |
      Done-when bullet 1 requires a recognizer fixture plus a live check; the Plan item "Live check after landing" is still unchecked and the Log records only a replay of the old build. Record the hub's live run on this build in the Log, or revise Done-when so the live check is explicitly a post-merge step.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"70517577-9cda-4b46-8adc-dffadf5efacf","repository":"github.com/xianxu/pair","issue_id":"000691","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/eb3d87f963a1b076","reviewed_head":"28bd9ba860eb03dc5f5c0aeb7f6cf97a66b1d0f3","slot":"pair:2","requested_at":"2026-10-20T01:09:53.516252Z"},"phase":"finished","pid":50597,"started_at":"2026-10-20T01:09:53.702198Z","finished_at":"2026-10-20T01:13:16.436478Z","session_id":"4dfa281b-a278-4e52-ad6b-990aa1f7ed4c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.630695334s
head: "28bd9ba860eb03dc5f5c0aeb7f6cf97a66b1d0f3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T18:19:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 691 — couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage) |
| repo | pair |
| issue file | workshop/issues/000691-couch-keys-allow-esc-to-dismiss-a-slash-command-panel-the-hub-opened-e-g-usage.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3dd13c0ea80aa05d1aff1ec3eb25437a5f082ddb..4675381be1ac22635fecf3523b303c7a15f6f433 |
| command | sdlc close --issue 691 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T18:19:52-07:00 |
| verdict | SHIP |

## Review

Tests ran. The couchcmd failures come from the sandbox refusing `git init` outside the review directories; I'm now running the drive and panel tests on their own.

```verdict
verdict: SHIP
confidence: high
```

This change does what the Spec asks. A single rule, `agentdrive.Request.Admits` (`cmd/internal/agentdrive/recipe.go:444`), now decides admission for the Couch service, its pre-input guard, the wrapper's start and pre-write checks, and the reducer. That rule accepts a recognized panel only for exactly `[esc]`. The recognizer reads only the current viewport, using the panel rows below the last full-width rule. It is tested against the hub's real Claude 2.1.304 `/usage` capture, with negatives for a composer, the closed panel, a missing rule, the model picker, a permission prompt, a numbered row, footer text that is only in history, another agent and broken markup. A dismissal is confirmed only when an empty composer comes back. A panel still showing at the deadline reports unconfirmed with its tail, and nothing retries.

I re-ran the tests on the pinned head:
- `agentdrive`, `couchmessage`, `wrapcmd` and `artifactpath` pass.
- In `couchcmd`, every drive and panel test passes. The other `couchcmd` failures are the sandbox refusing `git init` outside the reviewer's directories (sdlc #339), which has nothing to do with this diff.

I also removed the panel term from `Admits` as a mutation check. The `agentdrive`, `couchcmd` (`TestDrivePanel`) and `wrapcmd` (`AgentDrivePanel`) tests all failed, so the tests depend on that rule. BR-1 is answered by a Revisions entry that makes the live check a post-merge step owned by the hub.

Done-when: current. Both bullets still describe what the diff builds (Esc closes the panel, other keys refuse). The 2026-10-19 Revisions entry explicitly moves the live half of bullet 1 to a post-merge hub step, and a failure there reopens the issue. The fixture half is delivered.

1. **Strengths**
   - **One admission rule.** `Admits` replaces four hand-copied predicates (`agent_drive_service.go:492`, `agent_drive.go:639`, `:658`, `model.go:340`), so no layer can admit what another refuses.
   - **Ownership gates factored out.** `agentDriveScreenLocked` (`agent_drive.go:611-621`) puts the input-ownership conditions into `owned` and applies them to both `Recoverable` and `Dismissable`. `TestAgentDrivePanelProofNeedsInputOwnership` covers all six faults.
   - **Repaints don't confirm.** The reducer will not count a panel repaint as success (`model.go:406-411`), and the wrapper test repaints the real capture after the write to show it.
   - **Real-capture evidence.** The fixtures come from a live capture with recorded provenance (`metadata.json`). The 39-row capture exposed and fixed a bug: the old `❯ /usage` prompt row stays visible above the panel.
   - **Docs.** `atlas/couch.md`, `README.md` and the couch `SKILL.md` were all updated in the same range.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The wrapper's refusal text (`agent_drive.go:641`) and the reducer's (`model.go:341`) still say "composer must be empty" even when the screen is a panel and the key isn't `[esc]`. A hint like "only esc dismisses an open panel" would point the hub to the next step.
   - `Screen.Admission` and `DriveObservation.Admission` are twin three-field constructors. This is acceptable because they sit on separate types.

5. **Test coverage**
   - The recognizer, `DismissOnly`, the `Admits` matrix, and reducer sequences (composer returns, panel persists, unrecognized frame, plain esc on an occupied composer, other keys refused) are all covered without IO.
   - The wrapper is tested with the painted capture through the harness fake, and the service and guard are tested through `driveRig`. I confirmed the mutation results myself rather than relying on the Log.

6. **Architectural notes**
   - **ARCH-DRY: pass.** One admission rule.
   - **ARCH-PURE: pass.** The recognizer and reducer are pure; the wrapper only projects state.
   - **ARCH-PURPOSE: pass.** Esc-on-panel is delivered. Codex panels and `couch --usage` are declared, separable follow-ups.
   - **ARCH-MOCK: pass.** Tests use the drive fakes and the live capture; the live conformance check is the hub's post-merge step.
   - **ARCH-CONSTRAINTS: pass.** The scan is capped at 200 rows and each line by `MaxResultTailBytes`.
   - **ARCH-SECURE: pass.** Only the viewport is read, malformed markup is refused, and numbered choices are refused. Future risk: a Claude text-entry dialog whose footer is also exactly `Esc to cancel` would be recognized, and Esc would discard what the operator typed there. Add a negative fixture when such a dialog is captured.
   - **ARCH-ORDER: pass.** The one new `dismissal` field is set only at `EventStart` and goes through `ReduceDrive`. The deadline and confirm paths are sequence-tested.
   - **ARCH-FUNERAL: pass.** Nothing durable is created except test fixtures.
   - **ARCH-EVOLVE: pass.** `DriveObservation.Dismissable` is an additive field, and an older reader that ignores it falls back to refusing.

7. **Plan revisions:** none needed. The Plan's unchecked "Live check after landing" item already matches the BR-1 Revisions entry. If anything, tick it once the hub records the run.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Revisions 2026-10-19 (BR-1 entry) makes the live check an explicit post-merge hub step (a failure reopens the issue); the close is gated on the 2.1.304 fixture tests, which I re-ran green with the panel term mutation going red.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: refusal-guidance-wording
    title: |
      Panel refusals still say "composer must be empty" rather than naming esc as the only admitted key
    detail: |
      agent_drive.go:641 and model.go:341 give the same message for a non-esc key on a recognized panel; a panel-specific hint would point the hub at `--keys SLOT esc`.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"41e1cd41-d451-4353-a89a-891156aff7b8","repository":"github.com/xianxu/pair","issue_id":"000691","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/54fc0b5a3a246cb2","reviewed_head":"4675381be1ac22635fecf3523b303c7a15f6f433","slot":"pair:2","requested_at":"2026-10-20T01:14:42.068531Z"},"phase":"finished","pid":72874,"started_at":"2026-10-20T01:14:42.236741Z","finished_at":"2026-10-20T01:19:51.569289Z","session_id":"20a435e9-cd96-4b07-98d6-8f9b912dc76d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.401558333s
head: "4675381be1ac22635fecf3523b303c7a15f6f433"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
