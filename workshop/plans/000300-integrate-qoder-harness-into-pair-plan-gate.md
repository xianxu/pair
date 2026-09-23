---
gate: plan-quality
issue: 300
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-21T08:57:30-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Registry only half-joins in M1, and no parity test makes "every consumer derives" mechanical
          detail: Task 1 adds the enum constant but validAgent (model.go:446) and runcli.go:18 supportedAgents (which feeds pair_inventory.go:120 sidecar parsing) wait for Task 6. Step 5 re-runs suites that never exercise qoder, so the M1 "fail-closed verified by test" claim cannot fail. No AgentInventory parity test exists (only agent_defaults_test.go:90). Move the two pure registry rows into Task 1 and add one test ranging launcher.AgentInventory() over every per-agent table.
          family: agent-dispatch-registration-gap
          round: 1
        - id: PQ-2
          severity: Minor
          title: Hand inventory misses usage.go:60 (context meter) and the create-path session-id mint
          detail: sessioninventory/usage.go:60 ParseTokenUsage is claude-only, so qoder silently yields no usage for pair context (contextcmd.go:72); extend the claude case or list it as a non-goal. launcher/agentargs.go:198 shouldMintClaudeSessionID is the create-path twin of wrap.go:2280; Task 12 only covers the restart path, so honoring --session-id would leave the two paths inconsistent.
          family: agent-dispatch-registration-gap
          round: 1
        - id: PQ-3
          severity: Minor
          title: Task 9 Step 6 "every oracle the tests read" omits the machine-neutral fixture oracle
          detail: 'assertFixtureIsMachineNeutral (harness_tty_fixture_test.go:294-311, expiry :185-190) fails any capture embedding the home path unless ttyFixtureEnvironmentGaps has an entry. A qoder capture taken in the repo cwd will likely embed it (agy/muse captures do). Also an ARCH-SECURE point: state a neutral-cwd capture and account-identity scrub policy. Replace the hand-listed oracles with "satisfy each oracle by its failure message".'
          family: harness-test-oracle-mismatch
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tasks 2/3/5/6/7/13/14 embed full implementation bodies and enumerated test rows; compress to strategy lines
          detail: 'Already stale on arrival: Task 2''s short-flag loop treats p and d as value-taking while its own row says -p is a bool and the value-flag list omits both. Keep one strategy line per risky function (ValidateFreshAgentArgs / extractExplicitResume: table + fuzz over argv forms; scanClaudeFamily / validateClaudeFamilyDelta: real sanitized transcript replay with split-point framing; observationNativeID and recognizer: captured-byte replay at every split).'
          family: plan-prose-restates-diff
          round: 1
        - id: PQ-5
          severity: Minor
          title: Task 7 Step 5 names a live conformance test that does not exist
          detail: The real entry point is TestLiveNativeSessionShapeConformance, gated by PAIR_LIVE_NATIVE_SESSIONS=1 (conformance_live_test.go:9-11), not TestConformanceLive.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-6
          severity: Minor
          title: M5 has no milestone-close of its own; Task 20 goes straight to sdlc close
          detail: Per AGENTS.md section 3 every Mx row commits to a milestone-close. Either state that the final sdlc close is M5's boundary or drop the M5 tag. M1 and M4 are small but distinct review surfaces, so no over-split flag.
          family: milestone-boundary-granularity
          round: 1
        - id: PQ-7
          severity: Minor
          title: Plan states no non-goals
          detail: 'Name what is deliberately not built: qoder --remote/--teleport cloud sessions, subagent transcript resume, context-meter usage (if usage.go is not extended), claude-only progress-OSC lifecycle authority (notification_rewriter.go:129), and a per-tool allowlist if qoder has none.'
          family: no-stated-non-goals
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-09-21T08:59:49-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Task 1 still adds only the enum and supportedAgents; validAgent and runcli.go wait for Task 6, and no parity test over AgentInventory() exists. Carried to the close review.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: usage.go ParseTokenUsage and the create-path shouldMintClaudeSessionID (agentargs.go:197) are still unlisted; Task 12 covers only wrap.go. Carried to the close review.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Task 9 Step 6 still hand-lists oracles and omits assertFixtureIsMachineNeutral and a neutral-cwd/identity-scrub capture policy. Carried to the close review.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Tasks 2/3/5/6/7/13/14 still embed full bodies; Task 2's short-flag loop still treats p and d as value-taking. Carried to the close review.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: Task 7 Step 5 still names TestConformanceLive; the real test is TestLiveNativeSessionShapeConformance (conformance_live_test.go:9), gated by PAIR_LIVE_NATIVE_SESSIONS=1. Carried to the close review.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Task 20 still goes straight to sdlc close; M5 has no milestone-close and no statement that the final close is its boundary. Carried to the close review.
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: No non-goals section; the "Don't touch couch" note covers only one item. Carried to the close review.
          round: 2
      blocked: false
content_hash: 7fd64976255d09906f9e14e17ff910b0c07349b7633868d21b07fe11136207f0
---

# Gate ledger — pair#300 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-21T08:57:30-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `agent-dispatch-registration-gap` Registry only half-joins in M1, and no parity test makes "every consumer derives" mechanical
  Task 1 adds the enum constant but validAgent (model.go:446) and runcli.go:18 supportedAgents (which feeds pair_inventory.go:120 sidecar parsing) wait for Task 6. Step 5 re-runs suites that never exercise qoder, so the M1 "fail-closed verified by test" claim cannot fail. No AgentInventory parity test exists (only agent_defaults_test.go:90). Move the two pure registry rows into Task 1 and add one test ranging launcher.AgentInventory() over every per-agent table.
