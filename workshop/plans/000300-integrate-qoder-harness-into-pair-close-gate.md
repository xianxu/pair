---
gate: boundary-review
issue: 300
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-21T09:26:04-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Registry only half-joins in M1, and no parity test makes "every consumer derives" mechanical
          detail: |-
            Task 1 adds the enum constant but validAgent (model.go:446) and runcli.go:18 supportedAgents (which feeds pair_inventory.go:120 sidecar parsing) wait for Task 6. Step 5 re-runs suites that never exercise qoder, so the M1 "fail-closed verified by test" claim cannot fail. No AgentInventory parity test exists (only agent_defaults_test.go:90). Move the two pure registry rows into Task 1 and add one test ranging launcher.AgentInventory() over every per-agent table.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: agent-dispatch-registration-gap
          round: 1
        - id: BR-2
          severity: Minor
          title: Hand inventory misses usage.go:60 (context meter) and the create-path session-id mint
          detail: |-
            sessioninventory/usage.go:60 ParseTokenUsage is claude-only, so qoder silently yields no usage for pair context (contextcmd.go:72); extend the claude case or list it as a non-goal. launcher/agentargs.go:198 shouldMintClaudeSessionID is the create-path twin of wrap.go:2280; Task 12 only covers the restart path, so honoring --session-id would leave the two paths inconsistent.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: agent-dispatch-registration-gap
          round: 1
        - id: BR-3
          severity: Minor
          title: Task 9 Step 6 "every oracle the tests read" omits the machine-neutral fixture oracle
          detail: |-
            assertFixtureIsMachineNeutral (harness_tty_fixture_test.go:294-311, expiry :185-190) fails any capture embedding the home path unless ttyFixtureEnvironmentGaps has an entry. A qoder capture taken in the repo cwd will likely embed it (agy/muse captures do). Also an ARCH-SECURE point: state a neutral-cwd capture and account-identity scrub policy. Replace the hand-listed oracles with "satisfy each oracle by its failure message".
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: harness-test-oracle-mismatch
          round: 1
        - id: BR-4
          severity: Minor
          title: Tasks 2/3/5/6/7/13/14 embed full implementation bodies and enumerated test rows; compress to strategy lines
          detail: |-
            Already stale on arrival: Task 2's short-flag loop treats p and d as value-taking while its own row says -p is a bool and the value-flag list omits both. Keep one strategy line per risky function (ValidateFreshAgentArgs / extractExplicitResume: table + fuzz over argv forms; scanClaudeFamily / validateClaudeFamilyDelta: real sanitized transcript replay with split-point framing; observationNativeID and recognizer: captured-byte replay at every split).
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: plan-prose-restates-diff
          round: 1
        - id: BR-5
          severity: Minor
          title: Task 7 Step 5 names a live conformance test that does not exist
          detail: |-
            The real entry point is TestLiveNativeSessionShapeConformance, gated by PAIR_LIVE_NATIVE_SESSIONS=1 (conformance_live_test.go:9-11), not TestConformanceLive.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
        - id: BR-6
          severity: Minor
          title: M5 has no milestone-close of its own; Task 20 goes straight to sdlc close
          detail: |-
            Per AGENTS.md section 3 every Mx row commits to a milestone-close. Either state that the final sdlc close is M5's boundary or drop the M5 tag. M1 and M4 are small but distinct review surfaces, so no over-split flag.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: milestone-boundary-granularity
          round: 1
        - id: BR-7
          severity: Minor
          title: Plan states no non-goals
          detail: |-
            Name what is deliberately not built: qoder --remote/--teleport cloud sessions, subagent transcript resume, context-meter usage (if usage.go is not extended), claude-only progress-OSC lifecycle authority (notification_rewriter.go:129), and a per-tool allowlist if qoder has none.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: no-stated-non-goals
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-21T09:26:04-07:00"
      agent: claude
      findings:
        - id: BR-8
          severity: Critical
          title: freshVariadicOption dropped its claude-only gate, so codex `--add-dir X resume ID` now passes fresh-arg validation
          detail: 'fresh_args.go:92 calls freshVariadicOption(agent, flag) for every agent and the function falls through to claude''s variadic flag list, so codex --add-dir swallows the following positionals. Measured: ValidateFreshAgentArgs("codex", ["--add-dir","/x","resume","abc"]) errors at 1ec329f3 and returns nil at d1f36450, so a fresh launch can silently resume an existing conversation. No test covers it. Make the helper strictly per-agent (default false) and add codex and agy regression rows.'
          family: refactor-changes-sibling-agent-behavior
          round: 2
        - id: BR-9
          severity: Important
          title: qoder `-r <id>` is accepted by extractExplicitResume but never stripped from persisted or fresh args
          detail: 'persistedConfigArgs strips only --resume (both forms) and sessionwatch.StripResumeArgs only the space form; -r survives. Measured: relaunch composes `-r abc --resume sid2`, and FreshAgentArgs then ValidateFreshAgentArgs fails with `qoder argument "-r" selects an existing conversation`, breaking Alt+n and continuation. The claimed test only asserts --resume, which passes without qoder code. Enumerate the resume forms against every strip and validate site (or drop -r as claude does) and add a failing-first round-trip test.'
          family: resume-form-recognized-but-not-stripped
          round: 2
        - id: BR-10
          severity: Important
          title: M1 fail-closed intermediate is asserted but untested; no registry parity test, and `pair session-inventory` now emits a permanent qoder schema_near_miss
          detail: The diff adds no qoder row to wrapcmd, sessioninventory, couchcore or couchtty tests, so the "verified green" claim cannot fail. With qoder in runcli's supportedAgents and no scanner, `pair session-inventory` with no --agent emits a schema_near_miss warning for qoder on every run (measured, exit 0). Add a parity test over launcher.AgentInventory() with a named known-gap entry for qoder that M2 must delete, and pin the intermediate diagnostic.
          family: agent-dispatch-registration-gap
          round: 2
        - id: BR-11
          severity: Minor
          title: Agent set restated by hand at five sites (usage string x3, validAgent, runcli supportedAgents) plus golden; atlas says "join both"
          detail: Derive the usage line and validAgent from one list so the next harness touches one row, and correct the atlas checklist item 1, which under-counts the sessioninventory sites.
          family: hand-restated-registry
          round: 2
        - id: BR-12
          severity: Minor
          title: qoder extractExplicitResume branch and short-flag cluster loop are near-copies of claude's; --resume followed by a flag returns the flag as the id
          detail: Parameterize the per-agent flag names and cluster value letters. Qoder's --resume [id] is optional-valued, so add a HasPrefix("-") guard in the qoder branch (claude has the same latent bug).
          family: qoder-branch-copies-claude
          round: 2
        - id: BR-13
          severity: Minor
          title: qoder --tools variadic branch is unreachable because --tools is absent from freshValueOption("qoder")
          detail: The commit message claims "--tools is variadic" and the test row does not exercise it. Add --tools to the qoder value list or delete the branch.
          family: unreachable-branch-unbacked-claim
          round: 2
        - id: BR-14
          severity: Minor
          title: Plan Task 6 still lists validAgent and runcli.go edits that M1 already did; Task 2 stale on value flags
          detail: Append a plan Revisions entry moving those rows into Task 1 and recording the help-derived value-flag set.
          family: plan-prose-restates-diff
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-21T09:40:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: validAgent and runcli supportedAgents landed with the registry commit (model.go, runcli.go), and TestAgentInventoryParityWithSessionTables ranges AgentInventory() over the per-agent tables.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Plan Revisions records context-meter usage as a stated non-goal and puts the create-path shouldMintClaudeSessionID twin in Task 12 scope; no code change is owed at M1.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Plan Revisions folds a neutral-cwd capture and account-identity scrub policy, and says to satisfy assertFixtureIsMachineNeutral by its failure message; this is Task 9 scope and nothing in M1 contradicts it.
          round: 3
        - id: BR-4
          disposition: withdrawn
          note: Compressing approved task bodies conflicts with the append-only Revisions rule in the constitution (section 1); the Revisions entry enumerates the divergences instead.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Revisions names TestLiveNativeSessionShapeConformance with PAIR_LIVE_NATIVE_SESSIONS=1; I confirmed both in conformance_live_test.go:9-11.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Revisions states that M5's boundary is the final sdlc close, with no separate milestone-close.
          round: 3
        - id: BR-7
          disposition: addressed
          note: 'Revisions lists the non-goals: cloud sessions, subagent resume, context meter, progress-OSC authority, per-tool allowlist. None contradicts the issue Spec or Done-when.'
          round: 3
        - id: BR-8
          disposition: addressed
          note: freshVariadicOption is now per-agent with a default of false. In a scratch revert, TestValidateFreshAgentArgs (codex --add-dir rows) and TestFreshVariadicOptionIsPerAgent both fail.
          round: 3
        - id: BR-9
          disposition: addressed
          note: persistedConfigArgs and sessionwatch.StripResumeArgs now strip -r and --resume=. Reverting the -r strip turns TestQoderShortResumeRoundTrip and TestQoderExplicitResumeAndPersistedArgs red. The residual class gap is raised as a Minor below.
          round: 3
        - id: BR-10
          disposition: addressed
          note: The parity test carries a named qoder known-gap entry that M2 must delete, and the qoder known gap row in the CLI matrix pins the schema_near_miss diagnostic. Partial probe coverage is raised as a Minor below.
          round: 3
        - id: BR-11
          disposition: addressed
          note: validAgent and the usage line derive from sessioninventory supportedAgents, and atlas section 0 and checklist item 1 are corrected. The two-list launcher-versus-session split is bridged by the parity test.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Extractor and cluster scan are table-parameterized (explicitResumeForms, freshAgentSpecs), and a HasPrefix("-") guard is pinned for claude and qoder valueless --resume. The strip-site counterpart is in the new Minor.
          round: 3
        - id: BR-13
          disposition: addressed
          note: --tools is now in the qoder value-flag list, so the branch is reachable and the helper table pins it. The validator row itself is behaviourally inert, as noted under Minor.
          round: 3
        - id: BR-14
          disposition: addressed
          note: Revisions moves the validAgent and runcli rows into Task 1 and records the help-derived value-flag set, which matches qodercli --help.
          round: 3
      findings:
        - id: BR-15
          severity: Minor
          title: Resume-form set is hand-restated at four sites; glued `-r<id>` and valueless `--resume` still diverge between extract, strip and validate
          detail: 'Rule for the whole family: extractExplicitResume, persistedConfigArgs, sessionwatch.StripResumeArgs and the forbiddenFlags and forbiddenShort in freshAgentSpecs must all derive from ONE per-agent form table (explicitResumeForms). The round-trip test must then range over that table instead of three hand-listed rows. Two instances remain today. First, qoder `-rabc` is rejected by the validator''s cluster scan but is neither extracted nor stripped, so `pair qoder -rabc` persists it, then Alt+n composes `-rabc --resume sid2` and fails validation, the same failure BR-9 measured for `-r abc`. Claude has the same latent hole. Second, both strip sites drop the token after a valueless `--resume` or `-r` unconditionally, so `--resume --model m` leaves an orphan `m` prompt, although the extractor now guards this case. The two strip implementations (launcher and sessionwatch) are also near-duplicates that were each extended by hand.'
          family: resume-form-recognized-but-not-stripped
          round: 3
        - id: BR-16
          severity: Minor
          title: Parity test covers 3 of the session-side agent dispatch sites; sessionledger.isSupportedAgent is named in the gap message but never probed
          detail: 'This is the 4th finding in this family, so I state the rule instead of fixing another site: every per-agent dispatch reachable from AgentInventory() must either derive from one exported list or be probed by the parity test, and a known-gap message must not name a site the test does not probe. Today the test probes ScannerForAgent, CLI acceptance and sessionwatch.SupportsAgent. Its gap text also names sessionledger, whose `isSupportedAgent` (record.go:480) is a second hand copy of `claude|codex|agy|muse`. An M2 that flips scanner and watcher support but misses the ledger ends with a green parity test and ledger-rejected qoder records. Unprobed dispatch sites include NormalizeNativeEvent, ProviderContractFor, target.go:211, the incremental_inventory switches and the runtime_os native roots. Prefer deriving SupportsAgent and isSupportedAgent from one exported predicate, or add a ledger probe to the parity test.'
          family: agent-dispatch-registration-gap
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#300 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-21T09:26:04-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `agent-dispatch-registration-gap` Registry only half-joins in M1, and no parity test makes "every consumer derives" mechanical
  Task 1 adds the enum constant but validAgent (model.go:446) and runcli.go:18 supportedAgents (which feeds pair_inventory.go:120 sidecar parsing) wait for Task 6. Step 5 re-runs suites that never exercise qoder, so the M1 "fail-closed verified by test" claim cannot fail. No AgentInventory parity test exists (only agent_defaults_test.go:90). Move the two pure registry rows into Task 1 and add one test ranging launcher.AgentInventory() over every per-agent table.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `agent-dispatch-registration-gap` Hand inventory misses usage.go:60 (context meter) and the create-path session-id mint
  sessioninventory/usage.go:60 ParseTokenUsage is claude-only, so qoder silently yields no usage for pair context (contextcmd.go:72); extend the claude case or list it as a non-goal. launcher/agentargs.go:198 shouldMintClaudeSessionID is the create-path twin of wrap.go:2280; Task 12 only covers the restart path, so honoring --session-id would leave the two paths inconsistent.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `harness-test-oracle-mismatch` Task 9 Step 6 "every oracle the tests read" omits the machine-neutral fixture oracle
  assertFixtureIsMachineNeutral (harness_tty_fixture_test.go:294-311, expiry :185-190) fails any capture embedding the home path unless ttyFixtureEnvironmentGaps has an entry. A qoder capture taken in the repo cwd will likely embed it (agy/muse captures do). Also an ARCH-SECURE point: state a neutral-cwd capture and account-identity scrub policy. Replace the hand-listed oracles with "satisfy each oracle by its failure message".
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Minor] `plan-prose-restates-diff` Tasks 2/3/5/6/7/13/14 embed full implementation bodies and enumerated test rows; compress to strategy lines
  Already stale on arrival: Task 2's short-flag loop treats p and d as value-taking while its own row says -p is a bool and the value-flag list omits both. Keep one strategy line per risky function (ValidateFreshAgentArgs / extractExplicitResume: table + fuzz over argv forms; scanClaudeFamily / validateClaudeFamilyDelta: real sanitized transcript replay with split-point framing; observationNativeID and recognizer: captured-byte replay at every split).
  (carried from plan-quality PQ-4, deferred to the boundary review)
