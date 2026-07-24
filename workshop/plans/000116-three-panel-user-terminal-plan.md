# Three-panel Pair Layout Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Pair open as a left-side agent/draft stack plus a right-side user terminal with pane-local workbench shortcuts.

**Architecture:** Keep zellij KDL responsible for static geometry, and move pane-local shortcuts out of global KDL binds into the focused pane processes: `pair wrap` handles agent-pane chords, draft nvim handles draft-pane chords, and a new `pair term` wrapper handles right-terminal chords. Reuse `cmd/internal/zellijpane` for focused-pane discovery where a pane needs to target another pane (`ARCH-DRY`), and keep shortcut decisions as pure functions with thin PTY/zellij IO shells (`ARCH-PURE`). The plan fulfills the issue by delivering left-only Pair shortcuts, right-only tab helpers, and `Alt+k` last-left-pane navigation (`ARCH-PURPOSE`).

**Tech Stack:** Go 1.x, `github.com/creack/pty`, `golang.org/x/term`, zellij KDL/actions 0.44.3, shell fake integration tests, runtime-bundle generator.

## Revisions

### 2026-07-24 — Preserve pane identity during width-toggle collapse

Live smoke testing showed that the initial floating-overlay implementation
removed the terminal frame and restored the three-pane shape with
`override-layout`, which could recreate panes and lose the running agent and
Neovim processes. The toggle now keeps the overlay framed and collapses by
embedding the same terminal pane, then uses pane-targeted resize actions until
the existing left and right panes are balanced. No layout is reapplied.

### 2026-07-24 — Preserve the tiled split tree in both directions

The embed-based collapse preserved pane identity but Zellij reinserted the
terminal at a different node in the split tree, leaving Neovim across the
bottom and agent/terminal across the top. Expansion and collapse now both keep
the terminal tiled and resize only its left boundary, targeting 2/3 and 1/2
width respectively. This preserves topology as well as pane identity.

### 2026-07-24 — Reconcile to the closest reachable width

Smoke testing showed that the original 5% acceptance tolerance stopped
collapse one Zellij resize step before the balanced split. Reconciliation now
uses a 1% target tolerance and, when the next discrete resize step is worse,
reverses that step so the pane remains at the closest reachable width.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `PaneRole` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ShortcutAction` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ShortcutDecision` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `LastLeftPaneStore` | `cmd/internal/workbenchshortcut/shortcut.go` | new |

- **PaneRole** — classifies a focused pane as left agent, left draft, right terminal, or other.
  - **Relationships:** 1:1 with a focused `zellijpane.Pane`; role is derived from `title`/`terminal_command`, not filesystem cwd.
  - **DRY rationale:** Reuses `cmd/internal/zellijpane.Parse` rather than duplicating list-panes JSON traversal.
  - **Future extensions:** Additional Pair-owned panes can add roles without changing the parser.

- **ShortcutAction** — the workbench action requested by a recognized chord.
  - **Relationships:** N:1 from terminal input sequences to one action; actions map to zellij commands only in the IO shell.
  - **DRY rationale:** One registry for the right-terminal and future pane wrappers.
  - **Future extensions:** Add more pane-local commands without changing PTY byte pumps.

- **ShortcutDecision** — pure decision result: pass bytes through, swallow, or run a zellij action.
  - **Relationships:** Depends on `PaneRole`, current last-left-pane id, and decoded input sequence.
  - **DRY rationale:** Keeps pane gating testable without spawning zellij or shells.
  - **Future extensions:** Persist more navigation state if Pair adds multiple user panes.

