---
gate: boundary-review
issue: 474
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T06:17:48-07:00"
      dispatched: "2026-10-12T06:15:52-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: TestLiveReplayServerMarks moved to rec-app but still samples the prototype app's note moments (12500/14000)
          detail: replay_live_test.go:127 checks 12200-14100, which surrounded the prototype's open-modal note and its resolve. rec-app's matching note is made at 9800 and resolved at 11000 (fixturegen/notes.go:34), so the test no longer covers a note appearing with its stroke or a resolve removing the pin. It passes only because it compares against the reducer. Re-pin to moments around 9800 and 11000. No other moved test still uses the old fixture's times.
          family: repin-fixture-content-expectations
          round: 1
        - id: BR-2
          severity: Minor
          title: recorder/mirror.go:22 cites the deleted fixturegen/cdpconv.go as its reference
          detail: Reword it to point at git history. A sweep for cdpconv and fixturegen -raw found no other stale references.
          family: stale-reference-to-deleted-source
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T06:20:52-07:00"
      dispatched: "2026-10-12T06:19:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: replay_live_test.go:131-137 derives samples from Note/NoteState/Stroke/Seg records (covers 9800 note and 11000 note-state resolve); placed filter mirrors replay.js drawMarks:441-443. Live run not possible here (Chrome CDP EOF in sandbox).
          round: 2
        - id: BR-2
          disposition: addressed
          note: mirror.go:22-23 now says the source was retired in 474 and lives in git history; tree-wide grep finds no other stale cdpconv / record-mjs / fixturegen -raw references.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: d7c3d9621eebf05ac5ff680b63bab22379fe0b50
      blocked: false
---

# Gate ledger — pair#474 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T06:17:48-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `repin-fixture-content-expectations` TestLiveReplayServerMarks moved to rec-app but still samples the prototype app's note moments (12500/14000)
  replay_live_test.go:127 checks 12200-14100, which surrounded the prototype's open-modal note and its resolve. rec-app's matching note is made at 9800 and resolved at 11000 (fixturegen/notes.go:34), so the test no longer covers a note appearing with its stroke or a resolve removing the pin. It passes only because it compares against the reducer. Re-pin to moments around 9800 and 11000. No other moved test still uses the old fixture's times.
- **BR-2** [Minor] `stale-reference-to-deleted-source` recorder/mirror.go:22 cites the deleted fixturegen/cdpconv.go as its reference
  Reword it to point at git history. A sweep for cdpconv and fixturegen -raw found no other stale references.

## Round 2 — 2026-10-12T06:20:52-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — replay_live_test.go:131-137 derives samples from Note/NoteState/Stroke/Seg records (covers 9800 note and 11000 note-state resolve); placed filter mirrors replay.js drawMarks:441-443. Live run not possible here (Chrome CDP EOF in sandbox).
- BR-2 — addressed — mirror.go:22-23 now says the source was retired in 474 and lives in git history; tree-wide grep finds no other stale cdpconv / record-mjs / fixturegen -raw references.

## Open findings

(none — every finding has been disposed)
