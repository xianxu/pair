---
gate: boundary-review
issue: 495
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T14:37:19-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
          detail: |-
            2nd in family. The rule: a revision that replaces a decision rewrites the body paragraph stating it, not just a Revisions entry. Sites: plan Couch Display/Click rotation/Documented trade-off, Session transport Strict, Plan M3 bullets, Shape lock page (name on every link per D6).
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: spec-moves-with-decisions
          round: 1
        - id: BR-2
          severity: Minor
          title: D1 (session cookie) still awaits the operator; the M2 credentials change depends on it
          detail: |-
            M1 can proceed behind sessionFrom(r). Confirm D1 before starting M2.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: open-decision-gates-milestone
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T14:37:19-07:00"
      dispatched: "2026-10-12T14:31:04-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: The session cookie ships SameSite=Strict, but the plan's PQ-5 revision chose Lax (D1 not recorded)
          detail: lockgate.go:107, its lockgate_test assertion and the atlas all say Strict, while plan Revisions PQ-5 and the gate dispositions say Lax. Viewers who re-open the link from chat are asked for the code again. Switch to Lax, or record the operator's D1 decision for Strict in Revisions.
          family: spec-moves-with-decisions
          round: 2
        - id: BR-4
          severity: Important
          title: castlock OnRotated callbacks run outside the mutex and carry no generation, so concurrent rotations can deliver a stale code last
          detail: castlock.go:142-147 and 199-201. A guess-rotation racing an operator Rotate can call OnRotated in reverse order. The M3 Couch consumer would then show or flash a dead code. Add a monotonic Gen to Rotation, stamped under the mutex, or require consumers to re-read Code(). Add a concurrent test.
          family: unordered-async-notifications
          round: 2
        - id: BR-5
          severity: Minor
          title: M1 locks the LIVE and pointer links, but nothing shows the operator the code until M3
          detail: Codes, RotateCode, OnCodeRotated and WithLocks have no production consumers. Don't merge the branch before M3.
          family: milestone-leaves-feature-unusable
          round: 2
        - id: BR-6
          severity: Minor
          title: 'The plan still says revoke sends event: locked and that a sessionFrom(r) seam exists; the code does neither'
          detail: The code cancels the request context and the page probes GET session. The cookie is read inline in gate.
          family: spec-moves-with-decisions
          round: 2
        - id: BR-7
          severity: Minor
          title: PQ-4 asked to fuzz the unlock body decoder; FuzzUnlock fuzzes castlock.Unlock instead
          family: fuzz-target-mismatch
          round: 2
        - id: BR-8
          severity: Minor
          title: A failed guess-rotation newCode error is silently dropped, and a token-generation failure is reported as "cast has ended"
          family: silent-error-swallow
          round: 2
        - id: BR-9
          severity: Minor
          title: couchtty test init sets http.DefaultClient.Jar package-wide, sharing one cookie jar across every test
          family: test-global-state
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T14:44:53-07:00"
      dispatched: "2026-10-12T14:41:47-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Body still states replaced designs (Host.Lock at line 10, Unlock-to-(session,ok) and Rotate(revoke) and Revoked() at 14-16, event: locked at 29-30, name on web links only at 37, inline codes and click rotation at 55-63, M3 click-to-rotate at 78, the draft status line at 3). The rule from the 2nd finding still applies: rewrite each body paragraph when a revision replaces it.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: D1 is recorded as Strict with no expiry (plan line 139), so M2's credentials change is no longer waiting on a decision.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Plan Revisions line 139 records D1=Strict, superseding PQ-5. It matches lockgate.go:121, lockgate_test:156 and the atlas trade-off. The issue Log has no separate record of the operator's answer.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Rotation.Gen is stamped under the mutex (castlock.go:169,217), with the concurrent test TestConcurrentRotationsNewestGenIsCurrentCode. No test covers Gen on a guess-rotation.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Plan line 146 commits to landing the branch only after M3, the remedy the finding asked for. This is a prose-only change and correct.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: 'Revisions (lines 140-143) now describe revocation as built, but body lines 29-30 still say event: locked and line 105 the sessionFrom(r) seam. Same rule as BR-1, so fix it in the same body rewrite.'
          round: 3
        - id: BR-7
          disposition: addressed
          note: FuzzUnlockBody drives the unlock HTTP handler. It ran for 10s (about 266k runs) and passed.
          round: 3
        - id: BR-8
          disposition: addressed
          note: A spent code that cannot rotate now fails closed, and a mint failure is Failed (500). Both tests went red when I reverted the fix. The rotation error still goes unreported, which is harmless on Go 1.26, where crypto/rand cannot fail.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Tests use a package-level castClient in both packages, and no DefaultClient.Jar remains in cmd/. I could not run couchtty here because its generated runtimebundle assets are missing.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: e137bb9d7e2e419d3505a519f4e8719ad06f4a75
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T16:13:52-07:00"
      dispatched: "2026-10-12T16:10:46-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The "Current design" summary was added, but it already contradicts the latest revision (four codes vs the web cast's three links), and the Couch Display/Click rotation/Documented trade-off sections and the M3 bullets still state the replaced design.
          round: 4
      findings:
        - id: BR-10
          severity: Important
          title: Web viewer's reload into the lock page (viewer.js:99-106) has no node test; only the env-gated Chrome test covers it
          detail: 'The terminal viewer''s twin is tested in connect.test.mjs. Add a viewer.test.mjs case with the fake EventSource at readyState 2 and an injected sessionLost: true calls relock, false shows Disconnected, and neither runs after an end event.'
          family: behavior-without-default-test
          round: 4
        - id: BR-11
          severity: Minor
          title: Plan M2 says the Chrome live tests were "updated to unlock first", but they bypass the Host; TestLiveLockPage replaces them, which the plan does not record
          detail: 'Repeat in this family. The rule: when an implementation substitutes for a plan item, the milestone bullet is rewritten in the same commit, not left for a Revisions entry. Same sweep: atlas/broadcast.md:82 and the Current design''s pre-session list omit collab.js.'
          family: spec-moves-with-decisions
          round: 4
        - id: BR-12
          severity: Minor
          title: README web-cast sections (e.g. line 1021, "Anyone with the viewer link can point") don't mention the code lock
          detail: Acceptable to defer while the operator can't see a code (M3), but the M3 close must update the README for codes, the lock page and the drive code.
          family: docs-gate-surface
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T16:19:26-07:00"
      dispatched: "2026-10-12T16:14:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The Current-design summary helps, but the body still states replaced decisions (event locked, inline Chrome code display, click-span rotation, trade-off, M3 bullets); the M3 bullets are the next build list and still describe the old design. Strict is correct (lockgate.go:123), so that site is resolved.
          round: 5
        - id: BR-10
          disposition: addressed
          note: tests/webcast/viewer.test.mjs adds sessionLost true/false/ended/reconnecting cases; reverting viewer.js to base makes the relock case fail (1 of 14).
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: The summary's Tests bullet records the substitution, but the M2 milestone bullet still says "updated to unlock first", and atlas/broadcast.md:82 plus the summary's pre-session list still omit collab.js (served pre-session at lockgate.go:37).
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: Still deferred on its own terms to the M3 close; the README has no code-lock text yet.
          round: 5
      findings:
        - id: BR-13
          severity: Minor
          title: Terminal viewer's note author (viewer.js beforeSave, collab.js storedName) has only a source-grep test, no behavioral one
          detail: '2nd in family. Rule: any page-side path that changes what a page POSTs is covered by a node test driving it through injected deps; a Go grep of the asset text is not a behavioral test. Add a node test: storedName over a fake storage (valid, invalid, missing storage) and the dialog''s beforeSave adding author only when a name is stored.'
          family: behavior-without-default-test
          round: 5
        - id: BR-14
          severity: Minor
          title: The 40-char viewer-name bound is restated in collab.js, notes.go and lock.html maxlength with no parity test
          detail: TestNoteAuthorRoundTripsOnTheBoardRule pins board.MaxNameRunes to broadcast only. If they drift, pages send names that ParseNote refuses as ErrNoteText. Pin MAX_VIEWER_NAME and maxlength from Go by reading the embedded assets, as server_test already does for keys. Note that maxlength counts UTF-16 units.
          family: restated-constant-unpinned
          round: 5
        - id: BR-15
          severity: Minor
          title: TestCastBrowserFirstThenLink now passes on the lock page's 200 and no longer proves the viewer page is served
          detail: cast_test.go:57 fetches u.Link without unlockCast, so the 200 is the lock page. Unlock first and assert the viewer page's marker, or assert the lock page explicitly and rename the test.
          family: test-asserts-weaker-than-name
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 7eb6a1dc1b90e0b88f9b8099ade5f7dcb8fbb73a
      blocked: false
    - "n": 6
      timestamp: "2026-10-12T17:25:49-07:00"
      dispatched: "2026-10-12T17:18:00-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan now leads with a Current design section (Couch layer/tooltip/Alt+click menu/flash, SameSite=Strict at line 16) and fences the draft under an explicit superseded banner (line 32); current M3 deliverables live in the issue Plan checklist.
          round: 6
      findings:
        - id: BR-16
          severity: Important
          title: Code-rotation ordering is keyed by (cast kind, link), so a late rotation from an ended cast poisons the next cast
          detail: 'castlock fires OnRotated outside its mutex and Gen restarts at 1 per lock; onCodeRotated (console_castcodes.go:283) keys gens by linkKey(kind, link), which forgetCastCodes clears at cast end. A late callback re-inserts the old Gen, so the next cast''s first guess rotations are dropped (no alert or flash), a dead code flashes, and a hovered tooltip shows the dead code. 2nd in family. Rule: order async notifications by (producer instance, Gen) and drop ones from a non-current producer. Fix: capture the cast attempt in the OnCodeRotated closures (console_broadcast.go:161, console_webcast.go:62) and check it on delivery; add a test that ends cast A, delivers A''s rotation, starts B and delivers B''s Gen 1.'
          family: unordered-async-notifications
          round: 6
        - id: BR-17
          severity: Important
          title: README never mentions the cast-link code, the lock page, the hover tooltip or Alt+click Codes menu
          detail: 'README.md:865-872 still says viewers open the link and see the screen live; the pointer (950-955), web view and drive (982-995) paragraphs omit the code, and the drive link needs a 6-digit code. 2nd in family. Rule: the README gate covers changed behaviour of already-documented surfaces, not only new commands. Sweep every README paragraph describing a cast link recipient''s or the operator''s actions in the same range.'
          family: docs-gate-surface
          round: 6
        - id: BR-18
          severity: Important
          title: M3's operator smoke test on desktop and iPad is not recorded
          detail: It is in the M3 checklist and Done-when, and absent from the issue Log. The iPad pass must confirm that hover and Alt+click (which need a trackpad and keyboard) can reach the codes, or record a fallback.
          family: done-when-item-unverified
          round: 6
        - id: BR-19
          severity: Minor
          title: Issue Spec item 6 still says clicking the code in the tab bar rotates it
          detail: 'This contradicts Spec item 7 (no inline codes; the Alt+click Codes menu). 5th in family. Rule: a decision that replaces another greps every artifact restating it (issue Spec, Done-when, plan, atlas) for the old terms in the same commit.'
          family: spec-moves-with-decisions
          round: 6
        - id: BR-20
          severity: Minor
          title: The Codes menu's "rotate all" item is never invoked by a test
          detail: 'TestWebCastCodesOnTheChromeCell only asserts the item is present. 3rd in family. Rule: every Spec''d user action (each menu item) has a test that invokes it and asserts its effect, here that every link''s code changes and its viewers disconnect.'
          family: behavior-without-default-test
          round: 6
        - id: BR-21
          severity: Minor
          title: Link-copy notices do not tell the operator that viewers need a code or where to see it
          detail: console_broadcast.go:228, console_pointer.go:130 and :148, and console_webcast.go:268 copy a now-locked link without hinting to hover for its code.
          family: operator-affordance-discoverability
          round: 6
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 7
      timestamp: "2026-10-12T21:13:35-07:00"
      dispatched: "2026-10-12T21:10:55-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: addressed
          note: linkKey now includes the start attempt and onCodeRotated drops non-current attempts; reverting the guard fails TestLateRotationFromAnEndedCastIsDropped.
          round: 7
        - id: BR-17
          disposition: addressed
          note: 'README #495 section covers code, lock page, hover, Alt+click menu and a per-control table; README:886 and :1015 could still gain a "after the code" clause.'
          round: 7
        - id: BR-18
          disposition: addressed
          note: Issue Log records the operator smoke test on desktop and iPad (build bd801297) plus the re-test of f7e6cc7a.
          round: 7
        - id: BR-19
          disposition: not-addressed
          note: Issue line 35 still says clicking the code in the tab bar rotates it; atlas/webcast.md:178 is a sibling (raised below).
          round: 7
        - id: BR-20
          disposition: not-addressed
          note: No test runs chooseCodes(t, codesAction{}); rotate all is only selected in the pure layout test.
          round: 7
        - id: BR-21
          disposition: not-addressed
          note: console_pointer.go:109/130/148, console_broadcast.go:231, console_webcast.go:285/306 still copy locked links with no code hint.
          round: 7
      findings:
        - id: BR-22
          severity: Minor
          title: atlas/webcast.md:178 still says Alt+right-click on the Chrome cell copies the drive link, which this range removed
          detail: 'This is the 6th finding in family spec-moves-with-decisions. Rule: when a commit replaces a binding or decision, grep the issue Spec, Done-when, plan current-design section, atlas and README for the old terms in that same commit. Here grep "Alt.*right" and "clicking the code" across atlas/, README.md, and the issue file; fix the two hits (atlas/webcast.md:178, issue line 35).'
          family: spec-moves-with-decisions
          round: 7
        - id: BR-23
          severity: Minor
          title: onCodeRotated checks the cast attempt under c.mu but writes the flash later under codes.mu, so a cast ending in between still flashes its dead code
          detail: 'This is the 3rd finding in family unordered-async-notifications. Rule: the currency check and the state write for an async notification happen in one critical section against the same state that end-of-producer clears. Fix: have forgetCastCodes record the ended attempt (or the current attempt) in castCodes and check it under codes.mu at console_castcodes.go:436.'
          family: unordered-async-notifications
          round: 7
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      reviewed: 3bf5f83bfac0357b746abeff1d8e76fdd738ed53
      blocked: false
    - "n": 8
      timestamp: "2026-10-12T21:40:42-07:00"
      dispatched: "2026-10-12T21:35:26-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: 'Current design line 15 states there is no event: locked; the draft is fenced as superseded (line 32), naming event: locked and sessionFrom(r) explicitly.'
          round: 8
        - id: BR-11
          disposition: not-addressed
          note: The Tests bullet (line 28) records TestLiveLockPage, but collab.js is still missing from the pre-session lists at plan line 13 and atlas/broadcast.md:82, while lockgate.go:38 serves it.
          round: 8
        - id: BR-12
          disposition: addressed
          note: README.md:1071-1095 adds a code-lock section covering every link (view, pointer, drive), the lock page and URL#code, which covers the web-cast paragraph at :1051.
          round: 8
        - id: BR-13
          disposition: not-addressed
          note: storedName and beforeSave author are still pinned only by the source grep in server_test.go:167-172; no .mjs test drives them.
          round: 8
        - id: BR-14
          disposition: not-addressed
          note: lock.html:20 maxlength="40" and collab.js:96 MAX_VIEWER_NAME=40 still have no parity test against board.MaxNameRunes.
          round: 8
        - id: BR-15
          disposition: not-addressed
          note: cast_test.go:57 still GETs u.Link without unlockCast and accepts the lock page's 200 as "viewer page".
          round: 8
        - id: BR-19
          disposition: addressed
          note: Issue Spec item 6 now describes rotation from the Codes menu (Alt+click, rotate / rotate all); there is no click-to-rotate text left.
          round: 8
        - id: BR-20
          disposition: addressed
          note: console_castcodes_test.go:170-177 clicks the rotate all word through chooseCodes(t, codesAction{}) and asserts both codes change. Passes under -race.
          round: 8
        - id: BR-21
          disposition: addressed
          note: Every link-copy notice (console_broadcast.go:231, console_pointer.go:109/130/148, console_webcast.go:285/306) now says viewers need the code and where to hover, without the code itself.
          round: 8
        - id: BR-22
          disposition: not-addressed
          note: atlas/webcast.md:178-179 still says "Alt (Option)+right-click copies the drive link"; commit 276d2dd8 edited lines 160-164 instead. The test at console_castcodes_test.go:165 asserts the opposite.
          round: 8
        - id: BR-23
          disposition: not-addressed
          note: 'Ineffective: broadcastEnded/webcastEnded keep attempt unchanged (console_broadcast.go:268, console_webcast.go:242), so a rotation after end passes attempt==current. A probe (stop, wait for session nil, onCodeRotated with the old attempt) sets a flash for code 9999. Fix: also require session/cast non-nil under c.mu.'
          round: 8
      findings:
        - id: BR-24
          severity: Minor
          title: 'Plan Current design contradicts the closed M3: tooltip on link controls, status "M3 in progress", LinkLocks "Add"'
          detail: "7th finding in this family. Plan line 23 says the hover hangs on \U0001F446/\U0001F47D, but castCellAt (console_castcodes.go:115) and atlas/webcast.md:163 put it on the LIVE/Chrome cells. Line 3 says M3 is in progress, and line 11 names Add instead of Mint. Earlier rules applied per artifact still miss sites. Rule: at a milestone close, compare the Current design summary line by line against the closing Revisions and the code, as part of the close commit."
          family: spec-moves-with-decisions
          round: 8
      recipe: milestone-review
      isolation: fenced
      reviewed: dab29e48d993290317187021ff68229ea1179cbf
      blocked: false
    - "n": 9
      timestamp: "2026-10-12T21:51:42-07:00"
      dispatched: "2026-10-12T21:49:17-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: not-addressed
          note: 'The plan''s Current design now records TestLiveLockPage, but the sweep is undone: atlas/broadcast.md:81-82 and the plan''s Gate bullet still list only lock.js/lock.css, while lockgate.go:37 also serves collab.js before a session. The issue''s M2 checkbox still says "headless-Chrome tests unlock first".'
          round: 9
        - id: BR-13
          disposition: not-addressed
          note: server_test.go:167-169 is still a source grep. No node test drives storedName or the beforeSave in viewer.js:296.
          round: 9
        - id: BR-14
          disposition: not-addressed
          note: MAX_VIEWER_NAME = 40 (collab.js:96) and maxlength="40" (lock.html:20) are still unpinned against board.MaxNameRunes.
          round: 9
        - id: BR-15
          disposition: not-addressed
          note: cast_test.go:57 still GETs u.Link without unlockCast and passes on the lock page's 200.
          round: 9
        - id: BR-22
          disposition: addressed
          note: "atlas/webcast.md:178-180 now says right-click copies the view link and \U0001F47D or the Codes menu copies drive (#495 dropped Alt+right-click). The issue's Spec item 6 now names the Codes menu. Prose-only fix, checked against codes_menu.go and castCellAt."
          round: 9
        - id: BR-23
          disposition: addressed
          note: 'console_castcodes.go:435-441 adds the liveness check in the same c.mu section. The regression goes red when I revert it (TestRotationAfterTheCastEndedIsDropped: "an ended cast''s rotation flashed"). Only the castScreen branch is exercised; castWeb liveness is untested.'
          round: 9
        - id: BR-24
          disposition: not-addressed
          note: "Plan line 3 still says \"M3 in progress\", line 11 still says \"A new link is one Add\", and line 23 still hangs the tooltip on \U0001F446/\U0001F47D, contradicting castCellAt (console_castcodes.go:115) and atlas/webcast.md. Rule: at a milestone close, diff the Current design line by line against the code and atlas in the close commit itself. That rule also covers the BR-11 sweep."
          round: 9
      recipe: milestone-review
      isolation: fenced
      reviewed: 20ae28ee57f3a54b9210f45bef8974d0e524adb0
      blocked: false
    - "n": 10
      timestamp: "2026-10-12T21:53:44-07:00"
      dispatched: "2026-10-12T21:52:22-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: not-addressed
          note: Plan line 105 still says "updated to unlock first" (only the Current design notes the swap), and atlas/broadcast.md around line 82 still lists only lock.js, lock.css and unlock before a session; lockgate.go:37 also serves collab.js.
          round: 10
        - id: BR-13
          disposition: not-addressed
          note: Only server_test.go:167 greps for "storedName(storage)". No node test drives storedName or the beforeSave author path through injected storage.
          round: 10
        - id: BR-14
          disposition: not-addressed
          note: MaxViewerNameRunes (notes.go:152), MAX_VIEWER_NAME (collab.js:96) and lock.html maxlength="40" still have no test that reads the embedded assets and checks them against Go.
          round: 10
        - id: BR-15
          disposition: not-addressed
          note: cast_test.go:57 still GETs u.Link without unlockCast and asserts only a 200, which is the lock page's.
          round: 10
        - id: BR-24
          disposition: not-addressed
          note: "The plan file is unchanged since the last round. Line 3 says \"M3 in progress\", line 11 says \"Add\" (the code has Mint), and line 23 still puts the tooltip on \U0001F446/\U0001F47D."
          round: 10
      recipe: milestone-review
      isolation: fenced
      reviewed: 20ae28ee57f3a54b9210f45bef8974d0e524adb0
      blocked: false
