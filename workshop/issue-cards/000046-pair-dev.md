---
id: '000046'
status: done
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1
actual_hours: 1
---

# pair-dev dev-mode rebuild entrypoint

## Problem

The agent pane is spawned by `zellij/layouts/main.kdl` via
`sh -c '... exec pair-wrap ... $PAIR_AGENT ...'`, which resolves `pair-wrap`
through PATH (repo `bin/` is first, per `.zshrc`). That path **never rebuilds**:
`sh -c` doesn't source `.zshenv`, and `exec` bypasses shell functions, so the
`construct/dev-aliases.sh` rebuild-on-call function cannot reach it. When
`repo/bin/pair-wrap` is stale — or absent, since it's gitignored — PATH silently
falls through to an old `~/.local/bin/pair-wrap`, and the running wrapper drifts
from source with no error.

Observed this session: the adaptation flight recorder (#000045) went silent for
all Go-emitted aspects (1/2/4/5) while only nvim's Lua emitter (aspect 7) logged
— because the installed `pair-wrap`/`pair-slug` binaries predated #000045. See
`workshop/lessons.md` (stale-`~/.local/bin` footgun) and `atlas/architecture.md`
§ pair-wrap staleness.

We want **two modes**: deployed (prebuilt binary, no toolchain dependency) and
dev (always fresh from source). dev-aliases' freshness is an interactive-shell
property that structurally does not extend to the zellij-launched wrapper.
