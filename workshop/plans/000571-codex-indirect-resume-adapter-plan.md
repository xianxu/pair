# Codex indirect-resume adapter implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Use bounded subagents for isolated tasks; the binary owns fresh-context review. Steps use checkboxes for tracking.

**Goal:** Resolve source-faithful read-only Codex configuration and supported latest selectors, reuse native writer admission for complete evidence, and retain native compatibility for uncertainty.

**Architecture:** A pure configuration projection and separate TUI/exec metadata collectors feed #556's canonical-ID pin/reserve/claim flow. Canonical native roots stay in inventory. Credential files are forbidden; cloud-unknown never authorizes admission.

**Tech stack:** Go, a real TOML parser, bounded inventory IO, real private SQLite/JSONL/config fixtures and the existing launcher/wrapper stateful test harness.

---

## Approved design — full flow, one atomic close

TL approved this design in receipt34e0ac39-2479-4ee2-aa83-463034e5d3d1; credential prohibition finalized in bed8618a. Space released and #556 landed as bcf734588, now merged. No implementation before change-code plan-quality gate.

**Goal:** Resolve supported local Codex indirect resumes from source-faithful read-only evidence once, reserve the exact ID, and hand off explicit argv in the same physical namespace. Keep unresolved native UX working through #556's one-line compatibility path.

**Architecture:** A pure configuration/selector projection consumes explicit invocation inputs and typed file/metadata observations. Thin inventory-owned readers collect config, native metadata and read-only SQLite evidence. Reuse #556's resolver, argv pinning, reservation and wrapper checks; do not add another admission engine or native-path scanner.

**Tech stack:** Go, a real TOML parser (no regex/line parser; pin a maintained Go TOML module at implementation after checking dependency policy), existing sessioninventory Runtime/RecoveryRuntime and SQLite adapter, private real SQLite/config/transcript fixtures, and native read-only macOS managed-preference observation.

### Core concepts

| Pure entity | Planned location | Status / responsibility |
|---|---|---|
| CodexInvocationConfig | cmd/internal/sessioninventory/codex_config.go | new; root/subcommand config overrides, profile, effective cwd, mode, relevant env and feature/provider flags, without secrets |
| CodexConfigResolution | cmd/internal/sessioninventory/codex_config.go | new; resolved physical transcript home, SQLite home, provider/history-provider and feature scope, or a typed unsupported/invalid/incomplete reason |
| NativeSelectorConfig / selector query | cmd/internal/sessioninventory/native_selector.go | modified after #556; carry resolved scope/mode rather than assuming a default home or one latest algorithm |
| Native admission scope | cmd/internal/launcher/native_createflow.go and matching wrapcmd handoff | modified only as needed after #556; bind invocation cwd/effective namespace as well as exact executable/argv/ID |

| Integration | Planned location | Wraps |
|---|---|---|
| CodexConfigReader | cmd/internal/sessioninventory/codex_config_os.go | new thin bounded filesystem/env/platform observations; no Codex startup, network config fetch, auth refresh or writes |
| Scoped inventory runtime | cmd/internal/sessioninventory/runtime_os.go | modified canonical root owner; resolved CODEX_HOME separate from SQLite home; preserve recovery interfaces/cache/work observers |
| Codex latest evidence | cmd/internal/sessioninventory/native_selector_collect.go | modified #556 real read-only SQLite/transcript seam; separate TUI and exec algorithms |
| Invocation/admission wiring | cmd/internal/launcher/native_createflow_os.go; cmd/internal/wrapcmd/native_admission.go | reuse #556 request/claim/pinning; effect-time scope validation with final command environment |

ARCH-DRY: native roots stay in inventory, argv pinning and writer admission stay in #556. ARCH-PURE: file loading is separate from precedence/selection; pure tests use typed values, integration tests use real temporary files/DBs.

### Configuration and selector contract

