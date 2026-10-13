# Boundary Review — pair#506 (whole-issue close)

| field | value |
|-------|-------|
| issue | 506 — broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽) |
| repo | pair |
| issue file | workshop/issues/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e14f2ae729a0181259101c65e20da79f4d1bd2..ddb76d21002827e710be0ffbb69713b9ab9c809e |
| command | sdlc close --issue 506 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T22:37:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Prior findings mostly check out. I found a stale README table row and am now running the tests, plus a revert check for BR-3.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The three-link model works and is enforced on the server. `castLink`/`linkOf` (`cmd/internal/webcast/link.go`) is the one place that decides what each link may do, and every route asks it. A board POST under the view-only link gets a 403 before admission. `palette.json` tells the viewer whether its link may annotate, and the viewer reads that answer defensively. Couch's three copy commands now share one helper. Four of the five prior findings are fixed with evidence. I reverted the BR-3 fix in this checkout, `TestCastLinksClearTogetherWhenTheMountIsLost` went red, and it passed again once restored. Two things are left before shipping:
- **README table:** the row at `README.md:1117` still says the web `👆` copies the view link. That is user-facing and wrong.
- **Old name in production comments:** "view link" still appears in comments in `cast.go`, so BR-8's sweep was incomplete.

The `webcast` package tests and `node --test tests/webcast/annotate.test.mjs` both pass.

1. **Strengths**
   - `link.go:22-44`: the ordered `castLink` enum, with `annotates()`/`drives()` as `>=`/`==` checks, means a single value answers every grant question. `Owns` becomes `linkOf != linkNone` (`server.go:1027`), which removed the old `drivesUnder` duplicate.
   - `server_board.go:106-111`: the view-only refusal comes before `admit`. The test sends an over-limit body and checks for 403, not 413 (`server_board_test.go:1225`), which proves the ordering.
   - `annotate.js:264` `readPalette` parses `palette.json` into a typed value at the boundary. An older server's bare array reads as "this cast takes no notes" instead of crashing, and node tests cover the bad shapes.
   - `console_webcast.go:359-437`: the `webcastLink` table plus `copyWebcastLinkOf` replaces three copy functions. It reads one snapshot under the lock, and its notices never contain the link.
   - `viewer_annotate_live_test.go:1293-1312`: a headless end-to-end check that a view-only viewer sees B's note, has no rail, opens the note without Resolve, and gets "403 This link is view-only." when it posts.

2. **Critical findings:** none.

3. **Important findings**
   - **`README.md:1117`: the cast-row table still reads `` `👆` (web) | copy view link | copy view link ``.** It should say "copy pointer link" in both columns. The row came in with the #495 merge (`7401b75f7`), but at head it contradicts the README's own new paragraph at lines 1004-1034. This is the **3rd finding in family `one-name-per-surface`**, so the fix is a rule, not this one row: when a surface's meaning changes, the rename sweep covers README.md (tables included), atlas, production comments and test comments, and it is re-run after every merge from main. The lesson the team added in `workshop/lessons.md` already names README.md, but its grep was not re-run after the #495 merge.

4. **Minor findings**
   - **BR-8 is not fully done.** These sites still use the retired name:
     - `cmd/internal/webcast/cast.go:159` ("The view link's lock") and `cast.go:162` ("locked like the view link") are production code.
     - Test comments are stale too: `server_drive_test.go:297` and `viewer_drive_live_test.go:216` say "the view link watches, points and notes", which is now false. `lock_helpers_test.go:24` and `server_board_test.go:290` also still say "view link".
   - **`lock_helpers_test.go:26-32`: `unlockCast` maps every non-drive code to `u.Link`.** On a recorded cast it would post the pointer code at the view-only link. Today it is only called on unrecorded casts, so it is a latent trap. Couch's `unlockWebCast` was fixed with a `linkOf` map, so the same link-kind → URL mapping is now written out three times: two test helpers plus the `url` closures in the `webcastLink` table. This is the **3rd finding in family `shared-helper-for-repeated-construction`**. The rule: one accessor on `Update`, e.g. `func (u Update) LinkFor(kind string) string`, maps a `broadcast.Link*` kind to its URL, and every caller (copy commands, both unlock helpers) uses it.
   - **`workshop/lessons.md:1439`: the new `## Re-meaning a named surface` heading was inserted in the middle of an existing section.** The #513 and #516 bullets that follow now sit under the #506 heading. Move the new section after them.

5. **Test coverage notes**
   - The BR-3 regression test is real and fails without the fix (checked by reverting).
   - Grants, the `linkOf` table, the 403 before admission, the palette answer, the pointer stream counting as a view stream, three distinct locks and codes, Couch's copy notices, and the 👆/👽 spans are all tested. The view-only viewer is tested end to end in headless Chrome.
   - There is no test that the drive link may also annotate on the board. `annotates()` covers it through the grant table, so this is low risk.

