---
gate: plan-quality
issue: 399
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-06T19:39:08-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: ServerProcess would be a third struct for the server-generation fact, alongside sessionServerIdentity and SessionServerIdentity
          detail: session_quiescence.go:17 and session_owner.go:23 already duplicate {PID, Identity, Session}, and session_owner_os.go:103-110 copies one into the other. The plan adds ServerProcess AND a Socket field on SessionServerIdentity. Collapse them into one type that Probe, KillServer, SessionObservation.Orphan and Reaper all share (ARCH-DRY).
          family: single-type-per-fact
          round: 1
        - id: PQ-2
          severity: Important
          title: ClassifyServers' exists func(string) bool turns any Lstat error into socket-gone, which reads as orphaned
          detail: ps -ax lists other users' zellij servers, so EACCES or an I/O error would mark a live server orphaned and offer reap. Only ENOENT means orphaned; any other error must make the session unresolved (fail closed), with a test row for it (ARCH-ORDER).
          family: failed-probe-is-not-absence
          round: 1
        - id: PQ-3
          severity: Important
          title: recover's per-plan confirmation changes the fixed per-operation Confirmation contract (ops.go:58-60) without naming the change
          detail: OperationConfirms (operationdispatch.go:81) and SlotOperationCommand rely on a fixed confirmation value per operation. The plan must name the new confirmation value, how the menu gets the confirmation text from the core, what couch --recover requires (--confirm when the plan includes reap or reboot), and how the operation audit tests treat it.
          family: unstated-seam-change
          round: 1
        - id: PQ-4
          severity: Minor
          title: 'Wrong file and fake names: fakeOwnerIO and session_owner_os_test.go, couchtty/menu.go; couchcmd/slot_operations.go is not listed'
          detail: The existing fake is ownerWorld in launcher/session_owner_test.go:49; the socket side of resume/reboot is couchcmd/slot_operations.go; the menu code lives in menu_actions.go and console_menu.go.
          family: unbacked-existing-code-claim
          round: 1
        - id: PQ-5
          severity: Minor
          title: Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
          family: test-strategy-not-enumeration
          round: 1
        - id: PQ-6
          severity: Minor
          title: No explicit non-goals (automatic reaping, a stale socket that refuses connections, lifecycle-queue changes)
          family: stated-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-06T19:40:28-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: SessionServerIdentity gains Socket and the duplicate is deleted. Stray ServerProcess references remain (see the new Minor).
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 'Tri-state SocketState: only ENOENT is gone. Unknown and a snapshot error both mean unresolved.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: ConfirmByPlan, PrepareRecover/RecoverPreview, the menu frame, the CLI --confirm rule and the audit handling are all named.
          round: 2
        - id: PQ-4
          disposition: addressed
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: Task 1 has strategy lines, but Tasks 2-4 keep full Go test bodies and Tasks 7, 8 and 9b still list cases in prose. Minor.
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
      findings:
        - id: PQ-7
          severity: Minor
          title: Plan lines 69, 138, 253 and 332 still name the abandoned ServerProcess type
          detail: 'This is the 2nd finding in family single-type-per-fact. Rule: the server-generation fact has exactly one type, SessionServerIdentity. Every signature in the plan (SessionObservation.Orphan, PlanReap''s root, Reaper.Reap, ServerState.Server) uses that type. Sweep the plan for ServerProcess, not just these four sites.'
          family: single-type-per-fact
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-06T19:41:33-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: not-addressed
          note: 'Tasks 1-2 now have strategy lines; Tasks 3, 7, 8, 9b still enumerate cases in prose and inline test bodies remain. Carry to close; compress Task 7 to one line (generated process forest with recycled pids and TERM-ignoring processes; guard: no signal to a pid whose identity changed).'
          round: 3
        - id: PQ-7
          disposition: addressed
          note: grep finds no ServerProcess type in the plan; every signature uses SessionServerIdentity, and the lowercase sessionServerIdentity is slated for deletion.
          round: 3
      blocked: false
