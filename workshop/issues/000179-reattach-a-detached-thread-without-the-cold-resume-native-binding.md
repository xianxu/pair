---
id: 000179
status: codecomplete
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-04
estimate_hours:
started: 2026-09-03T16:18:09-07:00
actual_hours: N/A
---

# Reattach a detached thread without the cold-resume native binding

## Problem

Operator report during pair#170 smoke: *"I can't even attach to detached but
running zellij/pair sessions."* couch-lite scales couch DOWN, but the switcher
over your own sessions has to work, and reattaching a warm session is the
smallest thing it owes.

Measured against the live store on 2026-09-03. `couch --list` classifies two
threads correctly:

```
couch-64bbe04986164fae /Users/xianxu/workspace/tools  (no client attached; agent may still be running)
couch-8d1a4da0f9fe730d /Users/xianxu/workspace/pair   (no client attached; agent may still be running)
```

Both zellij sessions are alive (`tools-couch-2`, `pair-couch-24`). A probe over
the real data through the production seams shows only one of them can ever
reach the switcher:

```
e108517d46ab4575/couch-8d1a4da0f9fe730d
  native binding: status=provisional id=""     <- DROPPED
  pair session:   name="pair-couch-24" present=true
  detached proof: complete except NativeID
434128d5ad68b26e/couch-64bbe04986164fae
  native binding: status=established id=ea6a5c9b-...
  pair session:   name="tools-couch-2" present=true
  detached proof: complete
```

**Root cause 1 (primary): couch's resume authority only exists at a CREATE
boundary, and a detached thread is an ATTACH boundary.** Proven by reading the
whole chain, every step exact:

```
couch      launch_existing.go:32   pair resume <tag> --layout2  +  COUCH_LAUNCH_PROFILE{ResumeRequired:true}
pair       args.go:111             → ForcedTag
pair       decision.go:32-36,90-98 → sessionBlocksReuse(SessionDetached)=true ⇒ ActionAttach
pair       createflow.go:238       → ResumeRequired && Action != ActionCreate ⇒ REFUSED
                                     "required Couch resume no longer resolves to a create boundary"
```

`ResumeRequired` is couch's trusted authority: don't prompt, don't pick, resume
this exact native session. It was built for the COLD case, where the agent is
dead and pair must create a fresh session running `--resume <native id>`. It
additionally asserts "and this will be a create", which is false for every
detached thread -- whose zellij session is alive by definition. Since #170 made
`leave` detach rather than park, detached is the NORMAL resting state, so the
normal way back in is the one path that cannot work.

This was never covered end to end: M2/M3's reattach tests are couchcore-level
with fakes on the artifact seam, and the refusal lives on the far side of a
process boundary in pair's launcher. Both halves are green and the whole is
broken.

**Root cause 2: the cold proof gates the warm row.** `DecideResume` applies
`bindingResumeDiagnostic` to every
resume (`couchcore/resume.go:120-125`), and `ActionableThreadInventoryContext`
applies the same gate before a row is even offered
(`couchcore/actionableinventory.go:259-262`). That gate is the **cold** resume's
proof: a parked thread has no agent, so pair must restart it with
`--resume <native session id>`, and an absent or provisional binding means the
restart cannot work. A **detached** thread restarts nothing -- the agent is
still running inside its live zellij session and `pair resume <tag>` reattaches
to it (README: "If the tag's public zellij session is still running (for
example, after `Alt+d` detach), `pair resume <tag>` re-attaches without
prompting"). The native session id is irrelevant to that path, so demanding it
hides a session that would reattach fine. This one fires EARLIER: a row it
drops never reaches the create-boundary refusal, which is why the two threads
fail differently. `pair-couch-24` is hidden by root cause 2; `tools-couch-2`
passes every couch-side gate and would be refused by root cause 1.

The gate reached detached rows deliberately, in pair#170 M3
(`actionableinventory.go:249-258`): an ungated detached row could be
auto-selected by startup, refused by Resume, and take couch down in that tree
with no fallback. That diagnosis was right about the symptom and wrong about
the cause -- the refusal to fix was `RequiredSessionID` being demanded where
nothing consumes it, not the row being offered.

## Spec

Two resume shapes, each refused at the other's boundary.

