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
- [x] Update indexed inventory/recovery atlas and compatibility support/version docs; run targeted normal/race and the full keep-going suite, and prepare evidence for binary-owned close and landing.

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

### 2026-10-15 — close checklist and verification outcome

Clarify the final checkbox as pre-close deliverables so future close/landing actions are not falsely marked complete. Full keep-going suite passed all recipes/packages exit0 on code identical to8bd4baa1; required recorded same-code rerun is executing after independent flakes were filed as unassigned #585 and #586, per the TL-approved bounded retry rule. Its publish status remains pending. No adapter design or scope changed.

### 2026-10-15 — BR-1/BR-2 source-fidelity correction

Ordered CLI overrides need native final-leaf replacement, separate from recursive config-layer merging; preserve only the tagged structured-feature exception. Sweep supported table-valued assignments and test both effective projection and selected ID. Native record authority requires successful native schema eligibility for the rollout envelope, session metadata and turn context, including typed optional-present fields. Support a source-proven bounded subset and retain compatibility for unproven records; do not change generic inventory validation to claim native selection proof. Replace incomplete positive fixtures with native-valid records, test malformed newer records and no wrong-ID reservation, then run normal/race and fresh recorded full suite before the next close. This narrows faithful guarded admission within the original scope; no credentials or startup oracle.

### 2026-10-15 — implemented native eligibility subset and scoped flag fallback

CLI leaf replacement and all four structured-feature exceptions are tested separately from config-layer merges, including selected-ID integration across linked checkouts. The parser conservatively retains native compatibility for global worktree toggles mixed with subcommand-scoped config; selector-independent Couch shell_snapshot injection remains supported. Native record eligibility validates required fields and a source-proven optional/default subset; unknown complex shapes and JSON Unicode decoder gaps retain compatibility. Generic legacy inventory semantics remain unchanged. Targeted normal/race and read-only machine checks pass; fresh recorded suite and REWORK disposition remain pending.

### 2026-10-15 — TL-approved exact map-key vocabulary context

Add one counted direct Go map-key context to the artifact-source vocabulary grammar (function, exact literal, count), with moved/duplicate/constructed/path-use negative controls. Classify the native record eligibility helper as native-session VocabularyConsumer. No non-artifact exemption or broad allowance. The final suite exposed this missing source classification; run the complete artifactpath package, not a name-filtered subset, before another recorded suite.

### 2026-10-15 — BR-3/BR-4 proposed Revision: discovery uncertainty and non-mutating WAL snapshots

Close round2 verified BR-1/BR-2 addressed, then raised two new Critical findings. Production edits for this revision are paused pending TL approval; private regressions already reproduce both defects.

