# Boundary Review — pair#363 (milestone M2)

| field | value |
|-------|-------|
| issue | 363 — Slot-world switcher: resume and reboot replace thread actions |
| repo | pair |
| issue file | workshop/issues/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d396a7bb143b10a0f445cdb658bedf7c7497294e..e6ab6bbe1c465d9300dbd6d5edbef36b7a0c3db8 |
| command | sdlc milestone-close --issue 363 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-03T01:17:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M2 delivers what the plan says: the switcher speaks the slot model. `menuRowActions(menuRowFactsOf(row))` in `cmd/internal/couchtty/menu_actions.go` is now the only place that decides a row's actions. Enter (`enterOperationFor`), mouse click, the Tab list, the park/relaunch chords, confirmation re-checks and frame reconciliation all read that table instead of repeating its rules. The deleted operations (`open-slot`, `fresh-slot`, `name`, `describe`, `archive`, `recover-thread`, `recover-checkpoint`) are gone from declarations, executors, `run.go` policy, README and atlas. A grep for the removed operation names across non-test Go code returns nothing. Labels and matching no longer read the stored name or description. I ran the M2-relevant tests in couchtty, couchcore, couchcmd and checkpoint, and they pass. The only failures in the full package run are PTY or socket tests, which report `operation not permitted` (34 failing tests, 46 such lines) — the sandbox, not the code. Nothing blocks the boundary; the findings below are Minor.

**Strengths**
- `menu_actions_test.go:31` `everyMenuRowShape` builds its row list from `AllThreadStates`, `AllThreadReasons` and `AllPhases`. `expectedRowActions` restates the Spec independently rather than calling production code. The two sweep tests and the Enter test iterate the same list, so a new state, reason or phase is covered without anyone adding it by hand.
- `menuActionItems` deliberately does not filter what it offers through the declarations, so the "offered implies declared" test can still fail. That design is kept and documented.
- The confirmation and reconcile paths now use one rule: keep a frame while its row still offers its action. There is one exemption, for a frame whose own operation is in flight (`menuFrameOperationInFlight`), and it matches both target and action. `TestRebootFrameSurvivesItsOwnStateChange` covers the `:0` new-tag case and the slot-turns-live case, and also checks that a result arriving under the new tag still completes the attempt.
- The tab bar and switcher labels for `:0` both come from `PresentThreads` (`thread_presentation.go:108`). The pane-label transport through `Name` in `console_presentation.go` is removed.
- `retiredResult` is now the single place both reboot kinds report what retirement did not do, and `RebootResult.Warning` reuses archive's wording.

**Critical**
- None.

**Important**
- None.

**Minor**
- **`:0` path-missing advice names add slot.** The text is in `reboot_decision.go:47`, surfaced at `menu_actions.go:170` and in the confirmation at `menu.go:1316`. On a `:0` row the table never offers add-slot (only live `:0` rows get it), and add slot creates `:N` slots, not a primary checkout. This is the 3rd finding in family `refusal-names-unoffered-action`. Rather than fixing this one string, state the rule: any next step a row tells the operator must be an action that row's kind can reach, with the text chosen by kind. The 2026-10-02 Log's operator decision reads as written for slots, so confirm the `:0` wording with the operator.
- **Stale comment at `menu.go:1636`.** It still says "relaunch and archive deliberately keep theirs"; archive no longer exists, and reboot is now the action that keeps its confirmation on failure.

**Test coverage**
- The Done-when row kinds for M2 (live `:0` and `:1+`, parked/detached, unusable, continuation pending) are covered by the derived-domain table. Non-Git is M3's (Task 3.4).
- The new Enter, busy, path-missing, unknown, filter and label tests pin behaviour, not mocks.
- The "transcript cannot come back → names reboot" path is covered by `TestStartFormOpenOfAnEmptySlotNamesReboot`. The table test also asserts that every row offering resume offers reboot.

**Architecture**
- **ARCH-DRY (pass):** the old per-action special cases (`menuArchiveOffered`, `slotFreshOffered`, the archive/fresh-slot guards) collapsed into the table.
- **ARCH-PURE (pass):** `menuRowFactsOf`, `menuRowActions` and `menuRowNotice` are pure and tested without IO.
- **ARCH-PURPOSE (pass, with the `:0` text caveat):** the sweep of stored `Name`/`Description` readers is clean, and the CLI `show`/`list` renderers don't print them.
- **ARCH-MOCK (pass):** no new external dependency.
- **ARCH-CONSTRAINTS (pass):** routed resume's extra classify round is now recorded in the plan's Revisions.
- **ARCH-SECURE (pass):** reboot by exact tag reaches unreadable records through `resolveThreadForArchive`, and an unreadable record never stops a session.
- **ARCH-ORDER (pass):** in-flight frame ownership is an explicit predicate tested across the replace event. The attempt-keyed match for reboot (`menuOperationMatches`) handles a result arriving under a new address.
- **ARCH-FUNERAL (pass):** nothing new is created on disk; the removed operations leave no residue.

**Notes for upcoming work**
- Retiring a thread outside the TUI is no longer possible: `archive` is gone and `reboot` accepts only the switcher's implicit tag or path, with no CLI `ref`. The unreadable-record refusal in `couch.go` works around this with "run couch in another repository". That is fine for now; if a headless retire is ever needed, give reboot an optional `ref` argument.
- `RecoverThread`'s `path` parameter remains deferred cleanup, as logged.

**Plan revisions**
- None required. The 2026-10-03 Revision already records the deviations (`:1+` path-missing offers nothing, matching by attempt, "may survive" wording).

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Overtaken: tasks 1.x and 2.x are executed and their tests exist; compressing completed prose now has no value.
  - id: BR-2
    disposition: addressed
    note: |
      TestRebootPrimaryCrashAfterJournalRecovers installs leakFirstClaim and asserts the next reboot starts past the leaked address (reboot_test.go:107-170).
  - id: BR-3
    disposition: addressed
    note: |
      menuRowNotice reads menuPhaseBusy for both threadStateText and enterRefusalNotice; pinned by TestBusyRowSaysStartingElsewhere.
findings:
  - id: new
    severity: Minor
    family: refusal-names-unoffered-action
    title: |
      A :0 path-missing row and its reboot confirmation say "add slot recreates it", which that row cannot reach
    detail: |
      3rd in family. Rule: a row's next-step text must name an action reachable from that row's kind (slot vs primary), chosen per kind next to the action table. Here a :0 never offers add-slot unless live, and add slot does not recreate a primary checkout (reboot_decision.go:47, menu_actions.go:170, menu.go:1316). Confirm the :0 wording with the operator.
  - id: new
    severity: Minor
    family: removed-action-sweep-misses-comments
    title: |
      menu.go:1636 comment still says relaunch and archive keep their confirmation on failure
    detail: |
      Archive was removed; reboot is now the action that keeps its confirmation. The Task 2.6 sweep grepped strings, not comments.
```