- **BR-5** [Minor] `unbacked-existing-behavior-claim` Task 7 Step 5 names a live conformance test that does not exist
  The real entry point is TestLiveNativeSessionShapeConformance, gated by PAIR_LIVE_NATIVE_SESSIONS=1 (conformance_live_test.go:9-11), not TestConformanceLive.
  (carried from plan-quality PQ-5, deferred to the boundary review)
- **BR-6** [Minor] `milestone-boundary-granularity` M5 has no milestone-close of its own; Task 20 goes straight to sdlc close
  Per AGENTS.md section 3 every Mx row commits to a milestone-close. Either state that the final sdlc close is M5's boundary or drop the M5 tag. M1 and M4 are small but distinct review surfaces, so no over-split flag.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-7** [Minor] `no-stated-non-goals` Plan states no non-goals
  Name what is deliberately not built: qoder --remote/--teleport cloud sessions, subagent transcript resume, context-meter usage (if usage.go is not extended), claude-only progress-OSC lifecycle authority (notification_rewriter.go:129), and a per-tool allowlist if qoder has none.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-21T09:26:04-07:00 (claude) — BLOCKED

### Raised

- **BR-8** [Critical] `refactor-changes-sibling-agent-behavior` freshVariadicOption dropped its claude-only gate, so codex `--add-dir X resume ID` now passes fresh-arg validation
  fresh_args.go:92 calls freshVariadicOption(agent, flag) for every agent and the function falls through to claude's variadic flag list, so codex --add-dir swallows the following positionals. Measured: ValidateFreshAgentArgs("codex", ["--add-dir","/x","resume","abc"]) errors at 1ec329f3 and returns nil at d1f36450, so a fresh launch can silently resume an existing conversation. No test covers it. Make the helper strictly per-agent (default false) and add codex and agy regression rows.
