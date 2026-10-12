---
gate: boundary-review
issue: 484
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T11:24:42-07:00"
      dispatched: "2026-10-12T11:20:08-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Select path commits focus/active after releasing publishMu, so a concurrent paint can re-panel over the new selection
          detail: 'terminal.go:165-175: publishSelection unlocks before selectActorContext sets c.active/c.focus, and the `already` read at :159 is also unlocked. A notice paint in that gap (switcher to actor) sees IsPanel()==true and calls showMenuPublishing, which re-panels over the endpoint just selected and sets focus=Panel; selectActorContext then overwrites focus to actor and its paintNow gets a routing answer that is silently dropped. Result: the screen shows the switcher while focus is the actor. All snapshot/commit sites: paintNow ok, showMenuPublishing ok, onResize ok, publishSelection leaks its commit. Fix: do the `already` read and the active/focus/menu.ActiveAddress commit inside publishSelection under publishMu.'
          family: publish-commit-outside-ordering-lock
          round: 1
        - id: BR-2
          severity: Important
          title: Only the chrome and resize sites have interleaving tests; select and panel publishes are unpinned
          detail: Hook "select" (terminal.go:134) is never used by a test; showMenuPublishing (console_menu.go:211) has no hook. Removing publishMu from publishSelection or showMenu fails no test. Add a panel hook and one test that parks a select/panel publish while a notice tries to overtake it.
          family: ordering-site-without-regression-test
          round: 1
        - id: BR-3
          severity: Minor
          title: Comments claim invariants the select path does not yet meet
          detail: console.go:1255 still says the mid-transition case comes from showMenu, which now commits focus under publishMu, so only the select gap can reach it. console.go:73 says the lock is held "snapshot-to-commit", which the select path does not do. Update both once the select commit moves under the lock.
          family: stale-concurrency-comment
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T12:54:55-07:00"
      dispatched: "2026-10-12T12:51:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: terminal.go:157-180 holds publishMu from the already-read through the focus/active/ActiveAddress commit; reverting it makes TestPaintDuringSelectionCannotRestoreTheMenu fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Added panel hook (console_menu.go:234) and select/panel tests; reverting each lock makes its test fail (verified in this checkout).
          round: 2
        - id: BR-3
          disposition: addressed
          note: console.go:73 snapshot-to-commit claim now holds on all four sites; the 1255 comment was rewritten (the remaining overclaim is a new Minor finding below).
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: paintNow routing-answer comment says the console's own transitions can no longer reach it, but the attach path sets FocusActor without a presenter Select
          detail: 'This is the 2nd finding in family stale-concurrency-comment. console.go:1255-1261 claims the case is unreachable via the console''s own transitions and names only showMenu and selection. console.go:542 (attach when c.active is empty and the attach is not background) sets c.focus=FocusActor without presenter.Select, so it may still reach this case. Rule for the family: a comment that says a state is unreachable must name the code that writes that state, so it can be checked. Instances in this window: console.go:1255 only (console.go:68-74 and terminal.go:154-156 now match the code).'
          family: stale-concurrency-comment
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: b3980a69db28f0f627ceb120e23e780e7eda243e
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T12:57:57-07:00"
      dispatched: "2026-10-12T12:56:40-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: 'console.go:1256-1263 now names the writers: showMenu/selection commit focus under publishMu, and Attach (console.go:542) can still set FocusActor before a selection presents the endpoint. Verified against the Run path (console.go:688-694).'
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: be0e87d795bbc1f9546834430b89842aadb63e86
      blocked: false
---

# Gate ledger — pair#484 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T11:24:42-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `publish-commit-outside-ordering-lock` Select path commits focus/active after releasing publishMu, so a concurrent paint can re-panel over the new selection
  terminal.go:165-175: publishSelection unlocks before selectActorContext sets c.active/c.focus, and the `already` read at :159 is also unlocked. A notice paint in that gap (switcher to actor) sees IsPanel()==true and calls showMenuPublishing, which re-panels over the endpoint just selected and sets focus=Panel; selectActorContext then overwrites focus to actor and its paintNow gets a routing answer that is silently dropped. Result: the screen shows the switcher while focus is the actor. All snapshot/commit sites: paintNow ok, showMenuPublishing ok, onResize ok, publishSelection leaks its commit. Fix: do the `already` read and the active/focus/menu.ActiveAddress commit inside publishSelection under publishMu.
- **BR-2** [Important] `ordering-site-without-regression-test` Only the chrome and resize sites have interleaving tests; select and panel publishes are unpinned
  Hook "select" (terminal.go:134) is never used by a test; showMenuPublishing (console_menu.go:211) has no hook. Removing publishMu from publishSelection or showMenu fails no test. Add a panel hook and one test that parks a select/panel publish while a notice tries to overtake it.
- **BR-3** [Minor] `stale-concurrency-comment` Comments claim invariants the select path does not yet meet
  console.go:1255 still says the mid-transition case comes from showMenu, which now commits focus under publishMu, so only the select gap can reach it. console.go:73 says the lock is held "snapshot-to-commit", which the select path does not do. Update both once the select commit moves under the lock.

## Round 2 — 2026-10-12T12:54:55-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — terminal.go:157-180 holds publishMu from the already-read through the focus/active/ActiveAddress commit; reverting it makes TestPaintDuringSelectionCannotRestoreTheMenu fail.
- BR-2 — addressed — Added panel hook (console_menu.go:234) and select/panel tests; reverting each lock makes its test fail (verified in this checkout).
- BR-3 — addressed — console.go:73 snapshot-to-commit claim now holds on all four sites; the 1255 comment was rewritten (the remaining overclaim is a new Minor finding below).

### Raised

- **BR-4** [Minor] `stale-concurrency-comment` paintNow routing-answer comment says the console's own transitions can no longer reach it, but the attach path sets FocusActor without a presenter Select
  This is the 2nd finding in family stale-concurrency-comment. console.go:1255-1261 claims the case is unreachable via the console's own transitions and names only showMenu and selection. console.go:542 (attach when c.active is empty and the attach is not background) sets c.focus=FocusActor without presenter.Select, so it may still reach this case. Rule for the family: a comment that says a state is unreachable must name the code that writes that state, so it can be checked. Instances in this window: console.go:1255 only (console.go:68-74 and terminal.go:154-156 now match the code).

## Round 3 — 2026-10-12T12:57:57-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — console.go:1256-1263 now names the writers: showMenu/selection commit focus under publishMu, and Attach (console.go:542) can still set FocusActor before a selection presents the endpoint. Verified against the Run path (console.go:688-694).

## Open findings

(none — every finding has been disposed)
