# Global Hotkey Draft Routing Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every Zellij-defined global Pair hotkey reach the draft Neovim
pane deterministically from the agent, draft, right terminal, and Pair-owned
Neovim overlays without leaking control or Lua text into applications.

**Architecture:** Extend `workbenchshortcut` as the pure registry for global
chords, actions, and Lua targets. Zellij forwards distinctive sequences only;
the focused Go wrapper consumes them and writes directly to the draft pane id,
while a small shared Lua integration module gives review and
scrollback/change-log Neovim runtimes the same pane-id-addressed route. Draft
Neovim executes its own functions locally.

**Tech Stack:** Go, Neovim Lua, Zellij 0.44.3 KDL/actions, shell integration
tests.

---

## Revisions

### 2026-07-24 — Reconcile first plan review

The plan review found that overlay discovery needs `list-panes --command
--state`, `wrapcmd` did not yet have the injected runtime seam assumed by the
first draft, and two wrapper-local route implementations would duplicate the
load-bearing four-write protocol. Add one shared `draftroute` integration
helper, explicit non-fatal error reporters in both stream shells, and separate
pure Lua parsing/argv tests from process-level Neovim integration tests. Revise
the estimate to map the added integration surface explicitly.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Chord` global variants | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `ShortcutAction` global variants | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `DraftLuaTarget` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ShortcutDecision` | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `OverlayRoutePlan` | `nvim/workbench_route.lua` | new |

- **`Chord` global variants** — the decoded identities for Alt+d, Alt+x,
  Alt+n/Ctrl+Alt+n, Shift+Alt+N, Alt+Up/Down, and Alt+c.
  - **Relationships:** N:1 from terminal encodings to one logical chord; aliases
    such as Alt+n/Ctrl+Alt+n map to the same action.
  - **DRY rationale:** Both Go PTY consumers use the existing decoder rather
    than maintaining wrapper-local byte tables.
  - **Future extensions:** A new KDL-forwarded chord adds one decoded identity
    and one decision row.

- **`ShortcutAction` global variants** — semantic global actions independent of
  the concrete Lua function spelling.
  - **Relationships:** 1:1 with a `DraftLuaTarget` for draft-routed globals.
  - **DRY rationale:** Role policy and Lua dispatch share one semantic action
    registry (`ARCH-DRY`).
  - **Future extensions:** Non-Lua global destinations can remain actions with
    no draft target.

- **`DraftLuaTarget`** — pure action-to-function lookup for
  `PairConfirmDetach`, `PairConfirmQuit`, `PairConfirmRestart`,
  `PairConfirmAgentRestart`, `PairLayoutBigger`, `PairLayoutSmaller`, and
  `PairReviewToggle`.
  - **Relationships:** N:1 chords-to-action for aliases; 1:1
    action-to-function.
  - **DRY rationale:** Replaces hard-coded `"PairConfirmQuit"` dispatch in both
    wrappers with one authoritative mapping.
  - **Future extensions:** Return an explicit absent target for direct or
    pane-local actions.

- **`ShortcutDecision`** — extended pure result carrying the global action and
  draft Lua target for every applicable `PaneRole`.
  - **Relationships:** produced by `Decide`; consumed by the two IO shells.
  - **DRY rationale:** One table defines whether bytes pass, are swallowed, or
    route globally.
  - **Future extensions:** Additional destinations can be represented without
    embedding Zellij calls in the decision core.

- **`OverlayRoutePlan`** — pure Lua functions that select the draft from
  `list-panes --json --command --state` data and build the addressed action
  argv.
  - **Relationships:** N:1 pane records to one draft destination; 1:N from one
    destination to the four write argv vectors.
  - **DRY rationale:** Review and scrollback/change-log runtimes share one
    parser and command constructor.
  - **Future extensions:** Other overlays consume the same plan without
    spawning Zellij in their tests.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `draftroute.Router` | `cmd/internal/draftroute/route.go` | new | Zellij pane listing and pane-id writes |
| Terminal global-route adapter | `cmd/internal/termcmd/run.go` | modified | terminal stream error reporting |
| Agent global-route adapter | `cmd/internal/wrapcmd/wrap.go` | modified | wrapper runtime injection and error reporting |
| `workbench_route` | `nvim/workbench_route.lua` | new | Zellij pane listing and pane-id writes from Neovim overlays |
| Draft global keymaps | `nvim/init.lua` | modified | local Neovim functions |
| Review global keymaps | `nvim/review.lua` | modified | shared overlay router |
| Scrollback global keymaps | `nvim/scrollback.lua` | modified | shared overlay router |
| Changelog global keymaps | `nvim/changelog.lua` | modified | shared overlay router |
| Global KDL bindings | `zellij/config.kdl` | modified | focused-pane key forwarding |
| Workbench shortcut integration suite | `tests/term-pane-shortcuts-test.sh`, `tests/review-toggle-test.sh` | modified | fake Zellij process boundary |

