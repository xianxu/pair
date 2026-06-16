#!/usr/bin/env bash
# Regression test for `pair continue` arg-parsing + the launch-time session-
# name guard (bin/pair, #54). Drives the REAL bin/pair through its
# PAIR_DEBUG_ARGS probe (which resolves argv, then exits before any zellij
# work) so the contract is pinned against the actual script, not a mirror.
#
# Covers the two #54 defects + the reshape:
#   1. `continue <slug>` does NOT force the tag (forced_tag empty → the
#      operator names the session at the normal prompt) — the fix for the
#      "session name must be less than 0 characters" zellij crash that the
#      forced long slug triggered.
#   2. `-- <args>` forward to the agent (the old `shift "$#"` dropped them).
#   3. agent comes from the doc frontmatter, or an explicit `[agent]` port.
#   4. bare list + error paths (unknown / invalid slug) still hold.
#   5. the guard's grep discriminates a short (ok) vs over-long (reject) name
#      against the REAL zellij --session validator.
#
# Run: bash tests/pair-continue-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR="$ROOT/bin/pair"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-continue-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

# Fixture: a continuation doc whose frontmatter agent is `claude`.
CDIR="$RT/workshop/continuation"
mkdir -p "$CDIR"
cat > "$CDIR/20260101T000000-demo.md" <<'DOC'
---
agent: claude
issues: [#99]
---
## NEXT ACTION
Do the demo next action.
DOC

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Resolve argv via the real script's debug probe; echo one field's value.
# XDG_DATA_HOME is pinned into the sandbox so the probe never reads/writes the
# operator's real pair data dir. cwd = RT so `continue` finds the fixture.
probe() { # field, then pair args...
  local field="$1"; shift
  ( cd "$RT" && XDG_DATA_HOME="$RT/xdg" PAIR_DEBUG_ARGS=1 "$PAIR" "$@" 2>/dev/null ) \
    | awk -F= -v k="$field" '$1==k{sub(/^[^=]*=/,""); print; exit}'
}

# 1. tag is NOT forced + agent resolved from the doc frontmatter
[ "$(probe FORCED_TAG continue demo)" = "" ] \
  && pass "continue: tag not forced (prompts like a normal create)" \
  || fail "continue: tag should not be forced"
[ "$(probe AGENT continue demo)" = "claude" ] \
  && pass "continue: agent from doc frontmatter" || fail "continue: agent should come from doc"
[ -n "$(probe CONTINUE_DOC continue demo)" ] \
  && pass "continue: slug resolved to the doc" || fail "continue: doc not resolved"

# 2. `-- <args>` forward to the agent (with and without an agent port)
[ "$(probe AGENT_EXTRA continue demo -- --dangerously-skip-permissions)" = "--dangerously-skip-permissions" ] \
  && pass "continue: forwards -- args" || fail "continue: -- args dropped"
[ "$(probe AGENT_EXTRA continue demo claude -- --foo bar)" = "--foo bar" ] \
  && pass "continue: forwards -- args after an agent port" || fail "continue: -- args after port dropped"

# 3. explicit [agent] port overrides the frontmatter agent
[ "$(probe AGENT continue demo codex)" = "codex" ] \
  && pass "continue: [agent] port overrides the doc" || fail "continue: port not honored"

# 4. bare list + error paths
( cd "$RT" && "$PAIR" continue 2>&1 | grep -q 'demo' ) \
  && pass "bare list shows the continuation" || fail "bare list missing entry"
( cd "$RT" && "$PAIR" continue nope >/dev/null 2>&1 ); [ $? -eq 1 ] \
  && pass "unknown slug exits 1" || fail "unknown slug should exit 1"
( cd "$RT" && "$PAIR" continue 'bad/slug' >/dev/null 2>&1 ); [ $? -eq 1 ] \
  && pass "invalid slug exits 1" || fail "invalid slug should exit 1"

# 4b. a long NEXT-ACTION first line is truncated in the bare list (#52)
LONG="$(printf 'X%.0s' $(seq 1 90))TAILMARKER"
cat > "$CDIR/20260101T000009-longrow.md" <<DOC
---
agent: claude
issues: [#1]
---
## NEXT ACTION
$LONG
DOC
LOUT="$( cd "$RT" && "$PAIR" continue 2>&1 )"
{ printf '%s' "$LOUT" | grep -q '…' && ! printf '%s' "$LOUT" | grep -q 'TAILMARKER'; } \
  && pass "bare list truncates a long NEXT ACTION line" \
  || fail "bare list did not truncate the long NEXT ACTION line"
rm -f "$CDIR/20260101T000009-longrow.md"

# 5. guard: zellij's own --session validator discriminates short vs over-long.
# Mirror the launch-time guard EXACTLY — capture-then-match, not `| grep` —
# since under pipefail the probe's non-zero exit would corrupt a piped test
# (the bug this very check exposed during development). Assert both verdicts so
# the guard neither false-positives on a valid name nor misses an over-long one.
guard_rejects() { # session-name → 0 if zellij rejects it as too long
  local out; out="$(zellij --session "$1" action list-clients 2>&1 || true)"
  case "$out" in *"session name must be less than"*) return 0 ;; *) return 1 ;; esac
}
if command -v zellij >/dev/null 2>&1; then
  guard_rejects "pair-demo" \
    && fail "guard: a short name was wrongly rejected" \
    || pass "guard: short name accepted"
  guard_rejects "pair-$(printf 'x%.0s' $(seq 1 120))" \
    && pass "guard: over-long name rejected (→ pair's clear-error path)" \
    || fail "guard: over-long name not rejected by zellij validator"
else
  printf '  skip guard discrimination (no zellij on PATH)\n'
