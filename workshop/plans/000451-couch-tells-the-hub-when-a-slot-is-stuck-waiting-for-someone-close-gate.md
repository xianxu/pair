---
gate: boundary-review
issue: 451
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T13:42:43-07:00"
      dispatched: "2026-10-11T13:40:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: dialogTitle returns the line above the FIRST "1." in the tail, so an earlier numbered list in prose is reported as the dialog's title
          detail: wait_notice.go:87-99 scans from the top; tail ["Plan:","1. run tests","2. push","", <permission dialog>] yields "Plan:" (confirmed with a scratch test). Scan from the bottom and add a table case.
          family: tail-reader-scan-direction
          round: 1
        - id: BR-2
          severity: Important
          title: No test covers the failed-admission retry path (episode left unmarked, next tick re-sends)
          detail: Moving w.seen[b]=notified above broker.Send (wait_notice.go:314-317) keeps all tests green and silently drops notices after one refused send. Add a rig test where the hub refuses once, then accepts.
          family: untested-retry-path
          round: 1
        - id: BR-3
          severity: Important
          title: Done-when 3 (live check that the TL receives a notice) is still unchecked at whole-issue close
          detail: Record the operator's live confirmation in the issue log, or explicitly move it to a follow-up, before closing.
          family: done-when-unmet-at-close
          round: 1
        - id: BR-4
          severity: Important
          title: Plan promises a diagnostic log line for an invalid COUCH_WAIT_NOTICE_AFTER; the code falls back to 10m silently
          detail: wait_notice.go:183-188. Either add a plan Revisions entry withdrawing the log line, or wire one in.
          family: plan-code-drift
          round: 1
        - id: BR-5
          severity: Minor
          title: The "binding is the hub" check (WorktreeRoot == authority.hub) is duplicated in message_service.go:562 and wait_notice.go:246
          family: duplicated-predicate
          round: 1
        - id: BR-6
          severity: Minor
          title: Errors reading the wait-notice-off mute list (other than not-exist) are silently treated as no mutes
          detail: wait_notice.go:286 discards the os.ReadFile error; a permission error un-mutes every slot without any signal.
          family: swallowed-io-error
          round: 1
        - id: BR-7
          severity: Minor
          title: hubBinding picks an arbitrary binding if more than one shares the hub worktree (map iteration order)
          family: nondeterministic-selection
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T13:49:31-07:00"
      dispatched: "2026-10-11T13:46:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: dialogTitle now scans from the bottom for the selected option (wait_notice.go:92-105); TestDialogTitleIgnoresNumberedProse goes red when the scan is reverted to top-down.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestWaitNoticeRetriesAfterARefusedAdmission goes red when w.seen is marked before broker.Send (verified by mutation).
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Live check moved to #452 (exists, matching Done-when) in Revisions and the Plan checklist; the Done-when bullet itself is still unedited, raised separately.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: Invalid threshold now refuses the start (message_service.go:217-221, run.go:780-784); plan Revisions records the change; TestWaitNoticeThreshold pins the parse.
          round: 2
        - id: BR-5
          disposition: addressed
          note: messageAuthority.isHub is the one predicate, used at message_service.go:572 and wait_notice.go:256.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Unreadable mute list is reported in the notice body; TestWaitNoticeSaysWhenTheMuteListIsUnreadable covers it.
          round: 2
        - id: BR-7
          disposition: addressed
          note: hubBinding resolves the slot via the registry liveness entry (newest incarnation), not map order.
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: The third Done-when bullet still requires the live TL check that Revisions moved to pair#452
          detail: '2nd finding in this family. Rule: any Done-when item deferred in Revisions must be edited in the Done-when list itself, in the same commit, struck through with a pointer to the issue that now owns it. Strike bullet 3 and point it at pair#452.'
          family: done-when-unmet-at-close
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T13:53:27-07:00"
      dispatched: "2026-10-11T13:49:51-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: addressed
          note: a0a0eb1b strikes Done-when bullet 3 in place with a pointer to pair#452 (issue file exists); Plan checkbox and Revisions agree.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: a0a0eb1bb45a9c5476ab94fd468f043af471fb0c
      blocked: false
