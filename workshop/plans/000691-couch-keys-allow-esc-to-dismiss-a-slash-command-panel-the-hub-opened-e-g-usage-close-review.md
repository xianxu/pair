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

---

## Re-review — 2026-10-19T23:45:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 691 — couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage) |
| repo | pair |
| issue file | workshop/issues/000691-couch-keys-allow-esc-to-dismiss-a-slash-command-panel-the-hub-opened-e-g-usage.md |
| boundary | whole-issue close |
| milestone | — |
| window | b98802e09dfa33a77fd723d34948d247a5dd2e34..0127de1adbef4af2f798ff5a796d05ab578d8f62 |
| command | sdlc close --issue 691 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T23:45:36-07:00 |
| verdict | SHIP |

## Review

Checking BR-2 now: #716 already added `PanelRefusal` and tested it. Next I'll run the tests and look at the wrapper's dispatch path.

```verdict
verdict: SHIP
confidence: high
```

This reopen fixes both problems the hub's live check measured. First, a panel taller than the screen is now recognized even though its opening rule has scrolled out of view. Second, a lone `esc` is now exempt from the turn stop, which Claude's "working" progress signal keeps active while a panel is open. The exemption is narrow. `panelOwnsInput` (`cmd/internal/agentdrive/model.go:162`) needs `DismissOnly()`, and either an admitted dismissal or a start event with `Dismissable` set. The wrapper's turn branch (`cmd/internal/wrapcmd/agent_drive.go:282`) checks the same `Dismissal()`. The check just before the write (`agent_drive.go:316`) still requires panel proof on the current frame. The agentdrive and wrapcmd tests pass, and so do the drive/keys/skill/panel tests in couchcmd. The full couchcmd run had 2 failing tests, both from my sandbox: `git init` was refused outside the reviewer's directories (sdlc #339), and neither test is in this diff. I re-ran two of the implementor's mutations myself. Removing `!r.state.Dismissal()` in the wrapper fails the wrapcmd test ("dismissal bytes \"\""). Putting the `panel < 0` early return back fails the agentdrive test. BR-2 was already fixed in #716. Nothing blocks.

1. **Strengths**
   - The safety cases are tested as hard as the success case, using real 2.1.304 captures of working turns. There are three of them, each with two variants that place the footer words where they must not count. Through the wrapper they must refuse, write nothing, and not give the panel hint. A guard checks that the capture painted a recognized composer, so a bad replay can't pass by accident (`agent_drive_panel_test.go:132`).
   - `TestAgentDriveDismissalStopsIfThePanelGoes` covers the race where a working turn replaces the panel after admission but before the write. It shows the pre-write check refuses the write.
   - The turn exemption is limited to exactly `[esc]` with panel proof. `TestDismissalToleratesThePanelsTurnFlag` shows that clear-line, double esc, esc on an occupied composer, and raw drives still stop on an active turn.
   - Confirmation still needs `Idle`, so a frame showing the composer while the turn flag is still set keeps the drive watching instead of confirming. Both the reducer test and the wrapper test with `turn=true` cover this.
   - The `MaxKeyTotalWaitMS` constant now drives the skill's limits assertion (`keys_help_test.go:60`).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `cmd/internal/agentdrive/panel_test.go:47`: the "footer words above the composer" variant in `workingTurnScreens` actually puts the footer inside the composer box. It lands after the opening rule, because `composer` finds the closing rule. So it passes without testing the case its name describes. The wrapcmd variant (`agent_drive_panel_test.go:126`) places it correctly above the opening rule. Fix: walk back to the opening rule, as the wrapcmd helper does.
   - `cmd/internal/agentdrive/panel.go:31`: one doc-comment line runs well past the wrapping of the rest of the block.

5. **Test coverage notes**
   - Recognizer: positive on both captures. The negatives now include "footer then other text". The old "footer with no rule" negative was dropped on purpose, because a panel with no rule on screen is now the tall-panel positive.
   - Wrapper: the full grid of short and tall panels, each with and without the turn flag. Every non-esc sequence gets the `PanelRefusal` wording, `esc` is written exactly once, a panel repaint doesn't end the drive, and confirmation comes only after the turn ends.
   - Not covered: a panel opened over a genuinely working turn. The plan lists it as unverified, and the hub said to try it only in a throwaway setup. That belongs to the post-merge live check, already handled under BR-1.

6. **Architecture notes**
   - ARCH-DRY: pass for production code. The admission rule is still the single `Request.Admits`. The two test packages each build the working-screen variants their own way. That's acceptable across packages, but it's how the mislabeled variant above slipped in.
   - ARCH-PURE: pass. The recognizer and the turn-stop exemption are pure and tested without IO. The wrapper only reads the turn flag and the picker flag.
   - ARCH-PURPOSE: pass. Both measured blockers are fixed, and the live check stays with the hub after merge (BR-1).
   - ARCH-MOCK: pass. The wrapper tests run against the harness session fake with replayed real captures. The hub's live check is the conformance step.
   - ARCH-CONSTRAINTS: pass. The 200-row and tail-size limits on the recognizer are kept. No new work on the keystroke path.
   - ARCH-SECURE: pass. Only the viewport is read, unbalanced markup is refused, and the footer has to be the last non-blank row of the panel area.
   - ARCH-ORDER: pass. The exemption goes through `ReduceDrive`, and the wrapper's turn branch keys off `Dismissal()`, which is reducer state. The tests include the panel-goes race. The one gap is the panel-over-a-real-turn row, which the plan documents.
   - ARCH-FUNERAL: pass. The change creates nothing durable beyond test fixtures.
   - ARCH-EVOLVE: pass. No persisted runtime format changed. The test metadata JSON only gained keys.

7. **Plan revision recommendations:** none. The Revised plan and the Log match the code. Item 5 (the live check) stays open by design.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Request.Refusal returns PanelRefusal naming `couch --keys SLOT esc` (recipe.go:91-99, #716); asserted in agentdrive panel_test.go:219, couchcmd agent_drive_service_test.go:407, wrapcmd agent_drive_panel_test.go:64.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: test-variant-fidelity
    title: |
      agentdrive "footer words above the composer" variant actually inserts inside the composer box
    detail: |
      panel_test.go:47 inserts at closing-rule-index minus 1 (after the opening rule), so the variant passes without testing the case its name describes; walk back to the opening rule as wrapcmd agent_drive_panel_test.go:118-126 does.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"965a6ab7-785a-49c1-90e3-271c5790cb9b","repository":"github.com/xianxu/pair","issue_id":"000691","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/16a4005f15d7a965","reviewed_head":"0127de1adbef4af2f798ff5a796d05ab578d8f62","slot":"pair:2","requested_at":"2026-10-20T06:41:47.822109Z"},"phase":"finished","pid":22170,"started_at":"2026-10-20T06:41:48.100993Z","finished_at":"2026-10-20T06:45:35.779041Z","session_id":"e75a8ce4-304d-441e-aed7-d2d5b73c10ab","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.662911375s
head: "0127de1adbef4af2f798ff5a796d05ab578d8f62"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T00:09:24-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 691 — couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage) |
| repo | pair |
| issue file | workshop/issues/000691-couch-keys-allow-esc-to-dismiss-a-slash-command-panel-the-hub-opened-e-g-usage.md |
| boundary | whole-issue close |
| milestone | — |
| window | d5450cd9578396d018b938062b2b3a0841ada13a..a5c7d1cc16da78c133c7ebf4740255058f6d57eb |
| command | sdlc close --issue 691 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T00:09:24-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This window has one code change: the `workingTurnScreens` helper in `cmd/internal/agentdrive/panel_test.go:44-58`. The other changed files are generated ledger and review records for round 3. The fix resolves BR-3. The helper now walks back past the composer's closing rule to its opening rule, puts `"  Esc to cancel"` on the row just above the box, and keeps the screen at 39 rows by dropping the top row. I checked placement by adding a temporary test to this checkout that printed the variant's last rows, then deleted it. For all three working-turn captures (working-thinking, working-after-usage, working-shell), the footer words now sit immediately above the opening `─` rule and the `❯` prompt row, outside the box. The new `Fatalf` guard (`panel_test.go:54`) stops the walk-back from silently landing on the wrong row if a capture changes. `TestDismissablePanel*` passes and `go vet` is clean. The spec, revisions and Done-when are unchanged, and there are no conflict resolutions to judge.

