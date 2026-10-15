# Codex indirect-resume adapter implementation plan

> **For agentic workers:** Consult AGENTS.md Section3 for bounded delegation. The SDLC binary owns fresh-context reviews. Checklist rows below are tasks, not milestone boundaries.

**Goal:** Source-faithful read-only Codex configuration and distinct TUI/exec latest selection, with exact-ID native admission when complete and unchanged native compatibility otherwise.

**Architecture:** Inventory owns the pure config projection, canonical native roots and bounded metadata collectors. Launcher/wrapper reuse #556's selector parser, pin/reserve/claim flow and add scope binding at handoff. No credential reads or startup/configuration oracle.

**Tech stack:** Go, a real TOML parser, existing bounded recovery IO/SQLite command seam, portable real config/SQLite/JSONL fixtures, and existing launcher/wrapper stateful harness.

## Approval and exclusions

TL receipts34e0ac39 and bed8618a approve config precedence, separate selectors, custom homes, native compatibility and credential prohibition. #556 landed bcf734588 and scratch space is released. One full-flow close. No picker/name/prefix resolution, native repair, remote/executor/environment/worktree-creation support, network config fetch, auth/credential/keychain access, secret-env classification, new saved defaults, or second admission engine. Unsupported native contexts continue their original invocation with exactly one not-guarded note; strict recovery/ResumeRequired never downgrades.

## Concrete production authority (PQ-1)

**Supported class:** local ordinary TUI resume --last or exec/e resume --last whose fully observed pre-cloud bootstrap forces API-only login, with no unresolved managed/system/project inputs or invocation relocation. Complete coverage is derived internally from real config sources; no caller-supplied completeness boolean.

Source tag rust-v0.162.1:
- TUI startup_orchestration.rs:142-150 and exec/src/lib.rs:496-515 both load local bootstrap with an empty cloud loader before deriving cloud-loader auth restrictions.
- core/config/auth_keyring.rs:39-78 retains forced_login_method/local managed auth policy; config/auth_policy.rs:11-39 requires allowed methods satisfy that restriction.
- login/auth/manager.rs:1221-1268 filters auth by that policy; every backend-auth method maps to ChatGPT. cloud-config/service.rs:52-60 requires backend auth to load cloud policy.
- Therefore resolved forced_login_method='api' excludes cloud without reading auth files. Use this narrow proof initially. A local API-only requirements policy can be added only if its exact semantics are implemented; otherwise requirements presence is compatibility, not a permissive default.
- Remote targets clear restrictions (TUI lib.rs:368-376); exec --worktree relocates/bootstraps twice (exec lib.rs:397-487): these contexts are compatibility.

**Observable provenance:** Read the real system/base/profile/trusted-project/session layers, detect unhandled managed requirements/MDM inputs, apply native precedence/path origins, and project effective forced_login_method. Only the source-backed API-only value plus complete local coverage authorizes the positive status. Unknown/unreadable/malformed/over-budget observations never imply absence. Current machine lacks this proof, so its ordinary invocation remains native compatibility. Real DB measurements are conditional source-query controls, not proof of guarded cloud support.

**Positive fixture:** real private CODEX_HOME/config.toml with forced_login_method='api', profile/CLI and trusted-project inputs, native SQLite and transcript tree. This fixture must pass through the SAME OS reader, pure projection, collector, pinning and reservation path used by production; no test-only enablement flag. A recording reader proves no auth path or keychain access. Cloud-unknown uses the same production reader and cannot reserve.

## Core concepts

| Pure entity | Location | Change |
|---|---|---|
| CodexConfigInvocation / CodexInvocationMode | sessioninventory/codex_config.go | new typed cwd/profile/ordered override/mode data; populated by existing launcher selector parser |
| CodexConfigProjection / CodexSelectionScope | sessioninventory/codex_config.go | new tagged complete/compatibility result, closed reason, minimal homes/provider/cwd/worktrees/mode; no raw configuration, secret env or auth state |
| NativeSelectorConfig | sessioninventory/native_selector.go | add typed Codex scope; existing other-agent contracts remain |
| Native admission namespace | launcher/native_admission.go | add optional serialized Codex scope to the existing comparable request/grant, validated and bound to the invocation digest |

