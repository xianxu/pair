---
id: 000894
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'f29191773bbbe2253486057dfcce459f52f425ef' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch reports a Codex version change to the hub once, from a durable per-agent baseline

## Problem

#891 lands the stall fix: a wrapped Codex ≥ 0.164 launches with `--no-daemon` and records its CLI version in
`PAIR_AGENT_VERSION` (carried into the ready record as `agent_version`). Nothing yet surfaces a version change
to the operator, so the next upstream change (a new prompt, a removed flag) is noticed only when a slot
stalls. The 10-25 incident began when Codex 0.164 arrived unannounced.

## Spec

Couch compares the version recorded on each fresh launch with a durable per-agent baseline and reports a
change to the hub exactly once. The baseline lives at `<namespace>/agent-versions.json` (atomic write; one
Couch writes a namespace), so it survives Couch restarts. The first observation for an agent is the silent
baseline. Observation rides the existing fresh-registration poll (`c.FreshRegistration`); the message service
wraps it after notice delivery is installed. A change publishes one `status` notice (class
`agent-version-changed`) through `publishSystemNotice`; default routing sends it to the owning hub, and the
change is recorded even when the hub is unavailable so it is not repeated later. A launch with no recorded
version (old wrapper, failed probe) is silent.

Design and exact steps: `workshop/plans/000891-codex-no-daemon-and-version-notice-plan.md` Tasks 3–5 (the
Part B half, split out of #891 by TL steer on 2026-10-25).

## Done when

- A changed recorded version produces exactly one hub notice per change (observer + store unit test; a second
  launch with the same version is quiet).
- The baseline survives a Couch restart (a reopened store reports no change).
- A launch with no recorded version (old wrapper, failed probe) is silent.

## Plan

- [ ] `OSReadyReader.RecordedAgentVersion` reads the ready file alone.
- [ ] `couchcore.AgentVersionStore` (`agent-versions.json`, atomic write, in-process mutex) plus
      `AgentVersionObserver` with once-per-change semantics.
- [ ] `NoticeAgentVersionChanged` joins the route table; `messageService.watchAgentVersions` wraps
      `c.FreshRegistration` in `startMessageServiceWithStartup`.
- [ ] Tests: store/observer, route table, message-service publish.
- [ ] Atlas `couch.md`: the store and the notice rule.

## Log

### 2026-10-25

- Split from #891 by TL steer: land the stall fix first in the cheap window; this notice is the second half.
  Designed and seam-verified together with #891; consumes the `agent_version` ready-record field #891 adds.