---

# Gate ledger — pair#495 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T14:37:19-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `spec-moves-with-decisions` Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
  2nd in family. The rule: a revision that replaces a decision rewrites the body paragraph stating it, not just a Revisions entry. Sites: plan Couch Display/Click rotation/Documented trade-off, Session transport Strict, Plan M3 bullets, Shape lock page (name on every link per D6).
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Minor] `open-decision-gates-milestone` D1 (session cookie) still awaits the operator; the M2 credentials change depends on it
  M1 can proceed behind sessionFrom(r). Confirm D1 before starting M2.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-10-12T14:37:19-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `spec-moves-with-decisions` The session cookie ships SameSite=Strict, but the plan's PQ-5 revision chose Lax (D1 not recorded)
  lockgate.go:107, its lockgate_test assertion and the atlas all say Strict, while plan Revisions PQ-5 and the gate dispositions say Lax. Viewers who re-open the link from chat are asked for the code again. Switch to Lax, or record the operator's D1 decision for Strict in Revisions.
- **BR-4** [Important] `unordered-async-notifications` castlock OnRotated callbacks run outside the mutex and carry no generation, so concurrent rotations can deliver a stale code last
  castlock.go:142-147 and 199-201. A guess-rotation racing an operator Rotate can call OnRotated in reverse order. The M3 Couch consumer would then show or flash a dead code. Add a monotonic Gen to Rotation, stamped under the mutex, or require consumers to re-read Code(). Add a concurrent test.