- **BR-9** [Important] `resume-form-recognized-but-not-stripped` qoder `-r <id>` is accepted by extractExplicitResume but never stripped from persisted or fresh args
  persistedConfigArgs strips only --resume (both forms) and sessionwatch.StripResumeArgs only the space form; -r survives. Measured: relaunch composes `-r abc --resume sid2`, and FreshAgentArgs then ValidateFreshAgentArgs fails with `qoder argument "-r" selects an existing conversation`, breaking Alt+n and continuation. The claimed test only asserts --resume, which passes without qoder code. Enumerate the resume forms against every strip and validate site (or drop -r as claude does) and add a failing-first round-trip test.
- **BR-10** [Important] `agent-dispatch-registration-gap` M1 fail-closed intermediate is asserted but untested; no registry parity test, and `pair session-inventory` now emits a permanent qoder schema_near_miss
  The diff adds no qoder row to wrapcmd, sessioninventory, couchcore or couchtty tests, so the "verified green" claim cannot fail. With qoder in runcli's supportedAgents and no scanner, `pair session-inventory` with no --agent emits a schema_near_miss warning for qoder on every run (measured, exit 0). Add a parity test over launcher.AgentInventory() with a named known-gap entry for qoder that M2 must delete, and pin the intermediate diagnostic.
