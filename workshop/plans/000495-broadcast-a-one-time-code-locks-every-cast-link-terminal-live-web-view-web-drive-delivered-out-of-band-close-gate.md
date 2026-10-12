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

## Open findings

- **BR-1** [Minor] `spec-moves-with-decisions` Durable plan body still shows the replaced inline-code, click-span, and Strict-cookie design
- **BR-6** [Minor] `spec-moves-with-decisions` The plan still says revoke sends event: locked and that a sessionFrom(r) seam exists; the code does neither
- **BR-11** [Minor] `spec-moves-with-decisions` Plan M2 says the Chrome live tests were "updated to unlock first", but they bypass the Host; TestLiveLockPage replaces them, which the plan does not record
- **BR-12** [Minor] `docs-gate-surface` README web-cast sections (e.g. line 1021, "Anyone with the viewer link can point") don't mention the code lock
- **BR-13** [Minor] `behavior-without-default-test` Terminal viewer's note author (viewer.js beforeSave, collab.js storedName) has only a source-grep test, no behavioral one
- **BR-14** [Minor] `restated-constant-unpinned` The 40-char viewer-name bound is restated in collab.js, notes.go and lock.html maxlength with no parity test
- **BR-15** [Minor] `test-asserts-weaker-than-name` TestCastBrowserFirstThenLink now passes on the lock page's 200 and no longer proves the viewer page is served