**Git rule separation (BR-3).** Tagged [worktree repository identity and discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/worktree.rs#L34-L177) canonicalizes admin pointer targets; tagged [trust discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/trust.rs#L139-L151) rejects the original git-dir symlink before canonicalizing. Keep strict trust rejection, but never treat its inability to prove a discovery candidate as authoritative exclusion. Any observed current/primary/linked candidate with unsupported alias, permission/IO failure, incomplete metadata or exhausted observation becomes whole-projection compatibility. Only positively native-proven exclusions may skip: no eligible Git root in fully observed ancestry, non-directory registration, a nonexistent/non-directory corresponding cwd, a canonical corresponding cwd outside its checkout root, or positively different repository/common/relative identity. No new UID policy. The private production-selector RED selects an older ID after an admin alias hides the newer linked match; the fix must select the native newer ID or retain compatibility, never the older one. Matrix includes current/primary/linked aliases, FIFO, missing metadata, permissions and positive native exclusions; project-only API proof must still refuse a trust alias.

**Read-only SQLite boundary (BR-4; proposed narrowed proof).** Native read-only SQLite can create or update native WAL-index sidecars; [SQLite WAL documentation](https://www.sqlite.org/wal.html#read_only_databases) explicitly permits this. Never open the source database through SQLite. Add one lazy selector-owned private DB+WAL snapshot per Codex resolution attempt, reused for pagination and independent selected-content inventory checks. Generic idle/tick inventory and other agents retain existing behavior; this boundary covers all Codex adapter query paths. Use existing context/deadline and aggregate256MiB source-read ceiling, including copy AND verification bytes and metadata counts. No retry on uncertainty.

The supported snapshot requires a present regular WAL, a WAL-mode DB header, secure no-follow source handles and unchanged file identity/size/generation fences. Copy WAL first, then DB into a private0700 temporary directory with0600 files. Re-read both complete sources and compare hashes/bytes to copies, with pre/during/post identity/size/header/absence checks. A same-generation checkpoint only copies committed WAL pages into DB: replaying the stable committed WAL overlay remains the same logical image. Reset/recycle changes WAL salts/generation and full content even if inode/size/mtime coincide; reject it, replacement, truncation or any changed observation. [SQLite file format WAL sections4.3–4.5](https://www.sqlite.org/fileformat.html#the_write_ahead_log) define committed-frame/checkpoint/reset semantics; the WAL-index contains no authoritative DB content and is rebuilt privately ([WAL format](https://www.sqlite.org/walformat.html#the_wal_index_or_shm_file)). Run private recovery and bounded quick_check before selection: this verifies the recovered image, while source fences establish coherence (quick_check alone is not coherence proof).

Do NOT stat/read/copy native SHM. SQLite executes only against the private copy, with hermetic safe CLI options (-batch, explicit empty init, -safe, -readonly) so a user rc or schema file functions cannot perform external effects; unsupported required CLI behavior yields compatibility. The selector runtime owns lazy snapshot construction and Close, with deferred cleanup in every Codex collector/validation exit; partial creation, query failure, timeout and cancellation remove the private DB/WAL and any private SQLite sidecars. Cleanup failure cannot finalize guarded authority. No new journal, source lock, app-server or credential reads.

**Explicit compatibility branch requiring TL decision.** An absent WAL can appear, checkpoint and disappear between probes; timestamp/absence brackets do not prove no ABA. Correct native byte-range locking would need pinned descriptor/process ownership and introduces a second mechanism. Proposed simple boundary: absent WAL, non-WAL mode, hot journal, unsafe/oversize/changing input => native compatibility, including ordinary closed DBs with no WAL. Preserve original argv and the existing not-guarded notice, no reservation. Upgrade positive SQLite fixtures to native WAL snapshots rather than pretending DELETE fixtures prove native observation. This intentionally narrows guarded evidence while keeping native UX; no blanket config or launch refusal.

**Verification.** Private TUI/exec WAL regression already resolves expected ID then exposes a new32768-byte native SHM before the fix. After the fix the entire native directory file set, bytes/hashes, identities, sizes and mtimes must be unchanged (including absent SHM), and all SQLite query paths must target isolated scratch. Test checkpoint/reset/replacement/truncation/create-delete during copy as conservative compatibility, same-generation stable WAL overlay, unsupported/absent WAL no-reservation fallback, source byte-budget exhaustion, rc/schema external-effect rejection and cleanup after success/error/cancel. Preserve all prior CLI/schema/namespace/ownership tests, then full suite and fresh close before landing.

### 2026-10-15 — BR-3/BR-4 implementation authorized

TL approved the preceding Revision at0aa26c95 (receipt3303cfe5-0e36-4214-8600-2a73a507022a). The proposed absent-WAL compatibility branch is selected: implement only stable-present-WAL evidence, without source locking or native SHM observation. Private REDs are checkpointed in ae9ec28a1 and e18f37604. The two issue checklist rows remain open until normal/race regression evidence and fresh full-suite/close disposition are recorded.

### 2026-10-15 — mechanism re-plan after BR-5: native selection surface first

**Reason and stop boundary.** TL receiptab449251-6998-4347-a9e1-f718c19a3be5 requires a mechanism re-plan after repeated fidelity deltas. No production code has changed since the two passing full suites. The four-case BR-5 selected-ID RED was completed before this stop and remains measurement only. Do not implement the following proposal until the TL has seen this Revision and chosen the direction.

**Answer: no suitable native resolver was found in installed-version source (rust-v0.162.1).** Codex has native selection code and public listing primitives, but no exposed read-only operation that accepts the actual TUI/exec resume invocation and returns its selected UUID before opening a session. This conclusion is source inspection, not an assertion that native was executed. No Codex/app-server process, credentials, auth files or keychain were used for this investigation.

| Native surface examined | What it actually provides | Why it cannot replace this adapter under the current contract |
|---|---|---|
| TUI resume/fork command | Session ID/name, last/all/include flags followed by normal TUI startup | No resolve-only/dry-run mode in the command shape; normal startup is not a read-only preflight. [CLI](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/cli/src/main.rs#L354-L446), [dispatch](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/cli/src/main.rs#L1456-L1487). |
| Exec resume, including JSON/ephemeral | Native client starts an in-process app-server, resolves, then immediately resumes or starts a thread | JSON changes output, not lifecycle; ephemeral is not resolve-only. The selected-ID event is too late for pre-start writer admission. [arguments](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/cli.rs#L13-L78), [flow](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/lib.rs#L1022-L1064). |
| App-server thread/list | Native DB listing and source filtering for explicit API parameters | Requires a server; it is not a resolver for CLI argv/profile/env. TUI computes provider/worktree/cwd parameters in its client, and exec separately inspects latest turn cwd. Using listing still duplicates those policies. Existing-server proxy does not solve invocation-config equivalence. [TUI parameters](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/lib.rs#L827-L900), [exec resolver](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/lib.rs#L1851-L1925). |
| Native state/index DB through the safe private snapshot | Existing adapter already reads Codex's own rows using SQLite | No stored latest-selector view/function was found: queries are constructed in Rust and API source filtering runs in Rust afterward. Snapshotting proves non-mutation/coherence, not native selection semantics. [state query](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/runtime/threads.rs#L510-L551), [post-filter](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/app-server/src/thread_processor.rs#L5539-L5559). |
| Linked native Rust helper | Native library list_threads_db expects a StateRuntime | Its pool is private; the public initializer creates/migrates state/logs/goals/memories/queue DBs. The non-owning get_state_db skips backfill but still calls that initializer. Read-only pool access is exposed for integrity checking, not a complete resume resolver. [runtime](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/runtime.rs#L91-L166), [optional reader](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/rollout/src/state_db.rs#L204-L235), [read-only pool](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/sqlite.rs#L449-L467). |

The missing surface is a native **resolve-resume invocation to UUID/namespace, without opening/resuming/migrating/repairing native state**, with the same TUI/exec semantics as the subsequent launch. Adding that upstream, or maintaining a version-matched native helper with an explicitly read-only constructor, is a separate architecture/dependency change. A helper bundled at one tag can still drift from a different installed Codex. Starting a server against redirected copies also changes configuration/path context and is not a demonstrated replacement. Do not call any of these alternatives version-independent today.

**Decision requested.** Under the existing #571 contract, the smallest continuation is the BR-5 source eligibility correction below, retaining all conservative fallback boundaries. It remains a maintained source-derived adapter, not a native oracle. If the operator now requires delegation that eliminates selection-policy drift, retain native-compatible indirect resumes until a native read-only resolver exists; exact-ID admission can remain. That would revise the feature's scope, so the worker will not choose it unilaterally. No new helper/server/version-probe mechanism is proposed for this round.

**Minimal BR-5 rule, only if continuing the current adapter.** Separate native source decoding/eligibility from transcript schema/root eligibility. Read the persisted source column in the shared bounded selector row; use one pure classification with eligible, positively excluded and unproven outcomes. Known excluded sources cannot affect target selection or tie authority. Unproven needed rows retain native compatibility and cannot be skipped to pin an older UUID. Preserve raw-row timestamp pagination and budgets; add no broad prefix exclusions or new source authority.

| Native SessionSource and persisted representation | Native TUI | Native TUI include-non-interactive | Native exec latest |
|---|---|---|---|
| Cli / bare cli | eligible | eligible | eligible |
| VSCode / bare vscode | eligible | eligible | eligible |
| Exec / bare exec | excluded | eligible | eligible |
| Mcp / bare mcp | excluded | eligible | eligible |
| Custom(String) / object with custom string | excluded | excluded | excluded |
| Internal Guardian or MemoryConsolidation / object with internal guardian or memory_consolidation | excluded | excluded | excluded |
| SubAgent Review/Compact/MemoryConsolidation/Other(String)/ThreadSpawn(UUID,depth,optional metadata) / typed subagent object | excluded | excluded | eligible by native source filter; existing independent schema/root uncertainty may still require adapter compatibility |
| Unknown / bare unknown; unknown scalar decoder fallback | excluded | excluded | eligible natively; unproven/noncanonical adapter decoding retains compatibility, never guessed exclusion |

Source: [SessionSource and variants](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/protocol/src/protocol.rs#L2854-L2951), [persisted enum encoding](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/extract.rs#L204-L210), [actual TUI source kinds](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/lib.rs#L770-L778), [exec kinds](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/exec/src/lib.rs#L1802-L1815), [native kind matching](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/app-server/src/filters.rs#L53-L81). Generic INTERACTIVE_SESSION_SOURCES is not the TUI request's explicit kind filter. Display labels such as internal_guardian and subagent_review are not their structured persisted representation. Bare atlas/chatgpt are not Custom objects: they must yield native-compatible/eligible behavior, never be wrongly skipped as Custom.

**Verification before another close.** The existing four private real-WAL selected-ID cases are RED (2.985s): newer valid custom atlas/chatgpt and internal guardian/memory_consolidation incorrectly win over an older cli row. After approval, sweep TUI defaults/include and exec; test positive unit kinds, all positively decoded excluded forms, unknown/malformed forms preserving original argv with no reservation, same-timestamp exclusions and a page of excluded rows before an eligible row. Keep native directory non-mutation and all prior scope/schema/CLI/namespace controls. Then normal/race affected packages, source guards, one fresh recorded full suite and binary-owned close. Prior green full suites do not verify future source-filter code.

### 2026-10-15 — TL zero-drift decision: reduced scope proposal before code

**Authority and active contract.** TL submitted receipt3b2dcde8-9ca1-431b-9391-4e5871a78655 chooses the zero-drift alternative. This supersedes the original guarded-indirect-selector Spec/Done-when and the proposed BR-5 classifier. Codex chooses every indirect resume itself; Pair does not predict a UUID. Only recognized complete canonical Codex IDs use the existing guard. Claude/Qoder and other-agent behavior stays unchanged. The proposed title is "Keep Codex indirect resumes native; retain exact-ID home routing". No implementation changes until the TL reviews this reduced scope.

**What remains.** Existing main already has the desired ordinary launcher/wrapper compatibility short circuit, exact-ID validation and Reserve/Claim/Publish path. Restore that behavior rather than building another selector. Retain only policy-free transcript-home routing: resolve nonempty CODEX_HOME as an existing physical directory relative to actual process startup cwd, or use the default home/.codex when empty/unset; WithCodexHome changes only codex-sessions. Explicit CODEX_HOME does not require HOME. Native config/profile/provider/SQLite/Git/managed/cloud data is irrelevant to this helper. Exact-ID inventory, ledger identity and recovery use the same transcript root; invalid explicit home cannot fall back to the default. A relative home must never be anchored to native -C or a remembered conversation cwd; where startup provenance cannot be established, return unavailable rather than infer it. [Native home utility](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/utils/home-dir/src/lib.rs#L13-L61).

Keep the admission protocol exactly as main: global(agent,UUID), existing argv digest, attempt/process-generation guards, existing grant bounds/CAS. That global key conservatively serializes equal IDs across physical homes; add no home-specific ownership, grant field, scope encoding or digest. #595 owns the subsequent wrapper-held flock replacement when the actual ID becomes known; do not implement it here.

| Area | Reduced patch |
|---|---|
| New configuration/selector mechanism | Delete codex_config projection/MDM/platform implementations and tests, codex_git_scope and tests, codex_platform_scope and tests, codex_latest and every dedicated latest/config/source/WAL/conformance test, codex_rollout_eligibility and tests. Extract only the small home resolver into codex_home.go with focused tests. |
| SQLite snapshots | Delete snapshot production/tests/WAL fixture helpers and all lazy ownership, factory, byte-charge, validation and cleanup glue. Exact Codex IDs validate transcripts and have no SQLite consumer. |
| Old main Codex latest implementation | Remove the Codex collector route/body from native_selector_collect.go and its Codex-only SQLite runtime/path helpers. Add an early Codex+Latest unsupported result before cwd/list/stat/read/query work in ResolveNativeSelector. Merely deleting the new scoped implementation would resurrect main's old SQL selector. |
| Generic native inventory | Restore baseline native_selector IO. Preserve generic bounded/cancellable SQLite support used by Agy and all Claude/Qoder logic. Remove Codex selection config/scope fields only when no remaining caller needs them. |
| Launcher/wrapper/admission | Restore all #571 selector/config capture, createflow prediction, scope serialization/digest/inspection and wrapper reprojection changes to main. Remove new scope/config/SQLite launcher tests. Keep only home-root routing in exact resolver, native_inventory_os helper and ledger/recovery callers, with focused custom-home tests. The ordinary indirect path bypasses resolution before even inspecting CODEX_HOME. |
| Dependencies and source guards | Remove Go TOML dependency/checksums. Remove the map-key vocabulary grammar and Git resource exemptions introduced solely for deleted code. Reclassify only the small remaining home/inventory sources normally; no blanket exemptions. |
| Documentation | Rewrite product-facing Codex mapping around native indirect behavior, exact-ID transcript roots and the missing upstream contract. Preserve issue/plan/gate evidence and historical Revisions rather than erase the abandoned mechanism. Add the mechanism lesson. |

**Line accounting (measured before code; projected result clearly separate).** At51d265c557ccd39e093536d75beabe7cac528726 versus origin/main: all63 files +8026/-133, net+7893; production/dependencies net+2640; Go tests net+3684; product docs net+174; workshop evidence net+1395. The new inventory mechanism is23 files/4480 added lines to remove as a unit, except roughly30 lines extracted for home resolution. Launcher/wrapper currently net+1504; home-only routing/tests retain roughly210-220 local net lines, removing about1290 there. Shared home support adds about50-60 lines outside those directories. Restoring branch-only audit allowances and removing main's old Codex SQL collector/helpers reduces the patch further. The expected remaining Go/dependency/test patch is about **net+150 to+400 lines versus main, instead of+6324** (about5900-6200 net lines removed from the current implementation). This is a projection, not a fabricated final diff: report exact signed production/test/total counts using git diff --numstat after implementation, separately from retained workshop history.

**Active Done when / verification.**

1. Ordinary Codex TUI/exec/e latest, picker and non-UUID name/prefix forms launch original argv unchanged, publish no predicted ID, create no reservation, and do zero config/home/transcript/Git/SQLite/provider/process probing. Opaque or malformed native configuration belongs to native; do not parse it to gain authority. The wrapper emits the existing one-line not-guarded notice once.
2. Strict ResumeRequired/recovery never treats indirect compatibility as guarded admission. Existing exact canonical UUID tests still validate root/content, reserve/claim one writer and reject disputed/child roots. The admission protocol remains unchanged.
3. Focused transcript-only home tests cover explicit absolute/relative/symlinked/unset/empty/invalid homes, explicit home without HOME, startup cwd different from -C, intended custom/default namespaces, and exact/ledger/recovery consistency. No config, DB, cloud or credential fixture is needed.
4. Trap/counter fixtures prove zero lower-level work for Codex latest even through the old inventory API with seemingly complete legacy provider/sqlite flags. Keep positive Claude/Qoder latest and Agy exact controls, then affected normal/race tests, complete source guards and one fresh recorded full suite before a fresh full close. Earlier green suites verify the removed mechanism, not this reduced patch.

**Condition for revisiting.** Revisit guarded indirect prediction only when native Codex exposes a supported read-only "resolve resume target" command/API which accepts the real invocation's TUI/exec arguments, cwd and configuration context and returns the actual selected ID/namespace without opening a conversation or modifying native state. Pair could then check that answer and pass the exact ID through its guard without making a second selection. Reading Codex's state DB with our own SQL/policy does not meet that condition. Until then, native compatibility is the deliberate result, not a pending source-kind patch.

### 2026-10-15 — reduced-scope execution approval

TL submitted receiptd27b887d-2cdd-427e-bcb9-d0287ec22e9a approves the preceding reduced-scope Revision atfafc33b8. It is the active implementation contract; earlier selector/classifier tasks are superseded. Keep the exact-ID path a thin caller of main's unchanged admission protocol for #595. Implementation now resumes, with independent home, selector-deletion and launcher/wrapper tasks, followed by combined verification and a fresh full close.
