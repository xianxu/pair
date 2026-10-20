---
id: 000720
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b9131c1a4b5abf54fb54b29fce8854075d5b316b' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

- Sibling: ariadne#477 (landed) makes sdlc override an unwritable inherited temp folder; this issue stops exporting
  one in the first place. Setup step for the allowlist line documented in ops `teams/devinfra/tools/testscratch/README.md`.
