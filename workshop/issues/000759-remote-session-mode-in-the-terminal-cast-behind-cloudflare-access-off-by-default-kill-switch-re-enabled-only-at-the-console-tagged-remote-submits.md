---
id: 000759
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9fa6bf216ed66532d11c08b13ec25a5522dd1c01' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T01:17:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits

## Problem

remote-session (ops project, operator-approved 10-20): write access to a session file instructs agents that run with
bypass permissions, so remote access is a security surface. Chrome cast and terminal cast assume the operator is at the
machine; 👽 mode assumes they're not.

## Spec

- **👽 mode in the terminal cast:** off by default (`couch --remote on` at the machine), served through the existing
  cloudflared tunnel, with **Cloudflare Access in front** (the operator's email one-time code or a passkey), so
  unauthenticated requests never reach Couch.
- **Least capability:** the remote API reaches only registered session files and side docs (through the allowlisted
  proxy) and submit; no arbitrary files, shell or lifecycle verbs.
- **Kill switch:** one action on the phone stops the tunnel, ends every remote session and writes a "remote disabled"
  mark; while it exists, Couch refuses to start remote access. **Only a keystroke in Couch's own UI at the machine**
  clears it: no CLI verb, no remote verb, and `couch --keys` must never be able to drive Couch's own UI (test it).
- **Remote submits are tagged** ("submitted remotely") so the TL knows the source.
- Short-lived sessions; an audit line per write and submit.

## Done when

- Tests: no request reaches Couch without Access; the kill switch persists across Couch restarts and refuses
  re-enable from CLI, remote and `--keys`; a console keystroke clears it; remote submits carry the tag.
- This is a fleet-wide, security-sensitive design: its plan goes to the operator before code.

## Plan

Durable proposed design: [remote security plan](../plans/000759-remote-security-plan.md). Operator approval is required before code; no change-code has run.

- [ ] Operator reviews threat model, exclusive named-tunnel use, session bounds and #758 integration/acceptance sequencing.
- [ ] M1 — Verified Access identity, pure lifecycle/session policy, durable latch, bounded audit and replay state.
- [ ] M2 — Exclusive protected exposure, console-only reset, CLI refusal and kill/late-completion conformance.
- [ ] M3 — Trusted remote provenance through submit/accounting, document-adapter integration, operator live security drill and landing.

## Log

### 2026-10-20


### 2026-10-21 — planning only

- Authenticated dispatch 942390da-f154-4785-bd03-d6108e1ae119; card unowned, claimed and start-plan completed in pair:6. Read ops remote-session project.md and design.md before design. No project edits, Cloudflare account/configuration/network changes or live tunnel starts.
- Source audit: current production cast links have a numeric-code lock in addition to URL tokens (#495); earlier shorthand saying bearer-only was corrected to TL. View-only, pointer annotations, and Chrome drive have separate tokens/codes and capability checks. Tokens and unlock sessions have no server wall-clock TTL; URL tokens last until cast end. All link kinds share a per-link cross-IP bucket (burst5, then1/sec), rotate the numeric code after8 wrong well-formed attempts, preserve already-unlocked sessions on forced rotation, and keep URL token unchanged. The existing default quick-tunnel exposure question was reported separately to TL, not silently folded into this item's code scope.
- Read-only parallel mapping identified the host-TTY input path versus broker/wrapper keys, shared Host last-mount shutdown, the public200 readiness probe, document ID/registry lifecycle, and SubmitPath's exact first-line parsing. Plan reuses these authorities rather than exposing generic IPC or raw nvim methods (ARCH-DRY/ARCH-SECURE).
- TL793b0f54-b454-4226-a0da-227c22d7a76b confirmed protected named ingress, application verification of the signed Cf-Access JWT against team keys, no quick/local fallback, separate routes and the explicit same-UID threat-model limit. Proposed plan also makes remote exclusive on the named transport, persists armed state before exposure to survive kill-publication failure, and reserves all reset authority for Couch's host-key path.
- #758 currently has no owner/design/code. The plan names a strict document backend/permit seam and asks TL/operator to settle real-nvim acceptance sequencing before implementation; fake-backed security tests alone do not establish the remote editor works.
- Awaiting one advisory fresh-eyes plan read, then send the durable plan to TL for operator review. No implementation approval inferred from this checkpoint.
