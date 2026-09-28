---
id: '000054'
status: done
created: 2026-06-11
updated: 2026-06-11
estimate_hours: 1.0
actual_hours: 0.14
---

# Fix pair continue: startup crash + don't force tag + forward args

## Problem

Dogfooding `pair continue <slug>` (shipped in `#50`) surfaced two defects:

1. **Trailing `-- <args>` were dropped.** The `continue` block ended with
   `shift "$#"`, discarding everything after the slug/agent — so
   `pair continue port claude -- --dangerously-skip-permissions` reached the
   saved-config picker with empty args.
2. **Startup crash on launch.** `continue` forced the slug as the session tag,
   so `pair-parley-readonly-harness` was handed to `zellij --session`. zellij
   caps a session name at its unix-socket **sun_path budget** (`capacity −
   socket_dir_length`); on macOS the `/var/folders/...` `$TMPDIR` socket dir is
   long enough that even a 28-char name overflows, and zellij aborts the launch
   with a cryptic clap error: `session name must be less than 0 characters`
   (the "0" is a zellij display bug; the rejection is real — reproduced locally
   with an ~60-char name).

**Decision (operator, 2026-06-11):** keep `pair continue`, fix the crash, and
reshape it to behave like `pair <agent> -- <args>` — pre-seeded but normal.
(Earlier this session I drafted a removal under this id; reversed.)
