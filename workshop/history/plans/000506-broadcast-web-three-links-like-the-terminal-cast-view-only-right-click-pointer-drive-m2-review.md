# Boundary Review — pair#506 (milestone M2)

| field | value |
|-------|-------|
| issue | 506 — broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽) |
| repo | pair |
| issue file | workshop/issues/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8569c2a9e6a397b0bf52624913cc5916a8531eac..a69efa50c4cce29fd3061d45ce4fe7ed782ba56d |
| command | sdlc milestone-close --issue 506 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:30:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Webcast tests pass. Next I'm generating the embedded runtime assets the checkout is missing, so the couchtty tests can build.

Checking stale docs that still describe two links, then running a quick mutation check on the 👆 routing.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 does what the issue asks. Right-click copies the view-only link, 👆 copies the pointer link, and 👽 and Alt+right-click copy the drive link. Each notice says what its link grants and never shows the link itself. A cast that isn't recorded has no pointer link, so 👆 says "This cast takes no notes" instead. The #501 merge conflict in `cast.go` was resolved correctly: the server chain keeps both `.WithPointer(pointerToken)` and `.WithDriveLog(cfg.Logf)`, and no conflict markers are left in the tree. `go vet` is clean on the couchtty and webcast packages. Tests pass for `couchtty -run 'TestWebcast|TestRenderWebcast|TestStatusRow'`, the whole `webcast` package and `artifactpath`; I had to generate the embedded runtime files first because this checkout didn't include them. I also changed the 👆 click back to `copyWebcastLink`, and `TestWebcastLinkButtons` then never finished: its wait for the pointer link never succeeded. I stopped it after 300s, so the test does catch that bug. One thing should be fixed before SHIP: `atlas/webcast-format.md` still says there are two links and that the view link allows pointing and notes, which is no longer true.

**Strengths**
- `cmd/internal/couchtty/console_webcast.go:291-307`: `copyWebcastPointerLink` reads the cast state once and handles every case separately: not running, link ready, link error, cast has no notes, and not ready yet. "Has no notes" is told apart from "not ready yet" by checking `u.Link != ""`.
- `cmd/internal/webcast/cast.go:193-199`: `linkURL` puts the "an empty token means no link" rule in one place. The three links are now built in a single statement, and `TestCastLinksClearTogetherWhenTheMountIsLost` checks that all three clear when the mount is lost.
- `cmd/internal/couchtty/console_webcast_test.go:381-387`: the 👆 test checks that the pointer link was copied and also that the view-only link was not. That second check catches the exact mistake this change could have made.
- `TestWebcastCopiesEachLink` checks that no notice contains "http", which enforces the rule that a screen broadcast never draws a link.
- `artifactpath/manifest.go:687`: `link.go` is added to the inventory with a reason, which fixes the full-suite failure.

**Critical:** none.

**Important**
- `atlas/webcast-format.md:534-538`: the "Two links (operator, 10-12, #467)" paragraph still says "The view link … is for watching, pointing and notes" and that Couch copies the view link with right-click. Since #506 the view link only watches, and there is a pointer token, a 403 refusal, and `palette.json` now returns `{palette, annotate}`. **This is the 2nd finding in family `atlas-tracks-surface-change`.** Don't just fix this paragraph. Apply the rule that covers every case: *when a change gives a named surface new meaning or grants, the same commit greps atlas/ and README.md for the old term and fixes every hit.* For this issue that means `git grep -n 'view link\|viewer link\|Two links' atlas README.md`. The hits are `webcast-format.md:534-538` (wrong), `webcast-format.md:693` (still describes two links, so check it), `README.md:901` (`web view` should say it also counts pointer-link viewers) and `README.md:1037` (still true, but "view-only" would read better).

