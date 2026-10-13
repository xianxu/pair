# #495 plan: a one-time code locks every cast link

Status: decided; M1 and M2 closed; M3 in progress. The body below is the original draft, and Revisions record each change. This summary is the current design.

## Current design (supersedes the draft where they differ)

- **Links and codes.** Each link token has its own code:
  - terminal `view` (4 digits) and `pointer` (4);
  - web `view` (4), web `pointer` (4) once pair:5 splits it out after #503, and web `drive` (6).
  - Replay links stay unlocked (D4, a follow-up issue).
- **Registry (M3 first step).** `broadcast.LinkLocks` is one registry per cast (label + token → lock), with `LockFor`, `Codes`, `Rotate` and `End`. A new link is one `Add`.
- **Gate.** `Mountable.LockFor(segment)` names a link's lock, and the `Host` gate runs before any mount.
  - Before a session: the lock page, `lock.js`, `lock.css` and `POST unlock`; everything else gets 401.
  - With a session: the request runs under a context that revoke cancels.
  - `GET session` is the gate's own probe (204 or 401). A page whose stream closed uses it and reloads into the lock page. There is no `event: locked`.
- **Session.** An `HttpOnly`, `SameSite=Strict`, path-scoped cookie with no expiry (D1). Pages use same-origin credentials.
- **Budget.** About 8 wrong tries per code, across all clients. A spent budget rotates the code and keeps the sessions.
  - Unlock attempts are rate-limited per lock (a burst of 5, then 1/s).
  - If the rotation fails, the lock fails closed until the operator rotates.
  - `Rotation.Gen` orders the notifications.
- **Lock page.** One page asks for the code and the name for every link. `URL#NNNN` pre-fills the code and is stripped from history.
- **Couch (M3).** An operator-only presenter layer that is never tapped holds:
  - a hover tooltip with the codes. It hangs on the link controls: LIVE's 👆 (terminal codes), and Chrome's 👆 (view) and 👽 (drive) from #501. Until #501 lands, it hangs on the Chrome cell;
  - an Alt+click Codes menu with per-link rotate and "rotate all";
  - a three-flash of a new code;
  - the guess-rotation alert, coalesced per lock every 30 s.

- **Tests.** The headless-Chrome viewer tests (`viewer_live_test` and its siblings) serve `Server` directly, not through the Host gate, so they stay unlocked. `TestLiveLockPage` covers the locked path end to end in Chrome, replacing the plan's "update them to unlock first". Node tests cover each viewer's relock on a lost session.

## Original draft (superseded)

> **Note:** Everything from here to Revisions is the draft as first proposed, kept for its history. Where it differs from **Current design** above, the summary and Revisions win; for example, it still describes inline codes, `Host.Lock`, `event: locked`, `sessionFrom(r)`, Lax cookies and a per-client budget.

## Shape

- **One lock per link, enforced at one place.**
  - `broadcast.Host` already routes every request by its link token: the first path segment, through `Mountable.Owns`, compared in constant time by `OwnsToken`.
  - The gate goes there, before any mount sees the request. A mount can't forget it, and a new route can't escape it (ARCH-SECURE: one enforcement point, not one check per handler).
  - Mounts register a lock for each link token they own: `Host.Lock(token, *castlock.Lock)`. A token with a lock is served only to a valid session. One without a lock behaves as today. Replay shares stay unlocked until **[D4]**.
- **Pure core: `cmd/internal/castlock`** (ARCH-PURE).
  - **State:** the code (digits, from crypto/rand via an injected source), its generation, the wrong-try budget, and the sessions. A session is a 32-byte token, bounded by a cap, with the oldest evicted first.
  - **Operations:**
    - `Unlock(code) → (session, ok)`: when the budget is exhausted it rotates, keeps the existing sessions, and returns a `Rotated{reason: guessing}` event.
    - `Valid(session)`.
    - `Rotate(revoke bool)`: a click rotation revokes every session and closes the generation's `Revoked()` channel.
    - `End()`: called when the cast stops (ARCH-FUNERAL: sessions end with the code that issued them, and the lock ends with its link).
  - **Ordering (ARCH-ORDER):** one mutex serializes unlock against rotation. An unlock that races a rotation is judged against the generation it lands in.
- **HTTP shell, inside the Host, for a locked token.** For a request to `/<token>/…`:
  - `GET /<token>/` with no session serves the lock page instead of the viewer.
  - `GET /<token>/lock.js` and `/lock.css` are always served. `POST /<token>/unlock` takes `{code, name?}`, sets the session, and answers `{ok}` or a refusal.
  - Everything else requires a session; without one the answer is `401`, with no body that hints at content. That includes `events`, assets, the note, stroke, point and drive routes, `palette.json`, and `assets/<hash>`.
  - Unknown tokens remain `404`.
  - Unlock attempts are also rate-limited per lock, reusing `broadcast.NewRateLimit` (ARCH-CONSTRAINTS: a bounded guessing rate on top of the wrong-try budget).
