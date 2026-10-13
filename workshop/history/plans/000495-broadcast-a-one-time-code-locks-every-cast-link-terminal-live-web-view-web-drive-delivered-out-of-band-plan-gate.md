---
gate: plan-quality
issue: 495
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T13:16:55-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Host.Lock as a separate call after Mount leaves an unlocked window and no removal on Unmount
          detail: Host.Mount makes a token routable and probes token+"/" right away (host.go:85-97; cast.go:235, session.go:111). A lock registered by a later Host.Lock call misses that window, and the plan never says what drops a registry entry when the mount unmounts. Attach locks atomically with the mount (a Mountable method like LockFor(segment), or a Mount argument) so the gate can never see an owned but unlocked token and the entry ends with its Mount (ARCH-SECURE, ARCH-ORDER, ARCH-FUNERAL).
          family: lock-registration-atomic-with-mount
          round: 1
        - id: PQ-2
          severity: Minor
          title: Spec items 4 and 7, and the plan's Couch section, still describe the superseded per-client budget and inline code display
          detail: The D2/D5 revisions changed both, so fold the answers into Spec and Done-when (operator-only layer, tooltip, Alt+click Codes menu, per-code budget of about 8). Otherwise the close review judges against stale text.
          family: spec-moves-with-decisions
          round: 1
        - id: PQ-3
          severity: Minor
          title: A per-code budget lets anyone holding the link force repeated rotations and lock out new joiners
          detail: Say what bounds this (the rate limit per lock, a cap on notices) and that it is an accepted tradeoff (ARCH-CONSTRAINTS).
          family: declare-overload-behavior
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tests are listed as cases rather than one strategy line per risky function
          detail: castlock Unlock/Rotate/Valid need a seeded random interleaving sequence test against independently stated invariants (a revoked session is never valid; the view code never opens the drive token). The unlock body and code parser should be fuzzed with malformed input.
          family: test-strategy-not-enumeration
          round: 1
        - id: PQ-5
          severity: Minor
          title: A SameSite=Strict cookie is not sent when the link is opened from a chat app
          detail: A returning viewer sees the lock page again on that cross-site navigation even with a valid session. Accept and document this, or use Lax, which is safe here because every state-changing route is a POST.
          family: cookie-samesite-navigation
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T13:17:52-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Mountable.LockFor(segment) is asked after Owns; locks are created before Host.Mount (the drive token is set at construction, server_drive.go:59) and end with the mount.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Spec 4/7 and Done-when are rewritten; the stale plan-body text is superseded by Revisions, as the append-only convention requires.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Per-lock rate limit (about 1/s, burst 5), about one forced rotation per 8 s at most, notices merged every 30 s; accepted tradeoff.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Seeded random sequence test against stated invariants, fuzzing of the decoder and parser, gate table over every route with none/invalid/revoked sessions.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Lax chosen, justified by POST-only state changes with a JSON content type.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T13:18:50-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: LockFor on Mountable, lock created before Mount, ends with the mount.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Issue Spec items 4/7 and Done-when rewritten.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Rate-limit bound, notice coalescing, accepted trade-off stated.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Seeded op-sequence invariants, fuzz targets, route x session-state table.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Lax chosen, with the POST+JSON CSRF argument.
          round: 3
      findings:
        - id: PQ-6
          severity: Minor
          title: Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
          detail: '2nd in family. The rule: a revision that replaces a decision rewrites the body paragraph stating it, not just a Revisions entry. Sites: plan Couch Display/Click rotation/Documented trade-off, Session transport Strict, Plan M3 bullets, Shape lock page (name on every link per D6).'
          family: spec-moves-with-decisions
          round: 3
        - id: PQ-7
          severity: Minor
          title: D1 (session cookie) still awaits the operator; the M2 credentials change depends on it
          detail: M1 can proceed behind sessionFrom(r). Confirm D1 before starting M2.
          family: open-decision-gates-milestone
          round: 3
      blocked: false
content_hash: aac5207220ae1213a220a59c82d95d28fc3db1cecb96baf3a7f2e73f71fd6b4d
---

