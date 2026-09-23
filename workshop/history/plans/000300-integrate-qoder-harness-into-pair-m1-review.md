# Boundary Review — pair#300 (milestone M1)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 08e9ec027c7a55bb1ae6d5054bbdeb0b61a82889..d1f36450aece424e091acc4cb4ae18c3393653f8 |
| command | sdlc milestone-close --issue 300 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-21T09:26:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers what the Plan claims: qoder joins both registries, fresh-args validation, `--resume <uuid>` resume token and explicit-resume extraction. The code is pure, table-tested and matches `qoder --help` v1.1.59 for the value-flag set. One Critical blocks the boundary: generalizing `freshVariadicOption` silently weakened the fresh-launch guard for `codex`. I reproduced it in scratch copies of the base and head commits. There are also two Important gaps: a resume form that is recognized but never stripped, and an untested M1 "fail-closed intermediate". The go tests I ran pass apart from PTY/pty-child failures ("operation not permitted") that look like the known sandbox limit; I did not run `make test`.

## Strengths
- Both `supportedAgents` and `AgentInventory()` derive from one slice, and `validAgent` and `runcli.go` joined in the same commit. That partly answers plan-gate PQ-1.
- The qoder value-flag table (`fresh_args.go:133`) and the forbidden set (`:55`) match the real `qoder --help`. This includes `--worktree [name]` being optional-valued while `-w` is `--cwd <dir>`, and `-r/--resume [id]`.
- The short-cluster handling is conservative in the safe direction. `-dc` is rejected, and `-mc` treats `c` as the model value. Both are in the table (`fresh_launch_test.go:15,27`).
- `composeResumeArgs` needed no change, because qoder falls into the append branch like claude.
- The launcher work stays in pure arg transforms with no IO. The atlas Part A claims I checked (`AgentInventory()` in `couchtty/menu_switchagent.go` and `gcruntime`, and the `pair resume <tag>` spawn) match the code.

## Critical
- **`fresh_args.go:92,143-153`** — the change from `agent == "claude" && freshVariadicOption(flag)` to `freshVariadicOption(agent, flag)` dropped the claude-only gate.
  - The function's fall-through `switch flag` now applies claude's variadic list (`--add-dir`, `--file`, …) to every agent.
  - `codex`'s `--add-dir` is in `codexValueGlobalOption`, so it now swallows the following positionals.
  - Measured: `ValidateFreshAgentArgs("codex", ["--add-dir","/x","resume","abc"])` and `["--add-dir","/x","fork","--last"]` were rejected at `1ec329f3` and return nil at `d1f36450`.
  - A "fresh" codex launch, such as couch switch-agent or continuation, can now silently resume an existing conversation, which is the guard's purpose.
  - Fix: make the function strictly per-agent (claude keeps its list, qoder gets its own list, default false) and add codex/agy rows such as `{"--add-dir","/x","resume","abc"}` to the rejected table. Nothing in the diff's tests fails on the regression.

## Important
- **`createlogic.go:87` vs `agentargs.go:211` / `sessionwatch.go:44`** — `extractExplicitResume("qoder")` recognizes `-r <id>` and `--resume=<id>`, but the strippers do not cover every form.
  - `persistedConfigArgs` strips only `--resume` in its space and inline forms, not `-r`.
  - `sessionwatch.StripResumeArgs` strips only the space form of `--resume`.
  - Measured with `-r abc`:
    - Persisted config args keep `-r abc`.
    - The tag-restart relaunch composes `[... -r abc --resume sid2]`, so resume args accumulate.
    - The fresh path (`FreshAgentArgs` then `ValidateFreshAgentArgs`) fails with `qoder argument "-r" selects an existing conversation`. That breaks Alt+n and continuation for a tag that was originally launched with `-r`.
  - Claude has no `-r` extract path, so this is new to qoder.
  - The commit message says accumulation is "pinned by test". `TestQoderExplicitResumeAndPersistedArgs`'s persisted assertion uses only `--resume`, which passes with no qoder change, so it pins nothing new.
  - ARCH-PURPOSE: the class is "every resume form the extractor accepts is stripped everywhere a persisted or fresh path reads argv". Write that enumeration, either by dropping `-r` from the extractor (as claude does) or by stripping `-r` and the inline form in both strippers, and add a failing-first round-trip test.
