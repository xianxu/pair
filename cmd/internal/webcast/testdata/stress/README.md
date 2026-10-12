# The stress page (#453)

#453's stress page (branch `453-recorder-prototypes`, `prototypes/broadcast-web-recorder/pages`), kept here so the
recorder's fidelity runs can be reproduced from this repository (#458). `{{FRAME}}` is replaced by the origin of the
cross-origin frame when served. `/assets/late.ttf`, a FontFace font #453 copied from a macOS system font, is left
out (as in #457's `stress` fixture): the font load fails the same way on every run.

```sh
WEBCAST_LIVE_CHROME=1 WEBCAST_RECORD_URL=stress WEBCAST_RECORD_OUT=DIR WEBCAST_RECORD_SCENARIO=idle \
  WEBCAST_RECORD_SECS=7 go test ./cmd/internal/webcast -run LiveRecordURL -v
FIDELITY_DIR=DIR go test ./cmd/internal/webcast/recorder/internal/fidelity -run Dir -v
```
