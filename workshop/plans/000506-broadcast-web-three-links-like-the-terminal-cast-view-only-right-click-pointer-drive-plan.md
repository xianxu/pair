# pair#506: Web cast — three links (view-only, pointer, drive) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** a web cast hands out three links like the terminal cast: view-only (watch), pointer (watch, point, notes) and drive (#467, unchanged). The server enforces each link's grants, and the viewer shows only the controls its link allows.

**Architecture:** the cast mints a third token, the pointer token. The server classifies every request by the token it came under (`castLink`, a small enum with grant methods). Board POSTs refuse a link that can't annotate, before the body is read. `palette.json` reports whether this link may annotate, and the viewer reads that to mount its strokes and pins read-only or with the rail. Couch's Update carries the pointer link, and Couch maps right-click, 👆 and 👽 to the view-only, pointer and drive links.

**Tech Stack:** Go (`cmd/internal/webcast`, `cmd/internal/couchtty`), browser ES modules (`cmd/internal/webcast/web`), node tests (`tests/webcast`), headless Chrome live tests (`viewer_annotate_live_test.go`).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `castLink` | `cmd/internal/webcast/link.go` | new |
| `paletteAnswer` | `cmd/internal/webcast/server_board.go` | new |
| `readPalette` | `cmd/internal/webcast/web/annotate.js` | new |
| `Update` | `cmd/internal/webcast/cast.go` | modified |

- **castLink** — which link a request came under: `linkNone | linkView | linkPointer | linkDrive`. It has the grant methods `annotates()` (pointer, drive) and `drives()` (drive).
  - **Relationships:** one per request, derived from the URL's first segment and the server's three tokens by `Server.linkOf`.
  - **DRY rationale:** today two booleans carry this (`Owns` and `drivesUnder`), and a third would multiply the call sites. One enum answers "may this request X" in one place. Board routes, palette.json and the drive routes all ask it.
  - **Future extensions:** #495's codes attach per link. A fourth level (e.g. comment-only) is one more constant and one more grant.
- **paletteAnswer** — the JSON `palette.json` returns, `{"palette": []string, "annotate": bool}`.
- **readPalette(json)** — the viewer's pure reading of that answer. It returns `{palette, annotate}`, or `null` when the answer is malformed (the cast takes no notes). Node-tested.
- **Update** — gains `PointerLink string`. It is `""` until the mount answers, or when the cast has no board (nothing to point at).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Server.linkOf` / `Owns` / `ServeHTTP` | `cmd/internal/webcast/server.go` | modified | HTTP requests on the shared Host |
| `boardRoute` / `boardAsset` | `cmd/internal/webcast/server_board.go` | modified | the Board |
| `NewCast` token minting | `cmd/internal/webcast/cast.go` | modified | `broadcast.NewToken`, the Host mount |
| `mountAnnotate` | `cmd/internal/webcast/web/annotate.js` | modified | the viewer DOM, `fetch` |
| `copyWebcastLink`, `copyWebcastPointerLink`, `copyWebcastDriveLink` | `cmd/internal/couchtty/console_webcast.go` | modified / new | the clipboard, the notice |
| 👆/👽 web-cast controls | `cmd/internal/couchtty/terminal_input.go`, `reserve.go` | from #501, re-pointed | mouse input |

- **Server.linkOf** is injected nowhere; it is a method over the server's tokens. The Board stays unaware of links (ARCH-PURPOSE): refusal happens at the HTTP edge, before admission.
- Fakes: the existing `NewFakeBrowser`, `newBoardCast` and `newDriveFixture` fixtures, plus the headless Chrome harness. No new fake.

**ARCH-FUNERAL:** creates nothing durable. The pointer token lives in memory for the cast's lifetime, like the view and drive tokens, and dies with the Cast.
**ARCH-SECURE:** the token is compared in constant time (`broadcast.OwnsToken`). The refusal text is fixed and never echoes the request. The view token is refused before the body is read, so a view-only link can't spend the Board's rate budget or in-flight slots.

## Chunk 1: M1 — Server

### Task 1: castLink and linkOf

**Files:**
- Create: `cmd/internal/webcast/link.go`, `cmd/internal/webcast/link_test.go`
- Modify: `cmd/internal/webcast/server.go` (the `Server` struct, `Owns`, `drivesUnder`, `ServeHTTP`, `events`)

- [ ] **Step 1: failing test** `TestCastLinkGrants` in `link_test.go`: the table `{linkNone: false/false, linkView: false/false, linkPointer: true/false, linkDrive: true/true}` over `annotates()` / `drives()`.
- [ ] **Step 2: run** `go test ./cmd/internal/webcast -run TestCastLinkGrants`: FAIL (undefined).
- [ ] **Step 3: implement**

```go
// castLink is which of a cast's links a request came under (#506): each
// grants what the one before it does, and more.
type castLink uint8

