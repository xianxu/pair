---
gate: boundary-review
issue: 499
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T15:40:25-07:00"
      dispatched: "2026-10-12T15:39:14-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Note dialog Save/Resolve stay active while paused; README claims resolve is disabled
          detail: 'annotate.js closes the dialog only on stop:end (line 531), and the pin callback (486-490) still opens it with resolvable:true during a pause. Instances: Save on an open draft, Resolve on an open view, Resolve on a pin opened while paused. README.md:1044 says resolve is disabled via the greyed rail, but the rail has no resolve. Server still refuses (409). Fix: close the dialog or disable save/resolve while paused(), use resolvable:!paused(), and extend the live Chrome test.'
          family: paused-disables-every-viewer-action
          round: 1
        - id: BR-2
          severity: Minor
          title: Stale comments and stale-page sentence for the paused writer drop
          detail: recorder.go:515 ("The viewer's document has gone") and board/writer.go Emit doc ("isn't current any more") now also cover the paused case. A Writer drop during a pause returns ErrStale, so the viewer sees "The page changed. Try again." instead of "The cast is paused."
          family: doc-claims-track-code
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T16:05:53-07:00"
      dispatched: "2026-10-12T16:03:03-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: noteDialog.hold (collab.js:274) is called from renderRail on every stop/seg; held Save/Resolve verified via node fakes (no post, draft text kept, release re-enables); pin view while held shows Resolve disabled. Regression evidence is only the opt-in live Chrome test (see new finding).
          round: 2
        - id: BR-2
          disposition: addressed
          note: recorder.go:515 comment names the pause; writer.go Emit doc updated and returns ErrPaused for a pause drop, pinned by TestWriterRefusesARecordDuringAPause (ErrPaused paused, ErrStale after navigate).
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: noteDialog.hold has no default-run regression test; only the opt-in WEBCAST_LIVE_CHROME test covers BR-1
          detail: 'tests/broadcast-viewer/notes.test.mjs already drives noteDialog with fake elements and runs by default. Add hold() cases there: a held draft disables Save, shows the reason and keeps the text, and release re-enables it; view() while held disables Resolve and a click posts nothing; release clears the message. Instances: hold() at collab.js:274, the held branches in draft()/view() at collab.js:246 and 260, and the resolve click guard at collab.js:229. The live test could not launch Chrome in this environment (CDP EOF), so nothing that runs by default protects the fix.'
          family: regression-test-runs-by-default
          round: 2
        - id: BR-4
          severity: Minor
          title: Truncated held comment in collab.js; annotate.js renderRail comment is stale and separated from its function
          detail: 'This is the 2nd finding in family doc-claims-track-code. Rule: when a diff changes what a function does, re-read and update every comment that sits on or names that code in the same commit. Instances in this window: collab.js:180 ("why Save and Resolve are off, or " is cut off); annotate.js:354 ("rendered only when their state changes (a pick, a mode)" now also changes on pause/resume, and the new paused() comment and const sit between it and renderRail).'
          family: doc-claims-track-code
          round: 2
        - id: BR-5
          severity: Minor
          title: 'ARCH-DRY: the paused predicate is restated in writer.go, board.go, annotate.js and viewer.js'
          detail: writer.go:42 (Seg>0 && !Live && !Ended && DocSeg==doc), board.go:284 (implied by switch order), annotate.js:359 (seg>0 && !live && !ended), viewer.js:144 (page && !live). Add a recording.State.Paused() method for the Go sites, and one JS helper (e.g. in model.js) for the viewer sites.
          family: state-predicate-single-source
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T16:45:06-07:00"
      dispatched: "2026-10-12T16:43:11-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: notes.test.mjs hold test runs by default; reverting each of the 3 guards in collab.js (Save rule, Resolve click guard, resolve.disabled) turns it red (14 pass / 1 fail).
          round: 3
        - id: BR-4
          disposition: addressed
          note: collab.js:56 held comment is complete; annotate.js renderRail comment names pause/resume and directly precedes renderRail, with paused() moved above it.
          round: 3
        - id: BR-5
          disposition: addressed
          note: recording.State.Paused() used by board.go:284 and writer.go:42; Model.paused used by annotate.js; viewer.js view() is pre-existing show-state shared with replay, kept with a stated reason.
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 366152fc9adb5d7f317f19861053053147f3038f
      blocked: false
