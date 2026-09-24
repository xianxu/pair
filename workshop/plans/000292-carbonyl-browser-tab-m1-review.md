# Boundary Review — pair#292 (milestone M1)

| field | value |
|-------|-------|
| issue | 292 — Carbonyl browser tab in the right pane, shared with the agent over DevTools |
| repo | pair |
| issue file | workshop/issues/000292-carbonyl-browser-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 058f2f7894551790276b6506ab121d0bd6bf6ae8..b3185a8d08070877e842d6c3f5f6c2f8ca3621ee |
| command | sdlc milestone-close --issue 292 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-20T17:58:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 implementation and focused tests pass. One Important docs-gate issue remains: README does not document the new browser-tab shortcuts/workflow.

1. Strengths

- Pure browser helpers are well separated and fuzz-tested.
- Stateful fake Carbonyl exercises real PTY/process-group behavior.
- Group-kill coverage includes close, cancellation, delivery failure, EOF, and initialization failure.
- Profile ownership uses birth hashes and retains unknown owners safely.
- Atlas architecture documentation was updated.

2. Critical findings

None.

3. Important findings

- `README.md:145-151` — README still documents Alt+B only as scrollback navigation and contains no Carbonyl browser-tab workflow. Add Alt+B/Shift+Alt+B, URL entry, confirmation, and lifecycle usage. This is an `ARCH-PURPOSE` / docs update gate finding.

4. Minor findings

None.

5. Test coverage notes

Focused packages pass:

`browsertab`, `ptychild`, `procutil`, `termcmd`, and `workbenchshortcut`.

`go vet ./...` passes. M1 live operator smoke testing remains explicitly pending in the plan.

6. Architectural notes

- ARCH-DRY: Pass — shared URL, naming, profile, and strip-field helpers are reused.
- ARCH-PURE: Pass — pure browser policy is separated from PTY/profile IO.
- ARCH-PURPOSE: Flag README documentation gap above.
- ARCH-MOCK: Pass for M1 — stateful fake Carbonyl exists behind the process boundary.
- ARCH-CONSTRAINTS: Pass — launch zoom, FPS, cleanup, and timeout bounds are explicit.
- ARCH-SECURE: Pass — argv execution, private profiles, validated ownership, and bounded URL handling are present.
- ARCH-ORDER: Pass — M1 lifecycle transitions are centralized in `Step`.
- ARCH-FUNERAL: Pass — profiles and process groups have cleanup and dead-owner sweep paths.

7. Plan revision recommendations

None.

```findings
findings:
  - id: new
    severity: Important
    family: user-facing-surface-docs
    title: |
      README does not document the new browser-tab shortcuts and workflow
    detail: |
      `shortcut.go:212,234` adds Shift+Alt+B and Alt+B browser-tab actions, but `README.md:145-151` still documents Alt+B only as scrollback navigation. Add the operator-facing browser-tab usage and confirmation/lifecycle behavior before closing the boundary.
```

---

## Re-review — 2026-09-20T18:06:59-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 292 — Carbonyl browser tab in the right pane, shared with the agent over DevTools |
| repo | pair |
| issue file | workshop/issues/000292-carbonyl-browser-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 058f2f7894551790276b6506ab121d0bd6bf6ae8..42f4eb0986f48226cbf7679ade79b40efdf79974 |
| command | sdlc milestone-close --issue 292 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-20T18:06:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 is substantially implemented: lifecycle cleanup, process-group ownership, shortcuts, URL confirmation, profiles, tests, atlas, and README coverage are present. One Important issue remains: controller shutdown is not bounded as required by the plan, so a close can theoretically hang.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md now documents Alt+b, Shift+Alt+b, remote confirmation, Alt+w lifecycle, and Alt+h at lines 30-37 and 144-145.
findings:
  - id: new
    severity: Important
    family: bounded-controller-shutdown
    title: |
      browserController.Close is unbounded despite the M1 shutdown contract
    detail: |
      cmd/internal/termcmd/browser.go:81 waits indefinitely on c.done, while the plan requires Close to be bounded at 3 seconds. Add a timeout and define the fallback behavior so pair term shutdown cannot hang on a blocked controller effect or child cleanup. ARCH-ORDER / ARCH-CONSTRAINTS.