const (
	linkNone castLink = iota
	linkView    // watch
	linkPointer // watch, point, notes
	linkDrive   // watch, point, notes, ask to drive (#467)
)

func (l castLink) annotates() bool { return l >= linkPointer }
func (l castLink) drives() bool    { return l == linkDrive }
```

  In `server.go`:
  - Add `pointerToken string` to `Server`, set by `WithPointer(token string) *Server`.
  - `linkOf(segment)` checks drive, then pointer, then view, each with `broadcast.OwnsToken`. A token that is `""` never matches.
  - `Owns(segment) = s.linkOf(segment) != linkNone`.
  - Replace `drivesUnder(segment)` with `s.linkOf(segment).drives()`.
  - `ServeHTTP` computes `link := s.linkOf(segment)` once and passes it to `boardRoute`, `driveRoute`, `events` and `boardAsset`.
- [ ] **Step 4:** `TestOwnsAnswersEachLink` in `link_test.go`: a server with view `tok`, pointer `ptr` and drive `drv` owns all three and not `nope`, and `linkOf` returns each kind. Run and pass, along with `TestDriveOnlyUnderTheDriveLink` unchanged.
- [ ] **Step 5: commit** `#506 M1: castLink: one place says what each cast link grants`.

### Task 2: Board routes refuse the view link

**Files:**
- Modify: `cmd/internal/webcast/server_board.go` (`boardRoute`, the refusals block)
- Test: `cmd/internal/webcast/server_board_test.go`

- [ ] **Step 1: failing test** `TestBoardRoutesByLink`:
  - Build `newBoardCast` with `.WithPointer("ptr")`.
  - For each route in stroke, note, note-state:
    - Under `/tok/` it answers 403 with body `This link is view-only.`, and the board holds no note afterwards (`board.Notes()` empty, or the hub got no record).
    - Under `/ptr/` it answers 204 for a valid note.
  - A drive fixture with a board answers 204 under `/drv/`.
  - Also assert that a view-link POST with an over-limit body gets 403, not 413, which proves refusal happens before the body is read.
- [ ] **Step 2: run**: FAIL (the view link answers 204).
- [ ] **Step 3: implement**
  - Add `refuseViewOnly = "This link is view-only."` to the refusals.
  - In `boardRoute(w, r, rest, link)`, once the route is recognised and the board exists: `if !link.annotates() { http.Error(w, refuseViewOnly, http.StatusForbidden); return true }`. This happens before `admit`.
  - Update the header comment (lines 16-21): the routes take the pointer and drive links, not the view link.
- [ ] **Step 4: fixtures, then run** the webcast package: PASS.
  - Existing board tests post under `c.url` = `/tok/` and must keep annotating.
  - So `newBoardCast` serves view token `view` and pointer token `tok` (`NewServer("view", …).WithPointer("tok")`), and the new test's view-only requests use `/view/`.
  - `TestBoardRoutesNeedABoard`'s unknown-token case keeps answering 405.
- [ ] **Step 5: commit** `#506 M1: the board refuses the view-only link before reading a body`.

### Task 3: palette.json says whether this link annotates

**Files:**
- Modify: `cmd/internal/webcast/server_board.go` (`boardAsset`)
- Test: `cmd/internal/webcast/server_board_test.go` (`TestBoardAssets`)

