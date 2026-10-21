---
id: 000765
status: done
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-21T03:09:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 1.29
tracker:
    version: 1
    completion:
        token: close-608451f261c3
        repository: github.com/xianxu/pair
        reviewed_head: 991599da3c42aba8eac2fa7fd9c6ac28572b50fe
        evidence_commit: 93cd418a2e52a99df971ed42efd676835d5d4fec
        landed_commit: feb1d595834706d2f785214c4e3ea003f980559d
---

# Cache-enabled Go tests exceed physical write envelopes

## Problem

#739 close review observes different physical-write envelopes for plain cache-enabled go test and go test -count=1. At branch 1b948b3c040f6979eaafa77b1ad2da127f524da4, the reviewer measured couchcore 157257728 bytes against its 134217728 cap and couchtty 88002560 against its 67108864 cap using plain go test across five packages. The assertions pass; TestMain rejects the write counts. Both packages pass -count=1, as used by the required make suite. Earlier tty controls measured exactly 39800832 bytes with and without #739-added tests; new tests alone wrote zero.

The reviewer traced Go's cache-enabled filesystem-access logging as an invocation-dependent source of writes. Main-versus-branch cold/warm measurements are requested by the hub and pending; attribution is not yet proven. This issue is independent of #739's startup behavior and does not authorize any cap increase.
