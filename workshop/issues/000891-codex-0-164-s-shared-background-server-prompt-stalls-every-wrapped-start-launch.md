---
id: 000891
status: working
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'cb11592656da35aba73a63fbbc30e4e8a75f0c5f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T09:36:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
---

# Codex 0.164's shared background server prompt stalls every wrapped start; launch so it can't appear

## Problem

## Spec

**Codex 0.164 stops every Couch-launched start at a "Background server has incompatible feature settings" prompt
(10-25).** Codex 0.164 runs a shared background server for every session on the machine. Couch launches Codex with
`shell_snapshot` disabled; once a server is running with other settings (here, a Codex started outside Couch right
after the update), each Couch launch stops at a three-way prompt: run without daemon this time / restart the shared
server with these settings (which can interrupt other sessions) / cancel. Couch can't answer prompts (`couch --keys`
refuses), so the slot sits stalled until the operator presses a key; pair:11 did this twice on 10-25. Every Codex
slot will meet it on its next start.

`pair wrap` launches Codex so the prompt can't appear: either opt out of the shared server for wrapped sessions (the
equivalent of "run without daemon", via a supported flag or config if 0.164 has one), or launch with settings that
match the shared server's. Never restart the shared server from a slot. Also: Couch records the Codex version per
launch and surfaces a version change to the hub, so the next upstream change is noticed before slots stall.

## Done when

- A wrapped Codex 0.164 start with a mismatched shared server reaches its composer with no prompt (native test, or a
  documented manual check if the server can't be faked), and the old-version path is unchanged.
- A Codex version change since the last launch is reported to the hub once.

## Plan

Durable plan: `workshop/plans/000891-codex-no-daemon-and-version-notice-plan.md`.

- [ ] A1: ready record carries `agent_version` (validated, optional); the wrapper publishes it from
      `PAIR_AGENT_VERSION`.
- [ ] A2: launcher probes `codex --version` once per launch (3s bound, failure silent), exports the version,
      and prepends `--no-daemon` at ≥ 0.164.0 unless the caller already passed it. All Codex launches; the
      shared server is never contacted or restarted.
- [ ] A3: launcher/readiness/wrap tests pin 0.164.0 (flag + env), 0.163.0 and failed probe (argv unchanged,
      version still recorded when known), and an explicit `--no-daemon` (no duplicate).
- [ ] B1: `OSReadyReader.RecordedAgentVersion` reads the version from the ready file alone.
- [ ] B2: durable `agent-versions.json` store + observer: first observation is the silent baseline, a change
      reports the previous version exactly once (in-process mutex; one Couch writes a namespace).
- [ ] B3: `NoticeAgentVersionChanged` routes to the hub by default; the message service wraps
      `c.FreshRegistration` and publishes one `status` notice per change.
- [ ] V1: focused + package tests green; `make test-affected-inventory` audited.
- [ ] V2: documented manual check: wrapped Codex 0.164 with this machine's mismatched shared server reaches
      its composer with no prompt; the same argv without `--no-daemon` shows the prompt.
- [ ] V3: atlas updated (`codex-resume.md`, `couch.md`); suite recorded on the pushed head; PR merged;
      outcome reported to ops:0.

## Log

### 2026-10-25

- Codex 0.164 evidence (read-only probes, no session started): `codex --version` → `codex-cli 0.164.0`;
  `codex --help` lists `--no-daemon` ("Run without the shared background server, even if it is already
  running"); `codex --no-daemon resume --help` exits 0, so the flag is top-level and precedes the subcommand.
  Design: private app server per wrapped launch, gated on a parsed version so older CLIs are untouched.
- Seams: the launcher composes agent argv in `createflow.go` before `EncodeAgentCommand`; the wrapper writes
  `PAIR_AGENT_VERSION` into the ready record; Couch's `c.FreshRegistration` polls every fresh launch
  (`launch_existing.go:awaitFreshRegistration`). Notices with an unspecial-cased class route to the owning
  hub by default (`RouteNotice`), and `publishSystemNotice` is the established publish seam; the message
  service is the first place console + delivery + reader all exist. `readiness.Decode` tolerates unknown
  members, so the new field is compatible across an upgrade in both directions.
- Design exception, deliberate: the first-ever `StartupFresh` launch runs before the message service exists;
  it is silent anyway (empty store = baseline).
- Size: ~230 added non-test lines across launcher/readiness/wrapcmd/couchcore/couchcmd — outside the
  quick-flow shell (100 lines), so this is a full-flow close. Estimate proposed in the durable plan.