- **Session transport [D1]:** recommended is an `HttpOnly; SameSite=Strict; Path=/<token>/` cookie (`Secure` when the base is https).
  - The viewer pages change `credentials: 'omit'` to `'same-origin'`. EventSource sends same-origin cookies already.
  - The alternative is a session segment in the path (`/<token>/~<session>/`). It needs no JS change, but it puts a revocable bearer in history.
- **Revocation of live streams:**
  - Each `events` loop also selects on the lock's `Revoked()` channel for the generation it was admitted under.
  - On revoke it sends `event: locked`, then closes. The page returns to the lock screen.
  - POST routes recheck the session on every request at the gate, so a revoked session fails at once.
- **`URL#NNNN`:**
  - `lock.js` reads the fragment, removes it at once with `history.replaceState`, and submits it like typed entry.
  - The fragment never reaches the server or a link-preview bot.
- **Lock page:**
  - One standalone page with a code field.
  - On web cast links it also asks for a name **[D6]**. The name is stored under `couch.viewerName`, the key `annotate.js`'s `identityStore` already reads, so the #470 note dialog no longer asks for it (its name field remains only as a fallback when the name is empty).
  - After unlock it calls `location.replace('./')`, and the viewer page loads under the session.

## Codes

- **Terminal LIVE (view):** 4 digits.
- **Terminal pointer link [D3]:** it can post points and notes. Proposed: it gets its own 4-digit code.
- **Web cast view:** 4 digits.
- **Web cast drive:** 6 digits, and different from the view code. The view code never unlocks the drive token, because each lock is per token.
- **New cast:** fresh random codes. Stopping a cast ends its locks.
- **Wrong tries [D2]:** proposed is a budget **per code** (about 8 wrong tries in total) rather than per client.
  - Behind the tunnel there is no trustworthy client identity: an IP is the tunnel's, and cookies can be cleared.
  - A per-client budget therefore lets a guesser multiply its attempts.
  - When the budget runs out, the code rotates and Couch shows a notice: "Too many wrong codes on the LIVE link: new code 5193." Existing viewers keep their sessions.
  - A click rotation, by contrast, revokes everyone.

## Couch

- **Display [D5]:**
  - `Chrome ⏸ 4821` on the Chrome cell.
  - The drive code appears in the drive menu header, and in the notice after Alt+right-click copies the drive link. The copy is the plain link with no fragment; the operator says the code.
  - On the LIVE cell, the pointer and marks geometry (`indicator.go` `IndicatorShown`, `pointerColumn`, `StatusGuardCols`) assumes 👆 sits right after `LIVE ⏸ `. Proposed: the code goes at the end of the LIVE group (`LIVE ⏸ 👆 💭2 4821`) rather than reworking that geometry.
- **Click rotation:**
  - Clicking a code span rotates that link's code and revokes its sessions, with a notice ("LIVE code is now 7302; viewers were disconnected").
  - New span kinds go in `RenderedStatusRow`, and the hit test goes in `terminal_input.go`.
- **Guess-rotation alert:** the lock's `OnRotated` callback goes through `GoTracked(setNotice)`. The notice never includes a link or token; the code is fine to show.
- **Documented trade-off:** LIVE casts the tab bar itself, so terminal viewers can see the codes. They are already viewers.

## Milestones

- **M1: lock core and the Host gate, with the terminal LIVE link.**
  - `castlock` with unit tests: budget, rotation, revoke, session cap, and the unlock-versus-rotate ordering.
  - The Host gate, the lock page, unlock, and the cookie.
  - The LIVE link locked; its `events` loop honours revoke.
  - Tests: an `httptest` request for each route with no session gets 401; this explicitly includes a direct `GET events`, the assets, and the POST routes. Also a wrong code, budget rotation, a fragment unlock (node test of `lock.js`), and a revoke that ends a live SSE stream.
- **M2: web cast view and drive, and the pointer link.**
  - Separate locks, with the view code refused on the drive token.
  - Name on the lock page, and the note dialog stops prompting.
  - The `credentials` change.
  - Headless-Chrome live tests (`viewer_live_test`, `viewer_drive_live_test`, `viewer_annotate_live_test`) updated to unlock first.
