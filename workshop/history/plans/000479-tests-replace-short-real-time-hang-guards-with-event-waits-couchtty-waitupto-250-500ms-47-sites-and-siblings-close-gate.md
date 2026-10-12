---
gate: boundary-review
issue: 479
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T08:07:55-07:00"
      dispatched: "2026-10-12T08:04:38-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Touched files keep fatal 1-3 s time.After guards of the same class as the converted sites
          detail: wrapcmd/terminal_model_test.go:424,778,789,854,891,943,952,1021; terminalqualify/candidate_test.go:103,108,154; wrapcmd/peer_input_test.go:190,348; notifytransport/transport_test.go:54,136,149,437; wrapcmd/orientation_replies_test.go:30,129,137,143,152. Convert each to testwait.After(t); file a follow-up issue for the untouched packages (broadcast, webcast, termcmd, couchmessage, couchcmd, couchcore, ...).
          family: real-time-window-as-event-guard
          round: 1
        - id: BR-2
          severity: Minor
          title: TestSendToUnavailableReaderIsBounded dropped its stopwatch without asserting the bounding outcome
          detail: notifytransport/transport_test.go:319-340 only checks sendErr != nil; the Log claims notifytransport x2 assert outcomes. Either assert the write-deadline error (where the platform produces it) or correct the Log/Revisions entry.
          family: ceiling-replaced-by-outcome
          round: 1
        - id: BR-3
          severity: Minor
          title: testwait unit tests cover only the success path and the margin arithmetic
          detail: Neither the Wait give-up (false) path nor the After nil-channel case with no deadline is exercised; injecting the deadline into Wait would make the give-up path testable.
          family: helper-failure-path-untested
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T08:49:40-07:00"
      dispatched: "2026-10-12T08:48:02-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Sweep matched only guards whose next line is t.Fatal/t.Error; residual positive guards remain in touched packages: wrapcmd/agent_restart_test.go:109 (3s), wrapcmd/shortcut_passthrough_test.go:208 (1s), couchcmd/capture_test.go:235 (30s), couchtty/terminal_pressure_test.go:252 (5s), couchtty/console_messages_test.go:85 (3s deadline var), couchtty/console_pointer_test.go:279 (10s deadline var), couchcore/park_test.go:887 (5s), couchcore/park_lifecycle_live_test.go:193 (5s); couchtty/menu_perf_test.go:266 needs convert-or-exempt. Rule: any timer arm whose body (any statement) fails the test while a sibling arm awaits the event, including hoisted deadline := time.After(N), becomes testwait.After(t). The Log''s "every positive select guard" claim is false until then.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: notifytransport/transport_test.go:345-350 now asserts ENOBUFS or os.ErrDeadlineExceeded; test passes on darwin.
          round: 2
        - id: BR-3
          disposition: addressed
          note: testwait_test.go:41-60 covers waitUntil past give-up (returns false after one poll), afterAt unbounded (nil channel) and afterAt past time (fires); dropping the bounded check would hang the test.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T09:12:02-07:00"
      dispatched: "2026-10-12T09:09:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: A scan of all 37 time.After/NewTimer sites left in the touched packages finds only absence windows, fake clocks or subject bounds; no fatal timer arm remains.
          round: 3
      findings:
        - id: BR-4
          severity: Important
          title: Polling deadlines (time.Now().Add(N) + Fatal) in touched files still guard events with 1-3 s windows
          detail: '2nd finding in this family. Rule: every wall-clock bound whose expiry fails the test while it awaits an event (select arm, hoisted After, polling deadline, ctx timeout) becomes testwait unless the bound is the subject. Touched files: wrapcmd/codex_working_test.go:186 (1 s, verbatim testwait.Until copy), couchcmd/warm_reattach_test.go:208 (also an Until copy), wrapcmd/shortcut_passthrough_test.go:222, wrapcmd/agent_restart_test.go:87, wrapcmd/detach_acceptance_test.go:235, couchtty/console_broadcast_test.go:150, couchtty/console_pointer_test.go:72,221, couchtty/notification_pty_test.go:29, couchtty/terminal_soak_test.go:200, couchcore/couch_test.go:148,1030,1748. Untouched files in touched packages: couchcmd message_deliveries:49, message_lifecycle:170, message_service:257,339,378,652, wait_notice:272,324,465; couchcore provision_lock:117, ptyrunner:17; couchsingleton subprocess:64; wrapcmd launch_watcher:139. About 32 in total. Add polling deadlines to the #485 sweep.'
          family: real-time-window-as-event-guard
          round: 3
        - id: BR-5
          severity: Minor
          title: Stopwatch ceilings remain in touched files (console_broadcast 3 s, provision_subprocess 5 s)
          detail: '2nd finding in this family. Rule: a time.Since(start) > N failure is replaced by the outcome that proves the bound. Instances: couchtty/console_broadcast_test.go:357 (the select on f.done with CloseBlock unreleased already proves the bound; delete the ceiling), couchcore/provision_subprocess_test.go:54 (DeadlineExceeded already asserted; assert how the child ended), launcher/session_reap_test.go:175 (untouched file).'
          family: ceiling-replaced-by-outcome
          round: 3
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 4
      timestamp: "2026-10-12T09:24:04-07:00"
      dispatched: "2026-10-12T09:21:58-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: 'Polling sites all converted, but the rule''s 4th form (ctx timeout) was not swept. The rule: every wall-clock bound in a test whose expiry fails the test while it awaits an event, whatever its syntax (select arm, hoisted After, time.Now().Add deadline, context.WithTimeout/ctx.Done arm), takes testwait''s budget unless the bound is the subject. Sweep by classifying every WithTimeout/After/Now().Add/NewTimer in touched files, not by regex on one form. Remaining: couchtty/terminal_exit_test.go:178 (ctx.Done Fatal arms :195,:201), couchcore/couch_test.go:753, couchtty/console_relaunch_chord_test.go:65, couchtty/terminal_soak_test.go:217, wrapcmd/detach_acceptance_test.go:211, couchcore/recovery_execute_test.go:607, couchcmd/continuation_acceptance_test.go:250, couchcore/couch_test.go:500. Suggest a testwait.Context(t) helper. 3rd finding in family real-time-window-as-event-guard.'
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: 'console_broadcast fixed correctly. provision_subprocess_test.go:49 claims a lost escalation hangs into the binary timeout, but WaitDelay plus the ErrWaitDelay group kill (provision_io.go:72,99) bound it anyway: with the SIGKILL at provision_io.go:83 removed, the test passed in 3.10s. Assert how the child ended, or correct the claim. session_reap_test:175 is deferred to #485.'
          round: 4
      recipe: small-diff-review
      isolation: fenced
      reviewed: d3e54be349829f1b38e0b8ec71d6c0901530b6a6
      blocked: false
