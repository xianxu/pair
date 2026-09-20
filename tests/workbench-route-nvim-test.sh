#!/usr/bin/env bash
# Process-level coverage for global shortcut routing from Pair-owned nvim panes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-nvim-route.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/bin"
export PAIR_RETENTION_PROTOCOL=''
export PAIR_TEST_REAL_BIN="$ROOT/bin/pair"
cat > "$tmp/bin/zellij" <<EOF
#!/bin/sh
printf '%s\n' "\$*" >> "$tmp/all-actions"
if [ "\$*" = "action list-panes --json --command --state" ]; then
  printf '%s\n' '[{"id":9,"title":"terminal","terminal_command":"pair term"},{"id":42,"title":"draft","terminal_command":"nvim -u $ROOT/nvim/init.lua /tmp/draft.md"}]'
  exit 0
fi
if [ "\${FAIL_VIEW:-}" = "1" ] && [ "\$1" = "run" ]; then
  printf 'view launch failed\n'
  exit 1
fi
if [ "\${FAIL_FOCUS:-}" = "1" ] && [ "\$*" = "action focus-pane-id 42" ]; then
  exit 1
fi
printf '%s\n' "\$*" >> "$tmp/actions"
EOF
chmod +x "$tmp/bin/zellij"

cat > "$tmp/driver.lua" <<'LUA'
local route = dofile(os.getenv('PAIR_HOME') .. '/nvim/workbench_route.lua')
vim.fn.writefile({ vim.json.encode({
  session = vim.env.ZELLIJ_SESSION_NAME,
  pane_id = '42',
  pid = vim.fn.getpid(),
}) }, vim.env.PAIR_DRAFT_PANE_PATH)
local routed = route.route('PairConfirmRestart', true)
if vim.env.EXPECT_FAIL == '1' then
  assert(not routed)
else
  assert(routed)
end
vim.cmd('qa!')
LUA

mkdir -p "$tmp/data"
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
  PAIR_DRAFT_PANE_PATH="$tmp/data/draft-pane-t.json" \
  ZELLIJ_SESSION_NAME=pair-t \
  run_headless -- nvim --headless -u NONE -l "$tmp/driver.lua"

want='action focus-pane-id 42
action write --pane-id 42 28
action write --pane-id 42 14
action write-chars --pane-id 42 :lua PairConfirmRestart()
action write --pane-id 42 13'
got="$(cat "$tmp/actions")"
[ "$got" = "$want" ] || {
  printf 'FAIL addressed draft route:\n%s\n' "$got"
  exit 1
}
if grep -Fq 'list-panes' "$tmp/all-actions"; then
  printf 'FAIL valid pane locator still invoked slow list-panes path\n'
  exit 1
fi

: > "$tmp/actions"
: > "$tmp/all-actions"
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
  PAIR_DRAFT_PANE_PATH="$tmp/data/draft-pane-t.json" \
  ZELLIJ_SESSION_NAME=pair-t FAIL_FOCUS=1 EXPECT_FAIL=1 \
  run_headless -- nvim --headless -u NONE -l "$tmp/driver.lua"
[ ! -s "$tmp/actions" ] || {
  printf 'FAIL focus failure issued draft writes:\n%s\n' "$(cat "$tmp/actions")"
  exit 1
}
got="$(cat "$tmp/all-actions")"
[ "$got" = "action focus-pane-id 42" ] || {
  printf 'FAIL focus failure actions:\n%s\n' "$got"
  exit 1
}

cat > "$tmp/overlay-driver.lua" <<'LUA'
vim.fn.writefile({ vim.json.encode({
  session = vim.env.ZELLIJ_SESSION_NAME,
  pane_id = '42',
  pid = vim.fn.getpid(),
}) }, vim.env.PAIR_DRAFT_PANE_PATH)
local mapping = vim.fn.maparg(vim.env.TEST_KEY, 'n', false, true)
assert(type(mapping.callback) == 'function',
  vim.env.TEST_INIT .. ' missing effective ' .. vim.env.TEST_KEY .. ' callback')
mapping.callback()
vim.cmd('qa!')
LUA

run_overlay_map() {
  init="$1"
  key="$2"
  fail_focus="${3:-0}"
  : > "$tmp/actions"
  : > "$tmp/all-actions"
  : > "$tmp/view-$init.md"
  PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
    PAIR_DRAFT_PANE_PATH="$tmp/data/draft-pane-t.json" \
    ZELLIJ_SESSION_NAME=pair-t TEST_INIT="$init" TEST_KEY="$key" \
    FAIL_FOCUS="$fail_focus" \
    run_headless -- nvim --headless -u "$ROOT/nvim/$init.lua" \
      "$tmp/view-$init.md" -l "$tmp/overlay-driver.lua"
}

for init in review scrollback changelog; do
  run_overlay_map "$init" '<M-x>'
  got="$(cat "$tmp/actions")"
  want_quit='action focus-pane-id 42
action write --pane-id 42 28
action write --pane-id 42 14
action write-chars --pane-id 42 :lua PairConfirmQuit()
action write --pane-id 42 13'
  [ "$got" = "$want_quit" ] || {
    printf 'FAIL %s effective Alt+x route:\n%s\n' "$init" "$got"
    exit 1
  }

  run_overlay_map "$init" '<M-x>' 1
  [ ! -s "$tmp/actions" ] || {
    printf 'FAIL %s Alt+x wrote after focus failure:\n%s\n' "$init" "$(cat "$tmp/actions")"
    exit 1
  }
  got="$(cat "$tmp/all-actions")"
  [ "$got" = "action focus-pane-id 42" ] || {
    printf 'FAIL %s Alt+x focus-failure actions:\n%s\n' "$init" "$got"
    exit 1
  }

