---
id: 000720
status: done
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 1.56
github_issue:
started: 2026-10-19T21:04:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 0.68
tracker:
    version: 1
    completion:
        token: close-dc800b42d3c6
        repository: github.com/xianxu/pair
        reviewed_head: 5e2686d4bb1354880671ae468a280c9b9383db6c
        evidence_commit: 998af57acd82c2136db82a7a1015abb6ee70b16a
        landed_commit: c2c6b9279dbdb77aaa7d3b68498ea4326156cd20
---

# Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox)

## Problem

pair#708 exports `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR` pointing at `/Volumes/TestScratch/tmp/<slot>` into every
slot (couchscratch/temp.go, `Prepare` → env at ~line 189). It does probe the folder with a real write first, but the
probe runs in Couch's own process, outside the agent's sandbox. On 10-19 after the reboot the probe passed while the
Claude sandbox denied the path, so every sandboxed `go build` in every slot failed ("creating work dir … operation not
permitted") until the operator added `/Volumes/TestScratch` to `sandbox.filesystem.allowWrite`. A probe that can pass
while the consumer can't write proves nothing about the consumer.
