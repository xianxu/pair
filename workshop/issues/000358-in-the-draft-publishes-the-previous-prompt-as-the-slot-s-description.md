---
id: 000358
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: '576a97ceba953728cd941ba00b2211c9ad516a3a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T11:33:55-07:00
flow: {kind: quick, provenance: inferred, spec: "b18212a4", done: "6e8b6d5b"}
---

# !! in the draft publishes the previous prompt as the slot's description

## Problem

`! text` sets the couch thread's description while sending `text` (#337). The
operator often forgets the `!` prefix, and the prompt that should have become
the description goes out untagged. Today the only fix is to retype it with a
`!`, which also re-sends it to the agent.

## Spec

`!!` is the "I forgot the `!`" shortcut: it sets the hosting thread's
description after the fact and **never submits anything to the agent**. It has
two forms:

- **`!!` alone** (whitespace around it allowed) publishes the previous prompt,
  history position -1, as the description.
- **`!! sentence`** publishes `sentence` as the description. The prompt it
  describes has already been sent, so the sentence is not sent again.

Contrast with `! text` (#337), which sends `text` *and* sets the description.

- **Source (bare `!!`):** the newest pair log entry (`read_history()` n=1, `nvim/init.lua`),
  i.e. what -1 shows in history navigation. Use its agent-facing text: `===`
  comment lines stripped (`normalization.lua`), and a leading `!` dropped if that
  prompt was itself a bang line.
- **One line:** a description is a one-liner, from either form. Pick a rule at design: first
  non-blank line, whitespace collapsed, truncated to the width couch shows. Pin
  the rule with a test on a multi-line prompt.
- **No agent traffic:** no `write-chars`, no submit. `!!` itself is not written
  to the agent history (it would become the new -1 and a second `!!` would
  describe `!!`); the same holds for `!! sentence`. Recommend the draft clears
  as after a send.
- **Parsing:** `bang_tag.parse` owns the draft's leading `!` (#337). `!!` is
  checked before the `!` rule, so `!! sentence` stops parsing as `! ! sentence`
  (today it would send `! sentence` to the agent). Single-line only, like `!`.
  The `!!` then text boundary is whitespace-tolerant: `!!sentence` and
  `!! sentence` mean the same.
- **Edges:** outside couch, no history, or an empty -1 after stripping (bare
  form) means nothing is published and the operator is notified with the reason. A failed
  publish is notified, not reported as success.

Related: #357 (bare `!` clears the description). Same channel, same "draft
action that doesn't submit" routing in `submit_operator_text`, so design them
together; whichever lands second reuses the first's routing.

## Done when

- `!!` in a couch-hosted draft makes the thread's description the one-line form
  of the previous prompt, visible in couch on its next refresh
- `!! sentence` makes `sentence` the description, visible the same way
- For both forms, no bytes reach the agent pane (a stateful fake fails on any
  write or submit), and nothing is appended to the agent history
- A multi-line and a bang-tagged previous prompt each yield the pinned one-line
  description
- Outside couch or with no usable previous prompt, nothing is published and the
  operator is told why
- `! text` (#337) and bare `!` (#357) behavior is unchanged, with tests green

## Plan

- [ ] Decide the one-line rule: first non-blank line, runs of whitespace
      collapsed to one space, trimmed, capped at 120 characters (UTF-8 aware,
      `…` marks a cut). Couch clips to its terminal width at render time
      (`couchtty/menu_render.go` `clipMenuLine`), so the cap only bounds what
      is stored; it is not a display width.
- [ ] Publish synchronously for `!!` (unlike `!`'s detached publish): nothing
      is sent to the agent, so there is no prompt to delay, and a sync result
      lets a failure keep the draft and notify instead of claiming success.
      Bounded by a 5 s `vim.system` timeout.
- [ ] `bang_tag.parse`: recognize `!!` and `!! sentence` ahead of `!`
- [ ] `submit_operator_text`: route bare `!!` to history -1 → normalize →
      publish, and `!! sentence` to normalize → publish, both without the send
- [ ] Tests: `bang_tag` unit + `bang_tag_integration_test` (no agent traffic,
      history not appended)

## Log

### 2026-09-30

- Filed at the operator's request, to recover from forgetting the `!` prefix.
  History source verified: `read_history()` n=1 is the -1 entry.
- `!! sentence` added at the operator's request: it sets the description
  without submitting, so `!!` in both forms is the after-the-fact fix for a
  forgotten `!`. It supersedes the open "`!! text` needs a decision" point.