---

# Gate ledger — pair#499 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T15:40:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `paused-disables-every-viewer-action` Note dialog Save/Resolve stay active while paused; README claims resolve is disabled
  annotate.js closes the dialog only on stop:end (line 531), and the pin callback (486-490) still opens it with resolvable:true during a pause. Instances: Save on an open draft, Resolve on an open view, Resolve on a pin opened while paused. README.md:1044 says resolve is disabled via the greyed rail, but the rail has no resolve. Server still refuses (409). Fix: close the dialog or disable save/resolve while paused(), use resolvable:!paused(), and extend the live Chrome test.
- **BR-2** [Minor] `doc-claims-track-code` Stale comments and stale-page sentence for the paused writer drop
  recorder.go:515 ("The viewer's document has gone") and board/writer.go Emit doc ("isn't current any more") now also cover the paused case. A Writer drop during a pause returns ErrStale, so the viewer sees "The page changed. Try again." instead of "The cast is paused."

## Round 2 — 2026-10-12T16:05:53-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — noteDialog.hold (collab.js:274) is called from renderRail on every stop/seg; held Save/Resolve verified via node fakes (no post, draft text kept, release re-enables); pin view while held shows Resolve disabled. Regression evidence is only the opt-in live Chrome test (see new finding).
- BR-2 — addressed — recorder.go:515 comment names the pause; writer.go Emit doc updated and returns ErrPaused for a pause drop, pinned by TestWriterRefusesARecordDuringAPause (ErrPaused paused, ErrStale after navigate).

### Raised

- **BR-3** [Important] `regression-test-runs-by-default` noteDialog.hold has no default-run regression test; only the opt-in WEBCAST_LIVE_CHROME test covers BR-1
  tests/broadcast-viewer/notes.test.mjs already drives noteDialog with fake elements and runs by default. Add hold() cases there: a held draft disables Save, shows the reason and keeps the text, and release re-enables it; view() while held disables Resolve and a click posts nothing; release clears the message. Instances: hold() at collab.js:274, the held branches in draft()/view() at collab.js:246 and 260, and the resolve click guard at collab.js:229. The live test could not launch Chrome in this environment (CDP EOF), so nothing that runs by default protects the fix.
- **BR-4** [Minor] `doc-claims-track-code` Truncated held comment in collab.js; annotate.js renderRail comment is stale and separated from its function
  This is the 2nd finding in family doc-claims-track-code. Rule: when a diff changes what a function does, re-read and update every comment that sits on or names that code in the same commit. Instances in this window: collab.js:180 ("why Save and Resolve are off, or " is cut off); annotate.js:354 ("rendered only when their state changes (a pick, a mode)" now also changes on pause/resume, and the new paused() comment and const sit between it and renderRail).
- **BR-5** [Minor] `state-predicate-single-source` ARCH-DRY: the paused predicate is restated in writer.go, board.go, annotate.js and viewer.js
  writer.go:42 (Seg>0 && !Live && !Ended && DocSeg==doc), board.go:284 (implied by switch order), annotate.js:359 (seg>0 && !live && !ended), viewer.js:144 (page && !live). Add a recording.State.Paused() method for the Go sites, and one JS helper (e.g. in model.js) for the viewer sites.

## Round 3 — 2026-10-12T16:45:06-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — notes.test.mjs hold test runs by default; reverting each of the 3 guards in collab.js (Save rule, Resolve click guard, resolve.disabled) turns it red (14 pass / 1 fail).
- BR-4 — addressed — collab.js:56 held comment is complete; annotate.js renderRail comment names pause/resume and directly precedes renderRail, with paused() moved above it.
- BR-5 — addressed — recording.State.Paused() used by board.go:284 and writer.go:42; Model.paused used by annotate.js; viewer.js view() is pre-existing show-state shared with replay, kept with a stated reason.

## Open findings

(none — every finding has been disposed)