- **BR-5** [Minor] `milestone-leaves-feature-unusable` M1 locks the LIVE and pointer links, but nothing shows the operator the code until M3
  Codes, RotateCode, OnCodeRotated and WithLocks have no production consumers. Don't merge the branch before M3.
- **BR-6** [Minor] `spec-moves-with-decisions` The plan still says revoke sends event: locked and that a sessionFrom(r) seam exists; the code does neither
  The code cancels the request context and the page probes GET session. The cookie is read inline in gate.
- **BR-7** [Minor] `fuzz-target-mismatch` PQ-4 asked to fuzz the unlock body decoder; FuzzUnlock fuzzes castlock.Unlock instead
- **BR-8** [Minor] `silent-error-swallow` A failed guess-rotation newCode error is silently dropped, and a token-generation failure is reported as "cast has ended"
- **BR-9** [Minor] `test-global-state` couchtty test init sets http.DefaultClient.Jar package-wide, sharing one cookie jar across every test

## Round 3 — 2026-10-12T14:44:53-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Body still states replaced designs (Host.Lock at line 10, Unlock-to-(session,ok) and Rotate(revoke) and Revoked() at 14-16, event: locked at 29-30, name on web links only at 37, inline codes and click rotation at 55-63, M3 click-to-rotate at 78, the draft status line at 3). The rule from the 2nd finding still applies: rewrite each body paragraph when a revision replaces it.
- BR-2 — addressed — D1 is recorded as Strict with no expiry (plan line 139), so M2's credentials change is no longer waiting on a decision.
- BR-3 — addressed — Plan Revisions line 139 records D1=Strict, superseding PQ-5. It matches lockgate.go:121, lockgate_test:156 and the atlas trade-off. The issue Log has no separate record of the operator's answer.
- BR-4 — addressed — Rotation.Gen is stamped under the mutex (castlock.go:169,217), with the concurrent test TestConcurrentRotationsNewestGenIsCurrentCode. No test covers Gen on a guess-rotation.
- BR-5 — addressed — Plan line 146 commits to landing the branch only after M3, the remedy the finding asked for. This is a prose-only change and correct.
- BR-6 — not-addressed — Revisions (lines 140-143) now describe revocation as built, but body lines 29-30 still say event: locked and line 105 the sessionFrom(r) seam. Same rule as BR-1, so fix it in the same body rewrite.
- BR-7 — addressed — FuzzUnlockBody drives the unlock HTTP handler. It ran for 10s (about 266k runs) and passed.
- BR-8 — addressed — A spent code that cannot rotate now fails closed, and a mint failure is Failed (500). Both tests went red when I reverted the fix. The rotation error still goes unreported, which is harmless on Go 1.26, where crypto/rand cannot fail.
- BR-9 — addressed — Tests use a package-level castClient in both packages, and no DefaultClient.Jar remains in cmd/. I could not run couchtty here because its generated runtimebundle assets are missing.

