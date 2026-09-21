# Boundary Review — pair#300 (milestone M2)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 367610e7e4ddb3c17a12ec7573c162f0a71dae48..fa89157c0b2415a2718bbdb92b0c26ba174f9839 |
| command | sdlc milestone-close --issue 300 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-21T10:28:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M2 wiring is broadly right and works against real data. A qoder-only live conformance run against the real `~/.qoder` (in a scratch copy) reported `ok`: 10 nodes, 7 roots, no diagnostics. The claude-family extraction is neat and the qoder scanner is a thin parameterized producer of it. Two things block:

- **Critical:** commit `daeb781b` (the BR-15 rule-level fix) introduced a regression. A differential run against base `367610e7` shows the fresh-launch validator no longer rejects resume selectors inside short-flag clusters such as `-pr sid`.
- **Important:** several qoder dispatches are wired but pinned by no test. Deleting them turns nothing red, and the plan's own BR-16 rule was only partly carried out.

The fixes are small and mechanical.

## 1. Strengths
- `scan_claude.go:54-110`: `scanClaudeFamily` / `validateClaudeFamilyDelta` mean `ScanQoder` and `ValidateQoderDelta` are one-liners. The scan and incremental paths share the one record transition (ARCH-DRY, ARCH-PURE pass).
- The `runtime-config`/`active-leaf` epoch-millis quirk is handled and pinned end to end. `TestScanQoderV1` asserts the root chronology equals the runtime-config instant, and the atlas records the lesson in the right place.
- `resumeform` collapses four hand-kept spelling tables into one, with per-agent `Selector` semantics. `TestSelectorIsPerAgent` guards the BR-8 leak, and the table round-trip tests are the right shape (though see the Critical finding).
- Sessionwatch and sessionledger membership flip together with the scanner. The launcher parity test's known-gap map is emptied, and it now also probes the ledger.
- The golden CLI matrix row is honestly renamed ("no native storage") rather than left as a stale qoder gap row.

## 2. Critical findings
See the findings block: `resume-form-recognized-but-not-stripped`, `fresh_args.go:26`.

## 3. Important findings
See the findings block: `agent-dispatch-registration-gap` (5th in its family), `untrusted-input-parsed-without-bounds` (a new family), and `refactor-changes-sibling-agent-behavior` (2nd in its family).

## 4. Minor findings
See the findings block: the `file-history-snapshot` near-miss and comment overstatement, agent-agnostic `Strip` for glued `-r<x>`, exported mutable `Forms`, and unlogged M2 evidence in the plan.

## 5. Test coverage notes
- **Mutation-verified gaps** (deleting the qoder case turns no test red): `ProviderContractFor` and `AdvanceTargetValidation`.
- **Kills that worked:**
  - `artifactScannerShape` and `runtime_os` roots are killed by `TestOSRuntimeAgentSessionExistsFindsQoderTranscript`.
  - `ScannerForAgent` is killed by the parity test.
  - Reverting the numeric-timestamp type is killed by `TestScanQoderV1`.
- **Missing qoder rows in per-agent tables:** `TestProviderContractFor`, `TestAppendOnlyProviderConformance` (adding a qoder row passes in scratch), `provider_live_fake_test.go`, `native_large_record_test`, `scan_fuzz_test` and `query_test`.
- **Missing negative tests:**
  - There is no cluster-position resume test (`-p<r>`).
  - There is no out-of-range timestamp test.
  - There is no claude-side test pinning that numeric timestamps or the four ignored record types are still rejected.
- **Environment noise, not M2:**
  - `TestCreateLayoutWrapperPreservesAgentCommand` fails under the sandbox (`operation not permitted`).
  - The whole `TestLiveNativeSessionShapeConformance` fails on this machine because of agy schema drift, which predates this window.

