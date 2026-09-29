#!/usr/bin/env bash
# tests/review-toggle-test.sh — the Alt+c review/collaboration toggle, now a draft-nvim lua
# fn (#66 M3 rework; the old transient pair-review-toggle floating pane caused
# the open delay / auto-hide / half-size / mis-fire smoke bugs).
#
#   _pair_review_toggle_action(alive, visible) (pure):
#     not alive          → 'open'   (file-select)
#     alive  + visible   → 'hide'
#     alive  + hidden    → 'show'
#   PairReviewToggle() (integration, zellij stubbed on $PATH):
#     resolved identity + acknowledged pane + visible → hide-floating-panes
#     resolved identity + acknowledged pane + hidden → show-floating-panes
#     non-review branch → file-select, including when a stale target exists
#   and NEVER toggle-floating-panes (the footgun).
#
# Live zellij pane/focus behaviour is the manual smoke (M3 plan Task 5). Here
# zellij is a $PATH stub that records argv and answers are-floating-panes-visible
# from a file the driver rewrites between branches.
#
# Run: bash tests/review-toggle-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-toggle-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"; ZLOG="$RT/zlog.txt"; SYSTEM_CALLS="$RT/session-system-calls"; FLOATVIS="$RT/floatvis"; : > "$ZLOG"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# zellij stub: record every action; answer are-floating-panes-visible from a file.
mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ZLOG"
if [ "\$1" = action ] && [ "\$2" = list-panes ]; then
  printf '%s\n' '{"panes":[{"id":7,"terminal_command":"/bin/pair wrap"},{"id":3,"terminal_command":"nvim -u /pair/nvim/init.lua"}]}'
  exit 0
fi
if [ "\$1" = action ] && [ "\$2" = are-floating-panes-visible ]; then
  cat "$FLOATVIS" 2>/dev/null || echo false
  rg -q '^true$' "$FLOATVIS" || exit 1
fi
exit 0
EOF
cat > "$RT/bin/ps" <<'EOF'
#!/usr/bin/env bash
touch "$SYSTEM_CALLS"
if [ "$1" = "-axo" ]; then
  printf '111 1\n222 111\n'
  exit 0
fi
exec /bin/ps "$@"
EOF
cat > "$RT/bin/lsof" <<'EOF'
#!/usr/bin/env bash
touch "$SYSTEM_CALLS"
if [ "$1" = "-p" ] && [ "$2" = "222" ]; then
  printf 'p222\nn%s/.codex/sessions/2026/06/21/rollout-2026-06-21T00-00-00-12345678-1234-1234-1234-123456789abc.jsonl\n' "$HOME"
  exit 0
fi
printf 'p%s\n' "${2:-}"
EOF
chmod +x "$RT/bin/zellij" "$RT/bin/ps" "$RT/bin/lsof"
cat > "$RT/bin/pair" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = session-inventory ]; then
  case " $* " in
    *" --owner "*)
      if [ -s "$INVENTORY_SID_FILE" ]; then
        cat "$INVENTORY_SID_FILE"
        printf '\n'
        exit 0
      fi
      exit 1
      ;;
  esac
  if [ -s "$INVENTORY_SID_FILE" ]; then
    sid="$(cat "$INVENTORY_SID_FILE")"
    agent="${3:-claude}"
    printf '{"schema_version":1,"forests":[{"agent":"%s","roots":[{"node_id":"root","native_id":"%s"}]}],"correlations":[{"tag":"test","agent":"%s","status":"established","root_node_id":"root"}]}\n' "$agent" "$sid" "$agent"
  else
    printf '{"schema_version":1,"forests":[],"correlations":[]}\n'
  fi
  exit 0
fi
# Only inventory is faked; editor retention and other services use the real CLI.
exec "$PAIR_HOME/bin/pair" "$@"
EOF
chmod +x "$RT/bin/pair"

