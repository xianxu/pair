# Boundary Review — pair#571 (whole-issue close)

| field | value |
|-------|-------|
| issue | 571 — Resolve Codex indirect resumes from faithful read-only configuration |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8792d2b7ee59e86a12bb8471441efe24d963f818..73a3a6a321de20044b092c8469be71b3a82b31a4 |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T16:39:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The admission flow, bounded reads, compatibility fallback, and documentation are well integrated. Focused normal and race tests pass. Two source-fidelity defects still allow the adapter to pin a different conversation from native Codex, blocking this boundary.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-config-override-fidelity
    title: |
      CLI table overrides merge keys that native Codex replaces
    detail: |
      cmd/internal/sessioninventory/codex_config_os.go:292 recursively merges each CLI override. Native apply_toml_override replaces the value at the final dotted-key segment, except for explicitly structured features. With API-only configuration and overrides features={worktrees=false} followed by features={shell_snapshot=false}, Pair retains worktrees=false while native restores the default true. A newer conversation in a corresponding linked checkout is therefore excluded by Pair, allowing it to pin an older current-checkout conversation. Implement native leaf-assignment semantics separately from config-layer merging, sweep other table-valued overrides, and add a regression asserting both projected scope and selected ID.
  - id: new
    severity: Critical
    kind: defect
    family: native-transcript-schema-fidelity
    title: |
      Exec selection trusts transcript records rejected by native decoding
    detail: |
      cmd/internal/sessioninventory/codex_latest.go:330-343 treats any JSON object containing type=turn_context and payload.cwd as authoritative. Native exec only uses successfully decoded RolloutLine/TurnContextItem records, whose required fields include approval_policy, sandbox_policy, and model. A newer transcript with DB cwd elsewhere and a schema-invalid turn_context naming the current cwd is selected by Pair, while native ignores that record and can select an older conversation. TestCodexLatestTUIAndExecHaveDifferentCWDSources currently asserts this invalid record is authoritative. Validate native record eligibility or retain compatibility when it cannot be established; include validateCodexLatestFirst in the same sweep because its reused inventory validator also accepts incomplete native session_meta records. Replace positive fixtures with native-valid records and add malformed-schema regressions.
