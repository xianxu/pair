---
gate: boundary-review
issue: 467
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T07:00:57-07:00"
      dispatched: "2026-10-12T06:56:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'The dialog signal isn''t tied to the cast tab: an open while paused is missed, and any tab''s close clears it'
          detail: 'chrome.go only counts dialogOpened when castTab is set (empty while paused), accepts dialogClosed from any session, and never re-checks on a cast-tab change or destroy. Confirmed with temporary loop tests. The driver''s calls then hang until the 5 s timeout (500, not the fixed sentence) while the recorder loop stalls. Fix: a per-session set of open dialogs in loopState; the signal is "cast tab''s session is in the set", recomputed on every change.'
          family: state-scope-unmodelled
          round: 1
        - id: BR-2
          severity: Important
          title: TestLiveDrive doesn't check the file chooser, the download denial or the restore after SetDriving("")
          detail: live_drive_test.go only checks the file and download clicks return no error. Task 5 and Done-when ask for fileChooserOpened seen with the intercept on, no download, and the intercept off after SetDriving(""). The fake-conn loop test only checks commands are sent, not real Chrome's behavior.
          family: live-conformance-unasserted
          round: 1
        - id: BR-3
          severity: Minor
          title: Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
          detail: server_drive.go limits() clears the whole map at maxSeatLimits; a departed seat's bucket stays until its key's next request. A reset also restores well-behaved seats' budgets. Prune on the controller's leaves, or record the change in Revisions.
          family: residue-bound-on-writer
          round: 1
        - id: BR-4
          severity: Minor
          title: Route budgets (no maxViewers x2 hello cap) and the all-pages file-chooser intercept differ from the plan text
          detail: Bounded equivalently by the hello rate times the grace period; record both in the plan's Revisions.
          family: plan-drift-unrecorded
          round: 1
        - id: BR-5
          severity: Minor
          title: A page the cast tab opens while paused-but-driving isn't closed (castTab is empty while paused)
          family: guard-scope-by-phase
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T07:08:49-07:00"
      dispatched: "2026-10-12T07:05:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Per-session dialogs set, recomputed in publish (chrome.go:545); TestChromeCastTabDialog fails with the old logic restored ("dialog signal never became true").
          round: 2
        - id: BR-2
          disposition: addressed
          note: live_drive_test.go now asserts Page.fileChooserOpened via tap, download denial against a positive control, and no chooser after SetDriving(""); not executed here (Chrome GPU crash on this host).
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: Sweep and Revisions entry landed, but TestDriveSeatTableBounded passes against the old reset (flood keys are dropped on ErrNoSeat, so the table never fills); limits() uses time.Now() inline, so the idle sweep can't be tested — inject now and pin that a full table sweeps idle keys and doesn't refill a seat's budget.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Plan Revisions 2026-10-12 records the route budgets without the maxViewers x2 cap and the all-pages chooser intercept.
          round: 2
        - id: BR-5
          disposition: addressed
          note: castTab drops the PhaseLive gate (chrome_drive.go:90); restoring the gate makes TestChromeCastTabDialog fail ("never got Target.closeTarget T4").
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: loopState.tap is a test-only event hook carried in the production loop struct
          detail: chrome.go:140; nil in production and only set on the loop goroutine, so harmless today. Prefer one injected observer seam if more test hooks accumulate.
          family: test-hook-in-production-state
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 34a9eff353447e2edf56dde561955a1736ed346c
      blocked: false
---

# Gate ledger — pair#467 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T07:00:57-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `state-scope-unmodelled` The dialog signal isn't tied to the cast tab: an open while paused is missed, and any tab's close clears it
  chrome.go only counts dialogOpened when castTab is set (empty while paused), accepts dialogClosed from any session, and never re-checks on a cast-tab change or destroy. Confirmed with temporary loop tests. The driver's calls then hang until the 5 s timeout (500, not the fixed sentence) while the recorder loop stalls. Fix: a per-session set of open dialogs in loopState; the signal is "cast tab's session is in the set", recomputed on every change.
- **BR-2** [Important] `live-conformance-unasserted` TestLiveDrive doesn't check the file chooser, the download denial or the restore after SetDriving("")
  live_drive_test.go only checks the file and download clicks return no error. Task 5 and Done-when ask for fileChooserOpened seen with the intercept on, no download, and the intercept off after SetDriving(""). The fake-conn loop test only checks commands are sent, not real Chrome's behavior.
- **BR-3** [Minor] `residue-bound-on-writer` Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
  server_drive.go limits() clears the whole map at maxSeatLimits; a departed seat's bucket stays until its key's next request. A reset also restores well-behaved seats' budgets. Prune on the controller's leaves, or record the change in Revisions.
- **BR-4** [Minor] `plan-drift-unrecorded` Route budgets (no maxViewers x2 hello cap) and the all-pages file-chooser intercept differ from the plan text
  Bounded equivalently by the hello rate times the grace period; record both in the plan's Revisions.
- **BR-5** [Minor] `guard-scope-by-phase` A page the cast tab opens while paused-but-driving isn't closed (castTab is empty while paused)

## Round 2 — 2026-10-12T07:08:49-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Per-session dialogs set, recomputed in publish (chrome.go:545); TestChromeCastTabDialog fails with the old logic restored ("dialog signal never became true").
- BR-2 — addressed — live_drive_test.go now asserts Page.fileChooserOpened via tap, download denial against a positive control, and no chooser after SetDriving(""); not executed here (Chrome GPU crash on this host).
- BR-3 — not-addressed — Sweep and Revisions entry landed, but TestDriveSeatTableBounded passes against the old reset (flood keys are dropped on ErrNoSeat, so the table never fills); limits() uses time.Now() inline, so the idle sweep can't be tested — inject now and pin that a full table sweeps idle keys and doesn't refill a seat's budget.
- BR-4 — addressed — Plan Revisions 2026-10-12 records the route budgets without the maxViewers x2 cap and the all-pages chooser intercept.
- BR-5 — addressed — castTab drops the PhaseLive gate (chrome_drive.go:90); restoring the gate makes TestChromeCastTabDialog fail ("never got Target.closeTarget T4").

### Raised

- **BR-6** [Minor] `test-hook-in-production-state` loopState.tap is a test-only event hook carried in the production loop struct
  chrome.go:140; nil in production and only set on the loop goroutine, so harmless today. Prefer one injected observer seam if more test hooks accumulate.

## Open findings

- **BR-3** [Minor] `residue-bound-on-writer` Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
- **BR-6** [Minor] `test-hook-in-production-state` loopState.tap is a test-only event hook carried in the production loop struct
