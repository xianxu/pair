---
id: 000761
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bd47f40dd12cc5e69a644821cfa0355b012eb26c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T00:55:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0c1c0f31", done: "f1e23da3"}
---

# Couch native detach intermittently loses post-client ownership proof

## Problem

During #744 native qualification, real Couch leave-detach refused after stopping a worker client: `Couch: leave couch: detach 1-fixture-2: observe Pair session before retiring its incarnation: session "📁 1-2" ownership unresolved`. The remaining diagnostic was clipped by the terminal row. This preserves a real post-client-exit ownership refusal; no owner check was weakened.

## Spec

Diagnose the post-client-exit SessionOwnerUnknown result at couchcore/detach.go:191 and launcher/session_owner_os.go. Retain the exact refusal and both authoritative native pane replies plus server PID/generation/socket evidence. Fix a demonstrated product defect if found; otherwise identify the invalid fixture condition with a regression. Do not grant ownership from cached layout, pane titles or absence of evidence.

## Done when

- Reproduce or positively classify the observed refusal using an isolated real Zellij/Couch fleet.
- Preserve strict pre/post ownership fences and add a regression at the actual failing boundary.
- Recheck real detach/restart without leaving running owned generations.

## Plan

- Inspect the full diagnostic and native query evidence; use #744 shared fleet or existing TestStartupManagedAttachDetachTimingLive.
- Fix the established class and run focused plus required verification.

## Log

Observed on clean #744 head5aa2570ca7f9cb1be8771903df943618759ca492, native Zellij0.45.1 and installed SDLC8d34e76c815fe389f7979c966418dcdb2a04aaa5. One genuine Claude steward plus three non-hub workers registered; concurrent CLI queries and submitted/accepted ordinary message passed before detach refused. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-xyneu9_b/native.log, SHA2569030df57986142026b443b6bd53158a0f1a29a049bb378813f4a94f457bfb2e1. Every owned process generation was later confirmed absent and the private root removed.

A deliberate diagnostic rerun at dcdffcea8f14b4465f025c3d40839968082e2a32 added public copy-notice capture only; its Claude subcase completed detach/restart30.60s and cleanup. That run did not reproduce the ownership refusal; it later failed the independent Codex fixture composer check. Full diagnostic log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-diagnostic-phd4gpjx/native.log, SHA256c83886c74d8553e3a44fef3082f2d198410bcdb94baad557602a71b69aff2dcb. This is not registered as a known flake: there is no same-exact-head fail/pass evidence. No product fix or root cause is claimed.

### 2026-10-21 — recurrence and initial trace

Final #744 clean candidate bfa769c1dec8ee80eabba775220d7084ae1c4983 failed48.08s after genuine Claude four-actor membership and submitted/accepted ordinary message. Public refusal: detach1-fixture-3, observe Pair session before detach, session 📁1-3 ownership unresolved: invalid live pane JSON, operation-392b33c981ed0bbda69e1a8a9d6763ed. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-final.nEqfD0/native.log SHA2561d4b86f5c114a28e372bb025ee7c5eaf30355f05d301802d0f1ecddc420ce7b5. Exact owned generations absent, IO joined, root removed; no Codex attempt after failure. The authoritative bytes were not retained, so truncation, wrong reply and concatenation remain unproved.

Pair collection uses distinct stdout/stderr buffers and joins process/pipe collection; parser input is not limited by the observer's64KiB copy. Upstream taggedv0.45.1 emits pane JSON via serde pretty serialization into one Log; IPC reads a length-prefixed payload with read_exact. Numeric connection-ID reuse and late Log routing are established #748 context, but #761's offending bytes are still needed. Existing #748 recovery intentionally handles only empty or fully validated crossed kinds; malformed nonempty output remains a refusal. No product behavior changed during this trace.

## Revisions

### 2026-10-21 — TL159f625e: recurrence at the pre-detach fence

The first observation was post-client retirement. The final #744 rerun now refused at the pre-detach ownership probe with concrete diagnostic invalid live pane JSON. Diagnose the shared authoritative pane-query class across both fences, preserving all ownership requirements. #744 is parked/pushed f86a6c6a023ec4a794d0b7e2bdbe202c9c3ab689; this issue starts independently from landed main. No unlanded helper source is imported.

Investigation plan: use the existing #748 bare-native diagnostic as an owned Go overlay, with physical private HOME/XDG/Zellij roots and shell stand-ins only. Capture every completed production query, including successful subprocesses later rejected by ParsePanes, plus independent final validation. Run one bounded mixed pane/client read schedule; retain exact stdout/stderr, argv, exit and byte counts. Check cleanup errors and prove exact private native generations absent. No live slot or Couch broker is opened. After establishing bytes/root cause, add a deterministic failing boundary regression and record the smallest fix plan. Reject blind retries and widened ownership proof (ARCH-SECURE/ORDER).

Observer copies suffice for bounded replies and include truncation flags; an owned forwarding shim is available only if full bytes exceed64KiB. Reusing #744 source here is rejected because it would stack independent issues. Existing private native fixtures avoid that dependency. No product workaround is selected before capture.