- **LastLeftPaneStore** — path contract and atomic read/write helpers for the last focused left Pair pane id.
  - **Relationships:** 1:1 with a Pair tag under `$PAIR_DATA_DIR/last-left-pane-<tag>`. Written by agent/draft local shortcut handlers before moving right; read by `pair term` before moving left.
  - **DRY rationale:** One shared sidecar path prevents `pair wrap`, nvim, and `pair term` from inventing incompatible state keys.
  - **Future extensions:** If Pair tracks more pane focus history, the file can become JSON with versioned fields.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair term` | `cmd/internal/termcmd/run.go` | new | PTY child shell + stdin/stdout |
| `TermRuntime` | `cmd/internal/termcmd/runtime.go` | new | zellij IPC + process exec |
| `pair wrap` shortcut hook | `cmd/internal/wrapcmd/wrap.go` | modified | agent-pane PTY stdin |
| `draft nvim` shortcut maps | `nvim/init.lua` | modified | draft-pane key mappings |
| `main.kdl` terminal pane | `zellij/layouts/main.kdl` | modified | zellij layout |
| `config.kdl` global unbinds | `zellij/config.kdl` | modified | zellij global keybinds |

- **pair term** — transparent wrapper for the right user shell. It starts `$SHELL` (fallback `/bin/sh`), forwards bytes, recognizes only the agreed workbench chords, and leaves all other shell/nvim input untouched.
  - **Injected into:** zellij layout command for the right terminal pane.
  - **Future extensions:** More pane-local helpers can be added to the pure shortcut registry.

- **TermRuntime** — side-effect seam for listing panes, focusing panes, and running zellij tab actions.
  - **Injected into:** `Run` so tests can use fakes for zellij and a fake child command.
  - **Future extensions:** Process-level fake can grow to cover live rename prompt behavior.

- **pair wrap shortcut hook** — recognizes only left-agent workbench chords before forwarding stdin to the agent PTY.
  - **Injected into:** existing `translateStdin` path in `wrapcmd`.
  - **Future extensions:** Any new left-agent shortcut should go through the shared pure shortcut decision layer, not ad hoc byte checks.

- **draft nvim shortcut maps** — local draft-pane mappings for `Alt+j`, `Alt+k`, `Alt+/`, `Alt+Shift+C` / `Ctrl+Alt+c`, and no-op `Alt+t/w/r`.
  - **Injected into:** `nvim/init.lua` alongside existing Pair layout/restart mappings.
  - **Future extensions:** If nvim Lua grows too large, extract a `nvim/workbench.lua` helper.

- **main.kdl terminal pane** — zellij static geometry: root left/right split; left child keeps agent over draft; right child runs `pair term`.
  - **Injected into:** launcher `--new-session-with-layout` flow already reading `zellij/layouts/main.kdl`.
  - **Future extensions:** Width percentage can be tuned without changing command wrappers.

- **config.kdl global unbinds** — zellij must release pane-local chords so the focused pane process can decide whether to handle or ignore them.
  - **Injected into:** zellij session config.
  - **Future extensions:** If zellij adds conditional pane binds later, this layer can own gating directly.

---

## Chunk 1: Pure Shortcut Model and Dispatcher Surface

### Task 1: Add `termcmd` pure shortcut decisions

**Files:**
- Create: `cmd/internal/workbenchshortcut/shortcut.go`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [ ] **Step 1: Write failing tests for role classification**

Cover:
- `pair wrap` command => left agent role.
- `nvim -u .../nvim/init.lua ...draft...` => left draft role.
- `pair term` command or `title == "terminal"` => right terminal role.
- focused floating/plugin panes => other role.

Run: `go test ./cmd/internal/workbenchshortcut -run TestPaneRole -count=1`
Expected: fails because package does not exist.

- [ ] **Step 2: Implement role classification**

Use `zellijpane.Pane` as input. Match `terminal_command` for stable commands, with title only as a fallback for the new terminal pane name. Avoid cwd-sensitive title matching.

- [ ] **Step 3: Write failing tests for shortcut decisions**

Cover:
- Right terminal: `Alt+t` => new tab, `Alt+w` => close tab, `Alt+r` => rename-tab prompt, `Alt+j` => swallow/no-op, `Alt+k` => focus last left pane or draft fallback.
- Left agent/draft: `Alt+j` toggles vertical focus, `Alt+k` records the focused left pane id and focuses right terminal, `Alt+t/w/r` are swallowed no-ops, `Alt+Shift+C`/`Alt+/` are allowed only left.
- Other/review/floating panes: `Alt+r` is not captured.

Run: `go test ./cmd/internal/workbenchshortcut -run TestShortcutDecision -count=1`
Expected: fails until decisions exist.

- [ ] **Step 4: Implement `ShortcutDecision`**

Represent decisions without IO: `Pass`, `Swallow`, `FocusPane(id)`, `MoveFocus(direction)`, `RecordLastLeftAndFocusTerminal`, `FocusLastLeftOrDraft`, `NewTab`, `CloseTab`, `RenameTabPrompt`, `OpenScrollback`, `Compact`. The decision receives current last-left id as input; persistence is owned by `LastLeftPaneStore`.

- [ ] **Step 5: Add sidecar path/read-write tests**

Cover:
- `LastLeftPath("/data", "work") == "/data/last-left-pane-work"`;
- atomic write then read returns the pane id;
- empty/missing file returns no id.

Run: `go test ./cmd/internal/workbenchshortcut -run TestLastLeft -count=1`
Expected: PASS after helpers exist.

- [ ] **Step 6: Run focused tests**

Run: `go test ./cmd/internal/workbenchshortcut -count=1`
Expected: PASS.

### Task 2: Register the `pair term` command

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/pair-go/main.go`
- Test: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Add failing dispatcher tests**

