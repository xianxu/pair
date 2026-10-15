---
id: 000571
status: open
deps: [pair#556]
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'baca79c953bd5ce9ac656a7a3ee4d184f2e8378e' # card fields mirrored from issue-cards; edit via sdlc
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