printf 'draft\n' > "$RT/draft.md"
cat > "$RT/driver.lua" <<'LUA'
local OUT = io.open(os.getenv('RESULT'), 'w')
local ZLOG = os.getenv('ZLOG')
local FLOATVIS = os.getenv('FLOATVIS')
local sf = vim.env.PAIR_REVIEW_OPEN_PATH

local function read_zlog()
  local f = io.open(ZLOG, 'r'); if not f then return {} end
  local t = {}; for l in f:lines() do t[#t + 1] = l end; f:close(); return t
end
local function new_since(n)
  local all = read_zlog(); local out = {}
  for i = n + 1, #all do out[#out + 1] = all[i] end; return out
end
local function has(lines, pat)
  for _, l in ipairs(lines) do if l:find(pat, 1, true) then return true end end
  return false
end
local function setfloat(v) local f = io.open(FLOATVIS, 'w'); f:write(v); f:close() end

-- pure decision (5 cases: a live pane → hide/show; else target-driven prompt/open/wait)
local A = _G._pair_review_toggle_action
OUT:write((A(false, false, nil) == 'prompt') and 'pure-prompt ok\n' or 'pure-prompt FAIL\n')
OUT:write((A(false, false, 'ready') == 'open') and 'pure-open ok\n' or 'pure-open FAIL\n')
OUT:write((A(false, false, 'proposed') == 'wait') and 'pure-wait ok\n' or 'pure-wait FAIL\n')
OUT:write((A(true, true) == 'hide') and 'pure-hide ok\n' or 'pure-hide FAIL\n')
OUT:write((A(true, false) == 'show') and 'pure-show ok\n' or 'pure-show FAIL\n')

local R = _G._pair_review
local target = vim.env.PAIR_REVIEW_TARGET_PATH
local draft = vim.env.PAIR_DRAFT_PATH -- exists (the test wrote it)

-- conversation-scope (#66 smoke #6): a target written under a DIFFERENT session
-- (PAIR_SESSION_ID=oldsid, pre-written below) is ignored by this session (testsid),
-- so a fresh session prompts instead of reopening the previous review.
OUT:write((R.read_target() == nil) and 'session-scope ok\n' or 'session-scope FAIL\n')

local prepbin = vim.env.PAIR_DATA_DIR .. '/prep-ok'
vim.fn.writefile({
  '#!/usr/bin/env bash',
  'set -eu',
  '"' .. vim.env.PAIR_HOME .. '/bin/pair" review target "$2" ready >/dev/null',
  'printf "%s\\n" "review prepared: $2 on review/draft. Reply \\"ready\\"."',
}, prepbin)
vim.fn.system({ 'chmod', '+x', prepbin })
vim.env.PAIR_REVIEW_READINESS_BIN = prepbin
R.propose(draft)
local proposed = R.read_target()
OUT:write((proposed and proposed.status == 'ready') and 'propose-prepares-ready ok\n' or 'propose-prepares-ready FAIL\n')
vim.env.PAIR_REVIEW_READINESS_BIN = nil
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- pure target_stale: same id → fresh; different / empty-current / no-id → stale.
local TS = R.target_stale
OUT:write((TS({ session = 'testsid' }, 'testsid') == false) and 'ts-same ok\n' or 'ts-same FAIL\n')
OUT:write((TS({ session = 'oldsid' }, 'testsid') == true) and 'ts-diff ok\n' or 'ts-diff FAIL\n')
OUT:write((TS({ session = 'x' }, '') == true) and 'ts-nocur ok\n' or 'ts-nocur FAIL\n')
OUT:write((TS({}, 'testsid') == true) and 'ts-noid ok\n' or 'ts-noid FAIL\n')

-- Codex fresh-start race: :PairReview may prepare before the async session
-- watcher has discovered a session id. The target was created by this same
-- draft nvim, so it must remain readable even though it is not yet session
-- stamped; otherwise the second Alt+c falls back to :PairReview again.
vim.env.PAIR_SESSION_ID = ''
vim.env.PAIR_AGENT = 'claude'
os.remove(vim.env.PAIR_AGENT_CONFIG_PATH)
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":""}' }, target)
vim.fn.system({ 'touch', '-t', '202001010000', target })
OUT:write((R.read_target() == nil) and 'old-unscoped-target-stale ok\n' or 'old-unscoped-target-stale FAIL\n')
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":""}' }, target)
OUT:write((R.read_target() ~= nil) and 'fresh-unscoped-target-read ok\n' or 'fresh-unscoped-target-read FAIL\n')
vim.env.PAIR_SESSION_ID = 'testsid'
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- Fresh sessions learn their id after nvim starts; review-target must use only
-- the shared inventory's established projection when PAIR_SESSION_ID is empty.
vim.env.PAIR_SESSION_ID = ''
vim.fn.writefile({ 'invsid' }, vim.env.INVENTORY_SID_FILE)
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"invsid"}' }, target)
OUT:write((R.read_target() ~= nil) and 'inventory-session-read ok\n' or 'inventory-session-read FAIL\n')
R.write_target(draft, 'ready')
local written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
OUT:write((written.session == 'invsid') and 'inventory-session-write ok\n' or 'inventory-session-write FAIL\n')
vim.env.PAIR_SESSION_ID = 'testsid'
os.remove(vim.env.INVENTORY_SID_FILE)
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

vim.env.PAIR_SESSION_ID = ''
vim.env.PAIR_AGENT = 'codex'
vim.env.PAIR_AGENT_CONFIG_PATH = os.getenv('CODEX_CONFIG')
os.remove(vim.env.PAIR_AGENT_CONFIG_PATH)
vim.fn.writefile({ '111' }, vim.env.PAIR_AGENT_PID_PATH)
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"12345678-1234-1234-1234-123456789abc"}' }, target)
OUT:write((R.current_session_id() == nil) and 'no-live-codex-fallback ok\n' or 'no-live-codex-fallback FAIL\n')
OUT:write((R.read_target() == nil) and 'unverified-live-target-stale ok\n' or 'unverified-live-target-stale FAIL\n')
R.write_target(draft, 'ready')
written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
OUT:write((written.session == '') and 'unverified-live-target-unstamped ok\n' or 'unverified-live-target-unstamped FAIL\n')
OUT:write((vim.fn.filereadable(vim.env.SYSTEM_CALLS) == 0) and 'no-session-subprocess ok\n' or 'no-session-subprocess FAIL\n')
vim.env.PAIR_AGENT = 'claude'
vim.env.PAIR_SESSION_ID = 'testsid'
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- The resolver seam supplies branch facts; the pane handshake crosses a real
-- Neovim RPC connection. PID liveness alone no longer authorizes visibility.
local resolved = {status='resolved',repo=vim.fn.getcwd(),branch='review/draft',head='verified-head',file='draft.md'}
local context = {repo=resolved.repo,branch=resolved.branch,file=resolved.file,activation='test-activation'}
local metadata = {version=1,endpoint=vim.fn.serverstart(vim.env.PAIR_DATA_DIR .. '/toggle.sock'),token='test-token',session='testsid',context=context}
_G.PairReviewPane = {restore=function(raw)
  local req=vim.json.decode(raw)
  assert(req.token==metadata.token and req.session=='testsid')
  assert(vim.deep_equal(req.identity,resolved))
  return {ok=true,same=true,identity=resolved,context=context}
end}
local resolver_status='resolved'
R.client.opts.run=function(cmd,opts,cb)
  if vim.tbl_contains(cmd,'--resolve') then
    local value=vim.deepcopy(resolved); value.status=resolver_status
    vim.schedule(function() cb({code=0,stdout=vim.json.encode(value),stderr=''}) end)
    return {}
  end
  return vim.system(cmd,opts,cb)
