---
id: 000882
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'cbc5ffe5329a2297743d5891a276f6ab80627a46' # card fields mirrored from issue-cards; edit via sdlc
---

# Notification links: wrap-safe hyperlinks, never truncated, open/copy action

## Problem

## Spec

**Links in Couch notifications can't be opened reliably (operator, 10-24).** A URL in a `couch --notify` message is
sometimes truncated in the display and sometimes wrapped across two lines, so Shift+Cmd+click (the terminal's
open-link gesture) opens nothing or a broken fragment. It mattered tonight: the iPhone-remote test sent a short-lived
link (3-minute window) by notification.

Make every URL in a notification (the bell panel, the switcher feed, and any toast) openable as one unit, whatever
the pane width: render it as a terminal hyperlink (OSC 8) so a wrapped link still opens whole, never truncate inside
a URL (truncate the surrounding text instead, or show the URL on its own line), and offer a keyboard action to open
or copy the item's link.

## Done when

- A notification whose URL is longer than the panel width opens the full URL with Shift+Cmd+click, and copies whole.
- Truncation never cuts a URL; tests over narrow widths with one and two URLs.
- A keyboard action opens or copies the selected item's first link.

## Plan

- [ ]

## Log

### 2026-10-24