- **M3: Couch.**
  - Code display, click-to-rotate with revoke, the guess-rotation alert, and the drive code in the drive menu.
  - Atlas: a threat-model paragraph in `atlas/broadcast.md`, linked from `atlas/webcast.md`. It protects against leaked links, previews and history; it does not protect against a viewer who has the code, a viewer re-sharing what they see, or the tab bar showing codes to LIVE viewers.
  - Operator smoke test on desktop and iPad.

## Out of scope

- Replay share links **[D4]** (proposed follow-up issue).
- Per-viewer kick.

## Revisions

- 2026-10-12, operator answers via the hub.
  - **D3:** LIVE and the terminal pointer link each get their own code.
  - **D5:** no codes inline in the tab bar.
    - Hovering a cast cell shows its codes as a tooltip: `view 4821 · pointer 7730` on LIVE, `view 5512 · drive 903114` on Chrome.
    - Rotation moves to a per-cell codes menu, with one item per link plus "rotate all".
  - **Finding sent back:** overlays, the status row and the notice line all reach the LIVE tap, so the tooltip and the menu would show codes to terminal viewers.
  - **Proposed:** an operator-only presenter layer that is never tapped, and Alt+click to open the codes menu. Awaiting confirmation.
  - D1, D2, D4 and D6 are still pending.
- 2026-10-12, operator answers via the hub (round 2).
  - **D2: yes.** One budget per code, about 8 wrong tries, then rotate and alert.
    - On any rotation, the new code flashes inline in the tab bar three times, then returns to the hover tooltip.
    - **Flag:** the flash is broadcast on LIVE. If the operator-only layer lands, the flash should draw there; otherwise terminal viewers see the pointer or drive code during the flash.
  - **D3:** confirmed.
  - **D4:** replay links are read-only and lower severity, so a follow-up issue, not now.
  - **D6:** one lock page asks for the code and the name for **every** link. Terminal notes (pointer link) gain an author from the lock page's name.
  - **D1:** on hold until the operator confirms the path-scoped HttpOnly SameSite=Strict session cookie with no expiry.
    - The gate reads the session through one seam (`sessionFrom(r)`), so the transport is one function.
    - The page-side `credentials` change waits for the confirmation.
  - **D5 follow-ups** (operator-only layer; Alt+click for the codes menu) are not answered yet. The Couch work in M3 waits for them.
  - Build order under the holds:
    - first, the `castlock` core (independent of both holds);
    - then the Host gate behind the session seam;
    - then the lock page, with the name field.
- 2026-10-12, the D5 follow-ups are answered.
  - **(a)** The operator-only presenter layer, never in the tapped frame. It carries everything that shows a code: the hover tooltip, the codes menu, and the rotation flash. The flash appears three times in that layer, not in the broadcast tab bar, which resolves the round-2 flag.
  - **(b)** Alt+click on a cast cell opens the Codes menu. The operator may change the gesture at the smoke test.
  - Only D1 (the session cookie) is still open.
- 2026-10-12, plan-quality round 1.
  - **PQ-1 (lock registration is atomic with the mount).** There is no `Host.Lock` call.
    - `Mountable` gains `LockFor(segment string) *castlock.Lock`, which the Host asks on every request after `Owns`.
    - Locks are created with their link's token, before `Host.Mount`, and live inside the mount. So the gate never sees an owned but unlocked token, and the locks end with the mount (`Unmount`, or `End()` at cast stop).
    - `nil` means unlocked. Only the replay-share mount returns it (D4 follow-up).
    - A table test asserts which mounts gate, and that the first request after `Mount` (the reachability probe) already meets the lock.
  - **PQ-2.** Spec items 4 and 7 and Done-when are rewritten to the decided answers.
  - **PQ-3 (overload, ARCH-CONSTRAINTS).**
    - Anyone holding a link can spend the budget and force rotations. That locks out new joiners until the operator says the new code. Existing sessions are unaffected.
    - The bound is the per-lock unlock rate limit (`broadcast.NewRateLimit`, about 1 attempt per second with a burst of 5), so at most one forced rotation per about 8 s. Rotation notices coalesce into one per lock per 30 s ("rotated N times").
    - This is an accepted trade-off. The operator's remedy is to stop the cast.
  - **PQ-4 (test strategy).**
    - **castlock:** a seeded random sequence of unlock (right or wrong code), rotate, revoke and validate operations, checked after every step against invariants stated independently of the implementation:
      - a revoked session is never valid;
      - a session issued under one lock never validates on another (so the view code never opens drive);
      - at most `budget` wrong tries per code generation;
      - sessions never exceed their cap.
    - **Fuzzing:** go-fuzz targets for the unlock body decoder and the code parser.
    - **Gate:** a table over every route of every mount, each requested with no session, an invalid session, and a revoked session.
  - **PQ-5.** The cookie (if D1 is confirmed) is `SameSite=Lax`, not Strict.
    - Strict is not sent on a link opened from a chat app, so a returning viewer would see the lock again.
    - Lax is safe here because every state-changing route is a POST requiring a JSON content type, and Lax cookies don't ride cross-site POSTs. To be raised with the hub as part of D1.
