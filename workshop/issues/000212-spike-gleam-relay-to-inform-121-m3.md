---
id: 000212
status: open
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
---

# spike: Gleam relay to inform 121 M3

## Problem

`#121 M3` is *"implement the Oracle-friendly relay as a dumb authenticated
mailbox, local daemon long-poll connection"*. That component is small, stateless,
network-facing, connection-concurrency-heavy, and holds **no domain logic** — the
whole security posture of `#121` is that the local machine stays authoritative
and the relay widens no trust boundary.

That shape is an unusually good candidate for Gleam/BEAM, and an unusually good
first Gleam project: the thin-corpus penalty scales with domain complexity, and a
dumb mailbox has none. High concurrency value, low domain risk, small blast
radius.

But the useful question is not "would it work" — it is **what is it like to
operate**, which cannot be answered by reading. Hence a spike whose deliverable
is a decision, not a component.

**This issue does not implement `#121 M3` and does not commit it to Gleam.** It
is time-boxed, throwaway-by-default, and exists to make that later choice
evidence-based. Reasoning captured in
`brain/workshop/pensive/2026-09-07-01-pensive-agentic-web-language.md`.

## Spec

**Build the smallest honest version of M3's relay in Gleam, deploy it, operate
it, and record what was learned.**

Scope — deliberately the dumb-mailbox core and nothing else:

- Long-poll connection from a fake "local daemon" side. **Long-poll, not
  WebSocket** — `#121 M3` already chose it, and the spike should test the
  decision that was made rather than substitute a different one.
- HTTP request/response from a fake "browser" side.
- **No retention.** `#121 M5` calls for relay non-retention checks; the spike
  should honour the property even though it is not testing it.
- Auth **stubbed** — `#121 M2` owns it and lands first. A shared secret is
  enough to keep the shape honest.

Explicitly out: real auth, session identity, artifact browsing, terminal
streaming, anything from `#119`/`#120`/`#122`.

### What the spike must answer

The point is the measurements, not the code. Four questions, each with a
recorded answer:

**1. Agentic development friction — the corpus-vs-shell question.** Gleam is the
extreme case: minimum training corpus, maximum compiler feedback. Record
**compile-error rounds per feature** and what class of error the compiler caught
versus what a review caught. This is the one number that generalises past this
spike.

**2. Operational observability — the agentic-SRE question.** BEAM's claim is
*introspection without instrumentation*: `process_info(Pid, message_queue_len)`
gives backpressure with no metrics plumbing, `sys:get_state` reads a live actor's
state whose author never added a debug endpoint, and supervisor restart intensity
(`max_restarts` in `max_seconds`) is a health signal built into the runtime.

Test whether that reaches a *Gleam* actor in practice, and whether the answers
are legible from outside the process. Note the connection to `#204`: those are
**counts, not timings** — mailbox depth, restart counts — which is exactly the
flake-free tier-1 shape, and OTP supervisors already implement a counted health
model. If it works, the relay gets an operational surface for free that a Go
implementation would have to build.

**3. Deployment reality.** `gleam export erlang-shipment` plus a container, end
to end to a real host. Record what it actually takes, and whether the project's
own view of shipment as a stopgap (a proper `otp-release` command is planned)
bites at this scale.

**4. Does OTP supervision fit the relay's failure model?** A dumb mailbox has an
easy one — a dropped connection should not take down other connections. Whether
that falls out of a supervision tree or needs designing is worth knowing before
committing M3.

## Done when

- A Gleam relay carrying long-poll and request traffic between two fake ends is
  deployed to a real host and exercised by hand.
- All four questions above have recorded answers in `## Log`, with the
  observability ones stated as what was *actually* readable from a live process,
  not what the docs claim.
- A recommendation for `#121 M3` — Gleam or Go — with the reason, written so the
  decision does not have to be re-derived.
- Anything worth keeping is noted; the rest is explicitly disposable. A spike
  that quietly becomes production is the failure mode to avoid.

## Plan

- [ ] Minimal relay: long-poll in, HTTP out, no retention, stubbed auth.
- [ ] Deploy via `erlang-shipment` + container to a real host; record friction.
- [ ] Probe observability from outside: mailbox depth, `sys:get_state` against a
      Gleam actor, supervisor restart counters. Record what is legible.
- [ ] Kill connections and processes deliberately; confirm isolation.
- [ ] Write the four answers and the M3 recommendation in `## Log`.

## Log

### 2026-09-07

Opened from a brainstorm on founding the cloud side of remote Pair on the BEAM.
Filed as a spike rather than as `#121 M3` itself because M3 is already specified
and claimed by `#121`; duplicating it would fragment that issue. The deliverable
here is a decision plus operational feel.

Two things found while scoping that should reach whoever picks up `#120`:

**The remote client is a third consumer of `#209`.** `#120` plans to replay
terminal history from the existing scrollback raw artifacts, and `#209`
established that a bounded byte-tail cannot reliably reconstruct a screen — a
browser joining mid-stream hits exactly what couch hits on thread switch and
`pair term` hits on tab switch. That argues for sequencing `#209` before or with
`#120`, and for its fix living in the shared seam rather than in couch alone.

**`#120`'s "use a proven terminal renderer" is the right call for a nameable
reason.** A terminal emulator is *world-determined* correctness — ANSI, wide
characters, mouse encodings — where an independent reimplementation is subtly
wrong in ways local tests do not show. Build what your domain determines; depend
on what the world determines. The relay is the former, the emulator the latter.
