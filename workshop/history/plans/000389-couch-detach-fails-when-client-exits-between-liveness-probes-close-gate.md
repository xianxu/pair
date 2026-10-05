---
gate: boundary-review
issue: 389
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-03T12:05:57-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Couch.Liveness duplicates observeExactProcess and keeps the two-probe race the fix closed
          detail: 'couch.go:1040 Couch.Liveness repeats the pre-fix Exists-then-Identity logic line for line and still returns Unknown when an identity read fails; it is called from switchagent.go:94, couch.go:1177 and couch.go:1199, so the Spec''s claim that liveness callers get the fix is false. Fix: keep the zero guard, then return observeExactProcess(c.Proc, ProcessIdentity{PID: a.PID, Identity: a.Identity}), and add a ReapedOnIdentity test for Liveness. Same family, lower impact (these report an error instead of Unknown): supervisorlease.go:108-122 VerifiedOwner, supervisor_observe.go:77-85, switchcontext.go:232-240 orientation target (a target reaped between the probes gives a raw no-identity error instead of the exited message).'
          family: single-source-process-observation
          round: 1
        - id: BR-2
          severity: Minor
          title: darwin zombie between exit and reap may still give Unknown if sysctl lacks its identity
          detail: kill(pid,0) succeeds on a zombie, so if kern.proc.pid gives a zombie no matching identity, the re-check sees Live and stays Unknown. Unverified; log it if the symptom comes back.
          family: zombie-identity-window
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-03T12:20:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Liveness delegates to observeExactProcess (couch.go:1049) with TestLivenessReapedBetweenProbesIsDead; VerifiedOwner, ObserveSupervisor, ReadOrientationStatus and message_service also delegate now; grep finds no other Exists-then-Identity copy outside tests.
          round: 2
        - id: BR-2
          disposition: withdrawn
          note: Speculative and conditional (log it only if the symptom comes back); darwin sysctl kern.proc.pid normally still returns a zombie's start time, and nothing in this window can be changed to verify it.
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#389 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-03T12:05:57-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `single-source-process-observation` Couch.Liveness duplicates observeExactProcess and keeps the two-probe race the fix closed
  couch.go:1040 Couch.Liveness repeats the pre-fix Exists-then-Identity logic line for line and still returns Unknown when an identity read fails; it is called from switchagent.go:94, couch.go:1177 and couch.go:1199, so the Spec's claim that liveness callers get the fix is false. Fix: keep the zero guard, then return observeExactProcess(c.Proc, ProcessIdentity{PID: a.PID, Identity: a.Identity}), and add a ReapedOnIdentity test for Liveness. Same family, lower impact (these report an error instead of Unknown): supervisorlease.go:108-122 VerifiedOwner, supervisor_observe.go:77-85, switchcontext.go:232-240 orientation target (a target reaped between the probes gives a raw no-identity error instead of the exited message).
- **BR-2** [Minor] `zombie-identity-window` darwin zombie between exit and reap may still give Unknown if sysctl lacks its identity
  kill(pid,0) succeeds on a zombie, so if kern.proc.pid gives a zombie no matching identity, the re-check sees Live and stays Unknown. Unverified; log it if the symptom comes back.

## Round 2 — 2026-10-03T12:20:56-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Liveness delegates to observeExactProcess (couch.go:1049) with TestLivenessReapedBetweenProbesIsDead; VerifiedOwner, ObserveSupervisor, ReadOrientationStatus and message_service also delegate now; grep finds no other Exists-then-Identity copy outside tests.
- BR-2 — withdrawn — Speculative and conditional (log it only if the symptom comes back); darwin sysctl kern.proc.pid normally still returns a zombie's start time, and nothing in this window can be changed to verify it.

## Open findings

(none — every finding has been disposed)
