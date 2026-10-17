---
id: 000654
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '48e506345a93b2e4bb514264ba3f77fa985dc486' # card fields mirrored from issue-cards; edit via sdlc
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

## Plan

- [ ]

## Log

### 2026-10-17
