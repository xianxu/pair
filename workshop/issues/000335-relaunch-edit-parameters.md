---
id: 000335
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '624bf2304f7b599551e60502870617f9a21ed0a9' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch relaunch: same conversation, new parameters

## Problem

Couch relaunch preserves the conversation but reuses its saved launch arguments.
Changing path preferences does not update an existing thread, even after park
and resume. Switch coding agent allows editing arguments, including Codex to
Codex, but creates a fresh conversation. There is no menu for changing startup
parameters while retaining the current conversation (for example, changing
Codex's sandbox option).

## Spec

After Relaunch is selected, show a "same conversation, new parameters" menu.
Prefill editable parameters from the thread's saved launch profile. Keep the
coding agent, Couch thread, and exact native conversation identity unchanged.
Do not silently replace the saved parameters with current path defaults.

Allow explicit confirmation or cancellation. Confirming unchanged parameters
retains ordinary relaunch behavior. Validate edited parameters before stopping
the running session; reject arguments that conflict with resuming the same
conversation. Cancellation or invalid input must leave the session running.
After successful relaunch, retain the accepted parameters for later relaunches.

Reuse Couch's existing relaunch/resume lifecycle and parameter editor where
appropriate; do not route this through the fresh-conversation agent switch.
Apply the menu consistently to menu and keyboard entry points for Relaunch.

## Done when

- Selecting Relaunch opens the parameter menu before any stop/restart action.
- Edited parameters take effect while the exact conversation and agent remain
  unchanged; unchanged parameters still support ordinary relaunch.
- Cancel, invalid parameters, and conflicting resume selectors cannot stop the
  source or silently create a fresh conversation.
- Successful relaunch saves accepted parameters; failures clearly report the
  resulting state and available recovery without claiming success.
- Production-boundary tests cover edited/unchanged parameters, cancellation,
  validation, conversation preservation, and failed relaunch; help reflects
  the new menu.

## Plan

- [ ] Design parameter acceptance and persistence around the existing relaunch
  lifecycle, including failures after the source stops.
- [ ] Implement the menu and shared relaunch path with regression tests.
- [ ] Verify same-conversation parameter changes and document the behavior.

## Log

### 2026-09-28

Requested during #260 investigation after discovering that changing Couch path
preferences cannot change an existing conversation's startup parameters.
Operator explicitly requested the menu after selecting Relaunch. Ticket only;
implementation is separate from #260.
