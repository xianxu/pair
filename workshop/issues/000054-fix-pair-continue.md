---
id: 000054
status: working
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-11
estimate_hours: 1.0
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

## Spec

1. **Don't force the tag.** Drop `forced_tag="$_cslug"` from the `continue`
   block. `continue` then flows through the normal picker/name-prompt, so the
   operator names the session (default = the cwd-derived free slot, short).
   The draft-seeding is already create-path-only, so it seeds whichever tag is
   chosen; attach (existing live tag) never clobbers a draft.
2. **Forward `-- <args>`.** Replace `shift "$#"` with `shift 2` + a conditional
   agent-port shift, leaving everything from `--` for the positional loop to
   capture as `agent_extra`. `[agent]` port and agent-from-frontmatter both
   preserved.
3. **Root-cause guard (covers every create path, not just continue).** Before
   the `zellij --new-session-with-layout` launch, probe zellij's own validator
   with a harmless `zj --session "$SESSION" action list-clients`; if it returns
   the "must be less than" rejection, abort with a clear "pick a shorter tag"
   message instead of letting the cryptic clap error abort the launch.

## Done when

- `pair continue <slug> [agent] -- <args>` forwards the args and prompts for the tag.
- A long tag (continue, resume, or manually typed) fails with pair's clear message, not zellij's.
- Bare `pair continue` list + slug resolution + `[agent]` port still work.
- `bash -n bin/pair` clean; docs (README + atlas) updated.

## Plan

- [x] `bin/pair`: remove `forced_tag` from continue block; `shift 2` + agent-port shift; update header comment
- [x] `bin/pair`: launch-time session-name guard (probe zellij validator)
- [x] `README.md` + `atlas/architecture.md`: reflect tag-prompt + arg-forward + guard
- [x] Verify: durable `tests/pair-continue-test.sh` (real bin/pair via `PAIR_DEBUG_ARGS` probe) + guard short-vs-long; wired into `make test`

## Log

### 2026-06-11
- Root-caused the crash: zellij `--session` sun_path budget = capacity − socket_dir_len; long macOS `$TMPDIR` shrinks it below the forced slug length. Reproduced locally (≥~60 chars rejected here; ≥28 on the operator's longer-TMPDIR machine). Message "less than 0 characters" is a zellij cosmetic bug.
- Reversed an earlier in-session removal plan (operator changed direction twice → keep + fix). See `#50` (the feature, merged PR #23), `#91` (continuation datatype).
- Implemented all three Spec items in `bin/pair`. Added a `PAIR_DEBUG_ARGS=1` probe (sibling of `PAIR_DEBUG_HISTORY`) that dumps resolved argv and exits pre-launch, so the new `tests/pair-continue-test.sh` drives the REAL script (not a mirror) — per the repo's drive-the-artifact discipline.
- **The test caught a bug in the fix itself:** the guard was first written `zj … | grep -q`, which under `bin/pair`'s `set -euo pipefail` returns zellij's non-zero exit (not grep's), so the `if` was always false and the guard would never fire. Rewrote it capture-then-`case`-match with `|| true`. The existing line-916/939 comments warn about exactly this pipefail footgun. Test mirrors the same capture-then-match shape.
- Verified: `bash -n bin/pair` clean; `make build` (Go binaries incl. `pair-continuation` writer untouched); `bash tests/pair-continue-test.sh` → 11/11 ok (4 arg forms, `[agent]` port, bare list, 2 error paths, guard short-vs-long). Reproduced the original crash locally: zellij rejects an ~60-char `--session` here (`/tmp` TMPDIR); on the operator's longer macOS `/var/folders` TMPDIR the budget drops below 28, hence the 28-char forced slug crashed.
