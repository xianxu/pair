# Boundary Review — pair#460 (whole-issue close)

| field | value |
|-------|-------|
| issue | 460 — broadcast-web: live viewer — sandboxed, script-free rebuild, scaled, join mid-cast |
| repo | pair |
| issue file | workshop/issues/000460-broadcast-web-live-viewer-sandboxed-script-free-rebuild-scaled-join-mid-cast.md |
| boundary | whole-issue close |
| milestone | — |
| window | f1fbac36ac8eb920629962f2152fb781634f37e5..a1ec873906fcd4ba45085af9791d2f7813fdce2c |
| command | sdlc close --issue 460 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:42:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the #460 diff: I've read the Go server/hub and the viewer JS, and I'm running the test suites next.

Tests pass; now checking how catch-up treats hover/pointer records and the atlas docs.

Wrapping up: confirming test coverage for hover/resync before I write findings.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The window delivers the plan. There are three new JS modules: `model.js` (a pure port of the reducer and canonical), `css.js` (pure URL and CSS rewriting) and `render.js` (the sandboxed rebuild). The `viewer.js` shell has injected dependencies, and the server gains `rec` events and the presence hook. `FakeHub` has been promoted to `recording.Hub` and the atlas is updated. `go test ./cmd/internal/webcast/...` passes, and so do all four node suites (30 tests). I could not reproduce the headless Chrome evidence: in this environment Chrome dies at startup ("GPU process isn't usable"), with or without the sandbox, and the older `TestLive*` tests fail the same way. So the render-tolerance numbers rest on the plan's log, and I have read but not run `replay_test.go` and `harness.js`. Two cheap fixes should land before close, and nothing here is Critical:
- **Stream end on a closed fan-out:** the events route treats a closed fan-out like a dropped slow viewer, which can start a reconnect loop that resends the whole catch-up.
- **Hover refusal:** the JS model refuses a `hover` for an unknown node, and that forces a full resync for a record that is only cosmetic.

**Strengths**
1. Every JS consumer derives from the Go reducer. `tests/webcast/model.test.mjs:18` checks the port against every fixture's pinned SHA. `render.js` `canonical()` reads structure from the drawn DOM rather than the model, so the conformance check tests the rebuild, not the model twice.
2. Neutralizing goes beyond the sandbox:
   - the `DROP` set (`http-equiv`, `srcdoc`, `action`, `ping`, `data` …) at `render.js:35`;
   - `<base>`, `<link>`, `<iframe>` `src` and `sandbox`, and `on*` attributes are never set;
   - `script-src` stays `'self'`, and `server_test.go` checks that.
3. Pure core with a thin shell: `connect(deps)` in `viewer.js:55` takes `open`, `schedule` and `retry` as injected dependencies. That makes reconnect catch-up, resync and end testable in node with a fake EventSource. `fit`, `toPage` and `fromPage` are pure.
4. The server tests use the real Hub, not a mock. They cover join mid-cast at three cut points, plus presence join/leave and a malformed id refused before anything opens (`server_test.go`).
5. Lifecycle is handled: `blob:` stylesheets are revoked on replace and remove, by `forget`, and by `revokeAll` on a fresh catch-up. The Hub is bounded by `CatchUp` at each `seg`.

**Critical findings:** none.

**Important findings**
- **`cmd/internal/webcast/server.go:149`: a closed `recs` channel always ends the stream** (ARCH-ORDER). That one signal covers two different events: "this viewer was too slow, so reconnect" and "the fan-out is closed for good". If the Hub closes while the Browser is still live, the viewer's EventSource reconnects by itself. `Hub.Subscribe` then sends the full catch-up and closes again, so the viewer loops every few seconds and rebuilds the page from `head` each time. That happens if #458 closes it on a recorder failure, or before the browser end arrives during shutdown. `TestServerEndsStreamWhenRecordsClose` pins this behaviour. Fix: give the `Live` seam a way to tell "closed" from "dropped" (for example `Done()`, or a `Closed()` on the Hub). On a closed fan-out, set `recs = nil` and keep streaming `state`/`end`, and add a test for that.
- **`cmd/internal/webcast/web/model.js:241`: `hover` naming an unknown node throws, and the shell turns that into a full resync** (ARCH-ORDER / ARCH-PURPOSE). Hover is outside `Canonical`, and the Go reducer ignores the kind, so this refusal is stricter than the Go side. A hovered node the recording never holds would make every viewer reconnect and re-receive the catch-up on each hover record. This log already notes that the blog's `astro-dev-toolbar` shadow root is recorded empty, and a `hover` emitted after the node's `rm` is another way to get there. Fix: treat an unknown hover id as no hover (`hover = 0`) instead of refusing, and update `model.test.mjs:91`.