# Gate ledger — pair#495 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T13:16:55-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `lock-registration-atomic-with-mount` Host.Lock as a separate call after Mount leaves an unlocked window and no removal on Unmount
  Host.Mount makes a token routable and probes token+"/" right away (host.go:85-97; cast.go:235, session.go:111). A lock registered by a later Host.Lock call misses that window, and the plan never says what drops a registry entry when the mount unmounts. Attach locks atomically with the mount (a Mountable method like LockFor(segment), or a Mount argument) so the gate can never see an owned but unlocked token and the entry ends with its Mount (ARCH-SECURE, ARCH-ORDER, ARCH-FUNERAL).
- **PQ-2** [Minor] `spec-moves-with-decisions` Spec items 4 and 7, and the plan's Couch section, still describe the superseded per-client budget and inline code display
  The D2/D5 revisions changed both, so fold the answers into Spec and Done-when (operator-only layer, tooltip, Alt+click Codes menu, per-code budget of about 8). Otherwise the close review judges against stale text.
- **PQ-3** [Minor] `declare-overload-behavior` A per-code budget lets anyone holding the link force repeated rotations and lock out new joiners
  Say what bounds this (the rate limit per lock, a cap on notices) and that it is an accepted tradeoff (ARCH-CONSTRAINTS).
- **PQ-4** [Minor] `test-strategy-not-enumeration` Tests are listed as cases rather than one strategy line per risky function
  castlock Unlock/Rotate/Valid need a seeded random interleaving sequence test against independently stated invariants (a revoked session is never valid; the view code never opens the drive token). The unlock body and code parser should be fuzzed with malformed input.
- **PQ-5** [Minor] `cookie-samesite-navigation` A SameSite=Strict cookie is not sent when the link is opened from a chat app
  A returning viewer sees the lock page again on that cross-site navigation even with a valid session. Accept and document this, or use Lax, which is safe here because every state-changing route is a POST.

## Round 2 — 2026-10-12T13:17:52-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Mountable.LockFor(segment) is asked after Owns; locks are created before Host.Mount (the drive token is set at construction, server_drive.go:59) and end with the mount.
- PQ-2 — addressed — Spec 4/7 and Done-when are rewritten; the stale plan-body text is superseded by Revisions, as the append-only convention requires.
- PQ-3 — addressed — Per-lock rate limit (about 1/s, burst 5), about one forced rotation per 8 s at most, notices merged every 30 s; accepted tradeoff.
- PQ-4 — addressed — Seeded random sequence test against stated invariants, fuzzing of the decoder and parser, gate table over every route with none/invalid/revoked sessions.
- PQ-5 — addressed — Lax chosen, justified by POST-only state changes with a JSON content type.

## Round 3 — 2026-10-12T13:18:50-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — LockFor on Mountable, lock created before Mount, ends with the mount.
- PQ-2 — addressed — Issue Spec items 4/7 and Done-when rewritten.
- PQ-3 — addressed — Rate-limit bound, notice coalescing, accepted trade-off stated.
- PQ-4 — addressed — Seeded op-sequence invariants, fuzz targets, route x session-state table.
- PQ-5 — addressed — Lax chosen, with the POST+JSON CSRF argument.

### Raised

- **PQ-6** [Minor] `spec-moves-with-decisions` Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
  2nd in family. The rule: a revision that replaces a decision rewrites the body paragraph stating it, not just a Revisions entry. Sites: plan Couch Display/Click rotation/Documented trade-off, Session transport Strict, Plan M3 bullets, Shape lock page (name on every link per D6).
- **PQ-7** [Minor] `open-decision-gates-milestone` D1 (session cookie) still awaits the operator; the M2 credentials change depends on it
  M1 can proceed behind sessionFrom(r). Confirm D1 before starting M2.

## Open findings

- **PQ-6** [Minor] `spec-moves-with-decisions` Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
- **PQ-7** [Minor] `open-decision-gates-milestone` D1 (session cookie) still awaits the operator; the M2 credentials change depends on it
