# broadcast-web recording fixtures (#457)

Recordings in the format of `atlas/webcast-format.md`, for every wave-1 component's tests. Name them with
`recording.Fixture(name)` and list them with `recording.Fixtures`.

| Fixture | Source | Length |
|---|---|---|
| `stress` | #453's stress page, idle scenario | 7 s, 4 frames |
| `app` | #453's React + Tailwind app, `app` scenario (typing, clicks, a modal, a cross-document navigation) | 22 s, 11 frames |
| `blog` | the operator's blog, `/2026/10/ariadne-1-building-blocks-of-ai-coding` on `npm run dev`, `browse` scenario | 12 s, 6 frames |
| `segments` | hand-made by `fixturegen -segments`: pause, resume with ids kept, a tab switch | 9 s, no frames |
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

The three real fixtures were recorded on 2026-10-11 by #453's prototype recorder: branch `453-recorder-prototypes`,
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

- **The stress page's late font is not included.** It is a copy of a macOS system font (Apple Chancery), which this
  repository must not redistribute. Its `font` record stays, with no asset bound to it.
- **Layout covers the top document only.** `DOMSnapshot.captureSnapshot` on the top session leaves out
  out-of-process frames. The recorder (#458) has to capture each frame session.
- **The `app` run lost one click.** The click that navigates to app2 never arrived: the prototype's observer is polled,
  and the page unloaded before the poll. The recorder (#458) must report clicks synchronously.
- **Text nodes other than `<style>` are cut at CDP's 10,000 characters.** None in these pages exceed it.