**Minor findings**
- `render.js:31`: `URL_ATTRS.href` lists `'feimage'`, but SVG's `localName` is `feImage` (case-sensitive), so `<feImage href>` skips `resolveAsset`. The CSP `img-src 'self'` still blocks it (ARCH-SECURE).
- `css.js`: `image-set("a.png" 1x)` and `-webkit-image-set` with bare strings are not rewritten. This loses fidelity only; CSP blocks the fetch.
- `render.js`: `textFor`, owned `drawSheetOwner` and `fillSheet` resolve CSS against `model.url`, not `docURL(el.ownerDocument)`. Relative `url()`s in nested-frame styles therefore resolve against the top page. `<link>` already uses `docURL`.
- `server.go:153`: each record is encoded once per viewer, not once and shared as the plan's Architecture says. Also, an `Encode` error is dropped silently with `continue` (ARCH-CONSTRAINTS).
- `recording/hub.go`: `ptr` and `hover` records pile up in the Hub log for the whole segment and are all replayed in every catch-up. The ~1.4 MB/min bound was measured before `ptr` existed (ARCH-FUNERAL / ARCH-CONSTRAINTS). Consider having `CatchUp` keep only the latest `ptr`/`hover`.
- `viewer.js:144,157`: the asset-URL lambda is written twice in `mount` (ARCH-DRY).
- `model.test.mjs:36`: this re-implements `CatchUp` in JS for the join-cut test. It can drift from Go's `CatchUp`, which `harness.js` gets from the server.

**Test coverage notes**
- The node and Go tests pass here.
- The `WEBCAST_LIVE_CHROME=1` replay, join-mid-cast and viewer-page tests could not run in this environment (Chrome GPU fatal at startup, an environment problem shared with older tests).
- The test design looks sound:
  - canonical is compared with `recording.Reduce` at every pin;
  - the join-at-every-pin test uses the server's catch-up;
  - the 3% tolerance comes with a stated pixel threshold;
  - the observer-lag allowance is narrow (it only retries at +150 ms) and is documented to go to 0.
- Nothing covers a closed fan-out with a live browser, or a hover on an unrecorded node.

**Architecture markers**
- ARCH-DRY: minor flag (the asset lambda; the test-side `CatchUp`).
- ARCH-PURE: pass.
- ARCH-PURPOSE: pass. The iPad smoke test is moved out through a TL Revision and is a separable operator step, not the point of the issue. The hover finding is filed under ARCH-ORDER.
- ARCH-MOCK: pass. The Hub is its own double, and the browser tests are opt-in and headless.
- ARCH-CONSTRAINTS: minor flag (per-viewer encode; `ptr` growth).
- ARCH-SECURE: pass with a minor flag (`feImage`). The viewer id is validated by regex, and the CSP plus the sandbox keep scripts out.
- ARCH-ORDER: flagged (closed-versus-dropped conflated; cosmetic records forcing a resync).
- ARCH-FUNERAL: pass with a minor note (`ptr` within a segment).

**Architectural notes for upcoming work**
- #458 owns `Hub.Close` timing. Settle the close-versus-drop signal on `recording.Live` now, while the seam is still easy to change.
- #467 can use `onRecord` without touching the apply path.

**Plan revision recommendations**
- In the Architecture section, the Go-side line "each record encoded once (`recording.Encode`) and shared by every viewer" should match the code, which encodes per viewer in `Server.events`. Either change the code or add a Revision.
- The plan's CSP line (`img-src 'self' blob:`, `font-src 'self'`) leaves out the `data:` sources the code allows. Note this in a Revision.