- **BR-11** [Minor] `hand-restated-registry` Agent set restated by hand at five sites (usage string x3, validAgent, runcli supportedAgents) plus golden; atlas says "join both"
  Derive the usage line and validAgent from one list so the next harness touches one row, and correct the atlas checklist item 1, which under-counts the sessioninventory sites.
- **BR-12** [Minor] `qoder-branch-copies-claude` qoder extractExplicitResume branch and short-flag cluster loop are near-copies of claude's; --resume followed by a flag returns the flag as the id
  Parameterize the per-agent flag names and cluster value letters. Qoder's --resume [id] is optional-valued, so add a HasPrefix("-") guard in the qoder branch (claude has the same latent bug).
- **BR-13** [Minor] `unreachable-branch-unbacked-claim` qoder --tools variadic branch is unreachable because --tools is absent from freshValueOption("qoder")
  The commit message claims "--tools is variadic" and the test row does not exercise it. Add --tools to the qoder value list or delete the branch.
- **BR-14** [Minor] `plan-prose-restates-diff` Plan Task 6 still lists validAgent and runcli.go edits that M1 already did; Task 2 stale on value flags
  Append a plan Revisions entry moving those rows into Task 1 and recording the help-derived value-flag set.

## Round 3 — 2026-09-21T09:40:44-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — validAgent and runcli supportedAgents landed with the registry commit (model.go, runcli.go), and TestAgentInventoryParityWithSessionTables ranges AgentInventory() over the per-agent tables.
- BR-2 — addressed — Plan Revisions records context-meter usage as a stated non-goal and puts the create-path shouldMintClaudeSessionID twin in Task 12 scope; no code change is owed at M1.
- BR-3 — addressed — Plan Revisions folds a neutral-cwd capture and account-identity scrub policy, and says to satisfy assertFixtureIsMachineNeutral by its failure message; this is Task 9 scope and nothing in M1 contradicts it.
- BR-4 — withdrawn — Compressing approved task bodies conflicts with the append-only Revisions rule in the constitution (section 1); the Revisions entry enumerates the divergences instead.
- BR-5 — addressed — Revisions names TestLiveNativeSessionShapeConformance with PAIR_LIVE_NATIVE_SESSIONS=1; I confirmed both in conformance_live_test.go:9-11.
- BR-6 — addressed — Revisions states that M5's boundary is the final sdlc close, with no separate milestone-close.
- BR-7 — addressed — Revisions lists the non-goals: cloud sessions, subagent resume, context meter, progress-OSC authority, per-tool allowlist. None contradicts the issue Spec or Done-when.
- BR-8 — addressed — freshVariadicOption is now per-agent with a default of false. In a scratch revert, TestValidateFreshAgentArgs (codex --add-dir rows) and TestFreshVariadicOptionIsPerAgent both fail.
- BR-9 — addressed — persistedConfigArgs and sessionwatch.StripResumeArgs now strip -r and --resume=. Reverting the -r strip turns TestQoderShortResumeRoundTrip and TestQoderExplicitResumeAndPersistedArgs red. The residual class gap is raised as a Minor below.
- BR-10 — addressed — The parity test carries a named qoder known-gap entry that M2 must delete, and the qoder known gap row in the CLI matrix pins the schema_near_miss diagnostic. Partial probe coverage is raised as a Minor below.
- BR-11 — addressed — validAgent and the usage line derive from sessioninventory supportedAgents, and atlas section 0 and checklist item 1 are corrected. The two-list launcher-versus-session split is bridged by the parity test.
- BR-12 — addressed — Extractor and cluster scan are table-parameterized (explicitResumeForms, freshAgentSpecs), and a HasPrefix("-") guard is pinned for claude and qoder valueless --resume. The strip-site counterpart is in the new Minor.
- BR-13 — addressed — --tools is now in the qoder value-flag list, so the branch is reachable and the helper table pins it. The validator row itself is behaviourally inert, as noted under Minor.
- BR-14 — addressed — Revisions moves the validAgent and runcli rows into Task 1 and records the help-derived value-flag set, which matches qodercli --help.

