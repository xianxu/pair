---
id: 000720
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c579a433e56e76b78f97c4c056bfeb5d0ad5c3e2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:04:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox)

## Problem

pair#708 exports `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR` pointing at `/Volumes/TestScratch/tmp/<slot>` into every
slot (couchscratch/temp.go, `Prepare` → env at ~line 189). It does probe the folder with a real write first, but the
probe runs in Couch's own process, outside the agent's sandbox. On 10-19 after the reboot the probe passed while the
Claude sandbox denied the path, so every sandboxed `go build` in every slot failed ("creating work dir … operation not
permitted") until the operator added `/Volumes/TestScratch` to `sandbox.filesystem.allowWrite`. A probe that can pass
while the consumer can't write proves nothing about the consumer.

## Spec

Export the scratch variables only when the slot's agent can write there; otherwise leave the variables unset (the
normal temp folder) and print one note naming the missing allowance.
- Claude: read the effective `sandbox.filesystem.allowWrite` (user, project and local settings; `sandbox.enabled`) and
  require the scratch root, or a parent, to be listed. Codex: the same against its sandbox's writable roots if it has
  one, else treat it as allowed.
- One pure predicate (settings → allowed?) with table tests; the IO stays at the existing seam.
- `couch --show` / slot diagnostics show "scratch: RAM disk" or "scratch: normal temp (sandbox doesn't allow
  /Volumes/TestScratch)".

## Done when

- A slot whose agent sandbox lacks the RAM disk gets no scratch variables and a note; with it allowed, it gets them
  (tests over settings fixtures, including an absent settings file and sandbox disabled).
- The 10-19 failure mode is a test: probe passes, sandbox denies → no export.

## Plan

- [ ] Implement the shared pure sandbox predicate and bounded standard settings projection with red-first table/fixture tests.
- [ ] Gate scratch exports with the actual selected launch profile; share the read-only selection with show diagnostics and update the atlas.
- [ ] Run focused checks and recorded full suite, close review, land through SDLC and report to the hub.

Durable design: [sandbox-aware slot scratch](../plans/000720-sandbox-scratch-plan.md). One atomic close boundary; full flow because settings and diagnostic plumbing exceed the quick production-line shell.

## Log

- Sibling: ariadne#477 (landed) makes sdlc override an unwritable inherited temp folder; this issue stops exporting
  one in the first place. Setup step for the allowlist line documented in ops `teams/devinfra/tools/testscratch/README.md`.

### 2026-10-19 — design approved

Claimed in pair:7 and started from fresh main. Root seam: scratchSlotEnvironment delegates to a host write probe with no consumer-policy admission. A read-only research subagent checked official Claude settings/managed-policy semantics and the absence of an existing sandbox reader. Hub approved the conservative Claude projection and narrow Codex TMPDIR check, then the compact full-flow shape (6cfb6a23-fb86-4013-93fd-1047d7a7b73c). Required positive control: enabled Claude user settings with allowWrite containing /Volumes/TestScratch must stay on RAM. No settings/credential writes or native agent startup. Current Codex tool calls are filesystem-unsandboxed (danger-full-access); they are not sandboxed-conformance evidence. Codex's source explicitly defaults TMPDIR exclusion false and adds TMPDIR to workspace-write roots; links and trust envelope are in the durable plan. Live follow-up is the next natural relaunch; no staging on pair:0.