- Resolve CODEX_HOME exactly as0.162.1: empty/unset defaults to HOME/.codex; nonempty must exist as a directory and canonicalizes physically. Do not trim/expand the env value as though it were sqlite_home. Keep physical transcript root separate from SQLite directory.
- Read base config.toml and selected NAME.config.toml using profile-v2 rules; missing required profile, malformed/unreadable TOML, legacy profile selectors, invalid types and unavailable custom roots cannot silently become defaults. Parse CLI TOML dotted overrides with native raw-string fallback; preserve root-before-subcommand precedence and flag/prompt boundaries.
- Resolve paths relative to the originating config directory before merge; CLI and trimmed CODEX_SQLITE_HOME relative to effective cwd. SQLite precedence: exact requirement, merged configured value, env, Codex home. Provider: required, invocation/OSS selection, merged value, openai; history-provider projection follows the TUI source rather than assuming every invocation shares config.model_provider.
- Include source-backed system/local project layers and trust/root discovery. Project provider keys are ignored by native and must not influence selection; trusted project sqlite_home/features may influence it. Worktrees defaults ON: reproduce corresponding existing cwd sets from validated current/primary/linked checkout metadata, including subdirectory and canonical-path behavior; --all removes the cwd predicate.
- Detect macOS forced managed values with the source's CoreFoundation forced-preference checks, not plist existence/defaults-read. Relevant unmanaged/unsupported requirement, remote, cloud, packaged or invocation layers must remain explicit uncertainty. Implement local fields only where all material layers are observable; otherwise preserve compatibility with a reason. Do not mistake absent cloud cache for absent cloud policy. Auth and credential files are forbidden inputs, including classification-only reads; do not inspect credentials, keychains, auth metadata or secret environment values to decide eligibility. If non-credential evidence cannot rule out a material cloud layer, classify configuration as cloud-unknown and preserve native pass-through with one one-line not-guarded note. Cache absence is insufficient. No remote configuration requests or fabricated local-only mode. This applies to the current machine: the measurements below do not establish cloud absence or guarded support for its ordinary invocation.
- **TUI latest:** active, nonempty preview, native interactive sources (plus correct exec/mcp mapping for include-non-interactive), history provider and effective worktree/all cwd scope, UpdatedAt descending. Validate first chosen native row's actual path/root/ID. Do not scan past an unusable limit1 native result to select a different guarded thread; preserve native repair fallback instead.
- **Exec/e latest:** distinct source algorithm: UpdatedAt pages of100, all native source kinds, provider filter, no SQL cwd restriction. Validate state-only session metadata, then compare last turn_context.cwd from reverse rollout scan (fallback thread.cwd) with normalized invocation cwd; --all bypasses comparison. Do not route this through TUI source/cwd filters. Bound reads with #556's existing recovery budgets; exhausted/incomplete evidence uses compatibility, never a lower candidate that may be wrong.
- Do not invent ID ordering on equal UpdatedAt maxima. Prove a unique winning result across the relevant candidate boundary or keep compatibility. Explicit IDs retain #556 admission; bare picker, unresolved names/prefixes, remote/fork/new-conversation semantics retain their existing native path unless separately proven faithful. No-match/incomplete DB evidence must preserve scan-and-repair/StartFresh behavior through native passthrough; the adapter itself does neither repair nor native startup.
- For a complete resolution, reserve agent+canonical ID, reuse PinNativeResumeSelector, preserve unrelated arguments/prompt/defaults, and verify the same physical namespace/cwd at wrapper handoff. Namespace/config changes may invalidate the attempt; never rerun indirect selection after granting it. No newer DB row may change the granted ID.

### Operator-visible projection and authority

| Internal observation | Person sees | One next action |
|---|---|---|
| Complete faithful selection, reservation granted | Ordinary resume of the pinned conversation | Continue working |
| Another admitted writer owns that ID | Existing #556 busy/refusal with its conversation identity | End the other writer and retry |
| Cloud-unknown, unsupported, ambiguous, incomplete, no match, or unresolved config layer | Native resume/picker continues with exactly one not-guarded compatibility reason | Use native UI as usual |
| Namespace changed after reservation / uncertain grant handoff | Existing actionable failed attempt; no native child starts with a mismatched grant | Retry the launch |

ARCH-ORDER: resolution -> reservation -> explicit command -> scope/grant claim -> native spawn, never a second latest lookup. Reuse #556's state transitions and release/join ownership for failure/cancel/child-exit. A late read from an abandoned attempt cannot publish a reservation; a grant does not turn incomplete scope into authority.

### Implementation sequence (after both blockers clear)

