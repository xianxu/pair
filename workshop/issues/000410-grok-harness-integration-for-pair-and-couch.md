---
id: 000410
status: open
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'a40342a4435fa5f340fa393db4030e196ee747fd' # card fields mirrored from issue-cards; edit via sdlc
---

# Grok harness integration for pair and couch

## Problem

Pair and Couch host `claude`, `codex`, `agy`, `muse` and `qoder`, but not Grok.
To bring Grok in, it has to join the agent registry and pass every integration
surface that the existing harnesses already pass, so that it runs under pair as
smoothly as they do and Couch can launch, park and resume it.

## Spec

Follow [How to Bring Up a New Harness CLI](../../atlas/how-to-bring-up-a-new-harness-cli.md)
(§0 registry plus the §2 ten-item checklist) for a `grok` harness.

Open questions to settle in brainstorm before planning:

- **Which CLI is "grok"?** Pin the executable, vendor, version and install path,
  and find out whether it is a full-screen TUI like Claude/Codex or a line REPL.
  That decides the aspects that matter most: Return remapping and overlay
  detection.
- **Session identity / resume:** where Grok keeps its transcripts, whether it
  has a resume flag (and its spellings for `resumeform.Forms`), and what shape a
  session ID takes. With no resumable session, there is no Couch park/resume
  (guide §0), and that limit must be written down explicitly.
- **Couch peer delivery:** today only Claude Code and Codex CLI have a receiver
  profile (`peerReceiverAgents`, `cmd/internal/wrapcmd/peer_runtime.go`). Decide
  whether Grok gets one in this issue or in a follow-up.

## Done when

- `grok` is in the launcher `supportedAgents`, the sessioninventory `Agent` enum
  and its `supportedAgents`, and `TestAgentInventoryParityWithSessionTables`
  passes with no known-gap entry.
- A captured TTY fixture pins Return remapping (and the overlay detector, if
  Grok needs one) under `cmd/internal/wrapcmd/testdata/tty/`.
- The scanner, resume token and `pair-slug` support cover Grok, or a documented
  gap explains why a surface does not apply.
- Live smoke test by the operator: `pair` launches Grok, and Couch's start and
  switch-agent menus show it. A Couch-hosted Grok thread launches, parks and
  cold-resumes (if resume exists).
- Peer delivery to Grok is either implemented or filed as a follow-up.
- The atlas guide lists Grok among the supported harnesses.

## Plan

- [ ]

## Log

### 2026-10-07