## Round 4 — 2026-10-12T16:13:52-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The "Current design" summary was added, but it already contradicts the latest revision (four codes vs the web cast's three links), and the Couch Display/Click rotation/Documented trade-off sections and the M3 bullets still state the replaced design.

### Raised

- **BR-10** [Important] `behavior-without-default-test` Web viewer's reload into the lock page (viewer.js:99-106) has no node test; only the env-gated Chrome test covers it
  The terminal viewer's twin is tested in connect.test.mjs. Add a viewer.test.mjs case with the fake EventSource at readyState 2 and an injected sessionLost: true calls relock, false shows Disconnected, and neither runs after an end event.
- **BR-11** [Minor] `spec-moves-with-decisions` Plan M2 says the Chrome live tests were "updated to unlock first", but they bypass the Host; TestLiveLockPage replaces them, which the plan does not record
  Repeat in this family. The rule: when an implementation substitutes for a plan item, the milestone bullet is rewritten in the same commit, not left for a Revisions entry. Same sweep: atlas/broadcast.md:82 and the Current design's pre-session list omit collab.js.
- **BR-12** [Minor] `docs-gate-surface` README web-cast sections (e.g. line 1021, "Anyone with the viewer link can point") don't mention the code lock
  Acceptable to defer while the operator can't see a code (M3), but the M3 close must update the README for codes, the lock page and the drive code.