```

1. **Strengths**

   - Configuration observation and pure projection are separated; credential reads and startup oracles are avoided.
   - Wrapper scope verification occurs before Claim, while the explicit ID prevents later arrivals from retargeting the launch.
   - Snapshot fences cover inspected higher candidates as well as the selected transcript.
   - README and indexed atlas updates cover the new support boundary and limitations.

2. **Critical findings**

   - **CLI override fidelity — ARCH-PURPOSE.** The faulty merge is at [codex_config_os.go:292](/Users/xianxu/Library/Caches/sdlc/review-checkouts/6c18a209f836610c/cmd/internal/sessioninventory/codex_config_os.go:292). Native final-segment assignment is explicit in the [matching override implementation](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/config/src/overrides.rs#L26-L61).
   - **Transcript eligibility — ARCH-SECURE / ARCH-MOCK / ARCH-PURPOSE.** The partial decoder is at [codex_latest.go:330](/Users/xianxu/Library/Caches/sdlc/review-checkouts/6c18a209f836610c/cmd/internal/sessioninventory/codex_latest.go:330). Compare native’s [parsed-record-only scan](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/lib.rs#L1826-L1844) with the [required turn-context fields](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/protocol/src/protocol.rs#L3324-L3347).

3. **Important findings**

   None separately; the missing regression coverage belongs to the Critical findings above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused selector/config/admission/create tests across sessioninventory, launcher, and wrapcmd, both normally and with `-race`. Additional Git/platform/native-inventory tests, artifactpath tests, slot-resource audits, and diff whitespace checks passed.

   The full suite and live native comparison were not rerun. Existing green tests do not establish native protocol fidelity: several positive transcript fixtures omit required native fields. No repository files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing parser, inventory roots, and admission machinery are reused.
   - **ARCH-PURE — pass:** projection is directly tested without IO; observation stays at integration boundaries.
   - **ARCH-PURPOSE — flag:** the two mismatches violate faithful native selection.
   - **ARCH-MOCK — flag:** portable fixtures exercise production seams, but incomplete transcript shapes weaken their behavioral model.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, pagination, and compatibility on exhaustion are implemented and tested.
   - **ARCH-SECURE — flag:** partial protocol decoding promotes malformed persisted input into selection authority.
   - **ARCH-ORDER — pass:** selection, reservation, scope verification, Claim, spawn, and publication retain explicit sequencing.
   - **ARCH-FUNERAL — pass:** scope metadata uses the existing bounded grant lifecycle; no new durable artifact family was found.

   Core-concept locations and classifications otherwise match the implementation.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - Native CLI leaf replacement versus inter-layer merging, with a sweep of table-valued overrides.
   - Native record eligibility for both session metadata and turn contexts, native-valid positive fixtures, and schema-invalid selection regressions.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5313b275-fbc3-463c-8549-9eb73e49f8d9","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6c18a209f836610c","reviewed_head":"73a3a6a321de20044b092c8469be71b3a82b31a4","slot":"pair:7","requested_at":"2026-10-15T23:35:36.173076Z"},"phase":"finished","pid":77446,"started_at":"2026-10-15T23:35:36.382206Z","finished_at":"2026-10-15T23:39:17.961252Z","session_id":"01a141eb-d423-7cd1-a193-69bff2fb0560","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.251102875s
head: "73a3a6a321de20044b092c8469be71b3a82b31a4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T17:37:49-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 571 — Resolve Codex indirect resumes from faithful read-only configuration |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99e11ecb8c72b7e5325a58692c1efd679e50710c..7611621eff5cedd7df3160d62a6fa8b8f93b326b |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:37:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed with meaningful regression tests. Two additional defects block shipping: worktree discovery can exclude a native-valid checkout and pin the wrong conversation, and the “read-only” selector can create a native SQLite sidecar. Focused normal/race tests and ownership audits pass.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Native leaf replacement is implemented separately from layer merging. The linked-checkout regression checks projected scope and selected ID; a temporary overlay restoring recursive table merging makes it fail.
  - id: BR-2
    disposition: addressed
    note: |
      Both session metadata and turn-context authority now require native eligibility. Positive fixtures contain required native fields. Removing the eligibility guards through a temporary overlay makes the malformed-schema regressions select the invalid newer ID and fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-worktree-scope-fidelity
    title: |
      Trust-specific alias rejection silently narrows native worktree selection
    detail: |
      cmd/internal/sessioninventory/codex_git_scope.go:63 rejects symlinked admin targets for both trust and worktree discovery, and appendRoot at line 148 silently excludes that checkout. Native worktree repository_identity canonicalizes these targets, although native trust discovery rejects them. With API-only configuration, an older current-checkout conversation and a newer linked-checkout conversation whose .git points through an admin alias, a private production-selector regression resolves the older ID instead of the newer native match. ARCH-PURPOSE / ARCH-SECURE: separate trust eligibility from worktree discovery semantics, and sweep current/primary/linked discovery so unsupported observations yield compatibility rather than a smaller authoritative scope.
  - id: new
    severity: Critical
    kind: defect
    family: native-read-only-observation
    title: |
      Read-only latest selection creates native SQLite shared-memory files
    detail: |
      cmd/internal/sessioninventory/codex_latest.go:103 queries the live native database through QueryNativeSelectorSQLite, whose sqlite3 -readonly invocation at native_selector_os.go:78 still creates WAL shared-memory sidecars. A private WAL fixture containing state_5.sqlite and state_5.sqlite-wal but no SHM returned a resolved ID and acquired a new 32768-byte state_5.sqlite-shm. This violates the issue's no-native-mutation contract. ARCH-PURPOSE / ARCH-MOCK / ARCH-FUNERAL: use a bounded, coherent private snapshot or another genuinely non-mutating observation mechanism, retain compatibility when safe observation is unavailable, and add WAL-backed no-mutation regression coverage.
```

1. **Strengths**

   - CLI overrides now distinguish native leaf assignment from recursive configuration-layer merging.
   - Rollout eligibility covers required fields, typed optional fields, and Unicode decoder differences.
   - Scope binding reuses the existing reservation/Claim flow and verifies the wrapper’s final environment before spawning.
   - README and indexed atlas documentation cover the new support boundary and compatibility behavior.

