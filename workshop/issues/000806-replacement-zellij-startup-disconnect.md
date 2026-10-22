---
id: 000806
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'c6eae03080ca1f1650c63a51a8a447bc5b1c8f84' # card fields mirrored from issue-cards; edit via sdlc
---

# Replacement Zellij disconnects before first wrapper startup

## Problem

A disposable public reboot completed exact old-session retirement, but its replacement Zellij session disconnected before the first wrapper startup/agent-ready publication. Pair exited1 before Couch registration. This is distinct from #797's ownership refusal and quiescence ordering correction. Cause is unclassified; no flake is registered. It may relate to #773's missed-ready startup evidence, but no shared root cause is established.

## Spec

Retain and diagnose the new-session-before-first-wrapper disconnect through the shared disposable Couch harness. Distinguish a server initialization panic, an external termination, and failed client admission from downstream registration cancellation. Preserve exact-generation ownership and all startup/query fences, current time/frame/output/write bounds, and complete cleanup. Do not infer environmental causation from a passing rerun or weaken startup admission.

## Done when

- A retained failing startup includes the private Zellij server log and exact session/launch identities before root cleanup.
- Causal replay identifies the trigger and pins the failure with a regression through the existing startup path; unknown ownership remains refused.
- Disposable Claude and Codex startup/reboot+delivery acceptance and exact owned cleanup pass under existing bounds.

## Plan

## Log

Filed unassigned per authenticated ops:0 disposition5a670f4e-1ef7-49da-99ba-835f86d2814b while closing #797. The ownership fix passed native acceptance on both agents; the review's separate replacement-start failure remains recorded here rather than being classified as a safety-test flake.

Failure: reviewer pinned head003e3ba8deb1dd28cedb89b58774a1295b7916e6, Zellij0.45.1, TestLiveRecoveryOwnerReboot/codex. Go57.333s/sub46.39s, writes81424384/cap100663296. Public confirmation f6a256ff-6e79-4b9a-9f38-1769f302c783 ran10:15:11.169601–10:15:20.185623 PDT, failure nonce operation-d68b74db8d679ed8aedd38304f768342. Old session 📁9223559885839400961-2 / agent65380/start1792689303.200098 was absent. Replacement session 📁9223559885839400961-3, launch nonce start-9deeaa7f8d7bb690, lost its server before any replacement STARTUP-identity. PTY output: Lost connection to the Zellij server; terminal client exited before admission was verified. Existing bounded native query evidence is absent. Registration context cancellation follows Pair exit and is not the initiating cause. Cleanup joined direct child/IO and proved every owned generation absent.

Complete9233-byte reviewer output retained from its own transcript at /Volumes/TestScratch/pair797-br2-transcript-fgmhr1gg/reviewer-codex-native.log; SHA256b86a30bb1b3d2fb642557ee91b7a511807a9ce5926fcf6a2feb95166bf07a99d. The private root pc-2732206875, reviewer checkout and binaries were already cleaned; its server panic log was not retained. The original transcript is rollout-2026-10-22T10-11-57-01a16499-1829-7561-b781-9852d0ae2bbb.jsonl, CommandExecution output line156.

Controlled diagnostic: exact reviewed003e3ba8 source in a detached disposable checkout, a Go overlay adds only a bounded8192-byte read of existing private zellij.log AFTER confirmation; no production code, pre-confirmation action, assertion, or bound change. Overlay SHA25671e6f37992708b2d6525b562fb66c9b3a06cdd8f5c259f55e9a39e1aaae03c54. After normal runtimebundle generation, Codex confirmation e320a096-1397-4f90-bb76-4131c80df245 succeeded, old61659/start1792690077.658530 absent, new63356/start1792690086.142535 accepted public message, exact cleanup complete. Go30.182s/sub23.82s, writes81977344/cap100663296. Complete log /Volumes/TestScratch/tmp/pair-slot2/pair797-br2-probe-o0bm5t4z/native2.log; SHA256bd819f74b2424feaa2cf0886e4571d5a4581732bc7b9f72afd547fa3f9155b73. It retains all7236 bytes of private server log before cleanup, with several Server not ready/retry entries but no panic. The controlled pass does not explain or erase the failure.

Hypothesis only: the documented Zellij0.45.1 birth race can panic when a global query connects before its first client initializes (launcher/session_handoff_os.go observeAdmission comment; atlas/couch.md cold-resume birth section). This matches the failure boundary, but there is no triggering query or panic evidence in the failed run. The new quiescence carries only old session -2's immutable server generation and finishes synchronously before Fresh launches distinct -3; it cannot authorize the replacement. Ordinary park also waits for the old helper to die. Investigate independent background/global queries and any unproved helper cleanup rather than assigning causation from wording alone. #773 may be related; its evidence remains independently unclassified.

Read-only follow-up found independent query paths: message_service.go constructs ListingPoller(ZellijSource{}, messageListingInterval=1s) and starts its Run goroutine; immediate/periodic Refresh calls global list-sessions --no-formatting, with no birth-lease participation. Resource inventory seeding/lifecycle refresh also reaches SessionPresence/LivenessContext and current owner probes independently of terminal admission. The resource sampler itself and Git/filesystem hub prewarm do not issue Zellij queries; COUCH_RESOURCE_CADENCE=1s is not the listing cadence. These are concrete possible overlaps, not proof of the triggering query in the failed run.
