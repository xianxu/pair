---
id: '000094'
status: done
started: 2026-07-03T10:57:55-07:00
created: 2026-07-01
updated: 2026-07-03
estimate_hours: 4.4
actual_hours: 2.50
---

# stop extracting shell scripts from runtime bundle

## Problem

Once #93 has ported the stateful shell orchestrators into Go, the shell scripts
in the embedded runtime bundle (`bin/*.sh`, `bin/pair-shell`, and any remaining
shell helpers) are dead weight: the copied binary still extracts them to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` even though nothing execs them. The
runtime manifest (`cmd/internal/runtimebundle/assets/runtime/manifest.json`) is
the single source of what gets packaged and extracted, so shrinking it is the
concrete step that removes shell from the deployed footprint.
