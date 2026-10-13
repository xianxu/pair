# Boundary Review — pair#495 (milestone M2)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | f94b8f6defdb0ee361ac8829d29b9f21ed6f1749..ef9ca9ceffddce4bb29068e7e8bb6cb922163f66 |
| command | sdlc milestone-close --issue 495 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:13:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 code diff now; checking the webcast server lock wiring and running tests next.

Running the affected Go and node test suites now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 delivers what it claims. The web cast's view and drive links are each locked at the Host gate. The drive code is 6 digits and separate from the view code. The viewer name is one rule shared by Go and JS. Terminal notes carry an author. The pages switched to `credentials: 'same-origin'`. A closed web-cast stream probes `session` and goes back to the lock page. All of this is in the diff and covered by tests. Go tests pass for `broadcast`, `castlock` and every `webcast` package, and all 166 node tests pass. The `couchtty` tests could not run in this checkout: `runtimebundle` embeds a missing `assets/runtime/files`, so that is an environment setup failure, not a code failure. Nothing blocks SHIP. There are two things to fix:
- **Untested reload branch:** the web viewer's new "reload into the lock page" branch only runs in a Chrome test that is off by default, so normal test runs don't check it.
- **Plan drift (BR-1):** the plan body is still out of date, and its own new "Current design" summary has already fallen behind the latest revision.

**1. Strengths**
- **One rule for names.** `broadcast.ViewerName` (`cmd/internal/broadcast/notes.go:59`) replaces the board's private `nameText`. `board.MaxNameRunes` now comes from it, and `TestNoteAuthorRoundTripsOnTheBoardRule` checks the two bounds stay equal. On the JS side, `validName`/`storedName` moved into `collab.js`, and `annotate.js`, `lock.js` and the terminal `viewer.js` all import them.
- **Safe lock lifecycle in `StartCast`.** The view lock is minted before anything else (`cast.go:120`), so a failure there leaves nothing to undo. The drive lock is minted together with its token, and a failure clears the token, so there is never a drive link without a lock. `endLocks` runs after `Unmount`, so a viewer's stream is not cut before the cast's end reaches it.
- **Gate tests are a real table.** `TestWebCastLinksRefuseEveryRouteWithoutASession` requests every route, plus every embedded asset, on both tokens. `TestWebCastDriveCodeIsSeparate` proves the view code does not open the drive link and a view session is not a drive session.
- **The Couch notes menu shows `name: text`, but a click still copies the plain text.** The copy reads the board's note, not the menu's (`noteTextLocked`, `console_notes.go:316`), and a test checks the board's note is unchanged.
- **The sandboxed iframe sends the session cookie.** `TestLiveLockPage` checks in real Chrome that the recorded `assets/` load with status 200 under the session.

**2. Critical:** none.

**3. Important**
- **The web viewer's new reload path has no default test** (`cmd/internal/webcast/web/viewer.js:99-106`). Nothing in `tests/webcast/viewer.test.mjs` exercises the new `onerror` path (stream closed → `sessionLost` → `relock` or "Disconnected"). The terminal viewer's equivalent is tested in `tests/broadcast-viewer/connect.test.mjs`. The only coverage is `TestLiveLockPage`, which runs only with `WEBCAST_LIVE_CHROME=1`. If `lost`/`!lost` were swapped, or the `ended` check were dropped, normal test runs would still pass.
  - Fix: reuse the fake EventSource already in `viewer.test.mjs`. Set `readyState = 2`, inject `sessionLost` resolving true and then false, and assert `relock` is called in the first case and `status: 'Disconnected'` in the second. Also assert that neither is called after an `end` event.

**4. Minor**
- **BR-1 is still open** (family `spec-moves-with-decisions`; see the disposition below).
- **README not yet updated.** README §Pointing and notes (line 1021) still says "Anyone with the viewer link can point". This is acceptable while the operator has no way to see a code (that arrives in M3), but the M3 close should update the README with the lock, the codes and the drive code.
- **`lockAssets` now serves `collab.js` before a session** (`lockgate.go:37`). It is harmless (pure code, no data), but `atlas/broadcast.md:82` and the plan's Current design still list only `lock.js`, `lock.css` and `POST unlock` as pre-session.
- **The JS name bound is a second copy of the Go constant.** `MAX_VIEWER_NAME = 40` in `collab.js` repeats `MaxViewerNameRunes` with nothing checking they agree. The comment points at the Go side, which is acceptable across languages, but a one-line node test of the literal against a Go-generated value would close the gap (ARCH-DRY).
- **Duplicated `name: text` formatting.** `withAuthors` (`console_notes.go:87`) and the web-notes item builder (`console_notes.go:524-526`) each build `name: text`. A shared `authored(name, text)` helper would make one copy.
- **The driven-cast path of `StartCast` is untested.** `TestCastCodesAndRotation` uses an undriven cast. No test covers a driven cast's two codes from `StartCast`, or the path where the drive lock fails and `driveOff` is set.