- **`draftroute.Router`** — the single Go IO implementation that requests
  `list-panes --json --command --state`, identifies the draft through
  `zellijpane.Parse` plus `workbenchshortcut.RoleForPane`, and sends
  `<C-\><C-n>:lua Target()<CR>` with `--pane-id` on every write.
  - **Injected into:** terminal and wrapper stream handlers through a small
    `Runtime` interface (`ListPanes` plus `RunZellijAction`).
  - **Future extensions:** Any new Go-owned pane reuses this router rather than
    copying draft discovery or the four-command protocol (`ARCH-DRY`).

- **Terminal global-route adapter** — calls the shared router and reports a
  missing draft/action failure to the `runShell` stderr writer while consuming
  the chord.
  - **Injected into:** `terminalMux`; tests use the existing fake runtime and a
    buffer error sink.
  - **Future extensions:** Structured adaptation events can replace stderr
    without changing routing.

- **Agent global-route adapter** — adds a narrow injectable
  list/action/error-report runtime to `proxy`; production delegates to Zellij
  CLI and `fmt.Fprintf(os.Stderr, ...)`.
  - **Injected into:** `executeWorkbenchDecision`; tests drive the real stream
    decoder with a fake runtime, not the shortcut-handler bypass.
  - **Future extensions:** Other wrapper IO can move behind the same runtime
    incrementally without changing the pure shortcut model.

- **`workbench_route`** — overlay-only Lua router that invokes `list-panes
  --json --command --state`, applies `OverlayRoutePlan`, and performs
  pane-id-addressed writes. It swallows the mapping and uses `vim.notify` for
  missing-draft/action failures.
  - **Injected into:** review, scrollback, and changelog initializers.
  - **Future extensions:** Other Pair-owned Neovim overlays load the same
    module rather than adding no-op globals.

- **Draft global keymaps** — local mappings execute authoritative functions
  without round-tripping through Zellij.
  - **Injected into:** `nvim/init.lua` mapping setup.
  - **Future extensions:** A data-driven local table can add one mapping row per
    global action.

- **Global KDL bindings** — encode only the focused-pane sequence. They contain
  no `MoveFocus`, `WriteChars ":lua ..."`, or dependent multi-action sequence
  (`ARCH-PURPOSE`).
  - **Injected into:** Zellij normal mode.
  - **Future extensions:** The static inventory test forces every new binding
    to be classified.

## Estimate mapping

- Task 1: `smaller-go-module` for the existing pure shortcut registry.
- Task 2: `greenfield-go-module` for shared `draftroute`, plus two
  `smaller-go-module` rows for the terminal and wrapper adapters.
- Task 3: `tui-screen` for the KDL binding surface and its integration
  inventory.
- Task 4: `lua-neovim` plus `api-integration` for pure Lua planning, all eight
  forwarded key encodings (seven actions plus Ctrl+Alt+n), fake Zellij, and the
  three overlay consumers.
- Task 5: `atlas-docs`; `issue-spec` covers the durable design/spec work and
  `milestone-review` represents the single close review boundary.

## Chunk 1: Pure registry and Go production routes

### Task 1: Pin the global chord/action contract

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [ ] **Step 1: Add failing table-driven decoder and decision tests**

Cover the actual forwarded encodings:

```go
{"alt d", "\x1b[100;3u", ChordAltD, ActionConfirmDetach, "PairConfirmDetach"},
{"alt x", "\x1b[120;3u", ChordAltX, ActionConfirmQuit, "PairConfirmQuit"},
{"alt n", "\x1b[110;3u", ChordAltN, ActionRestartPair, "PairConfirmRestart"},
{"ctrl alt n", "\x1b[110;7u", ChordAltN, ActionRestartPair, "PairConfirmRestart"},
{"shift alt n", "\x1b[78;4u", ChordAltShiftN, ActionRestartAgent, "PairConfirmAgentRestart"},
{"alt up", "\x1b[1;3A", ChordAltUp, ActionGrowDraft, "PairLayoutBigger"},
{"alt down", "\x1b[1;3B", ChordAltDown, ActionShrinkDraft, "PairLayoutSmaller"},
{"alt c", "\x1b[99;3u", ChordAltC, ActionToggleReview, "PairReviewToggle"},
```

