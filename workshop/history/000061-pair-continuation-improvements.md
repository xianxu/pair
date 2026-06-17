---
id: 000061
status: done
deps: [ariadne#105]
created: 2026-06-15
updated: 2026-06-15
estimate_hours: 3
actual_hours: 2.28
---

# pair-continuation improvements

We should check if it does the following:

1/ make sure key exchange results have been captured in durable artifacts (mainly: pensive, notes, issues, targets). If not yet, this is a good time to do that. 

2/ if there are lessons you have learned in this session, you should write a section in the continuation file on it. 

3/ the continuation document should build around those durable artifacts, as they represent improve understanding of problems at hand. It should explain the history, reasoning and connections among those durable artifacts. 

4/ we should have a section of the arc of the whole thread, where we started, how it's side tracked to various topics. think about the underlying connections among those topics, and uncover user's latent intention among those pivots. the end goal is to establish a model of user's mental model; this model of user's mental model should be 1/ internally self consistent; 2/ fitting observation of user interaction in this session. when there are internal inconsistency, write a section of open questions, and ask user about them when this continuation restart in a new session.

5/ then connect the above (history and learning), to next steps that we need to continue on.

## Done when

- The continuation procedure (the **ariadne#105** datatype rewrite) addresses all
  five asks above: flush-first, lessons, build-around-artifacts, thread-arc/user-
  model/open-questions, and connect-to-next-steps.
- A real continuation of a live pair session, produced under the new procedure,
  exhibits the new sections (Thread arc & user model, Open questions w/ embedded
  resume directive, Artifact map, Lessons learned) and **round-trips** via
  `pair continue <slug>`.
- Confirmed pair resolves the updated datatype at runtime (no copy of
  `continuation.md` lives in pair today, so the dispatcher reads it from the
  ariadne substrate — recompose only if weave is changed to copy datatypes).

## Spec

The continuation **procedure** (body skeleton + authoring instructions) lives in
ariadne's `construct/datatype/continuation.md`; pair consumes it via weave and
provides only the mechanics (`pair-continuation` writer) + the park/continue flow.
So the substantive change lands in **ariadne#105**; this issue is the dogfood
umbrella, owns validation, and carries **one pair-side code change** (below).

Core reframe: the continuation becomes the **connective narrative over the
session's durable artifacts** — detail lives in the flushed artifacts, the
continuation explains how they connect and where the user's head is. Full design,
body skeleton, and rules: see **ariadne#105**.

Agreed decisions (operator sign-off):
1. Section set & order as in ariadne#105; Live deliberations and Decisions & dead
   ends stay separate.
2. Flush-first is a procedure step, not a body section; what was flushed shows up
   in the Artifact map.
3. **Automated flush is `pensive`-only** — `target` needs human review,
   `meeting-notes` doesn't fit.
4. **Writer unchanged** — `pair-continuation` keeps enforcing only `## NEXT ACTION`.
5. **No seed-prompt change** — the resume directive ("On resume, resolve the open
   questions with the user before continuing with the NEXT ACTION") is embedded in
   the generated continuation file itself.
6. Datatype change lands in ariadne#105 (gated by ariadne sdlc); pair#61 is the
   umbrella + dogfood.

Related: **ariadne#103** (maintain the user-model live every turn) is the
complementary half — #61/#105 persist/checkpoint what #103 maintains. **ariadne#90**
(docflow suspend/resume) is the sibling-domain analog.

**Pair-side change (found during dogfood prep).** The Alt+Shift+C compaction prompt
(`nvim/init.lua` `COMPACT_PROMPT`) enumerated the *old* skeleton ("NEXT ACTION,
open threads, decisions/dead-ends") and only softly pointed at "the continuation
mechanism" — so pair's **primary** park path would not reliably exercise #105's new
procedure. Reroute it to follow the continuation **datatype procedure** (flush-first
+ new skeleton), staying agent-agnostic (no skill name, no hardcoded path). This is
the load-bearing connection between #105's datatype work and pair's real workflow.
It is **distinct from decision 5**, which is about the *resume* seed prompt
(`bin/pair`, line ~1943) — that stays unchanged.

## Plan

- [x] ariadne#105 — rewrite the continuation datatype (the substance). Landed
      (PR #42, boundary review SHIP).
- [x] Confirm pair picks up the ariadne datatype change at runtime — verified:
      `construct/datatype` is a symlink → `../../ariadne/construct/datatype`, so
      pair resolves the updated `continuation.md` live (no recompose, no wiring).
- [x] Reroute the Alt+Shift+C compaction prompt (`nvim/init.lua` `COMPACT_PROMPT`)
      to follow the continuation datatype procedure — drop the stale old-skeleton
      enumeration; stay agent-agnostic. Landed (PR #30, `c0107aa`); `luac -p` +
      `make test-queue` green. No test pins the prompt text.
- [x] Dogfood (self-park of a live pair session). All pass conditions met:
      (a) `## Artifact map` points at the real pensive flushed *this* session
      (proves flush-first fired); (b) `## Open questions` leads with the verbatim
      resume directive; (c) the new reflective sections (Thread arc & user model,
      Lessons) are present + non-boilerplate; (d) `pair continue <slug>`
      round-trips — confirmed by *this* seeded session, which resolved the Open
      questions with the operator before acting on the NEXT ACTION.

## Log

### 2026-06-15
- 2026-06-15: closed — dogfood: a real Alt+Shift+C park drove the new COMPACT_PROMPT → the writer produced a new-shape continuation (Artifact map→this-session pensive, Open questions led by the resume directive, Thread arc & Lessons present); pair continue seeded THIS session, which honored the embedded resume directive (resolved the 3 open questions with the operator) before acting on the NEXT ACTION — pass-conditions a–d all met; review verdict: SHIP
- Claimed. Gap analysis of the current procedure vs the 5 asks; design + 6
  decisions agreed with operator. Opened **ariadne#105** for the datatype rewrite
  (its real home); this issue is the dogfood umbrella.
- ariadne#105 landed (PR #42, SHIP). Verified pair resolves the new datatype live
  via the `construct/datatype` symlink.
- Dogfood prep found the real pair-side change: `COMPACT_PROMPT` (Alt+Shift+C) still
  drove the old skeleton. Rerouting it through the datatype procedure so pair's
  primary park path picks up #105.
- **Dogfood complete.** A real Alt+Shift+C park drove the new `COMPACT_PROMPT` →
  the writer produced a new-shape continuation
  (`workshop/continuation/20260615T225403-cont-improve.md`); `pair continue`
  seeded a fresh session that honored the embedded resume directive (resolved the
  3 Open questions with the operator) before acting. Pass-conditions a–d all met.
- Open-question resolutions (recorded canonically in **ariadne#103** Log): (1) the
  user-model's two criteria collapse to #103 as single source — the continuation
  section defers by pointer; (2) the user-model stays **cross-cutting** (continuation
  section + #103 instruction + pensive flush), no separate durable artifact.
- Atlas updated: `atlas/architecture.md` In-session-compaction paragraph now notes
  `COMPACT_PROMPT` defers to the continuation datatype procedure (the drift pair#61
  fixed) — committed with this close.