```findings
findings:
  - id: new
    severity: Important
    family: closed-vs-dropped-conflated
    title: |
      events route ends the stream on any closed recs channel, so a closed Hub with a live browser loops reconnect plus full catch-up
    detail: |
      server.go:149 returns on !open. Hub.Subscribe after Close sends the catch-up and then closes, so EventSource reconnects forever, rebuilding from head each time. Distinguish close from drop on the Live seam (Done/Closed); on close set recs=nil and keep serving state/end. TestServerEndsStreamWhenRecordsClose pins the current behaviour.
  - id: new
    severity: Important
    family: cosmetic-record-forces-resync
    title: |
      model.js refuses hover with an unknown id, forcing every viewer to resync on a record outside Canonical
    detail: |
      model.js:241 throws, and viewer.js resync then closes, reconnects and replays the catch-up. The Go reducer ignores hover. An unrecorded node (the astro-dev-toolbar shadow root, recorded empty) or a hover after rm would trigger it repeatedly. Treat an unknown id as hover=0.
  - id: new
    severity: Minor
    family: url-attr-coverage
    title: |
      URL_ATTRS lists 'feimage' but SVG localName is 'feImage', so feImage href bypasses resolveAsset
    detail: |
      render.js:31. The CSP img-src 'self' backstops it. Also: css.js does not rewrite image-set()/-webkit-image-set bare strings.
  - id: new
    severity: Minor
    family: css-base-url
    title: |
      style text, owned sheets and constructed sheets resolve against model.url, not the owning document's URL
    detail: |
      render.js textFor, drawSheetOwner (owned) and fillSheet use this.model.url, while drawAttr and link href use docURL(el.ownerDocument). Nested-frame relative url()s resolve against the top page.
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan says each record is encoded once and shared; Server encodes per viewer and silently skips Encode errors
    detail: |
      server.go:153. Either share the encoded bytes from the Hub or add a plan Revision; CSP data: sources are also undeclared in the plan.
  - id: new
    severity: Minor
    family: superseded-records-retained
    title: |
      ptr/hover records accumulate in the Hub log for the whole segment and are all replayed in every catch-up
    detail: |
      The ~1.4 MB/min bound predates ptr. CatchUp could keep only the latest ptr/hover.
  - id: new
    severity: Minor
    family: duplicated-helper
    title: |
      viewer.js mount builds the asset-URL resolver twice; model.test.mjs re-implements CatchUp in JS
```

---

## Re-review — 2026-10-12T02:11:50-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 460 — broadcast-web: live viewer — sandboxed, script-free rebuild, scaled, join mid-cast |
| repo | pair |
| issue file | workshop/issues/000460-broadcast-web-live-viewer-sandboxed-script-free-rebuild-scaled-join-mid-cast.md |
| boundary | whole-issue close |
| milestone | — |
| window | 002c1dc018823075fbcb7dfcf17775e05aa7c50f..655e97c366e7fd99cc1f4ec8d28cedd6f6b0ff4f |
| command | sdlc close --issue 460 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:11:50-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on the round-1 fix commit first.

I'm checking whether the BR-1 and BR-2 regression tests fail when the fixes are reverted. The Go run with the fix reverted seems to hang rather than fail, so I'm looking at its output now.

I've checked the prior-round fixes and the reducer parity. Last step is checking that the atlas covers the new `Done()` seam, then I'll write up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The round-1 fixes work. **BR-1** gives `recording.Live` a `Done()` channel, so the events route can tell a dropped viewer from a fan-out that has ended. **BR-2** makes a hover on an unknown node clear the hover instead of forcing a resync. Both fixes have regression tests, and I reverted each in the disposable checkout to confirm the tests go red:
- With the BR-2 fix reverted, the node tests fail (2 failures).
- With the BR-1 fix reverted, `TestServerKeepsStreamWhenRecordsEnd` hangs until the 10-minute `go test` timeout instead of failing cleanly. That is a test-quality problem, listed below.

At head, `go test ./cmd/internal/webcast/...` and `node --test tests/webcast/*.test.mjs` pass (33/33). `./cmd/internal/artifactpath` fails, but only because the `runtimebundle` embed assets are missing from this checkout. Those files are outside this diff, so the failure comes from the environment, not this change. Two things stop a plain SHIP. First, `atlas/webcast.md` still describes the BR-1 behaviour this round replaced. Second, the drop-vs-close distinction is still inferred after the fact rather than reported by the Hub, which leaves a small race.

**1. Strengths**
- `cmd/internal/webcast/server.go:151-160`: when the fan-out has ended, the route sets `recs = nil` and keeps serving `state`/`end`. That breaks the reconnect-plus-full-catch-up loop with very little code.
- `recording/contract.go:169-185`: `RunLiveContract` now checks `Done` on both endings (dropped and closed). Any other `Live` implementation inherits that check.
- `server_test.go:217-226`: `stubLive` lets the test trigger a drop and a close separately, so it covers both orderings, not just whichever one a run happens to produce.
- `viewer.js:140-160`: one `resolveAsset` now feeds both the renderer and `couchViewer`, removing the copy BR-7 named.
- The Go-vs-JS refusal rules for `focus`, `click` and `selection` match the Go reducer (`reduce.go:390-415`). Hover was the only record that is drawn but not in the canonical state and still refused, so the `cosmetic-record-forces-resync` class is fully covered.

