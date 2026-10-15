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