## Round 5 — 2026-10-12T16:19:26-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — The Current-design summary helps, but the body still states replaced decisions (event locked, inline Chrome code display, click-span rotation, trade-off, M3 bullets); the M3 bullets are the next build list and still describe the old design. Strict is correct (lockgate.go:123), so that site is resolved.
- BR-10 — addressed — tests/webcast/viewer.test.mjs adds sessionLost true/false/ended/reconnecting cases; reverting viewer.js to base makes the relock case fail (1 of 14).
- BR-11 — not-addressed — The summary's Tests bullet records the substitution, but the M2 milestone bullet still says "updated to unlock first", and atlas/broadcast.md:82 plus the summary's pre-session list still omit collab.js (served pre-session at lockgate.go:37).
- BR-12 — not-addressed — Still deferred on its own terms to the M3 close; the README has no code-lock text yet.

### Raised

- **BR-13** [Minor] `behavior-without-default-test` Terminal viewer's note author (viewer.js beforeSave, collab.js storedName) has only a source-grep test, no behavioral one
  2nd in family. Rule: any page-side path that changes what a page POSTs is covered by a node test driving it through injected deps; a Go grep of the asset text is not a behavioral test. Add a node test: storedName over a fake storage (valid, invalid, missing storage) and the dialog's beforeSave adding author only when a name is stored.