- [ ] **Step 1: failing test:**
  - Extend `TestBoardAssets`: `palette.json` under the pointer link decodes as `paletteAnswer{Palette: 6 colors, Annotate: true}`, and under the view link as `Annotate: false` with the same palette.
  - Without a board it is still 404.
- [ ] **Step 2: run**: FAIL (an array decodes into a struct with an error).
- [ ] **Step 3: implement** `type paletteAnswer struct { Palette []string `json:"palette"`; Annotate bool `json:"annotate"` }`; `boardAsset(w, rest, link)` encodes `paletteAnswer{broadcast.PointerPalette(), link.annotates()}`.
- [ ] **Step 4: run**: PASS. **Step 5: commit** `#506 M1: palette.json tells the viewer whether its link may annotate`.

### Task 4: the cast mints the pointer token and publishes PointerLink

**Files:**
- Modify: `cmd/internal/webcast/cast.go` (`Update`, `NewCast` around lines 139-175, `run` around 250-262, the `lost` branch)
- Test: `cmd/internal/webcast/cast_drive_test.go` (or a new `cast_links_test.go`)

- [ ] **Step 1: failing test** `TestCastPublishesThreeLinks`, using the cast test harness `TestCastDriving` uses:
  - A recorded, driven cast's Update has three distinct non-empty links, `Link`, `PointerLink` and `DriveLink`.
  - Each ends in its own token, and `Server.linkOf` classifies each as view, pointer and drive.
  - A cast without a board has `PointerLink == ""`.
  - When the mount is lost, all three clear.
- [ ] **Step 2: run**: FAIL.
- [ ] **Step 3: implement:**
  - When `bd != nil`, mint `pointerToken` with `broadcast.NewToken()`. On error, close and return like the view token's error path: a board without its link is a broken cast, not a degraded one.
  - `.WithPointer(pointerToken)`.
  - Store `c.pointerToken`.
  - In `run`, set `cur.PointerLink = base + "/" + c.pointerToken + "/"` when it is non-empty, and clear it with the others on `lost`.
- [ ] **Step 4: run** `go test ./cmd/internal/webcast/...`: PASS. **Step 5: commit** `#506 M1: a web cast mints a pointer link beside the view and drive links`.

### Task 5: M1 boundary
- [ ] `go vet ./cmd/internal/webcast/...`, the webcast package tests, then `sdlc milestone-close --issue 506 --milestone M1`.

## Chunk 2: M2 — Viewer and Couch

### Task 6: the viewer reads the palette answer and mounts read-only

**Files:**
- Modify: `cmd/internal/webcast/web/annotate.js` (new export `readPalette`; `mountAnnotate` lines ~272-345)
- Test: `tests/webcast/annotate.test.mjs`

- [ ] **Step 1: failing node test** `readPalette`:
  - `{palette:['#112233'], annotate:true}` → `{palette:[…], annotate:true}`.
  - `annotate:false` → annotate false.
  - A bare array (an older server), a missing palette, a bad hex color or a non-boolean annotate → `null`.
- [ ] **Step 2: run** `node --test tests/webcast/annotate.test.mjs`: FAIL.
- [ ] **Step 3: implement** `export function readPalette(x)`.
  - In `mountAnnotate`: `const answer = readPalette(json); if (!answer) return null;`.
  - When `!answer.annotate`, still build the layer, strokes canvas, pins and the dialog. Do not build the rail, swatches or mode buttons. Hide the dialog's Resolve and Save, and never call `openDraft`.
  - The module's return value gains `annotate` so the live test can read it.
- [ ] **Step 4: run** node tests: PASS.

### Task 7: live test — a view-only viewer watches and cannot draw

**Files:** Modify `cmd/internal/webcast/viewer_annotate_live_test.go`.

- [ ] **Step 1:**
  - Serve the same board with `.WithPointer("ptr")`. The existing viewers A and B use `/ptr/`.
  - Add viewer C on `/tok/`. Assert C has no `.annotate-rail` and no `.mode` buttons, that `couchViewer.annotate.annotate === false`, and that C draws A's note pin. Clicking it opens the dialog with the text and Resolve hidden.
  - A `fetch('stroke', …)` from C's page answers 403 `This link is view-only.`