- **PQ-2** [Minor] `agent-dispatch-registration-gap` Hand inventory misses usage.go:60 (context meter) and the create-path session-id mint
  sessioninventory/usage.go:60 ParseTokenUsage is claude-only, so qoder silently yields no usage for pair context (contextcmd.go:72); extend the claude case or list it as a non-goal. launcher/agentargs.go:198 shouldMintClaudeSessionID is the create-path twin of wrap.go:2280; Task 12 only covers the restart path, so honoring --session-id would leave the two paths inconsistent.
- **PQ-3** [Minor] `harness-test-oracle-mismatch` Task 9 Step 6 "every oracle the tests read" omits the machine-neutral fixture oracle
  assertFixtureIsMachineNeutral (harness_tty_fixture_test.go:294-311, expiry :185-190) fails any capture embedding the home path unless ttyFixtureEnvironmentGaps has an entry. A qoder capture taken in the repo cwd will likely embed it (agy/muse captures do). Also an ARCH-SECURE point: state a neutral-cwd capture and account-identity scrub policy. Replace the hand-listed oracles with "satisfy each oracle by its failure message".
- **PQ-4** [Minor] `plan-prose-restates-diff` Tasks 2/3/5/6/7/13/14 embed full implementation bodies and enumerated test rows; compress to strategy lines
  Already stale on arrival: Task 2's short-flag loop treats p and d as value-taking while its own row says -p is a bool and the value-flag list omits both. Keep one strategy line per risky function (ValidateFreshAgentArgs / extractExplicitResume: table + fuzz over argv forms; scanClaudeFamily / validateClaudeFamilyDelta: real sanitized transcript replay with split-point framing; observationNativeID and recognizer: captured-byte replay at every split).
- **PQ-5** [Minor] `unbacked-existing-behavior-claim` Task 7 Step 5 names a live conformance test that does not exist
  The real entry point is TestLiveNativeSessionShapeConformance, gated by PAIR_LIVE_NATIVE_SESSIONS=1 (conformance_live_test.go:9-11), not TestConformanceLive.
- **PQ-6** [Minor] `milestone-boundary-granularity` M5 has no milestone-close of its own; Task 20 goes straight to sdlc close
  Per AGENTS.md section 3 every Mx row commits to a milestone-close. Either state that the final sdlc close is M5's boundary or drop the M5 tag. M1 and M4 are small but distinct review surfaces, so no over-split flag.
- **PQ-7** [Minor] `no-stated-non-goals` Plan states no non-goals
  Name what is deliberately not built: qoder --remote/--teleport cloud sessions, subagent transcript resume, context-meter usage (if usage.go is not extended), claude-only progress-OSC lifecycle authority (notification_rewriter.go:129), and a per-tool allowlist if qoder has none.

## Round 2 — 2026-09-21T08:59:49-07:00 (claude) — passed

### Disposed

- PQ-1 — not-addressed — Task 1 still adds only the enum and supportedAgents; validAgent and runcli.go wait for Task 6, and no parity test over AgentInventory() exists. Carried to the close review.
- PQ-2 — not-addressed — usage.go ParseTokenUsage and the create-path shouldMintClaudeSessionID (agentargs.go:197) are still unlisted; Task 12 covers only wrap.go. Carried to the close review.
- PQ-3 — not-addressed — Task 9 Step 6 still hand-lists oracles and omits assertFixtureIsMachineNeutral and a neutral-cwd/identity-scrub capture policy. Carried to the close review.
- PQ-4 — not-addressed — Tasks 2/3/5/6/7/13/14 still embed full bodies; Task 2's short-flag loop still treats p and d as value-taking. Carried to the close review.
- PQ-5 — not-addressed — Task 7 Step 5 still names TestConformanceLive; the real test is TestLiveNativeSessionShapeConformance (conformance_live_test.go:9), gated by PAIR_LIVE_NATIVE_SESSIONS=1. Carried to the close review.
- PQ-6 — not-addressed — Task 20 still goes straight to sdlc close; M5 has no milestone-close and no statement that the final close is its boundary. Carried to the close review.
- PQ-7 — not-addressed — No non-goals section; the "Don't touch couch" note covers only one item. Carried to the close review.

## Open findings

- **PQ-1** [Minor] `agent-dispatch-registration-gap` Registry only half-joins in M1, and no parity test makes "every consumer derives" mechanical
- **PQ-2** [Minor] `agent-dispatch-registration-gap` Hand inventory misses usage.go:60 (context meter) and the create-path session-id mint
- **PQ-3** [Minor] `harness-test-oracle-mismatch` Task 9 Step 6 "every oracle the tests read" omits the machine-neutral fixture oracle
- **PQ-4** [Minor] `plan-prose-restates-diff` Tasks 2/3/5/6/7/13/14 embed full implementation bodies and enumerated test rows; compress to strategy lines
- **PQ-5** [Minor] `unbacked-existing-behavior-claim` Task 7 Step 5 names a live conformance test that does not exist
- **PQ-6** [Minor] `milestone-boundary-granularity` M5 has no milestone-close of its own; Task 20 goes straight to sdlc close
- **PQ-7** [Minor] `no-stated-non-goals` Plan states no non-goals