## 6. Architectural notes
- **ARCH-DRY:** flag, covered by `agent-dispatch-registration-gap`. Four hand-kept lists of the same five agent names still exist (launcher, sessioninventory, sessionwatch, sessionledger). The parity test covers only some of the dispatches they drive.
- **ARCH-PURE:** pass. The scanner transition is pure; IO stays behind `Runtime`.
- **ARCH-PURPOSE:** flag. The shadow-sweep of per-agent dispatches is incomplete (the 5th agent-dispatch finding), and BR-15 was closed at the instance level (`-r<id>`) but not the grammar level (clusters).
- **ARCH-MOCK:** flag. The stateful append fake and the live-fake comparison do not cover qoder. Qoder is only in the RunConformance list, which passes live.
- **ARCH-CONSTRAINTS:** pass. Scans are unchanged whole-file reads.
- **ARCH-SECURE:** flag. The epoch-millis path is a new, less-validated input parser than the ISO path it sits beside.
- **ARCH-ORDER:** flag, minor. The two `AdvanceTargetValidation`/`ValidateTargetWork` switches turn an unhandled agent into a silent "advanced with no effect" (uncertainty collapsed into success). They need fail-closed defaults.
- **ARCH-FUNERAL:** pass. Nothing durable is created; only fixtures were added.
- **For M3 onward:** stop hand-listing per-agent tests. Range every per-agent test over `AgentInventory()` with fixtures at `testdata/native/<agent>/v1`.

## 7. Plan revision recommendations
Add a `## Revisions` entry that:
- records that the claude-family core carries qoder-only admissions (epoch-millis timestamps and four ignored event types) as parameters, and that claude's behavior is pinned unchanged;
- corrects Task 7's `events_test.go` to `event_test.go`;
- records the Task 7 Step 5 and Task 8 Step 3 evidence, namely the qoder live conformance: `ok`, 10 nodes, 7 roots, no diagnostics;
- notes that the whole-suite live conformance fails on agy drift, unrelated to M2;
- states the BR-16 rule as delivered: every per-agent dispatch is derived or probed, with fail-closed `default:` arms.

