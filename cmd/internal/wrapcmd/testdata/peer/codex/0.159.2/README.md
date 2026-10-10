# Codex 0.159.2 peer composer captures

Captured 2026-09-30 by `TestPeerLiveConformance` in new disposable PTYs with
isolated working/configuration directories, existing login credentials, read-only
sandbox, and no daemon. No prompts were submitted in these two captures.

- `startup.raw`: complete recognized empty composer, including modern unboxed
  title + working-directory row and uppercase `GPT-6.1-Sol` footer.
- `paste-short.raw`: startup plus one bracketed paste. The complete 100-byte peer
  envelope is visible exactly, with no collapsed paste marker.

Commands: `go test ./cmd/internal/wrapcmd -run '^TestPeerLiveConformance$'
-count=1 -v -args -peer-live-harness=codex -peer-live-use-local-auth
-peer-live-capture-dir=/tmp/pair-353-codex`, adding
`-peer-live-scenario=paste-short` for the second capture.

These captures qualify startup and text observation only. They do not establish
submission conformance or authorize adding a receiver version to the runtime
allowlist. Existing lowercase/boxed startup fixtures remain regression coverage.

`paste-wrapped.raw` records the successful no-submit wrapped-body experiment,
with the accidental trailing space removed from the generated payload. Codex
wraps at word boundaries, removing the single separating space at each boundary.
Captured using the command above with `-peer-live-scenario=paste-wrapped`.
SHA-256: `d035de627391ffecab182f16e1e3ad9617f640eded317427fc7c83bb66a06988`.

Since pair#427 delivery no longer matches a rendered paste against the
envelope; each paste capture must read as an occupied composer
(`TestPeerComposerCapturedPastesReadOccupied`).
This capture still establishes no submission outcome.

Main-session live submission on 2026-09-30 passed using `-peer-live-submit`:
exact envelope, one submit, followed by native composer clearing. Capture
SHA-256 `183bdb1b5bd64821744991d13f1de0d20cfe5d0c606bcc7c9bb4ca6c111539a3`.
Multiline paste, draft preservation and menu preservation also passed. Combined
with the fixture/replay tests, this qualifies the exact runtime version. These
receiver checks do not establish human acceptance of the Couch workflow.
