---
gate: boundary-review
issue: 506
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T17:37:46-07:00"
      dispatched: "2026-10-12T17:35:20-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Right-click notice still says the copied link points and takes notes; at M1 it is view-only
          detail: cmd/internal/couchtty/console_webcast.go:263 copies u.Link (now view-only) with the text "(watch, point, notes)". M2 Task 8 must change it; there is also no Couch path to the pointer link until then.
          family: couch-notice-matches-link-grants
          round: 1
        - id: BR-2
          severity: Minor
          title: atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
          detail: Under the view link Resolve is hidden (annotate.js:505); qualify the sentence or point to the new paragraph in atlas/webcast.md.
          family: atlas-tracks-surface-change
          round: 1
        - id: BR-3
          severity: Minor
          title: Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
          detail: TestCastPublishesThreeLinks only checks publication and that a cast without a board has no pointer link; cast.go:277 clearing PointerLink has no regression test.
          family: plan-step-test-delivered
          round: 1
        - id: BR-4
          severity: Minor
          title: The link URL Base()+"/"+token+"/" is built three times in Cast.run
          detail: cast.go:268-273; a small linkURL(base, token) helper would be the single source (ARCH-DRY nit).
          family: shared-helper-for-repeated-construction
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: e6ecba8e381805ad9c0558c6b5f3e2a78ff6b2dc
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T19:30:46-07:00"
      dispatched: "2026-10-12T19:23:09-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: atlas/webcast-format.md still describes two links; the view link is said to allow pointing and notes
          detail: 'atlas/webcast-format.md:534-538 ("Two links ... The view link is for watching, pointing and notes") contradicts pair#506: the view link only watches, the pointer link annotates, the board refuses view-link posts, palette.json returns annotate. 2nd finding in this family; rule: a change that gives a named surface new meaning or grants greps atlas/ and README.md for the old term in the same commit. Sweep: git grep ''view link|viewer link|Two links'' atlas README.md (webcast-format.md:534, :693; README.md:901 web view should say it counts pointer-link viewers; README.md:1037).'
          family: atlas-tracks-surface-change
          round: 2
        - id: BR-6
          severity: Minor
          title: Three near-identical copyWebcast*Link functions (ARCH-DRY)
          detail: 'console_webcast.go: copyWebcastLink, copyWebcastPointerLink and copyWebcastDriveLink repeat the same steps: snapshot, then not running / copy / LinkErr / unavailable reason / not ready. 2nd in family; rule: a third parallel copy of a construction becomes one helper taking the parts that differ (which link from Update, label, grants text, unavailable reason).'
          family: shared-helper-for-repeated-construction
          round: 2
        - id: BR-7
          severity: Minor
          title: View-only link error notices still say "viewer link"
          detail: console_webcast.go:283,285 say "Chrome cast viewer link unavailable/not ready" while the success notice says "view-only link"; terminal_input.go:142 comment likewise.
          family: one-name-per-surface
          round: 2
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T19:46:43-07:00"
      dispatched: "2026-10-12T19:32:49-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: webcast-format.md:534-545 now describes three links (view-only watches, pointer annotates, drive), and :700 dates the old two-link wording; README.md:901, :1037 and :1052 updated. A git grep of atlas and README for the old name finds only atlas/broadcast.md (terminal cast, correct) and atlas/webcast.md:196 (raised below in the code-comment family finding).
          round: 3
        - id: BR-6
          disposition: addressed
          note: "console_webcast.go:205-280 has one copyWebcastLinkOf over a webcastLink table. A mutation where \U0001F446 copies the view-only link fails TestWebcastCopiesEachLink and TestWebcastLinkButtons, and disabling the missing-reason path fails TestWebcastCopiesEachLink."
          round: 3
        - id: BR-7
          disposition: addressed
          note: The error notices are built from l.name ("Chrome cast view-only link unavailable/not ready"), and the terminal_input.go:142 comment now says view-only link.
          round: 3
      findings:
        - id: BR-8
          severity: Minor
          title: Web-cast code comments still say "view link" / "viewer link"; drive/controller.go:11 is now wrong
          detail: '2nd finding in this family. Sites: webcast/drive/controller.go:11 ("Anyone with the viewer link may take a seat", but only the drive link gets a seat), webcast/server.go:227 (the view count includes pointer streams), webcast/server_drive.go:113 (the drive routes are also not found under the pointer token), webcast/link.go:25, webcast/cast.go:106, couchtty/console_webcast.go:45, couchtty/console_drive.go:236, atlas/webcast.md:196. Rule: in web-cast code and docs "view link" is retired; use "view-only link" where only it applies, "the view-only and pointer links" where both do, and include code comments in rename sweeps (git grep -nE ''view(er)? link'' cmd/internal/webcast cmd/internal/couchtty atlas/webcast*.md). atlas/broadcast.md is the terminal cast and is correct.'
          family: one-name-per-surface
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 713fffbc2d845a1ef8bc27fda8d81c908bb3a124
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T22:37:31-07:00"
      dispatched: "2026-10-12T22:35:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: "console_webcast.go webcastViewOnlyLink grants \"its viewers watch\"; \U0001F446 copies PointerLink; TestWebcastCopiesEachLink pins both notices."
          round: 4
        - id: BR-2
          disposition: addressed
          note: atlas/webcast-format.md:426 now adds "(without it under the view-only link, pair#506)".
          round: 4
        - id: BR-3
          disposition: addressed
          note: TestCastLinksClearTogetherWhenTheMountIsLost; reverting the PointerLink clear in cast.go turns it red (verified).
          round: 4
        - id: BR-4
          disposition: addressed
          note: linkURL(base, token) in cast.go is the one constructor of all three URLs.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: 'Still says view link: cast.go:159 and cast.go:162 (production), plus test comments server_drive_test.go:297, viewer_drive_live_test.go:216 (now false), lock_helpers_test.go:24, server_board_test.go:290.'
          round: 4
      findings:
        - id: BR-9
          severity: Important
          title: "README.md:1117 cast-row table still says the web \U0001F446 copies the view link"
          detail: '3rd finding in this family. The row should read "copy pointer link" in both columns; it arrived with the 495 merge and contradicts README''s own paragraph at lines 1004-1034. Rule: a rename sweep covers README.md tables, atlas, production comments and test comments, and is re-run after every merge from main.'
          family: one-name-per-surface
          round: 4
        - id: BR-10
          severity: Minor
          title: unlockCast maps a pointer code to the view-only URL; link kind to URL is restated three times
          detail: '3rd finding in this family. lock_helpers_test.go:26-32 would post the pointer code at u.Link on a recorded cast (latent trap; today''s callers are unrecorded). Rule: one Update.LinkFor(kind) accessor used by the webcastLink table, unlockCast and unlockWebCast.'
          family: shared-helper-for-repeated-construction
          round: 4
        - id: BR-11
          severity: Minor
          title: 'lessons.md: the new pair 506 section is inserted mid-section, so the 513 and 516 bullets now sit under it'
          detail: workshop/lessons.md:1439; move the new heading and its two bullets after the 516 bullet.
          family: insert-preserves-section-structure
          round: 4
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T22:43:15-07:00"
      dispatched: "2026-10-12T22:40:15-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: All eight named sites fixed, but the rule's own grep still hits cmd/internal/webcast/viewer_drive_live_test.go:216 ("The view link watches, points and notes", now false) and :220; reword both to "view-only link".
          round: 5
        - id: BR-9
          disposition: addressed
          note: README.md:1117 now reads "copy pointer link" in both columns, matching the prose.
          round: 5
        - id: BR-10
          disposition: addressed
          note: Update.LinkFor (cast.go:76) is used by the webcastLink table, unlockCast and unlockWebCast; TestUpdateLinkFor pins the mapping; no other production PointerLink/DriveLink readers.
          round: 5
        - id: BR-11
          disposition: addressed
          note: The pair 506 section now follows the 516 bullet (lessons.md:1448ff); 513/516 are back in their own section.
          round: 5
      recipe: milestone-review
      isolation: fenced
      reviewed: 84a3d0cc1ed337e7530e746d77a2526697ab9512
      blocked: false