```findings
findings:
  - id: new
    severity: Critical
    family: resume-form-recognized-but-not-stripped
    title: |
      Fresh-arg validator no longer rejects short-flag clusters like -pr sid (regression from base)
    detail: |
      This is the 3rd finding in family resume-form-recognized-but-not-stripped. Removing "r" from forbiddenShort (fresh_args.go:26, :34; claude and qoder) in favour of resumeform.Selector loses rejection of any cluster where r is not the first letter. A differential run at base 367610e7 vs head shows ValidateFreshAgentArgs(claude|qoder, ["-pr","sid"]), ["-vr","sid"] and ["-hr","x"] error at base and return nil at head. A "fresh" launch given -pr <sid> reaches the CLI and resumes an existing conversation, which is what the guard exists to prevent. Selector only matches an exact -r token or a token starting with -r; the Form comment's claim that a cluster containing the selector "normalizes to the glued reading" is false for r in a non-first position. TestResumeFormTableRoundTrip and TestResumeSpellingDivergences only test -r<id> and -r. Rule covering the class: the set of tokens a fresh launch refuses must be a superset of what any earlier site refused, and cluster letters must derive from resumeform.Forms, not a hand-kept forbiddenShort string. Fix: add a resumeform helper (e.g. ClusterLetters(agent) from the single-letter Glued spellings) and have forbidsCluster consult it alongside 'c'. Add a table-ranged test that puts each Glued letter after a bool short letter (-p<letter>) and requires rejection, and keep the base rows -cr, -rp, -pc.
  - id: new
    severity: Important
    family: agent-dispatch-registration-gap
    title: |
      Qoder wiring at ProviderContractFor and AdvanceTargetValidation is pinned by no test; the BR-16 rule was only partly carried out
    detail: |
      This is the 5th finding in family agent-dispatch-registration-gap. Do not fix only this instance. Mutation checks on a scratch copy of head show that deleting the AgentQoder case from ProviderContractFor turns no test red, and neither does deleting it from AdvanceTargetValidation. AdvanceTargetValidation (switch at incremental_inventory.go:198, no default) then returns the prior state unchanged with the frame offset advanced, so the watcher consumes qoder bytes without applying them and emits no diagnostic. ValidateTargetWork (:138) silently skips. The per-agent test tables still list only claude/codex/muse: TestProviderContractFor, TestAppendOnlyProviderConformance (a qoder row passes when added in scratch), provider_live_fake_test.go (its validateLiveRecords default returns ErrArtifactChanged for qoder), native_large_record_test, scan_fuzz_test and query_test. The plan Revisions (BR-16) bound M2 to the rule that every per-agent dispatch reachable from AgentInventory() is derived from one list or probed by the parity test, but only sessionledger was added to the probe. Not probed: ProviderContractFor, observationNativeID, artifactScannerShape, ValidateTargetWork/AdvanceTargetValidation, NormalizeNativeEvent and the runtime roots. Rule: one per-agent capability table, or a parity test ranging AgentInventory() through the testdata/native/<agent>/v1 fixtures and exercising every dispatch, plus fail-closed default arms in the two switches. Range the per-agent tables above from the same list.
  - id: new
    severity: Important
    family: untrusted-input-parsed-without-bounds
    title: |
      Epoch-millis timestamps are accepted with no range check and fabricate chronology
    detail: |
      claudeFamilyTime.nativeTime (scan_claude.go:46) turns any int64 into a metadata-sourced instant via time.UnixMilli, while the ISO path is implicitly bounded to years 0000-9999 by RFC3339 parsing. A qoder record with "timestamp":9223372036854775807 is accepted with no diagnostic. `pair session-inventory --agent qoder --json` then printed created_at "292278994-08-17T07:12:55.807Z", which is not RFC3339, and the poisoned session sorts as the newest. activity.go:48 copies the value into a time.Time-typed json field, and json.Marshal of that year fails with "year outside of range [0,9999]". A record of -62135596800000 makes ValidateScannerState fail on the zero time. This transcript is input this program did not produce (ARCH-SECURE). Fix: parse the millis into a typed instant at the boundary and reject values outside a sane window (e.g. 2000-01-01 to year 9999), returning an error so the record disputes visibly like a malformed ISO string. Add a scan test row for it.
  - id: new
    severity: Important
    family: refactor-changes-sibling-agent-behavior
    title: |
      Qoder-only grammar admissions silently widened claude's scanner and event grammar
    detail: |
      This is the 2nd finding in family refactor-changes-sibling-agent-behavior. Task 5 promised the refactor is "invisible to the claude suite", but two claude admissions changed. First, claudeRecord.Timestamp is now claudeFamilyTime for every agent, so ValidateClaudeDelta on {"type":"user","sessionId":"<id>","timestamp":1787907630000} returned disputed=false with a 2026 chronology (probe), where the old string field made it a malformed-record dispute. Second, normalizeClaudeEvent (event.go:113) is shared, so claude records of type workspace-directories, runtime-config, worktree-state and active-leaf are now EventIgnored instead of EventNearMiss. Both weaken claude's near-miss drift detector, and no claude-side negative test pins them. Rule: every grammar admission a new family member needs is a parameter of the shared core (e.g. a family spec carrying acceptsEpochMillis and extraIgnoredTypes), defaulting to the existing agent's prior behavior, with a claude negative row per admission. Do not widen the shared type.
  - id: new
    severity: Minor
    family: unbacked-existing-behavior-claim
    title: |
      event.go comment overstates ignore-set completeness; file-history-snapshot still near-misses on real qoder transcripts
    detail: |
      Tally over the 10 real ~/.qoder transcripts: 1834 accepted, 2818 ignored, 53 near-miss, all 53 of type file-history-snapshot. Claude shows the same near-miss baseline for many types (mode, permission-mode, file-history-*), so this is consistent noise, not a blocker. Add the type with evidence or soften the comment "Without them the whole qoder stream is near-miss".
  - id: new
    severity: Minor
    family: resume-form-recognized-but-not-stripped
    title: |
      resumeform.Strip is agent-agnostic for glued -r<x> across all agents
    detail: |
      Strip drops any token starting with -r plus a non-empty, non-'=' remainder for every agent, including codex, muse and agy (single-dash long flags). A legitimate -r* flag of another agent would be lost from persisted args. Currently latent; the old strip only removed exact -r and -r=. Consider passing the agent so only its own spellings are removed.
  - id: new
    severity: Minor
    family: hand-restated-registry
    title: |
      resumeform.Forms is an exported mutable package map
    detail: |
      Four packages read one writable exported map that is the single source of truth. Keep it unexported behind Selector/Extract/Strip and expose an accessor for the round-trip tests.
  - id: new
    severity: Minor
    family: plan-prose-restates-diff
    title: |
      Plan has no M2 Revisions entry; live-conformance and manual steps are unlogged; Task 7 names the wrong test file
    detail: |
      The only in-window Revisions entry covers the M1 advisories. Task 7 names events_test.go (actual file is event_test.go), and Task 7 Step 5 and Task 8 Step 3 have no logged evidence. I ran the qoder-only conformance against real ~/.qoder in a scratch copy: ok, 10 nodes, 7 roots, no diagnostics. Whole-suite TestLiveNativeSessionShapeConformance fails on this machine because of agy drift (pre-existing, unrelated to M2). Record the evidence and the claude-family admissions in a Revisions entry.
```

