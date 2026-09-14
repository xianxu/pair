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