- **M1's "fail-closed intermediate" is asserted but untested (`1ec329f3` message; plan-gate PQ-1 not mechanized).** The diff adds no qoder row to any wrapcmd, sessioninventory, couchcore or couchtty test, so the "verified green" claim cannot fail on qoder.
  - A visible half-join already exists: with qoder in `runcli.go`'s `supportedAgents` and no scanner, `pair session-inventory` (no `--agent`) emits a permanent `schema_near_miss` warning diagnostic for `qoder` on every machine. I ran it against a fake runtime: `--json` shows `"code":"schema_near_miss","agent":"qoder"`, exit 0. Also, `ScannerForAgent`, `NormalizeNativeEvent`, `ProviderContractFor` and `incremental_inventory.go`'s three switches have no qoder case.
  - Add one parity test that ranges `launcher.AgentInventory()` over the per-agent tables. `launcher` already imports `sessioninventory`, so no import cycle. Give qoder a named, time-boxed known-gap entry that M2 must delete, and pin the intermediate diagnostic explicitly.

## Minor
- ARCH-DRY: `runcli.go:76,82,85` repeat the usage string three times, plus the golden JSON. `validAgent` (`model.go:447`) and runcli's `supportedAgents` restate the same set. This diff edited all five sites.
  - Derive the usage line and `validAgent` from one list.
  - The atlas checklist says "join both" registries, but M1 needed three touches (`AgentQoder`, `validAgent`, runcli's list). Update it, or better, make it true by derivation.
- ARCH-DRY: the `case "qoder"` in `extractExplicitResume` and the short-flag cluster loop in `fresh_args.go:58-68` are near-copies of the claude and agy branches with different literals. Parameterize them per agent.
  - Also, the extractor returns the next token as the id even when it is a flag: `["--resume","--model","m"]` gives `"--model"`. Qoder's `--resume [id]` is optional-valued, and claude has the same latent bug. A `HasPrefix(tok,"-")` guard in the qoder branch is cheap.
- `freshVariadicOption("qoder","--tools")` is unreachable, because `--tools` is not in `freshValueOption("qoder")` and so never enters the value-consuming block. The test row `{"--tools","a","b","--","hi"}` does not exercise it, and the commit message's "`--tools` is variadic" is unbacked. Either add `--tools` to the value list or delete the branch.
- README has no qoder mention. The plan defers the docs sweep to M5 (Task 19), which is acceptable because qoder is not yet operator-usable. Keep that task's sweep regex case- and separator-agnostic.
- The qoder flag table is hand-transcribed from `--help`. Consider a live, opt-in `PAIR_LIVE_*` check that every `--help` value option is classified, so drift is detected (ARCH-MOCK, later milestone).

## Test coverage notes
- Fresh-args rows for qoder are good but miss:
  - `-w` followed by a dash-leading value. It is accepted, which matches commander's required-value semantics, so it is fine, but it is worth one pinned row.
  - Any cross-agent regression row (the Critical).
- The resume tests cover `resumeToken`, `composeResumeArgs` and `extractExplicitResume`, but not the persist, relaunch and fresh round trip.

## Architectural notes
- **ARCH-PURE:** pass; the launcher changes are pure and table-tested.
- **ARCH-PURPOSE:** flagged, see the Critical and the `-r` finding.
- **ARCH-DRY:** flagged, see the Minors.
- **ARCH-SECURE:** flagged only through the Critical. Fresh argv comes from persisted config or couch records, so the validator is a trust boundary, and its NUL and `--` handling are preserved.
- **ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-ORDER, ARCH-FUNERAL:** pass at M1; no external calls, no state between events, nothing durable created.
- M2 must delete the known-gap parity entry and add qoder cases to `ScannerForAgent`, `NormalizeNativeEvent`, `ProviderContractFor`, the `incremental_inventory` switches, `observationNativeID`, and `SupportsAgent` in `sessionledger/record.go:480` and `sessionwatch.go:35`.
- Also do the still-unowned plan-gate items: `usage.go` context meter and the create-path `shouldMintClaudeSessionID`.

## Plan revision recommendations
Add a `## Revisions` entry saying:
1. Task 1 now also covers `validAgent`, `runcli.go`'s `supportedAgents` and usage strings, and the golden JSON, so Task 6 must drop those rows (its lines 383, 389, 478 and 480 are stale).
2. Task 2's implemented value-flag set follows `qoder --help`, not the plan's list, and `-p` and `-d` are not value-taking.
3. The `-r` handling decision and the parity-test known-gap mechanism from the findings above.

```findings
findings:
  - id: new
    severity: Critical
    family: refactor-changes-sibling-agent-behavior
    title: |
      freshVariadicOption dropped its claude-only gate, so codex `--add-dir X resume ID` now passes fresh-arg validation
    detail: |
      fresh_args.go:92 calls freshVariadicOption(agent, flag) for every agent and the function falls through to claude's variadic flag list, so codex --add-dir swallows the following positionals. Measured: ValidateFreshAgentArgs("codex", ["--add-dir","/x","resume","abc"]) errors at 1ec329f3 and returns nil at d1f36450, so a fresh launch can silently resume an existing conversation. No test covers it. Make the helper strictly per-agent (default false) and add codex and agy regression rows.
  - id: new
    severity: Important
    family: resume-form-recognized-but-not-stripped
    title: |
      qoder `-r <id>` is accepted by extractExplicitResume but never stripped from persisted or fresh args
    detail: |
      persistedConfigArgs strips only --resume (both forms) and sessionwatch.StripResumeArgs only the space form; -r survives. Measured: relaunch composes `-r abc --resume sid2`, and FreshAgentArgs then ValidateFreshAgentArgs fails with `qoder argument "-r" selects an existing conversation`, breaking Alt+n and continuation. The claimed test only asserts --resume, which passes without qoder code. Enumerate the resume forms against every strip and validate site (or drop -r as claude does) and add a failing-first round-trip test.
  - id: new
    severity: Important
    family: agent-dispatch-registration-gap
    title: |
      M1 fail-closed intermediate is asserted but untested; no registry parity test, and `pair session-inventory` now emits a permanent qoder schema_near_miss
    detail: |
      The diff adds no qoder row to wrapcmd, sessioninventory, couchcore or couchtty tests, so the "verified green" claim cannot fail. With qoder in runcli's supportedAgents and no scanner, `pair session-inventory` with no --agent emits a schema_near_miss warning for qoder on every run (measured, exit 0). Add a parity test over launcher.AgentInventory() with a named known-gap entry for qoder that M2 must delete, and pin the intermediate diagnostic.
  - id: new
    severity: Minor
    family: hand-restated-registry
    title: |
      Agent set restated by hand at five sites (usage string x3, validAgent, runcli supportedAgents) plus golden; atlas says "join both"
    detail: |
      Derive the usage line and validAgent from one list so the next harness touches one row, and correct the atlas checklist item 1, which under-counts the sessioninventory sites.
  - id: new
    severity: Minor
    family: qoder-branch-copies-claude
    title: |
      qoder extractExplicitResume branch and short-flag cluster loop are near-copies of claude's; --resume followed by a flag returns the flag as the id
    detail: |
      Parameterize the per-agent flag names and cluster value letters. Qoder's --resume [id] is optional-valued, so add a HasPrefix("-") guard in the qoder branch (claude has the same latent bug).
  - id: new
    severity: Minor
    family: unreachable-branch-unbacked-claim
    title: |
      qoder --tools variadic branch is unreachable because --tools is absent from freshValueOption("qoder")
    detail: |
      The commit message claims "--tools is variadic" and the test row does not exercise it. Add --tools to the qoder value list or delete the branch.
  - id: new
    severity: Minor
    family: plan-prose-restates-diff
    title: |
      Plan Task 6 still lists validAgent and runcli.go edits that M1 already did; Task 2 stale on value flags
    detail: |
      Append a plan Revisions entry moving those rows into Task 1 and recording the help-derived value-flag set.
```

---

## Re-review — 2026-09-21T09:40:44-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 08e9ec027c7a55bb1ae6d5054bbdeb0b61a82889..d997057740eb9d18c2ccc12fb90d5cdda5f3bfc6 |
| command | sdlc milestone-close --issue 300 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-21T09:40:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The Critical (BR-8) and both Important findings (BR-9, BR-10) from the prior round are fixed, and I confirmed each fix with a regression test that fails without it. In a scratch copy of `d9970577` I reverted the per-agent gating in `freshVariadicOption` and the `-r` strip in `persistedConfigArgs`. `TestValidateFreshAgentArgs`, `TestFreshVariadicOptionIsPerAgent`, `TestQoderShortResumeRoundTrip` and `TestQoderExplicitResumeAndPersistedArgs` all went red with the reverts. `sessioninventory` and `sessionwatch` pass at head. The one `launcher` failure is `TestCreateLayoutWrapperPreservesAgentCommand`, which fails with "operation not permitted" from the sandbox, not from this diff. Two new Minors remain. Both are class-level residues of families already in play, so I state the rule for each rather than fixing another instance. Neither blocks the gate.

## Strengths
- `freshAgentSpecs` and `explicitResumeForms` (`fresh_args.go:14-50`, `createlogic.go:53-60`) turn the qoder-copies-claude branches into per-agent tables. I checked the refactor against the old claude, agy and muse behaviour and found no drift beyond the deliberate valueless-`--resume` guard.
- `freshVariadicOption(agent, flag)` is strictly per-agent, so the codex `--add-dir X resume ID` hole is closed. `TestFreshVariadicOptionIsPerAgent` pins the whole agent × flag grid, not just the one row the finding named.
- `TestAgentInventoryParityWithSessionTables` (`agent_parity_test.go`) ranges `AgentInventory()` over three session-side tables. Its `sessionInventoryKnownGaps` entry forces M2 to delete it: the test fails when the gap closes or changes shape. The `qoder known gap` CLI-matrix row pins the intermediate `schema_near_miss` diagnostic.
- I checked `qodercli --help`. The value-flag set is right (`-w` is `--cwd <dir>`, `--worktree [name]` is long-only optional, `-d` and `-p` are bools, `--tools <tools...>` is variadic, `-r [id]` is optional-valued). `-r` does not collide with any flag in codex, agy or muse, so the agent-agnostic `-r` strip in `persistedConfigArgs` is harmless.
- `validAgent` and the usage line now derive from one `supportedAgents` list. An unknown agent in `harnessTTYProfiles` fails closed via the `ok` lookup, so M1's qoder-in-registry intermediate cannot panic the wrap path.

## Critical findings
None.

## Important findings
None. BR-9 and BR-10 are disposed `addressed` below.

## Minor findings
- Both new findings are in the findings block below: the resume-form rule (ARCH-DRY, ARCH-PURPOSE) and the parity-test coverage gap (ARCH-PURPOSE).
- `fresh_launch_test.go`: no blank line between `TestValidateFreshAgentArgs` and `TestFreshVariadicOptionIsPerAgent`.
- The `--tools a b -- hi` row cannot fail without qoder's variadic branch, because qoder has no positional-command check. The branch is inert in the validator, and only the direct helper table pins it.
- The session-inventory usage line changed order from `claude|codex|agy|muse` to alphabetical. The golden is updated and I found no doc restating it.

## Test coverage notes
- The BR-8 and BR-9 regression rows are real: they fail without the fixes, as measured above.
- The parity test probes `ScannerForAgent`, CLI acceptance and `sessionwatch.SupportsAgent`. It does not probe `sessionledger.isSupportedAgent`, which its own gap message names.
- `TestQoderShortResumeRoundTrip` hand-lists three forms. It does not range over `explicitResumeForms`.

## Architectural notes
- ARCH-PURE: pass. The new tables and validators are pure, and the tests need no IO.
- ARCH-MOCK: pass for M1. It uses `sessioninventorytest.NewFakeRuntime` behind the existing seam, and a qoder live conformance check is due in M2 (`TestLiveNativeSessionShapeConformance`).
- ARCH-CONSTRAINTS: pass. Argv transforms only, no runtime envelope change.
- ARCH-SECURE: pass. NUL checks are preserved, the forbidden-selector set matches `qodercli --help`, and the argv boundary is unchanged.
- ARCH-ORDER: pass. The change holds no state between events.
- ARCH-FUNERAL: pass. It creates nothing durable, and the parity known-gap entry has an enforced deletion path.
- ARCH-DRY: flag, Minor. It is the resume-form finding below.
- ARCH-PURPOSE: flag, Minor. It covers the resume-form class sweep and the session-side dispatch coverage below.
- Before M2 starts, decide whether `SupportsAgent` and `isSupportedAgent` derive from `sessioninventory`'s list or are probed by the parity test.

## Plan revision recommendations
The plan's `## Revisions` entry already covers BR-1 to BR-14. When M2 lands, add a line recording that the `sessionInventoryKnownGaps["qoder"]` entry and the `qoder known gap` golden row are deleted together.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      validAgent and runcli supportedAgents landed with the registry commit (model.go, runcli.go), and TestAgentInventoryParityWithSessionTables ranges AgentInventory() over the per-agent tables.
  - id: BR-2
    disposition: addressed
    note: |
      Plan Revisions records context-meter usage as a stated non-goal and puts the create-path shouldMintClaudeSessionID twin in Task 12 scope; no code change is owed at M1.
  - id: BR-3
    disposition: addressed
    note: |
      Plan Revisions folds a neutral-cwd capture and account-identity scrub policy, and says to satisfy assertFixtureIsMachineNeutral by its failure message; this is Task 9 scope and nothing in M1 contradicts it.
  - id: BR-4
    disposition: withdrawn
    note: |
      Compressing approved task bodies conflicts with the append-only Revisions rule in the constitution (section 1); the Revisions entry enumerates the divergences instead.
  - id: BR-5
    disposition: addressed
    note: |
      Revisions names TestLiveNativeSessionShapeConformance with PAIR_LIVE_NATIVE_SESSIONS=1; I confirmed both in conformance_live_test.go:9-11.
  - id: BR-6
    disposition: addressed
    note: |
      Revisions states that M5's boundary is the final sdlc close, with no separate milestone-close.
  - id: BR-7
    disposition: addressed
    note: |
      Revisions lists the non-goals: cloud sessions, subagent resume, context meter, progress-OSC authority, per-tool allowlist. None contradicts the issue Spec or Done-when.
  - id: BR-8
    disposition: addressed
    note: |
      freshVariadicOption is now per-agent with a default of false. In a scratch revert, TestValidateFreshAgentArgs (codex --add-dir rows) and TestFreshVariadicOptionIsPerAgent both fail.
  - id: BR-9
    disposition: addressed
    note: |
      persistedConfigArgs and sessionwatch.StripResumeArgs now strip -r and --resume=. Reverting the -r strip turns TestQoderShortResumeRoundTrip and TestQoderExplicitResumeAndPersistedArgs red. The residual class gap is raised as a Minor below.
  - id: BR-10
    disposition: addressed
    note: |
      The parity test carries a named qoder known-gap entry that M2 must delete, and the qoder known gap row in the CLI matrix pins the schema_near_miss diagnostic. Partial probe coverage is raised as a Minor below.
  - id: BR-11
    disposition: addressed
    note: |
      validAgent and the usage line derive from sessioninventory supportedAgents, and atlas section 0 and checklist item 1 are corrected. The two-list launcher-versus-session split is bridged by the parity test.
  - id: BR-12
    disposition: addressed
    note: |
      Extractor and cluster scan are table-parameterized (explicitResumeForms, freshAgentSpecs), and a HasPrefix("-") guard is pinned for claude and qoder valueless --resume. The strip-site counterpart is in the new Minor.
  - id: BR-13
    disposition: addressed
    note: |
      --tools is now in the qoder value-flag list, so the branch is reachable and the helper table pins it. The validator row itself is behaviourally inert, as noted under Minor.
  - id: BR-14
    disposition: addressed
    note: |
      Revisions moves the validAgent and runcli rows into Task 1 and records the help-derived value-flag set, which matches qodercli --help.
findings:
  - id: new
    severity: Minor
    family: resume-form-recognized-but-not-stripped
    title: |
      Resume-form set is hand-restated at four sites; glued `-r<id>` and valueless `--resume` still diverge between extract, strip and validate
    detail: |
      Rule for the whole family: extractExplicitResume, persistedConfigArgs, sessionwatch.StripResumeArgs and the forbiddenFlags and forbiddenShort in freshAgentSpecs must all derive from ONE per-agent form table (explicitResumeForms). The round-trip test must then range over that table instead of three hand-listed rows. Two instances remain today. First, qoder `-rabc` is rejected by the validator's cluster scan but is neither extracted nor stripped, so `pair qoder -rabc` persists it, then Alt+n composes `-rabc --resume sid2` and fails validation, the same failure BR-9 measured for `-r abc`. Claude has the same latent hole. Second, both strip sites drop the token after a valueless `--resume` or `-r` unconditionally, so `--resume --model m` leaves an orphan `m` prompt, although the extractor now guards this case. The two strip implementations (launcher and sessionwatch) are also near-duplicates that were each extended by hand.
  - id: new
    severity: Minor
    family: agent-dispatch-registration-gap
    title: |
      Parity test covers 3 of the session-side agent dispatch sites; sessionledger.isSupportedAgent is named in the gap message but never probed
    detail: |
      This is the 4th finding in this family, so I state the rule instead of fixing another site: every per-agent dispatch reachable from AgentInventory() must either derive from one exported list or be probed by the parity test, and a known-gap message must not name a site the test does not probe. Today the test probes ScannerForAgent, CLI acceptance and sessionwatch.SupportsAgent. Its gap text also names sessionledger, whose `isSupportedAgent` (record.go:480) is a second hand copy of `claude|codex|agy|muse`. An M2 that flips scanner and watcher support but misses the ledger ends with a green parity test and ledger-rejected qoder records. Unprobed dispatch sites include NormalizeNativeEvent, ProviderContractFor, target.go:211, the incremental_inventory switches and the runtime_os native roots. Prefer deriving SupportsAgent and isSupportedAgent from one exported predicate, or add a ledger probe to the parity test.
```