**Minor**
- `console_webcast.go:283,285`: the view-only error notices still say "viewer link", while the success notice says "view-only link". Use one name.
- `terminal_input.go:142`: the comment still says right-click copies "viewer link"; `console_drive.go:236` and `link.go:25` describe "view link" in ways that are still accurate.
- ARCH-DRY: there are now three near-identical copy functions (`copyWebcastLink`, `copyWebcastPointerLink`, `copyWebcastDriveLink`). Each does: read once, then not running / copy / error / unavailable reason / not ready. **This is the 2nd finding in family `shared-helper-for-repeated-construction`.** The rule: when a third parallel copy of a construction appears, turn it into one helper that takes the parts that differ. Here those are: which link to pick from `Update`, the label, the grants text, and the reason the link is unavailable. The plan asked for this mirroring, and with three copies it's still tolerable, so this doesn't block.

**Test coverage notes**
- 👆 and 👽 clicks, right-click, the no-notes cast, and the lost mount are all covered.
- `TestWebcastCopiesEachLink` doesn't check Alt+right-click itself. The existing #467 drive tests and `TestWebcastLinkButtons` (👽) cover the drive copy, and the Alt routing at `terminal_input.go:181` didn't change.
- `testwait.Until` has no timeout of its own, so a broken copy hangs until `go test -timeout` instead of failing quickly. This affects the whole repo, not just this change.

**Architecture**
- ARCH-DRY: flag (Minor), the three copy functions above. `linkURL` is a good consolidation.
- ARCH-PURE: pass. The new Couch code is thin glue over `Update`, and the decisions are a pure switch on a snapshot.
- ARCH-PURPOSE: pass on code, flag on the docs sweep. The Couch side derives from `Update`'s three links and nothing restates them by hand. The stale atlas paragraph is the restatement left behind (Important above).
- ARCH-MOCK: pass. Tests use the existing `NewFakeBrowser`, `FakeTunnel` and `FakeHost`; there's no new external dependency.
- ARCH-CONSTRAINTS: N/A. The change is a click handler and a notice, with no hot path.
- ARCH-SECURE: pass. Notices never include a link or token, and a test enforces that. Tokens are still compared in constant time on the server (from M1).
- ARCH-ORDER: pass. Each copy works from one snapshot of `(update, phase)` taken under the lock, so a cast ending between two reads can't produce a mixed answer. The lost-mount transition clears all three links together, and a test checks it.
- ARCH-FUNERAL: pass. Nothing durable is created; the pointer token lives in memory for the cast's lifetime.

**Plan revision recommendations**
- In Task 8, record that the notice wording changed from the plan's text (for example "view-only link copied: its viewers watch. 👆 copies the pointer link, 👽 the drive link."), so the plan doesn't keep stating strings the code doesn't use.
- Task 9 followed branch 2a: #501's span was renamed from `WebViewLink` to `WebPointerLink` (`reserve.go`, `console.go`, `terminal.go`, `terminal_input.go`). One line in Revisions would record that.

```findings
findings:
  - id: new
    severity: Important
    family: atlas-tracks-surface-change
    title: |
      atlas/webcast-format.md still describes two links; the view link is said to allow pointing and notes
    detail: |
      atlas/webcast-format.md:534-538 ("Two links ... The view link is for watching, pointing and notes") contradicts pair#506: the view link only watches, the pointer link annotates, the board refuses view-link posts, palette.json returns annotate. 2nd finding in this family; rule: a change that gives a named surface new meaning or grants greps atlas/ and README.md for the old term in the same commit. Sweep: git grep 'view link|viewer link|Two links' atlas README.md (webcast-format.md:534, :693; README.md:901 web view should say it counts pointer-link viewers; README.md:1037).
  - id: new
    severity: Minor
    family: shared-helper-for-repeated-construction
    title: |
      Three near-identical copyWebcast*Link functions (ARCH-DRY)
    detail: |
      console_webcast.go: copyWebcastLink, copyWebcastPointerLink and copyWebcastDriveLink repeat the same steps: snapshot, then not running / copy / LinkErr / unavailable reason / not ready. 2nd in family; rule: a third parallel copy of a construction becomes one helper taking the parts that differ (which link from Update, label, grants text, unavailable reason).
  - id: new
    severity: Minor
    family: one-name-per-surface
    title: |
      View-only link error notices still say "viewer link"
    detail: |
      console_webcast.go:283,285 say "Chrome cast viewer link unavailable/not ready" while the success notice says "view-only link"; terminal_input.go:142 comment likewise.
```

