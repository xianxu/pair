---
id: 000410
status: working
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: '194960644127153e2290f72d7ea6957c53f4ed4f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T22:22:25-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Grok harness integration for pair and couch

## Problem

Pair and Couch host `claude`, `codex`, `agy`, `muse` and `qoder`, but not Grok.
To bring Grok in, it has to join the agent registry and pass every integration
surface that the existing harnesses already pass, so that it runs under pair as
smoothly as they do and Couch can launch, park and resume it.

## Spec

Bring up a `grok` harness by following
[How to Bring Up a New Harness CLI](../../atlas/how-to-bring-up-a-new-harness-cli.md)
(§0 registry, §2 ten-item checklist), to full parity with the existing harnesses.
Couch peer delivery is out of scope (follow-up issue).

### The harness (measured 2026-10-07)

- **CLI:** `~/.local/bin/grok` is xAI's "Grok Build TUI", 1.0.46 stable, a
  full-screen TUI. It has `--no-alt-screen` and `--minimal` modes.
- **Keys:** Enter sends; Shift+Enter or Alt+Enter inserts a newline. This is the
  opposite of pair's convention, so Grok needs the Return remap.
- **Resume:** the TUI takes `-r/--resume <id-or-title>` and `-c/--continue`.
  `-s/--session-id` works headless only, so pair discovers the session ID after
  launch rather than assigning it.
- **Sessions:** stored at `~/.grok/sessions/<url-encoded-cwd>/<uuidv7>/`.
  `summary.json` holds `info.id`, `info.cwd`, `created_at` and `updated_at`;
  `updates.jsonl` is an ACP `session/update` stream with `params.sessionId` on
  every line. A stub session can have `summary.json` but no `updates.jsonl` (seen
  after `grok login`). The per-cwd `prompt_history.jsonl` lists
  `{session_id, prompt}`.
- **Permissions:** set in `~/.grok/config.toml` or with `--allow` /
  `--always-approve`. Headless mode is `grok -p <prompt>`.

### Bring-up loop

Grok is developed in pair first, not in Couch. Plain `pair-dev grok` already runs
an unregistered agent (only the fresh-launch, continuation and Couch-profile paths
check `IsSupportedAgent`), and each aspect that misbehaves there is a checklist
item. The work is done from an existing Claude slot. Couch enters at M3: once
`grok` is registered and has a scanner, Couch's start and switch-agent menus pick
it up (guide §0), and switching a slot to Grok is the acceptance test.

### Design: follow the per-harness pattern

Grok entries go next to the existing harnesses' in the existing tables. There is
no new mechanism, with one exception: Codex's forced inline mode
(`codexAltScreenArgs`, `cmd/internal/launcher/agentargs.go`) becomes a shared
per-harness field ("inline flag + opt-out env") that both Codex and Grok use, so
the alt-screen handling is not copied (ARCH-DRY).

- **M1, usable in pair:** registry entries (launcher `supportedAgents`,
  sessioninventory `Agent` enum and `supportedAgents`), the inline flag, Return
  remap (pair Enter becomes Grok newline, pair Alt+Enter becomes Grok send), an
  overlay detector for Grok's permission and selection pickers if plain Enter
  doesn't confirm them, and captured TTY fixtures under
  `cmd/internal/wrapcmd/testdata/tty/grok/<version>/`.
- **M2, Couch-resumable:** a versioned scanner over `summary.json` +
  `updates.jsonl` with a conformance fixture. A stub session without
  `updates.jsonl` reads as "no conversation", not an error. Also: the resume
  token (`--resume <id>`, with the `-r` and inline/glued spellings, in
  `resumeform.Forms`), `pair-slug` through `grok -p`, and the parity test passing
  with no known-gap entry.
- **M3, polish + live check:** the permission allowlist, the user-prompt glyph
  for Alt+b in `nvim/scrollback.lua`, the atlas guide listing Grok, and a live
  operator smoke test under Couch.

### Alternatives considered

- **ACP-generic scanner** (because `updates.jsonl` is ACP): no other harness in
  pair speaks ACP, so this is speculative generality. Revisit when a second ACP
  harness arrives.
- **Drive `grok agent stdio` headless:** abandons pair's model (a real agent TUI
  in a pane) and every existing surface. Rejected.
- **Provisional "attach any agent" mode in Couch:** it could only launch. Park,
  cold-resume and the live/parked listing all go through the scanner, so it would
  be a second, weaker supervision mode to maintain, for a capability that
  `pair-dev` already gives. Deferred; revisit if the next harness bring-up hits
  the same friction.

### Durable state (ARCH-FUNERAL)

Pair creates nothing durable of its own for Grok. Grok owns and retains its
session directories. Pair's existing adaptation log and inventory records gain
`grok` rows under their current retention, with no new family. Captured fixtures
are versioned test data, replaced when the harness version moves.

## Done when

- `grok` is in the launcher `supportedAgents`, the sessioninventory `Agent` enum
  and its `supportedAgents`, and `TestAgentInventoryParityWithSessionTables`
  passes with no known-gap entry.
- Codex and Grok share one per-harness inline-mode field; Codex behavior,
  including the `PAIR_CODEX_ALT_SCREEN` opt-out, is unchanged.
- Captured fixtures under `cmd/internal/wrapcmd/testdata/tty/grok/` pin the
  Return remap (and the overlay detector, if Grok needs one).
- The Grok scanner has a conformance fixture covering a stub session; the resume
  token round-trips `--resume`, `-r` and the inline/glued spellings; `pair-slug`
  generates a slug for a Grok session.
- Operator live smoke test: `pair-dev grok` works with the Return remap and
  scrollback; Couch's start and switch-agent menus list Grok; a Couch-hosted Grok
  thread launches, parks and cold-resumes.
- A follow-up issue for a Grok Couch peer-delivery receiver profile is filed.
- The atlas guide lists Grok among the supported harnesses.

## Plan

- [ ]

## Log

### 2026-10-07

- Brainstorm: scope is full parity in three milestones; peer delivery goes to a follow-up. Bring-up loop is `pair-dev grok` from a Claude slot; Couch at M3. Provisional Couch attach deferred. Grok login confirmed; session layout measured live (stub session without updates.jsonl observed).
