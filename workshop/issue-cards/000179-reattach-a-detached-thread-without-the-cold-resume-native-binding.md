---
id: '000179'
status: done
started: 2026-09-03T16:18:09-07:00
created: 2026-09-03
updated: 2026-09-04
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
