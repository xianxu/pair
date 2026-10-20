---
id: 000717
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ec2569fb29141a8b6c65dba2ca0dfc0665592799' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --keys: support special keys (tab, shift-tab, arrows, backspace, function keys, ctrl/alt chords)

## Problem

`couch --keys` accepts only `text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line` and `wait:MS`. The hub
regularly needs keys outside that set to drive a slot's agent UI without the operator: Tab and Shift+Tab (Claude's
mode cycling, completion), Left/Right, Backspace, Ctrl+C / Ctrl+O style chords, Page Up/Down, and function keys
(Codex's `f2 to view` warnings). Today each such moment becomes a "please press X" ask to the operator (10-19: the
operator pressed Esc on a `/usage` panel for the TL).

## Spec

1. A named-key vocabulary for `--keys`: `tab`, `shift-tab`, `left`, `right`, `backspace`, `delete`, `home`, `end`,
   `pageup`, `pagedown`, `f1`–`f12`, and chords `ctrl-<letter>` / `alt-<letter>` (one table: name → bytes, the single
   source the parser, `couch --keys --help` and the skill read; see pair#716).
2. Each key keeps today's admission rules: driving keys need an empty composer and a finished turn; only the
   recovery-only set (`clear-line`, `esc`, and pair#691's panel-esc) may start on an occupied state. Interrupting
   chords (`ctrl-c`) are a deliberate, separately named action with their own refusal rules, never a plain step: an
   interrupt kills a turn, so it must not ride along in a key sequence by accident.
3. Unknown key names refuse before anything is sent, listing the accepted names.

## Done when

- The new names send the right bytes (table test against the terminal sequences Pair's wrapper emits).
- A sequence containing an unknown name refuses with the list and sends nothing.
- `ctrl-c` refuses as a plain step and works only through its named interrupt form, with a test.
- `couch --keys --help` lists every name from the same table (shared with pair#716's test).

## Plan

- [ ]

## Log

- Edges: `~ pair#716` (keys help; one shared table) and `~ pair#691` (esc on an open panel). Land after #691 or
  account for its admissibility function.
- If "special characters" meant arbitrary text in `text:` steps (quotes, `$`, backticks, newlines) rather than named
  keys, fold that in as a Spec item: today the skill advises a file for messages with shell syntax, but `--keys` has no
  file form.
