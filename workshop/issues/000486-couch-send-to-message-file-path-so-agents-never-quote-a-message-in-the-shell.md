---
id: 000486
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '63db27ebcabd5a6da19884f4b0a87511f174a999' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T07:53:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# couch --send-to: --message-file PATH, so agents never quote a message in the shell

## Problem

Agents in Couch slots send with `couch --send-to X --message TEXT`. Claude Code exempts the call from its sandbox only when the shell text matches `couch *`, and ariadne#336 measured what defeats that match:
- **Substitution text in an argument, even quoted:** `$(`, `$((`, backticks, `<(`, `>(`.
- **Anything around the command:** a pipe, `;` chaining, or a redirect to a file. A bare `2>&1` is the one thing that passes.

A sandboxed send can't reach the supervisor's socket. It fails "outcome uncertain", and the message is never delivered.

Agents write markdown by habit, and code spans are backticks. So a rule against substitution text in messages fights how they write. Two of ariadne:1's reports to ops:0 on 10-12 failed this way: one had backticks, the other spelled out `<(` as prose. Reading from stdin doesn't help, because the heredoc or pipe that feeds it makes the call compound, and that is sandboxed too.

## Spec

- `couch --send-to X --message-file PATH` reads the message body from a file. Exactly one of `--message` or `--message-file` is given; both, or neither, is refused.
- The body is the file's contents verbatim: no shell interpretation, and no trimming beyond one trailing newline. An empty file is refused, like an empty `--message`.
- Admission, receipts, allowance and the canonical-content rules are unchanged. The receipt carries the body, not the path.
- The agent's flow is to write the file with its editor tool, then make a bare `couch --send-to X --message-file PATH` call. Nothing gets quoted in the shell. `couch --skill` recommends this for any message that contains code or shell syntax.
- Optional: `couch --send-to X --message-file PATH --remove-after-send` cleans up the file. Without it, the agent writes the file in its own scratch space, and the file's lifetime is that space's.

## Done when

- A message containing backticks, `$(…)`, `<(…)` and newlines is delivered verbatim from a file in a sandboxed agent slot. This is a live check: the same text sent with `--message` stays sandboxed.
- Tests cover a verbatim body, both flags given, neither given, an empty file, and a missing or unreadable file.
- `couch --skill` and `couch --help` document the option.

## Plan

- [ ] Parse `--message-file PATH` in the `--message TEXT` position (`couchcmd/cli.go`); the parser stays IO-free and records the path. Both or neither stays refused by the fixed shape.
- [ ] Read the file at send time (`couchcmd/messages.go`), bounded to `MaxBodyBytes`+2, strip one trailing `\n`; refuse an empty/whitespace, oversized, missing or unreadable file before any RPC. The broker's body validation is unchanged.
- [ ] Tests: parse forms + rejections (`cli_test.go`); verbatim body with backticks, `$(…)`, `<(…)`, newlines; empty, missing, unreadable, oversized (`messages_test.go`).
- [ ] Document in `couch --help` and the couch skill (the default for a message containing code or shell syntax).
- `--remove-after-send` (optional in the Spec) is left out: the scratch space owns the file's lifetime.

## Log

### 2026-10-12
- Filed from ariadne#336 (ariadne:1) at the TL's (ops:0) request. Once this ships, ariadne's AGENTS §15 names `--message-file` as the default way to send a message containing code.
