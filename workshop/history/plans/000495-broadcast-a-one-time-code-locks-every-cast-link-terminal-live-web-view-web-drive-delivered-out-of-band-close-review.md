# Boundary Review — pair#495 (whole-issue close)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7f9f58a6186de2c68e0be57da569358fab3bce9e..dab29e48d993290317187021ff68229ea1179cbf |
| command | sdlc close --issue 495 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T21:40:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the remaining prior-finding fixes (BR-23 ordering, the plan and README items), then I'll run the tests.

Verifying the README and test items next (BR-12, BR-13, BR-14, BR-15), then running the test suites.

castlock and broadcast pass under -race; couchtty won't build without generated assets — looking for the generator.

couchtty passes under -race. Now writing a throwaway test to check whether the BR-23 fix really drops a late rotation after a cast ends.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: FIX-THEN-SHIP.** Nothing blocks the close. Every finding is Minor, but the claimed fix for BR-23 doesn't work.

The core feature is solid. Every terminal and web link is locked at the Host gate, codes are shown only to the operator, and rotation is ordered by Gen and by the cast's start attempt. Tests pass: `castlock` and `broadcast` (`go test -race`), and the codes, rotation, pointer, broadcast and webcast tests in `couchtty` (`-race`). `couchtty` only built after I ran `make runtimebundle-generate` in the throwaway checkout.

Since round 7, BR-19, BR-20 and BR-21 are fixed, and BR-6 and BR-12 are resolved by earlier work. What's still open:

- **BR-23 fix doesn't work.** When a cast ends, `broadcastEnded` and `webcastEnded` keep the same `attempt` number. So holding `c.mu` across the check and the write changes nothing: a late rotation still passes the `attempt != current` check. I wrote a probe test to confirm it: stop the broadcast, wait until the session is nil, then deliver a rotation for the ended attempt. The test fails, and a flash for the dead code `9999` is set.
- **BR-22:** `atlas/webcast.md:178-179` still has the Alt+right-click claim; the fix edited a different paragraph.
- **Plan's Current design is stale for the M3 close.**
- **BR-11, BR-13, BR-14, BR-15** are unchanged.

### 1. Strengths
- `broadcast/linklocks.go`: one registry per cast. A link's lock is minted before its token can be routed, `LockFor` compares every token so timing doesn't reveal which one matched, and `End` is idempotent and blocks later mints.
- `lockgate.go`: there is a fixed list of assets served before a session, refusal text never echoes the request, the unlock body is capped at 256 bytes, and the session context is cancelled on revoke.
- `couchtty/console_castcodes.go`: the Codes menu is a pure layout and selection model (`layoutCodesMenu`, `WordAt`, `Move`), tested without IO in `TestCodesMenuLayoutAndSelection`.
- The codes are drawn in an operator-only presenter layer, so no code can reach a broadcast frame (`terminal/presenter.go`, `tap.go`, `overlay_test`).
- The new rotate-all test (`console_castcodes_test.go:170-177`) clicks the real menu word and checks that both codes change.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **BR-23 (not addressed):** `console_castcodes.go:431-439`. The check should ask "is this cast still live?", not "has a newer cast started?". Fix: inside the same `c.mu` section, also require `c.bcast.session != nil` (or `c.wcast.cast != nil` for web), or bump `attempt` at cast end. Add my probe as a regression test. The code comment at :427-430 currently claims protection it doesn't give.
- **BR-22 (not addressed):** `atlas/webcast.md:178-179` still says "Alt (Option)+right-click copies the drive link", but the test at `console_castcodes_test.go:165` asserts it copies the view link.
- **New:** the plan's Current design contradicts the code at M3 close:
  - Line 23 says the tooltip hangs on the 👆/👽 link controls, but `castCellAt` anchors it to the `LIVE ⏸`/`Chrome ⏸` cells, and the atlas says the same.
  - Line 3 still says "M3 in progress".
  - Line 11 says "A new link is one `Add`", but the method is `Mint`.
