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

- [ ] `bin/pair`: remove `forced_tag` from continue block; `shift 2` + agent-port shift; update header comment
- [ ] `bin/pair`: launch-time session-name guard (probe zellij validator)
- [ ] `README.md` + `atlas/architecture.md`: reflect tag-prompt + arg-forward + guard
- [ ] Verify: `bash -n`; arg-parse harness (4 forms); guard discriminates short vs long; bare list

## Log

### 2026-06-11
- Root-caused the crash: zellij `--session` sun_path budget = capacity − socket_dir_len; long macOS `$TMPDIR` shrinks it below the forced slug length. Reproduced locally (≥~60 chars rejected here; ≥28 on the operator's longer-TMPDIR machine). Message "less than 0 characters" is a zellij cosmetic bug.
- Reversed an earlier in-session removal plan (operator changed direction twice → keep + fix). See `#50` (the feature, merged PR #23), `#91` (continuation datatype).
