# Boundary Review — pair#137 (whole-issue close)

| field | value |
|-------|-------|
| issue | 137 — Codex Return rewrite only in composer |
| repo | pair |
| issue file | workshop/issues/000137-codex-return-rewrite-only-in-composer.md |
| boundary | whole-issue close |
| milestone | — |
| window | e2a12bf4f6373d1e06fb0bc584510b4500b1701a..HEAD |
| command | sdlc close --issue 137 |
| reviewer | codex |
| timestamp | 2026-08-16 |
| final verdict | FIX-THEN-SHIP |

## Round 1 — REWORK

The first close review found the runtime shape correct but blocked the boundary
on two issues:

- **Critical:** `codexComposerTracker` was mutated from the stdout pump and read
  from stdin Return translation without synchronization. Fix: add an internal
  mutex around `resize`, `feed`, and `state`; add a race-detector regression for
  concurrent `feed`/`state`.
- **Important:** stale composer evidence survived erase-display clears. Fix:
  handle `CSI J` erase-display modes and add a regression proving `CSI 2J`
  clears active composer state.

## Round 2 — FIX-THEN-SHIP

The second close review accepted the runtime fixes and reported no Critical
findings. It required one artifact-hygiene fix before shipping:

- **Important:** the generated close-review sidecar was unbounded and included
  the raw review prompt/transcript. Fix: replace this file with a bounded durable
  summary of verdicts, findings, fixes, and evidence.

## Verification

- `go test ./cmd/internal/wrapcmd -run 'TestCodexComposer|Test.*PlainEnter|TestTranslateChunk_Codex|TestEmitPlainCR|TestHandleChunk_CodexFeedsComposerTracker' -count=1`
- `go test -race ./cmd/internal/wrapcmd -run TestCodexComposerTrackerConcurrentFeedAndState -count=1`
- `go test ./cmd/internal/wrapcmd -count=1`
- `go test ./...`
- `sdlc issue validate --issue 137`
- `git diff --check HEAD`

## Close Trailers

Review-Verdict: FIX-THEN-SHIP
Review-Window: e2a12bf..HEAD
