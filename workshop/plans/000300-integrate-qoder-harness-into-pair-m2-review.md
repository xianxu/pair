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