- [ ] **Step 2: run** the live test (headless Chrome; `go test ./cmd/internal/webcast -run TestAnnotateLive`): FAIL before Task 6, PASS after.
- [ ] **Step 3: commit** `#506 M2: a view-only web cast viewer sees notes and strokes read-only`.

### Task 8: Couch's three copy commands

**Files:**
- Modify: `cmd/internal/couchtty/console_webcast.go` (`copyWebcastLink`, a new `copyWebcastPointerLink`)
- Test: `cmd/internal/couchtty/console_drive_test.go` (or `console_webcast_test.go`, next to the existing copy test)

- [ ] **Step 1: failing test** `TestWebcastCopiesEachLink`: with a fake cast Update carrying three links:
  - right-click on the Chrome cell copies `Link`, with the notice `Chrome cast view-only link copied (watch). 👆 for the pointer link, 👽 for the drive link.`;
  - `copyWebcastPointerLink` copies `PointerLink`, with the notice `Chrome cast pointer link copied (watch, point, notes).`;
  - Alt+right-click still copies `DriveLink`.
  - No notice contains a link.
  - `PointerLink == ""` on a cast without notes gives the notice `This cast takes no notes.`
- [ ] **Step 2: run**: FAIL. **Step 3: implement** both functions, mirroring `copyWebcastDriveLink`'s single-snapshot shape. **Step 4: PASS.**

### Task 9: point 👆 and 👽 at their links

- [ ] **Step 1:** `git merge origin/main`. #501 must have landed; if it hasn't, stop here and report to the hub that M2's last task waits on #501.
- [ ] **Step 2a, if #501 added web-cast 👆/👽 spans:** their click branches in `terminal_input.go` call `copyWebcastPointerLink` and `copyWebcastDriveLink`. Extend #501's span test to assert which link each copies.
- [ ] **Step 2b, otherwise:**
  - Add `WebPointer` and `WebDriveLink` spans to `RenderedStatusRow`, drawn after the Chrome cell from the same clipping pass (`reserve.go`).
  - Add their `statusWeb…` fields on the Console and their click branches.
  - Write a span test like `TestStatusRowPointerCells`, and a click test.
- [ ] **Step 3: commit** `#506 M2: 👆 and 👽 copy the web cast's pointer and drive links`.

### Task 10: docs, suite, close
- [ ] `atlas/webcast.md`: the three links, `castLink`, the refusal, `palette.json`'s answer. `README.md`: the web cast's link paragraph. #495's and #501's Logs: one line each on the mapping.
- [ ] Full `make -k test` (it queues on the suite lock), then merge main.
- [ ] Ask ops:0 to arrange the operator's smoke test, then `sdlc close --issue 506`.

## Revisions
- 2026-10-12, plan-quality round 1 (three Minors, no blockers):
  - **Viewer counts:** a pointer-link stream counts as a view stream. `Server.Viewers()` keeps `(view, drive)`, and `couch --broadcast-list` is unchanged. Both kinds are watchers without a seat; only the drive link's streams can hold one. `events(w, r, link)` picks `driveViewers` when `link.drives()`, and `viewViewers` otherwise. Task 1 asserts this with a pointer stream.
  - **Why `castLink` is the web cast's own:** the terminal cast's pointer link is a phase, minted lazily and switched on and off (`broadcast.PointerState`), and it has no drive level. The web cast's three links are fixed for the cast's life. Sharing one enum would bring the terminal's on/off phase into a model that doesn't have it. When #407 builds terminal remote control, the terminal gets a drive level, and moving `castLink` into `broadcast` becomes the natural step; until then a shared type is speculative (YAGNI).
  - **Test strategy, one line per risky function:**
    - `castLink`: a grant table.
    - `boardRoute`: ordering, proved by a 403-not-413 over-limit body under the view link.
    - `readPalette`: a table seeded with legacy (bare array) and malformed shapes.
    - Couch copies: one test per control asserting which link reaches the clipboard and that no notice contains a link.
    - The case lists in Tasks 2, 4, 6 and 8 are examples of these strategies, not separate requirements.