---

# Gate ledger — pair#506 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T17:37:46-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `couch-notice-matches-link-grants` Right-click notice still says the copied link points and takes notes; at M1 it is view-only
  cmd/internal/couchtty/console_webcast.go:263 copies u.Link (now view-only) with the text "(watch, point, notes)". M2 Task 8 must change it; there is also no Couch path to the pointer link until then.
- **BR-2** [Minor] `atlas-tracks-surface-change` atlas/webcast-format.md:431 still says a pin's chip opens the note with Resolve
  Under the view link Resolve is hidden (annotate.js:505); qualify the sentence or point to the new paragraph in atlas/webcast.md.
- **BR-3** [Minor] `plan-step-test-delivered` Plan Task 4's "all three links clear when the mount is lost" assertion is not tested
  TestCastPublishesThreeLinks only checks publication and that a cast without a board has no pointer link; cast.go:277 clearing PointerLink has no regression test.
- **BR-4** [Minor] `shared-helper-for-repeated-construction` The link URL Base()+"/"+token+"/" is built three times in Cast.run
  cast.go:268-273; a small linkURL(base, token) helper would be the single source (ARCH-DRY nit).

## Round 2 — 2026-10-12T19:30:46-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `atlas-tracks-surface-change` atlas/webcast-format.md still describes two links; the view link is said to allow pointing and notes
  atlas/webcast-format.md:534-538 ("Two links ... The view link is for watching, pointing and notes") contradicts pair#506: the view link only watches, the pointer link annotates, the board refuses view-link posts, palette.json returns annotate. 2nd finding in this family; rule: a change that gives a named surface new meaning or grants greps atlas/ and README.md for the old term in the same commit. Sweep: git grep 'view link|viewer link|Two links' atlas README.md (webcast-format.md:534, :693; README.md:901 web view should say it counts pointer-link viewers; README.md:1037).
