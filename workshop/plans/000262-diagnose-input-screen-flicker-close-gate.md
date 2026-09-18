---
gate: boundary-review
issue: 262
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T20:16:42-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: M1 Done-when "pair term under plain zellij smoked + zellij 2026 honour recorded" unmet, issue not revised
          detail: 'The Log says plain-zellij pair term was not smoked and zellij''s 2026 handling is unrecorded, and all live pair term panes ran the pre-M1 binary. The plan Task 5 box is ticked anyway. "Harmless either way" is unsupported: under plain pair, Emit''s full-pane erase+repaint can still be caught mid-way if zellij ignores 2026 from a pane. Run the operator smoke or add an issue Revisions entry that re-scopes the bullet.'
          family: done-when-waived-without-revision
          round: 1
        - id: BR-2
          severity: Minor
          title: Stale-prose sweep missed "both consumers" siblings (hostty/reserve.go:51-55, couchnestedrows/main.go:4-14)
          detail: reserve.go:52 still says both consumers assert the region and draw the row. The probe header still describes couch and pair term as reserving rows, in present tense. The new reserve.go:15-16 says the region reset is owned "not [by] this file" while this file's Release still writes it for the probe.
          family: unverified-existing-code-claim
          round: 1
        - id: BR-3
          severity: Minor
          title: Plan execution note says caret blink is still open (answered in 72c2bbfc); Task 5 is ticked over an unperformed sub-item
          family: plan-record-stale-after-log
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T20:24:17-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Revisions entry re-scopes bullets 2 and 5 with reasons; the zellij-honours-2026 question is answered by the kept sync_hold.py measurement, whose verdict logic is unit-tested (I could not rerun it live: pty/tmp blocked here).'
          round: 2
        - id: BR-2
          disposition: addressed
          note: hostty/reserve.go:13-19 and :52-57, the couchnestedrows header and runOuter doc are fixed; my phrase-vocabulary re-sweep at head finds only correctly framed text.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Plan Revisions entry (BR-3) corrects the caret-blink note (answered in 72c2bbfc) and states the Task 5 tick means question answered, not smoke run.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: sync_hold.py exits 1 ("NOT honoured") on any crash, so an environment failure reads as a zellij verdict
          detail: Uncaught exceptions exit 1, the code the docstring and README assign to "NOT honoured". Reproduced here twice (/tmp not writable at :37, "out of pty devices" at :71), and a zellij start slower than about 1s gives FileNotFoundError at :104-105. Catch setup/IO failures and exit 2 (inconclusive).
          family: instrument-failure-encoded-as-verdict
          round: 2
      boundary: M1
      blocked: false
    - "n": 3
      timestamp: "2026-09-17T20:36:29-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: main() catches instrument exceptions and exits 2 (sync_hold.py:139-146). InstrumentFailureTest fails with the try/except reverted in a scratch copy and passes at HEAD (4/4). A remaining timing sub-case is raised separately.
          round: 3
      findings:
        - id: BR-5
          severity: Minor
          title: M2 classification rests on two false code claims, repeated in atlas/terminal.md, the issue Log table and history_render.go:316-317
          detail: '(1) "its other writes (mode delta, effects, Copy, release) touch none of it": release writes ?6l, ESC[r, ?7h, SGR0, the OSC8 close, ESC[0 q and ?25h (presenter.go:276-277). The conclusion survives only because release is the presenter''s last write, and the prose should say so. (2) "the region reset is also functional here, since history pushes set 1;2r below": Emit''s push sets 1;2r at :331 and resets it itself at :364, so the preamble''s ESC[r at :318 is only a convergent re-assert. Keep stays the right outcome. This is the 2nd finding in this family this gate, after PQ-1 and BR-2, so fix the class, not these two sites. Rule: a statement that quantifies over code ("none of the N writes do X", "Y is functional because Z") cites evidence for each member in the same commit: each write site''s payload line, or the byte that depends on the sequence. Add the rule to workshop/lessons.md, then re-check every row of the M2 table against it.'
          family: unverified-existing-code-claim
          round: 3
        - id: BR-6
          severity: Minor
          title: Plan Task 5 "Close the milestone" is still unticked (plan:299), though M1 closed in 5961cb1a with a closed-M1 Log line
          detail: 'This is the 2nd finding in this family (BR-3 was the first), so fix the rule, not this instance. Rule: the commit that crosses a boundary reconciles the durable plan with the Log. Every plan row whose evidence is in the Log gets ticked, or a Revisions line says why not. Check it mechanically with grep ''- \[ \]'' over the plan before milestone-close or close. Better still, don''t put an sdlc verb in a plan as a checkbox: the verb records itself (trailer plus Log line).'
          family: plan-record-stale-after-log
          round: 3
        - id: BR-7
          severity: Minor
          title: sync_hold.py still reports "NOT honoured" when the observation window was too short to see the marker
          detail: 'The read deadline is a fixed sleep of SETTLE+HOLD+1.0 from Popen (:104), not a wait relative to t_end. If zellij takes about 1.0s to start the pane, t_end exists but the post-close marker has not been drained yet. marker_after is then None, and verdict() returns 1 (:132-134). This is the 2nd finding in this family (after BR-4, which fixed the crash case), so fix the rule. Rule: a verdict of 0 or 1 needs an observation window that could have seen the other outcome; a missing observation the window can''t account for is 2. Fix: poll for t_end, then wait a fixed grace period past it before reading. Also make test_a_marker_that_never_arrives_is_not_honoured depend on the window having covered that grace period.'
          family: instrument-failure-encoded-as-verdict
          round: 3
      blocked: false
---

