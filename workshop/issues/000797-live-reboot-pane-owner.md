---
id: 000797
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '7fcb4fdc4bdf23c556eb7633c56ec98f664c3034' # card fields mirrored from issue-cards; edit via sdlc
---

# Live reboot refuses owned pane as ambiguous

## Problem

The owning hub can obtain a live worker's exact reboot confirmation offer, but acknowledging it fails during session quiescence: `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. The current incarnation remains unreplaceable through the public recovery path. This is separate from #791's busy/banner admission problem, where the same reboot refusal appears only in the Log.

## Spec

Diagnose and correct the native pane-owner proof used by live reboot/recovery confirmation. Preserve fail-closed ownership and exact process-generation fences; do not treat an unknown actual command as permission to kill, weaken the validator, or increase deadlines. Return distinct evidence for unsupported/missing/ambiguous pane data so a legitimate owned session can be recovered through its public confirmation protocol.

## Done when

- A deterministic fixture reproduces the observed live worker and its native pane metadata through the actual ownership parser/decision; the corrected proof succeeds only for positively owned panes.
- Public live reboot followed by the returned recovery-confirmation token succeeds for genuine disposable Claude and Codex workers, while foreign/unknown/reused ownership remains refused.
- Preserve exact stop-set confirmation, cleanup and current caller authority; no numeric deadline tuning or permissive fallback.

## Plan

- [ ] Capture the native pane metadata and identify the lost/ambiguous ownership witness before changing the proof.
- [ ] Implement and verify the correction through the existing shared disposable-Couch harness and public commands.

## Log

2026-10-22: filed unclaimed per ops:0 approval d2356d3f-461c-4576-9ceb-a71acb4179f6 while qualifying #792. New fixture retained six descendants, proved their genuine ancestry, then released their parent to reparent them. All barriers passed. Public `couch --reboot fixture:1 --confirm --json` returned a valid confirmation-required offer excluding those detached children. Acknowledging its token failed before any #792 orphan-reap effect.

Exact failure: `slot zellij session ... kill not confirmed: session ... ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. Recovery-confirmation receipt ID `1e3f6b40-c5cb-4e3d-8436-8bdb70a0e78e`, failure nonce `operation-c585c70f0788d3a5d29ae5372fb3526c`; queued 0s, prepare 1ms, operation 393ms. The offered wrapper generation was PID80337/start1792659043.371776 and Claude child PID80545/start1792659043.605850. Exact disposable cleanup passed. Overall native command joined in31.258s.

Local retained log: /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-confirmation.kamQRO/native.log, SHA256794f8c0dce57ec187b5ae7eda2c90e80fd35a9974b280c4036e96eb8b40f4d0e (verified with shasum). The private runtime root was removed by normal cleanup; this text retains the actual refusal and correlation independently of scratch retention. Product/harness source is #792's f3f724125 checkpoint; subsequent uncommitted native-test changes only followed the valid public confirmation contract. No validator fix was attempted here. #792 uses a separately approved fixture-only crash to qualify its independent reap behavior.

An earlier native steward start missed its wrapper-ready publication before any retained fixture setup. Later unchanged-product startups succeeded. Cause remains unclassified; see #773's related missing-ready evidence class without assuming that issue's root cause. No flake registration or passing retry is claimed for that startup failure.
