#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
# Integration test for `pair rename <old> <new>` (issue #000022, M1).
#
# Builds a fixtured $PAIR_DATA_DIR with representative tag-scoped files,
# exercises the rename CLI, and verifies post-state. Exits 0 on pass,
# non-zero on fail.
#
# Run:  bash tests/pair-rename.sh
#
# The substring case (tag `brain` vs `brain-2`) is the marquee correctness
# property — a naive glob would sweep both; the implementation must touch
# only the exact tag.

set -euo pipefail

SELF_DIR="$(cd -P "$(dirname "$0")" && pwd)"
fixture_root="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename.XXXXXX")" || exit 1
trap 'rm -rf "${fixture_root:?}"' EXIT
PAIR_BIN="$fixture_root/pair"
bash "$SELF_DIR/lib/pair-test-binary.sh" "$PAIR_BIN"

pass=0
fail=0
case_name=""

red()    { printf '\033[31m%s\033[0m' "$*"; }
green()  { printf '\033[32m%s\033[0m' "$*"; }
yellow() { printf '\033[33m%s\033[0m' "$*"; }

ok()   { printf '  %s %s\n' "$(green ok)" "$1"; pass=$((pass + 1)); }
bad()  { printf '  %s %s\n' "$(red FAIL)" "$1"; fail=$((fail + 1)); }
case_begin() { case_name="$1"; printf '%s %s\n' "$(yellow ::)" "$case_name"; }

# Drop a representative file set for (tag, agent) into $1. Files mirror
# the families enumerated by launcher.renamePathsFor.
seed_tag() {
    local dd="$1" tag="$2" agent="${3:-claude}"
    : > "$dd/agent-$tag"; echo "$agent" > "$dd/agent-$tag"
    : > "$dd/agent-pid-$tag"
    : > "$dd/outer-tty-$tag"
    : > "$dd/pair-wrap-pid-$tag"
    : > "$dd/layout-mode-$tag"
    printf 'layout3\n' > "$dd/workbench-layout-$tag"
    : > "$dd/quote-$tag"
    mkdir -p "$dd/queue-$tag"; : > "$dd/queue-$tag/some-prompt.md"
    : > "$dd/draft-$tag.md"
    : > "$dd/log-$tag.md"
    : > "$dd/nvim-pid-$tag-draft"
    : > "$dd/nvim-pid-$tag-scrollback"
    printf '{"agent":"%s","args":[],"session_id":"abc"}\n' "$agent" \
        > "$dd/config-$tag-$agent.json"
    : > "$dd/scrollback-$tag-$agent.raw"
    : > "$dd/scrollback-$tag-$agent.ansi"
    : > "$dd/scrollback-$tag-$agent.viewport"
    : > "$dd/scrollback-$tag-$agent.events.jsonl"
}

# Disable zellij integration in tests: PATH-shim a stub that always
# returns no sessions. (The real zellij would refuse the rename if a
# stale session existed under either tag name; for the offline test
# matrix we want pure file-system behaviour.)
shim_zellij() {
    local shim_dir="$1"
    cat > "$shim_dir/zellij" <<'STUB'
#!/usr/bin/env bash
exit 0
STUB
    chmod +x "$shim_dir/zellij"
}

# Run `pair rename` in an isolated $PAIR_DATA_DIR with the zellij shim.
# $1 = data dir; $2.. = args to forward.
run_rename() {
    local dd="$1"; shift
    local shim
    shim="$(mktemp -d "${fixture_root:?}/shim.XXXXXX")" || exit 1
    shim_zellij "$shim"
    PAIR_DATA_DIR="$dd" PATH="$shim:$PATH" "$PAIR_BIN" rename "$@"
    local rc=$?
    rm -rf "$shim"
    return $rc
}

# Assertions.
assert_gone() {
    if [ -e "$1" ]; then bad "expected gone: $1"; else ok "gone: $1"; fi
}
assert_exists() {
    if [ -e "$1" ]; then ok "exists: $1"; else bad "expected exists: $1"; fi
}
assert_exits_nonzero() {
    if "$@" >/dev/null 2>&1; then bad "expected non-zero exit: $*"; else ok "rejected: $*"; fi
}

# ── T1: clean rename ─────────────────────────────────────────────────────────
case_begin "T1 clean rename"
DD="$(mktemp -d "${fixture_root:?}/t1.XXXXXX")" || exit 1
seed_tag "$DD" t1 claude
run_rename "$DD" t1 t2 >/dev/null
for f in agent-t1 agent-pid-t1 outer-tty-t1 draft-t1.md log-t1.md \
         workbench-layout-t1 \
         config-t1-claude.json scrollback-t1-claude.raw queue-t1 \
         nvim-pid-t1-draft; do
    assert_gone "$DD/$f"
done
for f in agent-t2 agent-pid-t2 outer-tty-t2 draft-t2.md log-t2.md \
         workbench-layout-t2 \
         config-t2-claude.json scrollback-t2-claude.raw queue-t2 \
         nvim-pid-t2-draft; do
    assert_exists "$DD/$f"
done
# Content preserved
agent_content="$(cat "$DD/agent-t2")"
[ "$agent_content" = "claude" ] && ok "agent file content preserved" \
    || bad "agent file content lost: '$agent_content'"
rm -rf "$DD"

# ── T2: substring safety (brain vs brain-2) — the marquee correctness case ──
case_begin "T2 substring safety (brain vs brain-2)"
DD="$(mktemp -d "${fixture_root:?}/t2.XXXXXX")" || exit 1
seed_tag "$DD" brain claude
seed_tag "$DD" brain-2 claude
# Rename only `brain`. `brain-2`'s files must be untouched.
run_rename "$DD" brain new-brain >/dev/null
# brain-2 must be intact.
for f in agent-brain-2 outer-tty-brain-2 draft-brain-2.md \
         config-brain-2-claude.json scrollback-brain-2-claude.raw \
         queue-brain-2; do
    assert_exists "$DD/$f"