---

## Re-review — 2026-10-12T19:46:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 506 — broadcast-web: three links like the terminal cast — view-only (right-click), pointer (👆), drive (👽) |
| repo | pair |
| issue file | workshop/issues/000506-broadcast-web-three-links-like-the-terminal-cast-view-only-right-click-pointer-drive.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8569c2a9e6a397b0bf52624913cc5916a8531eac..713fffbc2d845a1ef8bc27fda8d81c908bb3a124 |
| command | sdlc milestone-close --issue 506 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:46:43-07:00 |
| verdict | SHIP |

## Review

Checking the prior findings against the head commit. BR-5 looks fixed in the docs. Now I'm checking the copy helper and running the tests.

Now running the tests and checking that the revert-to-red evidence holds.

The build was missing generated assets. Generating them, then running the tests.

Both mutations fail their tests and the working tree is clean again. Last check: whether a test covers the drive link's "can't be driven" notice now that it goes through the shared helper.

```verdict
verdict: SHIP
confidence: high
```

The M2 window is ready to ship. All three prior findings are fixed. Couch copies the right link from each control:
- right-click on the Chrome cell: the view-only link
- 👆: the pointer link
- 👽 or Alt+right-click: the drive link

Each copy shows a notice that says what the link allows and never shows the link itself. The three copy functions are now one helper, `copyWebcastLinkOf`, driven by a small table. README and atlas now describe three links. The window also removes merge-conflict markers that the base commit had left in `cmd/internal/webcast/cast.go:185`.

The build, vet and targeted tests pass for `couchtty`, `webcast` and `artifactpath`; the build first needed the `runtimebundle-generate` assets. I ran two mutations and both turned the tests red:
- **👆 copying the view-only link:** `TestWebcastCopiesEachLink` and `TestWebcastLinkButtons` both failed.
- **The "missing" reason disabled:** `TestWebcastCopiesEachLink` failed.

The only new finding is Minor: some code comments still use the old "view link" / "viewer link" name.

### 1. Strengths
- `cmd/internal/couchtty/console_webcast.go:205-280`: the `webcastLink` table holds name, link accessor, what it grants and why a link may be missing. All three copy commands are one-line wrappers over `copyWebcastLinkOf`, which reads the cast state once (one locked snapshot).
- `cmd/internal/webcast/cast.go:518-542`: `linkURL` gives the three links one construction and one empty-token rule. `TestCastLinksClearTogetherWhenTheMountIsLost` checks that all three links clear when the tunnel is lost, so Couch never copies a dead one.
- `cmd/internal/couchtty/console_webnotes_test.go:348`: the test checks which link reaches the clipboard (OSC 52), that the notice never contains `http`, and the "takes no notes" path on an unrecorded cast. `TestWebcastLinkButtons` also asserts that 👆 does not copy the view-only link.
- The span rename `WebViewLink` → `WebPointerLink` is applied everywhere: `reserve.go`, `terminal.go`, `console.go` and the tests.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- **Stale "view link" / "viewer link" in web-cast code comments** (family `one-name-per-surface`, 2nd finding). The README and atlas sweep for BR-5 didn't cover code comments. One comment is now wrong, not just old:
  - `cmd/internal/webcast/drive/controller.go:11` says "Anyone with the viewer link may take a seat", but only the drive link gets a seat.

  The others use the old name:
  - `cmd/internal/webcast/server.go:227`: the view count also includes pointer-link streams.
  - `cmd/internal/webcast/server_drive.go:113`: the drive routes are also not found under the pointer token.
  - `cmd/internal/webcast/link.go:25`
  - `cmd/internal/webcast/cast.go:106`
  - `cmd/internal/couchtty/console_webcast.go:45`
  - `cmd/internal/couchtty/console_drive.go:236`
  - `atlas/webcast.md:196`

  **The rule:** in web-cast code and docs, "view link" is retired. Say "view-only link" where only that link applies, "the view-only and pointer links" where both do, and include code comments when sweeping a rename (`git grep -nE 'view(er)? link' cmd/internal/webcast cmd/internal/couchtty atlas/webcast*.md`). `atlas/broadcast.md` describes the terminal cast, where "view link" is still the correct name; leave it alone.