2. **Critical findings**

   - **Worktree scope:** [codex_git_scope.go:63](/Users/xianxu/Library/Caches/sdlc/review-checkouts/a5eabb22d0c795a8/cmd/internal/sessioninventory/codex_git_scope.go:63). Native [worktree discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/worktree.rs) and [trust discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/trust.rs) apply different alias rules. Preserve that distinction or return compatibility.
   - **Native mutation:** [codex_latest.go:103](/Users/xianxu/Library/Caches/sdlc/review-checkouts/a5eabb22d0c795a8/cmd/internal/sessioninventory/codex_latest.go:103). Move WAL observation behind a mechanism that cannot create or modify native sidecars.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused inventory, launcher, and wrapper tests passed normally and with `-race`.
   - Artifactpath, native inventory, Git/platform/eligibility tests, slot-resource audits, and whitespace checks passed.
   - Both prior fixes were independently shown necessary using temporary overlays.
   - Temporary production-path regressions reproduced both new defects. Repository files remain unchanged.
   - The complete suite and real-machine acceptance were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared parser, root ownership, and admission machinery.
   - **ARCH-PURE — pass:** pure projection/eligibility logic remains separate from observation and spawning.
   - **ARCH-PURPOSE — flag:** wrong-ID selection and native mutation violate the central contract.
   - **ARCH-MOCK — flag:** portable SQLite fixtures omit the WAL lifecycle behavior reproduced here.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, pagination, and exhaustion handling have focused coverage.
   - **ARCH-SECURE — flag:** unsupported Git provenance becomes authoritative exclusion.
   - **ARCH-ORDER — pass:** scoped Claim checks, transcript fences, and pinned-ID handoff preserve existing ordering safeguards.
   - **ARCH-FUNERAL — flag:** selection creates native SHM residue without an owned removal path.

   Core-concept locations and classifications otherwise match the implementation. The README/atlas update gate is satisfied.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - Separate native trust and worktree-discovery rules; uncertainty must not silently narrow selection scope.
   - Non-mutating WAL observation, snapshot consistency and cleanup, and regression coverage asserting unchanged native artifacts.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"96ffc194-c1b3-4a55-8f0e-235a5498f95c","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a5eabb22d0c795a8","reviewed_head":"7611621eff5cedd7df3160d62a6fa8b8f93b326b","slot":"pair:7","requested_at":"2026-10-16T00:33:14.646291Z"},"phase":"finished","pid":8625,"started_at":"2026-10-16T00:33:14.753069Z","finished_at":"2026-10-16T00:37:27.201852Z","session_id":"01a14220-98ec-7001-9b98-3951b8611aee","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.463680958s
head: "7611621eff5cedd7df3160d62a6fa8b8f93b326b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T18:46:59-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 571 — Resolve Codex indirect resumes from faithful read-only configuration |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c6fcc8d437ce52c708b0a6bdef09ce42b86917b..2084334f6e1ecef502f755df75507d0443373189 |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T18:46:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-3 and BR-4 are addressed with reachable regressions that fail when their fixes are removed. However, exec latest selection still admits source kinds that native Codex excludes, allowing Pair to pin a different conversation. README and atlas updates are present; the targeted normal, race and ownership checks passed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Leaf-replacement logic and its projection/selected-ID regressions remain present; the targeted configuration tests passed.
  - id: BR-2
    disposition: addressed
    note: |
      Native record eligibility checks remain reachable before selection authority; malformed metadata and turn-context regressions passed. The new finding concerns filtering valid records, not decoding invalid records.
  - id: BR-3
    disposition: addressed
    note: |
      Current/primary/linked discovery uncertainty now yields whole-projection compatibility. Restoring the pre-fix Git implementation through a temporary Go overlay made TestCodexLatestGitAdminAliasCannotSelectOlderConversation fail by selecting the older ID; pinned-head tests passed.
  - id: BR-4
    disposition: addressed
    note: |
      Production selection uses an owned private DB/WAL snapshot. Bypassing snapshot dispatch through temporary overlays made both TUI and exec WAL regressions fail with a newly created 32768-byte native SHM file; pinned-head non-mutation, cleanup and race tests passed.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-selector-filter-fidelity
    title: |
      Exec latest can pin Custom or Internal conversations excluded by native selection
    detail: |
      cmd/internal/sessioninventory/codex_latest.go:181-190 filters only by provider and never applies native source-kind filtering. In rust-v0.162.1, exec all_thread_source_kinds feeds app-server source_kind_matches, which excludes SessionSource::Custom and SessionSource::Internal. A temporary production-selector regression with an older eligible cli conversation and a newer matching conversation reproduced wrong-ID resolution for custom atlas, custom chatgpt, internal guardian and internal memory_consolidation sources. ARCH-PURPOSE / ARCH-SECURE: decode and filter persisted DB source values according to the native selection rule, or retain compatibility when their meaning is unproven. Sweep the source-kind mapping across both collectors and add selected-ID regressions; valid transcript decoding alone does not establish selector eligibility.
