# Codex Tag Quit Resume Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Alt+x` fully release a workspace tag and make old Codex saved sessions recoverable under the current config filename.

**Architecture:** Keep `pair-<tag>` as the live workspace identity; agent-specific state stays in `config-<tag>-<agent>.json`. Add one stale-zellij helper and one saved-config helper in `bin/pair` so collision checks and resume paths share logic (`ARCH-DRY`). The helpers live with the early test-dispatchable helpers above `PAIR_TEST_CALL`, remain shell-level IO seams driven by fake `zellij`/fixture files in tests, and keep decision logic out of scattered call sites (`ARCH-PURE`).

**Tech Stack:** Bash launcher (`bin/pair`), shell integration tests, zellij CLI fakes, jq.

---

## Core Concepts

| Name | Lives in | Status |
|------|----------|--------|
| `WorkspaceTag` | `bin/pair` | modified |
| `SavedAgentConfig` | `bin/pair` | modified |

**WorkspaceTag** — the user-facing pair tag; maps to one live zellij session `pair-<tag>`.
- **Relationships:** 1:1 with a live zellij session; 1:N with saved agent configs across agents.
- **DRY rationale:** Every collision path should use the same stale-EXITED cleanup behavior instead of open-coded `list-sessions --short` checks.
- **Future extensions:** Agent-specific live names would be a separate model change, not part of this fix.

**SavedAgentConfig** — JSON restart config for a workspace tag and agent.
- **Relationships:** N:1 with `WorkspaceTag`; one canonical file per `(tag, agent)`.
- **DRY rationale:** Agent inference, restart, and picker logic should all resolve the same config path.
- **Future extensions:** Additional legacy filename migrations can extend the same helper with explicit cases.

## Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `zellij session cleanup` | `bin/pair` | modified | `zellij list-sessions`, `zellij delete-session` |
| `pair data-dir config migration` | `bin/pair` | modified | `$PAIR_DATA_DIR/config-*.json` |

**zellij session cleanup** — deletes an EXITED resurrect record before tag-occupancy decisions.
- **Injected into:** Collision checks in `pair resume`, free-slot scan, and interactive name prompt.
- **Future extensions:** Could become a separate diagnostic/doctor repair operation.

**pair data-dir config migration** — promotes the legacy Codex config filename to the canonical filename when safe.
- **Injected into:** Agent inference and tag-restart picker config lookup.
- **Future extensions:** Add telemetry for migrations if this becomes common.

## Chunk 1: Launcher Helpers And Tests

### Task 1: Stale EXITED zellij records

**Files:**
- Modify: `bin/pair`
- Modify: `tests/pair-continue-test.sh`

- [x] **Step 1: Write failing tests**
  Add `PAIR_TEST_CALL` coverage that runs the new helper against a fake `zellij`.
  Cases: EXITED row is deleted and reports absent; running row remains occupied;
  detached/non-EXITED row remains occupied.

- [x] **Step 2: Verify red**
  Run: `bash tests/pair-continue-test.sh`
  Expected: FAIL because the helper does not exist yet.

- [x] **Step 3: Implement helper**
  Add `session_blocks_reuse <session>` in `bin/pair` above the `PAIR_TEST_CALL`
  dispatcher. It reads
  `zj list-sessions --no-formatting`, deletes the session when the matching row
  contains `EXITED`, and returns non-zero afterward. Otherwise it falls back to
  `zj list-sessions --short` exact-match occupancy.

- [x] **Step 4: Replace collision call sites**
  Use the helper in forced resume, free-slot scan, prompt-name collision, and
  cmux-owner liveness decisions where stale EXITED rows should not block reuse.
  Do not change `pair rename` in this issue; its documented offline-only
  resurrectable-session contract is separate.

- [x] **Step 5: Verify green**
  Run: `bash tests/pair-continue-test.sh`
  Expected: PASS.

### Task 2: Legacy Codex config migration

**Files:**
- Modify: `bin/pair`
- Modify: `tests/pair-continue-test.sh`

- [x] **Step 1: Write failing tests**
  Add `PAIR_TEST_CALL` coverage for `resolve_config_file <tag> codex`: canonical
  file wins when present; absent canonical plus `config-<tag>-codex-codex.json`
  with `"agent":"codex"` migrates to the canonical path; a non-Codex legacy file
  is ignored.

- [x] **Step 2: Verify red**
  Run: `bash tests/pair-continue-test.sh`
  Expected: FAIL because the helper does not exist yet.

- [x] **Step 3: Implement helper**
  Add `resolve_config_file <tag> <agent>` in `bin/pair` above the
  `PAIR_TEST_CALL` dispatcher. Echo the canonical path after optional migration.
  Keep the migration narrow to Codex and validate the JSON agent field with
  `jq`.

- [x] **Step 4: Replace config lookup call sites**
  Use the helper where `(tag, agent)` are both already known: restart marker
  handling, tag-restart picker, explicit resume writes, Claude fresh-session
  config writes, and cleanup resume hints. Do not use it for the agent-inference
  glob loop; that loop is discovering the agent and already sees legacy
  `config-<tag>-codex-codex.json` files.

- [x] **Step 5: Verify green**
  Run: `bash tests/pair-continue-test.sh`
  Expected: PASS.

### Task 3: Final verification

**Files:**
- Modify: `workshop/issues/000067-codex-tag-quit-resume.md`

- [x] **Step 1: Run shell syntax checks**
  Run: `bash -n bin/pair bin/pair-session-watch.sh bin/pair-quit.sh`
  Expected: no output.

- [x] **Step 2: Run focused suite**
  Run: `bash tests/pair-continue-test.sh && bash tests/pair-rename.sh`
  Expected: PASS.

- [x] **Step 3: Run broader suite**
  Run: `make test`
  Expected: PASS.

- [x] **Step 4: Update issue plan/log**
  Tick completed issue plan items and record verification commands.