- **BR-14** [Minor] `restated-constant-unpinned` The 40-char viewer-name bound is restated in collab.js, notes.go and lock.html maxlength with no parity test
  TestNoteAuthorRoundTripsOnTheBoardRule pins board.MaxNameRunes to broadcast only. If they drift, pages send names that ParseNote refuses as ErrNoteText. Pin MAX_VIEWER_NAME and maxlength from Go by reading the embedded assets, as server_test already does for keys. Note that maxlength counts UTF-16 units.
- **BR-15** [Minor] `test-asserts-weaker-than-name` TestCastBrowserFirstThenLink now passes on the lock page's 200 and no longer proves the viewer page is served
  cast_test.go:57 fetches u.Link without unlockCast, so the 200 is the lock page. Unlock first and assert the viewer page's marker, or assert the lock page explicitly and rename the test.

## Round 6 — 2026-10-12T17:25:49-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Plan now leads with a Current design section (Couch layer/tooltip/Alt+click menu/flash, SameSite=Strict at line 16) and fences the draft under an explicit superseded banner (line 32); current M3 deliverables live in the issue Plan checklist.

### Raised

- **BR-16** [Important] `unordered-async-notifications` Code-rotation ordering is keyed by (cast kind, link), so a late rotation from an ended cast poisons the next cast
  castlock fires OnRotated outside its mutex and Gen restarts at 1 per lock; onCodeRotated (console_castcodes.go:283) keys gens by linkKey(kind, link), which forgetCastCodes clears at cast end. A late callback re-inserts the old Gen, so the next cast's first guess rotations are dropped (no alert or flash), a dead code flashes, and a hovered tooltip shows the dead code. 2nd in family. Rule: order async notifications by (producer instance, Gen) and drop ones from a non-current producer. Fix: capture the cast attempt in the OnCodeRotated closures (console_broadcast.go:161, console_webcast.go:62) and check it on delivery; add a test that ends cast A, delivers A's rotation, starts B and delivers B's Gen 1.
