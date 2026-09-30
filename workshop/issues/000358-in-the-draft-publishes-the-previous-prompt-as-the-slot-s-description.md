---
id: 000358
status: open
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: 'a0445d01eb9b66e9fc01aee93221b1acb4bbcf4c' # card fields mirrored from issue-cards; edit via sdlc
---

# !! in the draft publishes the previous prompt as the slot's description

## Problem

`! text` sets the couch thread's description while sending `text` (#337). The
operator often forgets the `!` prefix, and the prompt that should have become
the description goes out untagged. Today the only fix is to retype it with a
`!`, which also re-sends it to the agent.

## Spec

A draft submission of exactly `!!` (whitespace around it allowed) publishes the
previous prompt, history position -1, as the hosting thread's description, and
sends nothing to the agent.

- **Source:** the newest pair log entry (`read_history()` n=1, `nvim/init.lua`),
  i.e. what -1 shows in history navigation. Use its agent-facing text: `===`
  comment lines stripped (`normalization.lua`), and a leading `!` dropped if that
  prompt was itself a bang line.
- **One line:** a description is a one-liner. Pick a rule at design: first
  non-blank line, whitespace collapsed, truncated to the width couch shows. Pin
  the rule with a test on a multi-line prompt.
- **No agent traffic:** no `write-chars`, no submit. `!!` itself is not written
  to the agent history (it would become the new -1 and a second `!!` would
  describe `!!`). Recommend the draft clears as after a send.
- **Parsing:** `bang_tag.parse` owns the draft's leading `!` (#337). `!!` is
  checked before the `!` rule. `!! text` (text after `!!`) needs a decision:
  either an error, or `! ! text` as today. Don't let it silently mean something
  new.
- **Edges:** outside couch, no history, or an empty -1 after stripping means
  nothing is published and the operator is notified with the reason. A failed
  publish is notified, not reported as success.

Related: #357 (bare `!` clears the description). Same channel, same "draft
action that doesn't submit" routing in `submit_operator_text`, so design them
together; whichever lands second reuses the first's routing.

## Done when

- `!!` in a couch-hosted draft makes the thread's description the one-line form
  of the previous prompt, visible in couch on its next refresh
- No bytes reach the agent pane (stateful fake fails on any write/submit), and
  `!!` is not appended to the agent history
- A multi-line and a bang-tagged previous prompt each yield the pinned one-line
  description
- Outside couch or with no usable previous prompt, nothing is published and the
  operator is told why
- `! text` (#337) and bare `!` (#357) behavior is unchanged, with tests green

## Plan

- [ ] Decide the one-line rule and the `!! text` handling
- [ ] `bang_tag.parse`: recognize `!!` ahead of `!`
- [ ] `submit_operator_text`: route `!!` to read history -1 → normalize → publish,
      without the send
- [ ] Tests: `bang_tag` unit + `bang_tag_integration_test` (no agent traffic,
      history not appended)

## Log

### 2026-09-30

- Filed at the operator's request, to recover from forgetting the `!` prefix.
  History source verified: `read_history()` n=1 is the -1 entry.