1. **Strengths**
   - `panel_test.go:46-57` now uses the same walk-back as the correct wrapcmd variant (`agent_drive_panel_test.go:113-123`), so the two packages test the same case.
   - The new guard at `panel_test.go:54-56` checks the placement rather than assuming it, which closes off the way BR-3 got in.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage notes:** The "above the composer" variant now tests the case its name describes, and `TestDismissablePanelRefusesEveryWorkingTurn` still refuses all nine screens.
   - On the family rule: this is the only finding in `test-variant-fidelity`, not a repeat. Its rule is that a negative variant must assert its own placement. The guard does that here, and the wrapcmd sibling was already correct, so I found no other instance to fix.
6. **Architecture notes:**
   - **ARCH-DRY:** pass. The fix copies the wrapcmd walk-back instead of sharing a helper with it. That duplication across two test packages is old code outside this window. It could become a shared testdata helper later.
   - **ARCH-PURE:** pass. Test-only change, no IO beyond reading the captures.
   - **ARCH-PURPOSE:** pass. The finding's class was swept: wrapcmd, the only other instance, was already correct.
   - **ARCH-EVOLVE:** pass. No persisted format changed.
7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      panel_test.go:44-58 now walks back to the opening rule (guarded by a prompt-row Fatalf) and inserts above it; probe confirmed footer sits directly above the opening rule on all three captures, length kept at 39; tests pass.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"325d111e-ee4c-423e-90c7-66a1e4222045","repository":"github.com/xianxu/pair","issue_id":"000691","recipe":"fix-review","round":4,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/91f5e0a3336859dd","reviewed_head":"a5c7d1cc16da78c133c7ebf4740255058f6d57eb","slot":"pair:2","requested_at":"2026-10-20T07:08:42.472123Z"},"phase":"finished","pid":67484,"started_at":"2026-10-20T07:08:42.646142Z","finished_at":"2026-10-20T07:09:22.784988Z","session_id":"55c5b588-66df-40cb-8a15-0f87b10439ad","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.238876875s
head: "a5c7d1cc16da78c133c7ebf4740255058f6d57eb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
