---
id: 000337
status: working
deps: [pair#173]
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: 'cc44e2f2b938d514b2abdc7fe1fad0810eb5e664' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T10:31:09-07:00
flow: {kind: quick, provenance: inferred, spec: "de93869c", done: "63bd5b23"}
---

# Draft bang line tags the couch thread description

## Problem

A couch thread (a "slot" in some contexts) has a one-line description meant to
say what that workspace is focused on. Today there is no cheap way for the
operator to set it while working. They have to use couch's switcher or edit
flow, and `couch --internal publish-description` has no caller (#173). The
operator already states their intent in the first prompt they give the agent
("start working on #xxx"). That same line should also tag the thread.

## Spec

In pair's draft window, a submitted draft that is a **single line** starting
with `!` does two things:

1. Pair strips the `!` and any spaces after it, then submits the rest to the
   agent as a normal prompt. For example, `! start working on #xxx` sends
   `start working on #xxx`.
2. The same remaining text replaces the description of the couch thread that
   hosts this pair session, through the existing description write path
   (`Couch.PublishDescription` / `couch --internal publish-description`,
   `cmd/internal/couchcore/couch.go`).

Scope and edge cases:

- Only single-line drafts qualify. A multi-line draft whose first line starts
  with `!` is sent unchanged.
- A lone `!`, or `!` followed only by spaces, sends nothing and leaves the
  description unchanged.
- When pair runs outside couch, it still strips the `!` and sends the text.
  The description step is a silent no-op, so pair keeps working standalone
  (couch must not degrade pair).
- The description text goes through couch's existing untrusted-text
  sanitization, and couch owns any truncation limits.
- Updating the description must not block or delay the send. If the
  description write fails, the prompt still reaches the agent.

Decisions:

- The draft owns a leading `!`, with no `!!` escape. This gives up Claude
  Code's `!` bash mode from the draft, and that is intended. The operator finds
  it low value: shell commands need precision, so it works better to describe
  the command and let the agent write it. Bash mode is still available by
  typing directly in the agent pane.
- Pair finds its hosting thread through the environment couch already sets.
  Couch launches each thread's child with `COUCH_THREAD_SCOPE` and
  `COUCH_THREAD_TAG`. `couch --internal publish-description <text>` falls back
  to those variables when it gets no explicit thread
  (`cmd/internal/couchcmd/run.go:231`). Pair and its draft run inside that
  child and inherit them. So pair runs `couch --internal publish-description`
  when the variables are set and skips it otherwise; the variables also serve
  as the "am I inside couch?" test.
- `PublishDescription` writes the session-published description, which
  `Describe` prefers over an operator-typed one
  (`cmd/internal/couchcore/couch.go:1185`). A `!` tag therefore wins, which is
  right because it is the latest thing the operator said.
- Showing the description is #173's job. This issue only needs the value to be
  written and persisted, but it is most useful once #173 displays it.

## Done when

- Submitting `! start working on #xxx` from the draft sends
  `start working on #xxx` to the agent. The hosting couch thread's description
  then reads `start working on #xxx`, and the value survives a couch restart.
- Multi-line drafts, drafts without a leading `!`, and a bare `!` behave as
  specified above. Tests cover each case at the draft-submit boundary.
- Outside couch, the same draft sends the stripped text and reports no error.
- An operator smoke test inside couch confirms the description updates live.
- README documents the complete syntax. Boundary tests prove unavailable,
  failing, and blocked publishers preserve prompt delivery, a failed dispatch
  publishes nothing, and its successful retry publishes exactly once.

## Plan

- [x] Find where the draft submit path runs, and confirm the draft process
  inherits `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`.
- [x] Add a pure parser, draft to (payload, description), with unit tests for
  the edge cases (`nvim/bang_tag.lua`, run by `make test-lua`).
- [x] Wire the parser into submit. Call the description writer asynchronously
  and without failing the send; stay a no-op outside couch.
- [x] Add an integration test that uses a fake couch description sink
  (`tests/bang-tag-nvim-test.sh`, `make test-bang-tag`).
- [x] Operator live smoke test inside couch.

## Revisions

- 2026-09-28: close review requested README coverage and regression proof for
  the existing failure contracts (BR-1/BR-2). Add documentation for the complete
  bang syntax and boundary tests for missing, failing, and slow publishers,
  plus failed dispatch and successful retry. Behavior and acceptance criteria
  remain unchanged.

## Log

### 2026-09-28 — close review fixes

- BR-1: README now covers bang stripping, Couch publication, standalone use,
  bare/multiline handling, comments, and the deliberate bash-mode tradeoff.
- BR-2: the real Neovim boundary now covers missing, nonzero, and blocked
  publishers plus failed dispatch/retry. A release-file handshake proves submit
  returns before the blocked publisher completes. Fresh `make test-bang-tag
  test-submission-transaction` passed all six bang cases and the transaction
  matrix. Removing `pcall`, removing the successful-send guard, and replacing
  asynchronous launch with a blocking call each made its corresponding test
  fail; production was restored byte-for-byte. ARCH-PURPOSE: the complete
  optional-side-effect failure family is tested, not just the happy path.

### 2026-09-28
- Operator resolved the open questions: no `!!` escape (bash mode from the draft is unwanted); thread identity comes from the `COUCH_THREAD_SCOPE` / `COUCH_THREAD_TAG` env that couch already injects.
- Design, from tracing the code:
  - Seam: both operator sends (`send_and_clear`, `ship_buffer_and_reset`) call `_G.submit_operator_text(authored, agent_text)` (`nvim/init.lua`). The wrapper parses the comment-stripped `agent_text`. It sends the text after `!`; the Pair log keeps the authored body with the `!`. Sticky `===` lines are stripped before the single-line test, so a draft with a sticky block still qualifies.
  - Publish only after the send is confirmed, so it never delays the send, and a failed send publishes nothing (the retry publishes). Run it as a detached `jobstart` of bare `couch --internal publish-description --description=<text>`, which resolves `couch` from PATH the same way the launcher's `request-continuation` call does (`cmd/internal/launcher/checkpoint_io.go`). The `--description=` form stops a tag starting with `-` from being parsed as a flag.
  - Inside couch means both `COUCH_THREAD_SCOPE` and `COUCH_THREAD_TAG` are set. Checked live: this pair session's panes carry both, plus `COUCH_STORE_DIR`. couch's CLI fills the thread from those variables itself.
  - A bare `!` returns false: nothing is sent or logged, and the draft stays as typed.
  - Growth (ARCH): this writes one existing couch description sidecar per thread and overwrites it on each tag. It creates no new durable family.
  - Integration test: a stub `couch` executable on PATH records its argv, so the real `jobstart` command is exercised with no test seam in production code.
- Implemented: `nvim/bang_tag.lua` (pure parse), wired in `_G.submit_operator_text` inside a `do` block, because `init.lua`'s main chunk is at Lua's 200-local limit (a new top-level local fails to load with "more than 200 local variables"). The real `couch` accepts the argv: with a throwaway store it got as far as the thread lookup.
- Side quests: main was red from #333. `artifactpath` coverage failed because `couchcmd/shortcut_focus.go` was never classified, and `workbench_route_test` still expected Alt+n unscoped. Fixed both in separate commits. Also recorded #338's revised spec (the switcher remembers the last view) on this branch.
- Resumed verification: operator confirmed the leading `!` was stripped and the live smoke passed. Fresh `bin/couch --list` and `bin/couch --show pair` processes read the saved published description `ok, continue to test #337` for address `e108517d46ab4575/couch-d3926507aef66e8c`. This proves persistence across reader processes; the running Couch UI was not restarted. The operator could not see the description in the UI; display remains separate work (#173/#338).
- `make test-lua test-bang-tag test-submission-transaction` passed (exit 0), including Couch/standalone draft submissions and send-transaction failure cases. Full and race suites are running; results follow.
- Targeted Couch persistence/CLI tests also passed with the race detector: `go test -race ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(PublishDescriptionUses|PublicInternalPublishDescription|ThreadStoreMetadataUpdate|ApplyThreadMetadataPreserves|DescribePrefersTheAgentsPublishedLine)' -count=1`.
- Broader verification is not green. `make test` stalled for over two minutes in `doctor/perf_test.sh` → `pair hoprtt -spawn 5 -- zellij action query-tab-names`; stopped the owned test processes (exit 130). Log: `/tmp/pair-337-full-tests.log`. `make test-race` detected concurrent appends to `FakeGit.Ops` at `cmd/internal/couchcore/git_fake.go:43`, between recovery resume and the slot-git probe in `TestRecoveryMenuReachesTerminalAfterActualHelperDeath/warm`; stopped the remaining broad run after recording the failure (exit 130). Log: `/tmp/pair-337-race-tests.log`. An isolated `go test -race ./cmd/internal/couchcmd -run '^TestRecoveryMenuReachesTerminalAfterActualHelperDeath$/warm$' -count=1` passed, so the race is intermittent. Neither the fake nor that acceptance test differs from main. No production code changed during this testing pass; issue remains open for the close/review boundary.