**2. Critical findings:** none.

**3. Important findings**
- `atlas/webcast.md:75-77` still says "A closed record channel (a viewer too slow to keep up, or the fan-out closing) ends the stream, and the viewer reconnects". That is the behaviour BR-1 removed.
  - **This is the 2nd finding in family `plan-code-drift`.** Rule: when a review-fix round changes a contract, the same commit updates every document that restates it (atlas route description, seam table, plan Revisions). Grep for the old wording rather than fixing only the plan.
  - Also add `Done` to the `recording.Live` row at `atlas/webcast-format.md:264`.

**4. Minor findings**
- **Drop-then-close race.** If the Hub drops a slow viewer and `Close()` runs before `events` reads the closed channel, the `select` on `recsDone` takes the "ended" branch. That viewer keeps an incomplete drawing and never reconnects. **2nd finding in family `closed-vs-dropped-conflated`.** Rule: the source reports why a subscription ended; nobody infers it by sampling `Done` afterwards. Fix: have the Hub record per subscription that it was dropped (for example a `dropped` flag set under the lock in `Put`), or close drops through a separate signal.
- **Test hangs instead of failing.** The loop in `TestServerKeepsStreamWhenRecordsEnd` (`server_test.go` ~266) spins forever when `s.next()` returns `""` at end of file. It should call `t.Fatal` when `ev == ""`.
- **Stale bound in a comment.** The `hub.go` doc comment still claims the ~1.4 MB/min bound, which was measured before `ptr` records existed. It should point to the deferral now recorded in the plan.

**5. Test coverage notes**
- BR-1 and BR-2 both have tests that fail without the fix (verified by revert).
- The feImage lower-casing (BR-3) and the per-document CSS base (BR-4) have no tests. Nothing in `tests/webcast`, `replay_test.go` or the harness has a nested frame with relative `url()`s or an SVG `feImage`.
- `model.test.mjs:27` still re-implements `CatchUp` in JS. The replay harness already gets the Go `CatchUp` through `/fx/.../catchup`, so the Go version could be served to it instead.

**6. Architectural notes**
- **ARCH-DRY:** flag (Minor). The JS `catchUp` copy (BR-7 remainder) is still there; the resolver duplication is fixed.
- **ARCH-PURE:** pass. The decisions live in pure `model.js`, and `server.go` stays thin.
- **ARCH-PURPOSE:** pass. The hover sweep covers its whole class (see Strengths).
- **ARCH-MOCK:** pass. `Hub` serves as its own double, `stubLive` sits behind the same seam, and `RunLiveContract` checks it.
- **ARCH-CONSTRAINTS:** pass. Encoding per viewer is justified in the plan revision.
- **ARCH-SECURE:** pass. The CSP `data:` source is now declared, and `img-src 'self'` backstops `image-set()`.
- **ARCH-ORDER:** flag (Minor). The drop-then-close race above: why a subscription ended is inferred from when a second channel closed, rather than carried by the subscription itself.
- **ARCH-FUNERAL:** flag (Minor, deferred). `ptr`/`hover` records pile up within a segment. The deferral to #457/#458 is recorded, but the Hub comment still claims the old bound.