- [ ] Start the issue branch from main after space is released; wait for #556 to land before importing/wiring its API. Materialize the canonical durable plan, obtain plan-quality approval and derive the estimate before code.
- [ ] Add pure table tests in codex_config_test.go for home/env distinctions, profile-v2, legacy rejection, file-vs-CLI relative paths, repeat/dotted overrides, root/subcommand precedence, provider/OSS flags, native defaults, project filtering/trust, requirements and explicit unsupported classifications. Include same-value fields from different path origins and unreadable/changed files.
- [ ] RED then implement the small config reader/projection and scoped runtime constructor. Test private config trees and metadata; platform seam uses a stateful forced-preference fake and source-backed absence observations. Tests must not read real HOME/auth/managed state. Add a recording file reader and synthetic forbidden-path/secret-env sentinels: cloud-unknown must pass through unchanged, report exactly one not-guarded note, make zero auth/credential/keychain reads and acquire no reservation. Cache absence must remain unknown. Resolved synthetic cases require non-credential evidence that all material layers are accounted for; never insert an eligibility boolean with no production provenance. Share the native policy projection with any CLI parse consumers.
- [ ] Add real private SQLite/transcript fixtures for TUI latest and exec latest separately: provider/source/cwd/worktree/all filters, negative rows, tied maxima, malformed/out-of-root/path-ID mismatch, differing last turn cwd, stale/deleted rows, missing or incompatible DBs, newer arrivals and scan budget exhaustion. RED then adapt #556's collector and preserve fallback. Canonical root tests include custom CODEX_HOME and independently configured sqlite_home plus symlink aliases.
- [ ] Add launcher/wrapper integration cases: resolved ID reserved once, exact pinned argv/prompt/environment preserved, scope checked at handoff, newer candidate cannot retarget, two concurrent attempts produce one admitted writer, canceled/late result cannot spawn, changed namespace invalidates rather than redirects. Extend existing #556 fixtures rather than inventing a second process harness. Compatibility tests explicitly preserve native picker/name/no-match/unsupported behavior and one notice; remove that notice only from genuinely guarded successes.
- [ ] Rerun the source-derived real-machine comparison against the implemented adapter using private snapshot copies/aggregate diagnostics; no native startup, resume, repair, defaults edits or credential output. Compare both TUI effective worktree scope and exec latest-turn cwd, not merely a flat cwd SQL fragment. Coordinate any live acceptance through the hub.
- [ ] Update atlas/session-inventory.md (or its current indexed recovery page), atlas/couch.md and atlas/index.md for source-version support, selection/namespace flow and honest compatibility limits. Keep README claims within demonstrated support.
- [ ] Run focused inventory/launcher/wrapper normal and race tests, then the recorded full suite through the machine gate; finish failed recipes with make -k and name all failures. Close through binary fresh review, merge main, land on SHIP and report exact hashes/PR to hub.

### Operating envelope, privacy and residue

ARCH-CONSTRAINTS: this is one explicit launch attempt, never a UI tick/peer-message scan. Reuse #556's context, per-read/output/candidate budgets; cap each config/metadata read and total traversal with an explicit unsupported result on exhaustion, preserving native UX. Do not cache native selection as enduring authority or add background refresh. Fix a measured safe budget in the durable plan from #556's landed APIs before implementation; no timing-based correctness assertions.

ARCH-SECURE: config, DB rows, transcript paths and filesystem metadata cross native/version boundaries. Decode to typed values, preserve origin, reject path escape and mismatched identities, retain current no-follow/read-budget controls. Use an explicit allowlist of required configuration/native-evidence inputs; no directory scan or symlink resolution may open an auth/credential file. Never access keychains or classify secret environment variables. Keep diagnostics to field names/reasons, never config contents, prompts, auth tokens or raw records. Namespace observation is not writer admission. Real-machine validation opens SQLite only on private copies; native files are read-only inputs.

ARCH-MOCK: the external shape is a portable private directory with real TOML, real SQLite and native JSONL/metadata, plus a controlled platform-settings reader; use stateful fixtures and mutation between observations. No function-call fake claiming to emulate Codex selection. Read-only real-machine/source conformance is required in addition to these fixtures; no app-server oracle.

ARCH-FUNERAL: production creates no new durable artifact/store/process; invocation snapshots die with the attempt and #556 owns reservation/child teardown. Measurement snapshots are private temporary copies removed after closing handles. No copied real DB, tokens or raw native records enter Git.

### Alternatives considered