end
local function toggle()
  _G.PairReviewToggle()
  assert(vim.wait(7000,function() return not R.client.busy end,10),'toggle activation timed out')
end
local function live()
  vim.fn.writefile({tostring(vim.fn.getpid()),draft,vim.json.encode(metadata)},sf)
end
live(); setfloat('true')
local n = #read_zlog(); toggle()
local d = new_since(n)
OUT:write((has(d, 'action are-floating-panes-visible') and has(d, 'action hide-floating-panes') and has(d, 'action focus-pane-id 3'))
  and 'hide ok\n' or 'hide FAIL\n')

live(); setfloat('false')
n = #read_zlog(); toggle()
d = new_since(n)
OUT:write(has(d, 'action show-floating-panes') and 'show ok\n' or 'show FAIL\n')

-- Non-review branches prompt even when a stale ready target exists. Cached
-- file names cannot authorize restoring another branch's review.
resolver_status='non_review'
os.remove(sf); os.remove(target)
n = #read_zlog(); toggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating') and not has(d, 'hide-floating-panes')
  and not has(d, 'show-floating-panes')) and 'prompt ok\n' or 'prompt FAIL\n')

R.write_target(draft, 'ready')
n = #read_zlog(); toggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating') and not has(d, 'show-floating-panes')) and 'target-refused ok\n' or 'target-refused FAIL\n')