| IO integration | Location | Wraps |
|---|---|---|
| ReadCodexSelectionConfig | sessioninventory/codex_config_os.go | bounded named local config/env/platform/physical-directory observations |
| Scoped Codex roots | sessioninventory/runtime_os.go | canonical native root owner; separate transcript and SQLite homes while preserving runtime capabilities/cache/work observers |
| Codex TUI/exec collectors | sessioninventory/native_selector_collect.go plus codex_latest.go | existing read-only SQLite and recovery IO, no migration or scan-and-repair |
| Resolution and handoff | launcher/native_createflow_os.go / createflow.go; wrapcmd/native_admission.go | existing #556 parser/pinning/reservation/claim/spawn |

ARCH-DRY: one parser/pinner, one canonical-root owner, one admission state machine. ARCH-PURE: projectCodexConfig is pure over typed observed layers; readCodexSelectionConfig is the injected IO shell. Export minimal ReadCodexSelectionConfig and projection/scope types; keep parsed-layer internals private.

## Configuration contract

CODEX_HOME empty/unset defaults to HOME/.codex; nonempty physically canonicalizes an existing directory, without applying sqlite-env trimming/tilde rules. NAME.config.toml is the runtime profile above base config.toml; legacy profile selection is not profile-v2. File paths resolve against each originating directory before merge, CLI/env paths against effective cwd. TOML dotted CLI overrides use native raw-string fallback and root-before-subcommand ordering. Use a maintained real TOML module, not regex/line parsing.

Read source-backed system/base/profile/trusted-project/session layers. Project discovery/trust must include validated main-worktree fallback; project provider keys are ignored, while sqlite_home/features/forced_login_method may apply. Relevant higher managed/requirements/MDM inputs not implemented are explicit compatibility. Forced MDM detection uses native CoreFoundation forced-preference checks, never plist/defaults-read absence inference.

SQLite precedence: exact requirements if supported, merged sqlite_home, trimmed nonempty CODEX_SQLITE_HOME, Codex home. Provider: required if supported, invocation/OSS override, merged provider, openai. TUI history provider follows explicit-layer history rules; unsupported provider or invocation semantics stay native. Preserve defaults/prompt/unrelated flags verbatim.

Worktrees defaults ON. Reproduce validated current/primary/linked corresponding existing cwd directories and canonical aliases from native Git metadata rules; --all removes cwd filtering. No creation or attach/resume side effects during config/selector observation.

## Separate native selectors

**collectCodexTUILatest:** active nonempty-preview records, interactive cli/vscode sources plus exec/mcp when requested, history provider, effective worktree/all cwd scope, updated_at_ms descending. Native requests limit1; fetch enough boundary metadata to prove a unique winner, but never skip its unusable first row to guard a different candidate. Validate actual path/root/ID; preserve native fallback if it would need repair.

**collectCodexExecLatest:** updated_at_ms pages100, all native source kinds, configured provider, no SQL cwd predicate. Validate state-only first session_meta identity; latestCodexTurnCWD reverse-scans for last turn_context.cwd, falling back to DB cwd only after complete absence, and compares normalized cwd; --all bypasses that predicate. Do not reuse TUI filters. Equal timestamp ambiguity or unreadable higher-ranked evidence cannot authorize a lower row.

Native UpdatedAt has no specified ID tiebreaker. Ties, no match, partial DBs, unsupported schema/source scope, canceled or exhausted reads retain native compatibility (and native repair/StartFresh) rather than inventing ordering/absence. Explicit UUIDs retain #556 admission. Pickers/names/remote retain native behavior. NativeSelectorResolved is the only status permitting pin/reservation.

## Existing integration and state ordering

