# JSON boundary validation

`internal/jsonstrict.Scan` is the shared structural authority for one JSON value.
It promotes Pair's existing byte scanner; schemas, source byte limits, typed
owned fields and raw unknown-field retention stay in each consumer.

The zero `Options` policy uses exact decoded-key equality and the historical
float64 range check. `FoldKeys` uses encoding/json-equivalent Unicode simple
folding, `PreserveNumbers` retains opaque out-of-range number tokens, `MaxDepth`
bounds value depth, and `VisitKey` sees each original decoded key at its object
depth before policy adoption. Owner choices are explicit:

| Consumer | Policy | Owner checks retained |
|---|---|---|
| cmd/internal/strictjson | exact, ordinary or preserve-number wrapper | strict typed decode and raw-object helpers |
| historyarchive | folded keys, preserved numbers, depth64 | strict/non-strict typed fields and retained raw extensions |
| submission | exact, preserved numbers, all-depth key visitor | sensitive aliases in opaque extensions and manifest schema |
| repolint debt | exact, ordinary numbers | known fields, single document and debt ratchet |

`internal/jsonstrict/scan_test.go` counts actual byte inspections and key visits
at128/256 input rows. The independent legacy-token fuzz oracle remains in
`cmd/internal/strictjson/decode_equivalence_test.go`; it must not delegate to the
scanner it qualifies. Its temporary per-site lint allow and the independent
standalone sample DOM materializer allow name ariadne#590 and retire when the
upstream role classification is corrected.

For fixture timing, `cmd/internal/testwait.DeadlineOnCue` advertises a budget,
propagates parent cancellation and makes the first terminal cause win. Tests
expire the request only after a child readiness marker or the exact runtime
stage; own-request tests inject that factory rather than cancel their parent.
Provision, VM, tunnel startup, fixture CLI and seat-grace seams default to the
ordinary standard-library contexts/timers. Child, callback and IO joins remain
the existing owners' responsibility. Socket fixtures reuse `testpair.ShortRoot`;
ordinary build/data temporaries retain inherited TMPDIR.
