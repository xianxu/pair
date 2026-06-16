---
type: pensive
date: 2026-06-15
topic: user-model as the cross-session throughline
mode: eureka
description: The real thread under the continuation work is making agents build AND persist a model of the user across sessions — #61/#105 persist it, #103 maintains it live, #90 is the docflow sibling.
references: [workshop/issues/000061-pair-continuation-improvements.md, ../ariadne/workshop/history/000105-continuation-datatype-connective-narrative-procedure-flush-to-pensive-thread-arc-user-model-open-questions-lessons.md, ../ariadne/workshop/issues/000103-agents-md-instruction-user-intention.md]
---

# Pensive: User-model as the cross-session throughline

The continuation work started as "improve the procedure," but the load-bearing
ask underneath it isn't really about continuations. It's about the **model of the
user's mental model** — and the realization is that I want that model treated as a
first-class artifact with a full lifecycle, not a thing each session re-derives
from scratch.

Seen that way, several things I filed separately are one idea wearing three hats:

- **Persist it.** A continuation should checkpoint, at park/handoff time, where my
  head is: the arc of the thread, the latent intention behind the pivots, and the
  open questions where the model is still inconsistent — and a fresh session should
  *resolve those questions with me before charging ahead*. (pair#61 → ariadne#105.)
- **Maintain it live.** Every turn should move the model — positively or
  negatively — and the agent should keep it (a) internally self-consistent and
  (b) fitting what it actually observes me do. (ariadne#103.)
- **The same shape shows up elsewhere.** docflow's suspend/resume with an
  auto-summary on resume is the same "don't make the next session re-orient by
  hand" instinct in a different domain. (ariadne#90.)

The bloat tension resolves once you see the continuation as the *connective
narrative over durable artifacts*: the detail lives in the artifacts (pensive,
issues, targets), and the continuation just explains how they connect and where
I am. You earn terseness by flushing loose understanding into pensive first — which
is exactly the move this note is dogfooding.

## Open questions

- Where is the **canonical** definition of "the user-model + its two criteria"?
  Today #103 is named canonical and the continuation section defers to it, but #103
  is still a stub — until it lands, the criteria live (lightly duplicated) in the
  datatype. Worth collapsing to a single source once #103 is written.
- Is "model of the user's mental model" really one datatype-shaped thing, or a
  *cross-cutting concern* that several datatypes/instructions each carry a slice
  of? Right now it's the latter (continuation section + #103 instruction). Is that
  the right factoring, or should there be a durable "user-model" artifact the
  continuation merely updates?
- How much of this should be automated vs. human-gated? The flush is `pensive`-only
  precisely because targets need human review — does the user-model deserve the
  same "agent drafts, human ratifies" boundary?
