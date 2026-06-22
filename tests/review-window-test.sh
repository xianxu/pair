#!/usr/bin/env bash
# tests/review-window-test.sh — the M3 review pane wiring (#66 Task 1):
#   1. :PairReview exists in the draft init with complete=file
#   2. pair-review-open validates + spawns `zellij run --floating ... nvim -u review.lua`
#   3. nvim/review.lua, on a real doc, starts the review (Alt+Return map),
#      writes the open-state file, renders 🤖 markers, and wires Alt+a/r/q.
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
  cat > syntax.md <<'MD'
In the context of agentic coding, 🤖<in the end of the day>{at the end of the day}, you assemble some system prompts to prime how an 🤖<LLM based intelligent agent work>{LLM-based intelligent agent works} in a session. **That's the mechanism for using the LLM -- the new stochastic computer.** And the current generation of 🤖<LLM based agents>{LLM-based agents} 🤖<use [agent skill](https://agentskills.io/home) convention>{use the [agent skill](https://agentskills.io/home) convention} for constructing 🤖<such prompt>{such prompts} (aka context). The convention works roughly as follows: there is a folder with the name of the skill, and within that folder a file named `SKILL.md` serves as the entry point. `SKILL.md` contains a 🤖<front-matter formatted meta-data>{frontmatter-formatted metadata} section at the top. The body is prose describing what the agent is supposed to do. One front-matter field is of particular interest: `description`, which is always loaded into the agent's context at startup. The rest of the document is pulled into context later in the session -- 🤖<by the agent's own determination>{at the agent's own discretion}, based on that description.

The skill's main prose should remain normal.
MD
  git add doc.md; git commit -q -m init )
cat > "$RT/syntax.lua" <<'LUA'
local OUT = io.open(os.getenv('SYNTAX_RESULT'), 'w')
local bad_html = false
for _, id in ipairs(vim.fn.synstack(3, 10)) do
  local name = vim.fn.synIDattr(id, 'name')
  if name == 'htmlTag' or name == 'htmlString' then bad_html = true end
end
OUT:write((not bad_html and 'review-markers-not-html\n') or 'NO-review-markers-not-html\n')
OUT:close()
vim.cmd('qa!')
LUA
( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
    PAIR_HOME="$ROOT" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" DOCFLOW_ARGLOG="$RT/doclog" \
    SYNTAX_RESULT="$RT/syntax-result" PANES_JSON="$RT/panes.json" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/review.lua" "$REPO/syntax.md" -c "luafile $RT/syntax.lua" )