6. **Architectural notes**
   - **ARCH-DRY:** flag (Minor). The link → URL mapping is written out three times (above).
   - **ARCH-PURE:** pass. `castLink`, `readPalette` and `linkURL` are pure and tested without IO.
   - **ARCH-PURPOSE:** pass. All three links exist, the server enforces what each grants, and the viewer and Couch both use them. Only the smoke test is deferred to close, as the plan says.
   - **ARCH-MOCK:** pass. Tests use FakeTunnel, FakeBrowser and httptest; nothing new calls an external service.
   - **ARCH-CONSTRAINTS:** pass. The view-only refusal spends no body read, in-flight slot or rate budget.
   - **ARCH-SECURE:** pass. Tokens are compared in constant time and an empty token matches nothing. Notices never contain a link. `palette.json` is parsed at the boundary. The `annotate` flag in the viewer is only cosmetic, because the server enforces the grant.
   - **ARCH-ORDER:** pass. The links are fixed for the cast's life, and when the mount is lost all three clear in one assignment.
   - **ARCH-FUNERAL:** pass. The pointer token and its lock live in memory and end with the mount, alongside the existing locks.

7. **Plan revision recommendations:** none. Tasks 1-9 match the code. Task 10's operator smoke test is still unchecked, which is correct at close.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      console_webcast.go webcastViewOnlyLink grants "its viewers watch"; 👆 copies PointerLink; TestWebcastCopiesEachLink pins both notices.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/webcast-format.md:426 now adds "(without it under the view-only link, pair#506)".
  - id: BR-3
    disposition: addressed
    note: |
      TestCastLinksClearTogetherWhenTheMountIsLost; reverting the PointerLink clear in cast.go turns it red (verified).
  - id: BR-4
    disposition: addressed
    note: |
      linkURL(base, token) in cast.go is the one constructor of all three URLs.
  - id: BR-8
    disposition: not-addressed
    note: |
      Still says view link: cast.go:159 and cast.go:162 (production), plus test comments server_drive_test.go:297, viewer_drive_live_test.go:216 (now false), lock_helpers_test.go:24, server_board_test.go:290.
