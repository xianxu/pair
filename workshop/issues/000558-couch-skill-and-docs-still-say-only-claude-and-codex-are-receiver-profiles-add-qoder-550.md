---
id: 000558
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'b6c118073c88d7be172c42d47505511412c4c973' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:23:08-07:00

---

# couch skill and docs still say only Claude and Codex are receiver profiles; add Qoder (#550)

## Problem

pair#550 (landed 10-14) made Qoder a supported Couch peer (receiver admission and composer parsing; every AgentInventory entry declares a receiver policy; Agy, Muse and Grok are explicitly disabled). The atlas was updated (`atlas/couch.md:493` names Claude Code, Codex CLI and Qoder CLI), but the embedded couch skill still says "Only agents with a registered Pair receiver profile (Claude Code, Codex CLI)" (`cmd/internal/couchcmd/skills/couch/SKILL.md:502`), so every agent reading `couch --skill` is told Qoder can't receive messages. Found by pair:3.

## Spec

Update the skill paragraph to name Qoder, and the explicitly disabled agents as disabled; prefer generating that list from the AgentInventory receiver policies over hard-coding. Grep the skill, README, atlas and help text for other two-agent lists ("Claude Code, Codex", "claude or codex") and fix the class.

## Done when

- `couch --skill` names Claude Code, Codex CLI and Qoder CLI as receiver profiles and the disabled agents as disabled; the grep for stale two-agent lists is clean (recorded in the Log).

## Plan

- [ ]

## Log

### 2026-10-14

## Revisions

- 2026-10-14 16:05 — prep by pair:4 (Qoder trial, paused before any claim; preserved here so a restart doesn't lose it): sweep found only `cmd/internal/couchcmd/skills/couch/SKILL.md:502` stale (README:649 and atlas already name Qoder; couchcmd help clean; the peer_live_test "claude or codex" flag text describes harness capability, so leave it and log why). Design: export a receiver-policy accessor from wrapcmd and pin the SKILL.md receiver sentence to the AgentInventory policies with a couchcmd test. The pair:7 claim was released (no owner); claim it fresh.