- BR-11, BR-13, BR-14 and BR-15 are unchanged since round 7 (details in the disposals below).

### 5. Test coverage notes
- Tests pass: `castlock` and `broadcast` (`-race`), and `couchtty -run 'Codes|Rotat|Chrome|Late|Notice|Pointer|Broadcast|Webcast'` (`-race`).
- `TestLateRotationFromAnEndedCastIsDropped` only covers a rotation tagged `current-1` while a newer cast is running. It doesn't cover a late rotation after the cast ends with no new cast started, which is exactly the BR-23 window.
- The terminal viewer's `storedName`/`beforeSave` path still has no node test.

### 6. Architecture
| Principle | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `validName`/`MAX_VIEWER_NAME` are shared through `collab.js`; only the HTML `maxlength` restates the bound (BR-14). |
| ARCH-PURE | pass | |
| ARCH-PURPOSE | pass | Every live link is locked; replay is deferred by decision D4 to #509. |
| ARCH-MOCK | pass | `FakeTunnel`; no new external dependencies. |
| ARCH-CONSTRAINTS | pass | Rate limit is a burst of 5, then 1/s; unlock body is bounded. |
| ARCH-SECURE | pass | Cookie is HttpOnly, SameSite=Strict, path-scoped; codes never appear in frames or notices. |
| ARCH-ORDER | flag | BR-23: the end-of-cast state transition doesn't change the value the async check reads. |
| ARCH-FUNERAL | pass | `LinkLocks.End` retires the locks; `forgetCastCodes` clears the per-kind maps. |

### 7. Plan revision recommendations
Rewrite Current design lines 3, 11, 13 and 23:
- the status line at close;
- `Mint`, not `Add`;
- add `collab.js` to the list of assets served before a session;
- the tooltip and Codes menu hang on the `LIVE ⏸`/`Chrome ⏸` cells, per the cast row's rule.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Current design line 15 states there is no event: locked; the draft is fenced as superseded (line 32), naming event: locked and sessionFrom(r) explicitly.
  - id: BR-11
    disposition: not-addressed
    note: |
      The Tests bullet (line 28) records TestLiveLockPage, but collab.js is still missing from the pre-session lists at plan line 13 and atlas/broadcast.md:82, while lockgate.go:38 serves it.
  - id: BR-12
    disposition: addressed
    note: |
      README.md:1071-1095 adds a code-lock section covering every link (view, pointer, drive), the lock page and URL#code, which covers the web-cast paragraph at :1051.
  - id: BR-13
    disposition: not-addressed
    note: |
      storedName and beforeSave author are still pinned only by the source grep in server_test.go:167-172; no .mjs test drives them.
  - id: BR-14
    disposition: not-addressed
    note: |
      lock.html:20 maxlength="40" and collab.js:96 MAX_VIEWER_NAME=40 still have no parity test against board.MaxNameRunes.
  - id: BR-15
    disposition: not-addressed
    note: |
      cast_test.go:57 still GETs u.Link without unlockCast and accepts the lock page's 200 as "viewer page".
  - id: BR-19
    disposition: addressed
    note: |
      Issue Spec item 6 now describes rotation from the Codes menu (Alt+click, rotate / rotate all); there is no click-to-rotate text left.
  - id: BR-20
    disposition: addressed
    note: |
      console_castcodes_test.go:170-177 clicks the rotate all word through chooseCodes(t, codesAction{}) and asserts both codes change. Passes under -race.
  - id: BR-21
    disposition: addressed
    note: |
      Every link-copy notice (console_broadcast.go:231, console_pointer.go:109/130/148, console_webcast.go:285/306) now says viewers need the code and where to hover, without the code itself.
  - id: BR-22
    disposition: not-addressed
    note: |
      atlas/webcast.md:178-179 still says "Alt (Option)+right-click copies the drive link"; commit 276d2dd8 edited lines 160-164 instead. The test at console_castcodes_test.go:165 asserts the opposite.
  - id: BR-23
    disposition: not-addressed
    note: |
      Ineffective: broadcastEnded/webcastEnded keep attempt unchanged (console_broadcast.go:268, console_webcast.go:242), so a rotation after end passes attempt==current. A probe (stop, wait for session nil, onCodeRotated with the old attempt) sets a flash for code 9999. Fix: also require session/cast non-nil under c.mu.
