# broadcast-web recording fixtures (#457)

Recordings in the format of `atlas/webcast-format.md`, for every wave-1 component's tests. Name them with
`recording.Fixture(name)` and list them with `recording.Fixtures`.

| Fixture | Source | Length |
|---|---|---|
| `stress` | #453's stress page, idle scenario | 7 s, 4 frames |
| `app` | #453's React + Tailwind app, `app` scenario (typing, clicks, a modal, a cross-document navigation) | 22 s, 11 frames |
| `blog` | the operator's blog, `/2026/10/ariadne-1-building-blocks-of-ai-coding` on `npm run dev`, `browse` scenario | 12 s, 6 frames |
| `segments` | hand-made by `fixturegen -segments`: pause, resume with ids kept, a tab switch | 9 s, no frames |
| `rec-stress` | the real recorder (#458), #453's stress page from `testdata/stress`, `idle` | 5 s, 3 frames |
| `rec-app` | the real recorder, #453's app, `app` scenario | 22 s, 11 frames |
| `rec-blog` | the real recorder, the blog post above, `browse` scenario | 12 s, 6 frames |
| `rec-segments` | the real recorder, #453's `app2.html`: a same-document navigation, a pause with an edit, a resume, a new tab, a navigation there | 10 s, 5 frames |
| `driven` | hand-made by `fixturegen -driven` from `recording.DrivenRecording()` (#466): viewers request, are granted, click and type (attributed `by`), drive through a pause, a resume and a navigation; release, revoke, deny, leave | 9.5 s, no frames |

Each directory has:
- the record files;
- `assets/`;
- `frames/` (the real tab's screenshots at 640×400, for render comparison);
- `canonical.sha256`, the reducer's canonical state at each frame time (or each segment boundary and note), by hash.

The fixture test fails if any hash moves. After a deliberate reducer change, run
`go test ./cmd/internal/webcast/recording -run Fixtures -update-pins`. To see the expected value itself (what a viewer's
`canonical()` must equal):

```sh
go run ./cmd/internal/webcast/recording/internal/fixturegen -canonical cmd/internal/webcast/recording/testdata/fixtures/app -at 6000
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

### #453's prototype's (`stress`, `app`, `blog`)

These retire once #470 and #471 have landed (#474).

The three prototype fixtures were recorded on 2026-10-11 by #453's prototype recorder: branch `453-recorder-prototypes`,
`prototypes/broadcast-web-recorder`, mode B, headless Chrome 154.0.8037.98, 1280×800. `record-mjs.patch` holds what #457
added to `record.mjs`:
- `DOMSnapshot` layout boxes with each screenshot;
- click and focus capture in the isolated-world observer;
- `frameNavigated` events;
- the viewport;
- asset bodies for images, fonts and stylesheets.

```sh
git archive origin/453-recorder-prototypes prototypes/broadcast-web-recorder | tar -x -C "$SCRATCH"
cd "$SCRATCH/prototypes/broadcast-web-recorder" && patch record.mjs < …/record-mjs.patch && npm install && npm run setup
node record.mjs runs/stress http://127.0.0.1:8801/stress.html idle B 7     # Chrome must run outside an agent sandbox
node record.mjs runs/app http://127.0.0.1:8801/app.html app B 22
node record.mjs runs/blog http://localhost:4321/2026/10/ariadne-1-building-blocks-of-ai-coding browse B 12
go run ./cmd/internal/webcast/recording/internal/fixturegen -raw "$SCRATCH/…/runs/app" -name app -out cmd/internal/webcast/recording/testdata/fixtures/app
```

The notes and strokes on each fixture come from `fixturegen/notes.go`. They are placed on known elements, at the
element's box at that moment.

## Known gaps

- **`rec-stress` frame 0 shows the page styled; the recording binds `stress.css` 19 ms later** (#475): a segment's linked
  stylesheet arrives when its body is fetched. `replay_test.go` lists the frame in `knownGaps`.
- **`rec-stress` has no FontFace font**: `testdata/stress` leaves the late font out, so the page never loads it.
- **The stress page's late font is not included.** It is a copy of a macOS system font (Apple Chancery), which this
  repository must not redistribute. Its `font` record stays, with no asset bound to it.
- **Layout covers the top document only.** `DOMSnapshot.captureSnapshot` on the top session leaves out
  out-of-process frames. The recorder (#458) has to capture each frame session.
- **The `app` run lost one click.** The click that navigates to app2 never arrived: the prototype's observer is polled,
  and the page unloaded before the poll. The recorder (#458) must report clicks synchronously.
- **Text nodes other than `<style>` are cut at CDP's 10,000 characters.** None in these pages exceed it.
