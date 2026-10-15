---
id: 000558
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
card_mirror: '132954123dc21db83f43eed839243e1c7f4ef548' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:23:08-07:00

claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# couch skill and docs still say only Claude and Codex are receiver profiles; add Qoder (#550)

## Problem

pair#550 (landed 10-14) made Qoder a supported Couch peer (receiver admission and composer parsing; every AgentInventory entry declares a receiver policy; Agy, Muse and Grok are explicitly disabled). The atlas was updated (`atlas/couch.md:493` names Claude Code, Codex CLI and Qoder CLI), but the embedded couch skill still says "Only agents with a registered Pair receiver profile (Claude Code, Codex CLI)" (`cmd/internal/couchcmd/skills/couch/SKILL.md:502`), so every agent reading `couch --skill` is told Qoder can't receive messages. Found by pair:3.

## Spec

Update the skill paragraph to name Qoder, and the explicitly disabled agents as disabled; prefer generating that list from the AgentInventory receiver policies over hard-coding. Grep the skill, README, atlas and help text for other two-agent lists ("Claude Code, Codex", "claude or codex") and fix the class.

## Done when

- `couch --skill` names Claude Code, Codex CLI and Qoder CLI as receiver profiles and the disabled agents as disabled; the grep for stale two-agent lists is clean (recorded in the Log).

## Plan

- [ ] `sdlc claim --issue 558`, then `sdlc start-plan` (fresh branch from main)
- [ ] wrapcmd: export `PeerReceiverAgents() map[string]bool` (a copy of the policy map in `peer_runtime.go`)
- [ ] couchcmd test: derive the enabled/disabled display names from that accessor over `launcher.AgentInventory()`; pin SKILL.md's receiver sentence to name exactly the enabled set and mark the disabled set
- [ ] SKILL.md:502: name Claude Code, Codex CLI and Qoder CLI as receiver profiles; Agy, Muse and Grok as explicitly disabled
- [ ] re-grep skill, README, atlas, help text for two-agent receiver lists; record results in ## Log
- [ ] `sdlc local-check run --name suite`
- [ ] `sdlc close --issue 558 --verified <evidence>`, then `sdlc merge --yes`

## Log

### 2026-10-14

- pair:4 prep sweep over skill/README/atlas/help (`Claude Code, Codex`, `claude or codex` and variants): only `SKILL.md:502` is stale. README:649-653 and `atlas/couch.md:493` already name Qoder (#550's doc commit); `atlas/index.md:3` lists the full launcher inventory, not a receiver claim; couchcmd help strings carry no agent lists.
- `peer_live_test.go:19,73` "isolated live peer receiver: claude or codex" is accurate, not stale: `TestPeerLiveConformance` refuses any other harness (`peer_live_test.go:75`); Qoder's coverage is recorded-screen fixtures. Leave it.
- Design: runtime substitution of the sentence into `couch --skill` output rejected — `messages_test.go:224` pins verbatim emission of the embedded file and operators save that file directly, so print-time templating diverges file from output. Chosen: a couchcmd test generates the expected sentence from `wrapcmd.PeerReceiverAgents()` × `launcher.AgentInventory()` (display names: claude→Claude Code, codex→Codex CLI, qoder→Qoder CLI, agy→Agy, muse→Muse, grok→Grok) and greps SKILL.md, so inventory drift fails a test instead of rotting the doc. No import cycle: couchcmd and wrapcmd are mutually independent.

## Revisions

- 2026-10-14 16:05 — prep by pair:4 (Qoder trial, paused before any claim; preserved here so a restart doesn't lose it): sweep found only `cmd/internal/couchcmd/skills/couch/SKILL.md:502` stale (README:649 and atlas already name Qoder; couchcmd help clean; the peer_live_test "claude or codex" flag text describes harness capability, so leave it and log why). Design: export a receiver-policy accessor from wrapcmd and pin the SKILL.md receiver sentence to the AgentInventory policies with a couchcmd test. The pair:7 claim was released (no owner); claim it fresh.
