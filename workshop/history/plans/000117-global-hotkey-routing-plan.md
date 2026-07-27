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

### 2026-07-24 — Remove synchronous pane discovery from the hot path

Fresh layout-3 smoke found a roughly one-second delay before a right-pane
Alt+n reached draft. Live timing isolates `zellij action list-panes --json
--command --state` at 0.62–0.84 seconds per call. Preserve metadata discovery
as the safe fallback, but have draft Neovim publish its session name, pane id,
and live process id at startup. Both shared routers validate the record against
the caller's Zellij session and process liveness before using it. Add a
test-first fast-path assertion that routing performs no pane-list call for a
valid record, plus stale-session/dead-process fallback coverage. This keeps one
validated locator contract (`ARCH-DRY`), separates validation from IO
(`ARCH-PURE`), and removes the latency from every consumer rather than only the
reported terminal path (`ARCH-PURPOSE`).

### 2026-07-24 — Focus draft before confirmations

The operator confirmed that pane-id delivery alone is insufficient for a modal:
the draft must also become focused so the prompt is visible and answerable.
Extend the shared decision with a focus policy and make focus success an atomic
precondition for confirmation writes. Grow/shrink/review remain
focus-preserving. This applies across both Go PTY consumers and every Neovim
overlay (`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-07-24 — Bypass pane classification for terminal globals

The cached-locator fast path was still preceded by `pair term`'s generic
focused-pane inventory. Resolve a global chord directly from the authoritative
registry before entering the pane-relative decision path. Add a production
stream regression whose pane inventory fails while cached Alt+n routing must
still succeed, proving the hot path performs no inventory IO (`ARCH-PURE`,
`ARCH-PURPOSE`).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Chord` global variants | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `ShortcutAction` global variants | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `ShortcutDecision` | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `GlobalBinding` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `find_draft_pane` / `draft_commands` | `nvim/workbench_route.lua` | new |

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
  - **Relationships:** 1:1 with a `GlobalBinding` for draft-routed globals.
  - **DRY rationale:** Role policy and Lua dispatch share one semantic action
    registry (`ARCH-DRY`).
  - **Future extensions:** Non-Lua global destinations can remain actions with
    no draft target.

- **`ShortcutDecision`** — extended pure result carrying the global action and
  draft Lua target for every applicable `PaneRole`.
  - **Relationships:** produced by `Decide`; consumed by the two IO shells.
  - **DRY rationale:** One table defines whether bytes pass, are swallowed, or
    route globally.
  - **Future extensions:** Additional destinations can be represented without
    embedding Zellij calls in the decision core.

- **`GlobalBinding`** — the authoritative Go record for chord, action, Lua
  target, and confirmation-focus policy; it renders the committed Lua consumer
  table.
  - **Relationships:** 1:N registry to generated Neovim mappings.
  - **DRY rationale:** Go and Lua cannot hand-maintain independent focus policy;
    a structural generation/parity test makes drift fail CI (`ARCH-DRY`).
  - **Future extensions:** Additional cross-language routing fields widen this
    record and its renderer.

- **`find_draft_pane` / `draft_commands`** — pure Lua functions that select the draft from
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
| `RouteLua` / `Runtime` | `cmd/internal/draftroute/route.go` | new | cached draft lookup, fallback pane listing, and pane-id writes |
| Terminal global-route adapter | `cmd/internal/termcmd/run.go` | modified | terminal stream error reporting |
| Agent global-route adapter | `cmd/internal/wrapcmd/wrap.go` | modified | wrapper runtime injection and error reporting |
| `workbench_route` | `nvim/workbench_route.lua` | new | Zellij pane listing and pane-id writes from Neovim overlays |
| Draft global keymaps | `nvim/init.lua` | modified | local Neovim functions |
| Review global keymaps | `nvim/review.lua` | modified | shared overlay router |
| Scrollback global keymaps | `nvim/scrollback.lua` | modified | shared overlay router |
| Changelog global keymaps | `nvim/changelog.lua` | modified | shared overlay router |
| Global KDL bindings | `zellij/config.kdl` | modified | focused-pane key forwarding |
| Workbench shortcut integration suite | `tests/term-pane-shortcuts-test.sh`, `tests/review-toggle-test.sh` | modified | fake Zellij process boundary |

- **`RouteLua` / `Runtime`** — the single Go IO implementation that first uses
  the validated cached draft locator, falls back to
  `list-panes --json --command --state` plus `zellijpane.Parse` /
  `workbenchshortcut.RoleForPane`, and sends
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
  --json --command --state`, applies `find_draft_pane` / `draft_commands`, and performs
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

### 2026-07-24 — Reconcile close review

The implemented pure model uses `GlobalBinding` directly for action-to-Lua
and focus policy; there is no separate `DraftLuaTarget` entity. The overlay
pure core consists of the named `find_draft_pane` and `draft_commands`
functions rather than an `OverlayRoutePlan` type. Correct the Core concepts
table and narrative accordingly. Also preserve the shared overlay mappings
after scrollback buffer setup; remove its stale Alt+x override and keep
Alt+Up/Down no-ops visual-mode-only.

The next review found one more stale planned name: the Go integration is the
`RouteLua` function over an injected `Runtime`, not a `draftroute.Router` type.
Correct that table/narrative and exercise the effective keymaps from each
review, scrollback, and change-log configuration at the fake-Zellij process
boundary, covering focus-first confirmation, atomic focus failure, and
focus-preserving layout routing (`ARCH-PURPOSE`).

The operator’s final right-terminal Alt+n smoke confirmed prompt latency and
focus behavior were good enough to ship. Mark the manual-smoke row complete and
update Task 2’s older pre-focus prose to the later approved focus policy.
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
- Task 6 revision: one added `smaller-go-module` row covers `FocusDraft` plus
  generated Lua parity; one `lua-neovim` row covers overlay consumption; one
  `api-integration` row covers atomic focus/write failure handling in both
  injected IO shells. These rows were added to the issue estimate when the
  post-smoke scope was approved.

## Chunk 1: Pure registry and Go production routes

### Task 1: Pin the global chord/action contract

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [x] **Step 1: Add failing table-driven decoder and decision tests**

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

- [x] **Step 2: Run the focused test and verify RED**

Run:

```bash
go test ./cmd/internal/workbenchshortcut -run 'TestDecodeGlobalChord|TestGlobalDecisionMatrix' -count=1
```

Expected: FAIL because the new chord/action identities and Lua target do not
exist.

- [x] **Step 3: Implement the minimal pure registry**

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

- [x] **Step 4: Run focused tests and verify GREEN**

Run:

```bash
go test ./cmd/internal/workbenchshortcut -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

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

- [x] **Step 1: Add failing injected-runtime and stream tests**

First add a fake-runtime test for `RouteLua` over the injected `Runtime`: pane metadata identifies
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

- [x] **Step 2: Run focused tests and verify RED**

Run:

```bash
go test ./cmd/internal/draftroute ./cmd/internal/termcmd ./cmd/internal/wrapcmd -run 'Global|DraftRoute|NoDraft|RouteFailure' -count=1
```

Expected: FAIL because the shared package/injected wrapper runtime do not exist,
only Alt+x is intercepted, and terminal routing still focuses the draft before
unaddressed writes.

- [x] **Step 3: Implement pane-id-addressed routing**

Implement `RouteLua` once over the shared `Runtime`. Have both wrappers consume
`decision.DraftLuaFunction` and delegate to it. Add the narrow production
runtime/error reporter to `wrapcmd` and an stderr sink to the terminal mux.
Every write includes `--pane-id <draft-id>`; confirmation globals first focus
the draft atomically, while layout/review globals preserve focus. Missing draft,
focus, and action errors are reported non-fatally while the recognized chord
remains handled; they are never converted to `DispositionPass`.

- [x] **Step 4: Run focused packages and verify GREEN**

Run:

```bash
go test ./cmd/internal/draftroute ./cmd/internal/termcmd ./cmd/internal/wrapcmd -count=1
```

Expected: PASS with no child-byte leakage.

- [x] **Step 5: Commit**

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

- [x] **Step 1: Add failing static inventory checks**

Assert each authoritative global binding emits only its expected sequence.
Reject `MoveFocus` and `WriteChars ":lua` within those bindings. Assert Alt+h
and Alt+l remain direct `Run` actions and all focused-process bindings retain
their current sequences.

- [x] **Step 2: Run shell tests and verify RED**

Run:

```bash
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
```

Expected: FAIL on current Alt+d/n/N/Up/Down/c focus-and-write blocks.

- [x] **Step 3: Rewrite only the global KDL bindings**

Use one `WriteChars` sequence per binding, including the existing Alt+x KKP
sequence. Preserve direct and pane-local actions exactly.

- [x] **Step 4: Validate KDL and verify GREEN**

Run:

```bash
zellij --config-dir zellij setup --check
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
```

Expected: PASS.

- [x] **Step 5: Commit**

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

- [x] **Step 1: Add failing pure/helper and headless integration tests**

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

- [x] **Step 2: Run Lua tests and verify RED**

Run:

```bash
nvim -l nvim/workbench_route_test.lua
bash tests/workbench-route-nvim-test.sh
make test-lua
```

Expected: FAIL because the module does not exist and overlays still define
no-op globals.

- [x] **Step 3: Implement the shared Lua IO module and maps**

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

- [x] **Step 4: Run Lua and affected shell suites and verify GREEN**

Run:

```bash
nvim -l nvim/workbench_route_test.lua
bash tests/workbench-route-nvim-test.sh
make test-lua
bash tests/review-toggle-test.sh
bash tests/term-pane-shortcuts-test.sh
```

Expected: PASS.

- [x] **Step 5: Commit**

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

- [x] **Step 1: Update README and atlas**

Document that Zellij forwards global chords to the focused Pair process, which
routes them by current draft pane id; direct global surfaces and pane-local
shortcuts remain distinct. Remove all claims that global actions use relative
focus choreography or overlay no-op fallbacks.

- [x] **Step 2: Complete issue checkboxes and log evidence**

Record the shortcut inventory, TDD red/green commands, and final verification.
Do not mark the issue done; `sdlc close` owns that transition.

- [x] **Step 3: Run complete verification**

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

- [x] **Step 4: Manual smoke**

In a fresh layout-3 `pair-dev` session, focus each of agent, draft, and right
terminal and exercise all seven globals. Confirm Alt+n from the terminal opens
the draft confirmation and the shell receives no visible bytes. Open review,
scrollback, and changelog overlays and repeat representative destructive,
layout, and review actions. Record operator verification before landing.

- [x] **Step 5: Commit**

```bash
git add README.md atlas/architecture.md workshop/issues/000117-global-hotkey-routing.md
git commit -m "#117: document deterministic global hotkeys"
```

## Chunk 3: Confirmation visibility revision

### Task 6: Focus draft atomically before confirmation delivery

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Modify: `cmd/internal/workbenchshortcut/shortcut_test.go`
- Create: `nvim/workbench_actions.lua` (generated)
- Modify: `cmd/internal/draftroute/route.go`
- Modify: `cmd/internal/draftroute/route_test.go`
- Modify: `cmd/internal/termcmd/run.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/translate_test.go`
- Modify: `nvim/workbench_route.lua`
- Modify: `nvim/workbench_route_test.lua`
- Modify: `tests/workbench-route-nvim-test.sh`

- [x] **Step 1: Write failing pure-policy tests**

Assert `FocusDraft=true` for detach, quit, pair restart aliases, and agent
restart; assert false for grow, shrink, and review. Keep the function target
mapping unchanged. Add a renderer/parity assertion: the committed
`nvim/workbench_actions.lua` must exactly equal the Lua serialization of the
authoritative Go `GlobalBinding` registry.

- [x] **Step 2: Verify RED**

```bash
go test ./cmd/internal/workbenchshortcut -run TestGlobalDecisionMatrix -count=1
```

Expected: failure because `ShortcutDecision` has no focus policy.

- [x] **Step 3: Add the minimal shared policy**

Add `FocusDraft bool` to the decision and derive it in the single
`GlobalBinding` registry. Add a deterministic Lua renderer and generate
`nvim/workbench_actions.lua`; `workbench_route.lua` loads that generated table.
Do not infer confirmation behavior from Lua function-name prefixes or maintain
a second Lua policy table.

- [x] **Step 4: Write failing IO-order and failure tests**

For `draftroute`, both production PTY streams, and the Neovim process fake,
assert:

```text
focus-pane-id <draft>
write --pane-id <draft> 28
write --pane-id <draft> 14
write-chars --pane-id <draft> :lua PairConfirm…()
write --pane-id <draft> 13
```

Inject focus failure and assert zero write actions plus reported error. Assert
grow/shrink/review produce the existing four writes and no focus action.

- [x] **Step 5: Verify RED**

```bash
go test ./cmd/internal/draftroute ./cmd/internal/termcmd ./cmd/internal/wrapcmd -run 'Focus|Global' -count=1
nvim -l nvim/workbench_route_test.lua
bash tests/workbench-route-nvim-test.sh
```

Expected: ordering/failure assertions fail.

- [x] **Step 6: Implement focus-before-write in thin IO shells**

Pass the explicit policy into `draftroute.RouteLua`. In Lua, consume generated
records `{ fn, focus }`; draft-local execution ignores the routing flag, while
overlays focus the discovered pane before `send_to_draft`. Stop and report on
focus failure.

- [x] **Step 7: Verify GREEN**

Run the commands from Step 5 plus:

```bash
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
git diff --check
```

Expected: PASS.

- [x] **Step 8: Commit**

```bash
git add cmd/internal/workbenchshortcut cmd/internal/draftroute cmd/internal/termcmd cmd/internal/wrapcmd nvim tests workshop/issues/000117-global-hotkey-routing.md workshop/plans/000117-global-hotkey-routing-plan.md
git commit -m "#117: focus draft before global confirmations"
```
