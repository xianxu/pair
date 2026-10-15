# Boundary Review — pair#568 (whole-issue close)

| field | value |
|-------|-------|
| issue | 568 — Review fresh-restore headless Neovim exits without failure diagnostics |
| repo | pair |
| issue file | workshop/issues/000568-review-fresh-restore-headless-neovim-exits-without-failure-diagnostics.md |
| boundary | whole-issue close |
| milestone | — |
| window | 03f5ba2af12069d6d6d628a34e0195a3dd05a0e9..8d114ada12f6af3cd52a27ad1aaae235f93c6dd8 |
| command | sdlc close --issue 568 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T08:32:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers bounded failure diagnostics and explicit ownership of both draft filesystem watchers. Regression tests fail against the corresponding base implementations and pass at head. No blocking findings.

Done-when: current — its clauses cover the diagnostic contract, demonstrated lifecycle repair, real RPC boundary, and exact child cleanup.

```findings
{}
```

1. **Strengths**
   - `nvim/fs_watch.lua:8` provides idempotent cleanup and invalidates queued callbacks.
   - Both watcher consumers use the shared helper while preserving their filtering and reconciliation behavior.
   - Diagnostics capture separate bounded streams, shutdown messages, phase, logs, and artifact presence before cleanup.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Watcher lifecycle tests, both diagnostic tests, artifactpath tests, and all five real restore cases passed. Temporary base-version substitutions confirmed both regressions go red. The full fixture initially failed because the review TMPDIR produced an overlong socket path; it passed with `TMPDIR="$PWD"`, still inside the reviewer directory. Tracked files remain unchanged.

6. **Architectural notes:** **ARCH-DRY: pass** — both watcher sites share one owner. **ARCH-PURE: pass** — the small IO seam receives an injected editor and has lifecycle coverage through fakes. **ARCH-PURPOSE: pass** — both enumerable watcher consumers were migrated, and the real draft/opener/RPC exercise remains intact. Atlas documents the new internal seam; no new user-facing README surface was introduced.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"26b4eb28-68a0-45e1-b094-4ff6e55f3721","repository":"github.com/xianxu/pair","issue_id":"000568","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0c6ffa15928ba205","reviewed_head":"8d114ada12f6af3cd52a27ad1aaae235f93c6dd8","slot":"pair:3","requested_at":"2026-10-15T15:29:17.910399Z"},"phase":"finished","pid":12560,"started_at":"2026-10-15T15:29:18.004512Z","finished_at":"2026-10-15T15:32:08.604119Z","session_id":"01a1402e-9991-7943-a51e-a9544a18ec08","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.589833584s
head: "8d114ada12f6af3cd52a27ad1aaae235f93c6dd8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