---

## Re-review — 2026-09-21T10:59:59-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 77df4b5e8aa48d378ecba8255edd1f33704b2db9..42deeee430af17dcc1c226298492d48485872d99 |
| command | sdlc milestone-close --issue 300 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-21T10:59:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

**Summary.** The pinned window `77df4b5e..42deeee4` is a single commit that adds one line to the issue Log, and `workshop/issues/` is excluded from the diff. The `stat` and `names` recipes therefore show nothing. The open findings concern the M2 code, so I inspected head 42deeee4 via `git archive` scratch copies and used mutation runs instead of trusting the Log. Six of the eight open findings are addressed at head, each with a regression test that goes red when the fix is reverted: BR-17, 19, 20, 21, 22 and 23. Two are not. BR-18 (the qoder dispatch class) has no fix in any commit. The fix is in the dirty working tree only, and even there mutation runs show two named dispatch sites and the runtime roots still unpinned. BR-24 (plan Revisions) is absent at head; the working-tree draft misnames the code and overclaims. The Log line committed in this window (line 184) says both are "addressed". No commit contains that work, and the working-tree version only partly delivers it.

## 1. Strengths
- **BR-17:** cluster letters now derive from the Glued spellings (`resumeform/resumeform.go` `ShortLetters`, consulted at `launcher/fresh_args.go:55`). `TestFreshValidatorRefusesResumeLettersInClusters` (`resume_forms_test.go:38`) is ranged over `resumeform.Forms()`. It also keeps the base rows `-cr`, `-rp` and `-pc`. With the `ShortLetters` term removed, it fails on `-pr`, `-vr` and `-hr` for both claude and qoder.
- **BR-19:** the epoch-millis window (`scan_claude.go:39-42, 82-92`) is a typed, bounded value that disputes visibly. `TestQoderMillisTimestampBounds` covers `MaxInt64` and `-62135596800000`, and goes red without the bound.
- **BR-20:** the qoder admissions are now parameters. `claudeFamilySpec.acceptsMillis` and per-agent `claudeFamilyNoiseTypes` default to claude's prior behavior. Each has a claude negative row (`TestIncrementalClaudeRejectsNumericTimestamp`, `event_test.go:50-53`), and each mutation goes red.
- **BR-22/23:** `Strip(agent, …)` is strictly per-agent, and every production caller threads the agent (`agentargs.go:222`, `sessionwatch.go:52`). The `forms` map is unexported behind a copying `Forms()` accessor.
- **BR-21:** the ignore-set comment is now tied to a dated measurement.

## 2. Critical findings
None.

