---
id: 000571
status: open
deps: [pair#556]
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '642f9229e8bf6e0de143455cdf204533113a9a76' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:22:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Resolve Codex indirect resumes from faithful read-only configuration

## Problem

#556 M2 adds native conversation admission. Codex indirect resume selectors retain the native CLI's existing behavior, unreserved with a one-line notice, until faithful resolution can pin one exact conversation without a UX regression. The operator requires the three primary agents (Claude, Codex, Qoder) to keep working.

## Spec

Build a faithful read-only Codex latest/indirect selector adapter. Read effective configuration from files and documented source-backed precedence: CODEX_HOME (default ~/.codex), config.toml model_provider/sqlite_home, runtime profiles, CLI config overrides and relevant environment overrides. Do not start app-server as a configuration oracle: measurement proved startup initializes/migrates metadata and lacks runtime-profile parity. Honor the effective physical Codex transcript root as well as SQLite home; current inventory roots hardcode HOME/.codex/sessions.

Use the matching Codex source's selector semantics and existing native inventory: exact cwd/agent, provider/source filters, native DB ordering, path/ID validation and complete unique resolution. Resolve once, reserve that canonical ID, and execute an explicit ID so the native CLI cannot choose a newer/different conversation afterward. Preserve unrelated flags/prompts and saved defaults. Remove #556's unreserved compatibility exception only for selectors which the adapter handles faithfully; do not introduce a human UX regression or silently guess unsupported configuration.

Verification must include read-only comparison against the real DB/configuration on this machine, without starting/resuming any real conversation or changing native files/defaults. Headless tests use private real transcript/SQLite/config shapes, profile/env/CLI precedence, custom CODEX_HOME/sqlite_home, ties/incomplete/unsupported scopes, newer arrivals after selection, and exact-ID admission at wrapper handoff. Coordinate any live acceptance through the hub/operator.

## Done when

- Ordinary operator Codex resume --last and other supported indirect selectors resolve faithfully to the same canonical ID the native CLI would select, then use native admission with explicit pinned argv.
- Runtime profile, CLI/env precedence and custom native/state homes follow verified source semantics; unsupported mappings are honest and do not silently choose a different conversation.
- Read-only real-machine comparison and headless tests establish ordering/filter/path correctness, no native/config/default mutations, and no two live writers for the resolved ID.
- The one-line unguarded compatibility notice is removed where faithful guarded resolution is available; any remaining limitations are documented.

## Plan

Start from main; do not stack on #556's unlanded branch. Planning/source measurement can begin immediately; integrate the admission seams after #556 lands.

### Final plan for TL review — full flow, one atomic close

Use the writing-plans structure here while issue-branch preparation is paused. After space is released, run start-plan --issue571 and copy the approved design to the canonical workshop/plans/000571-codex-indirect-resume-adapter-plan.md before change-code. No implementation or estimate before plan approval/quality review. Do not stack on #556.

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


## Log

Operator decision relayed by ops:0, verified receipt b927c539-6b36-4710-b39e-8b9142e1721b. Split unassigned from #556. Existing evidence is on #556's branch in workshop/plans/000556-codex-config-probe.md and its plan revisions. Private config/read experiments establish that API is merged configuration, omits defaults/env and rejects runtime profiles; this follow-up explicitly uses read-only files instead.

### 2026-10-15 — read-only planning and machine measurement

Claimed by pair:7. Ran bare sdlc start-plan to load architectural principles; deferred start-plan --issue because its automatic dependency refresh compiles, and the operator has not released the disk/build restriction. The local Ariadne checkout trails its remote, so this is a real refresh risk. Shape and publish this record from main under the claim; do not commit on the resting branch. No builds, suites, Codex app-server or real resume started.

Installed npm metadata identifies Codex0.162.1. Read matching upstream rust-v0.162.1 source into bounded scratch files, plus #556's explicitly referenced probe and admission branch using git show only. Findings: profile-v2 uses NAME.config.toml above config.toml; legacy profile/[profiles.NAME] is not the runtime-profile implementation. Configured sqlite_home beats CODEX_SQLITE_HOME; defaults are Codex home and provider openai. Custom CODEX_HOME canonicalizes an existing directory and changes physical transcript scope independently of sqlite_home. Trusted project config may set sqlite_home but cannot set provider; higher managed requirements/layers must not be silently ignored.

Private real-DB measurement: cloned only state_5.sqlite and its WAL to a temporary directory on the main data volume using cp -c; source identity/size/mtime were stable before/after the bounded copy. SQLite opened only the copies in mode=ro/query_only; no source SHM opened or copied. The temporary snapshot was removed after measurement. Native config was parsed in memory, selecting no explicit provider/sqlite/profile override; CODEX_HOME and CODEX_SQLITE_HOME unset. The real DB snapshot contains2928 threads and59 migrations, with required selector columns. Interactive sources:214cli,22vscode;1696exec;996other structured. Two providers present. Source-derived latest query in this slot and main checkout each found a unique timestamp winner; each existing rollout lies in the selected native root and its session_meta ID agrees with the DB. No contents, full IDs, credentials or native path listings retained in this record. This compares source-derived query behavior with the real DB, not a claim to have run native selection or the future adapter.

Matching TUI latest requests state-DB-only first, UpdatedAt, limit1, active/visible threads and provider/source/cwd filters. It may fall back to scan-and-repair and eventually StartFresh. We must retain native passthrough when read-only evidence is absent/incomplete rather than fabricate absence. UpdatedAt with no relation does NOT request an ID tie breaker: equal maxima cannot be assigned invented ID ordering. A research suggestion to add ID DESC was rejected after inspecting the conditional in state/runtime/threads.rs:1270-1275,1579-1583.

Source links (matching tag):
- [Config/profile layering](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/config/src/loader/mod.rs#L270)
- [SQLite precedence](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/core/src/config/mod.rs#L4112)
- [CODEX_HOME](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/utils/home-dir/src/lib.rs#L13)
- [TUI latest selection](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/lib.rs#L823)
- [Conditional ordering](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/runtime/threads.rs#L1269)
- [History provider selection](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/app_server_session/provider_selection.rs#L40)
- [Official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)

## Revisions

### 2026-10-15 — planning under the disk restriction

Append source-backed configuration/selector measurements and a proposal before implementation. No architecture or implementation gate is waived: issue-branch preparation, canonical durable plan and change-code wait until compilation is permitted; admission integration waits for #556 on main. Preserve native passthrough for unresolved scopes, picker/names, incomplete DB evidence and nondeterministic ties instead of changing ordinary resume behavior into a refusal.

### 2026-10-15 — selector/config scope refinement

The earlier flat-cwd real-DB query is only a lower-level shape/path/ID control, NOT full native selection parity: default Worktrees widens TUI scope, while exec selects from all sources and uses latest turn_context.cwd. Plan now explicitly models both. Native UpdatedAt-only ties remain compatibility; no invented ID ordering. Config source confirms system/managed files and forced MDM values absent here; cloud-cache absence alone is insufficient, so nonsecret eligibility classification is pending the hub's answer. This proposed fallback is a fidelity boundary, never a claim that cloud policy was observed.

### 2026-10-15 — source-derived effective-scope controls

Expanded the private-snapshot measurement to the ordinary Worktrees-on scope:24 existing registered checkouts cross-checked against the same common Git directory, source-derived active/visible/interactive/provider query, unique timestamp maximum and an existing winning rollout. Independently followed exec's all-source provider query and reverse latest-turn-context cwd: its first candidate was usable, matching current cwd and the same digest-only ID. A1MiB exploratory tail bound was incomplete; a16MiB bounded read found the latest turn context. This is measurement of read budget/fidelity, not permission to ignore an exhausted budget in production. No native selector/app-server was executed; cloud eligibility remains unproved, so these are conditional local-provider evidence controls rather than complete runtime-configuration conformance.

Scratch Python measurement scripts used only stdlib/file reads plus owned temporary clone snapshots and read-only Git observations; no builds or suites. Snapshot removed on context exit. The draft plan and all discoveries are published in the issue record while branch preparation remains paused; canonical durable plan will be materialized on the issue branch before change-code.

### 2026-10-15 — final cloud and credential boundary

TL decision bed8618a-f832-428c-9b32-e697bf43e0e0: do not open Codex auth or credential files, even for classification. Updated the active plan to make them forbidden inputs and remove the pending request. Cloud eligibility is unknown unless source-backed non-credential evidence rules out a material cloud layer; unknown invokes native compatibility unchanged, with exactly one not-guarded note and no reservation. Neither an absent cloud cache nor a successful source-derived query establishes guarded support. No auth/credential file was opened during this item's measurements.

Final plan retains separate TUI and exec latest algorithms, NAME.config.toml profiles, configured sqlite_home ahead of CODEX_SQLITE_HOME, default worktree-inclusive TUI scope, isolated physical native/state homes, explicit-ID pinning and namespace checks through #556 admission. Added tests proving forbidden inputs are not touched and cloud-unknown never gains admission authority. Build/refresh/test restriction remains in force; branch preparation/durable-plan materialization waits for space and integration waits for #556 on main.