```

1. **Strengths**

   - Git discovery now distinguishes uncertainty from proven exclusion and propagates uncertainty to compatibility.
   - SQLite observation uses bounded private snapshots, hermetic invocation flags and cleanup that can refuse authority.
   - Scope binding reuses the existing reservation/Claim flow and revalidates the child’s configuration before Claim.
   - README and indexed atlas documentation describe the supported configuration and fallback boundaries.

2. **Critical findings**

   - [codex_latest.go:181](/Users/xianxu/Library/Caches/sdlc/review-checkouts/2d3f1ba50376b04a/cmd/internal/sessioninventory/codex_latest.go:181): apply native source-kind eligibility before accepting exec candidates. Native’s [exec kind enumeration](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/lib.rs#L1801) and [source matcher](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/app-server/src/filters.rs#L58) exclude Custom and Internal sources. All four private reproductions selected the newer excluded ID.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Targeted configuration, Git, latest-selection, snapshot, launcher and wrapper tests passed normally; focused race tests also passed. Artifactpath and Couch resource guards passed. Real-machine read-only configuration conformance passed with compatibility/cloud-unknown for both modes.

   Both prior fixes failed their removal controls. The new source-filter regression fails at pinned head. Temporary overlays were removed; the checkout remains unchanged. I did not rerun the full recorded suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared parser, native-root ownership and admission machinery are reused.
   - **ARCH-PURE — pass:** projection remains pure; filesystem and subprocess work stays behind runtime seams.
   - **ARCH-PURPOSE — flag:** source-filter drift breaks faithful native selection.
   - **ARCH-MOCK — pass:** private real files/SQLite and injected process/admission seams exercise production paths.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, metadata counts, output and cancellation have regression coverage.
   - **ARCH-SECURE — flag:** schema-valid persisted source data is accepted without proving selector eligibility.
   - **ARCH-ORDER — pass:** selection is pinned once; scope mismatch prevents Claim, and cleanup failure prevents authority.
   - **ARCH-FUNERAL — pass:** owned snapshots have cleanup paths tested across success, failure and cancellation.

   The Core concepts entries exist at their stated locations; no PURE/INTEGRATION contradiction was found.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the source eligibility rule separately from transcript schema validity. Enumerate every native source variant, its persisted representation and its eligibility for each collector; require selected-ID tests for excluded Custom/Internal candidates and compatibility for unproven representations.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1012d4e3-b66e-43ed-a040-b9b8fa5e21a1","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/2d3f1ba50376b04a","reviewed_head":"2084334f6e1ecef502f755df75507d0443373189","slot":"pair:7","requested_at":"2026-10-16T01:42:49.985484Z"},"phase":"finished","pid":13326,"started_at":"2026-10-16T01:42:50.079819Z","finished_at":"2026-10-16T01:46:52.36869Z","session_id":"01a14260-4ebf-73a3-9ab9-b47c486ecece","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.311152875s
head: "2084334f6e1ecef502f755df75507d0443373189"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T19:32:49-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 571 — Keep Codex indirect resumes native; route exact-ID transcript homes |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0169e962f95a932fd9884b20b2d5dd07b73022f1..1e3129bcbe93bb7aaf8d27b6b33785f5757ca7f3 |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T19:32:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The reduced native-pass-through design removes BR-5’s faulty selection mechanism, and focused normal/race tests pass. Two reproducible gaps block shipping: Codex home errors can disable unrelated Claude launches, and recovery discovery still scans the default home while recovery validation uses CODEX_HOME.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The configuration projection and override machinery are removed; indirect Codex invocations remain native.
  - id: BR-2
    disposition: addressed
    note: |
      The native-selector transcript eligibility machinery is removed; Pair no longer predicts indirect Codex targets.
  - id: BR-3
    disposition: addressed
    note: |
      Codex worktree-scope emulation is removed; indirect requests return unsupported before scope observation.
  - id: BR-4
    disposition: addressed
    note: |
      Codex SQLite collectors and snapshot machinery are removed; exact-ID admission uses transcript validation.
  - id: BR-5
    disposition: addressed
    note: |
      Both Codex collectors are absent and indirect selection is fenced at pure and IO boundaries. Removing each guard independently made TestNativeSelectorCodexIndirectNeverProbesEvenWithResolvedConfiguration fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-agent-error-isolation
    title: |
      A Codex home error blocks unrelated agents sharing the ledger
    detail: |
      cmd/internal/launcher/osruntime.go:705-715 returns a whole-ledger error when any Codex owner has an unavailable home. With bound Codex and Claude entries in one tag and CODEX_HOME naming a missing directory, a temporary regression calling readSavedConfigForTag for Claude failed with "cannot read native session authority ... Codex native home unavailable". Valid relative CODEX_HOME also cannot resolve here because startup cwd is deliberately unavailable. ARCH-PURPOSE: preserve per-agent availability; represent the Codex target as unavailable without discarding usable Claude/Qoder entries, and add mixed-agent ledger launch/restart regressions.
  - id: new
    severity: Critical
    kind: defect
    family: native-transcript-namespace-consistency
    title: |
      Recovery discovery and validation use different Codex transcript homes
    detail: |
      cmd/internal/launcher/recovery_authority_os.go:167-171 now validates candidates under CODEX_HOME, but cmd/internal/couchcore/recovery_evidence_os.go:199-200 still discovers them through NewOSRuntime(HomeDir, DataDir), which scans HOME/.codex. A temporary production-reader regression found one valid candidate in the default home, then found zero after moving the same tree to an absolute CODEX_HOME. This contradicts the active plan's exact/ledger/recovery consistency requirement. ARCH-PURPOSE / ARCH-DRY: share transcript namespace construction across discovery and validation, sweep remaining recovery root constructors, and add an end-to-end custom-home recovery regression.
```