### 5. Test coverage notes
- Covered: the clipboard and notice for the view-only and pointer links, the pointer link missing on an unrecorded cast, the drive link copy, and the "can't be driven" notice (`console_webcast_test.go:394`).
- Not covered, and already uncovered before this window: the `LinkErr` and "not ready" branches of the shared helper.
- When the 👆 mutation is applied, `TestWebcastLinkButtons` takes about 590s to fail, because it waits on the shared `testwait.Until` until it gives up. A regression there would show up very late in the suite.

### 6. Architecture
- **ARCH-DRY: pass.** The three copy functions are one helper and the link URLs have one constructor.
- **ARCH-PURE: pass.** The table is pure data and the helper is thin glue.
- **ARCH-PURPOSE: pass.** Every Couch control the spec covers maps to its link. The operator's smoke test is correctly moved to the issue close in the plan revision.
- **ARCH-MOCK: pass.** The tests use `FakeTunnel`, the fake browser and the fake host.
- **ARCH-CONSTRAINTS: N/A.** Click handlers only; nothing on a hot path.
- **ARCH-SECURE: pass.** No notice ever contains a token, and a test checks this.
- **ARCH-ORDER: pass.** No new state across events, and the single snapshot avoids reading a half-updated cast.
- **ARCH-FUNERAL: pass.** Nothing durable is created; tokens and links stay in memory.

### 7. Plan revisions
None needed. The notice wording differs from Task 8's sample strings, but the round-1 revision says those case lists are examples, not requirements.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      webcast-format.md:534-545 now describes three links (view-only watches, pointer annotates, drive), and :700 dates the old two-link wording; README.md:901, :1037 and :1052 updated. A git grep of atlas and README for the old name finds only atlas/broadcast.md (terminal cast, correct) and atlas/webcast.md:196 (raised below in the code-comment family finding).
  - id: BR-6
    disposition: addressed
    note: |
      console_webcast.go:205-280 has one copyWebcastLinkOf over a webcastLink table. A mutation where 👆 copies the view-only link fails TestWebcastCopiesEachLink and TestWebcastLinkButtons, and disabling the missing-reason path fails TestWebcastCopiesEachLink.
  - id: BR-7
    disposition: addressed
    note: |
      The error notices are built from l.name ("Chrome cast view-only link unavailable/not ready"), and the terminal_input.go:142 comment now says view-only link.
findings:
  - id: new
    severity: Minor
    family: one-name-per-surface
    title: |
      Web-cast code comments still say "view link" / "viewer link"; drive/controller.go:11 is now wrong
    detail: |
      2nd finding in this family. Sites: webcast/drive/controller.go:11 ("Anyone with the viewer link may take a seat", but only the drive link gets a seat), webcast/server.go:227 (the view count includes pointer streams), webcast/server_drive.go:113 (the drive routes are also not found under the pointer token), webcast/link.go:25, webcast/cast.go:106, couchtty/console_webcast.go:45, couchtty/console_drive.go:236, atlas/webcast.md:196. Rule: in web-cast code and docs "view link" is retired; use "view-only link" where only it applies, "the view-only and pointer links" where both do, and include code comments in rename sweeps (git grep -nE 'view(er)? link' cmd/internal/webcast cmd/internal/couchtty atlas/webcast*.md). atlas/broadcast.md is the terminal cast and is correct.
```