**7. Plan revision recommendations**
- In the 2026-10-12 revision, note that the atlas events-route text and the `recording.Live` seam row change along with `Done()`. If the drop-then-close race is left open, record it explicitly as accepted.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Live.Done plus recs=nil in server.go:151; TestServerKeepsStreamWhenRecordsEnd goes red (hangs to timeout) with the fix reverted.
  - id: BR-2
    disposition: addressed
    note: |
      model.js hover clears on unknown id; model.test.mjs and viewer.test.mjs both fail with the fix reverted.
  - id: BR-3
    disposition: not-addressed
    note: |
      isURLAttr lower-cases the tag (render.js:597) but there is no test that fails without it; image-set deferral is in the plan revision.
  - id: BR-4
    disposition: not-addressed
    note: |
      docURL(ownerDocument) now threaded through textFor/drawSheetOwner/fillSheet, but no nested-frame relative url() test pins it.
  - id: BR-5
    disposition: addressed
    note: |
      Plan Revision 2026-10-12 records per-viewer Encode and the CSP data: sources.
  - id: BR-6
    disposition: addressed
    note: |
      Recorded as a deferral to the format owners (#457/#458) in the plan revision; the hub.go bound comment is still stale (see new Minor).
  - id: BR-7
    disposition: not-addressed
    note: |
      Resolver deduplicated in viewer.js; model.test.mjs:27 still re-implements CatchUp in JS.
findings:
  - id: new
    severity: Important
    family: plan-code-drift
    title: |
      atlas/webcast.md:75 still says a closed record channel always ends the stream and the viewer reconnects
    detail: |
      2nd in family plan-code-drift. Rule: a contract change made in a fix round updates every document that restates it in the same commit. Also add Done to the recording.Live row at atlas/webcast-format.md:264.
  - id: new
    severity: Minor
    family: closed-vs-dropped-conflated
    title: |
      A viewer dropped just before Hub.Close is read as ended and never reconnects
    detail: |
      2nd in family closed-vs-dropped-conflated. server.go infers drop vs close by sampling Done after the channel closes. Rule: the source reports why a subscription ended (for example a per-subscription dropped flag set under the Hub lock).
  - id: new
    severity: Minor
    family: test-hangs-instead-of-failing
    title: |
      TestServerKeepsStreamWhenRecordsEnd loops forever when the stream reaches EOF
    detail: |
      s.next() returns "" at EOF and the for-loop never exits; with the BR-1 fix reverted the test hangs to the 10-minute go test timeout. Fail when ev is empty.
  - id: new
    severity: Minor
    family: superseded-records-retained
    title: |
      hub.go doc still claims the 1.4 MB/min bound that predates ptr/hover accumulation
    detail: |
      2nd in family superseded-records-retained. The comment should point to the deferral recorded in the plan rather than restate a bound that no longer holds.
```

---

## Re-review — 2026-10-12T02:17:32-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 460 — broadcast-web: live viewer — sandboxed, script-free rebuild, scaled, join mid-cast |
| repo | pair |
| issue file | workshop/issues/000460-broadcast-web-live-viewer-sandboxed-script-free-rebuild-scaled-join-mid-cast.md |
| boundary | whole-issue close |
| milestone | — |
| window | 002c1dc018823075fbcb7dfcf17775e05aa7c50f..38b1d6ea1046a7d59ba6e2aa894afe0d4d33ff70 |
| command | sdlc close --issue 460 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:17:32-07:00 |
| verdict | SHIP |

## Review

Let me see the last round's commit, which disposes the open findings.

I've confirmed the fixes for BR-3/4/7/8/9/10/11 by reading the code. Next I'll check one stale plan line and run the test suites.

The Chrome-gated tests fail at startup here. Checking whether that's the environment or the code.

Chrome's GPU process crashes on this host, which is an environment problem rather than a code one. I'll do the Go revert check for BR-9 instead.

```verdict
verdict: SHIP
confidence: medium
```

Round 2 fixes all seven open findings, and I checked each against the code. The main defect class, telling a dropped viewer from an ended fan-out, is now fixed at its source. `Live.Ended(recs)` replaces `Done()`, and the Hub records each drop under its lock (`hub.go:45-49`), so the server no longer has to guess from timing. I put the old behaviour back temporarily (`Ended` returning only `h.closed`): `TestHubMeetsTheLiveContract` failed with "a subscription dropped before Close reads as the fan-out's end", and passed again once restored. The atlas restatements of the events route and the seams row now match the new contract, and no `Live.Done` reference is left outside the dated plan log.

All Go webcast packages and all 32 Node tests pass. I could not run the three headless-Chrome tests (`WEBCAST_LIVE_CHROME=1`): Chrome's GPU process crashes on this host ("GPU process isn't usable"), even when launched directly, so this is the environment and not the code. For BR-3 and BR-4 I therefore judged by reading the test: without each fix, `TestLiveRenderResolvesPerDocument` would produce a different string than it expects. Nothing blocks SHIP. One Minor plan-drift line remains.

1. **Strengths**
   - `recording/hub.go:95-101`: `Ended` is answered by the code that closed the channel, under the same lock. The contract (`contract.go:218-243`) checks both orders: drop then close, and close alone.
   - `server.go:151-157`: the route asks `s.live.Ended(sub)` instead of a `select` on a second channel, which removes the timing window.
   - `replay_test.go:421-478`: a synthetic recording pins both URL bugs from BR-3 and BR-4 (a frame's `<style>` resolving against the frame's URL, and the mixed-case `feImage`). These are behaviour checks, not wording checks.
   - The fix for the URL bug class is general: `isURLAttr` lowercases the tag (`render.js:595-597`), and every sheet path now goes through `docURL(...)` (`render.js:287, 369, 378, 409`).
   - `server_test.go:265-274`: hitting EOF now fails the test immediately instead of hanging until the timeout.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Plan line 86 (Lifecycle) still says the Hub log is bounded at "~1.4 MB/min on the stress page". That is the same bound BR-11 removed from `hub.go`. This is the 3rd finding in family `plan-code-drift`; see the findings block for the rule.
   - `drawFont` (`render.js:433`) resolves against `model.url`. That is correct today, because fonts are drawn only into `topDoc`, but it is the only resolver that does not go through `docURL`. If fonts ever become per-document, it will be wrong.

5. **Test coverage**
   - With the JS copy of `catchUp` removed (BR-7), joining mid-cast is checked only by `TestLiveReplayJoinMidCast`, which runs only with `WEBCAST_LIVE_CHROME=1`. A default run therefore no longer checks that `model.js` reduces a real catch-up to the same canonical state.
   - Suggestion: have Go write each fixture's `CatchUp` output at every cut as testdata, and have the Node test reduce it. Go's `CatchUp` stays the only implementation, and the check runs by default.

6. **Architecture**
   - **ARCH-DRY: pass.** The asset resolver is built once (`viewer.js:141`), and the JS `CatchUp` copy is gone.
   - **ARCH-PURE: pass.** `model.js` and `css.js` are pure and tested in Node; the renderer and shell are the IO layer.
   - **ARCH-PURPOSE: pass.** `image-set()` and latest-only `ptr`/`hover` are deferred, and the plan records both for the team lead.
   - **ARCH-MOCK: pass.** The Hub serves as its own test double, and the Live contract runs against it.
   - **ARCH-CONSTRAINTS: pass, with a note.** Hub memory within a segment grows with `ptr`/`hover` records; this is the recorded deferral.
   - **ARCH-SECURE: pass.** URLs with no asset are blocked, and the CSP `img-src 'self'` backstops `image-set()`.
   - **ARCH-ORDER: pass.** The reason a subscription closed is now owned by one component, which closes the old drop-versus-close race. The `dropped` map is pruned on cancel, and the server's deferred unsubscribe always runs, so it stays bounded.
   - **ARCH-FUNERAL: pass.** Hub entries die with the cast; `blob:` URLs are revoked.

7. **Plan revisions:** rewrite the Lifecycle bullet at plan line 86. It should say the log is bounded by one segment, and that within a segment it grows with `ptr`/`hover` until #458's latest-only rule; it should not restate the ~1.4 MB/min figure.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      isURLAttr lowercases tag (render.js:595); TestLiveRenderResolvesPerDocument pins feImage href; image-set deferral recorded in plan.
  - id: BR-4
    disposition: addressed
    note: |
      textFor, owned style/link and fillSheet now use docURL (render.js:287,369,378,409); frame style pinned by the per-document test.
  - id: BR-7
    disposition: addressed
    note: |
      viewer.js builds resolveAsset once and couchViewer wraps it; JS catchUp removed, join checked against Go CatchUp (Chrome-gated).
  - id: BR-8
    disposition: addressed
    note: |
      atlas/webcast.md:74-78 and webcast-format.md:264 describe Ended; no Live.Done restatement remains outside the dated plan log.
  - id: BR-9
    disposition: addressed
    note: |
      Hub records drops under its lock; reverting Ended to h.closed fails TestHubMeetsTheLiveContract (verified).
  - id: BR-10
    disposition: addressed
    note: |
      server_test.go:272 fails on an empty event at EOF instead of looping.
  - id: BR-11
    disposition: addressed
    note: |
      hub.go doc drops the 1.4 MB/min bound and points to the ptr/hover deferral.
findings:
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan Lifecycle (line 86) still claims the ~1.4 MB/min Hub bound that BR-11 retracted from hub.go
    detail: |
      This is the 3rd finding in family plan-code-drift. Rule: when a fix round retracts or changes a claim, grep the claim's distinctive wording (here "1.4 MB") across atlas, plan and doc comments, and rewrite every hit in the same commit; workshop/lessons.md already states this for seam names, so extend it to retracted figures. Measured prevalence: 3 instances over 3 rounds (BR-8 atlas events route, the BR-8 seams row, this one).
```