#556 NativeResumeRequest already carries Agent/CWD/Selector/Config; OSRuntime.ResolveNativeResume is the config-loading boundary. Extend ParseNativeResumeSelector's existing option-position walk to retain relevant config/profile/feature/mode inputs, rather than installing a second argv parser. ResolveNativeResume and PinNativeResumeSelector remain the sole ID rewriting path.

createflow.go:750 currently bypasses every Codex indirect selector. Offer latest resolution before compatibility; a non-resolved ordinary result retains the original invocation, no reservation, and only the wrapper's existing single notice. Existing strict recovery/ResumeRequired remains a refusal. Do not let cancellation or partial attempt state start a native child.

Bind resolved Codex physical transcript/SQLite scope and effective cwd into the existing NativeAdmissionRequest/grant and scoped command digest. Namespace metadata is private and validated; it grants no authority alone. Expose a bounded read-only grant-scope inspection before Claim. Wrapper reprojects the actual final invocation environment/cwd through the same reader and compares scope before consuming the grant. Missing/altered scope cannot downgrade to legacy digest, and malformed/forged scope fails the stored request/grant CAS. Keep legacy unrelated-agent and old explicit-UUID request behavior intact.

ARCH-ORDER: observe -> unique selection -> reserve -> explicit argv -> verify scope -> Claim -> spawn -> publish child. Reuse #556 cancellation/uncertainty/teardown transitions. A namespace change before Claim is a retryable refusal with no child; never rerun indirect selection at handoff. A newer DB winner cannot retarget a granted ID. No new persistent state machine.

| Observation | Person sees | One action |
|---|---|---|
| Complete selection/admission | Ordinary pinned resume | Continue |
| Existing writer | Existing actionable busy refusal | End that writer, retry |
| Unknown cloud/config/selector/evidence | Original native invocation, one not-guarded note | Use native UI |
| Scope changed before claim | Failed attempt, no mismatched spawn | Retry |

## Function-level test strategies (PQ-2)

| Risky function | Strategy and mechanical guard |
|---|---|
| ParseNativeResumeSelector / PinNativeResumeSelector | Fuzz malformed/ambiguous argv with option-position seeds; preserve nonselector tokens/prompt and prove reparse is one explicit canonical ID |
| projectCodexConfig | Pure generated ordered-layer/path-origin combinations against independent precedence oracle; typed invalid/unknown cannot become complete and API-only proof cannot be caller-injected |
| readCodexSelectionConfig | Stateful real config tree plus recording bounded reader/forced-preference fake; mutate inputs between observations, enforce forbidden-path and read-budget guards |
| withCodexScope runtime construction | Private canonical/symlink namespaces; existing recovery/root authorization must confine transcript and DB independently without reaching real HOME |
| collectCodexTUILatest | Real SQLite/transcript fixtures with adversarial ordering/filter/boundary rows; independent source-derived expected result or explicit compatibility, never lower-row guess |
| collectCodexExecLatest / latestCodexTurnCWD | Real SQLite plus generated/fractured/reverse JSONL records; monotone read budget and snapshot identity prevent partial tails from establishing absent cwd |
| ResolveNativeResume | Existing stateful runtime with candidate arrivals after selection; exact pinned ID/argv survives without a second selection |
| NativeAdmissionScopedCommandDigest / grant-scope inspection | Typed serialization round-trip and malformed input generation; missing/mutated binding cannot match a scoped reservation |
| startNativePTY scope verification | Controllable pre-Claim config/env mutation and existing stateful admission store; assert no Claim/spawn on mismatch and one writer under competing attempts |

## Tasks (single atomic boundary)

