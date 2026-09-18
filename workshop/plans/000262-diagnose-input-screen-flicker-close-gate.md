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

## Open findings

- **BR-4** [Minor] `instrument-failure-encoded-as-verdict` sync_hold.py exits 1 ("NOT honoured") on any crash, so an environment failure reads as a zellij verdict