- **BR-17** [Important] `docs-gate-surface` README never mentions the cast-link code, the lock page, the hover tooltip or Alt+click Codes menu
  README.md:865-872 still says viewers open the link and see the screen live; the pointer (950-955), web view and drive (982-995) paragraphs omit the code, and the drive link needs a 6-digit code. 2nd in family. Rule: the README gate covers changed behaviour of already-documented surfaces, not only new commands. Sweep every README paragraph describing a cast link recipient's or the operator's actions in the same range.
- **BR-18** [Important] `done-when-item-unverified` M3's operator smoke test on desktop and iPad is not recorded
  It is in the M3 checklist and Done-when, and absent from the issue Log. The iPad pass must confirm that hover and Alt+click (which need a trackpad and keyboard) can reach the codes, or record a fallback.
- **BR-19** [Minor] `spec-moves-with-decisions` Issue Spec item 6 still says clicking the code in the tab bar rotates it
  This contradicts Spec item 7 (no inline codes; the Alt+click Codes menu). 5th in family. Rule: a decision that replaces another greps every artifact restating it (issue Spec, Done-when, plan, atlas) for the old terms in the same commit.
- **BR-20** [Minor] `behavior-without-default-test` The Codes menu's "rotate all" item is never invoked by a test
  TestWebCastCodesOnTheChromeCell only asserts the item is present. 3rd in family. Rule: every Spec'd user action (each menu item) has a test that invokes it and asserts its effect, here that every link's code changes and its viewers disconnect.
- **BR-21** [Minor] `operator-affordance-discoverability` Link-copy notices do not tell the operator that viewers need a code or where to see it
  console_broadcast.go:228, console_pointer.go:130 and :148, and console_webcast.go:268 copy a now-locked link without hinting to hover for its code.

## Round 7 — 2026-10-12T21:13:35-07:00 (claude) — passed

### Disposed

- BR-16 — addressed — linkKey now includes the start attempt and onCodeRotated drops non-current attempts; reverting the guard fails TestLateRotationFromAnEndedCastIsDropped.
- BR-17 — addressed — README #495 section covers code, lock page, hover, Alt+click menu and a per-control table; README:886 and :1015 could still gain a "after the code" clause.
- BR-18 — addressed — Issue Log records the operator smoke test on desktop and iPad (build bd801297) plus the re-test of f7e6cc7a.
- BR-19 — not-addressed — Issue line 35 still says clicking the code in the tab bar rotates it; atlas/webcast.md:178 is a sibling (raised below).
- BR-20 — not-addressed — No test runs chooseCodes(t, codesAction{}); rotate all is only selected in the pure layout test.
- BR-21 — not-addressed — console_pointer.go:109/130/148, console_broadcast.go:231, console_webcast.go:285/306 still copy locked links with no code hint.

### Raised

- **BR-22** [Minor] `spec-moves-with-decisions` atlas/webcast.md:178 still says Alt+right-click on the Chrome cell copies the drive link, which this range removed
  This is the 6th finding in family spec-moves-with-decisions. Rule: when a commit replaces a binding or decision, grep the issue Spec, Done-when, plan current-design section, atlas and README for the old terms in that same commit. Here grep "Alt.*right" and "clicking the code" across atlas/, README.md, and the issue file; fix the two hits (atlas/webcast.md:178, issue line 35).
- **BR-23** [Minor] `unordered-async-notifications` onCodeRotated checks the cast attempt under c.mu but writes the flash later under codes.mu, so a cast ending in between still flashes its dead code
  This is the 3rd finding in family unordered-async-notifications. Rule: the currency check and the state write for an async notification happen in one critical section against the same state that end-of-producer clears. Fix: have forgetCastCodes record the ended attempt (or the current attempt) in castCodes and check it under codes.mu at console_castcodes.go:436.

## Round 8 — 2026-10-12T21:40:42-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — Current design line 15 states there is no event: locked; the draft is fenced as superseded (line 32), naming event: locked and sessionFrom(r) explicitly.
- BR-11 — not-addressed — The Tests bullet (line 28) records TestLiveLockPage, but collab.js is still missing from the pre-session lists at plan line 13 and atlas/broadcast.md:82, while lockgate.go:38 serves it.
- BR-12 — addressed — README.md:1071-1095 adds a code-lock section covering every link (view, pointer, drive), the lock page and URL#code, which covers the web-cast paragraph at :1051.
- BR-13 — not-addressed — storedName and beforeSave author are still pinned only by the source grep in server_test.go:167-172; no .mjs test drives them.
- BR-14 — not-addressed — lock.html:20 maxlength="40" and collab.js:96 MAX_VIEWER_NAME=40 still have no parity test against board.MaxNameRunes.
- BR-15 — not-addressed — cast_test.go:57 still GETs u.Link without unlockCast and accepts the lock page's 200 as "viewer page".
- BR-19 — addressed — Issue Spec item 6 now describes rotation from the Codes menu (Alt+click, rotate / rotate all); there is no click-to-rotate text left.
- BR-20 — addressed — console_castcodes_test.go:170-177 clicks the rotate all word through chooseCodes(t, codesAction{}) and asserts both codes change. Passes under -race.
- BR-21 — addressed — Every link-copy notice (console_broadcast.go:231, console_pointer.go:109/130/148, console_webcast.go:285/306) now says viewers need the code and where to hover, without the code itself.
- BR-22 — not-addressed — atlas/webcast.md:178-179 still says "Alt (Option)+right-click copies the drive link"; commit 276d2dd8 edited lines 160-164 instead. The test at console_castcodes_test.go:165 asserts the opposite.
- BR-23 — not-addressed — Ineffective: broadcastEnded/webcastEnded keep attempt unchanged (console_broadcast.go:268, console_webcast.go:242), so a rotation after end passes attempt==current. A probe (stop, wait for session nil, onCodeRotated with the old attempt) sets a flash for code 9999. Fix: also require session/cast non-nil under c.mu.