1. **Strengths**

   - Indirect Codex requests stop before inventory IO, including legacy requests carrying resolved provider/SQLite metadata.
   - Separate guard-removal controls confirm the regression test detects both restored prediction authority and restored physical-CWD probing.
   - Exact-ID tests cover custom homes, explicit homes without HOME, canonical aliases, and startup cwd before `-C`.
   - README and indexed atlas updates describe the reduced scope.

2. **Critical findings**

   Both findings above have failing private regression reproductions at pinned head. Fix agent-local error isolation and align recovery discovery with validation.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused sessioninventory, launcher and wrapper tests passed normally and with `-race`; the complete artifactpath package passed. Existing tests miss mixed-agent ledgers and custom-home recovery discovery. Temporary Go overlays were removed; the checkout remains unchanged. I did not rerun the full suite.

6. **Architectural notes**

   - **ARCH-DRY — flag:** recovery bypasses the shared home-routing rule.
   - **ARCH-PURE — pass:** the remaining filesystem integration is small; selector decisions remain directly testable.
   - **ARCH-PURPOSE — flag:** recovery consistency and unchanged other-agent behavior are incomplete.
   - **ARCH-MOCK — pass:** inspected integration tests use private files and existing stateful admission seams.
   - **ARCH-CONSTRAINTS — pass:** indirect selection performs no probing; no new fan-out or background work.
   - **ARCH-SECURE — pass:** invalid explicit homes refuse authority; no new credential access.
   - **ARCH-ORDER — pass:** admission transitions remain unchanged; compatibility creates no reservation.
   - **ARCH-FUNERAL — pass:** no new durable production artifact family.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining per-agent ledger failure isolation and enumerating recovery namespace consumers, with mixed-agent and discovery-to-validation regression coverage. The explicit scope revision adequately supersedes the historical selector concepts table.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7ce7e0a5-aaf8-4265-a71c-cf42d766388a","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/d079df4c329f5341","reviewed_head":"1e3129bcbe93bb7aaf8d27b6b33785f5757ca7f3","slot":"pair:7","requested_at":"2026-10-16T02:28:35.857691Z"},"phase":"finished","pid":2099,"started_at":"2026-10-16T02:28:35.949395Z","finished_at":"2026-10-16T02:32:49.087685Z","session_id":"01a1428a-34c2-7f83-a48d-5fd7444e9bdb","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.244226667s