done
# brain is gone, new-brain is present.
for f in agent-brain config-brain-claude.json scrollback-brain-claude.raw; do
    assert_gone "$DD/$f"
done
for f in agent-new-brain config-new-brain-claude.json \
         scrollback-new-brain-claude.raw; do
    assert_exists "$DD/$f"
done
rm -rf "$DD"

# ── T3: NEW tag occupied → refuse ────────────────────────────────────────────
case_begin "T3 refuse when new tag is occupied"
DD="$(mktemp -d "${fixture_root:?}/t3.XXXXXX")" || exit 1
seed_tag "$DD" src claude
seed_tag "$DD" dst claude
assert_exits_nonzero run_rename "$DD" src dst
# Nothing moved.
assert_exists "$DD/agent-src"
assert_exists "$DD/agent-dst"
rm -rf "$DD"

# ── T4: OLD tag empty → refuse ───────────────────────────────────────────────
case_begin "T4 refuse when old tag has no files"
DD="$(mktemp -d "${fixture_root:?}/t4.XXXXXX")" || exit 1
assert_exits_nonzero run_rename "$DD" ghost newname
rm -rf "$DD"

# ── T5: same tag → refuse ────────────────────────────────────────────────────
case_begin "T5 refuse when old == new"
DD="$(mktemp -d "${fixture_root:?}/t5.XXXXXX")" || exit 1
seed_tag "$DD" t1 claude
assert_exits_nonzero run_rename "$DD" t1 t1
rm -rf "$DD"

# ── T6: invalid charset → refuse ─────────────────────────────────────────────
case_begin "T6 refuse invalid tag charset"
DD="$(mktemp -d "${fixture_root:?}/t6.XXXXXX")" || exit 1
seed_tag "$DD" t1 claude
assert_exits_nonzero run_rename "$DD" t1 'bad name'
assert_exits_nonzero run_rename "$DD" t1 'bad/slash'
assert_exits_nonzero run_rename "$DD" '' newname
# After all the refusals, t1's files are still in place.
assert_exists "$DD/agent-t1"
rm -rf "$DD"

# ── T7: pair- is literal tag content, not a removable prefix ────────────────
case_begin "T7 preserves exact pair-prefixed identities"
DD="$(mktemp -d "${fixture_root:?}/t7.XXXXXX")" || exit 1
seed_tag "$DD" t1 claude
seed_tag "$DD" t2 claude
seed_tag "$DD" pair-t1 claude
printf 'bare source\n' > "$DD/draft-t1.md"
printf 'bare destination\n' > "$DD/draft-t2.md"
printf 'exact source\n' > "$DD/draft-pair-t1.md"
run_rename "$DD" pair-t1 pair-t2 >/dev/null
assert_gone "$DD/agent-pair-t1"
assert_exists "$DD/agent-pair-t2"
assert_exists "$DD/agent-t1"
assert_exists "$DD/agent-t2"
for entry in 't1:bare source' 't2:bare destination' 'pair-t2:exact source'; do
    tag="${entry%%:*}"; expected="${entry#*:}"
    actual="$(cat "$DD/draft-$tag.md")"
    [ "$actual" = "$expected" ] && ok "exact content: $tag" \
        || bad "wrong content for $tag: $actual"
done
# Once the exact source is gone, a bare lookalike must not become a fallback.
assert_exits_nonzero run_rename "$DD" pair-t1 another
assert_exists "$DD/draft-t1.md"
assert_gone "$DD/draft-another.md"
rm -rf "$DD"

# ── T8: multiple agents under one tag ────────────────────────────────────────
case_begin "T8 renames all agents under one tag"
DD="$(mktemp -d "${fixture_root:?}/t8.XXXXXX")" || exit 1
seed_tag "$DD" multi claude
# Add codex + agy on top (overwriting the agent-multi file is fine).
printf '{"agent":"codex","args":[],"session_id":"x"}\n' \
    > "$DD/config-multi-codex.json"
: > "$DD/scrollback-multi-codex.raw"
printf '{"agent":"agy","args":[],"session_id":"x"}\n' \
    > "$DD/config-multi-agy.json"
: > "$DD/scrollback-multi-agy.raw"
run_rename "$DD" multi renamed >/dev/null
for a in claude codex agy; do
    assert_gone   "$DD/config-multi-$a.json"
    assert_exists "$DD/config-renamed-$a.json"
    assert_gone   "$DD/scrollback-multi-$a.raw"
    assert_exists "$DD/scrollback-renamed-$a.raw"
done
rm -rf "$DD"

# ── T9: --restart-check validates but doesn't move ──────────────────────────
case_begin "T9 --restart-check is a dry-run validate"
DD="$(mktemp -d "${fixture_root:?}/t9.XXXXXX")" || exit 1
seed_tag "$DD" src claude
run_rename "$DD" --restart-check src dst >/dev/null
# Files untouched.
assert_exists "$DD/agent-src"
assert_gone   "$DD/agent-dst"
# Same call with occupied dst should refuse.
seed_tag "$DD" dst claude
assert_exits_nonzero run_rename "$DD" --restart-check src dst
rm -rf "$DD"

# ── Summary ──────────────────────────────────────────────────────────────────
echo
printf 'pair rename: %s passed, %s failed\n' "$(green "$pass")" "$([ "$fail" -eq 0 ] && green "$fail" || red "$fail")"
[ "$fail" -eq 0 ]