fi

# ---------------------------------------------------------------------------
# #55 in-session compaction (Alt+Shift+C). Drives the REAL bin/pair via the
# PAIR_FORCE_IN_SESSION / PAIR_FAKE_IN_ZELLIJ / PAIR_KILL_CMD / PAIR_TEST_CALL /
# PAIR_REEXEC_CAPTURE seams — no real zellij/agent.
# ---------------------------------------------------------------------------
CRT="$(mktemp -d "${TMPDIR:-/tmp}/pair-compact-test.XXXXXX")"
trap 'rm -rf "$RT" "$CRT"' EXIT
mkdir -p "$CRT/workshop/continuation" "$CRT/xdg/pair" "$CRT/.cache/pair"
cat > "$CRT/workshop/continuation/20260101T000000-demo.md" <<'DOC'
---
agent: claude
issues: [#99]
---
## NEXT ACTION
demo.
DOC
seed_sb() { printf 'SCROLLBACK BYTES\n' > "$CRT/xdg/pair/scrollback-demo-claude.raw"; }
compact_env() { # common env for an in-pane invocation. `env` (not a bare
  # VAR=val prefix) so seam assignments passed via "$@" are treated as env,
  # not run as a command (bash only recognizes LITERAL leading assignments).
  env HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" PAIR_TAG=demo PAIR_AGENT=claude PAIR_KILL_CMD=true "$@"
}
MK="$CRT/.cache/pair/restart-pair-demo"

# 1. forced in-session: marker shape + park-copy
seed_sb; rm -f "$MK"
( cd "$CRT" && compact_env PAIR_FORCE_IN_SESSION=1 "$PAIR" continue demo >/dev/null 2>&1 )
grep -q '^continue=demo$' "$MK" 2>/dev/null && pass "compact: marker continue=slug" || fail "compact: marker missing continue="
grep -q '^new_session=1$' "$MK" 2>/dev/null && pass "compact: marker new_session=1" || fail "compact: marker missing new_session"
grep -q '^tag=demo$'      "$MK" 2>/dev/null && pass "compact: marker tag=demo"      || fail "compact: marker missing tag"
ls "$CRT/xdg/pair"/parked-scrollback-demo-*.raw >/dev/null 2>&1 && pass "compact: scrollback parked" || fail "compact: no parked scrollback"
[ -s "$CRT/xdg/pair/scrollback-demo-claude.raw" ] && pass "compact: original .raw intact (copy)" || fail "compact: original .raw lost (should copy)"

# 2. in-session + invalid slug → exit 1, NO marker, NO kill
rm -f "$MK"
( cd "$CRT" && compact_env PAIR_FORCE_IN_SESSION=1 "$PAIR" continue bogus >/dev/null 2>&1 ); rc=$?
{ [ "$rc" -eq 1 ] && [ ! -f "$MK" ]; } && pass "compact: invalid slug errors without killing" || fail "compact: invalid slug should exit 1 + no marker (rc=$rc)"

# 3. real tag-match predicate via PAIR_FAKE_IN_ZELLIJ (ancestry faked, tag-match REAL)
seed_sb; rm -f "$MK"
( cd "$CRT" && compact_env PAIR_FAKE_IN_ZELLIJ=1 ZELLIJ_SESSION_NAME=pair-demo "$PAIR" continue demo >/dev/null 2>&1 )
[ -f "$MK" ] && pass "compact: tag-match (pair-demo) triggers compaction" || fail "compact: tag-match should compact"
rm -f "$MK"
( cd "$CRT" && compact_env PAIR_FAKE_IN_ZELLIJ=1 ZELLIJ_SESSION_NAME=pair-other "$PAIR" continue demo >/dev/null 2>&1 ); rc=$?
{ [ ! -f "$MK" ] && [ "$rc" -ne 0 ]; } && pass "compact: tag-MISMATCH does not compact (falls to guard)" || fail "compact: tag-mismatch should NOT compact (rc=$rc, marker=$( [ -f "$MK" ] && echo present || echo absent ))"

# 4. park_scrollback move mode (quit path) via dispatcher — original REMOVED
seed_sb
( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" \
  PAIR_TEST_CALL=park_scrollback PAIR_TEST_ARGS="demo claude" "$PAIR" >/dev/null 2>&1 )
{ ls "$CRT/xdg/pair"/parked-scrollback-demo-*.raw >/dev/null 2>&1 && [ ! -f "$CRT/xdg/pair/scrollback-demo-claude.raw" ]; } \
  && pass "park_scrollback: move mode removes original" || fail "park_scrollback: move should remove original"

# 5. handle_restart_marker continue= → re-exec argv
cat > "$CRT/.cache/pair/restart-pair-demo" <<'EOF'
tag=demo
agent=claude
new_session=1
continue=demo
EOF
printf '{"args":["--dangerously-skip-permissions"],"session_id":"x"}' > "$CRT/xdg/pair/config-demo-claude.json"
CAP="$CRT/reexec.txt"
( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" SESSION=pair-demo \
  PAIR_TEST_CALL=handle_restart_marker PAIR_REEXEC_CAPTURE="$CAP" "$PAIR" >/dev/null 2>&1 )
grep -Eq 'continue demo claude -- .*--dangerously-skip-permissions' "$CAP" 2>/dev/null \
  && pass "restart: re-exec = continue <slug> <agent> -- <args>" || fail "restart: wrong re-exec ($(cat "$CAP" 2>/dev/null))"

if [ "$fails" -eq 0 ]; then
  printf 'PASS pair-continue-test\n'
else
  printf 'FAIL pair-continue-test (%d failure(s))\n' "$fails"
  exit 1
fi