content_hash: 85d2da65556a7917ddf3bf72e32f40ec440b49b9d0e207a5a23b6055e5a3e20c
---

# Gate ledger — pair#399 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T19:39:08-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `single-type-per-fact` ServerProcess would be a third struct for the server-generation fact, alongside sessionServerIdentity and SessionServerIdentity
  session_quiescence.go:17 and session_owner.go:23 already duplicate {PID, Identity, Session}, and session_owner_os.go:103-110 copies one into the other. The plan adds ServerProcess AND a Socket field on SessionServerIdentity. Collapse them into one type that Probe, KillServer, SessionObservation.Orphan and Reaper all share (ARCH-DRY).
- **PQ-2** [Important] `failed-probe-is-not-absence` ClassifyServers' exists func(string) bool turns any Lstat error into socket-gone, which reads as orphaned
  ps -ax lists other users' zellij servers, so EACCES or an I/O error would mark a live server orphaned and offer reap. Only ENOENT means orphaned; any other error must make the session unresolved (fail closed), with a test row for it (ARCH-ORDER).
- **PQ-3** [Important] `unstated-seam-change` recover's per-plan confirmation changes the fixed per-operation Confirmation contract (ops.go:58-60) without naming the change
  OperationConfirms (operationdispatch.go:81) and SlotOperationCommand rely on a fixed confirmation value per operation. The plan must name the new confirmation value, how the menu gets the confirmation text from the core, what couch --recover requires (--confirm when the plan includes reap or reboot), and how the operation audit tests treat it.
- **PQ-4** [Minor] `unbacked-existing-code-claim` Wrong file and fake names: fakeOwnerIO and session_owner_os_test.go, couchtty/menu.go; couchcmd/slot_operations.go is not listed
  The existing fake is ownerWorld in launcher/session_owner_test.go:49; the socket side of resume/reboot is couchcmd/slot_operations.go; the menu code lives in menu_actions.go and console_menu.go.
- **PQ-5** [Minor] `test-strategy-not-enumeration` Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
- **PQ-6** [Minor] `stated-non-goals` No explicit non-goals (automatic reaping, a stale socket that refuses connections, lifecycle-queue changes)

## Round 2 — 2026-10-06T19:40:28-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — SessionServerIdentity gains Socket and the duplicate is deleted. Stray ServerProcess references remain (see the new Minor).
- PQ-2 — addressed — Tri-state SocketState: only ENOENT is gone. Unknown and a snapshot error both mean unresolved.
- PQ-3 — addressed — ConfirmByPlan, PrepareRecover/RecoverPreview, the menu frame, the CLI --confirm rule and the audit handling are all named.
- PQ-4 — addressed
- PQ-5 — not-addressed — Task 1 has strategy lines, but Tasks 2-4 keep full Go test bodies and Tasks 7, 8 and 9b still list cases in prose. Minor.
- PQ-6 — addressed

### Raised

- **PQ-7** [Minor] `single-type-per-fact` Plan lines 69, 138, 253 and 332 still name the abandoned ServerProcess type
  This is the 2nd finding in family single-type-per-fact. Rule: the server-generation fact has exactly one type, SessionServerIdentity. Every signature in the plan (SessionObservation.Orphan, PlanReap's root, Reaper.Reap, ServerState.Server) uses that type. Sweep the plan for ServerProcess, not just these four sites.

## Round 3 — 2026-10-06T19:41:33-07:00 (claude) — passed

### Disposed

- PQ-5 — not-addressed — Tasks 1-2 now have strategy lines; Tasks 3, 7, 8, 9b still enumerate cases in prose and inline test bodies remain. Carry to close; compress Task 7 to one line (generated process forest with recycled pids and TERM-ignoring processes; guard: no signal to a pid whose identity changed).
- PQ-7 — addressed — grep finds no ServerProcess type in the plan; every signature uses SessionServerIdentity, and the lowercase sessionServerIdentity is slated for deletion.

## Open findings

- **PQ-5** [Minor] `test-strategy-not-enumeration` Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
