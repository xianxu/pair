# Native right-pane fullscreen toggle implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Alt+Shift+Return maximize the right terminal from any Pair pane and return to the invoking pane on the next press.

**Architecture:** Zellij owns maximization and layout restoration through `toggle-fullscreen --pane-id`. Pair selects the terminal with its existing picker and remembers the invoking pane for focus restoration. Keyboard routing, generated editor maps and help continue to derive from one binding table.

**Tech Stack:** Go, Lua/Neovim, zellij 0.45.1, existing terminal and layout test seams.

## Evidence and scope

The issue's earlier live test already established full-width expansion, automatic focus on entry, restored dimensions on exit, and the need to restore focus explicitly. The operator also confirmed Ctrl+Space worked during that test. Do not repeat these as prerequisite research.

A disposable four-pane session on 2026-09-20 confirmed that fullscreening a split half changes it from 80×20 to 160×40 and hides its sibling. Explicit focus to another pane exits fullscreen. Hidden panes retain old geometry, and after exiting, more than one pane can report `is_focused=true`. Use the invoking process's `ZELLIJ_PANE_ID` for the return pane. The disposable session was deleted after the probe.

The existing live registry for this thread includes pane 1 / PID 90791, matching the running `pair term`. The issue's earlier dead-registry observation is not current evidence of a blocker; retain the existing registry/classifier.

Two corrections to the issue's assumptions:

- Review runs plain nvim, with `workbench_route.lua` installing global maps. Its local send-menu map shadows the global spelling; there is no wrapper to intercept it. Direct pane actions must be represented in generated Lua routing too.
- Kitty's modifier 10 is Shift+Super, not Shift+Alt. Keep Return's modifier 4 and document the deliberate absence of 10; do not broaden this change into existing arrow aliases. Sources: [Kitty modifiers](https://sw.kovidgoyal.net/kitty/keyboard-protocol/#modifiers), [Ghostty features](https://ghostty.org/docs/features), [WezTerm KKP setting](https://wezterm.org/config/lua/config/enable_kitty_keyboard.html). Ghostty/kitty support negotiated KKP; WezTerm requires its documented setting. These are documented capabilities, not physical-key smoke results on every host.

Add `mouse_scroll_resize false` for standalone Pair. Leave #226 open: it also requires deleting Couch's filter and settling the supported version floor. Keeping that compatibility filter does not obstruct this feature.

## Core concepts

| Name | Lives in | Status |
|---|---|---|
| BindingScope / GlobalBinding | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| Pane | `cmd/internal/zellijpane/zellijpane.go` | modified |
| FullscreenPlan | `cmd/internal/layoutcmd/fullscreen.go` | new |
| terminalToggleBurst / terminalToggleSteps | `cmd/internal/layoutcmd/resizeplan.go` | deleted |

`BindingScope` distinguishes global from draft-only bindings. The existing table remains the source for Go routing, Lua generation and help (ARCH-DRY). Keep its existing exported names to limit churn, with comments explaining that it also holds generated draft bindings.

`Pane` gains observed fullscreen state. `FullscreenPlan` is a small pure decision over observed panes, invoking pane ID, last terminal, live terminal IDs and remembered return ID. It selects expand, collapse or no-op and the target/return IDs. It does not compute geometry or implement a new window manager (ARCH-PURE).

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| FullscreenReturnStore | `cmd/internal/workbenchshortcut/fullscreen_store.go` | new | existing pane-ID sidecar helpers |
| FullscreenRuntime / RunToggleFocused | `cmd/internal/layoutcmd/fullscreen.go`, `layoutcmd.go` | modified | pane observation, store and zellij actions |
| Paths fullscreen members | `cmd/internal/artifactpath/paths.go`, `manifest.go`, `gc.go` | modified | canonical artifact paths, export and collection |
| Pane action dispatch | `cmd/internal/wrapcmd/wrap.go`, `cmd/internal/termcmd/run.go`, `nvim/workbench_route.lua` | modified | chord to existing layout CLI |
| Stateful fullscreen fixture | `cmd/internal/layoutcmd/fullscreen_test.go` | new | same runtime interface as production |

The runtime extension is specific to fullscreen, avoiding new persistence requirements on unrelated pane-focus callers. `termcmd.OSRuntime` delegates storage to the same helper as `layoutcmd.OSRuntime`.

## Ordering and failure rules

The ordinary transition is deliberately short:

```text
tiled + press -> save invoking pane -> toggle-fullscreen --pane-id terminal
fullscreen + press -> toggle-fullscreen --pane-id terminal -> focus return -> clear record
```

- Determine direction from observed `is_fullscreen`, never sidecar presence. Collapse the observed fullscreen right terminal, even if the last-used-half record names its sibling.
- When expanding from a right-terminal half, that invoking half wins. From another pane, reuse `pickRightTerminal` and its recorded-half preference. Do not duplicate classification or change the last-used-half record during the round trip.
- Validate the invoking ID against non-plugin panes. Only use a focus-flag fallback when exactly one eligible pane reports focus. Ambiguous or missing identity refuses expansion before any mutation.
- Validate the recorded return ID against the current pane set. A missing/stale record falls back to the draft. If neither exists, exit fullscreen, remain on the terminal and clear the unusable record.
- Explicitly use `toggle-fullscreen`, preserving zellij's UI bars. Collapse restores existing tiling, including a manually resized split.
- Serialize overlapping toggle invocations within this thread with a nonblocking store lock. A second in-flight press is ignored; the lock is released on return/process death. No queued toggles or background jobs.
- A save failure prevents expansion. An action failure is reported, stops subsequent effects, and retains the return record. Do not blindly retry a toggle, because the command may have taken effect. The next invocation re-observes zellij; tiled state starts a fresh expansion and overwrites a stale record. Interrupted round trips do not promise automatic focus recovery.
- A focus failure after collapse retains the record and reports the error. A later press still derives direction from zellij, rather than treating the record as proof of fullscreen. Clear failures are reported; they cannot cause an inverse toggle.
- Reject failed/malformed pane observations before issuing actions. Unknown direction must not be interpreted as tiled. Test absent/wrong-type fullscreen fields explicitly.

These rules enumerate the relevant interrupted sequences without a retry service (ARCH-ORDER). The lock/store are per validated repo scope and tag; payload IDs are checked against current non-plugin panes (ARCH-SECURE). One bounded record and one stable lock file per thread are collected by the existing artifact GC; clear the record on successful collapse and reset stale state on a new expansion (ARCH-FUNERAL).

Interactive operating envelope: one pane-list read per normal invocation, O(number of panes) pure selection, one zellij toggle on expand and at most one focus call after collapse. No polling, resize bursts or sleeps on the keypress path. Tests assert call bounds; the issue's prior measurements establish the external pane-list cost, not a new latency promise (ARCH-CONSTRAINTS). Fake state carries fullscreen owner, focus, geometry, return record and controllable failures; use the recorded live findings for conformance (ARCH-MOCK).

## Chunk 1: Implement and verify one atomic change

### Task 1: Native toggle and return focus

**Files:** `cmd/internal/zellijpane/zellijpane{,_test}.go`; new `cmd/internal/layoutcmd/fullscreen{,_test}.go`; `cmd/internal/layoutcmd/layoutcmd{,_test}.go`; new `cmd/internal/workbenchshortcut/fullscreen_store{,_test}.go`; `cmd/internal/artifactpath/{paths,manifest,gc}.go` and their tests; `cmd/internal/termcmd/run{,_test}.go`.

- [ ] Write failing tests for the parser and pure expand/collapse selection: draft, agent, same terminal, split half, stale return, duplicate focus flags, invalid observations and absent terminal.
- [ ] Run `go test ./cmd/internal/zellijpane ./cmd/internal/layoutcmd ./cmd/internal/workbenchshortcut` and confirm failures concern the new behavior.
- [ ] Implement observed fullscreen parsing and the pure plan. Extend the runtime seam for current pane identity and return-store operations. Execute the two action sequences above.
- [ ] Implement bounded return storage using the existing atomic pane-ID helper, plus nonblocking mutual exclusion. Add canonical paths, family/consumer declarations, environment export and GC enumeration; do not construct filenames in executors.
- [ ] Replace burst tests with the stateful runtime tests. Assert dimensions/rung/process identities survive a round trip, selected half stays selected, and exact effect ordering. Inject failures at save, toggle, focus and clear, plus concurrent invocation and externally exited fullscreen.
- [ ] Delete `resizeplan.go`, `resizeplan_test.go` and now-unused geometry helpers after checking references. Keep geometry parsing used elsewhere.
- [ ] Run `go test ./cmd/internal/zellijpane ./cmd/internal/layoutcmd ./cmd/internal/workbenchshortcut ./cmd/internal/artifactpath ./cmd/internal/termcmd`. Expect PASS.
- [ ] Commit with issue reference and author trailer.

### Task 2: Global toggle, draft-local rungs and editor retirements

**Files:** `cmd/internal/workbenchshortcut/{shortcut.go,shortcut_test.go,render_lua.go}`; `cmd/internal/wrapcmd/{wrap.go,keymap_registry_test.go,shortcut_passthrough_test.go}`; `cmd/internal/termcmd/{run.go,passthrough_test.go}`; `nvim/{workbench_actions.lua,workbench_route.lua,workbench_route_test.lua,init.lua,review.lua,draft_send.lua,submission.lua}` and submission tests; `tests/{workbench-route-nvim-test.sh,review-window-test.sh,submission-transaction-nvim-test.sh,queue-send-test.sh}`.

- [ ] Add failing routing tests proving the fullscreen action executes from the agent and terminal and never reaches a fullscreen child; draft rungs pass through outside the draft while retaining generated draft maps.
- [ ] Add scope to the existing binding table; global Return has `HandledInPane` and `AgentReserved`. Remove its role-only entry/case. Filter actual global routing by scope.
- [ ] Add the wrapper executor case and reuse the terminal's existing layout action. Expose the native layout CLI as a generated direct Lua action so review/scrollback/changelog execute it from their own pane, preserving invoking identity. Do not route this action into the draft first.
- [ ] Make `workbench_route.lua` install draft-scoped rows only in the draft. Add the draft Lua function for the toggle. Test actual draft and review maps with headless nvim, including the removed local override.
- [ ] Remove the append-without-send map and its now-unreachable `no_submit` parameter/branches through `send_and_clear`, `submit_operator_text`, `submission.lua`, `send_to_agent` and `draft_send.lua`. Preserve normal submission retry/uncertain-write handling and wrapper Return behavior.
- [ ] Remove review's local Shift+Alt+Return menu map; keep its exported menu API. Update tests to assert normal submission and remaining menu behavior, retiring only compose-without-submit cases.
- [ ] Document at the Return encoding why modifier 10 is not added. Add an assertion that Shift+Super+Return is not consumed as Shift+Alt+Return.
- [ ] Run `go run ./cmd/internal/workbenchshortcut/generatecmd --out nvim/workbench_actions.lua`, then `make runtimebundle-generate`.
- [ ] Run `go test ./cmd/internal/workbenchshortcut ./cmd/internal/wrapcmd ./cmd/internal/termcmd` and `make test-lua test-queue test-submission-transaction test-review`. Expect PASS.
- [ ] Commit with issue reference and author trailer.

### Task 3: Help, config and acceptance

**Files:** `cmd/internal/keyhelp/{catalog.go,sections.go}` and tests; `cmd/internal/couchcmd/run.go` and help tests; `zellij/config.kdl`; `README.md`; `CHANGELOG.md`; `atlas/{architecture.md,review-workbench.md,index.md}`; `workshop/targets/review-protocol.md`; issue #297.

- [ ] Derive generated-binding help context from binding scope before existing agent-reservation handling. Replace the obsolete draft/terminal Return rows with one global toggle row. Update the Couch reserved-key heading.
- [ ] Add `mouse_scroll_resize false`; retain Couch's compatibility filter and record #226's remaining requirements.
- [ ] Update README shortcuts and Terminal setup: supported KKP configuration, host mapping caveats, no promise on legacy hosts unable to distinguish Shift+Alt+Return. Mark the changed bindings as breaking in CHANGELOG.
- [ ] Update architecture and review descriptions, removing obsolete menu/width claims. Follow the target datatype/review convention if editing its human-facing prose. Ensure atlas index remains complete.
- [ ] Run `go test ./cmd/internal/keyhelp ./cmd/internal/keyscmd ./cmd/internal/couchcmd`; then `make test` and `git diff --check`. Expect PASS; diagnose any failures before claiming completion.
- [ ] Build in `~/workspace/pair`; run the actual chord through draft, agent and terminal routes in a disposable live session, with a shell and nvim. Verify split-half round trip and strip redraw. The earlier native-command probe does not substitute for checking new keyboard wiring.
- [ ] Operator smoke in the workbench: draft cursor preserved after fullscreen/back; shell, nvim and carbonyl where available; Ctrl+Space still opens Couch. Record observations precisely; do not close that row on automated evidence alone.
- [ ] Update issue evidence, then `sdlc close --issue 297 --verified '<actual commands and observations>'`. The close boundary owns the mandatory fresh-context code review; resolve findings there. Publish through `sdlc pr` / `sdlc merge` after all required evidence is present.

One atomic review boundary; no milestone labels. Estimate follows the full-flow plan-quality gate, not this draft plan.