- 2026-10-12, M1 boundary review: the plan is aligned to what was decided and built.
  - **D1 (supersedes PQ-5's Lax).** The operator confirmed `SameSite=Strict`, with no expiry. The re-prompt after a chat-app navigation is the accepted trade-off, recorded in `atlas/broadcast.md`.
  - **Revocation, as built.**
    - The gate runs each admitted request under a context that the session's revoke cancels, so streams end with no `event: locked`.
    - A page whose stream closed probes `GET session`; on 401 it reloads into the lock page.
    - There is no `sessionFrom(r)` seam: the gate reads the cookie in one place (`lockgate.go`).
  - **Rotation order (BR-4).** Each `castlock.Rotation` carries a `Gen` that only increases. Callbacks run outside the mutex, so receivers keep the highest `Gen` (Couch, in M3).
  - **Fail closed.** A spent code whose rotation fails refuses every attempt until the operator rotates. A session-minting failure is `Failed` (500, "try again"), never "ended".
  - **Interim state.** M1 locks the terminal links, but Couch shows no code until M3. The branch lands only after M3, so no operator ever runs the interim state.
- 2026-10-12, operator via the hub: tooltip anchors.
  - #501 (pair:1) adds visible link controls to the Chrome cell: `Chrome ⏸ ↗ 👆 👽 💭N`, where 👆 copies the view link and 👽 the drive link.
  - The code tooltips hang on those controls:
    - Chrome's 👆 shows the view code; Chrome's 👽 shows the drive code;
    - LIVE's 👆 shows the terminal code.
  - M3 builds the operator-only layer and the code machinery against the current cells, then merges main for #501's cell code.
- 2026-10-12, operator via the hub: the web cast gets three links, as the terminal has.
  - The links: view-only (right-click on `Chrome ⏸`), pointer (👆), drive (👽). pair:5 builds the split after #503.
  - So the web cast gets three codes.
  - **M3 first step:** a shared `broadcast.LinkLocks` registry (label + token → lock), with `LockFor`, `Codes`, `Rotate` and `End`. Both the terminal `Session` and the web `Cast` use it, so a new link is one `Add`.
  - Coordination with pair:5 goes through the hub.
- 2026-10-12, operator smoke test of the codes: passed ("tested the code, it's working well"). Ergonomics revision from that test (via the hub), built before landing.
  - **The cast row's rule:** a left click acts or toggles, a right-click copies that control's link, and Alt+click opens the Codes menu, only on `LIVE ⏸` and `Chrome ⏸`.
  - **Hover:** it moves to `LIVE ⏸` and `Chrome ⏸` (the main entrance) and shows all of that cast's codes. The emoji show none. A new code flashes over the cell.
  - **Codes menu:** it gains copy actions, drawn in the link blue: "Copy view-only link", "Copy pointer link" (once the screen pointer link exists), and for Chrome "Copy drive link". The rotate entries keep their style.
  - **Chrome cell:** Alt+right-click no longer copies the drive link (👽 and the menu do), so a right-click on the cell copies the view-only link. `Ctrl`+click still ends the cast. The web 👆 and 👽 copy on a left or a right click.
  - **`LIVE ⏸` left click:** it stays "stop" (hub decision (a)). The screen broadcast has no pause; it is filed unassigned as #518, which notes that the ⏸ glyph promises a pause it doesn't have.
  - **#506 (pair:5)** renames `WebViewLink` to `WebPointerLink` and lands after #495, following this rule.
- 2026-10-12, operator via the hub (screenshot of the menu): the Codes menu is reorganized.
  - **Layout:** one row per link, with its label and code plain, then its actions: `view 3926 · rotate · copy-link`, `pointer 8538 · rotate · copy-link`, `rotate all`. The web cast's menu has view-only, (pointer, after #506) and drive rows, then `rotate all`. The separate "Copy … link" rows are dropped.
  - **Click targets:** each action word is its own click target, drawn in the link blue.
  - **Keyboard:** Up/Down pick the row, Left/Right the action, Enter runs it, Escape closes.
  - **Model:** `codes_menu.go` is a pure model (layout, hit-testing, selection, drawing, by grapheme width), and Couch wires it.
  - **Link not made yet:** a row whose link doesn't exist shows its label alone. See the Log.