## 3. Important findings
- **BR-18 (family `agent-dispatch-registration-gap`, 6th touch).** Head has no fix at all. The working tree adds qoder rows to six tables, a `dispatch_parity_test.go` and `default:` arms, but none of it is committed. I overlaid the working-tree `sessioninventory/*.go` onto a head export and deleted each qoder arm in turn:

  | Mutation (delete qoder arm) | Result |
  |---|---|
  | `ProviderContractFor` | red |
  | `artifactScannerShape` | red |
  | `observationNativeID` | red |
  | `NormalizeNativeEvent` | red |
  | `AdvanceTargetValidation` | **GREEN** |
  | `ValidateTargetWork` | **GREEN** |
  | `runtime_os.go` qoder native root | **GREEN** |

  - **What is still open:** the Advance/ValidateTargetWork/roots items are exactly the sites the finding named and BR-16 bound M2 to probe.
  - **The new parity test overclaims:** `TestEveryAgentDispatchParity`'s doc comment says deleting a case from `ValidateTargetWork` or `AdvanceTargetValidation` turns a sub-assertion red. The test never calls either function. It ranges a hard-coded four-agent list, not `AgentInventory()`. Its `validateAgentDelta` is a hand-written switch that restates the production one.
  - **The new `ValidateTargetWork` default arm is a no-op:** it sets `err`, and the next `if err != nil … continue` swallows it with no diagnostic. That is identical to the old silent skip, so the "fail-closed" claim holds only for `AdvanceTargetValidation`, which returns a generic `ErrArtifactChanged`.
  - **The rule that covers the class:**
    1. Replace the two duplicated four-arm delta switches (`incremental_inventory.go:138-148` and `:198-209`, ARCH-DRY) with one `deltaValidatorFor(agent)` per-agent capability entry that fails closed with a diagnostic.
    2. Export the single agent list (`runcli.go` `supportedAgents`) and range the parity test from it, failing on an agent with no row.
    3. Have the parity probe call `ValidateTargetWork` and `AdvanceTargetValidation` (append a suffix record, then advance) and assert each agent's native roots are non-empty.
  - **Verification:** re-run the seven mutations above and require every one to go red.

## 4. Minor findings
- **BR-24 (open):** the plan has no M2 `## Revisions` entry at head. The working-tree draft (`plan.md:815-828`) names fields that don't exist. It cites `acceptsEpochMillis` and `extraIgnoredTypes`, but the code has `acceptsMillis` and the function `claudeFamilyNoiseTypes`. It states "every per-agent dispatch … is probed" and "fail-closed default arms … instead of silently skipping", which the mutations above disprove. It has a typo, "untrusted-inputarsed". Commit it only after those statements are true.
- The Log line at head (line 184) records BR-18 and BR-24 as delivered when no commit contains them (raised as a new finding below).
- `event.go` counts ("10 files, `active-leaf` ×1751") are a dated snapshot. The corpus is now 17 files with 2095 `active-leaf` records, which is fine because the comment carries a date.

## 5. Test coverage notes
Coverage for BR-17, 19, 20, 22 and 23 is real, table-ranged where the class is enumerable, and verified by mutation. The gaps are the three green mutations under BR-18.

Two package tests, `TestEveryCoreConceptIntroductionMatchesDeclarations` and `TestEveryIssueOwnedTypeHasConceptDisposition`, fail in my scratch export only because it has no git history. The launcher's `TestCreateLayoutWrapperPreservesAgentCommand` fails there because the runtime assets aren't generated. Neither indicates a regression.

## 6. Architecture (each principle worked)
- **ARCH-DRY:** flag. The duplicated per-agent delta switches and the test-side `validateAgentDelta` are the class BR-18 belongs to.
- **ARCH-PURE:** pass. Validators are pure over framed records, with IO behind the `Runtime` seam.
- **ARCH-PURPOSE:** flag. The BR-18 sweep fixed the enumerable siblings that turned red but left `AdvanceTargetValidation`, `ValidateTargetWork` and the roots (shadow-sweep incomplete). BR-17/19/20 were fixed at rule level.
- **ARCH-MOCK:** pass. Fakes are used and the real `~/.qoder` live run is recorded in the Log. The qoder-only conformance is manual, so there is no scheduled cadence yet.
- **ARCH-CONSTRAINTS:** pass. Nothing new on a hot path.
- **ARCH-SECURE:** pass. The millis input is bounded and typed at the boundary, and the failure path disputes visibly.
- **ARCH-ORDER:** pass. No new state is held between events. The claude-family transition function is reused with per-agent parameters.
- **ARCH-FUNERAL:** pass. Nothing new is created durably.

