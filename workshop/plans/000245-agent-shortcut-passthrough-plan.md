# Agent shortcut passthrough Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every focused agent all unreserved shortcuts while retaining six explicit navigation/tab chords and existing role-local behavior in other Pair panes.

**Architecture:** Couch intercepts only its three navigation chords while an actor is active; Zellij delivers other keys to the receiving pane. The shared shortcut model declares the reserved set and makes the agent role passthrough by default. Pane-local consumers retain their actions, including help/changelog previously executed directly by Zellij. Paste framing precedes shortcut dispatch, so pasted bytes never become actions.

**Tech Stack:** Go, Couch terminal interceptor, Pair wrapper, generated Lua routing, Neovim, Zellij KDL, existing stateful terminal/process fixtures.

---

## Scope and accepted behavior

The operator continued after the choice between simplifying outer interception and adding an owner command bridge. This plan takes the recommended simplification. Root owns the issue revision, approval and SDLC gates. One atomic issue-close review boundary; task numbers are implementation units, not milestones.

The six reserved chords are Ctrl+Space (Couch switcher), Ctrl+Backspace (previous Couch thread; the operator's Apple Delete key), Ctrl+Return (newest notification thread), Shift+Alt+T (new right-terminal tab), and Shift+Alt+Left/Right (previous/next right-terminal tab). Forward Delete remains distinct. The first three belong to Couch; the last three execute in Pair pane consumers. Existing supported encodings remain accepted; unsupported terminal encodings do not acquire invented aliases.

When Couch's own panel is focused, its existing detach/park/relaunch shortcuts still open the existing confirmations. When an actor is focused, Alt+d, Alt+x, Alt+n and Ctrl+Alt+n pass to Zellij along with every other unreserved chord. An agent receives them as input. Draft/right-terminal consumers retain existing Pair actions. That is an accepted change from Couch's previous actor-wide lifecycle interception: Pair detach/quit/restart are not Couch's durable detach/park/relaunch. Couch lifecycle operations remain available through the switcher. Document that distinction without claiming the pane actions preserve Couch's old transaction/binary-reload semantics.

There is no inner-focus cache, session polling in the input path, generic command bridge or new supervisor. Couch's `FocusActor` means a whole hosted session and cannot prove which inner pane receives input. Role authority comes from the pane process receiving Zellij input. Existing Return/composer adaptations, mouse behavior and parked/resume semantics are outside this shortcut policy change.

## Core concepts

### Pure entities

| Name | Kind | Lives in | Status |
|---|---|---|---|
| Reserved shortcut policy and role decisions | PURE | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| Paste-aware shortcut framing | PURE | new `cmd/internal/workbenchshortcut/framing.go` | new |
| Focus-scoped Couch interception | PURE | `cmd/internal/couchtty/keys.go` | modified |
| Generated pane bindings | PURE | `cmd/internal/workbenchshortcut/render_lua.go`, `nvim/workbench_actions.lua` | modified |
| Shortcut help source mapping | PURE | `cmd/internal/keyhelp/catalog.go`, `sections.go` | modified |

**Reserved shortcut policy:** One declared set associates each reserved chord with its owner (Couch navigation or Pair pane action) and help. Existing `GlobalBindings`, `ChordEncodings`, `Decide` and help consumers derive behavior from it; do not add independent six-key lists to wrapper, Console and documentation generator. Extend the shared chord model for Couch navigation if necessary; preserve the existing legacy NUL/Ctrl+Backspace and Kitty press/repeat encodings. The model has many chords to one role decision and one reservation owner per reserved chord. Future exceptions are table entries, not agent-name conditionals.

The decision order is explicit: for `PaneRoleLeftAgent`, return passthrough unless the chord is reserved for a pane action; then use existing global/role decisions for other roles. Couch-owned navigation is consumed by Couch when hosted, not reimplemented as a Pair lifecycle action. Standalone Pair has no Couch navigation owner.

**Paste-aware shortcut framing:** One pure incremental scanner recognizes ordinary bytes, complete chord encodings and bracketed-paste boundaries. It retains bounded partial escape/marker state between reads and emits original bytes for passthrough; it performs no IO. Its state owns whether shortcut recognition is permitted. Reuse the existing chord encoding table and caller escape timeout. A shortcut finder that merely checks `inPaste` at the beginning of a chunk is insufficient: a single chunk may contain paste-start, a reserved chord, paste-end and a real chord. No duplicated competing paste state machines may authorize shortcuts. Existing Return adaptation consumes the same framed bytes and retains its established semantics.

**Focus-scoped interception:** `Interceptor.FeedHit` receives an explicit pure interception scope (panel or actor). It still recognizes paste and mouse framing in both scopes. Navigation stays intercepted in either scope; Pair lifecycle chords are intercepted only for the panel. Unclaimed recognized encodings are emitted byte-for-byte rather than reconstructed from a canonical spelling. `AllInterceptorHits` remains exhaustive for panel handlers.

### Integration points

| Name | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| Wrapper stdin routing | INTEGRATION | `cmd/internal/wrapcmd/wrap.go` | modified | incremental input, Return adapter, agent PTY, workbench action runtime |
| Couch input dispatch | INTEGRATION | `cmd/internal/couchtty/console.go` | modified | Interceptor, host input, child terminal, existing menu confirmations |
| Role-local help/changelog execution | INTEGRATION | `nvim/init.lua`, `nvim/workbench_route.lua`, `cmd/internal/termcmd/run.go`, `zellij/config.kdl` | modified | existing floating help/changelog processes and focused-pane routing |
| Deterministic input environment | INTEGRATION | existing `couchtty/console_test.go`, `wrapcmd/translate_test.go`, `termcmd/run_test.go`, `nvim/workbench_route_test.lua`; new focused test files as listed below | modified | stateful fake terminals, panes/actions, input streams and output capture |
| Isolated real input conformance | INTEGRATION | new `cmd/internal/couchcmd/shortcut_conformance_live_test.go`; `Makefile.local`, `.github/workflows/couch-zellij-conformance.yml` | new/modified | real Zellij, production input routing, fixture agent byte recorder |

The help/changelog commands are existing integrations, not new products. Declare their chords/actions in the shared model, generate the nvim bindings, and have non-agent consumers reuse the existing draft-routing/execution seam. Preserve floating geometry, command arguments and close-on-exit behavior. Any newly extracted executor belongs in the existing owning package and must be recorded in a plan revision before the final concept/diff audit.

The deterministic environment models terminal bytes delivered, focused role, opened panes/actions and lifecycle requests across multiple calls. Reuse existing fakes and process helpers; add state only for newly observable help/changelog actions. Pure tests do not need process mocks. Live conformance uses stand-in byte-recording agent processes and owned disposable sessions, never paid coding-agent sessions or operator data.

## Design constraints

- **ARCH-DRY:** One reserved set, one chord-encoding table, existing generated Lua/help pipelines. Sweep wrapper `Decide`, term `DecideGlobal` shortcuts, nvim maps, Couch matching and KDL direct actions together.
- **ARCH-PURE:** Role/focus/paste decisions take values. No Zellij command, filesystem focus read or timer creation inside them.
- **ARCH-PURPOSE:** Verify the full outer-Couch → Zellij → wrapper path. Fix the class of unreserved shortcuts, including help/changelog, not only Alt+Up.
- **ARCH-MOCK:** Deterministic byte/effect tests plus real isolated Zellij conformance. A table of action labels alone cannot prove input delivery.
- **ARCH-CONSTRAINTS:** Preserve bounded escape buffering, escape-timeout flush, paste boundaries and unsupported-byte forwarding. A partial sequence cannot retain arbitrary subsequent input indefinitely.
- **ARCH-ORDER:** Route bytes before a hit to the current surface, handle the hit, then reread Console focus before processing its suffix. A Ctrl+Space followed by a panel lifecycle chord in one host read must use the new panel scope. No stale per-read focus snapshot.
- **ARCH-SECURE:** Literal input stays literal, especially within paste. Structured argv only for help/changelog execution; no interpolation of pasted or agent bytes into commands.
- **ARCH-FUNERAL:** Remove obsolete actor-interception tests/help claims and direct KDL Run bindings when replacing them. New live fixtures register cleanup immediately and delete only recorded owned session/process identities.

## Chunk 1: Implement and prove pane ownership

### Task 1: Declare the shared reservation policy

**Files:** Modify `cmd/internal/workbenchshortcut/shortcut.go`, `shortcut_test.go`, `render_lua.go`; regenerate `nvim/workbench_actions.lua`. Modify `cmd/internal/keyhelp/catalog.go`, `sections.go` and colocated tests for reservation-derived help.

- [ ] Write a complete role × chord decision matrix: every known unreserved chord is passthrough for every agent role instance; exactly the three tab chords retain pane actions. Include new help/changelog chords, Alt+Up/Down, ordinary Alt+Left/Right, lifecycle chords and unknown bytes. Draft/terminal decisions remain unchanged apart from adding previously Zellij-owned actions.
- [ ] Add pure reservation tests proving exactly three Couch-owned and three pane-owned entries, no duplicate/ambiguous encoding ownership, and no Forward Delete alias for Ctrl+Backspace. Help coverage must derive from the declared set.
- [ ] Run `go test ./cmd/internal/workbenchshortcut ./cmd/internal/keyhelp -count=1`; record the expected policy/help regression failures before implementation.
- [ ] Add the minimal shared policy and move the agent-role gate before unconditional globals in `Decide`. Add help/changelog declarations and generated nvim mappings. Keep non-agent global routing intact; do not use `DecideGlobal` as an agent-policy bypass.
- [ ] Run `go run ./cmd/internal/workbenchshortcut/generatecmd --out nvim/workbench_actions.lua`, then rerun the two packages. Generated-map drift and complete decision tests must pass.

### Task 2: Make wrapper shortcut recognition respect paste framing

**Files:** Create `cmd/internal/workbenchshortcut/framing.go`, `framing_test.go`; modify `cmd/internal/wrapcmd/wrap.go`, `translate_test.go` and existing relevant Return/paste tests.

- [ ] Add red boundary tests through `translateStdin`: literal unreserved chords reach the agent unchanged and perform no action; reserved tab chords execute once outside paste and never inside paste. Run with Return adaptation enabled and disabled, including an agent without a harness profile.
- [ ] Enumerate every split point of each accepted chord and paste-start/end marker. Test ordinary prefix + partial marker, start/chord/end in one chunk, already-in-paste next chunk, end marker followed immediately by a real reserved chord, repeated adjacent chords, unknown escape sequences and final timeout/EOF flush. Verify byte concatenation, exact action count, unchanged pasted CRs and no false lifecycle submission observation.
- [ ] Run `go test ./cmd/internal/workbenchshortcut ./cmd/internal/wrapcmd -run 'Fram|Translate|Paste|Return|Submitting' -count=1`; capture the existing `FindChord`-inside-paste failure and remap-disabled split-marker failure where reproduced.
- [ ] Replace unconditional `FindChord(data)` dispatch with incremental paste-aware framing. Keep original raw chord bytes when `handleWorkbenchChord` declines them. Resolve partial marker/chord suffixes before the next chunk; reuse the existing escape timer, with a bounded buffer derived from marker/chord lengths.
- [ ] Ensure Return adaptation and shortcut dispatch agree on framing; do not independently scan a raw chunk for shortcuts before interpreting its paste boundaries. Preserve orientation/turn observation behavior outside the shortcut change.
- [ ] Rerun focused tests and `go test ./cmd/internal/wrapcmd ./cmd/internal/workbenchshortcut -count=1`; require all supported agent configurations to pass the same passthrough policy.

### Task 3: Scope Couch interception to its actual focus authority

**Files:** Modify `cmd/internal/couchtty/keys.go`, `console.go`, `keys_test.go`, `console_test.go`, `console_relaunch_chord_test.go`; add `cmd/internal/couchtty/console_shortcut_passthrough_test.go` for focused integration cases.

- [ ] Add red tests for actor input: every encoding of Alt+d/Alt+x/Alt+n/Ctrl+Alt+n reaches the child unchanged, no confirmation opens, no lifecycle operation runs. The three Couch navigation chords still work; all existing mouse/paste cases remain valid.
- [ ] Add panel counterparts retaining detach/park/relaunch confirmation and operation payload behavior. Update previous actor-relaunch expectations to assert the accepted new behavior instead of deleting useful confirmation coverage.
- [ ] Add same-read transition tests: ordinary bytes + Ctrl+Space + lifecycle chord routes suffix to panel; panel selection returning to an actor + unreserved suffix routes it to the child. Cover a chord split across reads, mouse focus changes and pasted navigation/lifecycle encodings.
- [ ] Run `go test ./cmd/internal/couchtty -run 'Interceptor|Hotkey|Relaunch|Shortcut|Paste' -count=1`; record expected actor-interception failures.
- [ ] Add the pure panel/actor scope to interception and read it per `FeedHit` iteration in `Console.processInput`. Preserve `route(before) → handle → process(rest)` ordering, handler exhaustiveness and mouse payload lifetime. Avoid cached inner-pane role or special-case agent names.
- [ ] Rerun focused tests and `go test -race ./cmd/internal/couchtty -count=1`; confirm no menu/confirmation/notification regressions.

### Task 4: Remove earlier Zellij consumption and preserve non-agent actions

**Files:** Modify `zellij/config.kdl`, `nvim/init.lua`, `nvim/workbench_route.lua`, `nvim/workbench_route_test.lua`, `cmd/internal/termcmd/run.go`, `run_test.go`, `tests/term-pane-shortcuts-test.sh`, `tests/workbench-route-nvim-test.sh`. Reuse generated `nvim/workbench_actions.lua` from Task 1.

- [ ] Add failing config/action-contract tests for Alt+h and Alt+l: KDL delivers bytes to the focused pane; agent receives them; draft/terminal opens the same help/changelog UI with the same command and geometry. Verify action failure reports do not silently swallow a requested non-agent action.
- [ ] Sweep all active KDL bind blocks and inherited defaults for unreserved keys consumed before a pane. List each direct Run/action binding encountered in the implementation log; migrate any workbench sibling of help/changelog to the same role-local policy. Do not reintroduce Zellij defaults when removing a binding: explicitly unbind inherited consuming defaults where necessary.
- [ ] Replace the help/changelog direct Run bindings with canonical chord forwarding. Add non-agent action handlers through the existing generated draft-routing model; terminal consumers continue their existing role-specific tab/focus behavior. Preserve floating help/changelog dimensions and close-on-exit, and do not focus the draft unnecessarily.
- [ ] Run `go test ./cmd/internal/termcmd ./cmd/internal/keyhelp ./cmd/internal/keyscmd -count=1`, `nvim -l nvim/workbench_route_test.lua`, `bash tests/workbench-route-nvim-test.sh`, and `bash tests/term-pane-shortcuts-test.sh`. Run `zellij --config-dir zellij setup --check`; verify successful config parsing and no remaining direct unreserved workbench consumption.

### Task 5: Prove real input delivery through the composed stack

**Files:** Create `cmd/internal/couchcmd/shortcut_conformance_live_test.go`; modify `Makefile.local` and `.github/workflows/couch-zellij-conformance.yml` to include the test and its source paths. Reuse existing Couch command/process fixtures and controlled Zellij helpers.

- [ ] Add a portable composed Console/child/role-consumer fixture first. Its oracle records the exact agent byte stream, role-local actions and Couch menu transitions, including paste and same-read focus changes. Actual process/stream state must support the assertions; do not merely assert which callback was invoked.
- [ ] Add gated `TestAgentShortcutInputConformanceLive`: temporary HOME/data namespace/repo, random owned Zellij session, real input connection through Couch Console, and a fixture byte-recording agent behind production `pair wrap`. Keep real Zellij input routing and production wrapper active. Any stand-in at the agent application or helper seam must be described in the test and evidence.
- [ ] Drive unreserved lifecycle/arrows/help/changelog chords through outer host input and assert agent receipt with zero workbench effects. Use terminal-equivalent canonical encodings where Zellij legitimately rewrites a chord; separately prove wrapper passthrough preserves received bytes. Test reserved tabs outside paste, reserved chords inside paste, and the three Couch navigation actions without leaking them to the agent.
- [ ] Change to draft/right-terminal using the real fixture's focus actions, then verify representative retained actions including help/changelog; return to the agent by mouse/focus and repeat delivery. Prove no polling-based role authority is needed. Record session/pane/process identities before actions and clean up only those resources.
- [ ] Add the gated command `PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcmd -run '^TestAgentShortcutInputConformanceLive$' -count=1 -v` to the existing live conformance target. Extend PR/push filters for changed wrapper, shortcut, Couch input, KDL/nvim and fixture files so recurring coverage runs for the actual sources, not just the new test.
- [ ] Run the live case directly and record byte/action evidence. If the environment cannot exercise a layer, leave that obligation unchecked and report the exact limitation; a fake-only result does not satisfy the composed live row.

### Task 6: Verify, document and close

**Files:** Modify `README.md`, `atlas/couch.md`, relevant existing shortcut atlas page discovered through `atlas/index.md`, and `atlas/index.md` only if adding a page. Root updates `workshop/issues/000245-agent-shortcut-passthrough.md`.

- [ ] Render `pair keys` and Couch help, verify both derive the six exceptions and pane scope from the declared policy. Remove actor-wide lifecycle claims. Explain the switcher as the Couch lifecycle entrypoint and the accepted Pair-local behavior in draft/terminal panes.
- [ ] Run `go test ./cmd/internal/workbenchshortcut ./cmd/internal/wrapcmd ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/termcmd ./cmd/internal/keyhelp ./cmd/internal/keyscmd -count=1`, then relevant race packages, `make test`, and `git diff --check`. Build current binaries with the existing Make target before operator smoke.
- [ ] Compare every concept-table row with the final diff, including any extracted framing/executor file and generated artifact. Append a timestamped revision for actual deviations; keep original design history. Record exact portable/live test results and limitations in the issue.
- [ ] Commit verified implementation with an issue reference and author trailer. Root runs `sdlc close --issue 245 --verified '<actual behavior and verification evidence>'`; fix Critical/Important findings before crossing the single issue-close boundary.
- [ ] Pause for operator smoke on current binaries: focused agent receives Alt+Up/Down, ordinary Alt+Left/Right and former lifecycle/help chords; six exceptions perform the agreed actions; mouse focus leaves the agent; draft/terminal retain their actions. Do not claim this operator acceptance from automated stand-in fixtures.

## Revisions

### 2026-09-14 — Minimal reservation ownership and complete upstream input coverage

Reason: root's final source/config audit identified inherited Zellij actions and
confirmed the existing live fixture can expose its real client PTY. These
refinements supersede broader implementation choices in the initial draft.

**Reservation representation:** Prefer one `AgentReserved` field on the existing
`GlobalBinding` rows for the three tab chords, plus the existing typed Couch
navigation table for its three owner-specific chords. These are separate owners,
not competing policy copies. Do not expand the shared chord enum merely to move
Couch-only navigation encodings. The role gate and generated help enumerate
`AgentReserved`; Couch scope derives its navigation matching/help from its
existing declarations. Assert exactly three entries in each owner table and
render their combined six-entry contract without another hand-maintained list.
This replaces Task 1's optional expansion into a unified reservation/chord model
and keeps the initial concept row's shared role-policy contract.

**Zellij inherited defaults:** The installed default config still consumes
Alt+f, Alt+=/+/-, Alt+[/], Alt+p and Alt+Shift+p for multiplexer operations. Set
`keybinds clear-defaults=true` and retain only the explicit reviewed Pair
forwarding bindings needed by the pane consumers. Inspect the complete effective
config, including modes, before concluding that the agent owns unreserved keys.
Undocumented inherited Zellij control shortcuts are retired throughout Pair;
existing Pair-owned non-agent actions remain. This is an explicit scope delta,
not a promise to reproduce hidden default multiplexer behavior in every pane.
Task 4's negative config checks must reject direct consuming actions and implicit
default inheritance; the live matrix must include Alt+f, Alt+[ and Alt++ with
unchanged pane/tab/layout state and captured agent input.

**Live input seam:** Extend existing
`cmd/internal/pairlifecycletest/live_zellij.go` (INTEGRATION, modified) with
optional config/layout parameters and a `WriteInput` operation on its actual
client PTY. Preserve defaults for existing fixture consumers. This is an
extension of `ControlledZellij`, not a second Zellij launcher. Use a raw byte
capture pane/stand-in behind the real wrapper and record pane/tab state before
and after keys. Sending `zellij action write` to a pane is not the live input
oracle: it bypasses client keybindings. Task 5 must drive client-PTY input, plus
Couch host input for the composed outer-interception case, using the reviewed
config. If the portable Console layer and real client layer are tested
separately, label that layered evidence and retain the composed-path obligation
until it is exercised.

Plan-quality review belongs to root's `sdlc change-code` gate. No separate
standalone plan review is requested here.

### 2026-09-14 — Bound Task 5 to complementary input conformance layers

Reason: a new complete production Couch/helper/Zellij/wrapper live application
fixture exceeds this change's bounded verification need. Root selects the
existing portable and real-client seams, with operator smoke as final acceptance.
This supersedes Task 5's full live composed-path requirement and the preceding
revision's sentence retaining that obligation.

Task 5 has exactly two automated layers:

1. **Portable composed input acceptance:** new
   `cmd/internal/wrapcmd/shortcut_passthrough_test.go` composes real Couch
   `Console` input handling with the production wrapper stdin translator and
   stateful in-process terminal/action seams. Check the import graph before
   placing the test; use an external test package or existing command fixture
   if an import cycle would result. Assert exact delivered bytes, reserved
   actions, panel-vs-actor behavior, split sequences and pasted chords. This
   layer does not claim to execute Zellij.
2. **Real Zellij config conformance:** new
   `cmd/internal/couchcmd/shortcut_conformance_live_test.go` uses the extended
   `ControlledZellij` client PTY, reviewed config and a raw byte-capture pane.
   It proves keybindings deliver unreserved input and do not unexpectedly
   create panes/tabs or change layout, including inherited-default examples.
   It does not claim to execute a real coding agent or the whole production
   Couch/helper/wrapper chain. Preserve fixture cleanup, recurring target and
   workflow filtering requirements from Task 5.

Accordingly, Task 5's live rows require real **client input → Zellij config →
raw capture** evidence; its first portable row requires the composed
**Console → wrapper → fake agent terminal** evidence. Role-local help/tab
side effects remain covered by Task 4's stateful/native-nvim integration tests.
These layers together satisfy the automated verification scope. The final
operator smoke checkbox is the only remaining whole running-workbench acceptance
obligation; do not describe the layered fixtures as full application end-to-end.

### 2026-09-14 — PQ-1/PQ-2: authorization ordering and function-level verification

Reason: plan-quality review found that prefix delivery can change focus before
a recognized chord is authorized, and requested named production functions with
mechanical test strategies. This section is the active implementation/testing
contract for these decisions. It supersedes the earlier scope argument to
`FeedHit`, the proposal for a new state-owning scanner, and the prose test-case
inventories in Tasks 1–5. Their integration boundaries, red-before-green steps,
commands and final acceptance/cleanup obligations remain active.

**PQ-1 / ARCH-ORDER:** Keep `Interceptor.FeedHit` independent of focus. It
recognizes a candidate and retains its exact raw bytes; add `RawHit() []byte`
with the same immediate-read lifetime as `Mouse()`. Console must copy/read that
candidate before another feed, call `route(before)`, then obtain current focus.
Only then authorize the hit. If the candidate is a lifecycle hit and focus is
now an actor, call `route(rawHit)` instead of its lifecycle handler. Navigation
and mouse handlers retain their contracts. Finally process `rest`, repeating
this sequence. A panel Return in `before` can select an actor; the immediately
following Alt+x must therefore reach that actor. Conversely Ctrl+Space can make
a later lifecycle candidate a panel action. No scope snapshot taken before
`route(before)` is authority, including snapshots taken once per feed iteration.

**PQ-2 / minimal framing API:** In the new pure
`cmd/internal/workbenchshortcut/framing.go`, add
`FindChordOutsidePaste(data []byte, inPaste bool) (before []byte, chord Chord,
rawChord []byte, rest []byte, found bool)` with the same tuple contract as
`FindChord`. Scan complete bracketed-paste markers locally before testing chord
encodings; never match a chord within paste. The wrapper's existing translator
owns persistent `inPaste` and pending state. The finder only predicts the next
eligible candidate; translating its preceding bytes advances the persistent
state before dispatch. No second persistent scanner state is introduced.

Add `PendingInputSuffix(data []byte) int` in that pure file: return the length
of the longest trailing proper prefix of any paste marker or supported chord
encoding (including bare ESC), otherwise zero. `passThroughChunk` emits the
preceding bytes and returns only that suffix as pending. Advance paste/Return
observations over emitted bytes only. This handles ordinary text followed by a
partial marker, which the current whole-buffer prefix check misses. Existing
translation mode must satisfy the same invariant using its existing pending
logic; consolidate matching helpers where needed, without replacing its Return
adapter. The held suffix is bounded strictly below the maximum finite marker/
chord length. Existing escape-timeout/EOF resolution flushes incomplete bytes
literally; it never executes an incomplete chord.

The following function strategies replace the prior repeated prose case lists.
Generated partitions include every two-way split and bytewise input; the finite
alphabet comes from all declared encodings, paste markers and arbitrary literal
bytes, avoiding hand-maintained examples as the coverage source.

| Production function/boundary | Named verification target | Adversarial class and mechanical guard |
|---|---|---|
| `workbenchshortcut.Decide`, `GlobalBindings` | `TestAgentReservationPolicyMatrix` | Generate role × declared-chord matrix; agent actions equal exactly the three `AgentReserved` entries, all others passthrough; compare non-agent decisions with retained contract. |
| `FindChordOutsidePaste` | `TestFindChordOutsidePastePartitions` | Generated mixed pasted/unpasted encoded streams with carried caller paste state; only unpasted complete chords may be candidates and tuple concatenation conserves every byte. |
| `PendingInputSuffix` | `TestPendingInputSuffixBound` | Generate proper prefixes and arbitrary leading/trailing bytes; held bytes are exactly a valid longest suffix and remain below the finite encoding bound. |
| `proxy.translateStdin`, `proxy.passThroughChunk`, `proxy.translateChunk` | `TestWrapperShortcutStreamPartitions` | Feed identical mixed streams under generated partitions and supported adaptation modes; concatenated literal bytes and action sequence are partition-invariant, pasted data unchanged, no pasted submission observation, timeout/EOF loses no byte. |
| `Interceptor.FeedHit`, `Interceptor.RawHit`, `Interceptor.Flush` | `TestInterceptorCandidateByteConservation` | Generate known/unknown/pasted candidate encodings and partitions; prefix + raw candidate + suffix/flush reproduces input, paste produces no action candidate, held state stays bounded. |
| `Console.processInput` closure and `route` closure | `TestConsoleShortcutAuthorizationAfterPrefix` | Generate panel/actor transition prefixes followed by candidate chords; stateful terminal/menu trace proves prefix effects precede authorization, exactly one destination receives each unconsumed byte, and lifecycle effects require panel focus at that point. |
| `Console.hitHandlers`, shared Couch navigation declarations | `TestEveryInterceptedChordHasAHandler` plus `TestCouchNavigationReservationContract` | Enumerate declared hits/encodings; all handlers are accounted for, precisely three navigation chords remain actor-owned, no forward-Delete alias appears. |
| `RenderLuaGlobalMaps`, `keyhelp.Sections` | existing generated drift test plus `TestReservedShortcutHelpMatchesPolicy` | Enumerate declaration rows; generated maps/help have no missing or extra reserved actions and help/changelog no longer depend on removed KDL Run binds. |
| `termcmd.handleChord`, `termcmd.runDecision`, `workbench_route.route`, new `PairOpenHelp`/`PairOpenChangelog` draft functions | `TestRoleLocalHelpAndChangelog` plus native-nvim routing tests | Stateful receiving-role/action sequences; non-agent commands retain argv/geometry and agent input opens no workbench pane, with one action per accepted chord. |
| effective `zellij/config.kdl` | `TestZellijShortcutConfigHasNoImplicitConsumers` | Parse all binding blocks/default policy; require clear-defaults and reviewed byte-forwarding actions only, rejecting consuming workbench siblings mechanically. |
| portable Console → wrapper input composition | `TestConsoleWrapperShortcutPassthrough` | Replay generated mixed streams through real routing and fake terminal/action state; compare bytes and actions to the pure role/paste oracle, preserving prefix-driven focus order. |
| real ControlledZellij client PTY → raw capture | `TestAgentShortcutInputConformanceLive` | Drive declared unreserved keys plus retired-default control keys through actual client input; capture expected encoding and compare pane/tab/layout state before/after, without claiming production-agent e2e. |

Use existing native-nvim/term helper tests when they already implement a table
row's oracle; avoid duplicating the same matrix across packages. Record red/green
evidence at the named function or composed boundary, then run the package commands
already prescribed by each task. Operator smoke remains unchecked until the
operator exercises the running workbench.
