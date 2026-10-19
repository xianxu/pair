---
id: 000704
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
severity: medium
---

# TestParkLifecycleLive fails locally on main: quit intent does not match observed session owner

## Problem

TestParkLifecycleLive fails deterministically on the operator's machine on unchanged main, while CI's macOS conformance job passes it. It fails on the operator's machine on every run so far: 5/5 on the #692 branch (`-count=5`) and 3/3 on origin/main 04e2216a in a clean detached worktree (2026-10-19). The sibling TestParkLifecycleLiveIntentOnlyMutation passes 5/5 in the same runs. #669 covers a different native test (continuation registration timeout) and does not cover this one.

Exact output (first failure; each repetition is identical apart from temp paths and timing):

```text
=== RUN   TestParkLifecycleLive
    park_lifecycle_live_test.go:237: quit intent does not match observed session owner
    park_lifecycle_live_test.go:481: owned helper PTY diagnostics: "...zellij screen with the owner stub pane: /tmp/pair-owner-stub-1189795670/pair wrap --scrollback-log /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/TestParkLifecycleLive2876366904/..."
    recovery_query_capture_test.go:249: recovery query capture: {"Records":[{"Kind":"lifecycle","Name":"fixture-roots home=\"/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/TestParkLifecycleLive2876366904/001/home\" ...
--- FAIL: TestParkLifecycleLive (0.67s)
```

Command: `PAIR_LIVE_COUCH=1 env -u COUCH_THREAD_SCOPE -u COUCH_THREAD_TAG -u ZELLIJ -u ZELLIJ_SESSION_NAME go test ./cmd/internal/couchcore -run '^TestParkLifecycleLive$' -count=3 -v`, run unsandboxed from a Couch slot (pair:2). Host: macOS 27.0.1 arm64, zellij 0.45.1, go1.27.1, TMPDIR=/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/.

Full retained log (both tests, five repetitions, complete query captures): /Users/xianxu/workspace/pair/.git/pair-parklive-evidence/local-TestParkLifecycleLive-count5.log, SHA256 af9d32b462c4e5068e00b910277ce90a5d070ee78126b1299f74299800fcfaf5.

The refusal is `ScopedThreadArtifactCollisionChecker.TriggerBoundQuit` (cmd/internal/couchcore/artifactcollision.go): the observed binding's Owner is nil, or its owner DataDir/RepoScope/Tag differs from the intent's. Not yet diagnosed. One candidate to check, not a conclusion: the owner observation and the intent may spell the data dir differently (/var/folders versus its canonical /private/var/folders), or the owner probe may not see the owner stub's process on this host. An environment-dependent difference between the operator's machine and CI is still a wrong assumption in Pair or its fixture.
