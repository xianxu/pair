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
