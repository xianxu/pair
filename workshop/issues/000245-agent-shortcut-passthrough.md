---
id: 000245
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
started: 2026-09-13T16:24:31-07:00
---

# Pass all Pair shortcuts through the agent pane

## Problem

Pair intercepts workbench shortcuts while the agent pane is focused. Agents can
assign those same keys: the operator reports Codex now uses Alt+Up, which Pair
currently routes to the draft's layout action.

## Spec

Proposed scope, pending operator design review: the focused agent owns all
keyboard shortcuts that reach Pair. Pair must not consume them for workbench
actions, including layout, focus, scrollback, terminal tabs, and quit commands.
This applies to every agent, without an agent-specific conflict list. Mouse
focus remains available to leave the agent pane.

Couch's outer shortcut interception is a separate layer and remains outside
this first change. Draft Neovim and right-terminal shortcuts retain their own
role-specific behavior. Existing agent input adaptation (such as Return handling)
is a separate concern; this change removes workbench shortcut interception.

Prefer a single agent-role pass-through policy in the shared routing model,
with the wrapper input path honoring it. Audit Zellij's bindings for earlier
consumption or rewriting that could prevent equivalent delivery to the agent.
ARCH-PURPOSE requires covering the whole shortcut class, not just Alt+Up;
ARCH-DRY favors one routing policy over a list of agent-specific exceptions.

Alternatives considered: reserve selected navigation/quit chords (retains
collisions), or add an opt-in pass-through mode (adds a mode users must track).
Neither matches the requested ownership by focused pane as directly.

## Done when

- All Pair workbench shortcuts reach the focused agent without triggering Pair actions.
- Regression coverage crosses the wrapper input boundary, including Alt+Up,
  other recognized chords, split escape sequences, and paste handling.
- Draft and terminal routing regression checks continue to pass.
- Zellij routing and operator help/docs agree with the new scope; manually
  verify agent delivery and mouse focus in a running Pair session.

## Plan

- [ ] Confirm proposed scope and author the implementation plan.
- [ ] Implement and verify agent shortcut ownership across routing layers.
- [ ] Update operator documentation and close through the SDLC review gate.

## Log

### 2026-09-13

Created and claimed from the operator's request. Initial inspection found
`workbenchshortcut.Decide` applies globals before agent-role routing, and
`wrapcmd.handleWorkbenchChord` executes those decisions. No implementation
changes yet. Design approval pending.

## Revisions

### 2026-09-14 — Agent owns keys by default, with reserved navigation chords

The operator refined the blanket pass-through proposal: pass most keys to the
focused agent while retaining a small explicit exception list. This supersedes
the original Spec's no-exceptions rule. The candidate reserved chords are:

- Ctrl+Space: open the Couch switcher.
- Ctrl+Delete: return to the previous Couch thread. Interpret Delete as the
  Apple backspace key, matching the existing Ctrl+Backspace binding; forward
  Delete is a distinct chord.
- Ctrl+Return: jump to the newest notification's thread.
- Shift+Alt+T: create a right-terminal tab.
- Shift+Alt+Left / Right: select the previous / next right-terminal tab.

Alt+Up/Down and ordinary Alt+Left/Right reach the focused agent. Every other
workbench shortcut also passes through unless explicitly added to this list.
Keep the policy agent-independent and derive routing and help from one declared
exception set (ARCH-DRY).

The proposed implementation scope now includes Couch's outer interception where
needed to enforce this rule end to end (ARCH-PURPOSE), superseding its original
exclusion. In particular, existing outer detach/relaunch shortcuts must not
silently remain extra exceptions while the agent pane has focus. Other pane
roles retain their existing behavior. Verification must cross Couch, Zellij,
and the agent wrapper, covering both delivery of unreserved keys and the retained
actions of reserved keys.

This records the operator's candidate list ("maybe"), not a completed design or
implementation. Finalize the exception set and focused-pane propagation in the
implementation plan; no code changed as part of this revision.