grep -q '^review-markers-not-html$' "$RT/syntax-result" && pass "review markers do not poison markdown HTML syntax" || fail "review marker html syntax bleed"
cat > "$RT/wdriver.lua" <<'LUA'
local function check()
  local OUT = io.open(os.getenv('RESULT2'), 'w')
  local pane = _G.PairReviewPane ~= nil
  local map = vim.fn.maparg('<M-CR>', 'n') ~= ''
  local mapa = vim.fn.maparg('<M-a>', 'n') ~= ''
  local mapr = vim.fn.maparg('<M-r>', 'n') ~= ''
  local mapqn = vim.fn.maparg('<M-q>', 'n') ~= ''
  local mapqi = vim.fn.maparg('<M-q>', 'i') ~= ''
  local mapqx = vim.fn.maparg('<M-q>', 'x') ~= ''
  local mapo = vim.fn.maparg('<M-o>', 'n') == ''
  local mapshiftcr = vim.fn.maparg('<M-S-CR>', 'n') ~= ''
  local ship_cmd = vim.api.nvim_get_commands({}).PairReviewShip ~= nil
  local sf = _G.PairReviewPane and _G.PairReviewPane.state_file()
  local sf_ok = sf and (vim.uv or vim.loop).fs_stat(sf) ~= nil
  local buf = vim.api.nvim_get_current_buf()
  local ns = vim.api.nvim_create_namespace('review_markers')
  local marks = vim.api.nvim_buf_get_extmarks(buf, ns, 0, -1, {})
  local apply = dofile(vim.env.PAIR_HOME .. '/nvim/review/apply.lua')
  OUT:write((pane and 'pane-loaded\n') or 'NO-pane\n')
  OUT:write((map and 'altcr-map\n') or 'NO-altcr\n')
  OUT:write((mapa and mapr and mapqn and mapqi and mapqx and 'review-alt-maps\n') or 'NO-review-alt-maps\n')
  OUT:write((mapo and 'no-alt-o-map\n') or 'HAS-alt-o-map\n')
  OUT:write((mapshiftcr and 'mode-menu-map\n') or 'NO-mode-menu-map\n')
  OUT:write((ship_cmd and 'ship-cmd\n') or 'NO-ship-cmd\n')
  OUT:write((sf_ok and 'state-file\n') or 'NO-state\n')
  OUT:write(((#marks >= 1) and 'markers\n') or 'NO-markers\n')
  OUT:write((vim.o.clipboard:find('unnamedplus', 1, true) and 'review-clipboard\n') or ('NO-review-clipboard ' .. vim.o.clipboard .. '\n'))
  OUT:write((vim.o.guicursor:find('blinkon', 1, true) and 'review-blink-cursor\n') or ('NO-review-blink-cursor ' .. vim.o.guicursor .. '\n'))
  OUT:write((vim.o.breakindent and 'review-breakindent\n') or 'NO-review-breakindent\n')
  OUT:write((vim.o.smoothscroll and 'review-smoothscroll\n') or 'NO-review-smoothscroll\n')
  OUT:write((vim.g.colors_name and 'review-colorscheme\n') or 'NO-review-colorscheme\n')
  local status = vim.o.statusline
  OUT:write((status:find('🪄 Edit', 1, true) and 'mode-statusline\n') or ('NO-mode-statusline ' .. status .. '\n'))
  local function link_of(name)
    local ok, hl = pcall(vim.api.nvim_get_hl, 0, { name = name, link = true })
    return ok and hl.link or nil
  end
  local quoted = vim.api.nvim_get_hl(0, { name = 'ParleyReviewQuoted', link = false })
  local strike = vim.api.nvim_get_hl(0, { name = 'ParleyReviewStrike', link = false })
  OUT:write((link_of('ParleyReviewUser') == 'DiagnosticWarn' and 'review-user-hl\n') or 'NO-review-user-hl\n')
  OUT:write((link_of('ParleyReviewAgent') == 'DiagnosticInfo' and 'review-agent-hl\n') or 'NO-review-agent-hl\n')
  OUT:write((quoted.reverse and quoted.bold and 'review-quoted-hl\n') or 'NO-review-quoted-hl\n')
  OUT:write((strike.strikethrough and 'review-strike-hl\n') or 'NO-review-strike-hl\n')
  -- A failed poke (no agent pane found) must not leave the statusline spinner
  -- waiting forever. This catches mark-awaiting-before-send regressions.
  local panes_path = os.getenv('PANES_JSON')
  local original_panes
  if panes_path then
    local pf = io.open(panes_path, 'r'); original_panes = pf and pf:read('*a'); if pf then pf:close() end
    local wf = io.open(panes_path, 'w')
    wf:write('{"t":{"panes":[{"id":9,"is_plugin":false,"is_floating":true,"is_focused":true,"title":"review"}]}}')
    wf:close()
  end
  pcall(_G.PairReviewPane.finish_human_turn, buf, 'doc.md')
  local failed_poke_status = vim.o.statusline
  OUT:write((not failed_poke_status:find('⠋', 1, true) and 'failed-poke-no-spinner\n')
    or ('NO-failed-poke-no-spinner ' .. failed_poke_status .. '\n'))
  if panes_path and original_panes then
    local wf = io.open(panes_path, 'w'); wf:write(original_panes); wf:close()
  end

  -- Alt+a / Alt+r semantics: quoted agent replacement accepts new text and
  -- rejection removes the marker while keeping the original quoted text.
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'hello 🤖<old>{new}', 'bye' })
  vim.api.nvim_win_set_cursor(0, { 1, 8 })
  local ok_accept = _G.PairReviewPane and _G.PairReviewPane.resolve_at_cursor
  if ok_accept then _G.PairReviewPane.resolve_at_cursor(buf, 'accept') end
  local accept_line = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  OUT:write((accept_line == 'hello new' and 'alt-a-accept\n') or ('NO-alt-a-accept ' .. tostring(accept_line) .. '\n'))

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'hello 🤖<old>{new}', 'bye' })
  vim.api.nvim_win_set_cursor(0, { 1, 8 })
  if _G.PairReviewPane and _G.PairReviewPane.resolve_at_cursor then
    _G.PairReviewPane.resolve_at_cursor(buf, 'reject')
  end
  local reject_line = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  OUT:write((reject_line == 'hello old' and 'alt-r-reject\n') or ('NO-alt-r-reject ' .. tostring(reject_line) .. '\n'))

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'one 🤖<a>{A} two 🤖<b>{B}' })
  vim.api.nvim_win_set_cursor(0, { 1, 24 })
  if _G.PairReviewPane and _G.PairReviewPane.resolve_at_cursor then
    _G.PairReviewPane.resolve_at_cursor(buf, 'accept')
  end
  local second_marker_line = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  OUT:write((second_marker_line == 'one 🤖<a>{A} two B' and 'alt-a-under-cursor\n') or ('NO-alt-a-under-cursor ' .. tostring(second_marker_line) .. '\n'))

  -- If Alt+a is pressed on agent-applied styling but outside any 🤖 marker, it
  -- clears that styling/diagnosis instead of reporting "no marker".
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'alpha', 'beta', 'gamma' })
  apply.apply(buf, {
    { old = 'alpha', occurrence = 1, new = 'ALPHA', explain = 'first' },
    { old = 'gamma', occurrence = 1, new = 'GAMMA', explain = 'third' },
  })
  vim.api.nvim_win_set_cursor(0, { 1, 0 })
  if _G.PairReviewPane and _G.PairReviewPane.resolve_at_cursor then
    _G.PairReviewPane.resolve_at_cursor(buf, 'accept')
  end
  local clear_marks = vim.api.nvim_buf_get_extmarks(buf, apply.HL, 0, -1, {})
  local clear_diags = vim.diagnostic.get(buf, { namespace = apply.DIAG })
  OUT:write((#clear_marks == 1 and clear_marks[1][2] == 2 and #clear_diags == 1 and clear_diags[1].lnum == 2
    and 'alt-a-clear-style\n') or ('NO-alt-a-clear-style marks=' .. #clear_marks .. ' diags=' .. #clear_diags .. '\n'))

  -- Alt+q in normal/insert inserts a bare human comment marker and leaves the
  -- cursor inside the brackets; visual Alt+q wraps the selection as quoted text.
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'abc' })
  vim.api.nvim_win_set_cursor(0, { 1, 2 })
  if _G.PairReviewPane and _G.PairReviewPane.insert_review_marker then
    _G.PairReviewPane.insert_review_marker(buf)
  end
  local inserted = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  local cur = vim.api.nvim_win_get_cursor(0)
  OUT:write((inserted == 'ab🤖[]c' and cur[2] == 7 and 'alt-q-insert\n') or ('NO-alt-q-insert ' .. tostring(inserted) .. ' col=' .. tostring(cur[2]) .. '\n'))

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'alpha beta' })
  if _G.PairReviewPane and _G.PairReviewPane.quote_selection then
    _G.PairReviewPane.quote_selection(buf, { 1, 0 }, { 1, 5 })
  end
  local quoted = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  cur = vim.api.nvim_win_get_cursor(0)
  OUT:write((quoted == '🤖<alpha>[] beta' and cur[2] == 12 and 'alt-q-visual\n') or ('NO-alt-q-visual ' .. tostring(quoted) .. ' col=' .. tostring(cur[2]) .. '\n'))

  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'a > b ] c \\ d tail' })
  if _G.PairReviewPane and _G.PairReviewPane.quote_selection then
    _G.PairReviewPane.quote_selection(buf, { 1, 0 }, { 1, 13 })
  end
  quoted = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  vim.api.nvim_win_set_cursor(0, { 1, 1 })
  if _G.PairReviewPane and _G.PairReviewPane.resolve_at_cursor then
    _G.PairReviewPane.resolve_at_cursor(buf, 'reject')
  end
  local rejected_quote = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
  OUT:write((rejected_quote == 'a > b ] c \\ d tail' and 'alt-q-escaped-visual\n') or ('NO-alt-q-escaped-visual ' .. tostring(quoted) .. ' -> ' .. tostring(rejected_quote) .. '\n'))

  OUT:close()
  -- drive Alt+Return (finish human turn): edit → human_round + poke
  apply.apply(buf, { { old = 'tail', occurrence = 1, new = 'TAIL', explain = 'stale' } })
  vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'a human edit' })
  pcall(_G.PairReviewPane.finish_human_turn, buf, 'doc.md')
  local post_submit_marks = vim.api.nvim_buf_get_extmarks(buf, apply.HL, 0, -1, {})
  local post_submit_diags = vim.diagnostic.get(buf, { namespace = apply.DIAG })
  local OUT2 = io.open(os.getenv('RESULT2'), 'a')
  OUT2:write((#post_submit_marks == 0 and #post_submit_diags == 0 and 'human-submit-clears-style\n')
    or ('NO-human-submit-clears-style marks=' .. #post_submit_marks .. ' diags=' .. #post_submit_diags .. '\n'))
  OUT2:close()
  pcall(vim.cmd, 'PairReviewShip')
  pcall(_G.PairReviewPane.finish_human_turn, buf, 'doc.md', 'proofread', 'keep the title')
  vim.api.nvim_set_current_buf(buf)
  vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'menu submit edit' })
  local h = _G.PairReviewPane.open_mode_menu('doc.md')
  if h then h.submit() end
  vim.cmd('qa!')
end
if vim.v.vim_did_enter == 1 then vim.schedule(check)
else vim.api.nvim_create_autocmd('VimEnter', { callback = function() vim.schedule(check) end }) end
LUA
: > "$RT/zlog"
: > "$RT/doclog"
( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
    PAIR_HOME="$ROOT" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" DOCFLOW_ARGLOG="$RT/doclog" RESULT2="$RT/r3" \
    PANES_JSON="$RT/panes.json" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/review.lua" "$REPO/doc.md" -c "luafile $RT/wdriver.lua" )
grep -q 'pane-loaded' "$RT/r3" && pass "review.lua loaded the review core" || fail "review.lua did not load"
grep -q 'altcr-map' "$RT/r3" && pass "Alt+Return keymap wired" || fail "no Alt+Return keymap"
grep -q 'review-alt-maps' "$RT/r3" && pass "Alt+a/Alt+r/Alt+q review maps wired" || fail "review Alt maps missing"
grep -q '^no-alt-o-map$' "$RT/r3" && pass "Alt+o is not bound in review pane" || fail "Alt+o still bound"
grep -q '^mode-menu-map$' "$RT/r3" && pass "Alt+Shift+Return send menu keymap wired" || fail "send menu map missing"
grep -q '^ship-cmd$' "$RT/r3" && pass ":PairReviewShip command wired" || fail ":PairReviewShip missing"
grep -q '^state-file$' "$RT/r3" && pass "open-state file written" || fail "no state file"
grep -q '^markers$' "$RT/r3" && pass "🤖 markers rendered" || fail "no marker extmarks"
grep -q '^review-clipboard$' "$RT/r3" && pass "review pane yanks to system clipboard" || fail "review clipboard option"
grep -q '^review-blink-cursor$' "$RT/r3" && pass "review pane uses blinking cursor" || fail "review cursor blink option"
grep -q '^review-breakindent$' "$RT/r3" && pass "review pane indents soft-wrapped lines" || fail "review breakindent option"
grep -q '^review-smoothscroll$' "$RT/r3" && pass "review pane smooth-scrolls soft-wrapped lines" || fail "review smoothscroll option"
grep -q '^review-colorscheme$' "$RT/r3" && pass "review pane loads a colorscheme" || fail "review colorscheme not loaded"
grep -q '^mode-statusline$' "$RT/r3" && pass "review statusline shows current mode" || fail "review statusline missing mode"
grep -q '^review-user-hl$' "$RT/r3" && pass "review user marker highlight matches parley" || fail "review user marker highlight"
grep -q '^review-agent-hl$' "$RT/r3" && pass "review agent marker highlight matches parley" || fail "review agent marker highlight"
grep -q '^review-quoted-hl$' "$RT/r3" && pass "review quoted marker highlight matches parley" || fail "review quoted marker highlight"
grep -q '^review-strike-hl$' "$RT/r3" && pass "review strike marker highlight matches parley" || fail "review strike marker highlight"
grep -q '^failed-poke-no-spinner$' "$RT/r3" && pass "failed poke does not leave spinner awaiting" || fail "failed-poke spinner behavior"
grep -q '^alt-a-accept$' "$RT/r3" && pass "Alt+a accepts quoted agent replacement" || fail "Alt+a accept behavior"
grep -q '^alt-r-reject$' "$RT/r3" && pass "Alt+r rejects to original quoted text" || fail "Alt+r reject behavior"
grep -q '^alt-a-under-cursor$' "$RT/r3" && pass "Alt+a resolves the marker under cursor" || fail "Alt+a marker-under-cursor behavior"
grep -q '^alt-a-clear-style$' "$RT/r3" && pass "Alt+a clears agent styling outside markers" || fail "Alt+a clear styling behavior"
grep -q '^alt-q-insert$' "$RT/r3" && pass "Alt+q inserts a human comment marker" || fail "Alt+q insert behavior"
grep -q '^alt-q-visual$' "$RT/r3" && pass "Alt+q wraps visual selection as quoted marker" || fail "Alt+q visual behavior"
grep -q '^alt-q-escaped-visual$' "$RT/r3" && pass "Alt+q quoted marker preserves delimiter text" || fail "Alt+q delimiter escaping"
# Alt+Return integration (M4a): the nvim SAVES the human edits but writes NO git
# (invariant #1) — the agent commits the human round; the nvim pokes the
# commit-request signal (human_finished), not a docflow round and not /xx-fix.
grep -q 'a human edit' "$REPO/doc.md" && pass "Alt+Return saves the human edits (agent commits the round)" || fail "human edit not saved"
grep -q '^human-submit-clears-style$' "$RT/r3" && pass "Alt+Return clears stale agent styling" || fail "human submit styling clear"
grep -q 'round --side human' "$RT/doclog" && fail "nvim ran a human docflow round (invariant #1: nvim writes no git)" || pass "nvim writes no git on Alt+Return"
grep -q '^ship$' "$RT/doclog" && fail "nvim ran docflow ship (invariant #1: agent owns git)" || pass "nvim writes no git on :PairReviewShip"
grep -q 'write-chars finished my edits .*Edit posture.*minimal 🤖<old>{new}/🤖{new}' "$RT/zlog" && pass "Alt+Return pokes human_finished with Edit marker rule" || fail "no direct human_finished marker-rule poke"
grep -q 'write-chars ship .*doc.md.*agent owns git' "$RT/zlog" && pass ":PairReviewShip pokes the agent ship request" || fail "no ship-request poke"
grep -q 'write-chars finished my edits .*Proofread posture.*keep the title' "$RT/zlog" && pass "menu send pokes human_finished with mode and instruction" || fail "no mode/instruction human_finished poke"
grep -q 'menu submit edit' "$REPO/doc.md" && pass "send menu submit saves the reviewed document buffer" || fail "send menu submit did not save reviewed document"
grep -q 'Review workbench open on' "$RT/zlog" && fail "review pane still sends redundant open handshake" || pass "review pane does not send redundant open handshake"

[ "$fails" -eq 0 ] || { printf 'review-window-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-window-test ok\n'
