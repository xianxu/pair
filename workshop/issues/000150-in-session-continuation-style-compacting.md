---
id: 000150
status: open
deps: []
github_issue:
created: 2026-08-22
updated: 2026-09-17
estimate_hours:
---

# in session continuation style compacting

when user use alt+shift+n, we can prompt if they want to make a continuation, in addition to just start with a new context. If user choose (C)ontinuation, a continuation style documentation should be generated, and inserted into the current draft nvim's * position before restart of the agent with new context. this way, user can just hit alt+return when the new context is available to continue work. this is essentially customized compacting.

## Problem

Token spend is dominated by context re-reads, and they scale with **session
depth**. Measured 2026-09-17 across ~52 days of local transcripts: 19.2B tokens,
**98% cache reads**; output — the actual work product — was 0.25%. Median
cache-read per assistant turn 324K, p99 960K, 98% of it main-thread. Splitting
one deep session into several fresh-context chunks doing identical work costs a
fraction.

Today the operator has two bad options at that moment:

- **Keep going** — every subsequent turn re-reads a context that only grows.
- **Shift+Alt+N** — fresh context, but the thread is simply dropped. Everything
  not already in a durable artifact is lost, so it's only safe at a boundary the
  operator has manually tidied.

Harness auto-compaction is the usual third answer and it's the wrong shape for
this repo: the carried summary is ephemeral, model-generated, and unreviewable —
you cannot read it, correct it, or diff it — and it fires on context *pressure*,
which is the most expensive possible moment (the turns that triggered it were
already paying the full window).

The gap is a **cheap, reviewable carry across a deliberate restart**.

## Spec

### The mechanism already exists in three pieces

- **Gauge** — `pair context <tag> <agent>` prints the live context-window size
  and is readable *from inside* the session (`PAIR_TAG` / `PAIR_SCOPE_KEY` are in
  the agent's env; verified 2026-09-17 returning `121k` from a running session).
- **Actuator** — the `NewSession` restart marker with a `ContinueSlug` relaunches
  fresh and seeds the nvim draft from a continuation doc
  (`launcher/markers.go:183`, `createflow.go:705`), surfaced today as
  `pair continue <slug>`.
- **Carry** — the `continuation` datatype: a durable, version-controlled file.

So this issue is mostly **wiring an existing actuator to a new entry point**, not
building a mechanism.

### Two entry points, one mechanism

1. **Operator-initiated** (the original note, and this issue's scope): Shift+Alt+N
   grows a `(C)ontinuation` branch. Choosing it generates the continuation doc,
   inserts it at the draft's `*` position, then restarts the agent fresh — so
   Alt+Return in the new context resumes work. Declining keeps today's plain
   fresh-restart behavior unchanged.
2. **Agent-initiated** (enabled, not required, by the same wiring): because the
   gauge is readable from inside, the agent can propose the checkpoint itself.
   AGENTS.md §14 already instructs this; what it lacked was an actuator the agent
   could name.

### Why not "expose compaction as a tool to the agent"

Considered and rejected on mechanism, not taste. A tool result only *appends* to
the transcript — the harness owns the context window exclusively, so no external
tool (MCP server or shell verb) can compact it. **Process-restart-with-carry is
the only externally implementable equivalent**, and it is strictly better here:
the carry is a file you can read, fix, and commit rather than a summary you
cannot inspect.

This also does **not** revive the confused-deputy surface that killed the
`sdlc`→harness control-channel design (declined 2026-09-17, ariadne's deferred
`ARCH-AUTHORITY`). That design let any process reaching stdout drive the harness.
Here the trust direction is prose → model → tool: the model is the decision
point, which is the ordinary agent path and grants no new authority. Residual
risk is prompt injection, whose blast radius is "the agent checkpoints itself and
restarts" — loud and recoverable.

### Trigger placement (scope boundary)

A context-pressure threshold is a *late* trigger: 60% of a 1M window is 600K per
turn already being paid. The cheap trigger is a **clean boundary**, where durable
state (issue, plan, atlas, git) already carries everything the next chunk needs —
which is why the operator's current habit (fresh session at `sdlc close`) beats
any threshold. Making the boundary prompt reflexive is **ariadne-side** work
(`close` / `milestone-close` printing the gauge reading), not pair's; nearest
neighbor is ariadne#90. This issue owns the pair-side mechanism and the operator
entry point only.

## Done when

- [ ] Shift+Alt+N offers a `(C)ontinuation` choice alongside the plain fresh
      restart; declining preserves today's behavior byte-for-byte.
- [ ] Choosing it writes a durable continuation doc, seeds the draft at `*`, and
      restarts with fresh context — Alt+Return resumes work with no retyping.
- [ ] A failed or empty continuation write does **not** restart: the thread is
      never dropped on the strength of a carry that didn't land.
- [ ] `pair context` remains readable from inside the session (the gauge half of
      the loop) — asserted, so a future refactor can't silently remove it.
- [ ] Regression coverage at the pure seam: a fake-runtime test over the
      choice → write → seed → restart sequence, including the failed-write branch.

## Plan

- [ ] Decide where the continuation doc is generated: the agent writes it on
      request (it has the session context) vs. pair synthesizes it. The former is
      the only one that can produce a real summary — settle the handshake for
      "pair asks, agent writes, pair waits for the file, then restarts."
- [ ] Wire the `(C)ontinuation` branch into the Shift+Alt+N path, reusing the
      existing `ContinueSlug` marker rather than adding a parallel route.
- [ ] Draft seeding at the `*` position + the no-restart-on-failed-write guard.
- [ ] Tests per Done-when.
- [ ] Atlas: document the gauge/actuator/carry loop wherever the restart paths
      are described.

## Log

### 2026-09-17

- Spec written from a brain advisor session. The original 2026-08-22 note is
  preserved above the `## Problem` heading; everything below is new.
- Key finding that changed the shape: **the gauge is already agent-readable** —
  `pair context $PAIR_TAG claude` returned `121k` from inside a live session. The
  loop (gauge → judgment → actuator → durable carry) therefore closes entirely at
  the pair layer, with no harness cooperation required. The question that prompted
  this ("could we expose compaction to the agent as a tool?") is answered in Spec:
  not as harness compaction, which no external tool can reach.
- Cost measurement cited in Problem comes from the 2026-09-17 brain transcript
  analysis (19.2B tokens / 98% cache reads over ~52 days).