R.write_target(draft, 'proposed')
n = #read_zlog(); toggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating')) and 'wait ok\n' or 'wait FAIL\n')

-- footgun: never toggle-floating-panes anywhere
OUT:write(has(read_zlog(), 'toggle-floating-panes') and 'footgun FAIL\n' or 'footgun ok\n')
OUT:close()
vim.cmd('qa!')
LUA

# a STALE review-target from a DIFFERENT conversation (session=oldsid). This session
# runs as PAIR_SESSION_ID=testsid, so read_target must ignore it (a fresh session
# prompts; an Alt+n resume — same id — would keep its target). (#66 smoke #6.)
printf '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}\n' > "$RT/review-target-test.json"
( cd "$RT" && PATH="$RT/bin:$PATH" \
    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
    PAIR_DRAFT_PATH="$RT/draft.md" PAIR_LAYOUT_MODE_PATH="$RT/layout-mode-test" \
    PAIR_REVIEW_OPEN_PATH="$RT/review-test.open" PAIR_REVIEW_TARGET_PATH="$RT/review-target-test.json" \
    PAIR_AGENT_CONFIG_PATH="$RT/config-test-claude.json" CODEX_CONFIG="$RT/config-test-codex.json" \
    PAIR_AGENT_PID_PATH="$RT/agent-pid-test" PAIR_ZELLIJ_ACTIONS_PATH="$RT/zellij-actions-test.jsonl" \
    INVENTORY_SID_FILE="$RT/inventory-sid" RESULT="$RESULT" ZLOG="$ZLOG" SYSTEM_CALLS="$SYSTEM_CALLS" FLOATVIS="$FLOATVIS" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" "$RT/draft.md" \
      -c "luafile $RT/driver.lua" )

grep -q 'session-scope ok' "$RESULT" && pass "other-session target ignored (smoke #6)" || fail "stale (other-session) target not ignored"
grep -q 'propose-prepares-ready ok' "$RESULT" && pass ":PairReview prepares target locally" || fail ":PairReview local prepare"
for c in ts-same ts-diff ts-nocur ts-noid; do
  grep -q "$c ok" "$RESULT" && pass "pure target_stale: $c" || fail "target_stale $c"
done
grep -q 'old-unscoped-target-stale ok' "$RESULT" && pass "old unscoped target remains stale" || fail "old unscoped target accepted"
grep -q 'fresh-unscoped-target-read ok' "$RESULT" && pass "same-nvim unscoped target remains readable" || fail "same-nvim unscoped target ignored"
grep -q 'inventory-session-read ok' "$RESULT" && pass "read_target uses established inventory session" || fail "read_target inventory authority"
grep -q 'inventory-session-write ok' "$RESULT" && pass "write_target stamps established inventory session" || fail "write_target inventory authority"
grep -q 'no-live-codex-fallback ok' "$RESULT" && pass "current session does not guess from live Codex files" || fail "live Codex fallback remains"
grep -q 'unverified-live-target-stale ok' "$RESULT" && pass "unverified live target is stale" || fail "unverified live target accepted"
grep -q 'unverified-live-target-unstamped ok' "$RESULT" && pass "unverified live target remains unstamped" || fail "unverified live target stamped"
grep -q 'no-session-subprocess ok' "$RESULT" && pass "session lookup launches no ps/lsof subprocess" || fail "session lookup launched ps/lsof"
grep -q 'pure-prompt ok'  "$RESULT" && pass "pure: no target → prompt"        || fail "pure prompt"
grep -q 'pure-open ok'    "$RESULT" && pass "pure: target ready → open"       || fail "pure open"
grep -q 'pure-wait ok'    "$RESULT" && pass "pure: target proposed → wait"    || fail "pure wait"
grep -q 'pure-hide ok'    "$RESULT" && pass "pure: alive+visible → hide"      || fail "pure hide"
grep -q 'pure-show ok'    "$RESULT" && pass "pure: alive+hidden → show"       || fail "pure show"
grep -q '^hide ok$'       "$RESULT" && pass "live+visible → hide-floating-panes" || fail "hide branch"
grep -q '^show ok$'       "$RESULT" && pass "live+hidden → show-floating-panes" || fail "show branch"
grep -q '^prompt ok$'     "$RESULT" && pass "no target → :PairReview prompt (no open/show/hide)" || fail "prompt branch"
grep -q '^target-refused ok$' "$RESULT" && pass "stale ready target cannot override non-review branch" || fail "stale target admitted"
grep -q '^wait ok$'       "$RESULT" && pass "target proposed → wait (no open)" || fail "wait branch"
grep -q '^footgun ok$'    "$RESULT" && pass "never toggle-floating-panes" || fail "footgun (toggle-floating-panes used)"

# ── config lint ───────────────────────────────────────────────────────────────
grep -q 'bind "Alt c"' "$ROOT/zellij/config.kdl" && pass "Alt+c bound in config.kdl" || fail "no Alt+c bind"
grep -Fq 'bind "Alt c" { WriteChars "\u{1b}[99;3u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+c forwards to focused process for draft routing" \
  || fail "Alt+c target wrong"
grep -Fq 'bind "Alt r" { Write 27; Write 114; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+r forwards ESC+r for review-pane reject" \
  || fail "Alt+r does not forward to focused pane"
grep -Eq '^[[:space:]]*bind "Alt r"[^{]*\\{.*(RenameTab|TabNameInput|NewTab)' "$ROOT/zellij/config.kdl" \
  && fail "Alt+r still globally owns tab behavior" \
  || pass "Alt+r has no global tab action"
grep -q '^keybinds clear-defaults=true' "$ROOT/zellij/config.kdl" && pass "Alt+o default zellij tab-move disabled" || fail "Alt+o still captured by zellij"
grep -q 'Run "pair-review-toggle"' "$ROOT/zellij/config.kdl" && fail "Alt+c still spawns the old toggle pane" || pass "old pair-review-toggle pane gone"
grep -Fq 'bind "Alt x" { WriteChars "\u{1b}[120;3u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+x forwards to focused process for local routing" \
  || fail "Alt+x is not locally routed"
grep -Fq 'WriteChars ":lua PairConfirmQuit()"' "$ROOT/zellij/config.kdl" \
  && fail "Alt+x can still write quit prompt into right terminal" \
  || pass "Alt+x no longer injects quit command from zellij"

[ "$fails" -eq 0 ] || { printf 'review-toggle-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'review-toggle-test ok\n'