Extend `TestDispatchNamesDeriveFromImplementedStatus` and `TestStreamingFlags` to include `term`.

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatchNamesDeriveFromImplementedStatus|TestStreamingFlags' -count=1`
Expected: fails because `term` is not registered.

- [ ] **Step 2: Register `term` as an implemented streaming command**

Add `{Name: "term", Summary: "user terminal pane wrapper", Status: "implemented", Streaming: true}` and wire `runStreamingSubcommand("term", ...)` to `termcmd.Run`.

- [ ] **Step 3: Run dispatcher tests**

Run: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

---

## Chunk 2: Terminal Wrapper IO

### Task 3: Implement transparent terminal wrapper

**Files:**
- Create: `cmd/internal/termcmd/run.go`
- Create: `cmd/internal/termcmd/runtime.go`
- Test: `cmd/internal/termcmd/run_test.go`

- [ ] **Step 1: Write failing run-level tests with fakes**

Use a fake runtime that records zellij actions and a fake child command. Cover:
- no args starts `$SHELL`, fallback `/bin/sh`;
- unrecognized input bytes pass through to child;
- right-terminal `Alt+t/w/r` records `NewTab`, `CloseTab`, and an internal rename prompt followed by `RenameTab(name)`;
- right-terminal `Alt+j` is swallowed;
- right-terminal `Alt+k` focuses the recorded last-left pane, falling back to draft.
- agent-recorded and draft-recorded last-left ids are visible to a later terminal shortcut through the shared `$PAIR_DATA_DIR/last-left-pane-<tag>` sidecar.

Run: `go test ./cmd/internal/termcmd -run TestRun -count=1`
Expected: fails until runtime exists.

- [ ] **Step 2: Implement PTY wrapper**

Mirror the existing `wrapcmd`/`scribecmd` PTY pattern: `pty.Start`, raw stdin when stdin is an `*os.File`, SIGWINCH size propagation, stdin byte pump, stdout byte pump, child exit code propagation. Keep only shortcut decoding in the stdin pump; do not inspect or mutate child stdout.

- [ ] **Step 3: Implement zellij runtime actions**