- **BR-6** [Minor] `shared-helper-for-repeated-construction` Three near-identical copyWebcast*Link functions (ARCH-DRY)
  console_webcast.go: copyWebcastLink, copyWebcastPointerLink and copyWebcastDriveLink repeat the same steps: snapshot, then not running / copy / LinkErr / unavailable reason / not ready. 2nd in family; rule: a third parallel copy of a construction becomes one helper taking the parts that differ (which link from Update, label, grants text, unavailable reason).
- **BR-7** [Minor] `one-name-per-surface` View-only link error notices still say "viewer link"
  console_webcast.go:283,285 say "Chrome cast viewer link unavailable/not ready" while the success notice says "view-only link"; terminal_input.go:142 comment likewise.

## Round 3 — 2026-10-12T19:46:43-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — webcast-format.md:534-545 now describes three links (view-only watches, pointer annotates, drive), and :700 dates the old two-link wording; README.md:901, :1037 and :1052 updated. A git grep of atlas and README for the old name finds only atlas/broadcast.md (terminal cast, correct) and atlas/webcast.md:196 (raised below in the code-comment family finding).
- BR-6 — addressed — console_webcast.go:205-280 has one copyWebcastLinkOf over a webcastLink table. A mutation where 👆 copies the view-only link fails TestWebcastCopiesEachLink and TestWebcastLinkButtons, and disabling the missing-reason path fails TestWebcastCopiesEachLink.
- BR-7 — addressed — The error notices are built from l.name ("Chrome cast view-only link unavailable/not ready"), and the terminal_input.go:142 comment now says view-only link.

### Raised

- **BR-8** [Minor] `one-name-per-surface` Web-cast code comments still say "view link" / "viewer link"; drive/controller.go:11 is now wrong
  2nd finding in this family. Sites: webcast/drive/controller.go:11 ("Anyone with the viewer link may take a seat", but only the drive link gets a seat), webcast/server.go:227 (the view count includes pointer streams), webcast/server_drive.go:113 (the drive routes are also not found under the pointer token), webcast/link.go:25, webcast/cast.go:106, couchtty/console_webcast.go:45, couchtty/console_drive.go:236, atlas/webcast.md:196. Rule: in web-cast code and docs "view link" is retired; use "view-only link" where only it applies, "the view-only and pointer links" where both do, and include code comments in rename sweeps (git grep -nE 'view(er)? link' cmd/internal/webcast cmd/internal/couchtty atlas/webcast*.md). atlas/broadcast.md is the terminal cast and is correct.

## Round 4 — 2026-10-12T22:37:31-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — console_webcast.go webcastViewOnlyLink grants "its viewers watch"; 👆 copies PointerLink; TestWebcastCopiesEachLink pins both notices.
- BR-2 — addressed — atlas/webcast-format.md:426 now adds "(without it under the view-only link, pair#506)".
- BR-3 — addressed — TestCastLinksClearTogetherWhenTheMountIsLost; reverting the PointerLink clear in cast.go turns it red (verified).
- BR-4 — addressed — linkURL(base, token) in cast.go is the one constructor of all three URLs.
- BR-8 — not-addressed — Still says view link: cast.go:159 and cast.go:162 (production), plus test comments server_drive_test.go:297, viewer_drive_live_test.go:216 (now false), lock_helpers_test.go:24, server_board_test.go:290.

### Raised

- **BR-9** [Important] `one-name-per-surface` README.md:1117 cast-row table still says the web 👆 copies the view link
  3rd finding in this family. The row should read "copy pointer link" in both columns; it arrived with the 495 merge and contradicts README's own paragraph at lines 1004-1034. Rule: a rename sweep covers README.md tables, atlas, production comments and test comments, and is re-run after every merge from main.
- **BR-10** [Minor] `shared-helper-for-repeated-construction` unlockCast maps a pointer code to the view-only URL; link kind to URL is restated three times
  3rd finding in this family. lock_helpers_test.go:26-32 would post the pointer code at u.Link on a recorded cast (latent trap; today's callers are unrecorded). Rule: one Update.LinkFor(kind) accessor used by the webcastLink table, unlockCast and unlockWebCast.
- **BR-11** [Minor] `insert-preserves-section-structure` lessons.md: the new pair 506 section is inserted mid-section, so the 513 and 516 bullets now sit under it
  workshop/lessons.md:1439; move the new heading and its two bullets after the 516 bullet.

## Round 5 — 2026-10-12T22:43:15-07:00 (claude) — passed

### Disposed

- BR-8 — not-addressed — All eight named sites fixed, but the rule's own grep still hits cmd/internal/webcast/viewer_drive_live_test.go:216 ("The view link watches, points and notes", now false) and :220; reword both to "view-only link".
- BR-9 — addressed — README.md:1117 now reads "copy pointer link" in both columns, matching the prose.
- BR-10 — addressed — Update.LinkFor (cast.go:76) is used by the webcastLink table, unlockCast and unlockWebCast; TestUpdateLinkFor pins the mapping; no other production PointerLink/DriveLink readers.
- BR-11 — addressed — The pair 506 section now follows the 516 bullet (lessons.md:1448ff); 513/516 are back in their own section.

## Open findings

- **BR-8** [Minor] `one-name-per-surface` Web-cast code comments still say "view link" / "viewer link"; drive/controller.go:11 is now wrong
