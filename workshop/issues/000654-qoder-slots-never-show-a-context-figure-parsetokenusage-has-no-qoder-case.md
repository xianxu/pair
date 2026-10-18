---
id: 000654
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7a5438980657539ff0e276d48ec59ce05ebf1640' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T00:48:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Qoder slots never show a context figure: ParseTokenUsage has no qoder case

## Problem

## Spec

Found during #653 on 2026-10-17, low priority. Qoder slots never show a context figure. Their title projection admits fine (a live probe on 1-pair-6 and 1-pair-60 showed admitted, no error), but `HasUsage` is always false.

The cause: `sessioninventory.ParseTokenUsage` (cmd/internal/sessioninventory/usage.go) handles `AgentCodex`, `AgentClaude` and `AgentGrok` only. It has no `AgentQoder` case, so no Qoder transcript record ever yields a `TokenUsage`. This is a gap, not a regression: Qoder had no count before #617 either, because `contextcmd` used the same parser.

Qoder transcripts are parsed like Claude's elsewhere (`event.go` groups `AgentClaude, AgentQoder`). So the fix is likely to accept Qoder in the Claude usage branch, or add its own branch if Qoder's usage record differs. Check a real Qoder transcript first.

## Done when

- `ParseTokenUsage(AgentQoder, line)` returns the context usage from a real Qoder usage record. It has a fixture test built from a captured Qoder line, plus a negative case.
- A Qoder slot's pane title shows its count. Checked live.

### Findings (2026-10-18, from pair:4 and pair:8 transcripts on disk)

- Qoder's assistant records carry a Claude-shaped `usage` block, but every token field is 0. That held for all 2,746 usage records across both transcripts (slot4: 2,564; slot8: 182). So reading Qoder as Claude would show "0" forever.
- Occupancy is in `usage.context_usage_ratio`, a float present on every usage record. It climbs within a session (slot8: 0.145 to 0.400; slot4: 0.142 to 0.422).
- The window behind the ratio is not reliably known. `runtime-config.contextWindow` is null in 130 of the 219 records surveyed (all of slot8's), and 200000 or 500000 in the rest.

### Design (quick flow)

- **The figure is a percentage for Qoder.** A count would need ratio times a window that is mostly unknown, so it would be a guess. The ratio is Qoder's own figure. The pane title shows e.g. `40%` where Claude and Codex show `80k`.
- `TokenUsage` gains `ContextRatio *float64`. Qoder sets it and leaves `InputTokens` 0.
- One renderer, `TokenUsage.Figure()`, returns `NN%` when a ratio is set and `ctxmeter.Humanize(InputTokens)` otherwise. It replaces the three `Humanize(usage.InputTokens)` sites: `couchcmd/titles.go`, `titlepoller/projection.go` and `contextcmd` (ARCH-DRY).
- `ParseTokenUsage(AgentQoder, line)` accepts a record only when all of these hold: `type` is `assistant`, it is not a sidechain, it has a usage block, and `context_usage_ratio` is present and within [0, 1]. Qoder's zero token fields are never used.

## Plan

- [ ] Fixture test from captured Qoder lines, with message content trimmed and `usage` byte for byte, so `ParseTokenUsage(AgentQoder, …)` returns the ratio. Negative cases: a usage block without the ratio; a ratio outside [0, 1]; a sidechain record; a non-assistant record. `TokenUsageFromJSONL` over a slot-shaped fixture returns the last ratio.
- [ ] Add `TokenUsage.ContextRatio` and the Qoder branch.
- [ ] `Figure()`, with its test (percent rounding, count fallback), replacing the three call sites. Run the existing title, titlepoller and contextcmd tests.
- [ ] Title projection test: a Qoder transcript yields `HasUsage` and a `%` figure.
- [ ] Live: after the next Couch restart, a Qoder slot's pane title shows its percentage. That restart is already scheduled for the #653/#657 checks, and Qoder is out of credits, so an idle Qoder slot shows its last ratio.

## Log

### 2026-10-17
