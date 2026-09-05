---
id: 000083
status: punt
deps: []
created: 2026-06-29
updated: 2026-09-01
---

# pair-workbench: prescribed way of when to use which agent
pair started as a thin wrapper around coding agent, and have become more sophisticated wrapper around harnesses, and strive to be agent neutral. 

in ariadne#129, we talked about customization of different agents for different workload, for example, to use one agent for main, either another or the same when running review. same I guess can be extended to other fresh context subagent use cases. in /xx-fix, we calls for a different agent to do fact check, for example.

pair-workbench would be a pre-configured way to start up main agent based on the above configuration, plus maybe various initial start up parameters. The goal here is to really make things very simple. 

I guess in terms of priority, ariadne#129 is more important, this ticket seems to be more cosmetic. 

## Done when

- Reconsider only if Pair and Couch's launch-profile/startup surfaces leave a
  concrete recurring workflow that requires a separate prescribed workbench.

## Spec


## Plan

- [ ] If reopened, identify the remaining startup gap and prove it does not
  duplicate Pair or Couch configuration.

## Log

### 2026-06-29

### 2026-09-01 — punted as obsolete

The prescribed workbench/startup layer is no longer compelling. Pair and Couch
now provide several more direct ways to select and remember agent launch
profiles, and the current Couch work is simplifying startup further. Defer this
indefinitely rather than add another overlapping configuration surface.
