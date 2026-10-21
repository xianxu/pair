---
id: 000761
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '519b75e9cbb29df8381a257b16b8490a05d5e20c' # card fields mirrored from issue-cards; edit via sdlc
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

### 2026-10-21 08:51 — read-only detach timing triage (pair:6, TL side task)

- Installed/running Couch reports clean vcs.revision 53a9d42e503efa47ecef42beec294a30ba87768f. It predates #761's landing and #764; neither can explain this image. Last main source before the previous build time (10-20 10:32) is 856876acd65a089426e4d1f62ced00444db347e4; that is a source comparison candidate, not a proven old binary revision.
- Leave/Detach/warm-helper paths are unchanged between those revisions. Fleet fan-out remains max(1, NumCPU/2): 6 on this 12-logical-CPU host. Borrowed warm helpers bypass Zellij ownership reads; recorded-incarnation global detach makes 3 ownership probes, each 2 sequential pane reads (6 logical reads/thread), versus 4 for direct Detach. #748 permits up to 3 native attempts for eligible empty/wrong-kind reads under the same 5-second deadline, with no retry sleep. This can amplify latency only if those replies occurred; it is not a measured cause. Registry persistence still serializes briefly; no newly serial fleet loop found.
- Retained metadata has 29 last_active_at timestamps around the 08:44 pre-restart exit, spanning 08:44:13.699611–08:44:25.350942 (11.651331 seconds). This is a retirement timestamp spread, not total detach duration or per-step timing; borrowed-helper detaches do not update it. No post-restart detach evidence found.
- COUCH_TRACE/INPUT_TRACE are unset; crash output has no lifecycle timings; supervision log stops 10-19. TL confirms the decision log is unavailable pending #736. Cause remains unproved. No live detach, slot stop, tracing change, product edit or old-binary run. TL is asking whether the report concerned the old-build 08:44 quit or a later detach; investigation stopped pending that evidence. #759 stays held for operator plan review.