Commands:
- list panes: `zellij action list-panes --json --command`
- focus pane: `zellij action focus-pane-id <id>` then `terminal_<id>` fallback if needed
- new tab: `zellij action new-tab`
- close tab: `zellij action close-tab`
- rename tab: `pair term` temporarily enters an internal line prompt in the terminal pane, then calls `zellij action rename-tab <name>` when the user presses Enter. `Esc` or an empty name cancels and restores normal pass-through. This avoids global KDL `TabNameInput` so review-pane `Alt+r` remains local.

- [ ] **Step 4: Run focused tests**

Run: `go test ./cmd/internal/termcmd -count=1`
Expected: PASS.

---

## Chunk 3: Zellij Layout and Key Routing

### Task 4: Update main layout to three panes

**Files:**
- Modify: `zellij/layouts/main.kdl`
- Generated: `cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main.kdl`

- [ ] **Step 1: Update layout**

Default geometry:
- root `split_direction="vertical"`
- left child `size="50%"` or equivalent containing the existing horizontal agent/draft stack
- right child `name="terminal"` running `sh -c 'exec pair term'`
- keep `focus=true` on draft for startup

Swap layouts:
- update `exact_panes=3`
- preserve right terminal leaf in every swap layout
- only vary draft height inside the left stack

- [ ] **Step 2: Validate layout**

Run: `zellij setup --dump-layout zellij/layouts/main.kdl`
Expected: exit 0.

### Task 5: Update config keybinds

**Files:**
- Modify: `zellij/config.kdl`
- Generated: `cmd/internal/runtimebundle/assets/runtime/files/zellij/config.kdl`
- Test: existing review tests plus any new shell test

- [ ] **Step 1: Release pane-local chords from global KDL**

Remove global Pair/zellij handling for pane-local chords that must be owned by focused pane processes: `Alt+j`, `Alt+k`, `Alt+t`, `Alt+w`, `Alt+r`, `Alt+/`, and `Alt C` / `Ctrl Alt c`. Keep non-pane-local Pair safety flows such as detach/quit/restart routed through draft nvim if they are still intended to work from every pane.

- [ ] **Step 2: Add agent-pane shortcut handling in `pair wrap`**

Intercept recognized left-agent chords before forwarding to the wrapped agent:
- `Alt+j`: focus draft.
- `Alt+k`: record the agent pane id as last-left, then focus right terminal.
- `Alt+/`: open scrollback viewer.
- `Alt C` / `Ctrl Alt c`: focus draft and invoke `PairConfirmCompact()`.

All other bytes pass through exactly as today.

- [ ] **Step 3: Add wrapcmd shortcut tests**

Extend `cmd/internal/wrapcmd/translate_test.go` or add a focused test file. Cover:
- recognized workbench chords are intercepted even when `sendKM` is empty, so no-Return-remap agents still get pane-local shortcuts;
- `Alt+t`, `Alt+w`, and `Alt+r` in the agent pane are swallowed no-ops and are not forwarded to the agent;
- ordinary text and existing Return/Alt+Return remaps still pass existing tests;
- split escape handling still holds back partial shortcut sequences just like the existing Alt+Enter handling.

Run: `go test ./cmd/internal/wrapcmd -run 'TestTranslate|TestWorkbenchShortcut' -count=1`
Expected: PASS.

- [ ] **Step 4: Add draft-pane shortcut mappings**

In `nvim/init.lua`, map:
- `Alt+j`: focus agent.
- `Alt+k`: record the draft pane id as last-left, then focus right terminal.
- `Alt+/`: open scrollback viewer.
- `Alt+Shift+C` / `Ctrl+Alt+c`: existing compaction confirm.
- `Alt+t`, `Alt+w`, `Alt+r`: no-op in the draft pane, so right-terminal tab helpers do not leak left. Do not change `nvim/review.lua`'s `Alt+r` reject mapping.

- [ ] **Step 5: Keep right-terminal tab helpers inside `pair term`**

Ensure `Alt+t`, `Alt+w`, and `Alt+r` are only decoded by `pair term`. The review pane must keep its own `Alt+r` reject behavior because zellij no longer owns a global `Alt+r`.