### Raised

- **BR-15** [Minor] `resume-form-recognized-but-not-stripped` Resume-form set is hand-restated at four sites; glued `-r<id>` and valueless `--resume` still diverge between extract, strip and validate
  Rule for the whole family: extractExplicitResume, persistedConfigArgs, sessionwatch.StripResumeArgs and the forbiddenFlags and forbiddenShort in freshAgentSpecs must all derive from ONE per-agent form table (explicitResumeForms). The round-trip test must then range over that table instead of three hand-listed rows. Two instances remain today. First, qoder `-rabc` is rejected by the validator's cluster scan but is neither extracted nor stripped, so `pair qoder -rabc` persists it, then Alt+n composes `-rabc --resume sid2` and fails validation, the same failure BR-9 measured for `-r abc`. Claude has the same latent hole. Second, both strip sites drop the token after a valueless `--resume` or `-r` unconditionally, so `--resume --model m` leaves an orphan `m` prompt, although the extractor now guards this case. The two strip implementations (launcher and sessionwatch) are also near-duplicates that were each extended by hand.
- **BR-16** [Minor] `agent-dispatch-registration-gap` Parity test covers 3 of the session-side agent dispatch sites; sessionledger.isSupportedAgent is named in the gap message but never probed
  This is the 4th finding in this family, so I state the rule instead of fixing another site: every per-agent dispatch reachable from AgentInventory() must either derive from one exported list or be probed by the parity test, and a known-gap message must not name a site the test does not probe. Today the test probes ScannerForAgent, CLI acceptance and sessionwatch.SupportsAgent. Its gap text also names sessionledger, whose `isSupportedAgent` (record.go:480) is a second hand copy of `claude|codex|agy|muse`. An M2 that flips scanner and watcher support but misses the ledger ends with a green parity test and ledger-rejected qoder records. Unprobed dispatch sites include NormalizeNativeEvent, ProviderContractFor, target.go:211, the incremental_inventory switches and the runtime_os native roots. Prefer deriving SupportsAgent and isSupportedAgent from one exported predicate, or add a ledger probe to the parity test.

## Open findings

- **BR-15** [Minor] `resume-form-recognized-but-not-stripped` Resume-form set is hand-restated at four sites; glued `-r<id>` and valueless `--resume` still diverge between extract, strip and validate
- **BR-16** [Minor] `agent-dispatch-registration-gap` Parity test covers 3 of the session-side agent dispatch sites; sessionledger.isSupportedAgent is named in the gap message but never probed