**Warm reattach (the thread is proved detached).** couch's profile says so, and
pair honours `ResumeRequired` at an ATTACH boundary: no config picker, no
`composeResumeArgs`, no `RequiredSessionID`, no native-binding check. The proof
is the SESSION -- an unambiguous name binding to this exact address, live, zero
clients, which `ProjectDetachedSessions` already establishes -- not the agent's
transcript id, which nothing on this path consumes.

**Cold resume (the thread is verified-parked).** Exactly as today: a create
boundary, an established binding with a non-empty root id, `--resume <id>`.

The authority must be EXPLICIT, not inferred by pair from the session state.
`TrustedLaunchProfile` carries which shape couch proved, and each shape is
refused at the wrong boundary -- warm at a create means couch's detached proof
went stale between projection and launch (the session died), and cold at an
attach is today's refusal, still correct. Loosening :238 to accept any action
under `ResumeRequired` would let a stale couch attach to a session it never
proved, which is what that guard is for.

**Pair must not regress.** Nothing changes when `ResumeRequired` is false --
standalone `pair`, `pair resume <tag>`, and the picker keep their current
behaviour, and the new profile field is optional with the cold shape as its
zero value. A pair-standalone check is part of Done.

Then, so the warm row can reach that path at all:

- **Cold (verified park):** unchanged. An established native binding with a
  non-empty root id, carried into `BuildCouchResumeLaunchProfile` as
  `RequiredSessionID` and re-checked immediately before launch.
- **Warm (proved detached):** the proof is the *session*, not the agent id --
  a live zellij session bound to this exact address with zero clients, which
  `ProjectDetachedSessions` already establishes. No `RequiredSessionID`, and
  no native-binding status gate.

The warm path must still refuse where refusal is real: an occupied
incarnation, a missing working path, a missing or unsupported launch profile,
an ambiguous session binding. Only the native-binding status/id gate moves.

Open questions to settle in design, not here:

1. Does `pair resume <tag>` need any argument change when the public session is
   live, or does the existing launch profile reattach as-is with
   `RequiredSessionID` empty? `RequireNativeResumeBinding` currently refuses an
   empty required id outright (`launcher/launch_args_policy.go:36-41`), so the
   warm path needs its own predicate rather than a relaxed shared one.
2. Whether a detached thread whose binding later becomes established should
   prefer the cold proof (it costs nothing and is stronger) or stay warm.
3. Whether startup auto-resume should treat a warm-reattach refusal as fatal at
   all, given the original M3 symptom was couch dying in that tree.

## Done when

- A thread whose zellij session is alive with zero clients reattaches from the
  switcher on Enter, landing in the running agent rather than refusing.
- A thread that is proved detached with a provisional, unbound or absent native
  binding is listed in the switcher and reattaches on Enter.
- A warm profile whose session died between projection and launch is REFUSED,
  not silently upgraded to a create.
- Standalone pair is unchanged: `pair`, `pair resume <tag>` and the picker
  behave identically with no couch profile present, checked on the real stack.
- A thread that is verified-parked with the same binding states is still
  refused, and the refusal still names its diagnostic code.
- A test crosses the two authorities against the four binding statuses, so the
  warm relaxation cannot leak into the cold path.
- The operator's own `pair-couch-24` (provisional binding, live session)
  reattaches on the real stack.

## Plan

Shipped as pair#181 M2. Ticked against what landed, with the one item that
turned out unnecessary marked as such rather than claimed.

- [x] Reproduce the refusal as a test before any fix. Only (b) was needed:
      `DecideResume` refusing detached + provisional binding, red first with
      `resume-binding-provisional`. (a) was never reached — see the item below.
- [x] Split the proof: `DecideResume` applies the native-binding diagnostic only
      to a verified park, and `ResumeContext` no longer resolves a binding at all
      on the warm path. `confirmStillDetached` is the warm precondition,
      mirroring `RequireNativeResumeBinding` rather than relaxing it.
- [x] The mirrored inventory gate went in pair#181 M1, which made classification
      total: the row is listed with its reason instead of being dropped.
- [x] Startup no longer dies mutely on a resume refusal — `startupResumeRefusal`
      names the thread and the ways forward. No-fallback was right; refusing
      without a next step was not.