---

# Gate ledger — pair#479 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T08:07:55-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `real-time-window-as-event-guard` Touched files keep fatal 1-3 s time.After guards of the same class as the converted sites
  wrapcmd/terminal_model_test.go:424,778,789,854,891,943,952,1021; terminalqualify/candidate_test.go:103,108,154; wrapcmd/peer_input_test.go:190,348; notifytransport/transport_test.go:54,136,149,437; wrapcmd/orientation_replies_test.go:30,129,137,143,152. Convert each to testwait.After(t); file a follow-up issue for the untouched packages (broadcast, webcast, termcmd, couchmessage, couchcmd, couchcore, ...).
- **BR-2** [Minor] `ceiling-replaced-by-outcome` TestSendToUnavailableReaderIsBounded dropped its stopwatch without asserting the bounding outcome
  notifytransport/transport_test.go:319-340 only checks sendErr != nil; the Log claims notifytransport x2 assert outcomes. Either assert the write-deadline error (where the platform produces it) or correct the Log/Revisions entry.
- **BR-3** [Minor] `helper-failure-path-untested` testwait unit tests cover only the success path and the margin arithmetic
  Neither the Wait give-up (false) path nor the After nil-channel case with no deadline is exercised; injecting the deadline into Wait would make the give-up path testable.

## Round 2 — 2026-10-12T08:49:40-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Sweep matched only guards whose next line is t.Fatal/t.Error; residual positive guards remain in touched packages: wrapcmd/agent_restart_test.go:109 (3s), wrapcmd/shortcut_passthrough_test.go:208 (1s), couchcmd/capture_test.go:235 (30s), couchtty/terminal_pressure_test.go:252 (5s), couchtty/console_messages_test.go:85 (3s deadline var), couchtty/console_pointer_test.go:279 (10s deadline var), couchcore/park_test.go:887 (5s), couchcore/park_lifecycle_live_test.go:193 (5s); couchtty/menu_perf_test.go:266 needs convert-or-exempt. Rule: any timer arm whose body (any statement) fails the test while a sibling arm awaits the event, including hoisted deadline := time.After(N), becomes testwait.After(t). The Log's "every positive select guard" claim is false until then.
- BR-2 — addressed — notifytransport/transport_test.go:345-350 now asserts ENOBUFS or os.ErrDeadlineExceeded; test passes on darwin.
- BR-3 — addressed — testwait_test.go:41-60 covers waitUntil past give-up (returns false after one poll), afterAt unbounded (nil channel) and afterAt past time (fires); dropping the bounded check would hang the test.