## 7. Plan revision recommendations
Add the M2 `## Revisions` entry, corrected to name `acceptsMillis` and `claudeFamilyNoiseTypes`. Record BR-18's delivered rule only once the probe for the two incremental sites and the roots exists and is verified by mutation.

Re-pin the review window: base 77df4b5e excludes all M2 code, and the previous milestone close is 367610e7.

```findings
dispose:
  - id: BR-17
    disposition: addressed
    note: |
      launcher/fresh_args.go:55 consults resumeform.ShortLetters; TestFreshValidatorRefusesResumeLettersInClusters (table-ranged over Forms(), base rows kept) goes red on -pr/-vr/-hr for claude and qoder when the derivation is removed (mutation run).
  - id: BR-18
    disposition: not-addressed
    note: |
      No fix is committed at head. In the uncommitted tree, deleting the qoder arm from AdvanceTargetValidation, from ValidateTargetWork, or from the runtime_os native roots turns NO test red. TestEveryAgentDispatchParity never calls ValidateTargetWork/AdvanceTargetValidation despite its doc comment, and it ranges a hard-coded list. The ValidateTargetWork default arm sets err but the loop's `if err != nil ... continue` drops it with no diagnostic, so it is still a silent skip. Rule: one per-agent delta-validator table used by both switches (fail closed with a diagnostic); export the single agent list and range the parity test from it; probe ValidateTargetWork, AdvanceTargetValidation and NativeRoots per agent; then re-run all seven arm-deletion mutations and require red.
  - id: BR-19
    disposition: addressed
    note: |
      scan_claude.go:39-42,82-92 bounds millis to 2000-01-01..9999-12-31 and disputes visibly; TestQoderMillisTimestampBounds (MaxInt64 and -62135596800000 rows) goes red when the bound is removed.
  - id: BR-20
    disposition: addressed
    note: |
      acceptsMillis lives on claudeFamilySpec and claudeFamilyNoiseTypes is per-agent; claude negative rows (TestIncrementalClaudeRejectsNumericTimestamp, event_test.go:50-53) go red under both mutations (claude accepting millis, claude gaining the qoder noise set).
  - id: BR-21
    disposition: addressed
    note: |
      file-history-snapshot joins qoder's ignore set, and the event.go comment now cites a dated measurement instead of the overclaim.
  - id: BR-22
    disposition: addressed
    note: |
      resumeform.Strip(agent, args) uses only the agent's own form; all production callers thread the agent (agentargs.go:222, sessionwatch.go:52); TestStripIsPerAgent pins codex/agy/claude preservation.
  - id: BR-23
    disposition: addressed
    note: |
      forms is unexported behind a copying Forms() accessor; tests range the accessor.
  - id: BR-24
    disposition: not-addressed
    note: |
      Head's plan has no M2 Revisions entry. The uncommitted draft (plan.md:815-828) names non-existent fields (acceptsEpochMillis, extraIgnoredTypes; code has acceptsMillis and the function claudeFamilyNoiseTypes), claims every dispatch is probed and the default arms fail closed (both disproved by the BR-18 mutations), and has a typo in a family slug. Correct it and commit it with the code it describes.
findings:
  - id: new
    severity: Minor
    family: unbacked-existing-behavior-claim
    title: |
      Issue Log line 184 (this window) records BR-18/BR-24 as delivered; no commit contains them and the working-tree version only partly delivers them
    detail: |
      The Log says qoder rows, fail-closed defaults and a parity probe for "every JSONL agent" landed; git shows the code/test/plan changes uncommitted (8 modified files plus untracked dispatch_parity_test.go). Rule: a Log/plan claim that a coverage or fail-closed property holds must be committed alongside the code and backed by an arm-deletion mutation that turns red. Correct the claim, or land the missing probes, before recording it.
```
