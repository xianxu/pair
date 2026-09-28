---
id: '000080'
status: wontfix
started: 2026-07-12T15:12:39-07:00
created: 2026-06-26
updated: 2026-07-12
---

# pair retro: the ability to go through tty log and identify workflow issues

## Problem

At the end of a session, Pair should be able to go through an agent transcript
and identify things that did not work well for that agent. Prefer the rendered
TTY log as the source because it is more portable across agents than each
agent's native transcript format.

The retro should find concrete workflow/tooling problems: base-layer bugs, agent
transcript format drift, tool-call errors, SDLC process friction,
permission/environment mismatches, and cases where a rigorous process becomes
form without essence.

The useful output is not a generic session summary. The useful output is an
evidence-backed list of process/tooling frictions that can become follow-up
issues, instruction changes, or binary improvements.

On the other hand, all it takes seems to just for operator to ask for a retro, this ticket might seem overly complicated. just need to keep things we want to retro on in a skill I think. our retro is around development process (sdlc and surroundings).
