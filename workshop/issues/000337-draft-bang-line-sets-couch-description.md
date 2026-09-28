---
id: 000337
status: open
deps: [pair#173]
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '455ae1fa790a6fffcb17331845dda2dc3270aa6e' # card fields mirrored from issue-cards; edit via sdlc
---

# Draft bang line tags the couch thread description

## Problem

A couch thread (a "slot" in some contexts) has a one-line description meant to
say what that workspace is focused on. Today there is no cheap way for the
operator to set it while working. They have to use couch's switcher or edit
flow, and `couch --internal publish-description` has no caller (#173). The
operator already states their intent in the first prompt they give the agent
("start working on #xxx"). That same line should also tag the thread.

## Spec

In pair's draft window, a submitted draft that is a **single line** starting
with `!` does two things:

1. Pair strips the `!` and any spaces after it, then submits the rest to the
   agent as a normal prompt. For example, `! start working on #xxx` sends
   `start working on #xxx`.
2. The same remaining text replaces the description of the couch thread that
   hosts this pair session, through the existing description write path
   (`Couch.PublishDescription` / `couch --internal publish-description`,
   `cmd/internal/couchcore/couch.go`).

Scope and edge cases:

- Only single-line drafts qualify. A multi-line draft whose first line starts
  with `!` is sent unchanged.
- A lone `!`, or `!` followed only by spaces, sends nothing and leaves the
  description unchanged.
- When pair runs outside couch, it still strips the `!` and sends the text.
  The description step is a silent no-op, so pair keeps working standalone
  (couch must not degrade pair).
- The description text goes through couch's existing untrusted-text
  sanitization, and couch owns any truncation limits.
- Updating the description must not block or delay the send. If the
  description write fails, the prompt still reaches the agent.

Decisions:

- The draft owns a leading `!`, with no `!!` escape. This gives up Claude
  Code's `!` bash mode from the draft, and that is intended. The operator finds
  it low value: shell commands need precision, so it works better to describe
  the command and let the agent write it. Bash mode is still available by
  typing directly in the agent pane.
- Pair finds its hosting thread through the environment couch already sets.
  Couch launches each thread's child with `COUCH_THREAD_SCOPE` and
  `COUCH_THREAD_TAG`. `couch --internal publish-description <text>` falls back
  to those variables when it gets no explicit thread
  (`cmd/internal/couchcmd/run.go:231`). Pair and its draft run inside that
  child and inherit them. So pair runs `couch --internal publish-description`
  when the variables are set and skips it otherwise; the variables also serve
  as the "am I inside couch?" test.
- `PublishDescription` writes the session-published description, which
  `Describe` prefers over an operator-typed one
  (`cmd/internal/couchcore/couch.go:1185`). A `!` tag therefore wins, which is
  right because it is the latest thing the operator said.
- Showing the description is #173's job. This issue only needs the value to be
  written and persisted, but it is most useful once #173 displays it.

## Done when

- Submitting `! start working on #xxx` from the draft sends
  `start working on #xxx` to the agent. The hosting couch thread's description
  then reads `start working on #xxx`, and the value survives a couch restart.
- Multi-line drafts, drafts without a leading `!`, and a bare `!` behave as
  specified above. Tests cover each case at the draft-submit boundary.
- Outside couch, the same draft sends the stripped text and reports no error.
- An operator smoke test inside couch confirms the description updates live.

## Plan

- [ ] Find where the draft submit path runs, and confirm the draft process
  inherits `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`.
- [ ] Add a pure parser, draft to (payload, description), with unit tests for
  the edge cases.
- [ ] Wire the parser into submit. Call the description writer asynchronously
  and without failing the send; stay a no-op outside couch.
- [ ] Add an integration test that uses a fake couch description sink, then ask
  the operator to run a live smoke test.

## Log

### 2026-09-28
- Operator resolved the open questions: no `!!` escape (bash mode from the draft is unwanted); thread identity comes from the `COUCH_THREAD_SCOPE` / `COUCH_THREAD_TAG` env that couch already injects.