done

# Fullscreen executes from the invoking editor, even without a draft locator.
# A failing CLI owns diagnostics; neither stdout/stderr nor vim.notify may leak.
cat > "$tmp/bin/pair" <<'SH'
#!/bin/sh
if [ "$*" != 'layout toggle-focused' ]; then
  exec "$PAIR_TEST_REAL_BIN" "$@"
fi
printf '%s:%s\n' "$ZELLIJ_PANE_ID" "$*" >> "$PAIR_TEST_DIRECT_LOG"
printf 'captured stdout\n'
printf 'captured stderr\n' >&2
exit 17
SH
chmod +x "$tmp/bin/pair"
cat > "$tmp/fullscreen-driver.lua" <<'LUA'
vim.notify = function() error('fullscreen must not notify') end
for _, key in ipairs({ '<M-Up>', '<M-Down>' }) do
  assert((vim.fn.maparg(key, 'n') ~= '') == (vim.env.TEST_INIT == 'init'), key .. ' scope')
end
for _, mode in ipairs({ 'n', 'i' }) do
  local mapping = vim.fn.maparg('<S-M-CR>', mode, false, true)
  assert(mapping.buffer == 0, 'review must not shadow fullscreen with a local menu')
  assert(type(mapping.callback) == 'function', 'fullscreen missing')
  mapping.callback()
  assert(vim.v.shell_error == 17, 'exercise a failing fullscreen command')
end
local messages = vim.api.nvim_exec2('messages', { output = true }).output
assert(not messages:find('captured stdout', 1, true) and not messages:find('captured stderr', 1, true),
  'fullscreen command output reached editor messages')
vim.cmd('qa!')
LUA
for init in init review scrollback changelog; do
  : > "$tmp/direct-log"
  PATH="$tmp/bin:$PATH" PAIR_HOME='' PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
    PAIR_DRAFT_PANE_PATH='' ZELLIJ_PANE_ID=81 TEST_INIT="$init" \
    PAIR_TEST_DIRECT_LOG="$tmp/direct-log" PAIR_TEST_REAL_BIN="$ROOT/bin/pair" \
    run_headless -- nvim --headless -u "$ROOT/nvim/$init.lua" \
      "$tmp/view-$init.md" -l "$tmp/fullscreen-driver.lua" > "$tmp/direct-output" 2>&1 || {
        cat "$tmp/direct-output"
        exit 1
      }
  [ "$(cat "$tmp/direct-log")" = '81:layout toggle-focused
81:layout toggle-focused' ] || { printf 'FAIL %s direct fullscreen argv\n' "$init"; exit 1; }
  if grep -Eq 'captured stdout|captured stderr|fullscreen must not notify' "$tmp/direct-output"; then
    printf 'FAIL %s fullscreen diagnostic leaked\n' "$init"; exit 1
  fi
done

for init in init.lua review.lua scrollback.lua changelog.lua; do
  grep -Fq "workbench_route.lua" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not load shared router\n' "$init"; exit 1; }
  grep -Fq "install_global_maps" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not install global maps\n' "$init"; exit 1; }
done

grep -Fq "PAIR_DRAFT_PANE_PATH" "$ROOT/nvim/init.lua" ||
  { printf 'FAIL draft init does not publish pane locator\n'; exit 1; }


# Exercise the actual draft functions, preserving the former Zellij Run argv.
cat > "$tmp/view-driver.lua" <<'LUA'
vim.fn.writefile({}, vim.env.PAIR_ACTION_LOG)
PairOpenHelp()
PairOpenChangelog()
vim.cmd('qa!')
LUA
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
  PAIR_ACTION_LOG="$tmp/actions" \
  run_headless -- nvim --headless -u "$ROOT/nvim/init.lua" "$tmp/draft.md" -l "$tmp/view-driver.lua"
want_views='run --floating --close-on-exit --name help --width 100% --height 70% --x 0 --y 15% -- pair-help
run --floating --close-on-exit --name changelog --width 100% --height 100% --x 0 --y 0 -- pair changelog open'
[ "$(cat "$tmp/actions")" = "$want_views" ] || {
  printf 'FAIL role-local help/changelog argv:\n%s\n' "$(cat "$tmp/actions")"
  exit 1
}

cat > "$tmp/view-failure-driver.lua" <<'LUA'
local notices = {}
vim.notify = function(message, level) table.insert(notices, { message, level }) end
PairOpenHelp()
PairOpenChangelog()
assert(#notices == 2, vim.inspect(notices))
for _, notice in ipairs(notices) do
  assert(notice[1]:find('view launch failed', 1, true), notice[1])
  assert(notice[2] == vim.log.levels.ERROR)
end
vim.cmd('qa!')
LUA
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t FAIL_VIEW=1 \
  run_headless -- nvim --headless -u "$ROOT/nvim/init.lua" "$tmp/draft.md" -l "$tmp/view-failure-driver.lua"
printf 'workbench-route-nvim-test ok\n'
