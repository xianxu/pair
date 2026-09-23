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
    - "n": 4
      timestamp: "2026-09-21T10:28:42-07:00"
      agent: claude
      findings:
        - id: BR-17
          severity: Critical
          title: Fresh-arg validator no longer rejects short-flag clusters like -pr sid (regression from base)
          detail: 'This is the 3rd finding in family resume-form-recognized-but-not-stripped. Removing "r" from forbiddenShort (fresh_args.go:26, :34; claude and qoder) in favour of resumeform.Selector loses rejection of any cluster where r is not the first letter. A differential run at base 367610e7 vs head shows ValidateFreshAgentArgs(claude|qoder, ["-pr","sid"]), ["-vr","sid"] and ["-hr","x"] error at base and return nil at head. A "fresh" launch given -pr <sid> reaches the CLI and resumes an existing conversation, which is what the guard exists to prevent. Selector only matches an exact -r token or a token starting with -r; the Form comment''s claim that a cluster containing the selector "normalizes to the glued reading" is false for r in a non-first position. TestResumeFormTableRoundTrip and TestResumeSpellingDivergences only test -r<id> and -r. Rule covering the class: the set of tokens a fresh launch refuses must be a superset of what any earlier site refused, and cluster letters must derive from resumeform.Forms, not a hand-kept forbiddenShort string. Fix: add a resumeform helper (e.g. ClusterLetters(agent) from the single-letter Glued spellings) and have forbidsCluster consult it alongside ''c''. Add a table-ranged test that puts each Glued letter after a bool short letter (-p<letter>) and requires rejection, and keep the base rows -cr, -rp, -pc.'
          family: resume-form-recognized-but-not-stripped
          round: 4
        - id: BR-18
          severity: Important
          title: Qoder wiring at ProviderContractFor and AdvanceTargetValidation is pinned by no test; the BR-16 rule was only partly carried out
          detail: 'This is the 5th finding in family agent-dispatch-registration-gap. Do not fix only this instance. Mutation checks on a scratch copy of head show that deleting the AgentQoder case from ProviderContractFor turns no test red, and neither does deleting it from AdvanceTargetValidation. AdvanceTargetValidation (switch at incremental_inventory.go:198, no default) then returns the prior state unchanged with the frame offset advanced, so the watcher consumes qoder bytes without applying them and emits no diagnostic. ValidateTargetWork (:138) silently skips. The per-agent test tables still list only claude/codex/muse: TestProviderContractFor, TestAppendOnlyProviderConformance (a qoder row passes when added in scratch), provider_live_fake_test.go (its validateLiveRecords default returns ErrArtifactChanged for qoder), native_large_record_test, scan_fuzz_test and query_test. The plan Revisions (BR-16) bound M2 to the rule that every per-agent dispatch reachable from AgentInventory() is derived from one list or probed by the parity test, but only sessionledger was added to the probe. Not probed: ProviderContractFor, observationNativeID, artifactScannerShape, ValidateTargetWork/AdvanceTargetValidation, NormalizeNativeEvent and the runtime roots. Rule: one per-agent capability table, or a parity test ranging AgentInventory() through the testdata/native/<agent>/v1 fixtures and exercising every dispatch, plus fail-closed default arms in the two switches. Range the per-agent tables above from the same list.'
          family: agent-dispatch-registration-gap
          round: 4
        - id: BR-19
          severity: Important
          title: Epoch-millis timestamps are accepted with no range check and fabricate chronology
          detail: 'claudeFamilyTime.nativeTime (scan_claude.go:46) turns any int64 into a metadata-sourced instant via time.UnixMilli, while the ISO path is implicitly bounded to years 0000-9999 by RFC3339 parsing. A qoder record with "timestamp":9223372036854775807 is accepted with no diagnostic. `pair session-inventory --agent qoder --json` then printed created_at "292278994-08-17T07:12:55.807Z", which is not RFC3339, and the poisoned session sorts as the newest. activity.go:48 copies the value into a time.Time-typed json field, and json.Marshal of that year fails with "year outside of range [0,9999]". A record of -62135596800000 makes ValidateScannerState fail on the zero time. This transcript is input this program did not produce (ARCH-SECURE). Fix: parse the millis into a typed instant at the boundary and reject values outside a sane window (e.g. 2000-01-01 to year 9999), returning an error so the record disputes visibly like a malformed ISO string. Add a scan test row for it.'
          family: untrusted-input-parsed-without-bounds
          round: 4
        - id: BR-20
          severity: Important
          title: Qoder-only grammar admissions silently widened claude's scanner and event grammar
          detail: 'This is the 2nd finding in family refactor-changes-sibling-agent-behavior. Task 5 promised the refactor is "invisible to the claude suite", but two claude admissions changed. First, claudeRecord.Timestamp is now claudeFamilyTime for every agent, so ValidateClaudeDelta on {"type":"user","sessionId":"<id>","timestamp":1787907630000} returned disputed=false with a 2026 chronology (probe), where the old string field made it a malformed-record dispute. Second, normalizeClaudeEvent (event.go:113) is shared, so claude records of type workspace-directories, runtime-config, worktree-state and active-leaf are now EventIgnored instead of EventNearMiss. Both weaken claude''s near-miss drift detector, and no claude-side negative test pins them. Rule: every grammar admission a new family member needs is a parameter of the shared core (e.g. a family spec carrying acceptsEpochMillis and extraIgnoredTypes), defaulting to the existing agent''s prior behavior, with a claude negative row per admission. Do not widen the shared type.'
          family: refactor-changes-sibling-agent-behavior
          round: 4
        - id: BR-21
          severity: Minor
          title: event.go comment overstates ignore-set completeness; file-history-snapshot still near-misses on real qoder transcripts
          detail: 'Tally over the 10 real ~/.qoder transcripts: 1834 accepted, 2818 ignored, 53 near-miss, all 53 of type file-history-snapshot. Claude shows the same near-miss baseline for many types (mode, permission-mode, file-history-*), so this is consistent noise, not a blocker. Add the type with evidence or soften the comment "Without them the whole qoder stream is near-miss".'
          family: unbacked-existing-behavior-claim
          round: 4
        - id: BR-22
          severity: Minor
          title: resumeform.Strip is agent-agnostic for glued -r<x> across all agents
          detail: Strip drops any token starting with -r plus a non-empty, non-'=' remainder for every agent, including codex, muse and agy (single-dash long flags). A legitimate -r* flag of another agent would be lost from persisted args. Currently latent; the old strip only removed exact -r and -r=. Consider passing the agent so only its own spellings are removed.
          family: resume-form-recognized-but-not-stripped
          round: 4
        - id: BR-23
          severity: Minor
          title: resumeform.Forms is an exported mutable package map
          detail: Four packages read one writable exported map that is the single source of truth. Keep it unexported behind Selector/Extract/Strip and expose an accessor for the round-trip tests.
          family: hand-restated-registry
          round: 4
        - id: BR-24
          severity: Minor
          title: Plan has no M2 Revisions entry; live-conformance and manual steps are unlogged; Task 7 names the wrong test file
          detail: 'The only in-window Revisions entry covers the M1 advisories. Task 7 names events_test.go (actual file is event_test.go), and Task 7 Step 5 and Task 8 Step 3 have no logged evidence. I ran the qoder-only conformance against real ~/.qoder in a scratch copy: ok, 10 nodes, 7 roots, no diagnostics. Whole-suite TestLiveNativeSessionShapeConformance fails on this machine because of agy drift (pre-existing, unrelated to M2). Record the evidence and the claude-family admissions in a Revisions entry.'
          family: plan-prose-restates-diff
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-09-21T10:59:59-07:00"
      agent: claude
      dispose:
        - id: BR-17
          disposition: addressed
          note: launcher/fresh_args.go:55 consults resumeform.ShortLetters; TestFreshValidatorRefusesResumeLettersInClusters (table-ranged over Forms(), base rows kept) goes red on -pr/-vr/-hr for claude and qoder when the derivation is removed (mutation run).
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: 'No fix is committed at head. In the uncommitted tree, deleting the qoder arm from AdvanceTargetValidation, from ValidateTargetWork, or from the runtime_os native roots turns NO test red. TestEveryAgentDispatchParity never calls ValidateTargetWork/AdvanceTargetValidation despite its doc comment, and it ranges a hard-coded list. The ValidateTargetWork default arm sets err but the loop''s `if err != nil ... continue` drops it with no diagnostic, so it is still a silent skip. Rule: one per-agent delta-validator table used by both switches (fail closed with a diagnostic); export the single agent list and range the parity test from it; probe ValidateTargetWork, AdvanceTargetValidation and NativeRoots per agent; then re-run all seven arm-deletion mutations and require red.'
          round: 5
        - id: BR-19
          disposition: addressed
          note: scan_claude.go:39-42,82-92 bounds millis to 2000-01-01..9999-12-31 and disputes visibly; TestQoderMillisTimestampBounds (MaxInt64 and -62135596800000 rows) goes red when the bound is removed.
          round: 5
        - id: BR-20
          disposition: addressed
          note: acceptsMillis lives on claudeFamilySpec and claudeFamilyNoiseTypes is per-agent; claude negative rows (TestIncrementalClaudeRejectsNumericTimestamp, event_test.go:50-53) go red under both mutations (claude accepting millis, claude gaining the qoder noise set).
          round: 5
        - id: BR-21
          disposition: addressed
          note: file-history-snapshot joins qoder's ignore set, and the event.go comment now cites a dated measurement instead of the overclaim.
          round: 5
        - id: BR-22
          disposition: addressed
          note: resumeform.Strip(agent, args) uses only the agent's own form; all production callers thread the agent (agentargs.go:222, sessionwatch.go:52); TestStripIsPerAgent pins codex/agy/claude preservation.
          round: 5
        - id: BR-23
          disposition: addressed
          note: forms is unexported behind a copying Forms() accessor; tests range the accessor.
          round: 5
        - id: BR-24
          disposition: not-addressed
          note: Head's plan has no M2 Revisions entry. The uncommitted draft (plan.md:815-828) names non-existent fields (acceptsEpochMillis, extraIgnoredTypes; code has acceptsMillis and the function claudeFamilyNoiseTypes), claims every dispatch is probed and the default arms fail closed (both disproved by the BR-18 mutations), and has a typo in a family slug. Correct it and commit it with the code it describes.
          round: 5
      findings:
        - id: BR-25
          severity: Minor
          title: Issue Log line 184 (this window) records BR-18/BR-24 as delivered; no commit contains them and the working-tree version only partly delivers them
          detail: 'The Log says qoder rows, fail-closed defaults and a parity probe for "every JSONL agent" landed; git shows the code/test/plan changes uncommitted (8 modified files plus untracked dispatch_parity_test.go). Rule: a Log/plan claim that a coverage or fail-closed property holds must be committed alongside the code and backed by an arm-deletion mutation that turns red. Correct the claim, or land the missing probes, before recording it.'
          family: unbacked-existing-behavior-claim
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-09-21T11:07:10-07:00"
      agent: claude
      dispose:
        - id: BR-18
          disposition: not-addressed
          note: 'Mutation on HEAD copy: deleting the qoder arm of AdvanceTargetValidation, either new default arm, or (in sessioninventory) the ValidateTargetWork qoder arm / runtime_os qoder root turns nothing red in sessioninventory, sessionwatch, launcher or sessionledger; ValidateTargetWork and the runtime root are caught only by an older launcher test. dispatch_parity_test.go never calls ValidateTargetWork/AdvanceTargetValidation (test-local validateAgentDelta switch), ranges a hard-coded list, and the ValidateTargetWork default arm still drops err silently at the `continue`. Rule: export the single agent list, range the parity test from it, drive the production incremental switches, add unknown-agent rows with a diagnostic, then require every arm-deletion mutation red.'
          round: 6
        - id: BR-24
          disposition: not-addressed
          note: Revisions entry now exists with the evidence, but it names non-existent fields (acceptsEpochMillis/extraIgnoredTypes vs acceptsMillis/claudeFamilyNoiseTypes), keeps the slug typo untrusted-inputarsed-without-bounds, and claims every dispatch is probed and both default arms fail closed (disproved by the BR-18 mutations). Correct the entry.
          round: 6
        - id: BR-25
          disposition: not-addressed
          note: Code is committed now, but the overclaim class persists at three sites (issue Log line, plan Revisions bullet, dispatch_parity_test.go:11-16 comment) claiming coverage that an arm-deletion mutation does not turn red. State only what is pinned, or land the BR-18 probes first.
          round: 6
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-09-21T11:24:03-07:00"
      agent: claude
      dispose:
        - id: BR-18
          disposition: addressed
          note: Qoder wiring is pinned at every per-agent dispatch site (16 arm-deletion mutations, all red) and the parity test ranges SupportedAgents(); the one unpinned fail-closed arm (Advance default) is carried under BR-25.
          round: 7
        - id: BR-24
          disposition: addressed
          note: Plan Revisions (plan lines ~815-829) records the Task 7 filename correction, live-conformance evidence and claude-family admissions; every cited test exists (TestQoderMillisTimestampBounds, TestIncrementalClaudeRejectsNumericTimestamp, TestEveryTableSpellingRoundTrips, TestResumeFormTableRoundTrip).
          round: 7
        - id: BR-25
          disposition: not-addressed
          note: 'The claim "unknown-agent tests pin both default arms" (Log 183, plan Revisions "BR-16 rule as delivered") is false for AdvanceTargetValidation: deleting its default arm turns no test red. The test returns ErrArtifactChanged at incremental_inventory.go:183-184 (prior.Results[key] is a zero IncrementalResult, so StableFileID "" != "dev:1/ino:1") and never reaches the switch. It also asserts only err != nil, not ErrArtifactChanged. Fix: build the prior from a real ValidateTargetWork result, set prior.State.Agent to "future", AddRoot for that agent, then assert errors.Is(err, ErrArtifactChanged) and that Results is unchanged; confirm red with the arm deleted. Then correct both prose claims to say what the test pins.'
          round: 7
      findings:
        - id: BR-26
          severity: Minor
          title: TestAdvanceTargetValidationPerAgent hardcodes its four-agent list instead of ranging SupportedAgents()
          detail: 'dispatch_parity_test.go:70-73 lists claude/codex/muse/qoder by hand, so a sixth JSONL agent is forced through the Validate chain (fixture required) but silently skipped by the Advance test; only the vacuous default-arm test would then stand between it and an unwired Advance switch. This is the 3rd finding in family hand-restated-registry. Rule: any test that claims to cover "every agent" ranges SupportedAgents() (skipping agy explicitly). The same test file also restates per-agent facts in five switch helpers (agentFixtureRoot, agentSchema, agentFixtureNativeID, agentFixtureRelative, agentAppendRecord), and targetKey duplicates the unexported targetArtifactKey; adding a fixture-descriptor table keyed by SupportedAgents() would make one row the only per-agent edit.'
          family: hand-restated-registry
          round: 7
        - id: BR-27
          severity: Minor
          title: The two fail-closed default arms have different shapes and neither uses the artifactDiagnostic helper
          detail: ValidateTargetWork's default (incremental_inventory.go:148-149) hand-builds a Diagnostic{} with no artifact context, while the lines just above use artifactDiagnostic(...); AdvanceTargetValidation's default (:210-211) emits no diagnostic at all and returns bare ErrArtifactChanged. Pick one shape (artifactDiagnostic with the observation's artifact in both, or a shared helper) so the watcher's fallback path reports why. The Advance default arm still carries no failing-without-it test, which is the BR-25 residual above.
          family: agent-dispatch-registration-gap
          round: 7
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 8
      timestamp: "2026-09-21T14:35:41-07:00"
      agent: claude
      findings:
        - id: BR-28
          severity: Critical
          title: Qoder raw rolling-buffer scan re-arms pickerActive after the confirming Enter (ARCH-ORDER)
          detail: 'Reproduced at head: paint overlay.raw or selection.raw, emitPlainCR consumes the flag, then one small chunk re-arms it from stale rolling bytes; the next composer Enter passes a bare CR and submits the draft. emitPlainCR clears only overlayTextTail; rolling is loop-local. Claude/codex OSC paths trim rolling at wrap.go:3115 and codex text never reads it. Codex has TestCheckOverlayOpen_CodexDoesNotRedetectStalePickerText (overlay_test.go:253); qoder has no counterpart. Rule: detector input that outlives flag consumption is reset at consumption. Fix by giving the raw window a proxy-owned tail cleared in emitPlainCR, or advance past the matched marker; add a paint, Enter, small-chunk test over both fixtures.'
          family: overlay-flag-rearmed-from-stale-input
          round: 8
        - id: BR-29
          severity: Important
          title: qoderComposerActive re-implements ruledBoxComposerActive instead of adding a spec (ARCH-DRY)
          detail: 'composer_recognizers.go duplicates the Cursor.Y+1 prompt scan and the ruledBoxBottomRule scan. The atlas paragraph edited in this diff says add a spec rather than a fourth near-copy, and plan Task 9 Step 2(b) orders the same. The real differences are promptCol and requireVisibleCursor spec fields. Rule: a ruled-box harness registers a ruledBoxComposerSpec and no other function owns that loop.'
          family: qoder-branch-copies-claude
          round: 8
        - id: BR-30
          severity: Important
          title: Qoder prompt column 1 and glyph set >/* are restated in the recognizer and in orientation.go
          detail: 'qoderComposerActive hard-codes promptCol=1 and ">"/"*"; orientationComposerActive and orientationPromptOK restate both. Muse''s precedent is musePromptGlyphs, shared so the two gates cannot disagree. Rule: one qoderPromptCol and one qoderPromptGlyphs authority read by both gates; M4''s scrollback/distill glyph consumers should derive from it too.'
          family: hand-restated-registry
          round: 8
        - id: BR-31
          severity: Important
          title: Shared orientationComposerActive now skips column-N rule cells for every agent; sibling behavior changes unpinned
          detail: 'orientation.go:190 skips claudeComposerRule for all agents, but only qoder needs it (composer.raw''s hidden cursor parks on the closing rule; without the skip the fixture fails). Measured with a claude box and cursor on the closing rule: orientationComposerActive goes false to true versus base. No test pins claude, codex, agy or muse. Rule: sibling behavior changes only through a per-profile field, with a negative row per sibling.'
          family: refactor-changes-sibling-agent-behavior
          round: 8
        - id: BR-32
          severity: Important
          title: Qoder create-path mint (createflow.go:596 AgentSessionExists(agent, ...)) is pinned by no test
          detail: 'Only the shouldMintSessionID predicate is tested. Reverting the probe to the literal "claude" leaves every launcher test green except sandbox failures. Rule: an agent-identity branch is either derived from a registry predicate or has a runCreate test per registry-true agent, using the agent-keyed fake so a hard-coded sibling is observable. Test: runCreate for qoder with agentSessions["qoder|MINTED-1"]=true expects MINTED-2 and --session-id MINTED-2.'
          family: agent-dispatch-registration-gap
          round: 8
        - id: BR-33
          severity: Minor
          title: ttyFixtureExpectation comment says selection.raw takes the shared default, but it has its own explicit row
          detail: Only overlay.raw is the shared default (""); the qoder row lists selection.raw false explicitly. Also the qoder prose-does-not-open-overlay test feeds spaced text while Qoder paints body rows glued, so it does not model the real false-positive risk for glued markers like forfuturesessions and Enterselect.
          family: unbacked-existing-behavior-claim
          round: 8
        - id: BR-34
          severity: Minor
          title: 'Plan and atlas lag M3: Task 14 still lists orientation.go, Goal cites 1.1.59, architecture.md:1203 says --session-id is claude-only'
          detail: orientation.go's qoder branch landed in M3 Task 9, so Task 14's orientationPromptOK map row is superseded and the M3 Revisions entry omits it. The plan Goal says v1.1.59 while fixtures are 1.1.60. atlas/architecture.md:1203 still says "For claude ... --session-id is deterministic" though qoder now pins too.
          family: plan-prose-restates-diff
          round: 8
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 9
      timestamp: "2026-09-21T15:03:05-07:00"
      agent: claude
      dispose:
        - id: BR-28
          disposition: addressed
          note: 'overlayRawTail is proxy-owned, mutated only under overlayMu, and cleared in emitPlainCR beside overlayTextTail. Scratch revert (drop the `p.overlayRawTail = nil` line) turns TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText red on both overlay.raw and selection.raw. The consumption sweep is complete: overlayTextTail and overlayRawTail are cleared, and the claude/codex OSC paths advance `rolling` past the last match (wrap.go:3136).'
          round: 9
        - id: BR-29
          disposition: addressed
          note: qoderComposerActive is a ruledBoxComposerSpec registration (promptCol, requireVisibleCursor). ruledBoxComposerActive owns the only ruled-box loop; the callers are muse, claude, the agy orientation fallback and qoder. The differential rows are unchanged and green.
          round: 9
        - id: BR-30
          disposition: addressed
          note: qoderPromptCol and qoderPromptGlyphs (composer_recognizers.go) are read by the recognizer spec, by orientationComposerActive (promptCol) and by orientationPromptOK. No second restatement of column 1 or of `>`/`*` remains in Go.
          round: 9
        - id: BR-31
          disposition: addressed
          note: The rule-cell skip is gated to qoder. With the gate removed in a scratch copy, TestOrientationRuleCellToleranceStaysPerProfile goes red for claude, muse and agy, while the qoder positive row stays true.
          round: 9
        - id: BR-32
          disposition: addressed
          note: TestRunLaunchForcedCreateQoderMintProbesQoderSessions uses the agent-keyed fake. With the probe reverted to the literal "claude" in a scratch copy, both subtests fail (MINTED-1 instead of MINTED-2; empty PAIR_SESSION_ID).
          round: 9
        - id: BR-33
          disposition: addressed
          note: The ttyFixtureExpectation comment now says only overlay.raw takes the shared declining default. A glued-prose negative row for "forfuturesessions" was added to TestOverlayDetectorByAgent. The class residual is raised as a new Minor below.
          round: 9
        - id: BR-34
          disposition: addressed
          note: Task 14 is amended, the Goal cites 1.1.60, and atlas/architecture.md:1203 now names the MintsSessionID set. The stale 1.1.59 fixture paths in the Core concepts table and Task 9 are covered by the M3 Revisions entry; the sweep gap is in the atlas finding below.
          round: 9
      findings:
        - id: BR-35
          severity: Important
          title: Qoder raw window is truncated to 512 bytes before it is scanned, so it is not split-proof for chunks longer than about 500 bytes
          detail: 'detectQoderOverlayOpen (wrap.go:925-930) appends the chunk, trims to the last rollingTailLen bytes, then scans. A marker that straddles a chunk boundary inside an escape is missed whenever the second chunk carries more than about 500 bytes after the split. Measured in a scratch copy: first chunk `...Enter\x1b[2`, second chunk `3mselect·Esccancel` plus filler. Armed=true with 0 and 100 bytes of filler, armed=false with 400, 600 and 2000. The atlas and plan claim the byte-contiguous window cannot be corrupted by a split. TestHarnessTTYFixtureConformance cannot see this, because both marker paints sit within about 200 bytes of the end of their fixtures, so every replayed split leaves a short second chunk. Fix: scan stripTerminalControls(prevTail+data) and only then bound the carry. Add a test that puts the split inside the marker''s escape with at least 1 KB of trailing bytes. The composer gate still forces bare CR on the captured picker shapes, which is why this is Important rather than Critical.'
          family: detector-carry-bounded-before-scan
          round: 9
        - id: BR-36
          severity: Important
          title: atlas/architecture.md still enumerates profiles without Qoder at :694 (keymaps), :702 (ruled-box sharing) and :704 (conformance expectation)
          detail: 'This is the 5th finding in family `hand-restated-registry`, and the rule matters more than this instance. Rule: when a harness registers, grep the previous newest harness (`grep -n -i muse atlas/*.md`) and extend every hit that enumerates sibling harnesses in the same commit; log the sweep. Today :694 lists keymaps for Claude/Codex/Agy/Muse only (not `\`-CR/CR/Ctrl-U for Qoder). :702 says muse and claude "share one ruledBoxComposerActive ... prompt glyph at column 0", which is now false: Qoder is a third spec, with promptCol 1 and requireVisibleCursor false. :704 lists the composer.raw keymap expectation without Qoder. :700 and :1203 were edited in this same range, two lines away. Line 903 ("Claude, Codex, Agy, and Muse record parsing") is also stale from M2; sweep it too.'
          family: hand-restated-registry
          round: 9
        - id: BR-37
          severity: Minor
          title: requireVisibleCursor defaults permissive, and the agy orientation fallback's hidden-cursor decline is pinned by no test
          detail: 'This is the 5th finding in family `refactor-changes-sibling-agent-behavior`. Rule: a spec field added for one harness must default to the prior behaviour of every existing spec, so invert it to `allowHiddenCursor` and only Qoder sets it. Then no sibling needs touching and a forgotten field cannot fail open. Measured: flipping requireVisibleCursor to false on claude reddens TestClaudeComposerActiveSnapshotDifferential, and on muse it reddens TestMuseComposerActiveSnapshotDifferential and TestMuseFixtureEvidence. Flipping it on agyUncoloredOrientationComposer (orientation.go:300) turns no test red. Add a hidden-cursor negative row for the agy uncolored path.'
          family: refactor-changes-sibling-agent-behavior
          round: 9
        - id: BR-38
          severity: Minor
          title: '"Permission Required" is ordinary English, which the atlas rule and the dropped forfuturesessions marker say a marker must not be'
          detail: BR-33 fixed the instance it named, not the class of markers that agent output can produce. "Permission Required" survives the strip with real spaces and appears in any transcript, tool output or source file that mentions the phrase (this repo's own atlas and wrap.go do). It arms pickerActive, and the next composer Enter then passes a bare CR and submits a draft. The code comment defends the header as the generic marker, which is a defensible tradeoff but the opposite of the rule stated two paragraphs later. Either require co-occurrence with a body marker, or amend the atlas rule to say the header is exempt and why, and pin a spaced-prose negative row.
          family: overlay-marker-matches-agent-prose
          round: 9
        - id: BR-39
          severity: Minor
          title: orientation.go branches on p.agentBasename == "qoder" at two sites plus orientationPromptOK, instead of a per-profile orientation field
          detail: Behaviour is now pinned per sibling, so this is design only. A profile-carried promptCol and ruleCellTolerant would drop the string compares and make the M4 glyph consumers derive the same way.
          family: agent-dispatch-registration-gap
          round: 9
        - id: BR-40
          severity: Minor
          title: Qoder adds the fourth copy of the overlay tail-carry block and a third identical marker-scan loop
          detail: ARCH-DRY. detectQoderOverlayText is identical to detectAgyOverlayText and detectCodexOverlayText apart from the marker slice, and the `visible = p.overlayTextTail + visible; p.overlayTextTail = textSuffix(...)` block now sits at wrap.go:792, 828, 865 and 935. A shared `firstMarker(visible, markers)` and a `p.overlayVisible(data)` helper would collapse them.
          family: hand-restated-registry
          round: 9
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 10
      timestamp: "2026-09-21T15:25:08-07:00"
      agent: claude
      dispose:
        - id: BR-35
          disposition: addressed
          note: wrap.go:940-946 scans stripTerminalControls(carry+data) before bounding; TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk reddens at filler=600/2000 when the order is reverted in scratch.
          round: 10
        - id: BR-36
          disposition: addressed
          note: architecture.md :694 (Qoder keymap), :702 (three specs, promptCol/allowHiddenCursor), :704 (conformance) and :903 now name Qoder; atlas couch.md:321 is explicitly M5 Task 19's.
          round: 10
        - id: BR-37
          disposition: addressed
          note: Field inverted to allowHiddenCursor (only Qoder sets it); TestOrientationUncoloredAgyRequiresVisibleCursor reddens when the agy uncolored spec allows a hidden cursor.
          round: 10
        - id: BR-38
          disposition: addressed
          note: 'Atlas amendment option taken: how-to line 87 states the header exemption and its bound (one Enter consumes pickerActive); spaced-prose negative row added in overlay_test.go.'
          round: 10
        - id: BR-39
          disposition: addressed
          note: orientationPromptCol and orientationRuleCellTolerant now live on harnessTTYProfile; orientationComposerActive has no agentBasename compare left. orientationPromptOK keeps its agent-keyed glyph read on purpose.
          round: 10
        - id: BR-40
          disposition: addressed
          note: overlayVisible and firstMarker replace the four carry blocks and three loops; Muse keeps its own folded loop for the reason stated at wrap.go:872.
          round: 10
      findings:
        - id: BR-41
          severity: Minor
          title: Shared chunk pump trims rolling to 512 bytes before checkOverlayOpen, so Claude/Codex OSC detectors miss an OSC followed by 512+ bytes in one chunk
          detail: Pre-existing and outside this window; it is the class BR-35 belongs to, and the Qoder instance was fixed while the siblings were not. wrap.go:3122-3127 appends to rolling, trims it to rollingTailLen, and only then calls checkOverlayOpen(data, *rolling) and the oscRe scan. detectClaudeOverlayOpen and detectCodexQuestionOSC read only rolling, so an OSC 777 or OSC 9 with more than 512 bytes after it in the same read is dropped before the scan. The rule is that any carry must be scanned at full carry+chunk length and bounded afterwards; here that means moving the trim below the two scans. Not shown to bite in practice (those OSCs usually arrive alone), so no gate blocks on it.
          family: detector-carry-bounded-before-scan
          round: 10
      boundary: M3
      recipe: milestone-review
      blocked: false
    - "n": 11
      timestamp: "2026-09-21T16:37:48-07:00"
      agent: claude
      findings:
        - id: BR-42
          severity: Important
          title: trimLiveTail's empty-box check `t == glyph` can never match qoder's space-prefixed " >" glyph
          detail: 'distill.go:70 compares TrimSpace(line) to the raw glyph, so ">" never equals " >". The registry comment calls promptGlyphChar the single source for both the turn-boundary regex and the empty-box detection, but only the regex reader is tested. Rule: a registry row is not landed until a table test ranges the registry and drives every reader. Fix: range promptGlyphChar over scanTurnBoundaries and trimLiveTail, and normalise the glyph once.'
          family: agent-dispatch-registration-gap
          round: 11
        - id: BR-43
          severity: Important
          title: distill's qoder glyph " >" restates qoderPromptGlyphs/qoderPromptCol by hand with no parity guard
          detail: 'Plan Task 14 requires every M4 consumer to derive from the authority. The Lua consumer got a parity test; distill.go:28 got a comment, and its own test only compares the literal to itself, so changing qoderPromptCol reddens nothing. Distill also drops yolo `*` while the Lua row keeps it, on the same evidence. Rule: enumerate consumers of qoderPromptGlyphs/Col (recognizer, orientation, Lua, distill) and require each derived or parity-pinned. Fix: export a changelogcmd accessor and assert it in a wrapcmd parity test, pinning the `*` omission.'
          family: hand-restated-registry
          round: 11
        - id: BR-44
          severity: Important
          title: runQoder persists a transcript per slug/changelog call; --no-session-persistence is not passed
          detail: 'ARCH-FUNERAL. Measured under ~/.qoder/projects/-private-tmp and the TMPDIR project dir: each `qoder -p` leaves an ~9KB jsonl plus a session dir, and `qoder --help` lists --no-session-persistence. The slug fires at turn end, so growth is per turn with no sweep. Fix: add the flag to runQoder''s argv, update wantArgs, and confirm it composes with -p in the live conformance test.'
          family: headless-call-leaves-durable-residue
          round: 11
        - id: BR-45
          severity: Important
          title: The carried qoder footer-trim item lives only in a Revisions paragraph; Task 17's steps omit it
          detail: 'The plan Revisions entry (plan line 872) says qoder''s live footer matches no isFooterChrome row, so trimLiveTail strips nothing and Alt+l anchors on volatile chrome (the #58 FullRedistill class), and it calls this "now an M5 Task 17 scope item". Task 17 Steps 1-4 (plan lines 736-739) have no Alt+l/distill step, so the M5 checklist will not exercise it although the registration ships the degraded state now. Fix: add a Task 17 step to capture the settled footer, extend isFooterChrome, and verify a no-op press.'
          family: deferred-work-not-in-executing-task
          round: 11
        - id: BR-46
          severity: Minor
          title: Parity test's class escape uses Lua-pattern dialect but scrollback.lua patterns are Vim regex
          detail: scrollback_glyph_parity_test.go:46 escapes with %], %-, %^ while PROMPT_PATTERN_BY_AGENT is consumed by vim.fn.search. The comment claims the derivation stays total for any future glyph; a `-` or `]` glyph would derive a wrong class (matching `%` too) and the test would then require it. Dead for `>` and `*` today. Use Vim collection escapes.
          family: unbacked-existing-behavior-claim
          round: 11
        - id: BR-47
          severity: Minor
          title: Standard-set allow rules registered at user scope, broader than claude's project-scoped precedent
          detail: ARCH-SECURE. Bash(git:*), Bash(make:*), Bash(zellij:*) in ~/.qoder/settings.json apply in every repo qoder opens, while claude's equivalents are project-scoped and verb-specific. The A/B ran five simple probes and did not confirm chained commands such as `git status && ...` fall outside the prefix rule. Prefer <repo>/.qoder/settings.local.json or verify the chained-command behavior.
          family: permission-allowlist-scope
          round: 11
        - id: BR-48
          severity: Minor
          title: Pump scan-before-bound change is pinned for Claude only; Codex's OSC detector and the OSC telemetry loop share the path
          detail: 'TestHandleChunk_OscScannedBeforeCarryIsBounded covers Claude''s picker OSC. detectCodexQuestionOSC(rolling) and the OSC telemetry loop now also see the unbounded carry+chunk with no regression row. Rule: a shared-path change is pinned over every profile that reads it. Fix: table-drive the test over each profile with an OSC overlay detector.'
          family: refactor-changes-sibling-agent-behavior
          round: 11
      boundary: M4
      recipe: milestone-review
      blocked: true
    - "n": 12
      timestamp: "2026-09-21T16:49:31-07:00"
      agent: claude
      dispose:
        - id: BR-42
          disposition: addressed
          note: trimLiveTail now trims the glyph once; TestPromptGlyphRowsDriveBothReaders ranges the registry over both readers. Reverting the TrimSpace makes the qoder row fail ("trimLiveTail leaves the bare input box").
          round: 12
        - id: BR-43
          disposition: addressed
          note: changelogcmd.PromptGlyph plus TestDistillQoderGlyphTracksPromptAuthority derive from qoderPromptCol and pin the * omission. Setting qoderPromptCol to 0 reddens both parity tests.
          round: 12
        - id: BR-44
          disposition: addressed
          note: runQoder passes --no-session-persistence and wantArgs pins it; reverting it fails TestRunQoderDispatchesToQoderCLI. The live conformance run passed and left no new files under ~/.qoder/projects.
          round: 12
        - id: BR-45
          disposition: addressed
          note: Plan Task 17 Step 3 now owns the settled-footer capture, the isFooterChrome extension and the no-op Alt+l check.
          round: 12
        - id: BR-46
          disposition: addressed
          note: The parity test escapes in the Vim dialect (backslash, ], -, ^), which is what vim.fn.search consumes; the sorted class still derives to ^ [*>] and matches the Lua row.
          round: 12
        - id: BR-47
          disposition: addressed
          note: The allowlist moved to the repo-local .qoder/settings.local.json with measured repo-bound scope and a measured chained-command denial; user scope is restored.
          round: 12
        - id: BR-48
          disposition: addressed
          note: The OSC pump test is table-driven over claude and codex; both rows fail when the bound-first order is restored in scratch.
          round: 12
      findings:
        - id: BR-49
          severity: Minor
          title: runClaude and runMuse still persist a transcript per headless call, the class BR-44 fixed only for qoder
          detail: 'This is the 3rd finding in family `headless-call-leaves-durable-residue`. The rule: every headless runner in model.go must pass its agent''s no-persistence flag or carry a comment naming why it cannot, pinned by one table test over the agents Run dispatches. `claude --help` lists `--no-session-persistence` and `muse exec --help` lists `--no-session-log`; `codex exec` already passes `--ephemeral`. `~/.claude/projects` holds residue project dirs from `-private-tmp` cwds. The gap predates this diff and is outside qoder''s scope, so it does not block M4. Log it as a follow-up issue rather than widening #300.'
          family: headless-call-leaves-durable-residue
          round: 12
      boundary: M4
      recipe: milestone-review
      blocked: false
    - "n": 13
      timestamp: "2026-09-22T20:59:09-07:00"
      agent: codex
      dispose:
        - id: BR-15
          disposition: addressed
          note: resumeform.Forms drives the consumers; table-ranging tests cover glued and valueless forms.
          round: 13
        - id: BR-16
          disposition: addressed
          note: TestAgentInventoryParityWithSessionTables now probes sessionledger.ParseLedger.
          round: 13
        - id: BR-25
          disposition: addressed
          note: The parity and fail-closed tests and implementation are committed in the pinned range.
          round: 13
        - id: BR-26
          disposition: addressed
          note: TestAdvanceTargetValidationPerAgent now ranges SupportedAgents().
          round: 13
        - id: BR-27
          disposition: addressed
          note: Both default arms use artifactDiagnostic; unknown-agent tests exercise both paths.
          round: 13
        - id: BR-41
          disposition: addressed
          note: The shared pump scans before trimming; the Claude and Codex long-chunk rows exercise it.
          round: 13
        - id: BR-49
          disposition: addressed
          note: The requested out-of-scope follow-up was committed as pair#304; Claude and Muse behavior remains for that issue.
          round: 13
      findings:
        - id: BR-50
          severity: Critical
          title: Core concepts labels scanner IO and runQoder as PURE (ARCH-PURE)
          detail: 'The plan''s PURE table lists scanClaudeFamily, ScanQoder, and runQoder, although the scanners consume Runtime and runQoder launches a subprocess. Reclassify these entry points as INTEGRATION and record the correction in ## Revisions.'
          family: pure-integration-classification-drift
          round: 13
      recipe: milestone-review
      blocked: true
    - "n": 14
      timestamp: "2026-09-22T21:03:51-07:00"
      agent: codex
      dispose:
        - id: BR-50
          disposition: addressed
          note: The pinned plan diff moves scanClaudeFamily, scanClaudeFamilyFile, ScanQoder, and runQoder out of PURE; scan_claude.go reads Runtime and model.go launches qoder. The 2026-09-22 Revisions entry records the taxonomy correction.
          round: 14
      findings:
        - id: BR-51
          severity: Critical
          title: Core concepts locations still contradict the delivered capture and settings locations
          detail: 'The plan table at lines 59 and 64 names qoder/1.1.59/ and user-scope ~/.qoder/settings.json; the pinned tree has captures only under qoder/1.1.60/, and the plan''s later revision says the allowlist moved to repo-local .qoder/settings.local.json. This is the 5th finding in family plan-prose-restates-diff. Sweep every Core concepts location against the delivered tree and final decisions, then correct the table and record the sweep in ## Revisions.'
          family: plan-prose-restates-diff
          round: 14
      recipe: milestone-review
      blocked: true
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

## Round 4 — 2026-09-21T10:28:42-07:00 (claude) — BLOCKED

### Raised

- **BR-17** [Critical] `resume-form-recognized-but-not-stripped` Fresh-arg validator no longer rejects short-flag clusters like -pr sid (regression from base)
  This is the 3rd finding in family resume-form-recognized-but-not-stripped. Removing "r" from forbiddenShort (fresh_args.go:26, :34; claude and qoder) in favour of resumeform.Selector loses rejection of any cluster where r is not the first letter. A differential run at base 367610e7 vs head shows ValidateFreshAgentArgs(claude|qoder, ["-pr","sid"]), ["-vr","sid"] and ["-hr","x"] error at base and return nil at head. A "fresh" launch given -pr <sid> reaches the CLI and resumes an existing conversation, which is what the guard exists to prevent. Selector only matches an exact -r token or a token starting with -r; the Form comment's claim that a cluster containing the selector "normalizes to the glued reading" is false for r in a non-first position. TestResumeFormTableRoundTrip and TestResumeSpellingDivergences only test -r<id> and -r. Rule covering the class: the set of tokens a fresh launch refuses must be a superset of what any earlier site refused, and cluster letters must derive from resumeform.Forms, not a hand-kept forbiddenShort string. Fix: add a resumeform helper (e.g. ClusterLetters(agent) from the single-letter Glued spellings) and have forbidsCluster consult it alongside 'c'. Add a table-ranged test that puts each Glued letter after a bool short letter (-p<letter>) and requires rejection, and keep the base rows -cr, -rp, -pc.
- **BR-18** [Important] `agent-dispatch-registration-gap` Qoder wiring at ProviderContractFor and AdvanceTargetValidation is pinned by no test; the BR-16 rule was only partly carried out
  This is the 5th finding in family agent-dispatch-registration-gap. Do not fix only this instance. Mutation checks on a scratch copy of head show that deleting the AgentQoder case from ProviderContractFor turns no test red, and neither does deleting it from AdvanceTargetValidation. AdvanceTargetValidation (switch at incremental_inventory.go:198, no default) then returns the prior state unchanged with the frame offset advanced, so the watcher consumes qoder bytes without applying them and emits no diagnostic. ValidateTargetWork (:138) silently skips. The per-agent test tables still list only claude/codex/muse: TestProviderContractFor, TestAppendOnlyProviderConformance (a qoder row passes when added in scratch), provider_live_fake_test.go (its validateLiveRecords default returns ErrArtifactChanged for qoder), native_large_record_test, scan_fuzz_test and query_test. The plan Revisions (BR-16) bound M2 to the rule that every per-agent dispatch reachable from AgentInventory() is derived from one list or probed by the parity test, but only sessionledger was added to the probe. Not probed: ProviderContractFor, observationNativeID, artifactScannerShape, ValidateTargetWork/AdvanceTargetValidation, NormalizeNativeEvent and the runtime roots. Rule: one per-agent capability table, or a parity test ranging AgentInventory() through the testdata/native/<agent>/v1 fixtures and exercising every dispatch, plus fail-closed default arms in the two switches. Range the per-agent tables above from the same list.
- **BR-19** [Important] `untrusted-input-parsed-without-bounds` Epoch-millis timestamps are accepted with no range check and fabricate chronology
  claudeFamilyTime.nativeTime (scan_claude.go:46) turns any int64 into a metadata-sourced instant via time.UnixMilli, while the ISO path is implicitly bounded to years 0000-9999 by RFC3339 parsing. A qoder record with "timestamp":9223372036854775807 is accepted with no diagnostic. `pair session-inventory --agent qoder --json` then printed created_at "292278994-08-17T07:12:55.807Z", which is not RFC3339, and the poisoned session sorts as the newest. activity.go:48 copies the value into a time.Time-typed json field, and json.Marshal of that year fails with "year outside of range [0,9999]". A record of -62135596800000 makes ValidateScannerState fail on the zero time. This transcript is input this program did not produce (ARCH-SECURE). Fix: parse the millis into a typed instant at the boundary and reject values outside a sane window (e.g. 2000-01-01 to year 9999), returning an error so the record disputes visibly like a malformed ISO string. Add a scan test row for it.
- **BR-20** [Important] `refactor-changes-sibling-agent-behavior` Qoder-only grammar admissions silently widened claude's scanner and event grammar
  This is the 2nd finding in family refactor-changes-sibling-agent-behavior. Task 5 promised the refactor is "invisible to the claude suite", but two claude admissions changed. First, claudeRecord.Timestamp is now claudeFamilyTime for every agent, so ValidateClaudeDelta on {"type":"user","sessionId":"<id>","timestamp":1787907630000} returned disputed=false with a 2026 chronology (probe), where the old string field made it a malformed-record dispute. Second, normalizeClaudeEvent (event.go:113) is shared, so claude records of type workspace-directories, runtime-config, worktree-state and active-leaf are now EventIgnored instead of EventNearMiss. Both weaken claude's near-miss drift detector, and no claude-side negative test pins them. Rule: every grammar admission a new family member needs is a parameter of the shared core (e.g. a family spec carrying acceptsEpochMillis and extraIgnoredTypes), defaulting to the existing agent's prior behavior, with a claude negative row per admission. Do not widen the shared type.
- **BR-21** [Minor] `unbacked-existing-behavior-claim` event.go comment overstates ignore-set completeness; file-history-snapshot still near-misses on real qoder transcripts
  Tally over the 10 real ~/.qoder transcripts: 1834 accepted, 2818 ignored, 53 near-miss, all 53 of type file-history-snapshot. Claude shows the same near-miss baseline for many types (mode, permission-mode, file-history-*), so this is consistent noise, not a blocker. Add the type with evidence or soften the comment "Without them the whole qoder stream is near-miss".
- **BR-22** [Minor] `resume-form-recognized-but-not-stripped` resumeform.Strip is agent-agnostic for glued -r<x> across all agents
  Strip drops any token starting with -r plus a non-empty, non-'=' remainder for every agent, including codex, muse and agy (single-dash long flags). A legitimate -r* flag of another agent would be lost from persisted args. Currently latent; the old strip only removed exact -r and -r=. Consider passing the agent so only its own spellings are removed.
- **BR-23** [Minor] `hand-restated-registry` resumeform.Forms is an exported mutable package map
  Four packages read one writable exported map that is the single source of truth. Keep it unexported behind Selector/Extract/Strip and expose an accessor for the round-trip tests.
- **BR-24** [Minor] `plan-prose-restates-diff` Plan has no M2 Revisions entry; live-conformance and manual steps are unlogged; Task 7 names the wrong test file
  The only in-window Revisions entry covers the M1 advisories. Task 7 names events_test.go (actual file is event_test.go), and Task 7 Step 5 and Task 8 Step 3 have no logged evidence. I ran the qoder-only conformance against real ~/.qoder in a scratch copy: ok, 10 nodes, 7 roots, no diagnostics. Whole-suite TestLiveNativeSessionShapeConformance fails on this machine because of agy drift (pre-existing, unrelated to M2). Record the evidence and the claude-family admissions in a Revisions entry.

## Round 5 — 2026-09-21T10:59:59-07:00 (claude) — BLOCKED

### Disposed

- BR-17 — addressed — launcher/fresh_args.go:55 consults resumeform.ShortLetters; TestFreshValidatorRefusesResumeLettersInClusters (table-ranged over Forms(), base rows kept) goes red on -pr/-vr/-hr for claude and qoder when the derivation is removed (mutation run).
- BR-18 — not-addressed — No fix is committed at head. In the uncommitted tree, deleting the qoder arm from AdvanceTargetValidation, from ValidateTargetWork, or from the runtime_os native roots turns NO test red. TestEveryAgentDispatchParity never calls ValidateTargetWork/AdvanceTargetValidation despite its doc comment, and it ranges a hard-coded list. The ValidateTargetWork default arm sets err but the loop's `if err != nil ... continue` drops it with no diagnostic, so it is still a silent skip. Rule: one per-agent delta-validator table used by both switches (fail closed with a diagnostic); export the single agent list and range the parity test from it; probe ValidateTargetWork, AdvanceTargetValidation and NativeRoots per agent; then re-run all seven arm-deletion mutations and require red.
- BR-19 — addressed — scan_claude.go:39-42,82-92 bounds millis to 2000-01-01..9999-12-31 and disputes visibly; TestQoderMillisTimestampBounds (MaxInt64 and -62135596800000 rows) goes red when the bound is removed.
- BR-20 — addressed — acceptsMillis lives on claudeFamilySpec and claudeFamilyNoiseTypes is per-agent; claude negative rows (TestIncrementalClaudeRejectsNumericTimestamp, event_test.go:50-53) go red under both mutations (claude accepting millis, claude gaining the qoder noise set).
- BR-21 — addressed — file-history-snapshot joins qoder's ignore set, and the event.go comment now cites a dated measurement instead of the overclaim.
- BR-22 — addressed — resumeform.Strip(agent, args) uses only the agent's own form; all production callers thread the agent (agentargs.go:222, sessionwatch.go:52); TestStripIsPerAgent pins codex/agy/claude preservation.
- BR-23 — addressed — forms is unexported behind a copying Forms() accessor; tests range the accessor.
- BR-24 — not-addressed — Head's plan has no M2 Revisions entry. The uncommitted draft (plan.md:815-828) names non-existent fields (acceptsEpochMillis, extraIgnoredTypes; code has acceptsMillis and the function claudeFamilyNoiseTypes), claims every dispatch is probed and the default arms fail closed (both disproved by the BR-18 mutations), and has a typo in a family slug. Correct it and commit it with the code it describes.

### Raised

- **BR-25** [Minor] `unbacked-existing-behavior-claim` Issue Log line 184 (this window) records BR-18/BR-24 as delivered; no commit contains them and the working-tree version only partly delivers them
  The Log says qoder rows, fail-closed defaults and a parity probe for "every JSONL agent" landed; git shows the code/test/plan changes uncommitted (8 modified files plus untracked dispatch_parity_test.go). Rule: a Log/plan claim that a coverage or fail-closed property holds must be committed alongside the code and backed by an arm-deletion mutation that turns red. Correct the claim, or land the missing probes, before recording it.

## Round 6 — 2026-09-21T11:07:10-07:00 (claude) — BLOCKED

### Disposed

- BR-18 — not-addressed — Mutation on HEAD copy: deleting the qoder arm of AdvanceTargetValidation, either new default arm, or (in sessioninventory) the ValidateTargetWork qoder arm / runtime_os qoder root turns nothing red in sessioninventory, sessionwatch, launcher or sessionledger; ValidateTargetWork and the runtime root are caught only by an older launcher test. dispatch_parity_test.go never calls ValidateTargetWork/AdvanceTargetValidation (test-local validateAgentDelta switch), ranges a hard-coded list, and the ValidateTargetWork default arm still drops err silently at the `continue`. Rule: export the single agent list, range the parity test from it, drive the production incremental switches, add unknown-agent rows with a diagnostic, then require every arm-deletion mutation red.
- BR-24 — not-addressed — Revisions entry now exists with the evidence, but it names non-existent fields (acceptsEpochMillis/extraIgnoredTypes vs acceptsMillis/claudeFamilyNoiseTypes), keeps the slug typo untrusted-inputarsed-without-bounds, and claims every dispatch is probed and both default arms fail closed (disproved by the BR-18 mutations). Correct the entry.
- BR-25 — not-addressed — Code is committed now, but the overclaim class persists at three sites (issue Log line, plan Revisions bullet, dispatch_parity_test.go:11-16 comment) claiming coverage that an arm-deletion mutation does not turn red. State only what is pinned, or land the BR-18 probes first.

## Round 7 — 2026-09-21T11:24:03-07:00 (claude) — passed

### Disposed

- BR-18 — addressed — Qoder wiring is pinned at every per-agent dispatch site (16 arm-deletion mutations, all red) and the parity test ranges SupportedAgents(); the one unpinned fail-closed arm (Advance default) is carried under BR-25.
- BR-24 — addressed — Plan Revisions (plan lines ~815-829) records the Task 7 filename correction, live-conformance evidence and claude-family admissions; every cited test exists (TestQoderMillisTimestampBounds, TestIncrementalClaudeRejectsNumericTimestamp, TestEveryTableSpellingRoundTrips, TestResumeFormTableRoundTrip).
- BR-25 — not-addressed — The claim "unknown-agent tests pin both default arms" (Log 183, plan Revisions "BR-16 rule as delivered") is false for AdvanceTargetValidation: deleting its default arm turns no test red. The test returns ErrArtifactChanged at incremental_inventory.go:183-184 (prior.Results[key] is a zero IncrementalResult, so StableFileID "" != "dev:1/ino:1") and never reaches the switch. It also asserts only err != nil, not ErrArtifactChanged. Fix: build the prior from a real ValidateTargetWork result, set prior.State.Agent to "future", AddRoot for that agent, then assert errors.Is(err, ErrArtifactChanged) and that Results is unchanged; confirm red with the arm deleted. Then correct both prose claims to say what the test pins.

### Raised

- **BR-26** [Minor] `hand-restated-registry` TestAdvanceTargetValidationPerAgent hardcodes its four-agent list instead of ranging SupportedAgents()
  dispatch_parity_test.go:70-73 lists claude/codex/muse/qoder by hand, so a sixth JSONL agent is forced through the Validate chain (fixture required) but silently skipped by the Advance test; only the vacuous default-arm test would then stand between it and an unwired Advance switch. This is the 3rd finding in family hand-restated-registry. Rule: any test that claims to cover "every agent" ranges SupportedAgents() (skipping agy explicitly). The same test file also restates per-agent facts in five switch helpers (agentFixtureRoot, agentSchema, agentFixtureNativeID, agentFixtureRelative, agentAppendRecord), and targetKey duplicates the unexported targetArtifactKey; adding a fixture-descriptor table keyed by SupportedAgents() would make one row the only per-agent edit.
- **BR-27** [Minor] `agent-dispatch-registration-gap` The two fail-closed default arms have different shapes and neither uses the artifactDiagnostic helper
  ValidateTargetWork's default (incremental_inventory.go:148-149) hand-builds a Diagnostic{} with no artifact context, while the lines just above use artifactDiagnostic(...); AdvanceTargetValidation's default (:210-211) emits no diagnostic at all and returns bare ErrArtifactChanged. Pick one shape (artifactDiagnostic with the observation's artifact in both, or a shared helper) so the watcher's fallback path reports why. The Advance default arm still carries no failing-without-it test, which is the BR-25 residual above.

## Round 8 — 2026-09-21T14:35:41-07:00 (claude) — BLOCKED

### Raised

- **BR-28** [Critical] `overlay-flag-rearmed-from-stale-input` Qoder raw rolling-buffer scan re-arms pickerActive after the confirming Enter (ARCH-ORDER)
  Reproduced at head: paint overlay.raw or selection.raw, emitPlainCR consumes the flag, then one small chunk re-arms it from stale rolling bytes; the next composer Enter passes a bare CR and submits the draft. emitPlainCR clears only overlayTextTail; rolling is loop-local. Claude/codex OSC paths trim rolling at wrap.go:3115 and codex text never reads it. Codex has TestCheckOverlayOpen_CodexDoesNotRedetectStalePickerText (overlay_test.go:253); qoder has no counterpart. Rule: detector input that outlives flag consumption is reset at consumption. Fix by giving the raw window a proxy-owned tail cleared in emitPlainCR, or advance past the matched marker; add a paint, Enter, small-chunk test over both fixtures.
- **BR-29** [Important] `qoder-branch-copies-claude` qoderComposerActive re-implements ruledBoxComposerActive instead of adding a spec (ARCH-DRY)
  composer_recognizers.go duplicates the Cursor.Y+1 prompt scan and the ruledBoxBottomRule scan. The atlas paragraph edited in this diff says add a spec rather than a fourth near-copy, and plan Task 9 Step 2(b) orders the same. The real differences are promptCol and requireVisibleCursor spec fields. Rule: a ruled-box harness registers a ruledBoxComposerSpec and no other function owns that loop.
- **BR-30** [Important] `hand-restated-registry` Qoder prompt column 1 and glyph set >/* are restated in the recognizer and in orientation.go
  qoderComposerActive hard-codes promptCol=1 and ">"/"*"; orientationComposerActive and orientationPromptOK restate both. Muse's precedent is musePromptGlyphs, shared so the two gates cannot disagree. Rule: one qoderPromptCol and one qoderPromptGlyphs authority read by both gates; M4's scrollback/distill glyph consumers should derive from it too.
- **BR-31** [Important] `refactor-changes-sibling-agent-behavior` Shared orientationComposerActive now skips column-N rule cells for every agent; sibling behavior changes unpinned
  orientation.go:190 skips claudeComposerRule for all agents, but only qoder needs it (composer.raw's hidden cursor parks on the closing rule; without the skip the fixture fails). Measured with a claude box and cursor on the closing rule: orientationComposerActive goes false to true versus base. No test pins claude, codex, agy or muse. Rule: sibling behavior changes only through a per-profile field, with a negative row per sibling.
- **BR-32** [Important] `agent-dispatch-registration-gap` Qoder create-path mint (createflow.go:596 AgentSessionExists(agent, ...)) is pinned by no test
  Only the shouldMintSessionID predicate is tested. Reverting the probe to the literal "claude" leaves every launcher test green except sandbox failures. Rule: an agent-identity branch is either derived from a registry predicate or has a runCreate test per registry-true agent, using the agent-keyed fake so a hard-coded sibling is observable. Test: runCreate for qoder with agentSessions["qoder|MINTED-1"]=true expects MINTED-2 and --session-id MINTED-2.
- **BR-33** [Minor] `unbacked-existing-behavior-claim` ttyFixtureExpectation comment says selection.raw takes the shared default, but it has its own explicit row
  Only overlay.raw is the shared default (""); the qoder row lists selection.raw false explicitly. Also the qoder prose-does-not-open-overlay test feeds spaced text while Qoder paints body rows glued, so it does not model the real false-positive risk for glued markers like forfuturesessions and Enterselect.
- **BR-34** [Minor] `plan-prose-restates-diff` Plan and atlas lag M3: Task 14 still lists orientation.go, Goal cites 1.1.59, architecture.md:1203 says --session-id is claude-only
  orientation.go's qoder branch landed in M3 Task 9, so Task 14's orientationPromptOK map row is superseded and the M3 Revisions entry omits it. The plan Goal says v1.1.59 while fixtures are 1.1.60. atlas/architecture.md:1203 still says "For claude ... --session-id is deterministic" though qoder now pins too.

## Round 9 — 2026-09-21T15:03:05-07:00 (claude) — BLOCKED

### Disposed

- BR-28 — addressed — overlayRawTail is proxy-owned, mutated only under overlayMu, and cleared in emitPlainCR beside overlayTextTail. Scratch revert (drop the `p.overlayRawTail = nil` line) turns TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText red on both overlay.raw and selection.raw. The consumption sweep is complete: overlayTextTail and overlayRawTail are cleared, and the claude/codex OSC paths advance `rolling` past the last match (wrap.go:3136).
- BR-29 — addressed — qoderComposerActive is a ruledBoxComposerSpec registration (promptCol, requireVisibleCursor). ruledBoxComposerActive owns the only ruled-box loop; the callers are muse, claude, the agy orientation fallback and qoder. The differential rows are unchanged and green.
- BR-30 — addressed — qoderPromptCol and qoderPromptGlyphs (composer_recognizers.go) are read by the recognizer spec, by orientationComposerActive (promptCol) and by orientationPromptOK. No second restatement of column 1 or of `>`/`*` remains in Go.
- BR-31 — addressed — The rule-cell skip is gated to qoder. With the gate removed in a scratch copy, TestOrientationRuleCellToleranceStaysPerProfile goes red for claude, muse and agy, while the qoder positive row stays true.
- BR-32 — addressed — TestRunLaunchForcedCreateQoderMintProbesQoderSessions uses the agent-keyed fake. With the probe reverted to the literal "claude" in a scratch copy, both subtests fail (MINTED-1 instead of MINTED-2; empty PAIR_SESSION_ID).
- BR-33 — addressed — The ttyFixtureExpectation comment now says only overlay.raw takes the shared declining default. A glued-prose negative row for "forfuturesessions" was added to TestOverlayDetectorByAgent. The class residual is raised as a new Minor below.
- BR-34 — addressed — Task 14 is amended, the Goal cites 1.1.60, and atlas/architecture.md:1203 now names the MintsSessionID set. The stale 1.1.59 fixture paths in the Core concepts table and Task 9 are covered by the M3 Revisions entry; the sweep gap is in the atlas finding below.

### Raised

- **BR-35** [Important] `detector-carry-bounded-before-scan` Qoder raw window is truncated to 512 bytes before it is scanned, so it is not split-proof for chunks longer than about 500 bytes
  detectQoderOverlayOpen (wrap.go:925-930) appends the chunk, trims to the last rollingTailLen bytes, then scans. A marker that straddles a chunk boundary inside an escape is missed whenever the second chunk carries more than about 500 bytes after the split. Measured in a scratch copy: first chunk `...Enter\x1b[2`, second chunk `3mselect·Esccancel` plus filler. Armed=true with 0 and 100 bytes of filler, armed=false with 400, 600 and 2000. The atlas and plan claim the byte-contiguous window cannot be corrupted by a split. TestHarnessTTYFixtureConformance cannot see this, because both marker paints sit within about 200 bytes of the end of their fixtures, so every replayed split leaves a short second chunk. Fix: scan stripTerminalControls(prevTail+data) and only then bound the carry. Add a test that puts the split inside the marker's escape with at least 1 KB of trailing bytes. The composer gate still forces bare CR on the captured picker shapes, which is why this is Important rather than Critical.
- **BR-36** [Important] `hand-restated-registry` atlas/architecture.md still enumerates profiles without Qoder at :694 (keymaps), :702 (ruled-box sharing) and :704 (conformance expectation)
  This is the 5th finding in family `hand-restated-registry`, and the rule matters more than this instance. Rule: when a harness registers, grep the previous newest harness (`grep -n -i muse atlas/*.md`) and extend every hit that enumerates sibling harnesses in the same commit; log the sweep. Today :694 lists keymaps for Claude/Codex/Agy/Muse only (not `\`-CR/CR/Ctrl-U for Qoder). :702 says muse and claude "share one ruledBoxComposerActive ... prompt glyph at column 0", which is now false: Qoder is a third spec, with promptCol 1 and requireVisibleCursor false. :704 lists the composer.raw keymap expectation without Qoder. :700 and :1203 were edited in this same range, two lines away. Line 903 ("Claude, Codex, Agy, and Muse record parsing") is also stale from M2; sweep it too.
- **BR-37** [Minor] `refactor-changes-sibling-agent-behavior` requireVisibleCursor defaults permissive, and the agy orientation fallback's hidden-cursor decline is pinned by no test
  This is the 5th finding in family `refactor-changes-sibling-agent-behavior`. Rule: a spec field added for one harness must default to the prior behaviour of every existing spec, so invert it to `allowHiddenCursor` and only Qoder sets it. Then no sibling needs touching and a forgotten field cannot fail open. Measured: flipping requireVisibleCursor to false on claude reddens TestClaudeComposerActiveSnapshotDifferential, and on muse it reddens TestMuseComposerActiveSnapshotDifferential and TestMuseFixtureEvidence. Flipping it on agyUncoloredOrientationComposer (orientation.go:300) turns no test red. Add a hidden-cursor negative row for the agy uncolored path.
- **BR-38** [Minor] `overlay-marker-matches-agent-prose` "Permission Required" is ordinary English, which the atlas rule and the dropped forfuturesessions marker say a marker must not be
  BR-33 fixed the instance it named, not the class of markers that agent output can produce. "Permission Required" survives the strip with real spaces and appears in any transcript, tool output or source file that mentions the phrase (this repo's own atlas and wrap.go do). It arms pickerActive, and the next composer Enter then passes a bare CR and submits a draft. The code comment defends the header as the generic marker, which is a defensible tradeoff but the opposite of the rule stated two paragraphs later. Either require co-occurrence with a body marker, or amend the atlas rule to say the header is exempt and why, and pin a spaced-prose negative row.
- **BR-39** [Minor] `agent-dispatch-registration-gap` orientation.go branches on p.agentBasename == "qoder" at two sites plus orientationPromptOK, instead of a per-profile orientation field
  Behaviour is now pinned per sibling, so this is design only. A profile-carried promptCol and ruleCellTolerant would drop the string compares and make the M4 glyph consumers derive the same way.
- **BR-40** [Minor] `hand-restated-registry` Qoder adds the fourth copy of the overlay tail-carry block and a third identical marker-scan loop
  ARCH-DRY. detectQoderOverlayText is identical to detectAgyOverlayText and detectCodexOverlayText apart from the marker slice, and the `visible = p.overlayTextTail + visible; p.overlayTextTail = textSuffix(...)` block now sits at wrap.go:792, 828, 865 and 935. A shared `firstMarker(visible, markers)` and a `p.overlayVisible(data)` helper would collapse them.

## Round 10 — 2026-09-21T15:25:08-07:00 (claude) — passed

### Disposed

- BR-35 — addressed — wrap.go:940-946 scans stripTerminalControls(carry+data) before bounding; TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk reddens at filler=600/2000 when the order is reverted in scratch.
- BR-36 — addressed — architecture.md :694 (Qoder keymap), :702 (three specs, promptCol/allowHiddenCursor), :704 (conformance) and :903 now name Qoder; atlas couch.md:321 is explicitly M5 Task 19's.
- BR-37 — addressed — Field inverted to allowHiddenCursor (only Qoder sets it); TestOrientationUncoloredAgyRequiresVisibleCursor reddens when the agy uncolored spec allows a hidden cursor.
- BR-38 — addressed — Atlas amendment option taken: how-to line 87 states the header exemption and its bound (one Enter consumes pickerActive); spaced-prose negative row added in overlay_test.go.
- BR-39 — addressed — orientationPromptCol and orientationRuleCellTolerant now live on harnessTTYProfile; orientationComposerActive has no agentBasename compare left. orientationPromptOK keeps its agent-keyed glyph read on purpose.
- BR-40 — addressed — overlayVisible and firstMarker replace the four carry blocks and three loops; Muse keeps its own folded loop for the reason stated at wrap.go:872.

### Raised

- **BR-41** [Minor] `detector-carry-bounded-before-scan` Shared chunk pump trims rolling to 512 bytes before checkOverlayOpen, so Claude/Codex OSC detectors miss an OSC followed by 512+ bytes in one chunk
  Pre-existing and outside this window; it is the class BR-35 belongs to, and the Qoder instance was fixed while the siblings were not. wrap.go:3122-3127 appends to rolling, trims it to rollingTailLen, and only then calls checkOverlayOpen(data, *rolling) and the oscRe scan. detectClaudeOverlayOpen and detectCodexQuestionOSC read only rolling, so an OSC 777 or OSC 9 with more than 512 bytes after it in the same read is dropped before the scan. The rule is that any carry must be scanned at full carry+chunk length and bounded afterwards; here that means moving the trim below the two scans. Not shown to bite in practice (those OSCs usually arrive alone), so no gate blocks on it.

## Round 11 — 2026-09-21T16:37:48-07:00 (claude) — BLOCKED

### Raised

- **BR-42** [Important] `agent-dispatch-registration-gap` trimLiveTail's empty-box check `t == glyph` can never match qoder's space-prefixed " >" glyph
  distill.go:70 compares TrimSpace(line) to the raw glyph, so ">" never equals " >". The registry comment calls promptGlyphChar the single source for both the turn-boundary regex and the empty-box detection, but only the regex reader is tested. Rule: a registry row is not landed until a table test ranges the registry and drives every reader. Fix: range promptGlyphChar over scanTurnBoundaries and trimLiveTail, and normalise the glyph once.
- **BR-43** [Important] `hand-restated-registry` distill's qoder glyph " >" restates qoderPromptGlyphs/qoderPromptCol by hand with no parity guard
  Plan Task 14 requires every M4 consumer to derive from the authority. The Lua consumer got a parity test; distill.go:28 got a comment, and its own test only compares the literal to itself, so changing qoderPromptCol reddens nothing. Distill also drops yolo `*` while the Lua row keeps it, on the same evidence. Rule: enumerate consumers of qoderPromptGlyphs/Col (recognizer, orientation, Lua, distill) and require each derived or parity-pinned. Fix: export a changelogcmd accessor and assert it in a wrapcmd parity test, pinning the `*` omission.
- **BR-44** [Important] `headless-call-leaves-durable-residue` runQoder persists a transcript per slug/changelog call; --no-session-persistence is not passed
  ARCH-FUNERAL. Measured under ~/.qoder/projects/-private-tmp and the TMPDIR project dir: each `qoder -p` leaves an ~9KB jsonl plus a session dir, and `qoder --help` lists --no-session-persistence. The slug fires at turn end, so growth is per turn with no sweep. Fix: add the flag to runQoder's argv, update wantArgs, and confirm it composes with -p in the live conformance test.
- **BR-45** [Important] `deferred-work-not-in-executing-task` The carried qoder footer-trim item lives only in a Revisions paragraph; Task 17's steps omit it
  The plan Revisions entry (plan line 872) says qoder's live footer matches no isFooterChrome row, so trimLiveTail strips nothing and Alt+l anchors on volatile chrome (the #58 FullRedistill class), and it calls this "now an M5 Task 17 scope item". Task 17 Steps 1-4 (plan lines 736-739) have no Alt+l/distill step, so the M5 checklist will not exercise it although the registration ships the degraded state now. Fix: add a Task 17 step to capture the settled footer, extend isFooterChrome, and verify a no-op press.
- **BR-46** [Minor] `unbacked-existing-behavior-claim` Parity test's class escape uses Lua-pattern dialect but scrollback.lua patterns are Vim regex
  scrollback_glyph_parity_test.go:46 escapes with %], %-, %^ while PROMPT_PATTERN_BY_AGENT is consumed by vim.fn.search. The comment claims the derivation stays total for any future glyph; a `-` or `]` glyph would derive a wrong class (matching `%` too) and the test would then require it. Dead for `>` and `*` today. Use Vim collection escapes.
- **BR-47** [Minor] `permission-allowlist-scope` Standard-set allow rules registered at user scope, broader than claude's project-scoped precedent
  ARCH-SECURE. Bash(git:*), Bash(make:*), Bash(zellij:*) in ~/.qoder/settings.json apply in every repo qoder opens, while claude's equivalents are project-scoped and verb-specific. The A/B ran five simple probes and did not confirm chained commands such as `git status && ...` fall outside the prefix rule. Prefer <repo>/.qoder/settings.local.json or verify the chained-command behavior.
- **BR-48** [Minor] `refactor-changes-sibling-agent-behavior` Pump scan-before-bound change is pinned for Claude only; Codex's OSC detector and the OSC telemetry loop share the path
  TestHandleChunk_OscScannedBeforeCarryIsBounded covers Claude's picker OSC. detectCodexQuestionOSC(rolling) and the OSC telemetry loop now also see the unbounded carry+chunk with no regression row. Rule: a shared-path change is pinned over every profile that reads it. Fix: table-drive the test over each profile with an OSC overlay detector.

## Round 12 — 2026-09-21T16:49:31-07:00 (claude) — passed

### Disposed

- BR-42 — addressed — trimLiveTail now trims the glyph once; TestPromptGlyphRowsDriveBothReaders ranges the registry over both readers. Reverting the TrimSpace makes the qoder row fail ("trimLiveTail leaves the bare input box").
- BR-43 — addressed — changelogcmd.PromptGlyph plus TestDistillQoderGlyphTracksPromptAuthority derive from qoderPromptCol and pin the * omission. Setting qoderPromptCol to 0 reddens both parity tests.
- BR-44 — addressed — runQoder passes --no-session-persistence and wantArgs pins it; reverting it fails TestRunQoderDispatchesToQoderCLI. The live conformance run passed and left no new files under ~/.qoder/projects.
- BR-45 — addressed — Plan Task 17 Step 3 now owns the settled-footer capture, the isFooterChrome extension and the no-op Alt+l check.
- BR-46 — addressed — The parity test escapes in the Vim dialect (backslash, ], -, ^), which is what vim.fn.search consumes; the sorted class still derives to ^ [*>] and matches the Lua row.
- BR-47 — addressed — The allowlist moved to the repo-local .qoder/settings.local.json with measured repo-bound scope and a measured chained-command denial; user scope is restored.
- BR-48 — addressed — The OSC pump test is table-driven over claude and codex; both rows fail when the bound-first order is restored in scratch.

### Raised

- **BR-49** [Minor] `headless-call-leaves-durable-residue` runClaude and runMuse still persist a transcript per headless call, the class BR-44 fixed only for qoder
  This is the 3rd finding in family `headless-call-leaves-durable-residue`. The rule: every headless runner in model.go must pass its agent's no-persistence flag or carry a comment naming why it cannot, pinned by one table test over the agents Run dispatches. `claude --help` lists `--no-session-persistence` and `muse exec --help` lists `--no-session-log`; `codex exec` already passes `--ephemeral`. `~/.claude/projects` holds residue project dirs from `-private-tmp` cwds. The gap predates this diff and is outside qoder's scope, so it does not block M4. Log it as a follow-up issue rather than widening #300.

## Round 13 — 2026-09-22T20:59:09-07:00 (codex) — BLOCKED

### Disposed

- BR-15 — addressed — resumeform.Forms drives the consumers; table-ranging tests cover glued and valueless forms.
- BR-16 — addressed — TestAgentInventoryParityWithSessionTables now probes sessionledger.ParseLedger.
- BR-25 — addressed — The parity and fail-closed tests and implementation are committed in the pinned range.
- BR-26 — addressed — TestAdvanceTargetValidationPerAgent now ranges SupportedAgents().
- BR-27 — addressed — Both default arms use artifactDiagnostic; unknown-agent tests exercise both paths.
- BR-41 — addressed — The shared pump scans before trimming; the Claude and Codex long-chunk rows exercise it.
- BR-49 — addressed — The requested out-of-scope follow-up was committed as pair#304; Claude and Muse behavior remains for that issue.

### Raised

- **BR-50** [Critical] `pure-integration-classification-drift` Core concepts labels scanner IO and runQoder as PURE (ARCH-PURE)
  The plan's PURE table lists scanClaudeFamily, ScanQoder, and runQoder, although the scanners consume Runtime and runQoder launches a subprocess. Reclassify these entry points as INTEGRATION and record the correction in ## Revisions.

## Round 14 — 2026-09-22T21:03:51-07:00 (codex) — BLOCKED

### Disposed

- BR-50 — addressed — The pinned plan diff moves scanClaudeFamily, scanClaudeFamilyFile, ScanQoder, and runQoder out of PURE; scan_claude.go reads Runtime and model.go launches qoder. The 2026-09-22 Revisions entry records the taxonomy correction.

### Raised

- **BR-51** [Critical] `plan-prose-restates-diff` Core concepts locations still contradict the delivered capture and settings locations
  The plan table at lines 59 and 64 names qoder/1.1.59/ and user-scope ~/.qoder/settings.json; the pinned tree has captures only under qoder/1.1.60/, and the plan's later revision says the allowlist moved to repo-local .qoder/settings.local.json. This is the 5th finding in family plan-prose-restates-diff. Sweep every Core concepts location against the delivered tree and final decisions, then correct the table and record the sweep in ## Revisions.

## Open findings

- **BR-51** [Critical] `plan-prose-restates-diff` Core concepts locations still contradict the delivered capture and settings locations