head: "1e3129bcbe93bb7aaf8d27b6b33785f5757ca7f3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T20:29:02-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 571 — Keep Codex indirect resumes native; route exact-ID transcript homes |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5414ef47b64388d7a7db9db8448f3a836bc62f6a..701270af8026d19ea079bca8d86b45bf526e03e2 |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T20:29:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The reduced scope is implemented coherently: indirect Codex resumes remain native, exact-ID routing honors custom homes, and BR-7 has meaningful regression coverage. BR-6 remains partially unresolved: compatibility-only ledgers bypass home validation, allowing an unavailable namespace to become a Fresh decision that deletes saved configuration.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Remains disposed; the superseding native-pass-through design removes Codex configuration projection.
  - id: BR-2
    disposition: addressed
    note: |
      Remains disposed; Codex indirect transcript selection is removed.
  - id: BR-3
    disposition: addressed
    note: |
      Remains disposed; native worktree selection is delegated to Codex.
  - id: BR-4
    disposition: addressed
    note: |
      Remains disposed; the Codex SQLite collector is removed and indirect requests return before IO.
  - id: BR-5
    disposition: addressed
    note: |
      Remains disposed; Pair no longer applies Codex indirect source filters.
  - id: BR-6
    disposition: not-addressed
    note: |
      Mixed typed ledgers now isolate unavailable Codex homes correctly, and reverting the fix makes the committed Claude/Qoder regressions fail. However, osruntime.go:705-720 populates unavailable only through parseLedger's typed-owner callback. A compatibility-only Codex ledger never invokes it. A disposable-head regression with invalid CODEX_HOME and saved configuration returned nil error, emitted “starting fresh,” and deleted config-work-codex.json. Validate namespace availability for every represented agent, including compatibility-only rows, and add saved-config/restart regressions for that case.
  - id: BR-7
    disposition: addressed
    note: |
      Recovery discovery now uses NewNativeTranscriptRuntime, matching validation and legacy held-file checks. Targeted normal/race tests pass; restoring the former discovery constructor makes the committed custom/invalid/relative-home cases fail.