```

1. Strengths:

- Process-group termination is structurally enforced through `ptychild.Options.KillGroup`.
- Crash cleanup and dead-owner profile sweeping have real process-tree tests.
- URL normalization, local/remote classification, endpoint parsing, and ownership names have fuzz coverage.
- Shortcut routing is tested through both Go and shell/Neovim paths.
- README and atlas were updated for the new surface.

2. Critical findings:

None.

3. Important findings:

See machine-readable finding above.

4. Minor findings:

None.

5. Test coverage notes:

Focused M1 packages pass, and `go vet ./...` passes. Repository-wide `go test ./...` entered a long-running integration portion after several packages and was stopped; no failure was observed.

6. Architectural notes:

- ARCH-DRY: Pass — shared URL, label, naming, ownership, and process-group helpers are used.
- ARCH-PURE: Pass — pure browser helpers and lifecycle transitions are separated from IO.
- ARCH-PURPOSE: Pass — M1 scope is delivered; M2/M3 are explicitly deferred.
- ARCH-MOCK: Pass — fake Carbonyl models process groups, PTY output, profiles, and startup files.
- ARCH-CONSTRAINTS: Flagged by the unbounded controller shutdown finding.
- ARCH-SECURE: Pass — argv arrays, private profiles, loopback DevTools construction, and fail-closed sweeping are present.
- ARCH-ORDER: Flagged by the unbounded `Close` join; lifecycle effects otherwise pass through `Step`.
- ARCH-FUNERAL: Pass — profiles have explicit removal and dead-owner sweep paths.

7. Plan revision recommendations:

None; the plan already states the required 3-second bounded controller close.

---

## Re-review — 2026-09-20T18:12:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 292 — Carbonyl browser tab in the right pane, shared with the agent over DevTools |
| repo | pair |
| issue file | workshop/issues/000292-carbonyl-browser-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 058f2f7894551790276b6506ab121d0bd6bf6ae8..5d574a945b87c040279cec4c31667c7121a5b3fb |
| command | sdlc milestone-close --issue 292 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-20T18:12:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 implementation is broadly sound: lifecycle ownership, process-group cleanup, private profiles, shortcut routing, tests, README, and atlas updates are present. The prior shutdown finding remains open because callers can still block indefinitely after the controller timeout, and the regression test does not exercise `browserController.Close` itself.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      browserController.Close now waits only 3 seconds, but removeTab and closeAll immediately call unbounded child.Close afterward (cmd/internal/termcmd/presentation.go:324-330, 495-498). The only regression test exercises waitBrowserController directly (browser_test.go:40-52), not a blocked controller or the complete shutdown path.
```

1. Strengths

- Process-group killing is structurally integrated into `ptychild`, including EOF, cancellation, delivery failure, and initialization failure paths.
- Profile ownership uses PID birth hashes and deletes only proven-dead owners.
- Browser tabs retain Pair’s tab shortcuts and do not pass them into Carbonyl.
- M1 documentation was updated in both README.md and atlas/architecture.md.
- Focused and race-enabled tests pass for `termcmd`, `ptychild`, `browsertab`, and `procutil`.

2. Critical findings

None.

3. Important findings

- `cmd/internal/termcmd/browser.go:94-99`, `presentation.go:324-330`, `presentation.go:482-498`: prior finding BR-2 remains unresolved. Define bounded fallback behavior that cannot later block in `child.Close`, and add a test that makes the actual controller cleanup hang and verifies the full shutdown path returns within the contract.

4. Minor findings

None.

5. Test coverage notes

The targeted tests and race tests pass. The timeout test is insufficient as a behavioral regression test because it tests the helper function directly rather than `browserController.Close` with a blocked effect.

6. Architectural notes

- ARCH-DRY: pass.
- ARCH-PURE: pass; lifecycle policy is in `browsertab.Step`, with effects in the controller.
- ARCH-PURPOSE: pass for M1; shortcuts, lifecycle, profile cleanup, and docs are delivered.
- ARCH-MOCK: pass; the stateful fake Carbonyl shares the process/PTY seam.
- ARCH-CONSTRAINTS: flag via BR-2; the declared shutdown budget is bypassed by later unbounded child cleanup.
- ARCH-SECURE: pass; profiles are private and owner identity is checked before sweeping.
- ARCH-ORDER: flag via BR-2; controller timeout and child cleanup do not define a bounded failure transition.
- ARCH-FUNERAL: pass; profiles have normal cleanup and dead-owner sweep paths.

7. Plan revision recommendations

Add a `## Revisions` entry documenting the shutdown-timeout fallback: what happens when controller effects do not finish within three seconds, how child cleanup remains bounded, and which residual profile/process cleanup path owns recovery.

---