### Raised

- **BR-24** [Minor] `spec-moves-with-decisions` Plan Current design contradicts the closed M3: tooltip on link controls, status "M3 in progress", LinkLocks "Add"
  7th finding in this family. Plan line 23 says the hover hangs on 👆/👽, but castCellAt (console_castcodes.go:115) and atlas/webcast.md:163 put it on the LIVE/Chrome cells. Line 3 says M3 is in progress, and line 11 names Add instead of Mint. Earlier rules applied per artifact still miss sites. Rule: at a milestone close, compare the Current design summary line by line against the closing Revisions and the code, as part of the close commit.

## Round 9 — 2026-10-12T21:51:42-07:00 (claude) — passed

### Disposed

- BR-11 — not-addressed — The plan's Current design now records TestLiveLockPage, but the sweep is undone: atlas/broadcast.md:81-82 and the plan's Gate bullet still list only lock.js/lock.css, while lockgate.go:37 also serves collab.js before a session. The issue's M2 checkbox still says "headless-Chrome tests unlock first".
- BR-13 — not-addressed — server_test.go:167-169 is still a source grep. No node test drives storedName or the beforeSave in viewer.js:296.
- BR-14 — not-addressed — MAX_VIEWER_NAME = 40 (collab.js:96) and maxlength="40" (lock.html:20) are still unpinned against board.MaxNameRunes.
- BR-15 — not-addressed — cast_test.go:57 still GETs u.Link without unlockCast and passes on the lock page's 200.
- BR-22 — addressed — atlas/webcast.md:178-180 now says right-click copies the view link and 👽 or the Codes menu copies drive (#495 dropped Alt+right-click). The issue's Spec item 6 now names the Codes menu. Prose-only fix, checked against codes_menu.go and castCellAt.
- BR-23 — addressed — console_castcodes.go:435-441 adds the liveness check in the same c.mu section. The regression goes red when I revert it (TestRotationAfterTheCastEndedIsDropped: "an ended cast's rotation flashed"). Only the castScreen branch is exercised; castWeb liveness is untested.
- BR-24 — not-addressed — Plan line 3 still says "M3 in progress", line 11 still says "A new link is one Add", and line 23 still hangs the tooltip on 👆/👽, contradicting castCellAt (console_castcodes.go:115) and atlas/webcast.md. Rule: at a milestone close, diff the Current design line by line against the code and atlas in the close commit itself. That rule also covers the BR-11 sweep.

## Round 10 — 2026-10-12T21:53:44-07:00 (claude) — passed

### Disposed

- BR-11 — not-addressed — Plan line 105 still says "updated to unlock first" (only the Current design notes the swap), and atlas/broadcast.md around line 82 still lists only lock.js, lock.css and unlock before a session; lockgate.go:37 also serves collab.js.
- BR-13 — not-addressed — Only server_test.go:167 greps for "storedName(storage)". No node test drives storedName or the beforeSave author path through injected storage.
- BR-14 — not-addressed — MaxViewerNameRunes (notes.go:152), MAX_VIEWER_NAME (collab.js:96) and lock.html maxlength="40" still have no test that reads the embedded assets and checks them against Go.
- BR-15 — not-addressed — cast_test.go:57 still GETs u.Link without unlockCast and asserts only a 200, which is the lock page's.
- BR-24 — not-addressed — The plan file is unchanged since the last round. Line 3 says "M3 in progress", line 11 says "Add" (the code has Mint), and line 23 still puts the tooltip on 👆/👽.

## Open findings

- **BR-11** [Minor] `spec-moves-with-decisions` Plan M2 says the Chrome live tests were "updated to unlock first", but they bypass the Host; TestLiveLockPage replaces them, which the plan does not record
- **BR-13** [Minor] `behavior-without-default-test` Terminal viewer's note author (viewer.js beforeSave, collab.js storedName) has only a source-grep test, no behavioral one
- **BR-14** [Minor] `restated-constant-unpinned` The 40-char viewer-name bound is restated in collab.js, notes.go and lock.html maxlength with no parity test
- **BR-15** [Minor] `test-asserts-weaker-than-name` TestCastBrowserFirstThenLink now passes on the lock page's 200 and no longer proves the viewer page is served
- **BR-24** [Minor] `spec-moves-with-decisions` Plan Current design contradicts the closed M3: tooltip on link controls, status "M3 in progress", LinkLocks "Add"