---

# Gate ledger — pair#451 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T13:42:43-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `tail-reader-scan-direction` dialogTitle returns the line above the FIRST "1." in the tail, so an earlier numbered list in prose is reported as the dialog's title
  wait_notice.go:87-99 scans from the top; tail ["Plan:","1. run tests","2. push","", <permission dialog>] yields "Plan:" (confirmed with a scratch test). Scan from the bottom and add a table case.
- **BR-2** [Important] `untested-retry-path` No test covers the failed-admission retry path (episode left unmarked, next tick re-sends)
  Moving w.seen[b]=notified above broker.Send (wait_notice.go:314-317) keeps all tests green and silently drops notices after one refused send. Add a rig test where the hub refuses once, then accepts.
- **BR-3** [Important] `done-when-unmet-at-close` Done-when 3 (live check that the TL receives a notice) is still unchecked at whole-issue close
  Record the operator's live confirmation in the issue log, or explicitly move it to a follow-up, before closing.
- **BR-4** [Important] `plan-code-drift` Plan promises a diagnostic log line for an invalid COUCH_WAIT_NOTICE_AFTER; the code falls back to 10m silently
  wait_notice.go:183-188. Either add a plan Revisions entry withdrawing the log line, or wire one in.
- **BR-5** [Minor] `duplicated-predicate` The "binding is the hub" check (WorktreeRoot == authority.hub) is duplicated in message_service.go:562 and wait_notice.go:246
- **BR-6** [Minor] `swallowed-io-error` Errors reading the wait-notice-off mute list (other than not-exist) are silently treated as no mutes
  wait_notice.go:286 discards the os.ReadFile error; a permission error un-mutes every slot without any signal.
- **BR-7** [Minor] `nondeterministic-selection` hubBinding picks an arbitrary binding if more than one shares the hub worktree (map iteration order)

## Round 2 — 2026-10-11T13:49:31-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — dialogTitle now scans from the bottom for the selected option (wait_notice.go:92-105); TestDialogTitleIgnoresNumberedProse goes red when the scan is reverted to top-down.
- BR-2 — addressed — TestWaitNoticeRetriesAfterARefusedAdmission goes red when w.seen is marked before broker.Send (verified by mutation).
- BR-3 — addressed — Live check moved to #452 (exists, matching Done-when) in Revisions and the Plan checklist; the Done-when bullet itself is still unedited, raised separately.
- BR-4 — addressed — Invalid threshold now refuses the start (message_service.go:217-221, run.go:780-784); plan Revisions records the change; TestWaitNoticeThreshold pins the parse.
- BR-5 — addressed — messageAuthority.isHub is the one predicate, used at message_service.go:572 and wait_notice.go:256.
- BR-6 — addressed — Unreadable mute list is reported in the notice body; TestWaitNoticeSaysWhenTheMuteListIsUnreadable covers it.
- BR-7 — addressed — hubBinding resolves the slot via the registry liveness entry (newest incarnation), not map order.

### Raised

- **BR-8** [Important] `done-when-unmet-at-close` The third Done-when bullet still requires the live TL check that Revisions moved to pair#452
  2nd finding in this family. Rule: any Done-when item deferred in Revisions must be edited in the Done-when list itself, in the same commit, struck through with a pointer to the issue that now owns it. Strike bullet 3 and point it at pair#452.

## Round 3 — 2026-10-11T13:53:27-07:00 (claude) — passed

### Disposed

- BR-8 — addressed — a0a0eb1b strikes Done-when bullet 3 in place with a pointer to pair#452 (issue file exists); Plan checkbox and Revisions agree.

## Open findings

(none — every finding has been disposed)