### 2026-09-14 — Routing audit and remaining design decision

Resumed after #250 shipped; ran `sdlc state` and `sdlc start-plan --issue 245`.
The agent wrapper calls `workbenchshortcut.Decide`, which currently resolves
all globals before role-specific actions. The agreed exception policy must
precede this branch. Existing input framing and Return adaptation remain;
regressions must include reserved chords inside bracketed paste as literal data.

Two upstream consumers also require changes: Couch's `Interceptor` consumes
Alt+d, Alt+x, Alt+n and Ctrl+Alt+n; Zellij directly runs help/changelog for
Alt+h/Alt+l. The latter must become pane-local actions so the agent receives
those chords too. Help classifications currently call these keys global and
must derive the revised scope.

Couch knows switcher versus hosted actor focus, but not the hosted Zellij's
inner pane. A list-panes query is an asynchronous snapshot, adds latency and
cannot establish which pane receives a particular key. No existing general
command bridge forwards pane-local lifecycle requests to the running Couch
owner; invoking an owner CLI operation collides with its supervisor lease.
ARCH-DRY/ARCH-ORDER rule out inventing a second, potentially stale focus owner.

Asked the operator to choose between simplifying Couch's outer interception to
its three reserved navigation chords everywhere (lifecycle actions remain in
the switcher), or preserving Couch lifecycle hotkeys in nonagent panes via a
new owner-addressed command bridge. The first changes nonagent Couch hotkey
semantics, so that scope choice requires an explicit answer. No code changed.


### 2026-09-14 — Proceed with simplified outer ownership

The operator instructed “continue” after the routing options were presented.
Proceed with the recommended simpler design, as stated back to the operator:
Couch reserves Ctrl+Space, Ctrl+Backspace and Ctrl+Return while displaying a
hosted actor. Its switcher retains detach/park/relaunch actions and keyboard
behavior. No inner-focus cache, query on keystrokes, or new lifecycle RPC bridge.
This supersedes the earlier requirement to retain identical Couch lifecycle
hotkeys outside the agent pane: those keys reach existing Pair actions there,
whose detach/quit/restart semantics differ from Couch's owner operations. The
switcher is the documented route to Couch lifecycle operations.

The receiving agent wrapper handles only Shift+Alt+T, Shift+Alt+Left and
Shift+Alt+Right. All other workbench chords pass through to every agent,
including Alt+Up/Down, Alt+Left/Right, Alt+j/k, Alt+d/x/n, Alt+h/l and compact
shortcuts. Return adaptation remains a separate existing concern. Terminal-tab
reservations apply only to actual keystrokes, never bracketed-paste content.
Zellij forwards help/changelog chords to pane-local handlers; other Pair pane
roles retain their actions. Shared binding metadata drives agent reservations
and help context; tests enumerate the full recognized chord set and cross the
Couch-to-wrapper boundary. Isolated real-Zellij conformance checks semantic key
delivery because Zellij may normalize encodings. No running user thread is
restarted during automated testing. Pause for operator smoke after implementation,
verification and the SDLC boundary review; do not ship #245 before that smoke.


### 2026-09-14 — Spec review and inherited Zellij bindings

Fresh spec review: Approved, no blocking findings. Review emphasized adding
Alt+h/l to shared chord enumeration, retaining literal paste handling at both
layers and distinguishing Backspace/Delete/Return encodings.

The installed `zellij setup --dump-config` also revealed inherited Alt+f,
Alt+=/+/-, Alt+[/], Alt+p and Alt+Shift+p actions. Clear inherited bindings
(`clear-defaults=true`) and retain explicit Pair byte-forward bindings. This
retires undocumented multiplexer actions for all panes while preserving
Pair-owned draft/terminal actions. Announced this consequence to the operator.
A growing unbind list would fail the promised default ownership on upgrades
(ARCH-PURPOSE); a closed forwarding configuration prevents that drift.