## Re-review — 2026-09-20T18:18:51-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 292 — Carbonyl browser tab in the right pane, shared with the agent over DevTools |
| repo | pair |
| issue file | workshop/issues/000292-carbonyl-browser-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 058f2f7894551790276b6506ab121d0bd6bf6ae8..671e6b0ab5cbcaf71f567a709585fd91d884849f |
| command | sdlc milestone-close --issue 292 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-20T18:18:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 implementation and targeted tests are strong, including the bounded shutdown fix. However, the durable plan’s Core concepts table claims several M2/M3 entities exist, while `record.go`, `cdp.go`, and `browsercmd` are absent. Per the review contract, this contradiction is blocking and requires plan revision before crossing the boundary.

1. Strengths:

- `browserController.Close` is now bounded to 3 seconds, with regression coverage in `browser_test.go:40`.
- Process-group cleanup and crash reclamation are tested with a stateful fake.
- Profile ownership checks preserve unknown/live owners and only sweep proven-dead owners.
- README and atlas document the new browser-tab surface.
- Targeted packages pass tests; `go vet ./...` completed without reported errors.

2. Critical findings:

- `workshop/plans/000292-carbonyl-browser-tab-plan.md:57-169` — the Core concepts tables list `Record`, `CDPClient`, `RecordStore`, `browsercmd.Run`, and `carbonylconformance` as new entities, but their stated files do not exist at this head. This contradicts the required plan/code entity inventory. Split the table by milestone or revise statuses so M1 does not claim undelivered entities. ARCH-PURPOSE / ARCH-ORDER.

3. Important findings:

- `cmd/internal/termcmd/browser.go:228-230` — the implementation returns a launch error when `ProfileStore.Sweep` fails, while the M1 plan says sweep errors should go to the diagnostic log and continue to profile creation (`workshop/plans/000292-carbonyl-browser-tab-plan.md:2025+`). Decide and document the intended fail-safe behavior, then add a regression test for an unreadable/malformed profile root.

4. Minor findings:

- None.

5. Test coverage notes:

- Passed: `go test ./cmd/internal/termcmd/ ./cmd/internal/browsertab/... ./cmd/internal/ptychild/ ./cmd/internal/procutil/ -count=1`.
- The bounded-close test proves timeout behavior, but the child-close timeout intentionally leaves a blocked goroutine; future work should give that seam cancellation/join semantics.

6. Architectural notes for upcoming work:

- ARCH-DRY: pass — shared URL, label, naming, profile, and process-group helpers are used.
- ARCH-PURE: pass — lifecycle decisions are isolated in `browsertab.Step`; IO remains in controller/profile seams.
- ARCH-PURPOSE: flag — the plan’s all-milestone concept inventory overclaims M1 delivery.
- ARCH-MOCK: pass for M1 — `fakecarbonyl` models the process contract behind the executable seam.
- ARCH-CONSTRAINTS: partial — shutdown is bounded, but tab count and aggregate browser memory remain operator-unbounded.
- ARCH-SECURE: pass — profiles are private, argv is structured, and unknown owners are not deleted.
- ARCH-ORDER: pass for the M1 state subset; `Close` supersedes later events.
- ARCH-FUNERAL: pass for M1 profiles/processes; cleanup and dead-owner sweep paths exist.

7. Plan revision recommendations:

- Add a `## Revisions` entry separating M1-delivered entities from M2/M3 planned entities in the Core concepts and Integration points tables.
- Add a `## Revisions` entry resolving whether profile-sweep failure is fatal or diagnostic-only, and align implementation, tests, and plan text.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      browserController.Close now waits through a 3-second timeout at cmd/internal/termcmd/browser.go:97-107, with regression coverage in browser_test.go:40-53.
findings:
  - id: new
    severity: Critical
    family: plan-core-concepts-drift
    title: |
      The Core concepts inventory claims M2/M3 entities exist although they are absent at the review head
    detail: |
      workshop/plans/000292-carbonyl-browser-tab-plan.md:57-169 lists record.go, cdp.go, browsercmd, and carbonylconformance as new entities, but those paths are absent. Split the inventory by milestone or revise the statuses before crossing M1. ARCH-PURPOSE / ARCH-ORDER.
  - id: new
    severity: Important
    family: sweep-error-policy
    title: |
      Profile sweep error handling differs from the M1 plan
    detail: |
      cmd/internal/termcmd/browser.go:228-230 aborts launch on ProfileStore.Sweep failure, while the plan specifies diagnostic logging and continued launch. Resolve the contract and add a regression test for sweep failure.
```
