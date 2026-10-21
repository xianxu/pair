---
id: 000758
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7df59f0c245152fff9c5f4369b6541c2c6f17917' # card fields mirrored from issue-cards; edit via sdlc
---

# remote-session: Couch remote server and allowlisted proxy to each session file's nvim (subscribe, edit re-wrapped, submit, navigate)

## Problem

remote-session (ops project, operator-approved 10-20): the phone talks only to Couch; Couch serves the session views by
proxying to the nvim that has each file open (registered by parley, parley.nvim issue filed alongside).

## Spec

- A Couch remote server (Go) behind the 👽-mode tunnel (sibling issue). The phone never reaches nvim directly.
- **Allowlisted proxy, not a pipe:** Couch translates exactly these into msgpack-RPC on the registered nvim: subscribe
  to a buffer's text and its highlights/extmarks (`nvim_buf_attach`, `nvim_buf_get_extmarks` with details), apply an
  edit, submit (calls parley's own submit, identical to Alt+Return), follow a link, go to the parent, jump back and
  forward (nvim's jump list is the truth, so desk and phone share history). Anything else is refused before nvim.
- **Edits keep the file hard-wrapped:** an edit inside a reflowed paragraph arrives as a replacement of that
  paragraph's line range and is re-wrapped at the file's width (like `gq`).
- **Headless fallback:** if no nvim has the file, Couch starts a headless nvim with parley for it, and hands over when
  a desktop nvim opens the file.
- Scope: every hub's `session.md` and its side docs. No agent panes or input boxes in v1.

## Done when

- Tests against a real headless nvim: subscribe receives text and highlights; an edit round-trips re-wrapped; submit
  produces the same session-sync submit as Alt+Return; a non-allowlisted call (e.g. an Ex command) is refused.

## Plan

- [ ]

## Log

### 2026-10-20