findings:
  - id: new
    severity: Important
    family: one-name-per-surface
    title: |
      README.md:1117 cast-row table still says the web 👆 copies the view link
    detail: |
      3rd finding in this family. The row should read "copy pointer link" in both columns; it arrived with the 495 merge and contradicts README's own paragraph at lines 1004-1034. Rule: a rename sweep covers README.md tables, atlas, production comments and test comments, and is re-run after every merge from main.
  - id: new
    severity: Minor
    family: shared-helper-for-repeated-construction
    title: |
      unlockCast maps a pointer code to the view-only URL; link kind to URL is restated three times
    detail: |
      3rd finding in this family. lock_helpers_test.go:26-32 would post the pointer code at u.Link on a recorded cast (latent trap; today's callers are unrecorded). Rule: one Update.LinkFor(kind) accessor used by the webcastLink table, unlockCast and unlockWebCast.
  - id: new
    severity: Minor
    family: insert-preserves-section-structure
    title: |
      lessons.md: the new pair 506 section is inserted mid-section, so the 513 and 516 bullets now sit under it
    detail: |
      workshop/lessons.md:1439; move the new heading and its two bullets after the 516 bullet.
```

---

## Re-review — 2026-10-12T22:43:15-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 506 — broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽) |
| repo | pair |
| issue file | workshop/issues/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e14f2ae729a0181259101c65e20da79f4d1bd2..84a3d0cc1ed337e7530e746d77a2526697ab9512 |
| command | sdlc close --issue 506 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T22:43:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on last commit for prior findings.

Running tests; the full build hits a generated-asset embed missing in this fresh checkout, so I'm testing the packages one at a time.

All tests pass. Now I'm spot-checking how the server enforces the board refusal.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Three of the four open findings are fixed: BR-9, BR-10 and BR-11. One small piece of BR-8 is still open. The feature works as the plan says. Each link type has a fixed level in `webcast/link.go`. `ServeHTTP` works out the link type once per request. The board refuses view-only posts before reading the body (`server_board.go:106-111`), and a pointer stream counts as a view stream, not a seat. I ran the tests in this disposable checkout and they all pass: `go test` for `webcast/...`, `couchtty` and `artifactpath`, plus the node test `annotate.test.mjs`. `couchtty` and `artifactpath` first needed the gitignored runtime bundle, which I generated with `make`'s generatecmd step. The leftover BR-8 site is a web-cast test comment that the sweep's own rule covers. It is Minor and does not block the gate.

### Prior-finding dispositions
- **BR-8 (not-addressed):** All eight listed sites are now correct. `drive/controller.go:11-13` now reads "Anyone with the drive link may take a seat (the view-only and pointer links offer none)". `server.go:242` and `server_drive.go:113` are also fixed. But the finding's own grep, `git grep -nE 'view(er)? link' cmd/internal/webcast ...`, still finds `cmd/internal/webcast/viewer_drive_live_test.go:216` and `:220`. Line 216 says "The view link watches, points and notes", which this issue made false: view-only only watches. Line 220's failure message says "the view link offers driving". The other hits are fine:
  - The `couchtty/console_{broadcast,castcodes,pointer}_test.go` hits are about the terminal cast, where "view link" is still correct.
  - `atlas/webcast-format.md:702-703` is dated history and says that #506 later split the link.
- **BR-9 (addressed):** `README.md:1117` now reads "copy pointer link | copy pointer link". That matches the README's own prose.
- **BR-10 (addressed):** `Update.LinkFor` (`cast.go:76-89`) is now the only place a link type becomes a URL.
  - It is used by the `webcastLink` table (`console_webcast.go:280,315`), `unlockCast` (`lock_helpers_test.go:27`) and `unlockWebCast` (`cast_lock_helpers_test.go:53`).
  - `git grep '\.(PointerLink|DriveLink)'` finds no other production readers.
  - `TestUpdateLinkFor` checks the mapping, including that an unknown type gives "".
  - The change doesn't alter behaviour for today's callers, so a unit test of the mapping is enough evidence.
- **BR-11 (addressed):** The pair#506 section in `workshop/lessons.md` now sits after the #516 bullet (line 1448ff). The #513 and #516 bullets are back under their own section. The removal left one blank line before the #513 bullet, which is harmless.

### 1. Strengths
- `webcast/link.go`: each link's grants come from one ordered type (`annotates()`, `drives()`). `Owns`, the board routes, the drive routes and the event stream all derive from `linkOf`, which replaced the old `drivesUnder` side path.
- `server_board.go:106-111`: the view-only refusal comes before admission, so a view-only client can't use up the body read, an in-flight slot or the rate budget. `TestBoardRoutesRefuseTheViewOnlyLink` covers this, including an oversized body.
- `TestCastLinksClearTogetherWhenTheMountIsLost` checks that all three links clear together when the tunnel drops, so Couch can't copy a dead link.
- The new rule in `lessons.md` covers the whole class: re-run the old-term grep after every merge from main. That is exactly how BR-9 got in.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- BR-8 leftover: `viewer_drive_live_test.go:216,220` should read "The view-only link only watches: no seat, no drive button." and "the view-only link offers driving".

### 5. Test coverage notes
- Server enforcement (403 for view-only board posts, 404 for drive routes outside the drive link, pointer streams holding no seat) is tested at the HTTP seam with real tokens and locks.
- The `LinkFor` table test is pure.
- The JS tests cover the viewer mounting notes only under a link that can annotate.

### 6. Architectural notes
- **ARCH-DRY: pass.** One `LinkFor`, one `copyWebcastLinkOf`, one `linkURL`.
- **ARCH-PURE: pass.** Link grants are pure methods, and `linkOf` is a pure classifier over the tokens.
- **ARCH-PURPOSE: pass.** Every consumer derives from `castLink` and `LinkFor`; the shadow sweep found no hand-kept restatement in production code.
- **ARCH-MOCK: pass.** Tests use `broadcast.FakeTunnel` and the existing cast fixtures behind the same seam.
- **ARCH-CONSTRAINTS: pass.** Refusing before admission keeps view-only clients outside the rate and in-flight limits.
- **ARCH-SECURE: pass.** Tokens are compared in constant time (`OwnsToken`), and a link that wasn't made has an empty token that matches nothing. The pointer token gets its own lock (`locks.Mint`). A recorded cast doesn't start if it fails to make its pointer token.
- **ARCH-ORDER: pass.** The link type is fixed for the cast's life. The state that changes, the mount, clears all three links together, and that is tested.
- **ARCH-FUNERAL: pass.** The pointer token and its lock live as long as the cast and go with the Server. Nothing new is written to disk.

### 7. Plan revision recommendations
None. The plan still matches the code.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      All eight named sites fixed, but the rule's own grep still hits cmd/internal/webcast/viewer_drive_live_test.go:216 ("The view link watches, points and notes", now false) and :220; reword both to "view-only link".
  - id: BR-9
    disposition: addressed
    note: |
      README.md:1117 now reads "copy pointer link" in both columns, matching the prose.
  - id: BR-10
    disposition: addressed
    note: |
      Update.LinkFor (cast.go:76) is used by the webcastLink table, unlockCast and unlockWebCast; TestUpdateLinkFor pins the mapping; no other production PointerLink/DriveLink readers.
  - id: BR-11
    disposition: addressed
    note: |
      The pair 506 section now follows the 516 bullet (lessons.md:1448ff); 513/516 are back in their own section.
```
