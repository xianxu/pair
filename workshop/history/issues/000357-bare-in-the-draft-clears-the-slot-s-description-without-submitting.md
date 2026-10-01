---
id: 000357
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: '0b30199d11de7d6ad8a514b928cceb6be48900a5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T13:04:06-07:00
flow: {kind: quick, provenance: inferred, spec: "6e173bb9", done: "73f78976"}
---

# Bare ! in the draft clears the slot's description without submitting

## Problem

A single-line draft starting with `!` tags the hosting couch thread (#337): the
text after `!` goes to the agent and also becomes the thread's description. There
is no way to take a description back from the draft pane. Today a bare `!` (only
whitespace after it) is a deliberate no-op: `bang_tag.parse` returns an empty
`agent_text` with no description, and `submit_operator_text` returns false
(`nvim/bang_tag.lua`, `nvim/init.lua` `submit_operator_text`).

## Spec

A bare `!` submission clears the current slot's description and sends nothing
to the agent.

- **Trigger:** exactly what `bang_tag.parse` treats as a bang line today: a
  single line whose first non-blank character is `!`, followed only by
  whitespace (after `===` comment stripping).
- **Effect:** the hosting couch thread's published description is cleared. No
  agent submission, no Alt+Enter, and nothing reaches the agent pane.
- **Draft and log:** decide whether the draft clears like a normal send, and
  whether the pair log records the action (it is operator intent, but not a
  prompt). Recommend: clear the draft, and log nothing to the agent history.
- **Outside couch** (no `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`): there is no
  description to clear. It stays a no-op, but tells the operator so rather than
  silently doing nothing.
- **Failure is visible:** if the clear fails, notify. Don't report success
  (lessons: a send result must reflect what happened, #208).

Questions to settle at design:

- Can `couch --internal publish-description` take an empty value? `description`
  is a `Required` flag (`couchcore/ops.go`), so `--description=` may be refused.
  If so, add an explicit clear (a flag or a separate op) rather than overloading
  empty text.
- What does "cleared" mean? `PublishDescription` (`couchcore/couch.go`)
  writes the agent-facing one-liner, which `Describe` prefers over an
  operator-typed one. Clearing it probably should fall back to the operator's
  description if one exists, not blank the row. Confirm with the operator.

Related: #358 (`!!` publishes the previous prompt as the description). It shares
the "draft action that doesn't submit" routing, so design them together.

## Done when

- A bare `!` (with any trailing whitespace) in a couch-hosted draft clears the
  thread's published description, and the couch row shows the fallback
  description (or none) on its next refresh
- Nothing reaches the agent pane, pinned by a stateful fake that fails the test
  on any `write-chars`/submit
- Outside couch, a bare `!` sends nothing and notifies that there is no thread
  to clear
- `! text` behavior (#337) is unchanged, with its tests still green

## Plan

- [x] Settle the two design questions. Both are already answered in couch:
      `--description=` binds an empty value (`description` is not
      `ValueRequired`, `couchcmd/run.go` `bindArgs`), and an empty
      `PublishedSummary` clears only that field
      (`threadmetadata_model.go`), so `ThreadSummary.DisplaySummary` falls
      back to the operator's `Description`, as recommended.
- [x] Couch side: no new op; a CLI test pins the exact argv the draft sends
- [x] Draft side: `bang_tag.parse` distinguishes "clear" from "not a bang line";
      `submit_operator_text` routes it to the clear without calling the send
- [x] Tests: `bang_tag` unit + `bang_tag_integration_test` (no agent traffic),
      plus couch store/op test for the clear

## Log

### 2026-09-30
- 2026-09-30: closed — Operator live smoke passed. Lua bang_tag unit and all ten draft integration cases passed. Expanded TestPublishDescriptionEmptyFlagFallsBackToOperatorDescription passes with both present and absent fallback, asserting persisted and displayed values; git diff --check passes. BR-2 comment wrapped; BR-3 coverage completed.; review verdict: SHIP
- 2026-09-30: closed — bang-tag-nvim-test: bare ! publishes --description= via stub couch with zero zellij executor calls and no log append; standalone and failing-couch cases notify and keep the draft; bang_tag_test pins bare ! as a clear; new couchcmd test runs the exact --description= argv and shows the row falling back to the operator description; make test green; go failures (artifactpath classification, TestBareCouchInstalledCommand, TestCouchReferencesLocalArchiveLocatorRoundTrip) fail identically on origin/main; review verdict: SHIP

- Filed at the operator's request. Current bare-`!` no-op behavior verified in
  `nvim/bang_tag.lua` (#337).
- Built on #358's no-send path: `bang_tag.parse` returns `{ clear = true }`
  for a bare `!`, and `describe_couch_thread` publishes `''` synchronously,
  so the draft clears only on success. That retires the old
  `agent_text == ''` no-op branch. The branch is based on the unmerged #358
  branch; rebase onto main once #358 lands.
- `previous_description` treats any line that sent nothing (bare `!`, `!!`)
  as having no description.

- Close review (SHIP, 2 minor advisories), both fixed in one follow-up commit:
  unit cases pin `previous_description` returning nil for logged `!!` and
  `!! text` lines, and a missing comma in `atlas/couch.md`.

- 2026-09-30: operator smoke test passed. Reconnected the original close ancestry after range-diff confirmed identical rebased patches. Re-close BR-3 requested coverage of both promised clear outcomes: parameterized the CLI test over present/absent operator description and asserted stored fields plus displayed summary. Both cases pass. Wrapped the remaining BR-2 header comment. Fresh Lua unit and ten draft integration cases also passed before these test/comment-only changes.
