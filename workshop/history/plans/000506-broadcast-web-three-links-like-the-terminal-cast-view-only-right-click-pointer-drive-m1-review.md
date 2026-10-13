# Boundary Review — pair#506 (milestone M1)

| field | value |
|-------|-------|
| issue | 506 — broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽) |
| repo | pair |
| issue file | workshop/issues/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 5a533d455b12f31e879dc526e6ac832d779c3be3..e6ecba8e381805ad9c0558c6b5f3e2a78ff6b2dc |
| command | sdlc milestone-close --issue 506 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:37:45-07:00 |
| verdict | SHIP |

## Review

Reading the rest of the diff's consumers now: the Couch callers of `Update.Link`, and how `annotate.js` enters a mode.

Webcast tests and node tests pass. `couchtty` won't build in this checkout because a generated runtime bundle is missing. Next I'll undo the board refusal to confirm a test catches it.

```verdict
verdict: SHIP
confidence: high
```

M1 delivers the cast side the revised plan says it does, and I found nothing that blocks SHIP. A cast now has three tokens. `castLink`/`linkOf` decides what each request may do. The board refuses the view link before it reads the body. `palette.json` returns `{palette, annotate}`, and the viewer reads that answer and mounts read-only under a view link. I ran `go vet` and `go test ./cmd/internal/webcast/...`: both green. The node tests pass (10/10). I made two changes by hand to check the tests catch real bugs: turning off the board refusal fails `TestBoardRoutesRefuseTheViewOnlyLink` (the post gets 413 instead of 403, and a note reaches the board), and widening `drives()` fails `TestPointerStreamCountsAsView`. `couchtty` would not build in this checkout because the generated `runtimebundle` assets are missing. That is a problem with this environment, not with the diff, but it means I could not run `console_webnotes_test.go`. I also did not run the headless-Chrome live test, which needs `WEBCAST_LIVE_CHROME=1`. The findings below are all Minor, and most are work M2 already plans.

**1. Strengths**
- `cmd/internal/webcast/link.go:10-17`: two scattered checks (`Owns` and `drivesUnder`) become one ordered enum with grant methods, and every route now asks it. `ServeHTTP` works out `link` once (`server.go:328`) and passes it down.
- `server_board.go:104-108`: the refusal comes after the route is recognised and the board is known to exist, but before `admit`. The over-limit body getting 403 rather than 413 proves that order (`server_board_test.go:500`).
- `readPalette` (`annotate.js:272`) turns untrusted JSON into a typed value at the edge. Its test covers the bare array an older server sent and other malformed shapes (ARCH-SECURE).
- `StartCast` refuses to start a recorded cast whose pointer token can't be made, instead of starting a weaker one (`cast.go:146-151`).
- The read-only mount keeps one code path: the rail is always hidden and `resolvable: annotate`, so `renderRail` and `placeRail` don't fork.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- `cmd/internal/couchtty/console_webcast.go:263`: right-click still says "view link copied (watch, point, notes)", but `u.Link` now only watches. At M1, Couch also has no way to hand out the pointer link. M2 Task 8 fixes this; until then, an M1 build tells the operator the wrong thing.
- `atlas/webcast-format.md:431` still says "The 💭 chip … opens the note with Resolve". Under the view link it now opens without Resolve. This sentence should be qualified or point to the new paragraph in `atlas/webcast.md`.
- Plan Task 4 asks for a test that all three links clear when the mount is lost. `TestCastPublishesThreeLinks` (`link_test.go:276`) doesn't test that, and `cast.go:277` has no regression test for it.
- `cast.go:268-273` builds the link URL `Base()+"/"+tok+"/"` three times. A small `linkURL(base, tok)` helper would make one source (ARCH-DRY, nit).
- README has no paragraph on the three links yet. That is fine at M1, since README is in M2 Task 10.

**5. Test coverage notes**
- Covered:
  - a grant table for each link;
  - `linkOf` and `Owns` for each token, including the empty token;
  - a pointer-link stream counted as a view viewer;
  - each board route refused under the view link, with a normal body and an over-limit one;
  - the palette answer for each link;
  - the node table for `readPalette`;
  - live viewer C: no rail, Resolve hidden, fetch refused with 403.
- Not run here: the live headless-Chrome test, and `couchtty` (environment).
- The `PointerLink` clearing on mount loss has no test (above).

**6. Architectural notes**
- **ARCH-DRY: pass.** One enum replaces two predicates. The only repetition is the link-URL nit above.
- **ARCH-PURE: pass.** `castLink`, its grants and `readPalette` are pure and tested without IO. `linkOf` is a thin method over the tokens.
- **ARCH-PURPOSE: pass for M1.** Every server consumer derives from `castLink`: board, palette, events, drive and `Owns`. The Couch side is planned M2 scope, not a deferral of the point of the issue.
- **ARCH-MOCK: pass.** No new external dependency; it uses the existing fakes and the Chrome harness.
- **ARCH-CONSTRAINTS: pass.** The view link is refused before it spends a body read, an in-flight slot or rate budget.
- **ARCH-SECURE: pass.**
  - Tokens are compared in constant time, and an empty token never matches.
  - The refusal text is fixed.
  - The palette answer is parsed into a type.
  - `Update` comments say links stay out of notices.
- **ARCH-ORDER: pass.** `annotate` is fixed when the viewer mounts, and the link kind is decided per request. No state is carried across events.
- **ARCH-FUNERAL: pass.** The pointer token lives in memory and dies with the Cast.
- For M2: change the right-click notice and the `Shares()` and `ShareWebView` wording in the same change, since view and pointer viewers are now counted together.

**7. Plan revision recommendations:** none needed. The M1 Revisions entry already records the viewer work moving into M1. Optionally, add a line that the "links clear on lost" check from Task 4 is still open.

```findings
findings:
  - id: new
    severity: Minor
    family: couch-notice-matches-link-grants
    title: |
      Right-click notice still says the copied link points and takes notes; at M1 it is view-only
    detail: |
      cmd/internal/couchtty/console_webcast.go:263 copies u.Link (now view-only) with the text "(watch, point, notes)". M2 Task 8 must change it; there is also no Couch path to the pointer link until then.
  - id: new
    severity: Minor
    family: atlas-tracks-surface-change
    title: |
      atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
    detail: |
      Under the view link Resolve is hidden (annotate.js:505); qualify the sentence or point to the new paragraph in atlas/webcast.md.
  - id: new
    severity: Minor
    family: plan-step-test-delivered
    title: |
      Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
    detail: |
      TestCastPublishesThreeLinks only checks publication and that a cast without a board has no pointer link; cast.go:277 clearing PointerLink has no regression test.
  - id: new
    severity: Minor
    family: shared-helper-for-repeated-construction
    title: |
      The link URL Base()+"/"+token+"/" is built three times in Cast.run
    detail: |
      cast.go:268-273; a small linkURL(base, token) helper would be the single source (ARCH-DRY nit).
```