For each logical action, assert `Decide` returns `DispositionHandle` and the
same Lua target for left-agent, left-draft, and right-terminal roles. Retain
the current pass/swallow behavior for all pane-local chords.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
go test ./cmd/internal/workbenchshortcut -run 'TestDecodeGlobalChord|TestGlobalDecisionMatrix' -count=1
```

Expected: FAIL because the new chord/action identities and Lua target do not
exist.

- [ ] **Step 3: Implement the minimal pure registry**

Add the chord/action constants and sequences, extend `ShortcutDecision` with a
`DraftLuaFunction string`, and make the global branch precede role-local
branches:

```go
if action, fn, ok := globalDraftAction(in.Chord); ok {
    return ShortcutDecision{
        Disposition: DispositionHandle,
        Action: action,
        DraftLuaFunction: fn,
    }
}
```

Keep `globalDraftAction` pure and exhaustive. `ChordFromName`/`ChordName` must
round-trip all new names so wrapper tests can use semantic chord fixtures.

- [ ] **Step 4: Run focused tests and verify GREEN**

Run:

```bash
go test ./cmd/internal/workbenchshortcut -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/workbenchshortcut/shortcut.go cmd/internal/workbenchshortcut/shortcut_test.go
git commit -m "#117: model global workbench hotkeys"
```

### Task 2: Route global chords through both Go PTY shells

**Files:**
- Create: `cmd/internal/draftroute/route.go`
- Create: `cmd/internal/draftroute/route_test.go`
- Modify: `cmd/internal/termcmd/run.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/wrap_test.go`

- [ ] **Step 1: Add failing injected-runtime and stream tests**

First add a fake-runtime test for `draftroute.Router`: pane metadata identifies
draft id 2 and produces:

```text
zellij:
  write --pane-id 2 28
  write --pane-id 2 14
  write-chars --pane-id 2 :lua <Target>()
  write --pane-id 2 13
```

Assert the runtime requested pane metadata equivalent to `list-panes --json
--command --state`.

Then, for every global chord, feed its actual byte sequence through the
terminal mux and wrapper production input path. Use a pane fixture where the
floating terminal is focused and the draft id is 2. Assert child PTY bytes are
empty and both adapters delegate the exact target to the shared router. Add
missing-draft and action-failure cases: the chord remains consumed, no bytes
reach the child, and the terminal stderr or wrapper error-reporter fake records
the failure.

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
go test ./cmd/internal/draftroute ./cmd/internal/termcmd ./cmd/internal/wrapcmd -run 'Global|DraftRoute|NoDraft|RouteFailure' -count=1
```

Expected: FAIL because the shared package/injected wrapper runtime do not exist,
only Alt+x is intercepted, and terminal routing still focuses the draft before
unaddressed writes.

- [ ] **Step 3: Implement pane-id-addressed routing**

Implement `draftroute.Router` once. Have both wrappers consume
`decision.DraftLuaFunction` and delegate to it. Add the narrow production
runtime/error reporter to `wrapcmd` and an stderr sink to the terminal mux.
Every write includes `--pane-id <draft-id>` and no focus action occurs. Missing
draft and action errors are reported non-fatally while the recognized chord
remains handled; they are never converted to `DispositionPass`.

- [ ] **Step 4: Run focused packages and verify GREEN**

Run:

```bash
go test ./cmd/internal/draftroute ./cmd/internal/termcmd ./cmd/internal/wrapcmd -count=1
```

Expected: PASS with no child-byte leakage.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/draftroute cmd/internal/termcmd cmd/internal/wrapcmd
git commit -m "#117: route global chords to draft by pane id"
```

## Chunk 2: Zellij forwarding and Neovim consumers

### Task 3: Replace KDL focus choreography with focused-pane forwarding

**Files:**
- Modify: `zellij/config.kdl`
- Modify: `tests/term-pane-shortcuts-test.sh`
- Modify: `tests/review-toggle-test.sh`

- [ ] **Step 1: Add failing static inventory checks**

Assert each authoritative global binding emits only its expected sequence.
Reject `MoveFocus` and `WriteChars ":lua` within those bindings. Assert Alt+h
and Alt+l remain direct `Run` actions and all focused-process bindings retain
their current sequences.

- [ ] **Step 2: Run shell tests and verify RED**

Run:

```bash
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
```

Expected: FAIL on current Alt+d/n/N/Up/Down/c focus-and-write blocks.

- [ ] **Step 3: Rewrite only the global KDL bindings**

Use one `WriteChars` sequence per binding, including the existing Alt+x KKP
sequence. Preserve direct and pane-local actions exactly.