## Round 3 — 2026-10-12T09:12:02-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — A scan of all 37 time.After/NewTimer sites left in the touched packages finds only absence windows, fake clocks or subject bounds; no fatal timer arm remains.

### Raised

- **BR-4** [Important] `real-time-window-as-event-guard` Polling deadlines (time.Now().Add(N) + Fatal) in touched files still guard events with 1-3 s windows
  2nd finding in this family. Rule: every wall-clock bound whose expiry fails the test while it awaits an event (select arm, hoisted After, polling deadline, ctx timeout) becomes testwait unless the bound is the subject. Touched files: wrapcmd/codex_working_test.go:186 (1 s, verbatim testwait.Until copy), couchcmd/warm_reattach_test.go:208 (also an Until copy), wrapcmd/shortcut_passthrough_test.go:222, wrapcmd/agent_restart_test.go:87, wrapcmd/detach_acceptance_test.go:235, couchtty/console_broadcast_test.go:150, couchtty/console_pointer_test.go:72,221, couchtty/notification_pty_test.go:29, couchtty/terminal_soak_test.go:200, couchcore/couch_test.go:148,1030,1748. Untouched files in touched packages: couchcmd message_deliveries:49, message_lifecycle:170, message_service:257,339,378,652, wait_notice:272,324,465; couchcore provision_lock:117, ptyrunner:17; couchsingleton subprocess:64; wrapcmd launch_watcher:139. About 32 in total. Add polling deadlines to the #485 sweep.
- **BR-5** [Minor] `ceiling-replaced-by-outcome` Stopwatch ceilings remain in touched files (console_broadcast 3 s, provision_subprocess 5 s)
  2nd finding in this family. Rule: a time.Since(start) > N failure is replaced by the outcome that proves the bound. Instances: couchtty/console_broadcast_test.go:357 (the select on f.done with CloseBlock unreleased already proves the bound; delete the ceiling), couchcore/provision_subprocess_test.go:54 (DeadlineExceeded already asserted; assert how the child ended), launcher/session_reap_test.go:175 (untouched file).

## Round 4 — 2026-10-12T09:24:04-07:00 (claude) — passed

### Disposed

- BR-4 — not-addressed — Polling sites all converted, but the rule's 4th form (ctx timeout) was not swept. The rule: every wall-clock bound in a test whose expiry fails the test while it awaits an event, whatever its syntax (select arm, hoisted After, time.Now().Add deadline, context.WithTimeout/ctx.Done arm), takes testwait's budget unless the bound is the subject. Sweep by classifying every WithTimeout/After/Now().Add/NewTimer in touched files, not by regex on one form. Remaining: couchtty/terminal_exit_test.go:178 (ctx.Done Fatal arms :195,:201), couchcore/couch_test.go:753, couchtty/console_relaunch_chord_test.go:65, couchtty/terminal_soak_test.go:217, wrapcmd/detach_acceptance_test.go:211, couchcore/recovery_execute_test.go:607, couchcmd/continuation_acceptance_test.go:250, couchcore/couch_test.go:500. Suggest a testwait.Context(t) helper. 3rd finding in family real-time-window-as-event-guard.
- BR-5 — not-addressed — console_broadcast fixed correctly. provision_subprocess_test.go:49 claims a lost escalation hangs into the binary timeout, but WaitDelay plus the ErrWaitDelay group kill (provision_io.go:72,99) bound it anyway: with the SIGKILL at provision_io.go:83 removed, the test passed in 3.10s. Assert how the child ended, or correct the claim. session_reap_test:175 is deferred to #485.

## Open findings

- **BR-4** [Important] `real-time-window-as-event-guard` Polling deadlines (time.Now().Add(N) + Fatal) in touched files still guard events with 1-3 s windows
- **BR-5** [Minor] `ceiling-replaced-by-outcome` Stopwatch ceilings remain in touched files (console_broadcast 3 s, provision_subprocess 5 s)