- [x] Atlas: `atlas/couch.md` records the two resume authorities and what each
      proves, and that the native-binding gate no longer applies to the warm path.
- [-] NOT NEEDED: the `createflow` create-boundary guard. The Spec assumed a
      `ResumeMode` threaded through Pair's launch profile to satisfy it. couch
      simply stops sending `ResumeRequired` on a warm reattach, so the guard is
      never reached and `launcher/` is untouched — which also keeps the standing
      constraint that Pair must not be degraded.

## Log


- 2026-09-04: closed — Shipped as pair#181 M2 and verified there: reattaching a detached session works on the real stack — tools-couch-2 reattached with its Pair ledger unchanged at 6 rows and its zellij session still the one created 8h56m earlier, proving a reattach rather than a relaunch; the couch restart auto-reattached the operator into the live pair session. The Spec was right about the split (the native binding is the COLD path proof; a warm reattach consumes it nowhere) and wrong about the mechanism: it assumed a ResumeMode threaded through Pair TrustedLaunchProfile and LaunchArgs, and the shipped fix touches launcher/ not at all — couch simply stops sending ResumeRequired, an authority Pair honours only at a create boundary, so the guard the Spec planned around is never reached. That also preserved the standing constraint that Pair keeps working standalone. --no-actual because the hours are recorded against pair#181 M2 and counting them twice would corrupt velocity calibration. --no-judge because this code was reviewed at the pair#181 M2 and M3 boundaries, five rounds, and a judge here would review a window containing none of it. --no-atlas/--no-project because atlas/couch.md and workshop/projects/couch.md were both updated under pair#181, which lists this issue as absorbed.; review verdict: not-run
### 2026-09-03 — shipped as pair#181 M2

Absorbed rather than worked separately: this issue's design became pair#181's
second milestone, and its code landed there.

What shipped matches the Spec's split -- the native binding is the COLD path's
proof and a warm reattach consumes it nowhere -- with one correction the Spec
had wrong. It assumed the fix needed a `ResumeMode` threaded through Pair's
`TrustedLaunchProfile`, `LaunchArgs` and a new boundary guard. The operator
asked why reattaching was not just a zellij command, and the honest answer was
that it is: `AttachExistingSession` is env, a title poller and `zellij attach`,
and standalone `pair resume <tag>` already reaches it. couch was blocked only
because it SENT `ResumeRequired`, an authority Pair honours at a create
boundary. So the fix is couch declining to send it, and `launcher/` was not
touched at all.

Also corrected: relaxing `DecideResume` alone would never have reached the
operator's thread, because `ResolveEstablished` refuses a provisional binding at
`resume.go:212` before any decision is made.

Verified on the real stack: `tools-couch-2` reattached with its ledger unchanged
at 6 rows (Pair appends a launch row only on create, so no new row proves no
relaunch) and its zellij session still the one created 8h56m earlier.

### 2026-09-03

- Filed from pair#170 operator smoke. Evidence above is measured against the
  live store and zellij, not derived from reading.
- **Corrected the root cause after tracing into pair.** Filed blaming only the
  native-binding gate; that gate is real but SECOND. The primary failure is
  pair refusing a couch resume at an attach boundary, which no couch-side test
  can see because it lives across the process seam in the launcher.
- **Why that binding is provisional: pair#168, measured.** The ledger for
  `couch-8d1a4da0f9fe730d` is a `launch` row with no `binding` row after it --
  pair#168's exact description ("Pair appends a new launch row but before it
  appends the corresponding binding"). The reattachable `couch-64bbe04986164fae`
  has the complete `launch`/`binding` pairs:

  ```
  couch-8d1a4da0f9fe730d:  0 legacy(session_id)  1 launch                       <- provisional
  couch-64bbe04986164fae:  0 legacy  1 launch  2 binding  3 legacy  4 launch  5 binding
  ```

  So #168 is the CAUSE and this issue is why the cause is FATAL. They are one
  operator-visible bug from two directions and should be fixed knowing about
  each other -- but they are not redundant: #168 restores a binding by falling
  back to the previous established one, and this thread has none to fall back
  to (row 0 is a legacy row, not a binding). Only the warm-path fix recovers
  this instance, which is also the argument that the warm path should never
  have depended on the binding.