**5. Test coverage notes**
- **Plan item not done as written.** The plan's M2 item "Headless-Chrome live tests (`viewer_live_test`, `viewer_drive_live_test`, `viewer_annotate_live_test`) updated to unlock first" was not done as written. Those tests use `httptest.NewServer(viewer)` directly and never go through the Host gate, so they stay unlocked. The new `TestLiveLockPage` replaces the intent. That is a sound substitution, but the plan should say so (folded into BR-1).
- The package's own tests use a dedicated cookie-jar client (`castClient`) and leave `http.DefaultClient` alone, so there is no new global-state problem (family `test-global-state` is not repeated).

**6. Architecture**
- **ARCH-DRY: flag (Minor).** The name rule is consolidated well; the remaining duplicates are the two Minor items above.
- **ARCH-PURE: pass.** `ViewerName`, `withAuthors` and `validName`/`storedName` are pure and unit-tested. The cast wires the locks in its thin shell.
- **ARCH-PURPOSE: pass for M2.**
  - Every web cast link that can carry data is gated. Replay stays unlocked, as decided in D4.
  - Shadow-sweep of `credentials: 'omit'`: none remain in either `web/` directory.
  - Name consumers: the lock page, both viewers' note dialogs, the board and terminal notes all derive from the one rule.
- **ARCH-MOCK: pass.** No new external dependency. The Chrome live test follows the existing env-gated pattern.
- **ARCH-CONSTRAINTS: pass.** Both web locks use `NewLinkLock`, so they share the per-lock rate limit (a burst of 5, then 1/s).
- **ARCH-SECURE: pass.**
  - An author name is parsed into a typed value at the boundary and refused off-rule (`ErrNoteText`, which never echoes input).
  - The 8 KB `MaxNoteBody` still covers a fully escaped 1 KB note plus a 40-rune escaped name.
  - The drive token has its own lock and cookie path.
  - `collab.js` before a session exposes no data.
- **ARCH-ORDER: pass.** `onerror` with `readyState === 2` is terminal and fires once, and the `ended` guard comes before the async probe. Rotation ordering is still `Gen`, from M1.
- **ARCH-FUNERAL: pass.** Locks are in memory and end after `Unmount`. A lock left behind by an early return in `StartCast` holds no goroutine or handle and is simply garbage-collected.
- **For M3:** the planned `broadcast.LinkLocks` registry should replace `Cast.viewLock`/`driveLock` and the `switch` in `RotateCode`/`Codes`/`endLocks`. As written, adding the third web link means changing four places.

