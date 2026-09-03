---
id: 000179
status: open
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
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

**Root cause.** `DecideResume` applies `bindingResumeDiagnostic` to every
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
hides a session that would reattach fine.

The gate reached detached rows deliberately, in pair#170 M3
(`actionableinventory.go:249-258`): an ungated detached row could be
auto-selected by startup, refused by Resume, and take couch down in that tree
with no fallback. That diagnosis was right about the symptom and wrong about
the cause -- the refusal to fix was `RequiredSessionID` being demanded where
nothing consumes it, not the row being offered.

## Spec

Split the resume proof by what the path actually needs:

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

- A thread that is proved detached with a provisional, unbound or absent native
  binding is listed in the switcher and reattaches on Enter.
- A thread that is verified-parked with the same binding states is still
  refused, and the refusal still names its diagnostic code.
- A test crosses the two authorities against the four binding statuses, so the
  warm relaxation cannot leak into the cold path.
- The operator's own `pair-couch-24` (provisional binding, live session)
  reattaches on the real stack.

## Plan

- [ ] Reproduce as a unit test over `DecideResume`: detached + provisional
      binding is currently refused, and that refusal is the bug.
- [ ] Split the proof: warm reattach stops consuming the native binding, cold
      resume keeps it. Own predicate for the warm launch precondition rather
      than a relaxed `RequireNativeResumeBinding`.
- [ ] Drop the mirrored gate in `ActionableThreadInventoryContext` for detached
      candidates only, keeping it for parked ones.
- [ ] Re-check the pair#170 M3 symptom: a detached row that cannot reattach
      must not take couch down at startup.
- [ ] Atlas: the two resume authorities and what each proves.

## Log

### 2026-09-03

- Filed from pair#170 operator smoke. Evidence above is measured against the
  live store and zellij, not derived from reading.
