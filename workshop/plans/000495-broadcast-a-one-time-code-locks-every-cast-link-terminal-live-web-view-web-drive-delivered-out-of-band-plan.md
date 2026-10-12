# #495 plan: a one-time code locks every cast link

Status: draft. The decisions marked **[D1]..[D6]** are proposals waiting on the operator (through the hub).

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