```

1. **Strengths**

   - Indirect inventory requests return before native IO; trap fixtures enforce this in `native_selector_test.go:186`.
   - Home construction is shared, preserves explicit reader defaults, and rejects relative homes without startup provenance.
   - README and atlas updates document the revised native-compatibility contract.
   - BR-6’s mixed-agent tests exercise saved configuration and restart behavior through production ledger projection.

2. **Critical findings**

   - **BR-6 — compatibility-only namespace failures become Fresh.** `cmd/internal/launcher/osruntime.go:705-720` misses compatibility-only agents. Consequently, `createflow.go:1287-1291` treats unavailable authority as absence and removes saved configuration. Resolve availability independently of typed-owner callbacks, preserve configuration on uncertainty, and test compatibility-only ledgers. **ARCH-PURPOSE, ARCH-SECURE, ARCH-ORDER.**

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Targeted normal and race checks passed for inventory, launcher, and Couch home/recovery behavior. Artifactpath tests and diff whitespace checks passed. Counterfactual removal confirmed the existing BR-6 and BR-7 regressions go red. The added compatibility-only reproduction fails on pinned head. The disposable copy was removed; the reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared transcript-runtime construction.
   - **ARCH-PURE: pass** — remaining filesystem work stays in narrow IO helpers.
   - **ARCH-PURPOSE: flag** — BR-6’s unavailable-agent rule misses compatibility-only ledgers.
   - **ARCH-MOCK: pass** — private fixtures and injected final-owner observation.
   - **ARCH-CONSTRAINTS: pass** — indirect prediction and its expensive collectors are removed.
   - **ARCH-SECURE: flag** — unavailable namespace becomes fabricated absence in BR-6.
   - **ARCH-ORDER: flag** — that uncertainty permits configuration deletion before authority is established.
   - **ARCH-FUNERAL: pass** — no new durable production artifact family.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating typed-only, compatibility-only, and mixed ledgers under the same per-agent availability rule, with saved-config preservation and restart-refusal coverage.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d4fe7e93-13da-43e6-be5d-ef0d33aa5acc","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9285354d67c930ab","reviewed_head":"701270af8026d19ea079bca8d86b45bf526e03e2","slot":"pair:7","requested_at":"2026-10-16T03:23:35.684432Z"},"phase":"finished","pid":53116,"started_at":"2026-10-16T03:23:35.790001Z","finished_at":"2026-10-16T03:28:59.988633Z","session_id":"01a142bc-8ef1-7e01-841e-58103e6c4cfd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.622105625s
head: "701270af8026d19ea079bca8d86b45bf526e03e2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T20:51:48-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 571 — Keep Codex indirect resumes native; route exact-ID transcript homes |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d96738f8a9034af12459699e30c80eb02f146bc..8190469bb00c2ae6b676042414434d17e8f6f18d |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T20:51:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved reduced scope: Codex indirect resumes remain native, exact IDs retain admission, and transcript-home routing is shared across the specified launcher and recovery paths. BR-6 is addressed across typed, compatibility-only, and mixed ledgers. No new blocking findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Configuration projection was removed under the approved revision; zero-probe indirect-selector tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Codex predictive transcript selection was removed; indirect requests cannot obtain selected-ID authority.
  - id: BR-3
    disposition: addressed
    note: |
      Codex worktree-selection emulation was removed; indirect selection remains native.
  - id: BR-4
    disposition: addressed
    note: |
      The Codex SQLite collector was deleted; indirect-selector trap tests verify rejection before native IO.
  - id: BR-5
    disposition: addressed
    note: |
      Codex source-filter prediction was removed rather than extended; supplied latest metadata cannot authorize selection.
  - id: BR-6
    disposition: addressed
    note: |
      ReadLedger checks every represented agent through one cached namespace lookup. The saved-config/restart matrix passes; removing the post-parse check through a temporary overlay reproduces configuration deletion and unsafe legacy restart.
  - id: BR-7
    disposition: addressed
    note: |
      Discovery, authority validation and legacy held-file checks share NewNativeTranscriptRuntime; custom/default/invalid/relative-home regressions pass.
```

1. **Strengths**

   - Per-agent availability is enforced for every returned ledger row, including compatibility-only rows (`cmd/internal/launcher/osruntime.go:705–735`).
   - Indirect Codex requests stop before home resolution or inventory IO; tests trap lower-level probing.
   - Home routing has one shared constructor, with explicit refusal when relative-home startup provenance is unavailable.
   - README and atlas describe the revised behavior. Admission and wrapper protocol files remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Targeted launcher, inventory and recovery tests passed normally and with `-race`. The complete artifactpath package passed, and `git diff --check` was clean.

   The BR-6 removal overlay failed as expected, demonstrating that the regression detects the correction’s absence. Temporary overlay files were removed; tracked files were unchanged. I did not independently rerun the full repository suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared transcript-runtime construction and cached per-agent lookup.
   - **ARCH-PURE — pass:** filesystem observation stays in runtime boundaries; selector decisions remain separately testable.
   - **ARCH-PURPOSE — pass:** approved native pass-through replaces prediction throughout the remaining Codex selector surfaces.
   - **ARCH-MOCK — pass:** private filesystem fixtures and injected owner observations exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** no Codex selector scan or subprocess work; namespace construction occurs once per represented agent per ledger read.
   - **ARCH-SECURE — pass:** invalid homes cannot fall back, and unknown historical startup context cannot authorize relative homes.
   - **ARCH-ORDER — pass:** admission sequencing remains unchanged; unavailable namespace evidence blocks restart before effects.
   - **ARCH-FUNERAL — pass:** no new durable production artifacts or background tasks.

