---
id: 000067
status: done
deps: []
github_issue:
created: 2026-06-22
updated: 2026-06-23
estimate_hours: 1.8
started: 2026-06-23T00:13:50-07:00
actual_hours: 0.81
---

# Fix Codex pair tag quit and resume

## Problem

`Alt+x` can leave a zellij resurrect entry such as `pair-2 (EXITED - attach to resurrect)`.
Pair skips EXITED rows in the picker, but forced resume and name-collision checks
still treat the same row as occupying the tag. The next `pair` launch then cannot
reuse the workspace tag even though the user intended a full quit.

Separately, Codex resume can fail to surface for older sessions. Pair's current
canonical config name is `config-<tag>-codex.json`, but this machine has older
Codex configs such as `config-2-codex-codex.json`. When the canonical file is
absent, Pair misses the saved Codex session even though the state exists.

## Spec

Pair tags are workspace identities. A live zellij session name remains globally
`pair-<tag>` regardless of agent; there is not a simultaneous `pair-1` for Claude
and another `pair-1` for Codex. Agent-specific native session state remains
stored below the workspace tag.

When Pair is about to decide whether a `pair-<tag>` zellij session blocks reuse,
it should treat EXITED resurrect entries as stale full-quit residue: delete the
zellij session record and continue as if no live session exists. This applies to
forced `pair resume <tag>`, interactive name collision checks, and the free-slot
scanner. Detached/running sessions still block or attach as before.

When Pair needs the saved config for `(tag, agent)`, it should first use the
canonical `config-<tag>-<agent>.json`. If absent, it may migrate a legacy Codex
shape `config-<tag>-codex-codex.json` to `config-<tag>-codex.json` when the JSON
declares `"agent": "codex"`. This is a narrow compatibility path, not a general
glob-based config resolver, so unrelated stale files cannot silently win
(`ARCH-DRY`, `ARCH-PURE`).

## Done when

- `pair resume 2` no longer attaches/refuses just because zellij lists
  `pair-2` as EXITED; Pair deletes the stale zellij record and proceeds to
  create/resume by tag.
- An old `config-<tag>-codex-codex.json` is migrated or recognized so the Codex
  saved-session picker can offer native resume.
- Running/detached `pair-<tag>` sessions still block name reuse or attach.
- Tests cover stale EXITED cleanup and legacy Codex config migration.

## Estimate

```estimate
model: estimate-logic-v2
familiarity: 1.0
item: issue-spec          design=0.2 impl=0.2
item: method-b-decisions  design=0.4 impl=0.8
design-buffer: 0.30
total: 1.8
```

## Plan

- [x] Add shell-test seams for zellij session state and config migration.
- [x] Implement stale EXITED cleanup through one helper used by collision paths.
- [x] Implement narrow legacy Codex config migration through one helper used by
      config lookup paths.
- [x] Run focused tests, then the relevant full test suite.

## Log

### 2026-06-22

### 2026-06-23
- 2026-06-23: closed — Reverified local red state, restored #67 bin/pair helpers and early inference/debug placement, then passed bash -n bin/pair bin/pair-session-watch.sh bin/pair-quit.sh; bash tests/pair-continue-test.sh; bash tests/pair-rename.sh; env -u PAIR_SESSION_ID -u PAIR_TAG make test. No atlas update: restored existing #67 architecture only.; review verdict: FIX-THEN-SHIP
- 2026-06-24: reopened — local `main` was left at commit `6aedcf1`, which
  reverted #67's `bin/pair` implementation while keeping the #67 tests. Fresh
  verification with `bash tests/pair-continue-test.sh` failed 11 cases,
  including the stale EXITED cleanup, legacy Codex config migration, and
  continue/debug probe placement cases.
