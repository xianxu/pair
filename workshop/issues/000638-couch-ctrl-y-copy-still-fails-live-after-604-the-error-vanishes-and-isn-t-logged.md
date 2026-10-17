---
id: 000638
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ba68e86d8a2cbc057d27880e2973a19bd9e27195' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch Ctrl+Y copy still fails live after #604; the error vanishes and isn't logged

## Problem

## Spec

pair#604 (PR #381) landed Ctrl+Y "copy last notice" in the Couch switcher using the local system clipboard. The operator's live check on 2026-10-16 at about 23:15, on Couch built at 712a6478 and running since 22:22, failed:
- pressing Ctrl+Y showed an error;
- nothing reached the clipboard;
- by the time they went back to screenshot it, the error was gone.

Nothing about it is in supervision.jsonl, so the error text is lost too.

Two defects:
1. The copy fails in the operator's real environment, even though the tests pass with a fake clipboard. Find which tool Couch picks (pbcopy?), and how it fails from the Couch process: PATH, sandboxing or the launch environment, a timeout, stdin handling.
2. A failure has to stay visible and be recorded: the notice persists until it is dismissed, and the error, with the tool and its exit status and stderr, goes to the supervision log.

The original OSC 52 loss layer is still undiagnosed (#604's close review).

## Done when

- In the operator's Couch, Ctrl+Y puts the last notice on the clipboard. Live check: paste it.
- A forced failure, such as a missing tool, shows a notice that persists and logs the tool, exit status and stderr; a test covers it.
- A live conformance check runs the real clipboard tool from a Couch-launched process environment.

## Plan

- [ ]

## Log

### 2026-10-16
