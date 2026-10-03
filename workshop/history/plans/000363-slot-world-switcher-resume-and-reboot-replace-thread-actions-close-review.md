# Boundary Review — pair#363 (whole-issue close)

| field | value |
|-------|-------|
| issue | 363 — Slot-world switcher: resume and reboot replace thread actions |
| repo | pair |
| issue file | workshop/issues/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11adc0740d0fb9b18d788bd0ace1a1986760f122..b1bf933a2691562e03f2c02e92e2d3ea443482ae |
| command | sdlc close --issue 363 |
| reviewer | claude |
| timestamp | 2026-10-03T12:16:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All nine open findings (BR-4 to BR-12) are fixed, and I'm raising nothing new. I checked each one against the code at `b1bf933a`, not the commit messages. The two newest commits add tests that pass: `go test -run TestRebootOfARolledBack` in couchcore and `-run TestRowAdviceNamesOnlyReachableActions` in couchtty both came back `ok`. For BR-11 the fix is a test that checks a rule, not a one-off edit at the site, which is what the review asked for. Nothing blocks SHIP.

1. **Strengths**
   - `retiredResult` in `cmd/internal/couchcore/reboot.go:195` is now the only place that builds the retirement half of a reboot's result. A :0 reboot and a :1+ reboot therefore can't report `SessionNotStopped` differently, which settles BR-7 for good.
   - `cmd/internal/couchtty/menu_actions_test.go:451` now checks every row: if a row has text for the reboot confirmation, that row must offer reboot. This catches every case like BR-11, not just the one that was found.
   - `withRebootAdvice` is applied at the two places that can reach `OpenSlot`: `cmd/internal/couchcore/slotstart.go:270` and `cmd/internal/couchcore/resume_route.go:123`. So no refusal from opening a slot skips the "reboot" advice.
   - The new rolled-back :0 test at `cmd/internal/couchcore/reboot_test.go:409` checks three outcomes: the refusal message, that nothing was started, and that the older primary is still live (the "rule" field records the mutation evidence).

2. **Critical findings**: none.

3. **Important findings**: none.

4. **Minor findings**: none new. The refusal at `cmd/internal/couchcore/slotrecovery.go:301` says "before starting fresh". That describes what reboot does rather than naming an action that no longer exists, so it is acceptable.

5. **Test coverage**
   - BR-12 is pinned by a test, and the commit message records its mutation check.
   - BR-11 is pinned by the rule-level assertion described above.
   - BR-9's wording is now kept apart per row kind: `RebootDirectoryMissing` for a :1+ slot, `RebootCheckoutMissing` for a :0 row.

6. **Architecture notes**
   - **ARCH-DRY: pass.** Advice text comes from one function, `menuRowAdviceOf`, plus constants in couchcore.
   - **ARCH-PURE: pass.** `DecideReboot`, `ChooseResumeRoute` and `menuRowAdviceOf` are pure.
   - **ARCH-PURPOSE: pass.** Resume and reboot replaced the old thread actions across all consumers; "fresh-slot" survives only in a historical comment at `cmd/internal/couchtty/menu.go:1565`.
   - **ARCH-MOCK: pass.** Tests run through `newTestEnv` and its fake process and session layer.
   - **ARCH-CONSTRAINTS: pass.** The plan's Revisions now state the extra classification round per resume (plan line 972).
   - **ARCH-SECURE: N/A.** No new untrusted input or secrets in this round's change.
   - **ARCH-ORDER: pass.** The case of a reboot rolling back an unfinished start next to an existing primary is now pinned by a test.
   - **ARCH-FUNERAL: pass.** Reboot archives the old record. No new durable files.

7. **Plan revision recommendations**: none.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Reboot is a live action in the switcher now (M2/M3), so the "Tab → reboot" advice at resume_route.go:94 names an action the operator can reach.
  - id: BR-5
    disposition: addressed
    note: |
      fresh-slot is gone; OpenSlot is reached only through slotstart.go:270 and resume_route.go:123, both of which wrap the error with withRebootAdvice.
  - id: BR-6
    disposition: addressed
    note: |
      console_continuation.go:232 now registers the watch for the "resume" operation's ContinuationResult.
  - id: BR-7
    disposition: addressed
    note: |
      rebootSlot and the :0 path both build their result through retiredResult (reboot.go:195), which carries SessionNotStopped.
  - id: BR-8
    disposition: addressed
    note: |
      Plan Revision at line 972 states the extra classifyForAction round per ordinary resume.
  - id: BR-9
    disposition: addressed
    note: |
      A :0 row now says RebootCheckoutMissing ("restore the checkout"); the add-slot text is used only for slot rows (menu_actions.go:189-198), and the advice test enforces reachability.
  - id: BR-10
    disposition: addressed
    note: |
      menu.go:1635 comment now says relaunch and reboot keep their confirmation.
  - id: BR-11
    disposition: addressed
    note: |
      Field removed in c8e5d395; menu_actions_test.go:451 now fails on any row that has reboot-confirmation text but offers no reboot.
  - id: BR-12
    disposition: addressed
    note: |
      TestRebootOfARolledBackStartRefusesBesideALegacyPrimary pins the refusal, that nothing starts, and that the legacy primary stays live; ran green.
```
