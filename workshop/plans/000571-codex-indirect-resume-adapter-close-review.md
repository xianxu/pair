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