findings:
  - id: new
    severity: Minor
    family: spec-moves-with-decisions
    title: |
      Plan Current design contradicts the closed M3: tooltip on link controls, status "M3 in progress", LinkLocks "Add"
    detail: |
      7th finding in this family. Plan line 23 says the hover hangs on 👆/👽, but castCellAt (console_castcodes.go:115) and atlas/webcast.md:163 put it on the LIVE/Chrome cells. Line 3 says M3 is in progress, and line 11 names Add instead of Mint. Earlier rules applied per artifact still miss sites. Rule: at a milestone close, compare the Current design summary line by line against the closing Revisions and the code, as part of the close commit.
```

---

## Re-review — 2026-10-12T21:51:41-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7f9f58a6186de2c68e0be57da569358fab3bce9e..20ae28ee57f3a54b9210f45bef8974d0e524adb0 |
| command | sdlc close --issue 495 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T21:51:41-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Let me check what the last three commits changed (they're the response to the open findings).

Still verifying the BR-23 fix: I'm generating the embedded runtime bundle so the couchtty tests can build. Then I'll run the fix's regression test with and without the change.

BR-23 is confirmed: the test fails without the fix and passes with it. Next I'm checking the doc and plan findings (BR-22, BR-24, BR-11).

BR-13 and BR-14 still have no fix (the test is still only a source grep, and nothing checks the three name limits match). Now checking BR-15's test, then running the affected test packages.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is ready to ship. This round's code fix (BR-23) is done and I checked it properly. The new `live` check in `onCodeRotated` sits in the same `c.mu` section that the end-of-cast reset uses, so a rotation reported after a cast ends can no longer flash a dead code. `TestRotationAfterTheCastEndedIsDropped` fails when I revert the check (it prints `an ended cast's rotation flashed: …code:9999`) and passes with it. BR-22's atlas line is fixed too. Five open Minors still have no fix: BR-11 (its sweep part), BR-13, BR-14, BR-15 and BR-24. None of them blocks the close, but four of the five come from one habit: the close commit doesn't sweep every doc and plan site where a claim lives. I found no new findings. All tests pass: the Go packages for castlock, broadcast, webcast, couchtty and terminal, plus 187/187 node tests.

1. **Strengths**
   - `console_castcodes.go:434-441`: the currency check (same attempt and still live) and the flash write now happen in one `c.mu` critical section, with the lock order `c.mu → codes.mu` written down. `castCodesOf` takes `c.mu → lock.mu`, and `castlock` calls `OnRotated` only after it releases `l.mu` (`castlock.go:140-147`, `:220-222`). So the new `c.mu` hold inside the callback can't deadlock.
   - `linkKey(kind, attempt, link)` plus the liveness gate means old rotations can't leak into a later cast, either across casts (BR-16) or after one ends (BR-23).
   - The regression test checks its own precondition: it fails if stopping the cast bumped the attempt, so it really tests the case where the attempt still matches.
   - The Spec's item 6 and atlas `webcast.md:178-180` now describe the Codes-menu and right-click rules that shipped.

2. **Critical**: none.

3. **Important**: none.

4. **Minor**: all of these are still-open prior findings, listed in the block below.

5. **Test coverage**
   - The BR-23 regression only drives `castScreen`. The `castWeb` liveness branch (`c.wcast.cast != nil && phase == webcastRunning`) has no test. A copy of the same test using the web-cast fixture would close it.
   - The BR-13, BR-14 and BR-15 test gaps are still open.