- [ ] **Step 6: Validate config**

Run: `zellij --config-dir zellij setup --check`
Expected: exit 0.

---

## Chunk 4: Tests, Docs, Runtime Bundle

### Task 6: Add integration smoke test for pane gating

**Files:**
- Create: `tests/term-pane-shortcuts-test.sh`
- Modify: `Makefile.local`

- [ ] **Step 1: Write fake-zellij shell test**

Use the real `bin/pair` and fake `zellij` output like `tests/copy-on-select-test.sh`. Assert:
- focused terminal + `pair term --test-shortcut Alt+t` records new-tab;
- focused terminal + `Alt+r` records rename prompt path;
- focused review/floating pane + `Alt+r` records no tab rename;
- focused terminal + `Alt+j` records no movement;
- `Alt+k` returns to the last left pane recorded by an agent/draft-side action and falls back to draft when the sidecar is missing.

If direct keystroke driving is awkward, add a test-only `pair term --test-shortcut <name>` path that exercises the same pure decision + runtime action path without starting a PTY.

- [ ] **Step 2: Add Make target**

Add `test-term-pane-shortcuts` and include it in `test`.

- [ ] **Step 3: Run focused integration test**

Run: `bash tests/term-pane-shortcuts-test.sh`
Expected: PASS.

### Task 7: Update atlas and runtime bundle

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md` if a new section/link is needed
- Generated: `cmd/internal/runtimebundle/assets/runtime/**`

- [ ] **Step 1: Update architecture docs**

Replace the two-pane invariant text with the new three-pane workbench: left Pair stack, right user terminal, pane-local shortcuts, and right-terminal tab helper limits.

- [ ] **Step 2: Regenerate runtime bundle assets**

Run: `make runtimebundle-generate`
Expected: generated KDL/runtime files match source changes.

- [ ] **Step 3: Runtime bundle drift check**

Run: `make runtimebundle-drift-check`
Expected: PASS.

---

## Chunk 5: Final Verification

### Task 8: Verify build and behavior

**Files:**
- Modify issue checklist/log after verification.

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd ./cmd/internal/wrapcmd ./cmd/internal/dispatcher ./cmd/pair-go ./cmd/internal/zellijpane -count=1`
Expected: PASS.

- [ ] **Step 2: Run affected integration tests**

Run: `bash tests/term-pane-shortcuts-test.sh`
Expected: PASS.

Run: `bash tests/copy-on-select-test.sh`
Expected: PASS.

Run: `bash tests/review-toggle-test.sh`
Expected: PASS, including `Alt+r` not globally stolen.

- [ ] **Step 3: Validate zellij assets**

Run: `zellij --config-dir zellij setup --check`
Expected: PASS.

Run: `zellij setup --dump-layout zellij/layouts/main.kdl`
Expected: PASS.

- [ ] **Step 4: Run broader tests**

Run: `go test ./... -count=1`
Expected: PASS.

Run: `make test-lua`
Expected: PASS.

- [ ] **Step 5: Manual smoke**

Start a dev session after `make build`:

```bash
PAIR_DEV=1 bin/pair codex
```

Verify:
- initial layout is left agent/draft stack plus right terminal;
- right terminal opens a shell and can run `nvim`;
- `Alt+t`, `Alt+w`, `Alt+r` work from right terminal only;
- `Alt+j` toggles agent/draft only from left;
- `Alt+k` goes right and returns to the last left pane;
- `Alt+Shift+C` / `Ctrl+Alt+c` and `Alt+/` do nothing from right terminal and work from left panes.

- [ ] **Step 6: Update issue and close**

Check off plan items in `workshop/issues/000116-three-panel-user-terminal.md`, add a dated log entry with verification, then run:

```bash
sdlc close --issue 116 --verified '<tests and smoke evidence>'
```
