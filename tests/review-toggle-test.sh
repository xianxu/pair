#!/usr/bin/env bash
# tests/review-toggle-test.sh — pair-review-toggle's mode-aware branch (#66 M3):
#   no review open  → focus the draft (by id) + open `:PairReview `
#   review visible  → hide-floating-panes  (state line2 → hidden)
#   review hidden   → show-floating-panes  (state line2 → visible)
# and NEVER toggle-floating-panes (the footgun). zellij is stubbed; jq is real.
#
# Run: bash tests/review-toggle-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-toggle-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
STATE="$RT/review-test.open"; ZLOG="$RT/zlog"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

cat > "$RT/panes.json" <<'JSON'
{"t":{"panes":[
  {"id":3,"is_plugin":false,"is_floating":false,"title":"draft"},
  {"id":7,"is_plugin":false,"is_floating":false,"title":"claude"}
]}}
JSON
mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
if [ "\$1" = action ] && [ "\$2" = list-panes ]; then cat "$RT/panes.json"; exit 0; fi
printf '%s\n' "\$*" >> "$ZLOG"
EOF
chmod +x "$RT/bin/zellij"
run_toggle() { PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test "$ROOT/bin/pair-review-toggle"; }

# ── branch 1: no review open → file-select ────────────────────────────────────
rm -f "$STATE"; : > "$ZLOG"; run_toggle
grep -q '^action focus-pane-id 3$' "$ZLOG" && pass "file-select: focuses the draft by id" || fail "no draft focus"
grep -q '^action write 27$' "$ZLOG" && pass "file-select: ESC to normal mode" || fail "no ESC"
grep -q 'write-chars :PairReview' "$ZLOG" && pass "file-select: opens :PairReview cmdline" || fail "no :PairReview inject"

# ── branch 2: review visible → hide ───────────────────────────────────────────
printf '%s\nvisible\n' "$$" > "$STATE"; : > "$ZLOG"; run_toggle
grep -q '^action hide-floating-panes$' "$ZLOG" && pass "visible → hide-floating-panes" || fail "no hide"
[ "$(sed -n 2p "$STATE")" = hidden ] && pass "state flips to hidden" || fail "state not hidden"

# ── branch 3: review hidden → show ────────────────────────────────────────────
printf '%s\nhidden\n' "$$" > "$STATE"; : > "$ZLOG"; run_toggle
grep -q '^action show-floating-panes$' "$ZLOG" && pass "hidden → show-floating-panes" || fail "no show"
[ "$(sed -n 2p "$STATE")" = visible ] && pass "state flips to visible" || fail "state not visible"

# ── the footgun is locked out everywhere ──────────────────────────────────────
grep -q 'toggle-floating-panes' "$ZLOG" && fail "used toggle-floating-panes (footgun)" || pass "never toggle-floating-panes"

# ── config lint ───────────────────────────────────────────────────────────────
grep -q 'bind "Alt r"' "$ROOT/zellij/config.kdl" && pass "Alt+r bound in config.kdl" || fail "no Alt+r bind"
grep -q 'Run "pair-review-toggle"' "$ROOT/zellij/config.kdl" && pass "Alt+r runs pair-review-toggle" || fail "Alt+r target wrong"

[ "$fails" -eq 0 ] || { printf 'review-toggle-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-toggle-test ok\n'