- 2026-06-24: restored the missing #67 `bin/pair` implementation on local
  `main`: `session_blocks_reuse` now deletes stale EXITED resurrect rows before
  tag-reuse decisions, `resolve_config_file` migrates verified legacy Codex
  configs, and forced-tag agent inference / `PAIR_DEBUG_ARGS` run before the
  early zellij guards. Verification: `bash -n bin/pair bin/pair-session-watch.sh
  bin/pair-quit.sh`; `bash tests/pair-continue-test.sh` (PASS after the earlier
  11-failure red run); `bash tests/pair-rename.sh` (57 passed, 0 failed);
  `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (PASS).
- 2026-06-24: boundary review returned `FIX-THEN-SHIP`: logic is sound, but the
  close commit must include the restored `bin/pair` implementation and the plan
  should acknowledge the early forced-tag agent-inference / `PAIR_DEBUG_ARGS`
  relocation. Added the plan revision; staging will include `bin/pair`.

- 2026-06-23: closed — session_blocks_reuse deletes stale EXITED resurrect rows so a fully-quit pair tag is reusable (running/detached still block); resolve_config_file migrates legacy config-<tag>-codex-codex.json -> canonical so Codex native resume surfaces in the picker. pair-continue-test (6 new cases) + pair-rename pass; full make test rc=0 in scrubbed env (PAIR_SESSION_ID/PAIR_TAG leak causes 2 false fails inside a live pair session).; review verdict: SHIP

- Root cause trace: zellij still lists `pair-2` as EXITED, while Pair skips
  EXITED rows in the picker but checks `list-sessions --short` for resume/name
  collisions. Same tag also has `agent-2=codex` and old config
  `config-2-codex-codex.json`, but no canonical `config-2-codex.json`.
- User confirmed tag semantics: "a pair tag is a workspace", so live zellij
  names stay global by tag; agent-specific recovery stays under the tag.
- Recovered after Codex crashed mid-implementation: it had landed the design
  + the failing tests (6 cases in `tests/pair-continue-test.sh`) but never
  implemented the `bin/pair` helpers. Picked up from red.
- Implemented `session_blocks_reuse <session>` (ARCH-DRY) above the
  `PAIR_TEST_CALL` dispatcher: reads `list-sessions --no-formatting`, deletes an
  EXITED resurrect row via `zj delete-session … --force` and reports the tag
  reusable; running/detached rows still block; absent never blocks. Wired into
  all four collision paths — forced resume, free-slot scan, name-prompt
  collision, cmux-owner liveness. `pair rename` left on its own offline-only
  contract per spec.
- Implemented `resolve_config_file <tag> <agent>` (ARCH-DRY/PURE): canonical
  `config-<tag>-<agent>.json` wins; when absent and `<agent>=codex`, migrates a
  legacy `config-<tag>-codex-codex.json` to canonical iff its JSON declares
  `"agent":"codex"`. Narrow agent-checked path, not a glob resolver. Wired into
  the 5 sites where (tag, agent) are both known — restart-marker read, cleanup
  resume hint, tag-restart picker (the path that surfaces the Codex resume),
  and the two config writes. Agent-inference glob loop deliberately untouched.
- Verification: `bash -n` clean; `tests/pair-continue-test.sh` PASS (6 new
  cases green); `tests/pair-rename.sh` PASS (57); full `make test` rc=0 in a
  scrubbed env. Two failures only appear when run inside a live pair session
  (`PAIR_SESSION_ID`/`PAIR_TAG` leak into `pair-review-target-test` +
  `changelog-open-test`); both PASS under `env -u PAIR_SESSION_ID -u PAIR_TAG`
  and reproduce identically on the pre-change `bin/pair`, so not regressions.
- Re-verified after resuming in a live Codex pane: `tests/pair-continue-test.sh`
  initially failed because its `PAIR_DEBUG_ARGS` probe sat below the zellij
  ancestry guard and exited before printing debug fields when the test itself
  ran inside zellij. Moved the existing debug probe (and the agent inference it
  depends on) above the in-session guards; production launch behavior is
  unchanged, but the test seam now works from a live pair pane. Verification:
  `bash -n bin/pair bin/pair-session-watch.sh bin/pair-quit.sh` PASS;
  `bash tests/pair-continue-test.sh` PASS; `env -u PAIR_SESSION_ID -u PAIR_TAG
  make test` PASS.
