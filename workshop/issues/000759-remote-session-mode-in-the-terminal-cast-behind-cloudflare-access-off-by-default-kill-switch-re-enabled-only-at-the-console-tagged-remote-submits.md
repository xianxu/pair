---
id: 000759
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5c1477e477b52366c3dc015473f8a57b768425c5' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-20