1. Recommended: source-derived file/config projection plus bounded native DB/transcript reads, wired into existing admission. Explicit compatibility where provenance is incomplete preserves UX without claiming guard coverage.
2. App-server/config RPC oracle: rejected; #556 measured startup writes/migrations, omitted defaults/env and runtime-profile incompatibility.
3. Transcript mtime/newest-file heuristic or a TUI query reused for exec: rejected; wrong filters/order/cwd and no faithful canonical selection.



## Landed integration and concrete bounds

- #556 ResolveNativeResume accepts NativeResumeRequest{Agent,CWD,Selector,Config}; OSRuntime.ResolveNativeResume is the config-loading boundary. Keep ParseNativeResumeSelector/PinNativeResumeSelector as the only argv authority.
- createflow.go currently skips resolution for every Codex indirect selector. Change it to offer latest resolution first, preserving compatibility only on non-resolved outcomes (and existing picker/name handling); strict ResumeRequired/recovery must never silently downgrade.
- NativeSelectorRequest gains Codex mode/scope from a typed config projection; native_selector_collect.go dispatches TUI versus exec. Keep SelectNativeLatest's existing agent/cwd checks; collector candidates project a proven effective cwd rather than weakening other agents' selection policy.
- Resolve physical roots only once per invocation using inventory-owned runtime construction; preserve Runtime, RecoveryRuntime, QueryNativeSelectorSQLite and work/cache interfaces. Do not make UI activity constructors parse config per tick.
- NativeAdmissionRequest currently binds only exact executable/argv, not cwd or native environment. Carry a private scope projection through the existing encoded invocation/handoff and require wrapper validation before Claim/spawn; source files/config generation changes yield a retryable refused attempt. Never let a missing scope weaken old explicit grants or require scope for unrelated agents.
- Configuration limits:1MiB per TOML file,8MiB aggregate,64 ancestor/project layers,64KiB per Git metadata file. These are compatibility bounds, not claims of native limits; exceeding them stays native. No glob scan of CODEX_HOME; only explicitly selected config/profile and known noncredential policy files.
- Existing RecoveryScanTimeout30s and shared RecoveryTotalReadLimit256MiB bound selector IO; metadata entries<=10000. Exec pages100, at most10000 rows under the same context; reverse-read chunks<=existing readChunkSize, at most32MiB per transcript charged to total. The real-machine16MiB control motivates a bound above1MiB; budgets never imply absence or allow skipping unknown higher-ranked candidates.
- SQLite output<=1MiB per bounded page. Parameterize or use the existing quote boundary for query values; no process shell. Inspect exact native schema and never migrate/repair. No sidecar writes to native state introduced by configuration discovery.
- Pure projection API: input contains invocation cwd, home and only allowed environment values, parsed ordered CLI fields, and typed config layers with origin directories. Output is tagged parsed/invalid/unsupported/incomplete configuration plus minimal provider, physical homes, mode/worktree features and policy coverage; no raw TOML, arbitrary env or credentials survive projection.
- File reader seam exposes only bounded named config/metadata observations, canonical-directory resolution and managed-policy presence. Stateful private fixture mutation covers missing/unreadable/changed inputs and forced settings. Cloud policy defaults Unknown; no permissive zero value or test-only production authorizer.

## Verification commands

Run after each affected TDD step:
- go test ./cmd/internal/sessioninventory -run 'TestCodexConfig|TestCodexLatest|TestNativeSelectorCodex' -count=1
- go test ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'Test.*(Codex|NativeResume|NativeAdmission|NativeCreate)' -count=1
- go test -race ./cmd/internal/sessioninventory ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'TestCodexConfig|TestCodexLatest|TestNativeSelectorCodex|Test.*Native(Create|Resume|Admission)' -count=1
- go test ./cmd/internal/artifactpath ./cmd/internal/sessioninventory -run 'TestNoNativeAuthorityShadow|TestNativeAuthorityShadowSweep|Test.*Inventory|Test.*Manifest' -count=1
Then sdlc pr; sdlc local-check run --name suite; sdlc close with exact evidence; merge main and sdlc merge --yes. No duplicate fresh review or bypass.

## Revisions

### 2026-10-15 — materialized after #556 landing

Transferred the approved issue design to its canonical durable path after the two prerequisites cleared. Added concrete landed integration points and bounded-IO tests. No credential classification, app-server oracle, remote configuration fetch, new stored defaults or native repair. All real-machine source-query measurements remain conditional on known local inputs, not a claim of observed cloud absence.
