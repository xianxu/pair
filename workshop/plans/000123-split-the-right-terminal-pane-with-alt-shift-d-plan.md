# Split Right Terminal Pane Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `Alt+Shift+d` as a terminal-local layout-3 shortcut that creates a Zellij top/bottom split in the right terminal area and focuses the new lower pane.

**Architecture:** Reuse the existing terminal-local shortcut pipeline (`workbenchshortcut` → `pair term` stdin pump → injected `Runtime.RunZellijAction`) so the behavior stays inside the current shortcut ownership model (`ARCH-DRY`). Use Zellij-native panes for the split, not Pair's internal terminal-tab mux, so Zellij owns mouse boundary resizing (`ARCH-PURPOSE`). The split action must create the same Pair terminal command shape as `zellij/layouts/main-3.kdl`: `zellij action new-pane --direction down --name terminal -- sh -c 'zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term'`.

**Tech Stack:** Go terminal command routing, generated workbench shortcut registry, Zellij KDL config/layouts, shell integration tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `ChordAltShiftD` | `cmd/internal/workbenchshortcut/shortcut.go` | new |

- **ChordAltShiftD** — canonical representation of the `Alt+Shift+d` byte sequence.
  - **Relationships:** N:1 with terminal routing tests and generated Neovim action metadata.
  - **DRY rationale:** Keeps shortcut bytes in the existing registry instead of duplicating raw escape sequences in each consumer.
  - **Future extensions:** Other right-pane management shortcuts should join this registry.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `TerminalSplitDownAction` | `cmd/internal/termcmd/run.go` | new | `zellij action new-pane` |
| `ZellijMouseResizeConfig` | `zellij/config.kdl` | modified | Zellij mouse pane resize behavior |

- **TerminalSplitDownAction** — terminal-local handler that invokes Zellij to split the focused right terminal pane downward.
  - **Injected into:** `pumpStdinWithTimer` through the existing `Runtime` fake.
  - **Action contract:** `RunZellijAction("new-pane", "--direction", "down", "--name", "terminal", "--", "sh", "-c", rightTerminalPaneShell)`.
  - **Command contract:** `rightTerminalPaneShell` is the same shell string used by layout 3's right terminal pane: `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`.
  - **Mouse contract:** do not pass `--borderless true`; Zellij pane borders must remain available for mouse boundary dragging.
  - **Future extensions:** Adjacent right-pane management actions such as close split or move focus between split panes.
- **ZellijMouseResizeConfig** — the minimal config needed so real Zellij pane boundaries remain mouse-draggable.
  - **Injected into:** Zellij at session start through the existing config file.
  - **Config contract:** do not set `mouse_mode false`; keep `focus_follows_mouse false`. `advanced_mouse_actions false` may remain because the default config documents it as hover/grouping behavior, not basic pane-boundary resizing.
  - **Future extensions:** Only widen if a live smoke proves boundary drag still cannot work.

## Task 1: Pin Shortcut Routing

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Modify: `cmd/internal/workbenchshortcut/shortcut_test.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write failing registry/routing tests**

Add coverage that `Alt+Shift+d` decodes through the shared shortcut registry and that `pair term` maps it to `RunZellijAction("new-pane", "--direction", "down", "--name", "terminal", "--", "sh", "-c", rightTerminalPaneShell)`.

- [x] **Step 2: Verify RED**

Run:

```bash
go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1
bash tests/term-pane-shortcuts-test.sh
```

Expected: fail because the chord/action is not registered or routed.

- [x] **Step 3: Implement minimal routing**

Add the chord to the registry, define the right-terminal shell command once in `cmd/internal/termcmd/run.go`, and route it in `handleTerminalChord` to the injected Zellij runtime action.

- [x] **Step 4: Verify GREEN**

Run the same commands. Expected: pass.

## Task 2: Preserve Layout And Mouse Resize Behavior

**Files:**
- Modify: `zellij/config.kdl`
- Modify: `zellij/layouts/main-3.kdl` if the split action needs a command-compatible terminal pane shape.
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write failing config/layout assertions**

Add shell assertions that `Alt+Shift+d` is terminal-local, the split action creates a named `pair term` pane rather than a raw shell, no split pane is borderless, and the config leaves Zellij pane boundary dragging enabled without enabling focus-follows-mouse.

- [x] **Step 2: Verify RED**

Run:

```bash
bash tests/term-pane-shortcuts-test.sh
zellij --config-dir zellij setup --check
```

Expected: fail until the config/action is updated.

- [x] **Step 3: Implement minimal config/layout changes**

Prefer Zellij's normal pane splitting and mouse boundary resizing. Keep `focus_follows_mouse false`, keep the new pane bordered, and avoid setting `mouse_mode false`.

- [x] **Step 4: Verify GREEN**

Run the same commands. Expected: pass.

## Task 3: Docs And Final Verification

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000123-split-the-right-terminal-pane-with-alt-shift-d.md`

- [x] **Step 1: Document the keybinding**

Add the new `Alt+Shift+d` right-terminal split behavior to README and atlas.

- [x] **Step 2: Run complete verification**

Run:

```bash
go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
zellij --config-dir zellij setup --check
git diff --check
```

Expected: all pass.

- [x] **Step 3: Record evidence and commit**

Update the issue log with test evidence and commit the implementation.
