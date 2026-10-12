# broadcast-web recording fixtures (#457)

Recordings in the format of `atlas/webcast-format.md`, for every wave-1 component's tests. Name them with
`recording.Fixture(name)` and list them with `recording.Fixtures()`, which finds every directory here with a
`canonical.sha256`.

| Fixture | Source | Length |
|---|---|---|
| `segments` | hand-made by `fixturegen -segments`: pause, resume with ids kept, a tab switch | 9 s, no frames |
| `rec-stress` | the real recorder (#458), #453's stress page from `testdata/stress`, `idle` | 5 s, 3 frames |
| `rec-app` | the real recorder, #453's app, `app` scenario | 22 s, 11 frames |
| `rec-blog` | the real recorder, the operator's blog, `/2026/10/ariadne-1-building-blocks-of-ai-coding` on `npm run dev`, `browse` scenario | 12 s, 6 frames |
| `rec-segments` | the real recorder, #453's `app2.html`: a same-document navigation, a pause with an edit, a resume, a new tab, a navigation there | 10 s, 5 frames |
| `rec-driven` | the real recorder and remote driving (#467): `driven` scenario over `serveDriven`'s two pages through MemController and TabInjector; Ana asks, is granted, clicks and types (`by`), the operator clicks, Ana follows a link (a navigation while driving), a pause and resume, Ana releases; Bo is denied; Ana is granted and revoked; Cy asks and leaves. A note by the driver (fixturegen's spec) | 21 s, 8 frames |

Each directory has:
- the record files;
- `assets/`;
- `frames/` (the real tab's screenshots at 640×400, for render comparison);
- `canonical.sha256`, the reducer's canonical state at each frame time (or each segment boundary and note), by hash.

The fixture test fails if any hash moves. After a deliberate reducer change, run
`go test ./cmd/internal/webcast/recording -run Fixtures -update-pins`. To see the expected value itself (what a viewer's
`canonical()` must equal):

```sh
go run ./cmd/internal/webcast/recording/internal/fixturegen -canonical cmd/internal/webcast/recording/testdata/fixtures/rec-app -at 6000
```

## Regenerating

### The real recorder's (`rec-*`, #472)

Recorded on 2026-10-12 by `TestRecordFixtures` (headless Chrome 154, 1280×800, a screenshot every 2 s), then made
fixtures by `fixturegen -recorded`, which converts nothing: it adds the notes of `fixturegen/notes.go` and the pins.

```sh
git archive origin/453-recorder-prototypes prototypes/broadcast-web-recorder | tar -x -C "$SCRATCH"
cd "$SCRATCH/prototypes/broadcast-web-recorder" && npm install && mkdir -p pages/assets && \
  cp node_modules/react/umd/react.production.min.js node_modules/react-dom/umd/react-dom.production.min.js \
     node_modules/htm/dist/htm.umd.js node_modules/@tailwindcss/browser/dist/index.global.js pages/assets/
(cd ~/workspace/xianxu.dev && npm run dev)   # the blog, on :4321
WEBCAST_LIVE_CHROME=1 WEBCAST_FIXTURE_RUNS="$SCRATCH/runs" \
  WEBCAST_APP_PAGES="$SCRATCH/prototypes/broadcast-web-recorder/pages" \
  WEBCAST_BLOG_URL=http://localhost:4321/2026/10/ariadne-1-building-blocks-of-ai-coding \
  go test ./cmd/internal/webcast -run RecordFixtures -v      # Chrome must run outside an agent sandbox
for n in stress app blog segments; do
  go run ./cmd/internal/webcast/recording/internal/fixturegen -recorded "$SCRATCH/runs/$n" -name rec-$n \
    -out cmd/internal/webcast/recording/testdata/fixtures/rec-$n
done
go test ./cmd/internal/webcast/query -run Golden -update   # review the diff
```

A re-recording differs in timing, so a note whose element isn't there at its time fails `fixturegen`; move its `t`.

The notes and strokes on each fixture come from `fixturegen/notes.go`. They are placed on known elements, at the
element's box at that moment.

#453's prototype recorder's fixtures (`stress`, `app`, `blog`), its `record-mjs.patch` and `fixturegen -raw` retired
in #474; git history has them.

## Known gaps

- **`rec-stress` frame 0 shows the page styled; the recording binds `stress.css` 19 ms later** (#475): a segment's linked
  stylesheet arrives when its body is fetched. `replay_test.go` lists the frame in `knownGaps`.
- **`rec-stress` has no FontFace font**: `testdata/stress` leaves the late font out (a copy of a macOS system font, which this repository must not redistribute), so the page never loads it.
- **Text nodes other than `<style>` are cut at CDP's 10,000 characters.** None in these pages exceed it.
