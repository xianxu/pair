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

if [ "$fails" -eq 0 ]; then
  printf 'PASS pair-continue-test\n'
else
  printf 'FAIL pair-continue-test (%d failure(s))\n' "$fails"
  exit 1
fi