- [x] Materialize approved plan, answer gate findings, then derive estimate and enter change-code.
- [x] RED/GREEN pure config projection and bounded reader, using real source-backed API-only positive fixtures and cloud-unknown negative controls.
- [x] RED/GREEN scoped native runtime and separate TUI/exec collectors via private real SQLite/transcript state; retain native fallback.
- [x] RED/GREEN parser/createflow and namespace-bound admission/handoff, preserving other agents and old strict routes.
- [x] Run read-only private-snapshot real-machine source/query and adapter comparison; observe compatibility here. Coordinate any live action through hub.
- [ ] Update current indexed inventory/recovery atlas and compatibility support/version docs; run targeted normal/race then recorded full suite; close, merge main, land SHIP, report real hashes.

## Operating envelope and residue

ARCH-CONSTRAINTS: explicit launch only, not ticks/messages. Config<=1MiB/file,8MiB total,64 ancestor layers; Git metadata<=64KiB/file. RecoveryScanTimeout30s, RecoveryTotalReadLimit256MiB and metadata<=10000 remain; exec pages100/rows<=10000, reverse chunks<=readChunkSize and<=32MiB per transcript charged to total; SQLite output<=1MiB/page. These are compatibility bounds, not native limits; exhaustion is unknown. No cache of selection authority, background task or timing correctness threshold.

ARCH-SECURE: bounded named noncredential inputs only; no auth/credential/keychain reads, secret-env classification or network config. Preserve no-follow/path identity controls. Diagnostics only field names/reasons; no raw TOML/native records or prompts. All fixtures use private HOME. Real-machine SQLite comparison opens only temporary DB/WAL snapshot copies.

ARCH-MOCK: portable real files/DB/JSONL plus stateful forced-preference and existing native process/admission worlds behind production seams. Read-only real-machine/source conformance complements synthetic fixtures; no app-server oracle.

ARCH-FUNERAL: no new durable production artifact or child outside #556 ownership. Attempt snapshots expire at return; private measurement copies removed after closing handles. Namespace binding is bounded metadata on existing per-attempt grants, removed by existing admission teardown.

## Verification

Targeted: go test ./cmd/internal/sessioninventory ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'Test.*(CodexConfig|CodexLatest|NativeSelectorCodex|NativeResume|NativeAdmission|NativeCreate)' -count=1

Race: same packages/regex with -race. Source ownership: artifactpath package plus native-authority-shadow tests. Then sdlc pr; sdlc local-check run --name suite; finish failed recipes with make -k and name all failures; sdlc close with evidence; merge origin/main; sdlc merge --yes. Binary owns fresh review; no bypass.

## Revisions

### 2026-10-15 — source comparison portability boundary

Exact tagged path-utils source confirms exec compares fully canonicalized cwd paths, matching the collector on macOS. On WSL it additionally folds ASCII case for /mnt/<single-letter-drive>. Keep Linux physical cwd paths in that domain on native compatibility rather than skip a newer native match and guard an older candidate. No WSL environment or platform-file reads are added; the policy is a pure path/platform predicate. No macOS behavior or accepted bootstrap class changes.

### 2026-10-15 — implemented limits and native-fidelity findings

Config budget is4MiB total (stricter than8MiB),64 reads,128 ancestry,256 bounded worktree registrations; native scope8KiB and existing admission grant64KiB (previous16KiB cannot hold worst-case encoded scope); store remains1MiB. Preserve startup cwd separately from -C for relative CODEX_HOME. Literal dot-split CLI keys, provider membership, secure no-follow metadata, original Git pointer alias validation, timestamp cursor pagination and every inspected transcript's identity are required by behavioral regressions. Propagate inventory-owned custom native home through existing ledger/recovery checks, not only latest selection. Real config projection remains unguarded observation-incomplete at the secure project-file boundary; do not infer cloud absence. No credential reads, second native selection, invented UID ownership rule or tie-breaker.

### 2026-10-15 — PQ-1/PQ-2

Replaces the materialized2e712aec design's open policy-provenance obligation with the independently source-verified API-only local bootstrap proof for TUI AND exec; ordinary current-machine config remains cloud-unknown. Replaces case inventories with named function/adversarial/mechanical-guard strategies. Prior full proposal remains versioned on2e712aec and in the approved issue publication history; no scope or credential boundary changed.
