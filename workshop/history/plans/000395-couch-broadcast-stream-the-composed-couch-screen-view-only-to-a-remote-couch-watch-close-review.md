# Boundary Review — pair#395 (whole-issue close)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | whole-issue close |
| milestone | — |
| window | fcce4b21c9cbfc3c6203df98259d34c1d4c5f191..5cf35123197c52bdbe405a95e94edc9555109d5a |
| command | sdlc close --issue 395 |
| reviewer | claude |
| timestamp | 2026-10-08T00:22:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The latest round settles every open finding, and I found nothing new. HEAD (5cf35123) puts record rewrites under the records lock and removes leftover `*.json.tmp` files when reaping. A hook (`afterRecordRead`) now makes the BR-22 claim race deterministic. To check it, I removed the lock in a copy taken from `git archive HEAD`. `TestRunRecordsConcurrentClaimHasOneWinner` then failed on all three runs ("2 claimers took one tunnel"), and passes with the lock in place. I checked the earlier advisories (BR-10, BR-16–BR-20) against the code and the issue file. All are fixed.

1. **What works**
   - The race test has a real ordering seam. Using an atomic first-caller flag instead of `sync.Once` keeps the race visible, and lessons.md records that trap (`cloudflared_test.go:443-470`).
   - Reaping `.tmp` files is safe: every rewrite holds the directory flock (`records.go:148`), and `record()` is only called outside the lock (`cloudflared.go:186`), so it can't deadlock on itself.
   - The two colour parsers are kept on purpose and justified in a comment, with `TestParseOSC4ReplyAgreesWithXParseColor` as the check that they agree.
   - BR-11's ordering is now tested deterministically (`console_broadcast_test.go:403`), not only by the race detector.
2. **Critical findings:** none.
3. **Important findings:** none.
4. **Minor findings:** none new.
5. **Test coverage:** the BR-22 race test fails when the lock is removed (checked). Removing the `.tmp` glob would make `TestReapRemovesLeftoverTempRecords` fail. The input-decoder test row now feeds the real `UnknownOscEvent`.
6. **Architecture:** I re-checked only the rounds' delta.
   - **ARCH-ORDER:** passes. The ordering hook is a test seam.
   - **ARCH-FUNERAL:** passes. `.tmp` files now have a removal path.
   - **ARCH-DRY:** passes. The second parser is justified and tested.
   - **ARCH-SECURE:** passes. Reaping removes only `*.json.tmp` in its own directory.
   - **ARCH-PURE, ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-PURPOSE:** unchanged from earlier rounds and pass.
7. **Plan revisions needed:** none.

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      atlas/broadcast.md, the plan and lessons.md no longer mention TestViewerFit (grep finds nothing).
  - id: BR-16
    disposition: addressed
    note: |
      tunnel.go:23-25 now says callers may cancel ctx once Start returns (Couch does).
  - id: BR-17
    disposition: addressed
    note: |
      console_broadcast_test.go:403 drives abandon-before-adopt deterministically.
  - id: BR-18
    disposition: addressed
    note: |
      The issue's M4 Plan row now says the font-file setting was dropped. Line 354 is the historical Revisions entry, so it can stay.
  - id: BR-19
    disposition: addressed
    note: |
      console_palette.go:104-108 explains why it keeps its own strict parser, and TestParseOSC4ReplyAgreesWithXParseColor pins that the two agree.
  - id: BR-20
    disposition: addressed
    note: |
      The input_test.go:99 row now expects the full uv.UnknownOscEvent from the real decoder.
  - id: BR-27
    disposition: addressed
    note: |
      The afterRecordRead hook makes the race deterministic. With the lock removed in a scratch copy, the test failed 3 of 3 runs. lessons.md is corrected.
  - id: BR-28
    disposition: addressed
    note: |
      Rewrites now hold the lock. reapLocked removes *.json.tmp, pinned by TestReapRemovesLeftoverTempRecords.
```
