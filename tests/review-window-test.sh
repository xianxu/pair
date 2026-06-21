#!/usr/bin/env bash
# tests/review-window-test.sh — the M3 review pane wiring (#66 Task 1):
#   1. :PairReview exists in the draft init with complete=file
#   2. pair-review-open validates + spawns `zellij run --floating ... nvim -u review.lua`
#   3. nvim/review.lua, on a real doc, starts the review (Alt+Return map),
#      writes the open-state file, and renders 🤖 markers.
# Live zellij pane behaviour is manual smoke; here zellij/tput/docflow are stubbed.
#
# Run: bash tests/review-window-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-window-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# stubs: zellij records argv; tput echoes a size
mkdir -p "$RT/bin"
cat > "$RT/panes.json" <<'JSON'
{"t":{"panes":[
  {"id":7,"is_plugin":false,"is_floating":false,"is_focused":false,"title":"claude"},
  {"id":3,"is_plugin":false,"is_floating":false,"is_focused":false,"title":"draft"},
  {"id":9,"is_plugin":false,"is_floating":true,"is_focused":true,"title":"review"}
]}}
JSON
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
if [ "\$1" = action ] && [ "\$2" = list-panes ]; then cat "$RT/panes.json"; exit 0; fi
printf '%s\n' "\$*" >> "$RT/zlog"
EOF
printf '#!/usr/bin/env bash\necho 80\n' > "$RT/bin/tput"
chmod +x "$RT/bin/zellij" "$RT/bin/tput"

# ── 1. :PairReview command ────────────────────────────────────────────────────
printf 'draft\n' > "$RT/draft.md"
cat > "$RT/cmd.lua" <<'LUA'
local pr = vim.api.nvim_get_commands({}).PairReview
local OUT = io.open(os.getenv('RESULT'), 'w')
OUT:write((pr and pr.complete == 'file') and 'cmd ok\n' or 'cmd FAIL\n'); OUT:close()
LUA
( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" "$RT/draft.md" -c "luafile $RT/cmd.lua" -c 'qa!' )
grep -q 'cmd ok' "$RT/r1" && pass ":PairReview exists with complete=file" || fail ":PairReview command/completion"

# ── 2. pair-review-open ───────────────────────────────────────────────────────
if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
  fail "pair-review-open should error on a missing file"
else
  pass "pair-review-open errors on a missing file"
fi
echo hi > "$RT/doc.md"; : > "$RT/zlog"
PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
z="$(cat "$RT/zlog")"
case "$z" in *"run --floating"*) pass "spawns a floating pane";; *) fail "no floating run ($z)";; esac
case "$z" in *"nvim -u $ROOT/nvim/review.lua"*) pass "launches nvim -u nvim/review.lua";; *) fail "no review.lua launch ($z)";; esac

# ── 3. nvim/review.lua on a real doc ──────────────────────────────────────────
REPO="$RT/repo"; mkdir -p "$REPO"
( cd "$REPO"
  git init -q; git config user.email t@e.com; git config user.name T
  printf 'hello 🤖[review this]\nworld\n' > doc.md
  git add doc.md; git commit -q -m init )
cat > "$RT/wdriver.lua" <<'LUA'
local function check()
  local OUT = io.open(os.getenv('RESULT2'), 'w')
  local pane = _G.PairReviewPane ~= nil
  local map = vim.fn.maparg('<M-CR>', 'n') ~= ''
  local sf = _G.PairReviewPane and _G.PairReviewPane.state_file()
  local sf_ok = sf and (vim.uv or vim.loop).fs_stat(sf) ~= nil
  local buf = vim.api.nvim_get_current_buf()
  local ns = vim.api.nvim_create_namespace('review_markers')
  local marks = vim.api.nvim_buf_get_extmarks(buf, ns, 0, -1, {})
  OUT:write((pane and 'pane-loaded\n') or 'NO-pane\n')
  OUT:write((map and 'altcr-map\n') or 'NO-altcr\n')
  OUT:write((sf_ok and 'state-file\n') or 'NO-state\n')
  OUT:write(((#marks >= 1) and 'markers\n') or 'NO-markers\n')
  OUT:close()
  -- drive Alt+Return (finish human turn): edit → human_round + poke
  vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'a human edit' })
  pcall(_G.PairReviewPane.finish_human_turn, buf, 'doc.md')
  vim.cmd('qa!')
end
if vim.v.vim_did_enter == 1 then vim.schedule(check)
else vim.api.nvim_create_autocmd('VimEnter', { callback = function() vim.schedule(check) end }) end
LUA
: > "$RT/zlog"
( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
    PAIR_HOME="$ROOT" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" DOCFLOW_ARGLOG="$RT/doclog" RESULT2="$RT/r3" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/review.lua" "$REPO/doc.md" -c "luafile $RT/wdriver.lua" )
grep -q 'pane-loaded' "$RT/r3" && pass "review.lua loaded the review core" || fail "review.lua did not load"
grep -q 'altcr-map' "$RT/r3" && pass "Alt+Return keymap wired" || fail "no Alt+Return keymap"
grep -q '^state-file$' "$RT/r3" && pass "open-state file written" || fail "no state file"
grep -q '^markers$' "$RT/r3" && pass "🤖 markers rendered" || fail "no marker extmarks"
# Alt+Return integration (M4a): the nvim SAVES the human edits but writes NO git
# (invariant #1) — the agent commits the human round; the nvim pokes the
# commit-request signal (human_finished), not a docflow round and not /xx-fix.
grep -q 'a human edit' "$REPO/doc.md" && pass "Alt+Return saves the human edits (agent commits the round)" || fail "human edit not saved"
grep -q 'round --side human' "$RT/doclog" && fail "nvim ran a human docflow round (invariant #1: nvim writes no git)" || pass "nvim writes no git on Alt+Return"
grep -q 'write-chars finished my edits' "$RT/zlog" && pass "Alt+Return pokes the agent commit-request signal (human_finished)" || fail "no commit-request poke"

[ "$fails" -eq 0 ] || { printf 'review-window-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-window-test ok\n'
