---
id: 000414
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '817b3074837475caba97dc10bd0e222365c75f92' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch switch-agent to grok: attach refused, orientation lost

## Problem

The operator (2026-10-12, Claude quota at 80%) wants slots moved to Codex as they free up, and wants the TL agent to do it. Today only the operator can switch a slot's coding agent (the thread's actions, **switch coding agent**); Couch's agent-facing verbs (`--prepare`, `--relaunch`) keep the slot's agent. And the switch's handoff feels robotic to the operator: it launches a fresh conversation and asks the new agent to read the outgoing context and summarize its orientation.

## Spec

1. **A verb for agents:** `couch --switch-agent repo:N --agent NAME [--confirm] [--json]`, the same operation as the switcher's form (same guards: refuses a busy or unknown slot; `--confirm` required). Cost-tagged in help (`[heavy]`).
2. **The handoff from durable state, not a transcript summary:** an idle slot on its resting branch gets a fresh agent and no orientation at all. A slot on an issue branch gets one short prompt naming the issue, the branch and the last commit, and telling it to read the issue's Spec, Plan, Log and Revisions, then report to the hub and wait. No "summarize your orientation" step. The operator judges the wording on a live switch.
3. The switcher's form uses the same handoff.

## Done when

- The TL switches an idle slot to Codex with one couch call, and a mid-issue slot resumes from its issue on Codex; the operator approves the handoff's feel on a live switch.
- Tests cover the guards and both handoff shapes.

## Plan

- [ ] Reproduce a Claude → Grok switch-agent on a scratch slot with Couch tracing; capture which `StartResult` reaches the console attach.
- [ ] Find why the handle is nil or mismatched; fix the cause with a couchcore/couchtty regression test.
- [ ] Verify orientation delivery, then Couch park/cold-resume and the `doctor/doctor.sh` tally on a live grok thread.

## Log

### 2026-10-08