7. **Plan revision recommendations:** None required; the explicit approved revisions supersede the original selector design.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0d7b8aa0-6f0b-447e-9598-c039af01b7ee","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"milestone-review","round":6,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7f8e5d01a3a87a87","reviewed_head":"8190469bb00c2ae6b676042414434d17e8f6f18d","slot":"pair:7","requested_at":"2026-10-16T03:50:00.362687Z"},"phase":"finished","pid":58943,"started_at":"2026-10-16T03:50:00.595056Z","finished_at":"2026-10-16T03:51:47.289056Z","session_id":"01a142d4-be46-7f71-9c9e-1e3b1d64cd51","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.330929125s
head: "8190469bb00c2ae6b676042414434d17e8f6f18d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T23:08:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 571 — Keep Codex indirect resumes native; route exact-ID transcript homes |
| repo | pair |
| issue file | workshop/issues/000571-codex-indirect-resume-adapter.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7876717d7d1fee28c3d6ff4734c38b38c31554b..f56a0d9faa0109cbabb85f0dcca8b1d5840cb135 |
| command | sdlc close --issue 571 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T23:08:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides’ intent: source classifications remain complete, and recovery retains shared transcript-home validation while adopting main’s registry-based admission. No new defects or scope changes found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Configuration prediction remains removed; indirect pass-through regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Predictive transcript selection remains removed; indirect requests cannot acquire predicted identity authority.
  - id: BR-3
    disposition: addressed
    note: |
      Worktree-selection emulation remains removed; native indirect selection is preserved.
  - id: BR-4
    disposition: addressed
    note: |
      Codex latest rejects before native IO; the zero-probe regression passes.
  - id: BR-5
    disposition: addressed
    note: |
      Source-filter prediction remains removed; Codex indirect selection stays native.
  - id: BR-6
    disposition: addressed
    note: |
      ReadLedger checks every represented agent through its shared namespace cache; isolation regressions pass.
  - id: BR-7
    disposition: addressed
    note: |
      The public recovery validator retains shared transcript-home routing; custom/default/invalid/relative controls pass normally and with race detection.
```

1. Strengths:
   - Both source inventories survive the conflict resolution (`cmd/internal/artifactpath/manifest.go:630`); the full artifactpath suite passes.
   - Recovery keeps capture, transcript, record-change and final identity checks (`cmd/internal/launcher/recovery_authority_os.go:191`).
   - Atlas wording accurately distinguishes standalone admission from Couch recovery (`atlas/codex-resume.md:9`).

2. Critical findings: none.

3. Important findings: none.

4. Minor findings: none.

5. Test coverage: affected launcher tests, recovery-home race test, registry/cold-resume integration controls, indirect-selector and ledger-isolation regressions all pass. `git diff --check` passes. The full repository suite was not rerun during this review.

6. Architecture: **ARCH-DRY — pass:** obsolete delegation seam removed; shared home construction retained. **ARCH-PURE — pass:** no new business logic mixed into IO. **ARCH-PURPOSE — pass:** both conflict resolutions preserve the approved purpose without restoring prediction or legacy ownership policy.

7. Plan revisions: none required; this window records integration verification without changing the active contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5a065fa9-4df3-4b9c-9380-c0395f75d4b4","repository":"github.com/xianxu/pair","issue_id":"000571","recipe":"fix-review","round":7,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6f3532bc99e07ea1","reviewed_head":"f56a0d9faa0109cbabb85f0dcca8b1d5840cb135","slot":"pair:7","requested_at":"2026-10-16T06:06:48.201691Z"},"phase":"finished","pid":99928,"started_at":"2026-10-16T06:06:48.31878Z","finished_at":"2026-10-16T06:08:23.298996Z","session_id":"01a14351-fae4-7f81-9205-252de018de2b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.464941084s
head: "f56a0d9faa0109cbabb85f0dcca8b1d5840cb135"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
