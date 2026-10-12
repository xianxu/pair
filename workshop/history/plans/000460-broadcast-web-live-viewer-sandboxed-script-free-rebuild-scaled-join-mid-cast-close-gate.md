---
gate: boundary-review
issue: 460
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T01:42:35-07:00"
      dispatched: "2026-10-12T01:38:48-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: events route ends the stream on any closed recs channel, so a closed Hub with a live browser loops reconnect plus full catch-up
          detail: server.go:149 returns on !open. Hub.Subscribe after Close sends the catch-up and then closes, so EventSource reconnects forever, rebuilding from head each time. Distinguish close from drop on the Live seam (Done/Closed); on close set recs=nil and keep serving state/end. TestServerEndsStreamWhenRecordsClose pins the current behaviour.
          family: closed-vs-dropped-conflated
          round: 1
        - id: BR-2
          severity: Important
          title: model.js refuses hover with an unknown id, forcing every viewer to resync on a record outside Canonical
          detail: model.js:241 throws, and viewer.js resync then closes, reconnects and replays the catch-up. The Go reducer ignores hover. An unrecorded node (the astro-dev-toolbar shadow root, recorded empty) or a hover after rm would trigger it repeatedly. Treat an unknown id as hover=0.
          family: cosmetic-record-forces-resync
          round: 1
        - id: BR-3
          severity: Minor
          title: URL_ATTRS lists 'feimage' but SVG localName is 'feImage', so feImage href bypasses resolveAsset
          detail: 'render.js:31. The CSP img-src ''self'' backstops it. Also: css.js does not rewrite image-set()/-webkit-image-set bare strings.'
          family: url-attr-coverage
          round: 1
        - id: BR-4
          severity: Minor
          title: style text, owned sheets and constructed sheets resolve against model.url, not the owning document's URL
          detail: render.js textFor, drawSheetOwner (owned) and fillSheet use this.model.url, while drawAttr and link href use docURL(el.ownerDocument). Nested-frame relative url()s resolve against the top page.
          family: css-base-url
          round: 1
        - id: BR-5
          severity: Minor
          title: Plan says each record is encoded once and shared; Server encodes per viewer and silently skips Encode errors
          detail: 'server.go:153. Either share the encoded bytes from the Hub or add a plan Revision; CSP data: sources are also undeclared in the plan.'
          family: plan-code-drift
          round: 1
        - id: BR-6
          severity: Minor
          title: ptr/hover records accumulate in the Hub log for the whole segment and are all replayed in every catch-up
          detail: The ~1.4 MB/min bound predates ptr. CatchUp could keep only the latest ptr/hover.
          family: superseded-records-retained
          round: 1
        - id: BR-7
          severity: Minor
          title: viewer.js mount builds the asset-URL resolver twice; model.test.mjs re-implements CatchUp in JS
          family: duplicated-helper
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T02:11:51-07:00"
      dispatched: "2026-10-12T01:59:50-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Live.Done plus recs=nil in server.go:151; TestServerKeepsStreamWhenRecordsEnd goes red (hangs to timeout) with the fix reverted.
          round: 2
        - id: BR-2
          disposition: addressed
          note: model.js hover clears on unknown id; model.test.mjs and viewer.test.mjs both fail with the fix reverted.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: isURLAttr lower-cases the tag (render.js:597) but there is no test that fails without it; image-set deferral is in the plan revision.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: docURL(ownerDocument) now threaded through textFor/drawSheetOwner/fillSheet, but no nested-frame relative url() test pins it.
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'Plan Revision 2026-10-12 records per-viewer Encode and the CSP data: sources.'
          round: 2
        - id: BR-6
          disposition: addressed
          note: Recorded as a deferral to the format owners (#457/#458) in the plan revision; the hub.go bound comment is still stale (see new Minor).
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: Resolver deduplicated in viewer.js; model.test.mjs:27 still re-implements CatchUp in JS.
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: atlas/webcast.md:75 still says a closed record channel always ends the stream and the viewer reconnects
          detail: '2nd in family plan-code-drift. Rule: a contract change made in a fix round updates every document that restates it in the same commit. Also add Done to the recording.Live row at atlas/webcast-format.md:264.'
          family: plan-code-drift
          round: 2
        - id: BR-9
          severity: Minor
          title: A viewer dropped just before Hub.Close is read as ended and never reconnects
          detail: '2nd in family closed-vs-dropped-conflated. server.go infers drop vs close by sampling Done after the channel closes. Rule: the source reports why a subscription ended (for example a per-subscription dropped flag set under the Hub lock).'
          family: closed-vs-dropped-conflated
          round: 2
        - id: BR-10
          severity: Minor
          title: TestServerKeepsStreamWhenRecordsEnd loops forever when the stream reaches EOF
          detail: s.next() returns "" at EOF and the for-loop never exits; with the BR-1 fix reverted the test hangs to the 10-minute go test timeout. Fail when ev is empty.
          family: test-hangs-instead-of-failing
          round: 2
        - id: BR-11
          severity: Minor
          title: hub.go doc still claims the 1.4 MB/min bound that predates ptr/hover accumulation
          detail: 2nd in family superseded-records-retained. The comment should point to the deferral recorded in the plan rather than restate a bound that no longer holds.
          family: superseded-records-retained
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T02:17:33-07:00"
      dispatched: "2026-10-12T02:15:29-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: isURLAttr lowercases tag (render.js:595); TestLiveRenderResolvesPerDocument pins feImage href; image-set deferral recorded in plan.
          round: 3
        - id: BR-4
          disposition: addressed
          note: textFor, owned style/link and fillSheet now use docURL (render.js:287,369,378,409); frame style pinned by the per-document test.
          round: 3
        - id: BR-7
          disposition: addressed
          note: viewer.js builds resolveAsset once and couchViewer wraps it; JS catchUp removed, join checked against Go CatchUp (Chrome-gated).
          round: 3
        - id: BR-8
          disposition: addressed
          note: atlas/webcast.md:74-78 and webcast-format.md:264 describe Ended; no Live.Done restatement remains outside the dated plan log.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Hub records drops under its lock; reverting Ended to h.closed fails TestHubMeetsTheLiveContract (verified).
          round: 3
        - id: BR-10
          disposition: addressed
          note: server_test.go:272 fails on an empty event at EOF instead of looping.
          round: 3
        - id: BR-11
          disposition: addressed
          note: hub.go doc drops the 1.4 MB/min bound and points to the ptr/hover deferral.
          round: 3
      findings:
        - id: BR-12
          severity: Minor
          title: Plan Lifecycle (line 86) still claims the ~1.4 MB/min Hub bound that BR-11 retracted from hub.go
          detail: 'This is the 3rd finding in family plan-code-drift. Rule: when a fix round retracts or changes a claim, grep the claim''s distinctive wording (here "1.4 MB") across atlas, plan and doc comments, and rewrite every hit in the same commit; workshop/lessons.md already states this for seam names, so extend it to retracted figures. Measured prevalence: 3 instances over 3 rounds (BR-8 atlas events route, the BR-8 seams row, this one).'
          family: plan-code-drift
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 38b1d6ea1046a7d59ba6e2aa894afe0d4d33ff70
      blocked: false
---

# Gate ledger — pair#460 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T01:42:35-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `closed-vs-dropped-conflated` events route ends the stream on any closed recs channel, so a closed Hub with a live browser loops reconnect plus full catch-up
  server.go:149 returns on !open. Hub.Subscribe after Close sends the catch-up and then closes, so EventSource reconnects forever, rebuilding from head each time. Distinguish close from drop on the Live seam (Done/Closed); on close set recs=nil and keep serving state/end. TestServerEndsStreamWhenRecordsClose pins the current behaviour.
- **BR-2** [Important] `cosmetic-record-forces-resync` model.js refuses hover with an unknown id, forcing every viewer to resync on a record outside Canonical
  model.js:241 throws, and viewer.js resync then closes, reconnects and replays the catch-up. The Go reducer ignores hover. An unrecorded node (the astro-dev-toolbar shadow root, recorded empty) or a hover after rm would trigger it repeatedly. Treat an unknown id as hover=0.
- **BR-3** [Minor] `url-attr-coverage` URL_ATTRS lists 'feimage' but SVG localName is 'feImage', so feImage href bypasses resolveAsset
  render.js:31. The CSP img-src 'self' backstops it. Also: css.js does not rewrite image-set()/-webkit-image-set bare strings.
- **BR-4** [Minor] `css-base-url` style text, owned sheets and constructed sheets resolve against model.url, not the owning document's URL
  render.js textFor, drawSheetOwner (owned) and fillSheet use this.model.url, while drawAttr and link href use docURL(el.ownerDocument). Nested-frame relative url()s resolve against the top page.
- **BR-5** [Minor] `plan-code-drift` Plan says each record is encoded once and shared; Server encodes per viewer and silently skips Encode errors
  server.go:153. Either share the encoded bytes from the Hub or add a plan Revision; CSP data: sources are also undeclared in the plan.
- **BR-6** [Minor] `superseded-records-retained` ptr/hover records accumulate in the Hub log for the whole segment and are all replayed in every catch-up
  The ~1.4 MB/min bound predates ptr. CatchUp could keep only the latest ptr/hover.
- **BR-7** [Minor] `duplicated-helper` viewer.js mount builds the asset-URL resolver twice; model.test.mjs re-implements CatchUp in JS

## Round 2 — 2026-10-12T02:11:51-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Live.Done plus recs=nil in server.go:151; TestServerKeepsStreamWhenRecordsEnd goes red (hangs to timeout) with the fix reverted.
- BR-2 — addressed — model.js hover clears on unknown id; model.test.mjs and viewer.test.mjs both fail with the fix reverted.
- BR-3 — not-addressed — isURLAttr lower-cases the tag (render.js:597) but there is no test that fails without it; image-set deferral is in the plan revision.
- BR-4 — not-addressed — docURL(ownerDocument) now threaded through textFor/drawSheetOwner/fillSheet, but no nested-frame relative url() test pins it.
- BR-5 — addressed — Plan Revision 2026-10-12 records per-viewer Encode and the CSP data: sources.
- BR-6 — addressed — Recorded as a deferral to the format owners (#457/#458) in the plan revision; the hub.go bound comment is still stale (see new Minor).
- BR-7 — not-addressed — Resolver deduplicated in viewer.js; model.test.mjs:27 still re-implements CatchUp in JS.

### Raised

- **BR-8** [Important] `plan-code-drift` atlas/webcast.md:75 still says a closed record channel always ends the stream and the viewer reconnects
  2nd in family plan-code-drift. Rule: a contract change made in a fix round updates every document that restates it in the same commit. Also add Done to the recording.Live row at atlas/webcast-format.md:264.
- **BR-9** [Minor] `closed-vs-dropped-conflated` A viewer dropped just before Hub.Close is read as ended and never reconnects
  2nd in family closed-vs-dropped-conflated. server.go infers drop vs close by sampling Done after the channel closes. Rule: the source reports why a subscription ended (for example a per-subscription dropped flag set under the Hub lock).
- **BR-10** [Minor] `test-hangs-instead-of-failing` TestServerKeepsStreamWhenRecordsEnd loops forever when the stream reaches EOF
  s.next() returns "" at EOF and the for-loop never exits; with the BR-1 fix reverted the test hangs to the 10-minute go test timeout. Fail when ev is empty.
- **BR-11** [Minor] `superseded-records-retained` hub.go doc still claims the 1.4 MB/min bound that predates ptr/hover accumulation
  2nd in family superseded-records-retained. The comment should point to the deferral recorded in the plan rather than restate a bound that no longer holds.

## Round 3 — 2026-10-12T02:17:33-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — isURLAttr lowercases tag (render.js:595); TestLiveRenderResolvesPerDocument pins feImage href; image-set deferral recorded in plan.
- BR-4 — addressed — textFor, owned style/link and fillSheet now use docURL (render.js:287,369,378,409); frame style pinned by the per-document test.
- BR-7 — addressed — viewer.js builds resolveAsset once and couchViewer wraps it; JS catchUp removed, join checked against Go CatchUp (Chrome-gated).
- BR-8 — addressed — atlas/webcast.md:74-78 and webcast-format.md:264 describe Ended; no Live.Done restatement remains outside the dated plan log.
- BR-9 — addressed — Hub records drops under its lock; reverting Ended to h.closed fails TestHubMeetsTheLiveContract (verified).
- BR-10 — addressed — server_test.go:272 fails on an empty event at EOF instead of looping.
- BR-11 — addressed — hub.go doc drops the 1.4 MB/min bound and points to the ptr/hover deferral.

### Raised

- **BR-12** [Minor] `plan-code-drift` Plan Lifecycle (line 86) still claims the ~1.4 MB/min Hub bound that BR-11 retracted from hub.go
  This is the 3rd finding in family plan-code-drift. Rule: when a fix round retracts or changes a claim, grep the claim's distinctive wording (here "1.4 MB") across atlas, plan and doc comments, and rewrite every hit in the same commit; workshop/lessons.md already states this for seam names, so extend it to retracted figures. Measured prevalence: 3 instances over 3 rounds (BR-8 atlas events route, the BR-8 seams row, this one).

## Open findings

- **BR-12** [Minor] `plan-code-drift` Plan Lifecycle (line 86) still claims the ~1.4 MB/min Hub bound that BR-11 retracted from hub.go
