---
id: 000604
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '8f96160e5215f61dbe333e2470cd89ce56af7e97' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally

## Problem

The operator (2026-10-15): "the ctrl+y to copy last notice in switcher never worked for me." Couch's console
(`couchtty/console_menu.go` `copyLastNotice`) copies through the terminal's OSC 52 clipboard escape
(`ESC ]52;c;<base64> ESC \`) and then says "Full notice copy requested; your terminal may not support it." On the
operator's setup (Ghostty, with Couch's console possibly behind a multiplexer layer) the clipboard never receives it, and
the UI can't tell. Tests only assert the escape bytes were written, never that a clipboard got them.

## Spec

- On macOS with a local console, copy with the system clipboard directly (`pbcopy`, or NSPasteboard via a tiny
  helper), and report success only when that returns OK; keep OSC 52 as the fallback for remote or unknown setups,
  labelled as unverifiable.
- Find out why OSC 52 doesn't reach Ghostty here (a multiplexer not passing it through, or Ghostty's
  `clipboard-write` setting), and note it in the atlas.

## Done when

- A test with a fake clipboard seam asserts the notice text reaches it; the operator confirms Ctrl+Y works live.

## Plan

- [ ]

## Log

### 2026-10-15
