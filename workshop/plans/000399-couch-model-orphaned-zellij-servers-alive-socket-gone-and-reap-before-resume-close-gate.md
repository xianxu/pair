---
gate: boundary-review
issue: 399
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-06T22:27:27-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
          detail: (carried from plan-quality PQ-5, deferred to the boundary review)
          family: test-strategy-not-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-06T22:27:27-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Startup orphan refusal says "kill PID; its agent goes with it", which the issue shows is false
          detail: couch.go:500 tells the operator to kill only the server. On 2026-10-06 wrap ignored SIGTERM and title helpers were left with PPID 1. Tell them to stop the whole process tree (or point to reap) instead.
          family: operator-advice-contradicts-evidence
          round: 2
        - id: BR-3
          severity: Minor
          title: sessionOwnerWord and AgentRunning do not handle the new orphaned values
          detail: launch_existing.go:466 prints "unknown" for SessionOwnerOrphaned; slotobserve.go:18 returns not-known for AgentOrphaned, though an orphan is known to be running.
          family: vocabulary-consumer-missing-member
          round: 2
        - id: BR-4
          severity: Minor
          title: ScopeHoldsOrphanedThread also requires Orphan != nil, so a nil-server orphan row reads as debris
          family: derived-field-guard-redundant
          round: 2
        - id: BR-5
          severity: Minor
          title: Ranking of orphaned vs unusable-unknown agent evidence is written in two places (slotreport rank map, recoverplan loop)
          family: single-ranking-source
          round: 2
        - id: BR-6
          severity: Minor
          title: An orphan plus a new live server for the same name reads Present, hiding the orphan
          family: orphan-shadowed-by-live-same-name
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-06T22:40:36-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: FuzzParseServerProcesses with a one-line Strategy comment, session_servers_test.go:28-46.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: couch.go:1281 pkill -P reaches only direct children, and the KILL follow-up runs after kill <server>, so SIGTERM-ignoring children have already moved to PID 1 and it matches nothing. Same failure as 2026-10-06; the test checks strings only.
          round: 3
        - id: BR-3
          disposition: addressed
          note: sessionOwnerWord and AgentRunning handle the orphan (TestVocabularyConsumersKnowTheOrphan); other SessionOwner switches fail closed by default.
          round: 3
        - id: BR-4
          disposition: addressed
          note: startup.go:125 keys on the reason only; TestAnOrphanRowWithoutItsServerStillBlocksStartup.
          round: 3
        - id: BR-5
          disposition: addressed
          note: One agentRank at recoverplan.go:1481 used by slotreport and slotEvidenceOf.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Contested outranks list-sessions live (sessionevidence.go:121, TestAContestedServerNameIsNeverPresent); Probe treats len!=1 as ambiguous.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: ServerState is three booleans (Orphaned/Unresolved/Contested) with unwritten legal combinations
          detail: Contested implies Unresolved and Orphaned+Unresolved can be represented; a tagged enum (Reachable, Orphaned, UnknownSocket, Contested) would rule the bad combinations out (ARCH-ORDER/ARCH-SECURE).
          family: boolean-constellation-not-enum
          round: 3
        - id: BR-8
          severity: Minor
          title: A server still starting (in ps, socket not yet bound) reads as orphaned on a single snapshot
          detail: Harmless in M1, where it only causes a refusal; M2 reap must require two observations or a minimum process age and re-check the identity before signalling, or it can kill a session that is starting (ARCH-ORDER).
          family: point-observation-as-settled-state
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-06T23:00:42-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: couch.go orphanStartRefusal now lists the tree, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident fails if the order changes or the old advice returns.
          round: 4
        - id: BR-7
          disposition: addressed
          note: ServerState carries one ServerVerdict enum (zero value Unresolved); every consumer uses hasServer or a verdict comparison; the table test pins all four verdicts.
          round: 4
        - id: BR-8
          disposition: addressed
          note: Harmless in M1 (it only refuses); the plan's 2026-10-07 Revisions requires two snapshots or a minimum process age, an identity re-read before each signal, and a just-started-server fixture in M2.
          round: 4
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#399 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T22:27:27-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-strategy-not-enumeration` Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-10-06T22:27:27-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `operator-advice-contradicts-evidence` Startup orphan refusal says "kill PID; its agent goes with it", which the issue shows is false
  couch.go:500 tells the operator to kill only the server. On 2026-10-06 wrap ignored SIGTERM and title helpers were left with PPID 1. Tell them to stop the whole process tree (or point to reap) instead.
- **BR-3** [Minor] `vocabulary-consumer-missing-member` sessionOwnerWord and AgentRunning do not handle the new orphaned values
  launch_existing.go:466 prints "unknown" for SessionOwnerOrphaned; slotobserve.go:18 returns not-known for AgentOrphaned, though an orphan is known to be running.
- **BR-4** [Minor] `derived-field-guard-redundant` ScopeHoldsOrphanedThread also requires Orphan != nil, so a nil-server orphan row reads as debris
- **BR-5** [Minor] `single-ranking-source` Ranking of orphaned vs unusable-unknown agent evidence is written in two places (slotreport rank map, recoverplan loop)
- **BR-6** [Minor] `orphan-shadowed-by-live-same-name` An orphan plus a new live server for the same name reads Present, hiding the orphan

## Round 3 — 2026-10-06T22:40:36-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — FuzzParseServerProcesses with a one-line Strategy comment, session_servers_test.go:28-46.
- BR-2 — not-addressed — couch.go:1281 pkill -P reaches only direct children, and the KILL follow-up runs after kill <server>, so SIGTERM-ignoring children have already moved to PID 1 and it matches nothing. Same failure as 2026-10-06; the test checks strings only.
- BR-3 — addressed — sessionOwnerWord and AgentRunning handle the orphan (TestVocabularyConsumersKnowTheOrphan); other SessionOwner switches fail closed by default.
- BR-4 — addressed — startup.go:125 keys on the reason only; TestAnOrphanRowWithoutItsServerStillBlocksStartup.
- BR-5 — addressed — One agentRank at recoverplan.go:1481 used by slotreport and slotEvidenceOf.
- BR-6 — addressed — Contested outranks list-sessions live (sessionevidence.go:121, TestAContestedServerNameIsNeverPresent); Probe treats len!=1 as ambiguous.

### Raised

- **BR-7** [Minor] `boolean-constellation-not-enum` ServerState is three booleans (Orphaned/Unresolved/Contested) with unwritten legal combinations
  Contested implies Unresolved and Orphaned+Unresolved can be represented; a tagged enum (Reachable, Orphaned, UnknownSocket, Contested) would rule the bad combinations out (ARCH-ORDER/ARCH-SECURE).
- **BR-8** [Minor] `point-observation-as-settled-state` A server still starting (in ps, socket not yet bound) reads as orphaned on a single snapshot
  Harmless in M1, where it only causes a refusal; M2 reap must require two observations or a minimum process age and re-check the identity before signalling, or it can kill a session that is starting (ARCH-ORDER).

## Round 4 — 2026-10-06T23:00:42-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — couch.go orphanStartRefusal now lists the tree, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident fails if the order changes or the old advice returns.
- BR-7 — addressed — ServerState carries one ServerVerdict enum (zero value Unresolved); every consumer uses hasServer or a verdict comparison; the table test pins all four verdicts.
- BR-8 — addressed — Harmless in M1 (it only refuses); the plan's 2026-10-07 Revisions requires two snapshots or a minimum process age, an identity re-read before each signal, and a just-started-server fixture in M2.

## Open findings

(none — every finding has been disposed)