- [ ] **Step 4: Validate KDL and verify GREEN**

Run:

```bash
zellij --config-dir zellij setup --check
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add zellij/config.kdl tests/term-pane-shortcuts-test.sh tests/review-toggle-test.sh
git commit -m "#117: forward global chords to focused panes"
```

### Task 4: Route draft and overlay Neovim keymaps

**Files:**
- Create: `nvim/workbench_route.lua`
- Create: `nvim/workbench_route_test.lua`
- Create: `tests/workbench-route-nvim-test.sh`
- Modify: `nvim/init.lua`
- Modify: `nvim/review.lua`
- Modify: `nvim/scrollback.lua`
- Modify: `nvim/changelog.lua`
- Modify: `Makefile`

- [ ] **Step 1: Add failing pure/helper and headless integration tests**

Test pane JSON classification and construction of pane-id-addressed action argv
as pure Lua functions in `workbench_route_test.lua`; these tests invoke no
process, Neovim API, or mocks.

Add `tests/workbench-route-nvim-test.sh` with fake `zellij` on PATH. Its headless
Neovim cases invoke all eight actual forwarded encodings—the seven actions plus
the distinct Ctrl+Alt+n alias for Alt+n—in:

- draft init: calls the local `Pair*` function without Zellij routing;
- review init: routes all eight encodings to draft, including Alt+c;
- scrollback init: routes all eight encodings to draft;
- changelog init: independently loads the shared router and routes all eight
  encodings to draft.

Assert the fake receives `list-panes --json --command --state`, every write is
pane-id addressed, and failures notify/log while consuming the key.

- [ ] **Step 2: Run Lua tests and verify RED**

Run:

```bash
nvim -l nvim/workbench_route_test.lua
bash tests/workbench-route-nvim-test.sh
make test-lua
```

Expected: FAIL because the module does not exist and overlays still define
no-op globals.

- [ ] **Step 3: Implement the shared Lua IO module and maps**

Keep parsing/action construction in testable pure functions. The runtime
function calls `zellij action list-panes --json --command --state`, selects the draft by the same
`nvim/init.lua` terminal-command signature as `RoleForPane`, then issues:

```text
zellij action write --pane-id <id> 28
zellij action write --pane-id <id> 14
zellij action write-chars --pane-id <id> :lua <Target>()
zellij action write --pane-id <id> 13
```

Replace overlay no-op globals and review-local Alt+c hide behavior with shared
routing in review, scrollback, and the separately launched `changelog.lua`
initializer. Add explicit local keymaps in draft init for all seven actions.

- [ ] **Step 4: Run Lua and affected shell suites and verify GREEN**

Run:

```bash
nvim -l nvim/workbench_route_test.lua
bash tests/workbench-route-nvim-test.sh
make test-lua
bash tests/review-toggle-test.sh
bash tests/term-pane-shortcuts-test.sh
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add nvim/workbench_route.lua nvim/workbench_route_test.lua nvim/init.lua nvim/review.lua nvim/scrollback.lua nvim/changelog.lua Makefile tests/workbench-route-nvim-test.sh
git commit -m "#117: route overlay globals through draft nvim"
```

## Chunk 3: Documentation and complete verification

### Task 5: Reconcile the public shortcut ownership model

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000117-global-hotkey-routing.md`

- [ ] **Step 1: Update README and atlas**

Document that Zellij forwards global chords to the focused Pair process, which
routes them by current draft pane id; direct global surfaces and pane-local
shortcuts remain distinct. Remove all claims that global actions use relative
focus choreography or overlay no-op fallbacks.

- [ ] **Step 2: Complete issue checkboxes and log evidence**

Record the shortcut inventory, TDD red/green commands, and final verification.
Do not mark the issue done; `sdlc close` owns that transition.

- [ ] **Step 3: Run complete verification**

Run:

```bash
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
make runtimebundle-drift-check
zellij --config-dir zellij setup --check
zellij setup --dump-layout zellij/layouts/main-2.kdl >/dev/null
zellij setup --dump-layout zellij/layouts/main-3.kdl >/dev/null
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 4: Manual smoke**

In a fresh layout-3 `pair-dev` session, focus each of agent, draft, and right
terminal and exercise all seven globals. Confirm Alt+n from the terminal opens
the draft confirmation and the shell receives no visible bytes. Open review,
scrollback, and changelog overlays and repeat representative destructive,
layout, and review actions. Record operator verification before landing.

- [ ] **Step 5: Commit**

```bash
git add README.md atlas/architecture.md workshop/issues/000117-global-hotkey-routing.md
git commit -m "#117: document deterministic global hotkeys"
```