# Gate ledger — pair#262 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T20:16:42-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-waived-without-revision` M1 Done-when "pair term under plain zellij smoked + zellij 2026 honour recorded" unmet, issue not revised
  The Log says plain-zellij pair term was not smoked and zellij's 2026 handling is unrecorded, and all live pair term panes ran the pre-M1 binary. The plan Task 5 box is ticked anyway. "Harmless either way" is unsupported: under plain pair, Emit's full-pane erase+repaint can still be caught mid-way if zellij ignores 2026 from a pane. Run the operator smoke or add an issue Revisions entry that re-scopes the bullet.
- **BR-2** [Minor] `unverified-existing-code-claim` Stale-prose sweep missed "both consumers" siblings (hostty/reserve.go:51-55, couchnestedrows/main.go:4-14)
  reserve.go:52 still says both consumers assert the region and draw the row. The probe header still describes couch and pair term as reserving rows, in present tense. The new reserve.go:15-16 says the region reset is owned "not [by] this file" while this file's Release still writes it for the probe.
- **BR-3** [Minor] `plan-record-stale-after-log` Plan execution note says caret blink is still open (answered in 72c2bbfc); Task 5 is ticked over an unperformed sub-item

## Round 2 — 2026-09-17T20:24:17-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Revisions entry re-scopes bullets 2 and 5 with reasons; the zellij-honours-2026 question is answered by the kept sync_hold.py measurement, whose verdict logic is unit-tested (I could not rerun it live: pty/tmp blocked here).
- BR-2 — addressed — hostty/reserve.go:13-19 and :52-57, the couchnestedrows header and runOuter doc are fixed; my phrase-vocabulary re-sweep at head finds only correctly framed text.
- BR-3 — addressed — Plan Revisions entry (BR-3) corrects the caret-blink note (answered in 72c2bbfc) and states the Task 5 tick means question answered, not smoke run.

### Raised

- **BR-4** [Minor] `instrument-failure-encoded-as-verdict` sync_hold.py exits 1 ("NOT honoured") on any crash, so an environment failure reads as a zellij verdict
  Uncaught exceptions exit 1, the code the docstring and README assign to "NOT honoured". Reproduced here twice (/tmp not writable at :37, "out of pty devices" at :71), and a zellij start slower than about 1s gives FileNotFoundError at :104-105. Catch setup/IO failures and exit 2 (inconclusive).

## Round 3 — 2026-09-17T20:36:29-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — main() catches instrument exceptions and exits 2 (sync_hold.py:139-146). InstrumentFailureTest fails with the try/except reverted in a scratch copy and passes at HEAD (4/4). A remaining timing sub-case is raised separately.

### Raised

- **BR-5** [Minor] `unverified-existing-code-claim` M2 classification rests on two false code claims, repeated in atlas/terminal.md, the issue Log table and history_render.go:316-317
  (1) "its other writes (mode delta, effects, Copy, release) touch none of it": release writes ?6l, ESC[r, ?7h, SGR0, the OSC8 close, ESC[0 q and ?25h (presenter.go:276-277). The conclusion survives only because release is the presenter's last write, and the prose should say so. (2) "the region reset is also functional here, since history pushes set 1;2r below": Emit's push sets 1;2r at :331 and resets it itself at :364, so the preamble's ESC[r at :318 is only a convergent re-assert. Keep stays the right outcome. This is the 2nd finding in this family this gate, after PQ-1 and BR-2, so fix the class, not these two sites. Rule: a statement that quantifies over code ("none of the N writes do X", "Y is functional because Z") cites evidence for each member in the same commit: each write site's payload line, or the byte that depends on the sequence. Add the rule to workshop/lessons.md, then re-check every row of the M2 table against it.
- **BR-6** [Minor] `plan-record-stale-after-log` Plan Task 5 "Close the milestone" is still unticked (plan:299), though M1 closed in 5961cb1a with a closed-M1 Log line
  This is the 2nd finding in this family (BR-3 was the first), so fix the rule, not this instance. Rule: the commit that crosses a boundary reconciles the durable plan with the Log. Every plan row whose evidence is in the Log gets ticked, or a Revisions line says why not. Check it mechanically with grep '- \[ \]' over the plan before milestone-close or close. Better still, don't put an sdlc verb in a plan as a checkbox: the verb records itself (trailer plus Log line).
- **BR-7** [Minor] `instrument-failure-encoded-as-verdict` sync_hold.py still reports "NOT honoured" when the observation window was too short to see the marker
  The read deadline is a fixed sleep of SETTLE+HOLD+1.0 from Popen (:104), not a wait relative to t_end. If zellij takes about 1.0s to start the pane, t_end exists but the post-close marker has not been drained yet. marker_after is then None, and verdict() returns 1 (:132-134). This is the 2nd finding in this family (after BR-4, which fixed the crash case), so fix the rule. Rule: a verdict of 0 or 1 needs an observation window that could have seen the other outcome; a missing observation the window can't account for is 2. Fix: poll for t_end, then wait a fixed grace period past it before reading. Also make test_a_marker_that_never_arrives_is_not_honoured depend on the window having covered that grace period.

## Open findings

- **BR-5** [Minor] `unverified-existing-code-claim` M2 classification rests on two false code claims, repeated in atlas/terminal.md, the issue Log table and history_render.go:316-317
- **BR-6** [Minor] `plan-record-stale-after-log` Plan Task 5 "Close the milestone" is still unticked (plan:299), though M1 closed in 5961cb1a with a closed-M1 Log line
- **BR-7** [Minor] `instrument-failure-encoded-as-verdict` sync_hold.py still reports "NOT honoured" when the observation window was too short to see the marker
