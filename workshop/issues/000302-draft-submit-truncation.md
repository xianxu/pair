---
id: 000302
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Preserve long draft submissions to agent

## Problem

Submitting a draft from Neovim to the agent pane can truncate the message in
the middle. The observed payload was only 24 logical lines; some lines wrapped
to five or six rows at 93 columns, so visual height alone should not explain
the loss.

## Spec

Trace the complete draft-to-agent submission path and identify the boundary
that drops bytes or lines. Preserve the exact submitted text, including long
wrapped lines, through framing, transport, PTY input, and agent receipt. Treat
partial writes, byte limits, newline conversion, and asynchronous handoff as
separate hypotheses and report the measured limit or failure stage.

## Done when

- A regression fixture reproduces the original 24-line payload and fails on
  the pre-fix behavior.
- The full payload arrives at the agent unchanged, including its final line
  and all long wrapped lines.
- The fix handles partial writes and any relevant framing/buffer boundary;
  it does not merely raise an arbitrary size constant.
- Tests cover payloads around the discovered boundary, embedded newlines,
  long lines, and the agent receipt/transport boundary.
- An operator smoke confirms a comparable draft is delivered intact.

## Plan

- [ ] Capture the exact payload and compare draft text, framed bytes, PTY
      writes, and agent-side receipt to locate the first divergence.
- [ ] Add the smallest boundary-level regression test and fix the owning
      transport/framing path.
- [ ] Verify neighboring sizes and run the operator smoke; document any
      intentional transport limit.

## Log

### 2026-09-20

Created from the report of a 24-line draft being cut off mid-message.