**7. Plan revision recommendations**
- **Fix the Current design summary.** It still lists four codes, with the web cast having two (view, drive). The latest revision (three web links: view, pointer, drive) contradicts this. Rewrite the summary, not just the Revisions list.
- **Rewrite the stale body sections.** §Couch Display, §Click rotation and §Documented trade-off, and the M3 bullets ("click-to-rotate", "the tab bar showing codes to LIVE viewers"), still state the replaced inline-code and click-span design. Rewrite them to the operator-only layer with Alt+click, or delete them.
- **Add a Revisions entry for M2 as built:**
  - the Chrome live tests bypass the Host, and `TestLiveLockPage` replaces "updated to unlock first";
  - `collab.js` is served before a session.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The "Current design" summary was added, but it already contradicts the latest revision (four codes vs the web cast's three links), and the Couch Display/Click rotation/Documented trade-off sections and the M3 bullets still state the replaced design.
findings:
  - id: new
    severity: Important
    family: behavior-without-default-test
    title: |
      Web viewer's reload into the lock page (viewer.js:99-106) has no node test; only the env-gated Chrome test covers it
    detail: |
      The terminal viewer's twin is tested in connect.test.mjs. Add a viewer.test.mjs case with the fake EventSource at readyState 2 and an injected sessionLost: true calls relock, false shows Disconnected, and neither runs after an end event.
  - id: new
    severity: Minor
    family: spec-moves-with-decisions
    title: |
      Plan M2 says the Chrome live tests were "updated to unlock first", but they bypass the Host; TestLiveLockPage replaces them, which the plan does not record
    detail: |
      Repeat in this family. The rule: when an implementation substitutes for a plan item, the milestone bullet is rewritten in the same commit, not left for a Revisions entry. Same sweep: atlas/broadcast.md:82 and the Current design's pre-session list omit collab.js.
  - id: new
    severity: Minor
    family: docs-gate-surface
    title: |
      README web-cast sections (e.g. line 1021, "Anyone with the viewer link can point") don't mention the code lock
    detail: |
      Acceptable to defer while the operator can't see a code (M3), but the M3 close must update the README for codes, the lock page and the drive code.
```

---

## Re-review — 2026-10-12T16:19:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | f94b8f6defdb0ee361ac8829d29b9f21ed6f1749..7eb6a1dc1b90e0b88f9b8099ade5f7dcb8fbb73a |
| command | sdlc milestone-close --issue 495 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:19:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing the M2 range: production diff read, now running the Go and node tests and checking the prior findings.

Now the node tests and the new webcast lock tests.

Checking the plan's prior-finding status (BR-1, BR-11) against the code: cookie SameSite, atlas collab.js mention.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 does what the plan asked for. A web cast now has two locks: `view` with 4 digits, created before anything else, and `drive` with 6 digits, created along with the drive token. Both are attached through `Server.WithLocks` before the mount and ended after `Unmount`. The Host gate refuses every web-cast route with 401 on both links until there is a session. Every viewer page uses the same name rule (`collab.js validName` and `broadcast.ViewerName`). Terminal notes carry an `author`. The web viewer reloads into the lock page when its session is gone.

The Go suites for webcast, broadcast, castlock and couchtty all pass. couchtty only ran after I generated the runtime bundle, which is gitignored. The node suites pass too: 169 of 169. The live Chrome tests could not run here because CDP gave EOF, and the untouched `TestLiveViewerPage` failed the same way, so that is the environment, not this diff. Nothing blocks the gate. What remains is two plan/atlas sweeps the prior rounds asked for and three small test or constant-parity gaps, all Minor.

**1. Strengths**
- **One choke point for every route.** `TestWebCastLinksRefuseEveryRouteWithoutASession` (`cmd/internal/webcast/lockgate_test.go:66`) loops over the `assets` map, so a new viewer file is checked automatically.
- **The drive code is independent.** `TestWebCastDriveCodeIsSeparate` checks the view code on the drive unlock, a view session on `drive/hello`, and then the drive code itself.
- **Safe lock lifetime.** `cast.go:120` creates the view lock before any IO, so a failure leaves nothing to undo. `cast.go:312` ends the locks after `Unmount`, so viewers still receive the cast's end. If the drive lock can't be created, the drive link is withheld and the reason goes into `driveOff` (`cast.go:183-195`).
- **One name rule instead of copies.** `board/wire.go` now delegates to `broadcast.ViewerName`, and `annotate.js` and `lock.js` import the rule from `collab.js`. `cleanName` now refuses an over-long name instead of truncating it, so the lock page can no longer store a name the board would reject.
- **The relock fix is proven.** When I put `viewer.js` back to its base version, `a closed stream whose session is gone reloads into the lock page` fails.

**2. Critical findings** — none.

**3. Important findings** — none new. BR-10 is addressed (see dispositions).

**4. Minor findings**
- **BR-1 / BR-11 are not fully addressed** (`spec-moves-with-decisions`). The new "Current design" summary is a good fix, but the body still states replaced decisions:
  - Revocation: `event: locked`.
  - Couch: the inline `Chrome ⏸ 4821` display, click-to-rotate spans, and the trade-off that "viewers see codes".
  - M3 bullets: "Code display, click-to-rotate…, drive code in the drive menu".
  - M2 bullet: Chrome tests "updated to unlock first".
  - `atlas/broadcast.md:82` and the summary's pre-session list both leave out `collab.js`, which `lockgate.go:37` now serves before a session.
- **The terminal viewer's `author` path has no behavioral test** (`broadcast/web/viewer.js:296`, `collab.js storedName`). The only coverage is a source-text check in `server_test.go`.
- **The 40-character name limit is written down three times** with no test tying them together: `collab.js:96 MAX_VIEWER_NAME`, Go `MaxViewerNameRunes`, and `lock.html:20 maxlength="40"`. The Go test only pins the board to broadcast. Also, `maxlength` counts UTF-16 units, so it is stricter for astral characters.
- **`TestCastBrowserFirstThenLink` (`cast_test.go:57`) got weaker.** It never unlocks, so its 200 now comes from the lock page and the test no longer shows that the viewer page is served.
- Nits:
  - The `castlock` import sits inside the stdlib group (`cast.go:4`).
  - `TestNoteAuthorRoundTripsOnTheBoardRule` tests `broadcast.ViewerName` from inside package webcast, which repeats `TestParseNoteAuthor`.
  - `unlockWebCast` in couchtty copies webcast's `unlockCast`.
  - `lock.js` and `annotate.js` keep alias constants (`NAME_KEY`, `MAX_NAME`) where they could use the shared names directly.

**5. Test coverage notes**
- Node tests now cover all four `onerror` branches. Only the relock case fails without the fix; the other three hold either way but still pin the right behavior.
- `TestCastCodesAndRotation` checks that rotating the code ends a live SSE stream on a real `Cast`.
- `TestLiveLockPage` covers the lock page end to end: typed code, fragment code, assets loading with the cookie, and relock after rotation. It is gated on `WEBCAST_LIVE_CHROME` and I could not run it here.
- The shared package-level `castClient` jar is the pattern BR-9 already accepted. I did not raise it again.

**6. Architecture**

| Marker | Result | Note |
|---|---|---|
| ARCH-DRY | pass, minor note | Name rule consolidated. The 40-character limit restated across Go, JS and HTML is unpinned (finding above). |
| ARCH-PURE | pass | `ViewerName` and `withAuthors` are pure and tested directly. |
| ARCH-PURPOSE | pass | Both web links are locked. The third (pointer) link is deferred to pair:5 by an operator decision, which is a separable extension. Couch display is M3. |
| ARCH-MOCK | pass | Fake browser and fake tunnel; the Chrome tests are opt-in. |
| ARCH-CONSTRAINTS | pass | Rate limit per lock through `NewLinkLock`. Adding `author` (up to 40 runes) stays well inside `MaxNoteBody`. |
| ARCH-SECURE | pass | Names are parsed into a typed value at the boundary, and a bad name is refused rather than cut. `collab.js` is served before a session but contains code only. Couch's menu shows names that have already been validated. A click copies the plain `noteTextLocked` text, not the "name: text" display. |
| ARCH-ORDER | pass | The probe only runs once the stream is permanently closed (readyState 2) and the cast has not ended. After unmount the probe sees 404, which counts as "not lost", so the page shows Disconnected. Locks end after unmount. |
| ARCH-FUNERAL | pass | Locks hold no goroutines and end in `run`. The name reuses the existing localStorage key. |

For M3: `CastConfig.OnCodeRotated` and `Cast.Codes`/`RotateCode` have no consumer yet. The planned `broadcast.LinkLocks` registry should absorb `Cast`'s `view`/`drive` switch and `endLocks`, and the terminal `Session`'s equivalents, so that a new link really is one `Add`.

**7. Plan revision recommendations**
- Rewrite the M3 milestone bullets to match the Current design (operator-only layer, hover tooltip on the link controls, Alt+click Codes menu, three-flash).
- Rewrite the M2 Chrome-test bullet to say `TestLiveLockPage`.
- Rewrite or strike the replaced body paragraphs: `event: locked`, the Couch Display, Click rotation and Documented trade-off sections.
- Add `collab.js` to the pre-session list in the summary and in `atlas/broadcast.md:82`.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The Current-design summary helps, but the body still states replaced decisions (event locked, inline Chrome code display, click-span rotation, trade-off, M3 bullets); the M3 bullets are the next build list and still describe the old design. Strict is correct (lockgate.go:123), so that site is resolved.
  - id: BR-10
    disposition: addressed
    note: |
      tests/webcast/viewer.test.mjs adds sessionLost true/false/ended/reconnecting cases; reverting viewer.js to base makes the relock case fail (1 of 14).
  - id: BR-11
    disposition: not-addressed
    note: |
      The summary's Tests bullet records the substitution, but the M2 milestone bullet still says "updated to unlock first", and atlas/broadcast.md:82 plus the summary's pre-session list still omit collab.js (served pre-session at lockgate.go:37).
  - id: BR-12
    disposition: not-addressed
    note: |
      Still deferred on its own terms to the M3 close; the README has no code-lock text yet.
findings:
  - id: new
    severity: Minor
    family: behavior-without-default-test
    title: |
      Terminal viewer's note author (viewer.js beforeSave, collab.js storedName) has only a source-grep test, no behavioral one
    detail: |
      2nd in family. Rule: any page-side path that changes what a page POSTs is covered by a node test driving it through injected deps; a Go grep of the asset text is not a behavioral test. Add a node test: storedName over a fake storage (valid, invalid, missing storage) and the dialog's beforeSave adding author only when a name is stored.
  - id: new
    severity: Minor
    family: restated-constant-unpinned
    title: |
      The 40-char viewer-name bound is restated in collab.js, notes.go and lock.html maxlength with no parity test
    detail: |
      TestNoteAuthorRoundTripsOnTheBoardRule pins board.MaxNameRunes to broadcast only. If they drift, pages send names that ParseNote refuses as ErrNoteText. Pin MAX_VIEWER_NAME and maxlength from Go by reading the embedded assets, as server_test already does for keys. Note that maxlength counts UTF-16 units.
  - id: new
    severity: Minor
    family: test-asserts-weaker-than-name
    title: |
      TestCastBrowserFirstThenLink now passes on the lock page's 200 and no longer proves the viewer page is served
    detail: |
      cast_test.go:57 fetches u.Link without unlockCast, so the 200 is the lock page. Unlock first and assert the viewer page's marker, or assert the lock page explicitly and rename the test.
```
