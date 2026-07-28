# Boundary Review — 000115-resurrect-a-session-across-agents#115 (milestone M3)

| field | value |
|-------|-------|
| issue | 115 — Switch the agent driving existing work |
| repo | 000115-resurrect-a-session-across-agents |
| issue file | workshop/issues/000115-resurrect-a-session-across-agents.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 68be56fbb8f43fc5b0efe78f76d02f5c5ef23afe..HEAD |
| command | sdlc milestone-close --issue 115 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-07-17T16:59:26-07:00 |
| verdict | REWORK |

## Review

> **Raw reviewer transcript trimmed.** This sidecar held the verbatim
> reviewer CLI transcript — for M4, 99,158 lines / 5.3 MB, mostly the
> echoed prompt and diff. That bulk is reconstructible from git (the diff
> is the review window) and it broke later `sdlc close` runs: the review
> dispatcher passes its prompt as argv, and these sidecars fell inside the
> next review window, pushing it past ARG_MAX (`fork/exec: argument list
> too long`). The verdict and findings — the durable part — are kept below.
> Full transcript: `git show e36c1dc~1:workshop/plans/000115-resurrect-a-session-across-agents-m3-review.md`.

## Verdict and findings
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 now delivers the crash-safe substrate: the prior draft hard-link bug is fixed with immutable copied draft backups plus digest reconciliation, queue keys are six ASCII digits at both journal and queue boundaries, README covers the new helper, and the named test/race/runtime checks pass. I found no Critical code blockers. One atlas statement still describes the old inode-retention model and should be corrected before closing the boundary.

1. Strengths:
- Pure journal/recovery ordering is isolated in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:111) and [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:359).
- Draft backup now copies immutable bytes and checks live digest before replacement in [handoff_store.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_store.go:570) and [handoff_store.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_store.go:364).
- Queue key validation is shared and strict in [queue.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/queuecmd/queue.go:117) and consumed by journal validation in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:248).
- Transcript publication uses copied inputs and one final directory rename in [handoff_transcript.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_transcript.go:39).

2. Critical findings:
- None.

3. Important findings:
- [atlas/session-identity.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/session-identity.md:122): atlas still says the transaction “retains the original draft and queue-item inodes,” which contradicts the corrected implementation and plan revision: draft history is now an immutable copied byte snapshot with digest comparison, while hard-link inode identity is used for transaction-published queue/instruction effects and retained queue items. Fix the atlas wording, and also soften [atlas/architecture.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/architecture.md:144) from broad “retained input inodes” to the corrected split. This is an ARCH-PURPOSE/Docs gate issue: the architectural restatement should not preserve the old unsafe model.

4. Minor findings:
- None.

5. Test coverage notes:
- Passed: `go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
- Passed: `go test ./cmd/internal/launcher -run 'TestOSHandOffStore' -race -count=1`
- Passed: `make -f Makefile.local test-runtimebundle && make -f Makefile.local runtimebundle-drift-check`
- Passed: `go test ./... -count=1`
- Passed: `make -f Makefile.local test-queue`

6. Architectural notes for upcoming work:
- ARCH-DRY: pass. Queue front allocation now derives from `queuecmd`, not parallel Lua logic.
- ARCH-PURE: pass. Journal/recovery planning is pure; lock/store/transcript effects sit behind runtime seams.
- ARCH-PURPOSE: pass for code after the immutable draft fix; flag only the stale atlas wording above.

7. Plan revision recommendations:
- None. The plan’s `## Revisions` entry matches the corrected code; update atlas so the broader docs match too.
164,556
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 now delivers the crash-safe substrate: the prior draft hard-link bug is fixed with immutable copied draft backups plus digest reconciliation, queue keys are six ASCII digits at both journal and queue boundaries, README covers the new helper, and the named test/race/runtime checks pass. I found no Critical code blockers. One atlas statement still describes the old inode-retention model and should be corrected before closing the boundary.

1. Strengths:
- Pure journal/recovery ordering is isolated in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:111) and [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:359).
- Draft backup now copies immutable bytes and checks live digest before replacement in [handoff_store.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_store.go:570) and [handoff_store.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_store.go:364).
- Queue key validation is shared and strict in [queue.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/queuecmd/queue.go:117) and consumed by journal validation in [handoff_state.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_state.go:248).
- Transcript publication uses copied inputs and one final directory rename in [handoff_transcript.go](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/cmd/internal/launcher/handoff_transcript.go:39).

2. Critical findings:
- None.

3. Important findings:
- [atlas/session-identity.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/session-identity.md:122): atlas still says the transaction “retains the original draft and queue-item inodes,” which contradicts the corrected implementation and plan revision: draft history is now an immutable copied byte snapshot with digest comparison, while hard-link inode identity is used for transaction-published queue/instruction effects and retained queue items. Fix the atlas wording, and also soften [atlas/architecture.md](/Users/xianxu/workspace/worktree/pair/000115-resurrect-a-session-across-agents/atlas/architecture.md:144) from broad “retained input inodes” to the corrected split. This is an ARCH-PURPOSE/Docs gate issue: the architectural restatement should not preserve the old unsafe model.

4. Minor findings:
- None.

5. Test coverage notes:
- Passed: `go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
- Passed: `go test ./cmd/internal/launcher -run 'TestOSHandOffStore' -race -count=1`
- Passed: `make -f Makefile.local test-runtimebundle && make -f Makefile.local runtimebundle-drift-check`
- Passed: `go test ./... -count=1`
- Passed: `make -f Makefile.local test-queue`

6. Architectural notes for upcoming work:
- ARCH-DRY: pass. Queue front allocation now derives from `queuecmd`, not parallel Lua logic.
- ARCH-PURE: pass. Journal/recovery planning is pure; lock/store/transcript effects sit behind runtime seams.
- ARCH-PURPOSE: pass for code after the immutable draft fix; flag only the stale atlas wording above.

7. Plan revision recommendations:
- None. The plan’s `## Revisions` entry matches the corrected code; update atlas so the broader docs match too.