6. **Architecture**
   - **ARCH-ORDER: pass for this round.** The async-notification rule from BR-23 now holds at the one site where it applied.
   - **ARCH-DRY, ARCH-PURE, ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-FUNERAL: pass.** The last three commits don't change anything these cover. `forgetCastCodes` still prunes the per-kind maps, so nothing new grows without a bound.
   - **ARCH-PURPOSE: flag (docs only).** The close commit fixed the sites the findings named but didn't sweep their siblings (BR-11, BR-24).
   - ARCH-DRY's restated 40-character name limit stays open as BR-14.

7. **Plan revisions needed** (in the plan's Current design section)
   - **Line 3:** replace "M3 in progress" with "closed".
   - **Line 11:** `Add` → `Mint`.
   - **Line 23:** the tooltip and the Alt+click menu hang on the `LIVE ⏸` / `Chrome ⏸` cells, not on 👆/👽.
   - **Gate bullet:** the lock page's pre-session files should list `collab.js`.

```findings
dispose:
  - id: BR-11
    disposition: not-addressed
    note: |
      The plan's Current design now records TestLiveLockPage, but the sweep is undone: atlas/broadcast.md:81-82 and the plan's Gate bullet still list only lock.js/lock.css, while lockgate.go:37 also serves collab.js before a session. The issue's M2 checkbox still says "headless-Chrome tests unlock first".
  - id: BR-13
    disposition: not-addressed
    note: |
      server_test.go:167-169 is still a source grep. No node test drives storedName or the beforeSave in viewer.js:296.
  - id: BR-14
    disposition: not-addressed
    note: |
      MAX_VIEWER_NAME = 40 (collab.js:96) and maxlength="40" (lock.html:20) are still unpinned against board.MaxNameRunes.
  - id: BR-15
    disposition: not-addressed
    note: |
      cast_test.go:57 still GETs u.Link without unlockCast and passes on the lock page's 200.
  - id: BR-22
    disposition: addressed
    note: |
      atlas/webcast.md:178-180 now says right-click copies the view link and 👽 or the Codes menu copies drive (#495 dropped Alt+right-click). The issue's Spec item 6 now names the Codes menu. Prose-only fix, checked against codes_menu.go and castCellAt.
  - id: BR-23
    disposition: addressed
    note: |
      console_castcodes.go:435-441 adds the liveness check in the same c.mu section. The regression goes red when I revert it (TestRotationAfterTheCastEndedIsDropped: "an ended cast's rotation flashed"). Only the castScreen branch is exercised; castWeb liveness is untested.
  - id: BR-24
    disposition: not-addressed
    note: |
      Plan line 3 still says "M3 in progress", line 11 still says "A new link is one Add", and line 23 still hangs the tooltip on 👆/👽, contradicting castCellAt (console_castcodes.go:115) and atlas/webcast.md. Rule: at a milestone close, diff the Current design line by line against the code and atlas in the close commit itself. That rule also covers the BR-11 sweep.
```

---

## Re-review — 2026-10-12T21:53:44-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7f9f58a6186de2c68e0be57da569358fab3bce9e..20ae28ee57f3a54b9210f45bef8974d0e524adb0 |
| command | sdlc close --issue 495 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T21:53:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the last two open findings (BR-15 and the plan's Current design) against the code, then I'll write up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's purpose is delivered. Every cast link (terminal LIVE, terminal pointer, web view, web drive) is locked by its own code and checked by the `Host` gate before any mount sees a request. The operator sees the codes only on the couch (Couch is the operator's screen), through a presenter layer the broadcast tap never sees, plus a tooltip and the Alt+click Codes menu. This round's code change was the BR-23 fix. It now checks the start attempt and whether the cast is running in the same `c.mu` section (`console_castcodes.go`), and a sequence test comes with it. The `broadcast`, `castlock` and `webcast` test packages all pass. `couchtty` would not build in this checkout: `runtimebundle/embed.go:8` needs `assets/runtime/files`, a generated asset this disposable checkout lacks. So the BR-23 regression test was not run here. That is a checkout setup problem, not a code finding.

Nothing blocks SHIP. The five open Minors are all still open, and nothing in the range touched them after the last round: three are stale docs and tests, two are gaps between JS and Go.

1. **Strengths**
   - `castlock` is pure, and the order of unlocks and rotations is fixed by one mutex plus `Rotation.Gen`. That closed the earlier async-ordering findings, and it holds up.
   - The Host gate has a single allowlist of what may be fetched before a session (`lockgate.go:35-38`, which includes `collab.js` with a reason). Everything else gets 401.
   - `LinkLocks` is a small, stable API: `Mint`, `LockFor`, `Codes`, `Rotate`, `End` (`linklocks.go:29-103`).
   - The BR-23 fix checks "attempt matches" and "cast is running" together under the same mutex, which is the right shape.

2. **Critical:** none.

3. **Important:** none new.

4. **Minor:** none new. All five open findings (BR-11, 13, 14, 15, 24) carry over unchanged; see the dispose block.

5. **Test coverage**
   - The viewer's name path, which decides what a page POSTs, is still covered only by a grep test (`server_test.go:167`). That is BR-13.
   - The 40-character name bound exists in three places with no test that they match (BR-14):
     - Go: `notes.go:152`
     - JS: `collab.js:96`
     - HTML: `lock.html:20`
   - `TestCastBrowserFirstThenLink` asserts only a 200, which is the lock page's 200 (BR-15).

6. **Architecture**
   - **ARCH-DRY:** flag, via the name bound restated in three places (BR-14).
   - **ARCH-PURE:** pass. `castlock` and `LinkLocks` are pure; the gate is thin.
   - **ARCH-PURPOSE:** pass. Every live link derives its lock from `LinkLocks`; replay links are explicitly deferred (D4, pair#509).
   - **ARCH-MOCK:** pass. `FakeTunnel` is used, and the gate is tested through `httptest`.
   - **ARCH-CONSTRAINTS:** pass. Unlocks are rate-limited per lock (a burst of 5, then 1/s) and the try budget is bounded.
   - **ARCH-SECURE:** pass. The cookie is HttpOnly, path-scoped and Strict (decision D1). Codes never appear in copy notices or the tap. The unlock fuzz target is in place.
   - **ARCH-ORDER:** pass. Rotation order is set by Gen, and stale late rotations are dropped (BR-16 and BR-23, both with sequence tests).
   - **ARCH-FUNERAL:** pass. `LinkLocks.End` ends the registry with the cast, and sessions live in memory with no expiry beyond the cast.

7. **Plan revisions needed**
   - Line 3: change "M3 in progress" to closed.
   - Line 11: change `Add` to `Mint`.
   - Line 23: the tooltip hangs on the LIVE and Chrome cells (`castCellAt`), not on 👆/👽.
   - Line 105: rewrite the M2 bullet to say `TestLiveLockPage` replaced "update them to unlock first".
   - The M3 bullet "click-to-rotate" should become the Alt+click Codes menu.
   - atlas/broadcast.md around line 82: add `collab.js` to the list of what may be fetched before a session.

```findings
dispose:
  - id: BR-11
    disposition: not-addressed
    note: |
      Plan line 105 still says "updated to unlock first" (only the Current design notes the swap), and atlas/broadcast.md around line 82 still lists only lock.js, lock.css and unlock before a session; lockgate.go:37 also serves collab.js.
  - id: BR-13
    disposition: not-addressed
    note: |
      Only server_test.go:167 greps for "storedName(storage)". No node test drives storedName or the beforeSave author path through injected storage.
  - id: BR-14
    disposition: not-addressed
    note: |
      MaxViewerNameRunes (notes.go:152), MAX_VIEWER_NAME (collab.js:96) and lock.html maxlength="40" still have no test that reads the embedded assets and checks them against Go.
  - id: BR-15
    disposition: not-addressed
    note: |
      cast_test.go:57 still GETs u.Link without unlockCast and asserts only a 200, which is the lock page's.
  - id: BR-24
    disposition: not-addressed
    note: |
      The plan file is unchanged since the last round. Line 3 says "M3 in progress", line 11 says "Add" (the code has Mint), and line 23 still puts the tooltip on 👆/👽.
```
